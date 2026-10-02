package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/tui/screens"
)

func TestTimelineScreenKeys(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("5"))
	if m.Active() != screens.Timeline {
		t.Fatalf("active = %v, want Timeline", m.Active())
	}
	view := m.View()
	if !strings.Contains(view, "Week ") {
		t.Errorf("timeline should show week title:\n%s", view)
	}
}

func TestTimelineCommandNav(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("/"))
	for _, r := range "timeline 2026-W41" {
		m = sendKey(t, m, runeKey(string(r)))
	}
	m = sendKey(t, m, specialKey(tea.KeyEnter))
	if m.Active() != screens.Timeline {
		t.Fatalf("active = %v", m.Active())
	}
	if view := m.View(); !strings.Contains(view, "Week 41") {
		t.Errorf("timeline should jump to week 41:\n%s", view)
	}
	m = sendKey(t, m, runeKey("/"))
	for _, r := range "timeline next" {
		m = sendKey(t, m, runeKey(string(r)))
	}
	m = sendKey(t, m, specialKey(tea.KeyEnter))
	if view := m.View(); !strings.Contains(view, "Week 42") {
		t.Errorf("timeline next should advance:\n%s", view)
	}
}

func TestAnalyticsCommandAndScreen(t *testing.T) {
	m := size(testModel(t))
	m = sendKey(t, m, runeKey("/"))
	for _, r := range "analytics" {
		m = sendKey(t, m, runeKey(string(r)))
	}
	m = sendKey(t, m, specialKey(tea.KeyEnter))
	if m.Active() != screens.Analytics {
		t.Fatalf("active = %v", m.Active())
	}
	if view := m.View(); !strings.Contains(view, "coverage = read") {
		t.Errorf("analytics should print formula:\n%s", view)
	}
}
