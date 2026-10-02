package main

import (
	"bytes"
	"strings"
	"testing"

	"zlanpiko/internal/app"
)

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
	var out, errOut bytes.Buffer
	if code := run([]string{"frobnicate"}, &out, &errOut); code != 2 {
		t.Fatalf("unknown exit = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "unknown command") {
		t.Errorf("stderr = %q, want usage error", errOut.String())
	}
}
