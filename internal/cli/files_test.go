package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// forceNonTTY pins the non-interactive confirmation path for tests.
func forceNonTTY(t *testing.T) {
	t.Helper()
	old := stdinIsTerminal
	stdinIsTerminal = func() bool { return false }
	t.Cleanup(func() { stdinIsTerminal = old })
}

func TestFilesListAndStat(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U")
	out := mustOK(t, ctx, "files", "list", "units/unit-001")
	if !strings.Contains(out, "topics/") || !strings.Contains(out, "unit.json") {
		t.Errorf("unit listing:\n%s", out)
	}
	out = mustOK(t, ctx, "files", "list", "units/unit-001/unit.json")
	if !strings.Contains(out, "unit.json") {
		t.Errorf("file stat:\n%s", out)
	}
	out = mustOK(t, ctx, "files", "list", "--format", "json")
	if !strings.Contains(out, `"name"`) {
		t.Errorf("json list:\n%s", out)
	}
}

func TestFilesMoveRenameDelete(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U")
	src := t.TempDir()
	writeFile(t, src, "doc.txt", "hello")
	mustOK(t, ctx, "files", "import", src+"/doc.txt", "--to", "inbox", "--yes")
	out := mustOK(t, ctx, "files", "list", "inbox")
	if !strings.Contains(out, "doc.txt") {
		t.Fatalf("imported file missing:\n%s", out)
	}
	mustOK(t, ctx, "files", "rename", "inbox/doc.txt", "renamed.txt")
	mustOK(t, ctx, "files", "move", "inbox/renamed.txt", "units/unit-001/notes")
	out = mustOK(t, ctx, "files", "search", "renamed")
	if !strings.Contains(out, "units/unit-001/notes/renamed.txt") {
		t.Errorf("search:\n%s", out)
	}
	// Delete requires --yes.
	if code, _, _ := runArgs(t, ctx, "files", "delete", "units/unit-001/notes/renamed.txt"); code != ExitUsage {
		t.Errorf("delete without --yes should be usage error, got %d", code)
	}
	mustOK(t, ctx, "files", "delete", "units/unit-001/notes/renamed.txt", "--yes")
	// Protected paths are refused.
	if code, _, _ := runArgs(t, ctx, "files", "delete", "units", "--recursive", "--yes"); code != ExitRuntime {
		t.Errorf("deleting units should fail, got %d", code)
	}
}

func TestImportPreviewRequiresConfirmation(t *testing.T) {
	forceNonTTY(t)
	ctx := testContext(t)
	src := t.TempDir()
	writeFile(t, src, "a.txt", "data")
	// No --yes on a non-TTY: preview prints, nothing imports.
	code, out, _ := runArgs(t, ctx, "files", "import", src+"/a.txt", "--to", "inbox")
	if code != ExitRuntime {
		t.Fatalf("exit = %d, want refusal %d", code, ExitRuntime)
	}
	if !strings.Contains(out, "import ") || !strings.Contains(out, "a.txt") {
		t.Errorf("preview should print first:\n%s", out)
	}
	out = mustOK(t, ctx, "files", "list", "inbox")
	if strings.Contains(out, "a.txt") {
		t.Errorf("nothing should import without confirmation:\n%s", out)
	}
	mustOK(t, ctx, "files", "import", src+"/a.txt", "--to", "inbox", "--yes")
	out = mustOK(t, ctx, "files", "list", "inbox")
	if !strings.Contains(out, "a.txt") {
		t.Errorf("file should import with --yes:\n%s", out)
	}
}

func TestDotImport(t *testing.T) {
	forceNonTTY(t)
	ctx := testContext(t)
	work := t.TempDir()
	writeFile(t, work, "handout.pdf", "pdf-bytes")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	// Preview-only without --yes.
	if code, _, _ := runArgs(t, ctx, "."); code != ExitRuntime {
		t.Errorf("dot without --yes should refuse, got %d", code)
	}
	mustOK(t, ctx, ".", "--to", "inbox", "--yes")
	out := mustOK(t, ctx, "files", "list", "inbox")
	if !strings.Contains(out, "handout.pdf") {
		t.Errorf("dot import should land in inbox:\n%s", out)
	}
}
