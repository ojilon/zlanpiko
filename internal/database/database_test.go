package database

import (
	"database/sql"
	"os"
	"testing"

	"zlanpiko/migrations"
)

// openMigrated opens a fresh temp-root database and migrates it to v1.
func openMigrated(t *testing.T) (*sql.DB, string) {
	t.Helper()
	root := t.TempDir()
	db, err := Open(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	v, err := Migrate(db, migrations.FS, 1)
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Fatalf("migrated version = %d, want 1", v)
	}
	return db, root
}

func reopen(t *testing.T, root string) *sql.DB {
	t.Helper()
	db, err := Open(Path(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var found string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found)
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return true
}

func TestMigrateFreshCreatesSchema(t *testing.T) {
	_, root := openMigrated(t)
	if _, err := os.Stat(Path(root)); err != nil {
		t.Fatalf("database file missing: %v", err)
	}
	for _, table := range []string{"units", "topics", "tasks", "files", "schema_meta"} {
		if !tableExists(t, reopen(t, root), table) {
			t.Errorf("table %s missing after migration", table)
		}
	}
}

func TestReopenIsIdempotent(t *testing.T) {
	db, root := openMigrated(t)
	if _, err := db.Exec(`INSERT INTO units(id, name, created_at, updated_at) VALUES ('unit-001', 'X', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	db2 := reopen(t, root)
	v, err := Migrate(db2, migrations.FS, 1) // second migrate is a no-op
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Errorf("version = %d, want 1", v)
	}
	var name string
	if err := db2.QueryRow(`SELECT name FROM units WHERE id = 'unit-001'`).Scan(&name); err != nil {
		t.Fatalf("row lost across reopen: %v", err)
	}
}

func TestMigrateRefusesNewerSchema(t *testing.T) {
	db, _ := openMigrated(t)
	if _, err := db.Exec(`UPDATE schema_meta SET value = '99' WHERE key = 'schema_version'`); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(db, migrations.FS, 1); err == nil {
		t.Error("expected refusal for schema newer than the build")
	}
}

func TestVerifyHealthy(t *testing.T) {
	db, _ := openMigrated(t)
	if err := Verify(db); err != nil {
		t.Errorf("Verify on fresh db: %v", err)
	}
}

func TestForeignKeysEnforced(t *testing.T) {
	db, _ := openMigrated(t)
	_, err := db.Exec(`INSERT INTO topics(unit_id, id, name, created_at, updated_at) VALUES ('unit-zzz', 'topic-001', 'T', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`)
	if err == nil {
		t.Error("expected foreign key violation for orphan topic")
	}
}
