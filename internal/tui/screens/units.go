package screens

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/domain"
	"zlanpiko/internal/services"
	"zlanpiko/internal/tui/components"
	"zlanpiko/internal/tui/styles"
)

// Units lists course units with coverage and supports add/rename/archive/
// delete (see docs/07).
type UnitsModel struct {
	shared  *Shared
	table   table.Model
	rows    []analytics.UnitCoverage
	form    *components.Form
	confirm *components.Confirm
	pending string // id awaiting delete confirmation
	formOp  string // add | rename
	msg     string
	width   int
}

// NewUnits builds the units screen.
func NewUnits(shared *Shared) *UnitsModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "ID", Width: 10}, {Title: "NAME", Width: 26},
			{Title: "CODE", Width: 10}, {Title: "STATUS", Width: 9},
			{Title: "TOPICS", Width: 7}, {Title: "COVERAGE", Width: 9},
			{Title: "TASKS", Width: 6},
		}),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	return &UnitsModel{shared: shared, table: t}
}

func (s *UnitsModel) ID() ScreenID { return Units }

func (s *UnitsModel) SetSize(width, height int) {
	s.width = width
	s.table.SetWidth(width)
	s.table.SetHeight(max(height-2, 3))
	s.table.SetStyles(styles.TableStyles())
}

// Reload rebuilds the table from live rows.
func (s *UnitsModel) Reload() error {
	ucs, err := analytics.UnitCoverages(s.shared.Ctx.DB, true, time.Now())
	if err != nil {
		return err
	}
	s.rows = ucs
	rows := make([]table.Row, 0, len(ucs))
	for _, uc := range ucs {
		rows = append(rows, table.Row{uc.UnitID, uc.Name, uc.Code, string(uc.Status),
			fmt.Sprint(uc.Total), uc.CoverageText(), fmt.Sprint(uc.ActiveTasks)})
	}
	s.table.SetRows(rows)
	return nil
}

func (s *UnitsModel) selected() *analytics.UnitCoverage {
	if len(s.rows) == 0 {
		return nil
	}
	i := s.table.Cursor()
	if i < 0 || i >= len(s.rows) {
		return nil
	}
	return &s.rows[i]
}

// Update routes keys to the dialog, form or table.
func (s *UnitsModel) Update(msg tea.Msg) tea.Cmd {
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
	case "a":
		s.formOp = "add"
		s.form = components.NewForm("Add unit", []components.Field{
			{Label: "Name", Required: true}, {Label: "Code"}, {Label: "Colour"},
		})
		return nil
	case "r":
		sel := s.selected()
		if sel == nil {
			s.msg = "no unit selected"
			return nil
		}
		s.formOp = "rename:" + sel.UnitID
		s.form = components.NewForm("Rename unit", []components.Field{
			{Label: "Name", Initial: sel.Name, Required: true},
		})
		return nil
	case "A":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		u, err := services.GetUnit(s.shared.Ctx.DB, sel.UnitID)
		if err != nil {
			s.msg = err.Error()
			return nil
		}
		archived := u.Status == domain.UnitActive
		if _, err := services.SetUnitArchived(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), sel.UnitID, archived); err != nil {
			s.msg = err.Error()
			return nil
		}
		if archived {
			s.msg = "archived " + sel.UnitID
		} else {
			s.msg = "unarchived " + sel.UnitID
		}
		return s.reloadCmd()
	case "D":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		if sel.Total == 0 && sel.ActiveTasks == 0 {
			if err := services.DeleteUnit(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), sel.UnitID, false); err != nil {
				s.msg = err.Error()
				return nil
			}
			s.msg = "deleted " + sel.UnitID
			return s.reloadCmd()
		}
		s.pending = sel.UnitID
		s.confirm = components.NewConfirm("Delete unit "+sel.UnitID+"?",
			fmt.Sprintf("%d topics, %d active tasks — archive keeps history", sel.Total, sel.ActiveTasks))
		return nil
	case "enter":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		s.shared.TopicsUnit = sel.UnitID
		s.shared.NavTo = Topics
		s.shared.Flash = "topics of " + sel.Name
		return nil
	}
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	return cmd
}

func (s *UnitsModel) updateForm(msg tea.Msg) tea.Cmd {
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
	s.form = nil
	var err error
	if s.formOp == "add" {
		_, err = services.CreateUnit(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), vals[0], valAt(vals, 1), valAt(vals, 2), "")
	} else {
		id := strings.TrimPrefix(s.formOp, "rename:")
		_, err = services.RenameUnit(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), id, vals[0])
	}
	if err != nil {
		s.msg = err.Error()
		return cmd
	}
	s.msg = "saved"
	return s.reloadCmd()
}

func (s *UnitsModel) updateConfirm(msg tea.Msg) tea.Cmd {
	cmd := s.confirm.Update(msg)
	if !s.confirm.Done {
		return cmd
	}
	accepted := s.confirm.Accepted
	id := s.pending
	s.confirm, s.pending = nil, ""
	if !accepted {
		s.msg = "cancelled"
		return cmd
	}
	if err := services.DeleteUnit(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), id, true); err != nil {
		var cerr *domain.ConfirmRequiredError
		if errors.As(err, &cerr) {
			s.msg = cerr.Error()
			return cmd
		}
		s.msg = err.Error()
		return cmd
	}
	s.msg = "deleted " + id
	return s.reloadCmd()
}

func (s *UnitsModel) reloadCmd() tea.Cmd {
	return func() tea.Msg {
		if err := s.Reload(); err != nil {
			s.msg = err.Error()
		}
		return nil
	}
}

// View renders the table plus any active dialog.
func (s *UnitsModel) View(width, height int) string {
	s.SetSize(width, height)
	var b strings.Builder
	if s.form != nil {
		b.WriteString(s.form.View())
		return b.String()
	}
	if s.confirm != nil {
		b.WriteString(s.table.View() + "\n\n" + s.confirm.View())
		return b.String()
	}
	b.WriteString(s.table.View())
	if s.msg != "" {
		b.WriteString("\n" + styles.Muted(s.msg))
	}
	return b.String()
}

// Help lists units keys.
func (s *UnitsModel) Help() string {
	return "a add · r rename · A archive · D delete · enter topics"
}
