// Package components holds reusable TUI widgets: multi-field forms and
// confirmation dialogs. They consume tea.Msg directly and are unit-tested
// without a terminal.
package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/tui/styles"
)

// Field is one form input.
type Field struct {
	Label    string
	Initial  string
	Required bool
}

// Form is a small multi-field prompt: tab cycles, enter submits on the last
// field (or moves next), esc cancels.
type Form struct {
	Title     string
	inputs    []textinput.Model
	labels    []string
	required  []bool
	focus     int
	Done      bool
	Cancelled bool
}

// NewForm builds a form with the first field focused.
func NewForm(title string, fields []Field) *Form {
	f := &Form{Title: title}
	for _, fld := range fields {
		ti := textinput.New()
		ti.Prompt = ""
		ti.SetValue(fld.Initial)
		f.inputs = append(f.inputs, ti)
		f.labels = append(f.labels, fld.Label)
		f.required = append(f.required, fld.Required)
	}
	if len(f.inputs) > 0 {
		f.inputs[0].Focus()
	}
	return f
}

// Update handles navigation and editing keys.
func (f *Form) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "esc":
		f.Cancelled = true
		return nil
	case "tab", "down":
		f.next()
		return nil
	case "shift+tab", "up":
		f.prev()
		return nil
	case "enter":
		if f.focus == len(f.inputs)-1 {
			if err := f.validate(); err == "" {
				f.Done = true
			}
			return nil
		}
		f.next()
		return nil
	}
	if f.focus < len(f.inputs) {
		var cmd tea.Cmd
		f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
		return cmd
	}
	return nil
}

func (f *Form) next() {
	f.inputs[f.focus].Blur()
	f.focus = (f.focus + 1) % len(f.inputs)
	f.inputs[f.focus].Focus()
}

func (f *Form) prev() {
	f.inputs[f.focus].Blur()
	f.focus = (f.focus - 1 + len(f.inputs)) % len(f.inputs)
	f.inputs[f.focus].Focus()
}

// validate reports the first missing required label, or "".
func (f *Form) validate() string {
	for i := range f.inputs {
		if f.required[i] && strings.TrimSpace(f.inputs[i].Value()) == "" {
			return f.labels[i]
		}
	}
	return ""
}

// ValidationError is the current blocking message, or "".
func (f *Form) ValidationError() string {
	if m := f.validate(); m != "" {
		return m + " is required"
	}
	return ""
}

// Values returns the submitted field values in order (valid after Done).
func (f *Form) Values() []string {
	out := make([]string, len(f.inputs))
	for i := range f.inputs {
		out[i] = strings.TrimSpace(f.inputs[i].Value())
	}
	return out
}

// View renders the form.
func (f *Form) View() string {
	var b strings.Builder
	b.WriteString(styles.Header(f.Title))
	b.WriteString("\n\n")
	for i := range f.inputs {
		marker := "  "
		if i == f.focus {
			marker = styles.Accent("> ")
		}
		b.WriteString(marker + styles.Muted(f.labels[i]+": ") + f.inputs[i].View() + "\n")
	}
	if msg := f.ValidationError(); msg != "" {
		b.WriteString("\n" + styles.Error(msg))
	}
	b.WriteString("\n" + styles.Muted("tab cycle · enter submit · esc cancel"))
	return b.String()
}
