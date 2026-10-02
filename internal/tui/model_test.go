package tui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/app"
	"zlanpiko/internal/tui/screens"
)

func testModel(t *testing.T) *Model {
	t.Helper()
	ctx, err := app.Open(app.OpenOptions{DataRoot: filepath.Join(t.TempDir(), "Data")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ctx.Close() })
	m := NewModel(ctx)
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init should return nil cmd (sync load)")
	}
	return m
}

func sendKey(t *testing.T, m *Model, key tea.KeyMsg) *Model {
	t.Helper()
	updated, _ := m.Update(key)
	out, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update returned %T", updated)
	}
	return out
}

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func specialKey(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func size(m *Model) *Model {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return updated.(*Model)
}

func TestTabNavigation(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("2"))
	if m.Active() != screens.Units {
		t.Errorf("active = %v, want Units", m.Active())
	}
	m = sendKey(t, m, runeKey("4"))
	if m.Active() != screens.Tasks {
		t.Errorf("active = %v, want Tasks", m.Active())
	}
}

func TestInputPersistsAcrossScreens(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("/"))
	m = sendKey(t, m, runeKey("u"))
	m = sendKey(t, m, runeKey("n"))
	m = sendKey(t, m, specialKey(tea.KeyEsc)) // blur, keep text
	if m.InputValue() != "un" {
		t.Fatalf("input = %q, want kept text", m.InputValue())
	}
	m = sendKey(t, m, runeKey("3")) // switch screen
	m = sendKey(t, m, runeKey("1")) // switch back
	if m.InputValue() != "un" {
		t.Errorf("input lost across screens: %q", m.InputValue())
	}
}

func TestCommandDispatch(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("/"))
	for _, r := range "topics unit-001" {
		m = sendKey(t, m, runeKey(string(r)))
	}
	m = sendKey(t, m, specialKey(tea.KeyEnter))
	if m.Active() != screens.Topics {
		t.Errorf("active = %v, want Topics", m.Active())
	}
	if m.shared.TopicsUnit != "unit-001" {
		t.Errorf("filter = %q", m.shared.TopicsUnit)
	}
	if m.InputValue() != "" {
		t.Errorf("input should clear after dispatch: %q", m.InputValue())
	}
}

func TestUnknownCommandSuggests(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("/"))
	for _, r := range "topix" {
		m = sendKey(t, m, runeKey(string(r)))
	}
	m = sendKey(t, m, specialKey(tea.KeyEnter))
	if !strings.Contains(m.Status(), "unknown command") {
		t.Errorf("status = %q", m.Status())
	}
	if !strings.Contains(m.Status(), "/topics") {
		t.Errorf("status should suggest /topics: %q", m.Status())
	}
}

func TestPhaseNotice(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("/"))
	for _, r := range "search" {
		m = sendKey(t, m, runeKey(string(r)))
	}
	m = sendKey(t, m, specialKey(tea.KeyEnter))
	if !strings.Contains(m.Status(), "later release") {
		t.Errorf("status = %q", m.Status())
	}
}

func TestHistoryNavigation(t *testing.T) {
	m := size(testModel(t))
	for _, cmd := range []string{"units", "tasks"} {
		m = sendKey(t, m, runeKey("/"))
		for _, r := range cmd {
			m = sendKey(t, m, runeKey(string(r)))
		}
		m = sendKey(t, m, specialKey(tea.KeyEnter))
	}
	m = sendKey(t, m, runeKey("/"))
	m = sendKey(t, m, specialKey(tea.KeyUp))
	if m.InputValue() != "tasks" {
		t.Errorf("history up = %q, want last command", m.InputValue())
	}
	m = sendKey(t, m, specialKey(tea.KeyUp))
	if m.InputValue() != "units" {
		t.Errorf("history up again = %q", m.InputValue())
	}
	m = sendKey(t, m, specialKey(tea.KeyDown))
	if m.InputValue() != "tasks" {
		t.Errorf("history down = %q", m.InputValue())
	}
}

func TestDashboardShowsSeededData(t *testing.T) {
	m := size(testModel(t))
	if _, err := m.shared.Ctx.DB.Exec(`INSERT INTO units(id, name, created_at, updated_at)
		VALUES ('unit-001', 'Physics', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	m = sendKey(t, m, runeKey("1"))
	view := m.View()
	if !strings.Contains(view, "Physics") {
		t.Errorf("dashboard missing unit:\n%s", view)
	}
	if !strings.Contains(view, "1 units") {
		t.Errorf("dashboard counts wrong:\n%s", view)
	}
}

func TestQuitCommand(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("/"))
	for _, r := range "quit" {
		m = sendKey(t, m, runeKey(string(r)))
	}
	_, cmd := m.Update(specialKey(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("expected a command from /quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg, got %T", cmd())
	}
}

func TestSmallTerminalMessage(t *testing.T) {
	m := testModel(t)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	view := updated.(*Model).View()
	if !strings.Contains(view, "80x24") {
		t.Errorf("small terminal should warn:\n%s", view)
	}
}
