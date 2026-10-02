package filesystem

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteMkdirRenameMoveDelete(t *testing.T) {
	root := t.TempDir()
	if err := Mkdir(root, "inbox/2026-10-02"); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(root, "inbox/2026-10-02/a.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(root, "inbox/2026-10-02/a.txt", []byte("again")); err == nil {
		t.Error("expected collision on overwrite")
	}
	if err := Rename(root, "inbox/2026-10-02/a.txt", "b.txt"); err != nil {
		t.Fatal(err)
	}
	if err := Rename(root, "inbox/2026-10-02/b.txt", "bad/name"); err == nil {
		t.Error("expected rejection of separators in new name")
	}
	if err := Move(root, "inbox/2026-10-02/b.txt", "inbox"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "inbox", "b.txt")); err != nil {
		t.Errorf("moved file missing: %v", err)
	}
	if err := Delete(root, "inbox/2026-10-02", false); err == nil {
		t.Error("expected refusal for non-empty dir without recursive")
	}
	if err := Delete(root, "inbox/2026-10-02", true); err != nil {
		t.Fatal(err)
	}
	if err := Delete(root, "inbox/b.txt", false); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteGuards(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"units", "units/unit-001", "database", ""} {
		if err := EnsureDir(filepath.Join(root, dir)); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{"", "units", "database", "units/unit-001"} {
		if err := Delete(root, rel, true); err == nil {
			t.Errorf("Delete(%q) should be refused", rel)
		}
	}
	// Deeper paths are allowed with recursive.
	if err := Delete(root, "units/unit-001", true); err == nil {
		t.Error("unit record root must stay protected")
	}
}

func TestStatListSearch(t *testing.T) {
	root := t.TempDir()
	if err := WriteFile(root, "inbox/Report.PDF", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(root, "inbox/notes.txt", []byte("yz")); err != nil {
		t.Fatal(err)
	}
	info, err := Stat(root, "inbox/Report.PDF")
	if err != nil {
		t.Fatal(err)
	}
	if info.Size != 1 || info.IsDir || info.Rel != "inbox/Report.PDF" {
		t.Errorf("unexpected stat %+v", info)
	}
	children, err := List(root, "inbox")
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 2 || children[0].Name != "notes.txt" {
		t.Errorf("unexpected listing %+v", children)
	}
	hits, err := Search(root, "report", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Name != "Report.PDF" {
		t.Errorf("unexpected search %+v", hits)
	}
	if hits, err := Search(root, "  ", 0); err == nil || len(hits) != 0 {
		t.Errorf("empty query should fail: %v %v", hits, err)
	}
}

func TestOpenValidationOnly(t *testing.T) {
	root := t.TempDir()
	if err := Open(root, "inbox/missing.pdf"); err == nil {
		t.Error("expected error for missing file")
	}
	if err := Mkdir(root, "inbox/d"); err != nil {
		t.Fatal(err)
	}
	if err := Open(root, "inbox/d"); err == nil {
		t.Error("expected refusal for directories")
	}
	if err := Open(root, "../outside"); err == nil {
		t.Error("expected traversal rejection")
	}
}

func TestScanFlagsMissing(t *testing.T) {
	root := t.TempDir()
	db := openTestDB(t, root)
	execSQL(t, db, `INSERT INTO units(id, name, created_at, updated_at)
		VALUES ('unit-001', 'U', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`)
	execSQL(t, db, `INSERT INTO files(unit_id, rel_path, size, created_at, updated_at)
		VALUES ('unit-001', 'inbox/gone.pdf', 3, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`)
	execSQL(t, db, `INSERT INTO files(unit_id, rel_path, size, created_at, updated_at)
		VALUES ('unit-001', 'inbox/here.pdf', 3, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`)
	if err := WriteFile(root, "inbox/here.pdf", []byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(root, "inbox/stray.pdf", []byte("z")); err != nil {
		t.Fatal(err)
	}
	report, err := Scan(root, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Missing) != 1 || report.Missing[0] != "inbox/gone.pdf" {
		t.Errorf("missing = %v", report.Missing)
	}
	if len(report.Unindexed) != 1 || !strings.HasSuffix(report.Unindexed[0], "stray.pdf") {
		t.Errorf("unindexed = %v", report.Unindexed)
	}
	var missing int
	if err := db.QueryRow(`SELECT missing FROM files WHERE rel_path = 'inbox/gone.pdf'`).Scan(&missing); err != nil || missing != 1 {
		t.Errorf("missing flag not set: %v %v", missing, err)
	}
	// File restored on disk clears the flag on rescan.
	if err := WriteFile(root, "inbox/gone.pdf", []byte("back")); err != nil {
		t.Fatal(err)
	}
	report, err = Scan(root, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Missing) != 0 {
		t.Errorf("missing after restore = %v", report.Missing)
	}
}
