// Package database owns the SQLite connection, schema migrations and
// integrity checks. It is the only package that imports the SQL driver,
// so driver swaps touch exactly one place (see docs/05, DECISIONS.md D2).
package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

// FileName is the database file name inside <root>/database.
const FileName = "zlanpiko.db"

// Path returns the database file path for a data root.
func Path(dataRoot string) string {
	return filepath.Join(dataRoot, "database", FileName)
}

// Open creates the parent directory, opens the database and applies the
// single-writer pragmas. The file is created on first open.
func Open(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("database: create database dir: %w", err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(dbPath))
	if err != nil {
		return nil, fmt.Errorf("database: open %s: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1) // single writer; TUI or CLI holds the handle
	for _, pragma := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("database: %s: %w", pragma, err)
		}
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("database: ping %s: %w", dbPath, err)
	}
	return db, nil
}

// CurrentVersion reads schema_meta.schema_version; a fresh database
// (no schema_meta table) reports 0.
func CurrentVersion(db *sql.DB) (int, error) {
	var table string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'schema_meta'`).Scan(&table)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("database: inspect schema_meta: %w", err)
	}
	var value string
	if err := db.QueryRow(`SELECT value FROM schema_meta WHERE key = 'schema_version'`).Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("database: schema_meta table exists but has no schema_version row")
		}
		return 0, fmt.Errorf("database: read schema version: %w", err)
	}
	v, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("database: bad schema version %q: %w", value, err)
	}
	return v, nil
}

// migration is one ordered NNNN_name.sql file.
type migration struct {
	version int
	name    string
	sql     string
}

func listMigrations(mfs fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(mfs, ".")
	if err != nil {
		return nil, fmt.Errorf("database: read migrations: %w", err)
	}
	var out []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		ver, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
		if err != nil {
			return nil, fmt.Errorf("database: bad migration name %q: %w", e.Name(), err)
		}
		raw, err := fs.ReadFile(mfs, e.Name())
		if err != nil {
			return nil, fmt.Errorf("database: read migration %s: %w", e.Name(), err)
		}
		out = append(out, migration{version: ver, name: e.Name(), sql: string(raw)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// Migrate applies pending migrations up to wantVersion, each in its own
// transaction, and returns the resulting version. It refuses to open a
// database newer than the embedded set (never auto-downgrades) and refuses
// to run when the binary is older than the embedded set.
func Migrate(db *sql.DB, mfs fs.FS, wantVersion int) (int, error) {
	migs, err := listMigrations(mfs)
	if err != nil {
		return 0, err
	}
	if len(migs) == 0 {
		return 0, fmt.Errorf("database: no migrations embedded")
	}
	maxEmbedded := migs[len(migs)-1].version
	if maxEmbedded > wantVersion {
		return 0, fmt.Errorf("database: embedded schema v%d is newer than this build understands (v%d); update the application", maxEmbedded, wantVersion)
	}
	current, err := CurrentVersion(db)
	if err != nil {
		return 0, err
	}
	if current > maxEmbedded {
		return 0, fmt.Errorf("database: schema v%d is newer than this build (embedded v%d); update the application, never downgrade", current, maxEmbedded)
	}
	for _, m := range migs {
		if m.version <= current || m.version > wantVersion {
			continue
		}
		if err := applyMigration(db, m); err != nil {
			return current, err
		}
		current = m.version
	}
	return current, nil
}

func applyMigration(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("database: begin migration %s: %w", m.name, err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(m.sql); err != nil {
		return fmt.Errorf("database: apply migration %s: %w", m.name, err)
	}
	if _, err := tx.Exec(`INSERT INTO schema_meta(key, value) VALUES ('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, strconv.Itoa(m.version)); err != nil {
		return fmt.Errorf("database: record migration %s: %w", m.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("database: commit migration %s: %w", m.name, err)
	}
	return nil
}

// Verify runs PRAGMA integrity_check plus a schema-version sanity read.
// Deeper orphan/sidecar checks arrive with maintenance verify (Phase 7).
func Verify(db *sql.DB) error {
	var result string
	rows, err := db.Query(`PRAGMA integrity_check`)
	if err != nil {
		return fmt.Errorf("database: integrity_check: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return fmt.Errorf("database: read integrity_check: %w", err)
		}
		result += line + "\n"
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("database: integrity_check rows: %w", err)
	}
	if strings.TrimSpace(result) != "ok" {
		return fmt.Errorf("database: integrity_check failed:\n%s", result)
	}
	if _, err := CurrentVersion(db); err != nil {
		return err
	}
	return nil
}
