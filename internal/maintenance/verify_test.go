package maintenance

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"zlanpiko/internal/database"
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
	return db, root
}

func TestVerifyCleanTree(t *testing.T) {
	db, root := setupDB(t)
	u, err := services.CreateUnit(db, root, nil, "U", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := services.CreateTopic(db, root, nil, u.ID, "T", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := tasks.CreateTask(db, root, nil, u.ID, "A", "assignment", nil, "", ""); err != nil {
		t.Fatal(err)
	}
	rep, err := Verify(db, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Clean() {
		t.Errorf("fresh tree should verify clean: %+v", rep)
	}
	if rep.Integrity != "ok" {
		t.Errorf("integrity = %q", rep.Integrity)
	}
}

func TestVerifyFindsAndRepairsSidecars(t *testing.T) {
	db, root := setupDB(t)
	u, err := services.CreateUnit(db, root, nil, "U", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	// Tamper: stale sidecar (old updated_at) + deleted sidecar + temp leftover.
	sidecar := filepath.Join(root, "units", u.ID, "unit.json")
	raw, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatal(err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}
	asMap["updated_at"] = "2000-01-01T00:00:00Z"
	stale, err := json.Marshal(asMap)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecar, stale, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := services.CreateTopic(db, root, nil, u.ID, "T", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "units", u.ID, "topics", "topic-001", "topic.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inbox", ".tmp-leftover"), []byte("x"), 0o644); err != nil {
		if mkErr := os.MkdirAll(filepath.Join(root, "inbox"), 0o755); mkErr != nil {
			t.Fatal(mkErr)
		}
		if err := os.WriteFile(filepath.Join(root, "inbox", ".tmp-leftover"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rep, err := Verify(db, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Sidecars) != 2 {
		t.Errorf("sidecar issues = %+v", rep.Sidecars)
	}
	if len(rep.TempFiles) != 1 {
		t.Errorf("temp files = %+v", rep.TempFiles)
	}
	if err := Repair(db, root, nil, rep); err != nil {
		t.Fatal(err)
	}
	rep2, err := Verify(db, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep2.Sidecars) != 0 || len(rep2.TempFiles) != 0 {
		t.Errorf("after repair: %+v", rep2)
	}
}

func TestVerifyReportsMissingAndUnindexed(t *testing.T) {
	db, root := setupDB(t)
	u, err := services.CreateUnit(db, root, nil, "U", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO files(unit_id, rel_path, size, created_at, updated_at)
		VALUES (?, 'inbox/gone.pdf', 1, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "inbox"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inbox", "stray.pdf"), []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err := Verify(db, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Missing) != 1 || rep.Missing[0] != "inbox/gone.pdf" {
		t.Errorf("missing = %v", rep.Missing)
	}
	if len(rep.Unindexed) != 1 {
		t.Errorf("unindexed = %v", rep.Unindexed)
	}
	// Repair does not touch file divergence (needs user decisions).
	if err := Repair(db, root, nil, rep); err != nil {
		t.Fatal(err)
	}
	rep2, err := Verify(db, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep2.Missing) != 1 || len(rep2.Unindexed) != 1 {
		t.Errorf("file divergence must survive repair: %+v", rep2)
	}
}
