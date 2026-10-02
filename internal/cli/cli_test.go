package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"zlanpiko/internal/app"
)

// testContext opens an isolated app context for CLI tests.
func testContext(t *testing.T) *app.Context {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Data")
	ctx, err := app.Open(app.OpenOptions{DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ctx.Close() })
	return ctx
}

func runArgs(t *testing.T, ctx *app.Context, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(ctx, args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func mustOK(t *testing.T, ctx *app.Context, args ...string) string {
	t.Helper()
	code, out, errOut := runArgs(t, ctx, args...)
	if code != ExitOK {
		t.Fatalf("%v exit = %d (stderr: %q)", args, code, errOut)
	}
	return out
}

func TestUnitsAddListShowJSON(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "Algebra", "--code", "MATH201")
	out := mustOK(t, ctx, "units", "list")
	if !strings.Contains(out, "unit-001") || !strings.Contains(out, "Algebra") {
		t.Errorf("list output:\n%s", out)
	}
	out = mustOK(t, ctx, "units", "list", "--format", "json")
	if !strings.Contains(out, `"code": "MATH201"`) {
		t.Errorf("json output:\n%s", out)
	}
	out = mustOK(t, ctx, "units", "show", "--unit", "unit-001", "--format", "json")
	if !strings.Contains(out, `"topics": 0`) {
		t.Errorf("show output:\n%s", out)
	}
}

func TestUnitsArchiveDeleteFlow(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "Temp")
	mustOK(t, ctx, "units", "archive", "--unit", "unit-001")
	out := mustOK(t, ctx, "units", "list", "--archived")
	if !strings.Contains(out, "archived") {
		t.Errorf("archived unit should list with --archived:\n%s", out)
	}
	out = mustOK(t, ctx, "units", "list")
	if strings.Contains(out, "unit-001") {
		t.Errorf("archived unit should hide by default:\n%s", out)
	}
	mustOK(t, ctx, "units", "unarchive", "--unit", "unit-001")
	mustOK(t, ctx, "units", "delete", "--unit", "unit-001", "--yes")
}

func TestUnitsDeleteNeedsConfirm(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U")
	mustOK(t, ctx, "topics", "add", "--unit", "unit-001", "--name", "T")
	code, _, errOut := runArgs(t, ctx, "units", "delete", "--unit", "unit-001")
	if code != ExitRuntime {
		t.Fatalf("exit = %d, want %d", code, ExitRuntime)
	}
	if !strings.Contains(errOut, "confirm") {
		t.Errorf("stderr should mention confirmation: %q", errOut)
	}
}

func TestTopicsStatusAndMove(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U1")
	mustOK(t, ctx, "units", "add", "--name", "U2")
	mustOK(t, ctx, "topics", "add", "--unit", "unit-001", "--name", "T1")
	mustOK(t, ctx, "topics", "status", "--unit", "unit-001", "--topic", "topic-001", "--status", "read")
	out := mustOK(t, ctx, "topics", "list", "--unit", "unit-001", "--status", "read")
	if !strings.Contains(out, "T1") {
		t.Errorf("filtered list:\n%s", out)
	}
	mustOK(t, ctx, "topics", "move", "--unit", "unit-001", "--topic", "topic-001", "--to", "unit-002")
	out = mustOK(t, ctx, "topics", "list", "--unit", "unit-002")
	if !strings.Contains(out, "T1") {
		t.Errorf("moved topic should list in unit-002:\n%s", out)
	}
}

func TestTasksAddEditDeadlines(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U")
	mustOK(t, ctx, "tasks", "add", "--unit", "unit-001", "--title", "Past due",
		"--kind", "assignment", "--due", "2020-01-01")
	mustOK(t, ctx, "tasks", "add", "--unit", "unit-001", "--title", "Future",
		"--kind", "test", "--due", "2099-01-01")
	out := mustOK(t, ctx, "tasks", "deadlines", "--days", "36500")
	if !strings.Contains(out, "OVERDUE") || !strings.Contains(out, "Past due") {
		t.Errorf("deadlines should flag overdue:\n%s", out)
	}
	if !strings.Contains(out, "UPCOMING") || !strings.Contains(out, "Future") {
		t.Errorf("deadlines should list upcoming:\n%s", out)
	}
	mustOK(t, ctx, "tasks", "submit", "--unit", "unit-001", "--task", "task-001")
	out = mustOK(t, ctx, "tasks", "list", "--status", "submitted")
	if !strings.Contains(out, "Past due") {
		t.Errorf("submitted list:\n%s", out)
	}
	mustOK(t, ctx, "tasks", "edit", "--unit", "unit-001", "--task", "task-002",
		"--title", "Renamed", "--priority", "high")
	out = mustOK(t, ctx, "tasks", "list", "--format", "json")
	if !strings.Contains(out, "Renamed") {
		t.Errorf("edited title missing:\n%s", out)
	}
}

func TestUsageErrors(t *testing.T) {
	ctx := testContext(t)
	for _, args := range [][]string{
		{"bogus"},
		{"units"},
		{"units", "bogus"},
		{"units", "add"},
		{"tasks", "add", "--unit", "unit-001"},
		{"topics", "list", "--format", "yaml"},
	} {
		if code, _, _ := runArgs(t, ctx, args...); code != ExitUsage {
			t.Errorf("%v exit = %d, want %d", args, code, ExitUsage)
		}
	}
}

func TestMissingUnitIsRuntimeError(t *testing.T) {
	ctx := testContext(t)
	if code, _, errOut := runArgs(t, ctx, "units", "show", "--unit", "unit-999"); code != ExitRuntime {
		t.Errorf("exit = %d, want %d (%q)", code, ExitRuntime, errOut)
	}
}
