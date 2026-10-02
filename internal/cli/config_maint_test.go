package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolateAppData(t *testing.T) {
	t.Helper()
	t.Setenv("APPDATA", t.TempDir())
}

func TestConfigShowSetPath(t *testing.T) {
	isolateAppData(t)
	ctx := testContext(t)
	out := mustOK(t, ctx, "config", "show")
	if !strings.Contains(out, ctx.DataRoot) || !strings.Contains(out, "auto") {
		t.Errorf("config show:\n%s", out)
	}
	mustOK(t, ctx, "config", "set", "theme", "16")
	out = mustOK(t, ctx, "config", "show", "--format", "json")
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("config json: %v", err)
	}
	if decoded["theme"] != "16" {
		t.Errorf("theme = %v", decoded["theme"])
	}
	if code, _, _ := runArgs(t, ctx, "config", "set", "theme", "neon"); code != ExitUsage {
		t.Errorf("bad theme exit = %d, want %d", code, ExitUsage)
	}
	if code, _, _ := runArgs(t, ctx, "config", "set", "bogus", "x"); code != ExitUsage {
		t.Errorf("bad key exit = %d, want %d", code, ExitUsage)
	}
	// data_root repoints the install pointer (isolated APPDATA here).
	other := filepath.Join(t.TempDir(), "Other")
	mustOK(t, ctx, "config", "set", "data_root", other)
	out = mustOK(t, ctx, "config", "path")
	if strings.TrimSpace(out) != ctx.DataRoot {
		t.Errorf("config path = %q, want current root %q", out, ctx.DataRoot)
	}
}

func TestMaintenanceVerifyAndRepair(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U")
	out := mustOK(t, ctx, "maintenance", "verify")
	if !strings.Contains(out, "verify: ok") {
		t.Errorf("clean verify:\n%s", out)
	}
	// Break a sidecar, then repair through the CLI.
	sidecar := filepath.Join(ctx.DataRoot, "units", "unit-001", "unit.json")
	raw, err := os.ReadFile(sidecar)
	if err != nil {
		t.Fatal(err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}
	asMap["updated_at"] = "2000-01-01T00:00:00Z"
	stale, _ := json.Marshal(asMap)
	if err := os.WriteFile(sidecar, stale, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runArgs(t, ctx, "maintenance", "verify")
	if code != ExitRuntime {
		t.Fatalf("dirty verify exit = %d, want %d", code, ExitRuntime)
	}
	if !strings.Contains(out, "stale") {
		t.Errorf("verify should report stale sidecar:\n%s", out)
	}
	out = mustOK(t, ctx, "maintenance", "verify", "--repair")
	if !strings.Contains(out, "verify: ok") {
		t.Errorf("after repair:\n%s", out)
	}
}

func TestStrictFlagsEverywhere(t *testing.T) {
	ctx := testContext(t)
	for _, args := range [][]string{
		{"units", "list", "--bogus"},
		{"topics", "list", "--unit", "unit-001", "--nope"},
		{"tasks", "list", "--what"},
		{"files", "list", "--nope"},
		{".", "--nope"},
		{"config", "show", "--nope"},
		{"maintenance", "verify", "--nope"},
		{"analytics", "--nope"},
		{"timeline", "--nope"},
	} {
		if code, _, _ := runArgs(t, ctx, args...); code != ExitUsage {
			t.Errorf("%v exit = %d, want %d", args, code, ExitUsage)
		}
	}
}
