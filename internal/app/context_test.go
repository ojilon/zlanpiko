package app

import (
	"os"
	"path/filepath"
	"testing"

	"zlanpiko/internal/config"
)

func TestOpenCreatesSkeletonAndReopens(t *testing.T) {
	root := filepath.Join(t.TempDir(), "ZlanpikoData")
	ctx, err := Open(OpenOptions{DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range append(SkeletonDirs, "database") {
		if st, err := os.Stat(filepath.Join(root, dir)); err != nil || !st.IsDir() {
			t.Errorf("skeleton dir %s missing: %v", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "database", "zlanpiko.db")); err != nil {
		t.Errorf("database file missing: %v", err)
	}
	if _, err := os.Stat(config.UserPath(root)); err != nil {
		t.Errorf("default user.json missing: %v", err)
	}
	if err := ctx.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen keeps data.
	ctx2, err := Open(OpenOptions{DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	defer ctx2.Close()
	if _, err := ctx2.DB.Exec(`INSERT INTO units(id, name, created_at, updated_at) VALUES ('unit-001', 'X', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := ctx2.Close(); err != nil {
		t.Fatal(err)
	}
	ctx3, err := Open(OpenOptions{DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	defer ctx3.Close()
	var count int
	if err := ctx3.DB.QueryRow(`SELECT COUNT(*) FROM units`).Scan(&count); err != nil || count != 1 {
		t.Errorf("row lost across reopen: count=%d err=%v", count, err)
	}
}

func TestOpenUnconfiguredRootIsError(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir()) // empty: no pointer file
	t.Setenv(config.EnvDataRoot, "")
	if _, err := Open(OpenOptions{}); err == nil {
		t.Error("expected error when no data root is configured")
	}
}

func TestOpenEnvRoot(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	root := filepath.Join(t.TempDir(), "EnvData")
	t.Setenv(config.EnvDataRoot, root)
	ctx, err := Open(OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Close()
	if ctx.DataRoot != root {
		t.Errorf("DataRoot = %q, want %q", ctx.DataRoot, root)
	}
}
