package screens

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/filesystem"
	"zlanpiko/internal/importer"
	"zlanpiko/internal/tui/components"
	"zlanpiko/internal/tui/styles"
)

// importedMsg carries an async import result back to the Files screen.
type importedMsg struct {
	report *importer.Report
	err    error
}

// Files browses the data-root tree and runs imports, renames, opens and
// guarded deletes (see docs/07).
type FilesModel struct {
	shared  *Shared
	table   table.Model
	rel     string // current dir, "" = root
	rows    []filesystem.Info
	form    *components.Form
	confirm *components.Confirm
	formOp  string // import | rename
	pending string // path awaiting delete confirmation (+recursive marker)
	busy    bool   // async import running
	plan    *importer.Plan
	msg     string
}

// NewFiles builds the files screen.
func NewFiles(shared *Shared) *FilesModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "NAME", Width: 44}, {Title: "SIZE", Width: 10}, {Title: "MODIFIED", Width: 16},
		}),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	return &FilesModel{shared: shared, table: t}
}

func (s *FilesModel) ID() ScreenID { return Files }

func (s *FilesModel) SetSize(width, height int) {
	s.table.SetWidth(width)
	s.table.SetHeight(max(height-3, 3))
	s.table.SetStyles(styles.TableStyles())
}

// Reload lists the current directory.
func (s *FilesModel) Reload() error {
	children, err := filesystem.List(rootOf(s.shared), s.rel)
	if err != nil {
		return err
	}
	s.rows = children
	rows := make([]table.Row, 0, len(children))
	for _, c := range children {
		name := c.Name
		if c.IsDir {
			name += "/"
		}
		rows = append(rows, table.Row{name, fmt.Sprint(c.Size), c.ModTime.Format("2006-01-02 15:04")})
	}
	s.table.SetRows(rows)
	return nil
}

func (s *FilesModel) selected() *filesystem.Info {
	if len(s.rows) == 0 {
		return nil
	}
	i := s.table.Cursor()
	if i < 0 || i >= len(s.rows) {
		return nil
	}
	return &s.rows[i]
}

func (s *FilesModel) childRel(info filesystem.Info) string {
	if s.rel == "" {
		return info.Name
	}
	return s.rel + "/" + info.Name
}

// Update routes keys to dialogs, the form, or the browser.
func (s *FilesModel) Update(msg tea.Msg) tea.Cmd {
	if im, ok := msg.(importedMsg); ok {
		s.busy = false
		if im.err != nil {
			s.msg = im.err.Error()
			return nil
		}
		s.msg = fmt.Sprintf("imported %d, skipped %d, failed %d",
			im.report.Copied, im.report.Skipped, im.report.Failed)
		return s.reloadCmd()
	}
	if s.confirm != nil {
		return s.updateConfirm(msg)
	}
	if s.form != nil {
		return s.updateForm(msg)
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "enter":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		if !sel.IsDir {
			s.msg = "not a directory (o opens files)"
			return nil
		}
		s.rel = s.childRel(*sel)
		return s.reloadCmd()
	case "backspace", "left", "h":
		if s.rel == "" {
			return nil
		}
		s.rel = parentRel(s.rel)
		return s.reloadCmd()
	case "i":
		s.formOp = "import"
		s.form = components.NewForm("Import files", []components.Field{
			{Label: "Source path", Required: true},
			{Label: "Destination (empty = here)", Initial: s.rel},
			{Label: "Recursive (y/N)"},
		})
		return nil
	case "o":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		if sel.IsDir {
			s.msg = "enter descends into directories"
			return nil
		}
		if err := filesystem.Open(rootOf(s.shared), s.childRel(*sel)); err != nil {
			s.msg = err.Error()
			return nil
		}
		s.msg = "opened " + sel.Name
		return nil
	case "r":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		s.formOp = "rename:" + s.childRel(*sel)
		s.form = components.NewForm("Rename", []components.Field{
			{Label: "New name", Initial: sel.Name, Required: true},
		})
		return nil
	case "D":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		rel := s.childRel(*sel)
		kind := "file"
		if sel.IsDir {
			kind = "folder and everything inside it"
		}
		s.pending = rel
		s.confirm = components.NewConfirm("Delete "+kind+" "+rel+"?", "This cannot be undone")
		return nil
	}
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	return cmd
}

func (s *FilesModel) updateForm(msg tea.Msg) tea.Cmd {
	cmd := s.form.Update(msg)
	if s.form.Cancelled {
		s.form = nil
		s.msg = "cancelled"
		return cmd
	}
	if !s.form.Done {
		return cmd
	}
	vals := s.form.Values()
	op := s.formOp
	s.form = nil
	if strings.HasPrefix(op, "rename:") {
		rel := strings.TrimPrefix(op, "rename:")
		if err := filesystem.Rename(rootOf(s.shared), rel, vals[0]); err != nil {
			s.msg = err.Error()
			return cmd
		}
		s.msg = "renamed"
		return s.reloadCmd()
	}
	// Import: preview first, then confirm with counts (docs/11, D9).
	dest := vals[1]
	if dest == "" {
		dest = s.rel
	}
	opts := importer.Options{Dest: dest, Recursive: strings.ToLower(valAt(vals, 2)) == "y"}
	pv, err := importer.Preview(s.shared.Ctx.DB, rootOf(s.shared), filepath.Clean(vals[0]), opts)
	if err != nil {
		s.msg = err.Error()
		return cmd
	}
	if pv.Copies() == 0 {
		s.msg = "nothing to import (all skipped — see preview in CLI)"
		return cmd
	}
	s.plan = pv
	s.confirm = components.NewConfirm(
		fmt.Sprintf("Import %d file(s) to %s?", pv.Copies(), pv.DestDir),
		renameNote(pv))
	return cmd
}

func renameNote(pv *importer.Plan) string {
	renamed, skipped := 0, 0
	for _, pl := range pv.Plans {
		if pl.Renamed {
			renamed++
		}
		if pl.Action == "skip" {
			skipped++
		}
	}
	return fmt.Sprintf("%d renamed on collision, %d skipped", renamed, skipped)
}

func (s *FilesModel) updateConfirm(msg tea.Msg) tea.Cmd {
	// An import plan confirm executes asynchronously; a delete confirm runs now.
	isImport := s.plan != nil
	cmd := s.confirm.Update(msg)
	if !s.confirm.Done {
		return cmd
	}
	accepted := s.confirm.Accepted
	s.confirm = nil
	if !accepted {
		s.plan = nil
		s.msg = "cancelled"
		return cmd
	}
	if isImport {
		pv := s.plan
		s.plan = nil
		s.busy = true
		s.msg = "importing…"
		return func() tea.Msg {
			report, err := importer.Execute(context.Background(), s.shared.Ctx.DB, rootOf(s.shared), pv, logOf(s.shared))
			if err != nil {
				return importedMsg{err: err}
			}
			return importedMsg{report: report}
		}
	}
	rel := s.pending
	s.pending = ""
	info, _ := filesystem.Stat(rootOf(s.shared), rel)
	if err := filesystem.Delete(rootOf(s.shared), rel, info.IsDir); err != nil {
		s.msg = err.Error()
		return cmd
	}
	s.msg = "deleted " + rel
	return s.reloadCmd()
}

func (s *FilesModel) reloadCmd() tea.Cmd {
	return func() tea.Msg {
		if err := s.Reload(); err != nil {
			s.msg = err.Error()
		}
		return nil
	}
}

// View renders the browser plus any active dialog.
func (s *FilesModel) View(width, height int) string {
	s.SetSize(width, height)
	var b strings.Builder
	if s.form != nil {
		return s.form.View()
	}
	where := s.rel
	if where == "" {
		where = "(root)"
	}
	b.WriteString(styles.Header("Files  ") + styles.Muted(where) + "\n")
	if s.confirm != nil {
		b.WriteString(s.table.View() + "\n\n" + s.confirm.View())
		return b.String()
	}
	b.WriteString(s.table.View())
	status := s.msg
	if s.busy {
		status = "importing…"
	}
	if status != "" {
		b.WriteString("\n" + styles.Muted(status))
	}
	return b.String()
}

// Help lists files keys.
func (s *FilesModel) Help() string {
	return "enter open · ⌫ up · i import · o open file · r rename · D delete"
}

func parentRel(rel string) string {
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		return rel[:i]
	}
	return ""
}
