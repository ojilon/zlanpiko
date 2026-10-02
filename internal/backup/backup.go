// Package backup implements local backups: a consistent database snapshot
// (VACUUM INTO, never a raw copy of a live DB) plus user files, sealed in a
// zip with a manifest. Restore verifies first, refuses newer schemas, and
// takes a safety backup before overwriting data (see docs/11).
package backup

import (
	"archive/zip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zlanpiko/internal/app"
	"zlanpiko/internal/database"
)

// KindFull includes user files; KindDB is the database snapshot only.
const (
	KindFull = "full"
	KindDB   = "db-only"
)

// FileEntry is one archived file with its integrity hash.
type FileEntry struct {
	Path   string `json:"path"` // zip-relative path
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Manifest describes a backup (stored as manifest.json inside the zip).
type Manifest struct {
	App           string      `json:"app"`
	Version       string      `json:"version"`
	SchemaVersion int         `json:"schema_version"`
	CreatedAt     time.Time   `json:"created_at"`
	Kind          string      `json:"kind"`
	DBPath        string      `json:"db_path"`
	DBSHA256      string      `json:"db_sha256"`
	DBSize        int64       `json:"db_size"`
	Files         []FileEntry `json:"files"`
}

// contentAreas are the user-content subtrees in a full backup, plus the
// data-local preferences file.
var contentAreas = []string{"units", "inbox"}

// DefaultName returns the timestamped default file name in backups/.
func DefaultName(root, kind string, now time.Time) string {
	stamp := now.UTC().Format("20060102-150405")
	return filepath.Join(root, "backups", "backup-"+kind+"-"+stamp+".zip")
}

func orDiscard(l *slog.Logger) *slog.Logger {
	if l == nil {
		return slog.New(slog.DiscardHandler)
	}
	return l
}

// snapshotDB writes a consistent snapshot via VACUUM INTO (checkpoint first).
func snapshotDB(db *sql.DB, dest string) error {
	if _, err := db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("backup: checkpoint: %w", err)
	}
	lit := "'" + strings.ReplaceAll(dest, "'", "''") + "'"
	if _, err := db.Exec(`VACUUM INTO ` + lit); err != nil {
		return fmt.Errorf("backup: snapshot: %w", err)
	}
	return nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// Create builds a backup of the open database plus (full) user files.
// outPath "" selects the default name; an existing destination is refused.
func Create(db *sql.DB, root, outPath string, full bool, logger *slog.Logger) (*Manifest, error) {
	logger = orDiscard(logger)
	kind := KindDB
	if full {
		kind = KindFull
	}
	now := time.Now().UTC().Truncate(time.Second)
	if outPath == "" {
		outPath = DefaultName(root, kind, now)
	}
	if _, err := os.Lstat(outPath); err == nil {
		return nil, fmt.Errorf("backup: refusing to overwrite %s", outPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("backup: inspect %s: %w", outPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return nil, fmt.Errorf("backup: create backups dir: %w", err)
	}
	stage, err := os.MkdirTemp("", "zlanpiko-backup-*")
	if err != nil {
		return nil, fmt.Errorf("backup: stage: %w", err)
	}
	defer os.RemoveAll(stage)
	snapPath := filepath.Join(stage, "snapshot.db")
	if err := snapshotDB(db, snapPath); err != nil {
		return nil, err
	}
	sum, size, err := hashFile(snapPath)
	if err != nil {
		return nil, err
	}
	manifest := &Manifest{
		App: app.Name, Version: app.Version, SchemaVersion: app.SchemaVersion,
		CreatedAt: now, Kind: kind, DBPath: "database/" + database.FileName,
		DBSHA256: sum, DBSize: size, Files: []FileEntry{},
	}
	zf, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("backup: create %s: %w", outPath, err)
	}
	zw := zip.NewWriter(zf)
	if err := addFile(zw, snapPath, manifest.DBPath); err != nil {
		zw.Close()
		zf.Close()
		os.Remove(outPath)
		return nil, err
	}
	if full {
		if err := addContent(zw, root, manifest); err != nil {
			zw.Close()
			zf.Close()
			os.Remove(outPath)
			return nil, err
		}
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		zw.Close()
		zf.Close()
		os.Remove(outPath)
		return nil, err
	}
	w, err := zw.Create("manifest.json")
	if err != nil {
		zw.Close()
		zf.Close()
		os.Remove(outPath)
		return nil, err
	}
	if _, err := w.Write(append(raw, '\n')); err != nil {
		zw.Close()
		zf.Close()
		os.Remove(outPath)
		return nil, err
	}
	if err := zw.Close(); err != nil {
		zf.Close()
		os.Remove(outPath)
		return nil, fmt.Errorf("backup: close zip: %w", err)
	}
	if err := zf.Close(); err != nil {
		os.Remove(outPath)
		return nil, fmt.Errorf("backup: close file: %w", err)
	}
	// Verify what we wrote before reporting success.
	if _, err := Verify(outPath); err != nil {
		os.Remove(outPath)
		return nil, fmt.Errorf("backup: self-verify failed: %w", err)
	}
	logger.Info("backup created", "path", outPath, "kind", kind, "files", len(manifest.Files))
	return manifest, nil
}

// addFile stores one host file under a zip path.
func addFile(zw *zip.Writer, hostPath, zipPath string) error {
	f, err := os.Open(hostPath)
	if err != nil {
		return fmt.Errorf("backup: open %s: %w", hostPath, err)
	}
	defer f.Close()
	w, err := zw.Create(zipPath)
	if err != nil {
		return fmt.Errorf("backup: zip entry %s: %w", zipPath, err)
	}
	if _, err := io.Copy(w, f); err != nil {
		return fmt.Errorf("backup: zip %s: %w", zipPath, err)
	}
	return nil
}

// addContent archives units/, inbox/ and config/user.json (skipping temp
// staging files, the live database dir and other app-managed areas).
func addContent(zw *zip.Writer, root string, manifest *Manifest) error {
	var rels []string
	collect := func(base string) error {
		abs := filepath.Join(root, base)
		if _, err := os.Stat(abs); err != nil {
			return nil
		}
		return filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || strings.HasPrefix(d.Name(), ".tmp-") {
				return nil
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				return rerr
			}
			rels = append(rels, filepath.ToSlash(rel))
			return nil
		})
	}
	for _, area := range contentAreas {
		if err := collect(area); err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(root, "config", "user.json")); err == nil {
		rels = append(rels, "config/user.json")
	}
	for _, rel := range rels {
		zipPath := "files/" + rel
		hostPath := filepath.Join(root, filepath.FromSlash(rel))
		sum, size, err := hashFile(hostPath)
		if err != nil {
			return fmt.Errorf("backup: hash %s: %w", rel, err)
		}
		if err := addFile(zw, hostPath, zipPath); err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, FileEntry{Path: zipPath, SHA256: sum, Size: size})
	}
	return nil
}

// extract unpacks a backup into stage (same volume as root).
func extract(backupPath, stage string) error {
	zr, err := zip.OpenReader(backupPath)
	if err != nil {
		return fmt.Errorf("backup: open %s: %w", backupPath, err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if !validEntry(f.Name) {
			return fmt.Errorf("backup: refusing unsafe entry %q", f.Name)
		}
		dest := filepath.Join(stage, filepath.FromSlash(f.Name))
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return fmt.Errorf("backup: stage dir: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("backup: stage dir: %w", err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("backup: read %s: %w", f.Name, err)
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			rc.Close()
			return fmt.Errorf("backup: stage %s: %w", f.Name, err)
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return fmt.Errorf("backup: stage %s: %w", f.Name, err)
		}
	}
	return nil
}

// validEntry rejects zip-slip paths and absolute names.
func validEntry(name string) bool {
	if filepath.IsAbs(name) || filepath.IsAbs(filepath.FromSlash(name)) {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	return clean != ".." && !strings.HasPrefix(clean, ".."+string(os.PathSeparator))
}

// publish moves staged content into the data root (remove-then-rename: os
// rename cannot replace on Windows). Only manifest-listed paths are moved.
func publish(stage, root string, m *Manifest) error {
	move := func(stagedRel, rootRel string) error {
		src := filepath.Join(stage, filepath.FromSlash(stagedRel))
		dst := filepath.Join(root, filepath.FromSlash(rootRel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("backup: prepare %s: %w", rootRel, err)
		}
		if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("backup: clear %s: %w", rootRel, err)
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("backup: publish %s: %w", rootRel, err)
		}
		return nil
	}
	if err := move(m.DBPath, m.DBPath); err != nil {
		return err
	}
	// A replaced database must not keep the old WAL sidecars.
	restored := filepath.Join(root, filepath.FromSlash(m.DBPath))
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if err := os.Remove(restored + suffix); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("backup: clear %s: %w", restored+suffix, err)
		}
	}
	for _, fe := range m.Files {
		rel := strings.TrimPrefix(fe.Path, "files/")
		if err := move(fe.Path, rel); err != nil {
			return err
		}
	}
	return nil
}

// readManifest opens a backup and parses its manifest.
func readManifest(backupPath string) (*zip.ReadCloser, *Manifest, error) {
	zr, err := zip.OpenReader(backupPath)
	if err != nil {
		return nil, nil, fmt.Errorf("backup: open %s: %w", backupPath, err)
	}
	for _, f := range zr.File {
		if f.Name != "manifest.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			zr.Close()
			return nil, nil, fmt.Errorf("backup: read manifest: %w", err)
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			zr.Close()
			return nil, nil, fmt.Errorf("backup: read manifest: %w", err)
		}
		var m Manifest
		if err := json.Unmarshal(raw, &m); err != nil {
			zr.Close()
			return nil, nil, fmt.Errorf("backup: parse manifest: %w", err)
		}
		return zr, &m, nil
	}
	zr.Close()
	return nil, nil, fmt.Errorf("backup: %s is not a zlanpiko backup (no manifest)", backupPath)
}

// Verify re-reads a backup and checks every hash (docs/11).
func Verify(backupPath string) (*Manifest, error) {
	zr, m, err := readManifest(backupPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	byName := map[string]*zip.File{}
	for _, f := range zr.File {
		byName[f.Name] = f
	}
	check := func(name, want string) error {
		f := byName[name]
		if f == nil {
			return fmt.Errorf("backup: entry missing: %s", name)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("backup: read %s: %w", name, err)
		}
		h := sha256.New()
		if _, err := io.Copy(h, rc); err != nil {
			rc.Close()
			return fmt.Errorf("backup: read %s: %w", name, err)
		}
		rc.Close()
		if got := hex.EncodeToString(h.Sum(nil)); got != want {
			return fmt.Errorf("backup: hash mismatch for %s", name)
		}
		return nil
	}
	if err := check(m.DBPath, m.DBSHA256); err != nil {
		return nil, err
	}
	for _, fe := range m.Files {
		if err := check(fe.Path, fe.SHA256); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// IsEmpty reports whether the database holds no academic records at all.
// An empty database counts as an empty restore target (app.Open recreates
// the file on every launch, so absence alone is not the test).
func IsEmpty(db *sql.DB) (bool, error) {
	for _, table := range []string{"units", "topics", "tasks", "files"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
			return false, fmt.Errorf("backup: inspect %s: %w", table, err)
		}
		if n > 0 {
			return false, nil
		}
	}
	return true, nil
}

// BackupInfo is one entry of List.
type BackupInfo struct {
	Path     string    `json:"path"`
	Kind     string    `json:"kind"`
	Created  time.Time `json:"created_at"`
	Files    int       `json:"files"`
	Size     int64     `json:"size"`
	Readable bool      `json:"readable"`
}

// List enumerates backups/*.zip newest first (best effort per file).
func List(root string) ([]BackupInfo, error) {
	matches, err := filepath.Glob(filepath.Join(root, "backups", "*.zip"))
	if err != nil {
		return nil, err
	}
	out := []BackupInfo{}
	for _, path := range matches {
		st, err := os.Stat(path)
		if err != nil {
			continue
		}
		info := BackupInfo{Path: path, Size: st.Size(), Created: st.ModTime()}
		if m, verr := Verify(path); verr == nil {
			info.Kind, info.Created, info.Files, info.Readable =
				m.Kind, m.CreatedAt, len(m.Files), true
		}
		out = append(out, info)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Created.After(out[j-1].Created); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}

// Restore replaces the data root's database and (full) user files from a
// verified backup. The target database must be absent or overwrite=true.
// Newer-schema backups are refused; older ones migrate on next open.
// The caller must close any open DB handle before calling Restore, and is
// responsible for the pre-overwrite safety backup (it needs the live handle;
// see the CLI restore flow).
func Restore(root, backupPath string, overwrite bool, logger *slog.Logger) error {
	logger = orDiscard(logger)
	zr, m, err := readManifest(backupPath)
	if err != nil {
		return err
	}
	zr.Close()
	if _, err := Verify(backupPath); err != nil {
		return err
	}
	if m.SchemaVersion > app.SchemaVersion {
		return fmt.Errorf("backup: schema v%d is newer than this build (v%d)", m.SchemaVersion, app.SchemaVersion)
	}
	targetDB := database.Path(root)
	if _, err := os.Lstat(targetDB); err == nil && !overwrite {
		return fmt.Errorf("backup: %s exists (re-run with overwrite enabled after taking a safety backup)", targetDB)
	}
	stage, err := os.MkdirTemp(root, ".restore-*")
	if err != nil {
		return fmt.Errorf("backup: stage restore: %w", err)
	}
	defer os.RemoveAll(stage)
	if err := extract(backupPath, stage); err != nil {
		return err
	}
	if err := publish(stage, root, m); err != nil {
		return err
	}
	logger.Info("restore complete", "from", backupPath, "kind", m.Kind)
	return nil
}
