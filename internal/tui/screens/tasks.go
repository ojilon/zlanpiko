package screens

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
	"zlanpiko/internal/tui/components"
	"zlanpiko/internal/tui/styles"
)

// task status filter cycle.
var taskFilters = []string{"", "not_started", "in_progress", "overdue", "completed", "submitted"}

// Tasks manages assignments, tests and deadlines: filter, add, edit,
// complete/submit and delete (see docs/07).
type TasksModel struct {
	shared  *Shared
	table   table.Model
	rows    []domain.Task
	status  string // "" = all
	form    *components.Form
	confirm *components.Confirm
	pending [2]string
	formOp  string // add | edit:unit/id
	msg     string
}

// NewTasks builds the tasks screen.
func NewTasks(shared *Shared) *TasksModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "UNIT", Width: 10}, {Title: "ID", Width: 10},
			{Title: "TITLE", Width: 26}, {Title: "KIND", Width: 11},
			{Title: "STATUS", Width: 12}, {Title: "DUE", Width: 16},
			{Title: "PRI", Width: 7},
		}),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	return &TasksModel{shared: shared, table: t}
}

func (s *TasksModel) ID() ScreenID { return Tasks }

func (s *TasksModel) SetSize(width, height int) {
	s.table.SetWidth(width)
	s.table.SetHeight(max(height-2, 3))
	s.table.SetStyles(styles.TableStyles())
}

// Reload lists tasks under the active filters.
func (s *TasksModel) Reload() error {
	now := time.Now()
	status := s.status
	if s.shared.TasksStatus != "" {
		status = s.shared.TasksStatus
	}
	list, err := tasks.ListTasks(s.shared.Ctx.DB, tasks.Filter{UnitID: s.shared.TasksUnit, Status: status}, now)
	if err != nil {
		return err
	}
	s.rows = list
	rows := make([]table.Row, 0, len(list))
	for _, t := range list {
		st := string(t.Status)
		if t.Overdue(now) {
			st = styles.Warn("overdue")
		}
		rows = append(rows, table.Row{t.UnitID, t.ID, t.Title, string(t.Kind), st, dueText(t.DueAt), string(t.Priority)})
	}
	s.table.SetRows(rows)
	return nil
}

func (s *TasksModel) selected() *domain.Task {
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
func (s *TasksModel) Update(msg tea.Msg) tea.Cmd {
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
	case "f":
		s.status = nextTaskFilter(s.status)
		s.shared.TasksStatus = ""
		s.msg = "filter: " + orAll(s.status)
		return s.reloadCmd()
	case "u":
		s.shared.TasksUnit = ""
		s.msg = "showing all units"
		return s.reloadCmd()
	case "a":
		s.formOp = "add"
		s.form = components.NewForm("Add task", []components.Field{
			{Label: "Unit", Initial: s.shared.TasksUnit, Required: true},
			{Label: "Title", Required: true},
			{Label: "Kind (assignment|coursework|test|examination|project|other)"},
			{Label: "Due (YYYY-MM-DD[THH:MM], empty = none)"},
			{Label: "Priority"},
		})
		return nil
	case "e":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		s.formOp = "edit:" + sel.UnitID + "/" + sel.ID
		due := ""
		if sel.DueAt != nil {
			due = sel.DueAt.Local().Format("2006-01-02T15:04")
		}
		s.form = components.NewForm("Edit task", []components.Field{
			{Label: "Title", Initial: sel.Title, Required: true},
			{Label: "Due (empty = none)", Initial: due},
			{Label: "Priority", Initial: string(sel.Priority)},
			{Label: "Description", Initial: sel.Description},
		})
		return nil
	case "c", "s":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		status := string(domain.TaskCompleted)
		if key.String() == "s" {
			status = string(domain.TaskSubmitted)
		}
		if _, err := tasks.CompleteTask(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), sel.UnitID, sel.ID, status); err != nil {
			s.msg = err.Error()
			return nil
		}
		s.msg = fmt.Sprintf("%s/%s → %s", sel.UnitID, sel.ID, status)
		return s.reloadCmd()
	case "D":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		if err := tasks.DeleteTask(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), sel.UnitID, sel.ID, false); err != nil {
			var cerr *domain.ConfirmRequiredError
			if errors.As(err, &cerr) {
				s.pending = [2]string{sel.UnitID, sel.ID}
				s.confirm = components.NewConfirm("Delete task "+sel.UnitID+"/"+sel.ID+"?", cerr.Error())
				return nil
			}
			s.msg = err.Error()
			return nil
		}
		s.msg = "deleted " + sel.ID
		return s.reloadCmd()
	}
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	return cmd
}

func (s *TasksModel) updateForm(msg tea.Msg) tea.Cmd {
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
	var err error
	if op == "add" {
		due, derr := tasks.ParseDue(valAt(vals, 3))
		if derr != nil {
			s.msg = derr.Error()
			return cmd
		}
		_, err = tasks.CreateTask(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared),
			vals[0], vals[1], valAt(vals, 2), due, valAt(vals, 4), "")
	} else {
		ids := strings.Split(strings.TrimPrefix(op, "edit:"), "/")
		due, derr := tasks.ParseDue(valAt(vals, 1))
		if derr != nil {
			s.msg = derr.Error()
			return cmd
		}
		title, desc := vals[0], valAt(vals, 3)
		up := tasks.Update{Title: &title, Description: &desc, SetDue: true, Due: due}
		if p := valAt(vals, 2); p != "" {
			up.Priority = &p
		}
		_, err = tasks.UpdateTask(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), ids[0], ids[1], up)
	}
	if err != nil {
		s.msg = err.Error()
		return cmd
	}
	s.msg = "saved"
	return s.reloadCmd()
}

func (s *TasksModel) updateConfirm(msg tea.Msg) tea.Cmd {
	cmd := s.confirm.Update(msg)
	if !s.confirm.Done {
		return cmd
	}
	accepted := s.confirm.Accepted
	unit, task := s.pending[0], s.pending[1]
	s.confirm, s.pending = nil, [2]string{}
	if !accepted {
		s.msg = "cancelled"
		return cmd
	}
	if err := tasks.DeleteTask(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), unit, task, true); err != nil {
		s.msg = err.Error()
		return cmd
	}
	s.msg = "deleted " + task
	return s.reloadCmd()
}

func (s *TasksModel) reloadCmd() tea.Cmd {
	return func() tea.Msg {
		if err := s.Reload(); err != nil {
			s.msg = err.Error()
		}
		return nil
	}
}

// View renders the table plus any active dialog.
func (s *TasksModel) View(width, height int) string {
	s.SetSize(width, height)
	var b strings.Builder
	if s.form != nil {
		return s.form.View()
	}
	title := styles.Header("Tasks")
	if s.shared.TasksUnit != "" {
		title += styles.Muted(" of " + s.shared.TasksUnit + "  (u: all units)")
	}
	b.WriteString(title + "\n")
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

// Help lists tasks keys.
func (s *TasksModel) Help() string {
	return "f filter · u all units · a add · e edit · c complete · s submit · D delete"
}

func nextTaskFilter(cur string) string {
	for i, f := range taskFilters {
		if f == cur {
			return taskFilters[(i+1)%len(taskFilters)]
		}
	}
	return ""
}
