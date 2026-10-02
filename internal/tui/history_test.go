package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/app"
	"zlanpiko/internal/tui/screens"
)

func typeCommand(t *testing.T, m *Model, line string) *Model {
	t.Helper()
	m = sendKey(t, m, runeKey("/"))
	for _, r := range line {
		m = sendKey(t, m, runeKey(string(r)))
	}
	m = sendKey(t, m, specialKey(tea.KeyEnter))
	return m
}

func openCtx(t *testing.T, root string) *app.Context {
	t.Helper()
	ctx, err := app.Open(app.OpenOptions{DataRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ctx.Close() })
	return ctx
}

func TestVerifyCommand(t *testing.T) {
	m := size(testModel(t))
	m = typeCommand(t, m, "verify")
	if !strings.Contains(m.Status(), "verify: ok") {
		t.Errorf("status = %q", m.Status())
	}
}

func TestConfigCommandSwitchesToSettings(t *testing.T) {
	m := size(testModel(t))
	m = typeCommand(t, m, "config")
	if m.Active() != screens.Settings {
		t.Errorf("active = %v", m.Active())
	}
}

func TestHistoryPersistsAcrossModels(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Data")
	m1 := NewModel(openCtx(t, root))
	m1.Init()
	m1 = size(m1)
	m1 = typeCommand(t, m1, "units")
	m2 := NewModel(openCtx(t, root))
	m2.Init()
	m2 = size(m2)
	m2 = sendKey(t, m2, runeKey("/"))
	m2 = sendKey(t, m2, specialKey(tea.KeyUp))
	if m2.InputValue() != "units" {
		t.Errorf("recalled history = %q, want last command", m2.InputValue())
	}
}
