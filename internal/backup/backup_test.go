package backup

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zlanpiko/internal/database"
	"zlanpiko/internal/exporter"
	"zlanpiko/internal/services"
	"zlanpiko/internal/tasks"
	"zlanpiko/migrations"
)

func setupDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(database.Path(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(db, migrations.FS, 1); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"units", "inbox", "backups", "config"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return db, root
}

func seedWork(t *testing.T, db *sql.DB, root string) {
	t.Helper()
	u, err := services.CreateUnit(db, root, nil, "Algebra", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := services.CreateTopic(db, root, nil, u.ID, "Eigen", ""); err != nil {
		t.Fatal(err)
	}
	due := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	if _, err := tasks.CreateTask(db, root, nil, u.ID, "Set", "assignment", &due, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inbox", "note.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsEmpty(t *testing.T) {
	db, root := setupDB(t)
	empty, err := IsEmpty(db)
	if err != nil || !empty {
		t.Fatalf("fresh db empty = %v, %v", empty, err)
	}
	seedWork(t, db, root)
	empty, err = IsEmpty(db)
	if err != nil || empty {
		t.Fatalf("seeded db empty = %v, %v", empty, err)
	}
}

func TestCreateVerifyList(t *testing.T) {
	db, root := setupDB(t)
	seedWork(t, db, root)
	m, err := Create(db, root, "", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Kind != KindFull || len(m.Files) == 0 {
		t.Errorf("manifest = %+v", m)
	}
	matches, _ := filepath.Glob(filepath.Join(root, "backups", "*.zip"))
	if len(matches) != 1 {
		t.Fatalf("backups = %v", matches)
	}
	if _, err := Verify(matches[0]); err != nil {
		t.Fatal(err)
	}
	list, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !list[0].Readable {
		t.Errorf("list = %+v", list)
	}
	// Refuse to overwrite an existing backup name.
	existing := filepath.Join(root, "backups", "taken.zip")
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(db, root, existing, true, nil); err == nil {
		t.Error("expected overwrite refusal")
	}
}

func TestRoundTrip(t *testing.T) {
	db, root := setupDB(t)
	seedWork(t, db, root)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	before, err := exporter.TextReport(db, "", now)
	if err != nil {
		t.Fatal(err)
	}
	jsonBefore, err := exporter.JSONExport(db, "", now)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Create(db, root, "", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = m
	matches, _ := filepath.Glob(filepath.Join(root, "backups", "*.zip"))
	backupPath := matches[0]

	// Wipe everything except the backup (simulates total data loss).
	db.Close()
	for _, dir := range []string{"units", "inbox", "database", "config"} {
		if err := os.RemoveAll(filepath.Join(root, dir)); err != nil {
			t.Fatal(err)
		}
	}

	// Restore into the wiped root (no overwrite needed, target absent).
	if err := Restore(root, backupPath, false, nil); err != nil {
		t.Fatal(err)
	}
	db2, err := database.Open(database.Path(root))
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	if _, err := database.Migrate(db2, migrations.FS, 1); err != nil {
		t.Fatal(err)
	}
	after, err := exporter.TextReport(db2, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Errorf("report changed across restore:\n--- before ---\n%s\n--- after ---\n%s", before, after)
	}
	jsonAfter, err := exporter.JSONExport(db2, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if string(jsonAfter) != string(jsonBefore) {
		t.Errorf("json changed across restore")
	}
	// Restored user files are back on disk.
	if raw, err := os.ReadFile(filepath.Join(root, "inbox", "note.txt")); err != nil || string(raw) != "hello" {
		t.Errorf("restored file = %q, %v", raw, err)
	}
	var count int
	if err := db2.QueryRow(`SELECT COUNT(*) FROM units`).Scan(&count); err != nil || count != 1 {
		t.Errorf("units = %d, %v", count, err)
	}
}

func TestRestoreNeedsEmptyOrOverwrite(t *testing.T) {
	db, root := setupDB(t)
	seedWork(t, db, root)
	if _, err := Create(db, root, "", true, nil); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(root, "backups", "*.zip"))
	backupPath := matches[0]
	// Target DB exists: refusal without overwrite.
	if err := Restore(root, backupPath, false, nil); err == nil {
		t.Error("expected refusal on non-empty target")
	} else if !strings.Contains(err.Error(), "overwrite") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestRestoreRefusesNewerSchema(t *testing.T) {
	db, root := setupDB(t)
	if _, err := Create(db, root, "", false, nil); err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(root, "backups", "*.zip"))
	bumped := bumpSchema(t, matches[0], 999)
	// Wipe the target so only the schema check can fail.
	db.Close()
	if err := os.RemoveAll(filepath.Join(root, "database")); err != nil {
		t.Fatal(err)
	}
	if err := Restore(root, bumped, false, nil); err == nil {
		t.Error("expected refusal for newer-schema backup")
	} else if !strings.Contains(err.Error(), "newer") {
		t.Errorf("wrong error: %v", err)
	}
}

// bumpSchema copies a backup zip with manifest.schema_version rewritten.
func bumpSchema(t *testing.T, src string, v int) string {
	t.Helper()
	zr, err := zip.OpenReader(src)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	dst := src + ".bumped.zip"
	zw, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(zw)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name == "manifest.json" {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			m["schema_version"] = v
			raw, err = json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
		}
		ew, err := w.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ew.Write(raw); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return dst
}
