package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/tui/styles"
)

// Confirm is a yes/no dialog for destructive actions. Default choice is No;
// y accepts, n/esc cancels, enter confirms the highlighted choice.
type Confirm struct {
	Message   string
	Detail    string
	yes       bool
	Done      bool
	Cancelled bool
	Accepted  bool
}

// NewConfirm builds a dialog defaulting to No.
func NewConfirm(message, detail string) *Confirm {
	return &Confirm{Message: message, Detail: detail}
}

// Update handles toggle and decision keys.
func (c *Confirm) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "left", "h", "right", "l", "tab":
		c.yes = !c.yes
	case "y", "Y":
		c.yes = true
		c.finish(false)
	case "n", "N", "esc":
		c.finish(true)
	case "enter", " ":
		c.finish(false)
	}
	return nil
}

func (c *Confirm) finish(cancelled bool) {
	c.Done = true
	c.Cancelled = cancelled
	c.Accepted = !cancelled && c.yes
	if cancelled {
		c.Accepted = false
	}
}

// View renders the dialog.
func (c *Confirm) View() string {
	var b strings.Builder
	b.WriteString(styles.Warn(c.Message))
	if c.Detail != "" {
		b.WriteString("\n" + styles.Muted(c.Detail))
	}
	b.WriteString("\n\n")
	yes, no := "[Yes]", "[No]"
	if c.yes {
		yes = styles.Selected("[Yes]")
	} else {
		no = styles.Selected("[No]")
	}
	b.WriteString(yes + "  " + no)
	b.WriteString("\n" + styles.Muted("←/→ toggle · enter confirm · esc cancel"))
	return b.String()
}
