package screens

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/app"
)

func testShared(t *testing.T) *Shared {
	t.Helper()
	ctx, err := app.Open(app.OpenOptions{DataRoot: filepath.Join(t.TempDir(), "Data")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ctx.Close() })
	return &Shared{Ctx: ctx, NavTo: NoNav}
}

func seedUnit(t *testing.T, s *Shared, id, name string) {
	t.Helper()
	if _, err := s.Ctx.DB.Exec(`INSERT INTO units(id, name, created_at, updated_at)
		VALUES (?, ?, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`, id, name); err != nil {
		t.Fatal(err)
	}
}

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keyType(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func TestAllScreensReloadAndView(t *testing.T) {
	shared := testShared(t)
	seedUnit(t, shared, "unit-001", "Physics")
	all := []Screen{
		NewDashboard(shared), NewUnits(shared), NewTopics(shared),
		NewTasks(shared), NewFiles(shared), NewSettings(shared),
	}
	for _, sc := range all {
		if err := sc.Reload(); err != nil {
			t.Errorf("%v Reload: %v", sc.ID(), err)
		}
		view := sc.View(100, 28)
		if strings.TrimSpace(view) == "" {
			t.Errorf("%v View is empty", sc.ID())
		}
		if sc.Help() == "" {
			t.Errorf("%v Help is empty", sc.ID())
		}
	}
	if view := all[0].View(100, 28); !strings.Contains(view, "Physics") {
		t.Errorf("dashboard missing seeded unit:\n%s", view)
	}
}

func TestUnitsAddThroughForm(t *testing.T) {
	shared := testShared(t)
	u := NewUnits(shared)
	if err := u.Reload(); err != nil {
		t.Fatal(err)
	}
	u.Update(keyRunes("a"))
	for _, r := range "Chemistry" {
		u.Update(keyRunes(string(r)))
	}
	u.Update(keyType(tea.KeyEnter)) // Name -> Code
	u.Update(keyType(tea.KeyEnter)) // Code -> Colour
	u.Update(keyType(tea.KeyEnter)) // submit
	var count int
	if err := shared.Ctx.DB.QueryRow(`SELECT COUNT(*) FROM units WHERE name = 'Chemistry'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("unit not created: count=%d err=%v (msg=%q)", count, err, u.msg)
	}
}

func TestTopicsStatusCycleThroughKeys(t *testing.T) {
	shared := testShared(t)
	seedUnit(t, shared, "unit-001", "U")
	if _, err := shared.Ctx.DB.Exec(`INSERT INTO topics(unit_id, id, name, reading_status, created_at, updated_at)
		VALUES ('unit-001', 'topic-001', 'T', 'unread', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	tp := NewTopics(shared)
	if err := tp.Reload(); err != nil {
		t.Fatal(err)
	}
	tp.Update(keyRunes("x"))
	var status string
	if err := shared.Ctx.DB.QueryRow(`SELECT reading_status FROM topics`).Scan(&status); err != nil || status != "pending" {
		t.Errorf("status = %q, %v", status, err)
	}
}

func TestFilesBrowseRoot(t *testing.T) {
	shared := testShared(t)
	seedUnit(t, shared, "unit-001", "U")
	f := NewFiles(shared)
	if err := f.Reload(); err != nil {
		t.Fatal(err)
	}
	view := f.View(100, 28)
	for _, want := range []string{"units/", "inbox/"} {
		if !strings.Contains(view, want) {
			t.Errorf("files root should list %s:\n%s", want, view)
		}
	}
	// Descend into the last row (units/) via cursor moves.
	for f.table.Cursor() < len(f.rows)-1 {
		f.Update(keyType(tea.KeyDown))
	}
	f.Update(keyType(tea.KeyEnter))
	if f.rel != "units" {
		t.Errorf("rel = %q, want units", f.rel)
	}
	f.Update(keyType(tea.KeyBackspace))
	if f.rel != "" {
		t.Errorf("rel after up = %q, want root", f.rel)
	}
}
