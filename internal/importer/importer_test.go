package importer

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zlanpiko/internal/database"
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

func writeSrc(t *testing.T, dir, name string, size int) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.Repeat("x", size)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPreviewSkipsAndPlans(t *testing.T) {
	db, root := setupDB(t)
	src := t.TempDir()
	writeSrc(t, src, "notes.txt", 10)
	writeSrc(t, src, "paper.pdf", 20)
	writeSrc(t, src, ".hidden", 5)
	writeSrc(t, src, "setup.exe", 30)
	writeSrc(t, src, "big.bin", 1000)

	pv, err := Preview(db, root, src, Options{MaxBytes: 100})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(pv.DestDir, "inbox/") {
		t.Errorf("default dest = %q, want inbox/<date>", pv.DestDir)
	}
	actions := map[string]string{}
	for _, pl := range pv.Plans {
		actions[pl.Name] = pl.Action + ":" + pl.Reason
	}
	if len(pv.Plans) != 5 {
		t.Fatalf("plans = %d, want 5 (recursive off still lists top level)", len(pv.Plans))
	}
	for _, name := range []string{"notes.txt", "paper.pdf"} {
		if !strings.HasPrefix(actions[name], "copy") {
			t.Errorf("%s -> %q, want copy", name, actions[name])
		}
	}
	for _, name := range []string{".hidden", "setup.exe", "big.bin"} {
		if !strings.HasPrefix(actions[name], "skip") {
			t.Errorf("%s -> %q, want skip", name, actions[name])
		}
	}
	// Preview copies nothing.
	if entries, _ := os.ReadDir(filepath.Join(root, pv.DestDir)); len(entries) != 0 {
		t.Errorf("preview must not copy, found %v", entries)
	}
}

func TestExecuteCopiesAndRecords(t *testing.T) {
	db, root := setupDB(t)
	src := t.TempDir()
	writeSrc(t, src, "a.txt", 10)
	pv, err := Preview(db, root, src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	report, err := Execute(context.Background(), db, root, pv, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Copied != 1 || report.Failed != 0 {
		t.Errorf("report = %+v", report)
	}
	if report.Results[0].SHA256 == "" {
		t.Error("expected recorded hash")
	}
	var rel string
	if err := db.QueryRow(`SELECT rel_path FROM files`).Scan(&rel); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, "inbox/") {
		t.Errorf("rel = %q", rel)
	}
	// Second run collides -> rename policy appends -2.
	pv2, err := Preview(db, root, src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !pv2.Plans[0].Renamed || !strings.Contains(pv2.Plans[0].DestRel, "a-2.txt") {
		t.Errorf("expected rename plan, got %+v", pv2.Plans[0])
	}
	report2, err := Execute(context.Background(), db, root, pv2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report2.Copied != 1 {
		t.Errorf("report2 = %+v", report2)
	}
}

func TestExecuteSkipPolicy(t *testing.T) {
	db, root := setupDB(t)
	src := t.TempDir()
	writeSrc(t, src, "a.txt", 10)
	pv, err := Preview(db, root, src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(context.Background(), db, root, pv, nil); err != nil {
		t.Fatal(err)
	}
	pv2, err := Preview(db, root, src, Options{Collision: CollisionSkip})
	if err != nil {
		t.Fatal(err)
	}
	report, err := Execute(context.Background(), db, root, pv2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Copied != 0 || report.Skipped != 1 {
		t.Errorf("report = %+v", report)
	}
}

func TestRecordDestLinks(t *testing.T) {
	db, root := setupDB(t)
	if _, err := db.Exec(`INSERT INTO units(id, name, created_at, updated_at)
		VALUES ('unit-001', 'U', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	writeSrc(t, src, "r.pdf", 10)
	pv, err := Preview(db, root, src, Options{Dest: "unit-001"})
	if err != nil {
		t.Fatal(err)
	}
	if pv.UnitID != "unit-001" {
		t.Errorf("unit link = %q", pv.UnitID)
	}
	if _, err := Execute(context.Background(), db, root, pv, nil); err != nil {
		t.Fatal(err)
	}
	var unitID string
	if err := db.QueryRow(`SELECT unit_id FROM files`).Scan(&unitID); err != nil || unitID != "unit-001" {
		t.Errorf("unit_id = %q, %v", unitID, err)
	}
	if _, err := Preview(db, root, src, Options{Dest: "unit-999"}); err == nil {
		t.Error("expected error for unknown unit dest")
	}
	if _, err := Preview(db, root, src, Options{Dest: "inbox"}); err != nil {
		t.Errorf("bare inbox should work: %v", err)
	}
}

func TestCancelledExecute(t *testing.T) {
	db, root := setupDB(t)
	src := t.TempDir()
	writeSrc(t, src, "a.txt", 10)
	pv, err := Preview(db, root, src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Execute(ctx, db, root, pv, nil); err == nil {
		t.Error("expected cancellation error")
	}
}
