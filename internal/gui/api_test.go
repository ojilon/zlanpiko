package gui

import (
	"strings"
	"testing"

	"zlanpiko/internal/app"
)

// The GUI entry constructs its context via app.Open (see main.go); unit
// tests use a bare context because Phase A methods touch no storage.
func TestGetVersion(t *testing.T) {
	api := NewGuiApi(&app.Context{})
	v, err := api.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if !strings.Contains(v, app.Name) {
		t.Fatalf("GetVersion %q does not mention %q", v, app.Name)
	}
}

func TestGetDashboardPlaceholder(t *testing.T) {
	api := NewGuiApi(&app.Context{})
	d, err := api.GetDashboard()
	if err != nil {
		t.Fatalf("GetDashboard: %v", err)
	}
	if !strings.Contains(d.Version, app.Name) {
		t.Fatalf("dashboard version %q does not mention %q", d.Version, app.Name)
	}
}

func TestGuiErrorText(t *testing.T) {
	e := fail(ErrValidation, "bad status", "want unread|pending|read")
	if !strings.Contains(e.Error(), "VALIDATION") || !strings.Contains(e.Error(), "bad status") {
		t.Fatalf("unexpected error text %q", e.Error())
	}
}
