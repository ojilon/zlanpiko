package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keyType(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func TestFormFillAndSubmit(t *testing.T) {
	f := NewForm("Add unit", []Field{{Label: "Name", Required: true}, {Label: "Code"}})
	f.Update(keyRunes("Maths"))
	if f.Done {
		t.Error("enter-less typing must not submit")
	}
	f.Update(keyType(tea.KeyEnter)) // move to Code
	f.Update(keyRunes("M1"))
	f.Update(keyType(tea.KeyEnter)) // submit
	if !f.Done || f.Cancelled {
		t.Errorf("form should be done: %+v", f)
	}
	vals := f.Values()
	if vals[0] != "Maths" || vals[1] != "M1" {
		t.Errorf("values = %q", vals)
	}
}

func TestFormRequiredBlocksSubmit(t *testing.T) {
	f := NewForm("Add", []Field{{Label: "Name", Required: true}})
	f.Update(keyType(tea.KeyEnter))
	if f.Done {
		t.Error("empty required field must block submit")
	}
	if !strings.Contains(f.View(), "required") {
		t.Errorf("view should show validation error:\n%s", f.View())
	}
	f.Update(keyType(tea.KeyEsc))
	if !f.Cancelled {
		t.Error("esc should cancel")
	}
}

func TestFormTabCycles(t *testing.T) {
	f := NewForm("F", []Field{{Label: "A"}, {Label: "B"}})
	f.Update(keyType(tea.KeyTab))
	f.Update(keyRunes("b-val"))
	f.Update(keyType(tea.KeyEnter))
	if !f.Done {
		t.Fatal("should submit from last field")
	}
	if vals := f.Values(); vals[1] != "b-val" {
		t.Errorf("values = %q", vals)
	}
}

func TestConfirmFlow(t *testing.T) {
	c := NewConfirm("Delete unit?", "1 topic")
	if c.Done {
		t.Error("should start pending")
	}
	c.Update(keyType(tea.KeyEnter)) // default No
	if !c.Done || c.Accepted || c.Cancelled {
		t.Errorf("enter-on-No should finish declined without cancel flag: %+v", c)
	}
	c2 := NewConfirm("Delete?", "")
	c2.Update(keyRunes("y"))
	if !c2.Done || !c2.Accepted {
		t.Errorf("y should accept: %+v", c2)
	}
	c3 := NewConfirm("Delete?", "")
	c3.Update(keyType(tea.KeyEsc))
	if !c3.Done || c3.Accepted || !c3.Cancelled {
		t.Errorf("esc should cancel: %+v", c3)
	}
	c4 := NewConfirm("Delete?", "")
	c4.Update(keyType(tea.KeyRight)) // toggle to Yes
	c4.Update(keyType(tea.KeyEnter))
	if !c4.Accepted {
		t.Errorf("toggled Yes + enter should accept: %+v", c4)
	}
}
