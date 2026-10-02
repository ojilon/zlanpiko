package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"zlanpiko/internal/app"
	"zlanpiko/internal/config"
)

func TestRunVersionJSON(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"version", "--format", "json"}, &out, &errOut); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), `"schema_version":1`) {
		t.Errorf("version json = %q", out.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"version"}, &out, &errOut); code != 0 {
		t.Fatalf("version exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), app.Version) {
		t.Errorf("version output = %q, want it to contain %q", out.String(), app.Version)
	}
}

func TestRunHelpSpellings(t *testing.T) {
	for _, spelling := range []string{"help", "--help", "-h", "/help"} {
		var out, errOut bytes.Buffer
		if code := run([]string{spelling}, &out, &errOut); code != 0 {
			t.Errorf("%s exit = %d, want 0", spelling, code)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("%s output missing usage text: %q", spelling, out.String())
		}
	}
}

func TestRunNoArgsShowsHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(nil, &out, &errOut); code != 0 {
		t.Fatalf("no-args exit = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("no-args output missing usage text: %q", out.String())
	}
}

func TestRunUnknownCommandFailsUsage(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Data")
	t.Setenv(config.EnvDataRoot, root)
	var out, errOut bytes.Buffer
	if code := run([]string{"frobnicate"}, &out, &errOut); code != 2 {
		t.Fatalf("unknown exit = %d, want 2 (stderr: %q)", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Errorf("stderr = %q, want usage error", errOut.String())
	}
}

func TestExtractDataRoot(t *testing.T) {
	rest, root := extractDataRoot([]string{"units", "list", "--data-root", "P:/x", "--format", "json"})
	if root != "P:/x" {
		t.Errorf("root = %q", root)
	}
	if strings.Join(rest, " ") != "units list --format json" {
		t.Errorf("rest = %q", rest)
	}
	rest, root = extractDataRoot([]string{"units", "--data-root=P:/y", "list"})
	if root != "P:/y" || strings.Join(rest, " ") != "units list" {
		t.Errorf("root = %q rest = %q", root, rest)
	}
}

func TestRunStoragePathEndToEnd(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Data")
	t.Setenv(config.EnvDataRoot, root)
	var out, errOut bytes.Buffer
	if code := run([]string{"units", "add", "--name", "Physics"}, &out, &errOut); code != 0 {
		t.Fatalf("units add exit = %d (stderr: %q)", code, errOut.String())
	}
	out.Reset()
	if code := run([]string{"units", "list"}, &out, &errOut); code != 0 {
		t.Fatalf("units list exit = %d (stderr: %q)", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Physics") {
		t.Errorf("list output missing unit: %q", out.String())
	}
}
