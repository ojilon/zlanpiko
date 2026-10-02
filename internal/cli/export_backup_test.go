package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportTextJSONAndOut(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U")
	out := mustOK(t, ctx, "export")
	if !strings.Contains(out, "ACADEMIC STATUS REPORT") {
		t.Errorf("export text:\n%s", out)
	}
	out = mustOK(t, ctx, "export", "--format", "json")
	if !strings.Contains(out, `"generated_at"`) {
		t.Errorf("export json:\n%s", out)
	}
	if code, _, _ := runArgs(t, ctx, "export", "--unit", "unit-999"); code != ExitRuntime {
		t.Errorf("bad unit exit = %d", code)
	}
	dest := filepath.Join(ctx.DataRoot, "exports", "r.txt")
	mustOK(t, ctx, "export", "--out", dest)
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("out file missing: %v", err)
	}
	if code, _, _ := runArgs(t, ctx, "export", "--out", dest); code != ExitRuntime {
		t.Errorf("overwrite exit = %d, want refusal", code)
	}
}

func TestBackupCreateListVerifyRestore(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U")
	mustOK(t, ctx, "topics", "add", "--unit", "unit-001", "--name", "T")
	out := mustOK(t, ctx, "backup", "create", "--full")
	if !strings.Contains(out, "backup created") {
		t.Errorf("create:\n%s", out)
	}
	out = mustOK(t, ctx, "backup", "list")
	if !strings.Contains(out, "full") {
		t.Errorf("list:\n%s", out)
	}
	matches, _ := filepath.Glob(filepath.Join(ctx.DataRoot, "backups", "*.zip"))
	if len(matches) != 1 {
		t.Fatalf("backups = %v", matches)
	}
	out = mustOK(t, ctx, "backup", "verify", matches[0])
	if !strings.Contains(out, "backup ok") {
		t.Errorf("verify:\n%s", out)
	}
	// Restore onto a non-empty target needs --overwrite-data (tested at the
	// package level); here restore refuses cleanly.
	if code, _, errOut := runArgs(t, ctx, "backup", "restore", matches[0], "--yes"); code != ExitRuntime {
		t.Errorf("restore exit = %d, want refusal (%q)", code, errOut)
	}
	if code, _, _ := runArgs(t, ctx, "backup", "restore", matches[0]); code != ExitUsage {
		t.Errorf("restore without --yes exit = %d", code)
	}
}
