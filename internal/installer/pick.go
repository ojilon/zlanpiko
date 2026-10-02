package installer

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/tui/styles"
)

// Picker steps.
type pickStep int

const (
	stepDrive pickStep = iota
	stepFolder
	stepManual
	stepConfirm
)

// Picker is the interactive parent-directory chooser: drive → folder (or
// drive root, or a typed name/subpath) → confirm. It returns the parent
// directory; the caller appends the app folder name.
type Picker struct {
	purpose   string
	appFolder string
	step      pickStep
	drives    []Drive
	cursor    int
	drive     string // e.g. "D:"
	folders   []string
	total     int
	fcursor   int
	input     textinput.Model
	inputErr  string
	parent    string
	aborted   bool
	done      bool
	status    string
	// driveRootOverride redirects drive listing for tests.
	driveRootOverride string
}

// root is the listed drive root (override in tests).
func (p *Picker) root() string {
	if p.driveRootOverride != "" {
		return p.driveRootOverride
	}
	return p.drive
}

func newPicker(purpose, appFolder string, drives []Drive) *Picker {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "folder name or subpath, e.g. Dev or Dev\\Projects"
	return &Picker{purpose: purpose, appFolder: appFolder, drives: drives, input: ti}
}

// PickParent runs the chooser on stdin/stdout and returns the parent dir.
func PickParent(purpose, appFolder string, stdin io.Reader, stdout io.Writer) (string, error) {
	drives, err := ListDrives()
	if err != nil {
		return "", err
	}
	p := newPicker(purpose, appFolder, drives)
	prog := tea.NewProgram(p, tea.WithInput(stdin), tea.WithOutput(stdout))
	final, err := prog.Run()
	if err != nil {
		return "", fmt.Errorf("installer: picker: %w", err)
	}
	got, ok := final.(*Picker)
	if !ok || got.aborted || !got.done {
		return "", fmt.Errorf("installer: folder selection cancelled")
	}
	return got.parent, nil
}

// Init implements tea.Model.
func (p *Picker) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (p *Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		if p.step == stepManual {
			var cmd tea.Cmd
			p.input, cmd = p.input.Update(msg)
			return p, cmd
		}
		return p, nil
	}
	switch p.step {
	case stepDrive:
		return p.updateDrive(key)
	case stepFolder:
		return p.updateFolder(key)
	case stepManual:
		return p.updateManual(key)
	case stepConfirm:
		return p.updateConfirm(key)
	}
	return p, nil
}

func (p *Picker) updateDrive(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "esc", "ctrl+c":
		p.aborted = true
		return p, tea.Quit
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
		}
		return p, nil
	case "down", "j":
		if p.cursor < len(p.drives)-1 {
			p.cursor++
		}
		return p, nil
	case "enter":
		d := p.drives[p.cursor]
		if !d.Ready {
			p.status = d.Letter + " is not ready"
			return p, nil
		}
		p.drive = d.Letter + `\`
		return p.enterFolders()
	default:
		// Digit quick-select.
		if len(key.String()) == 1 && key.String()[0] >= '1' && key.String()[0] <= '9' {
			i := int(key.String()[0] - '1')
			if i < len(p.drives) {
				p.cursor = i
				d := p.drives[i]
				if !d.Ready {
					p.status = d.Letter + " is not ready"
					return p, nil
				}
				p.drive = d.Letter + `\`
				return p.enterFolders()
			}
		}
		return p, nil
	}
}

func (p *Picker) enterFolders() (tea.Model, tea.Cmd) {
	folders, total, err := ListRootDirs(p.root(), folderPageSize)
	if err != nil {
		p.status = err.Error()
		return p, nil
	}
	p.folders, p.total, p.fcursor, p.step = folders, total, 0, stepFolder
	p.status = ""
	return p, nil
}

func (p *Picker) updateFolder(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "ctrl+c":
		p.aborted = true
		return p, tea.Quit
	case "esc":
		p.step = stepDrive
		return p, nil
	case "up", "k":
		if p.fcursor > 0 {
			p.fcursor--
		}
		return p, nil
	case "down", "j":
		if p.fcursor < len(p.folders)-1 {
			p.fcursor++
		}
		return p, nil
	case "enter":
		if len(p.folders) == 0 {
			p.status = "no folders to choose"
			return p, nil
		}
		p.parent = filepath.Join(p.drive, p.folders[p.fcursor])
		p.step = stepConfirm
		return p, nil
	case "r", "R":
		p.parent = p.root()
		p.step = stepConfirm
		return p, nil
	case "t", "T", "/":
		p.inputErr = ""
		p.input.SetValue("")
		p.step = stepManual
		return p, p.input.Focus()
	}
	return p, nil
}

func (p *Picker) updateManual(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		p.step = stepFolder
		p.input.Blur()
		return p, nil
	case "enter":
		abs, err := ResolveTypedChoice(p.root(), p.input.Value())
		if err != nil {
			p.inputErr = err.Error()
			return p, nil
		}
		p.parent = abs
		p.input.Blur()
		p.step = stepConfirm
		return p, nil
	}
	var cmd tea.Cmd
	p.input, cmd = p.input.Update(key)
	return p, cmd
}

func (p *Picker) updateConfirm(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "y", "Y", "enter":
		p.done = true
		return p, tea.Quit
	case "n", "N", "esc":
		p.step = stepFolder
		return p, nil
	case "q", "ctrl+c":
		p.aborted = true
		return p, tea.Quit
	}
	return p, nil
}

// finalPath is the app folder that will be created inside the parent.
func (p *Picker) finalPath() string {
	return filepath.Join(p.parent, p.appFolder)
}

// View implements tea.Model.
func (p *Picker) View() string {
	var b strings.Builder
	b.WriteString(styles.Header("Choose a location — "+p.purpose) + "\n")
	switch p.step {
	case stepDrive:
		b.WriteString(styles.Muted("Pick the drive that should hold the app folder.\n\n"))
		for i, d := range p.drives {
			marker := "  "
			if i == p.cursor {
				marker = styles.Accent("> ")
			}
			line := fmt.Sprintf("%d. %s  %s", i+1, d.Letter, d.Kind)
			if d.Ready {
				line += fmt.Sprintf("  %.1f GB free", d.FreeGB)
			} else {
				line += styles.Muted("  (not ready)")
			}
			b.WriteString(marker + line + "\n")
		}
		b.WriteString("\n" + styles.Muted("↑↓ move · 1-9 or enter select · q abort"))
	case stepFolder:
		shown := fmt.Sprintf("showing %d of %d", len(p.folders), p.total)
		if p.total > len(p.folders) {
			shown += " — type the rest by name with t"
		}
		b.WriteString(styles.Muted(fmt.Sprintf("Folders on %s (%s).\n\n", p.drive, shown)))
		for i, name := range p.folders {
			marker := "  "
			if i == p.fcursor {
				marker = styles.Accent("> ")
			}
			b.WriteString(marker + name + "\n")
		}
		b.WriteString("\n" + styles.Muted("↑↓ move · enter select · r drive root · t type a name · esc drives · q abort"))
	case stepManual:
		b.WriteString(styles.Muted(fmt.Sprintf("Type a folder on %s (must already exist).\n\n", p.drive)))
		b.WriteString(p.input.View() + "\n")
		if p.inputErr != "" {
			b.WriteString(styles.Error(p.inputErr) + "\n")
		}
		b.WriteString("\n" + styles.Muted("enter confirm · esc back"))
	case stepConfirm:
		b.WriteString(fmt.Sprintf("Create the app folder here?\n\n  %s\n\n", styles.Accent(p.finalPath())))
		b.WriteString(styles.Muted("y yes · n no · esc back"))
	}
	if p.status != "" {
		b.WriteString("\n" + styles.Error(p.status))
	}
	return b.String()
}
