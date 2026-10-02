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

// Topics manages reading progress: filter by status, cycle status, add,
// rename, move between units and delete (see docs/07).
type TopicsModel struct {
	shared  *Shared
	table   table.Model
	rows    []domain.Topic
	status  string // "" = all
	form    *components.Form
	confirm *components.Confirm
	pending [2]string // unit+topic awaiting delete confirmation
	formOp  string    // add | rename | move
	msg     string
}

// NewTopics builds the topics screen.
func NewTopics(shared *Shared) *TopicsModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "UNIT", Width: 10}, {Title: "ID", Width: 10},
			{Title: "NAME", Width: 26}, {Title: "STATUS", Width: 9},
			{Title: "PRI", Width: 7}, {Title: "ATTENTION", Width: 10},
		}),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	return &TopicsModel{shared: shared, table: t}
}

func (s *TopicsModel) ID() ScreenID { return Topics }

func (s *TopicsModel) SetSize(width, height int) {
	s.table.SetWidth(width)
	s.table.SetHeight(max(height-2, 3))
	s.table.SetStyles(styles.TableStyles())
}

// Reload lists the filtered unit's topics, or all units' when unfiltered.
func (s *TopicsModel) Reload() error {
	var all []domain.Topic
	if s.shared.TopicsUnit != "" {
		list, err := services.ListTopics(s.shared.Ctx.DB, s.shared.TopicsUnit, s.status)
		if err != nil {
			return err
		}
		all = list
	} else {
		units, err := services.ListUnits(s.shared.Ctx.DB, false)
		if err != nil {
			return err
		}
		for _, u := range units {
			list, err := services.ListTopics(s.shared.Ctx.DB, u.ID, s.status)
			if err != nil {
				return err
			}
			all = append(all, list...)
		}
	}
	s.rows = all
	// Nearest assessment per unit for the attention column (one query each).
	exams := map[string]*time.Time{}
	unitsSeen := map[string]bool{}
	for _, t := range all {
		unitsSeen[t.UnitID] = true
	}
	for unit := range unitsSeen {
		exam, err := analytics.NextAssessment(s.shared.Ctx.DB, unit, time.Now())
		if err != nil {
			return err
		}
		if exam != nil {
			exams[unit] = exam.DueAt
		}
	}
	rows := make([]table.Row, 0, len(all))
	for _, t := range all {
		att, _ := analytics.AssessTopic(t.ReadingStatus, t.Priority, exams[t.UnitID], time.Now())
		rows = append(rows, table.Row{t.UnitID, t.ID, t.Name, string(t.ReadingStatus), string(t.Priority), string(att)})
	}
	s.table.SetRows(rows)
	return nil
}

func (s *TopicsModel) selected() *domain.Topic {
	if len(s.rows) == 0 {
		return nil
	}
	i := s.table.Cursor()
	if i < 0 || i >= len(s.rows) {
		return nil
	}
	return &s.rows[i]
}

func (s *TopicsModel) title() string {
	if s.shared.TopicsUnit != "" {
		return "Topics of " + s.shared.TopicsUnit + "  (u: all units)"
	}
	return "Topics (all units)"
}

// Update routes keys to the dialog, form or table.
func (s *TopicsModel) Update(msg tea.Msg) tea.Cmd {
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
		s.status = nextStatusFilter(s.status)
		s.msg = "filter: " + orAll(s.status)
		return s.reloadCmd()
	case "u":
		s.shared.TopicsUnit = ""
		s.msg = "showing all units"
		return s.reloadCmd()
	case "x":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		next := map[domain.ReadingStatus]domain.ReadingStatus{
			domain.ReadingUnread:  domain.ReadingPending,
			domain.ReadingPending: domain.ReadingRead,
			domain.ReadingRead:    domain.ReadingUnread,
		}[sel.ReadingStatus]
		if _, err := services.SetTopicStatus(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), sel.UnitID, sel.ID, string(next)); err != nil {
			s.msg = err.Error()
			return nil
		}
		s.msg = fmt.Sprintf("%s/%s → %s", sel.UnitID, sel.ID, next)
		return s.reloadCmd()
	case "a":
		s.formOp = "add"
		s.form = components.NewForm("Add topic", []components.Field{
			{Label: "Unit", Initial: s.shared.TopicsUnit, Required: true},
			{Label: "Name", Required: true},
			{Label: "Priority"},
		})
		return nil
	case "r":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		s.formOp = "rename:" + sel.UnitID + "/" + sel.ID
		s.form = components.NewForm("Rename topic", []components.Field{
			{Label: "Name", Initial: sel.Name, Required: true},
		})
		return nil
	case "m":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		s.formOp = "move:" + sel.UnitID + "/" + sel.ID
		s.form = components.NewForm("Move topic to unit", []components.Field{
			{Label: "To unit", Required: true},
		})
		return nil
	case "D":
		sel := s.selected()
		if sel == nil {
			return nil
		}
		if err := services.DeleteTopic(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), sel.UnitID, sel.ID, false); err != nil {
			var cerr *domain.ConfirmRequiredError
			if errors.As(err, &cerr) {
				s.pending = [2]string{sel.UnitID, sel.ID}
				s.confirm = components.NewConfirm("Delete topic "+sel.UnitID+"/"+sel.ID+"?", cerr.Error())
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

func (s *TopicsModel) updateForm(msg tea.Msg) tea.Cmd {
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
	switch {
	case op == "add":
		_, err = services.CreateTopic(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), vals[0], vals[1], valAt(vals, 2))
	case strings.HasPrefix(op, "rename:"):
		ids := strings.Split(strings.TrimPrefix(op, "rename:"), "/")
		_, err = services.RenameTopic(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), ids[0], ids[1], vals[0])
	case strings.HasPrefix(op, "move:"):
		ids := strings.Split(strings.TrimPrefix(op, "move:"), "/")
		_, err = services.MoveTopic(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), ids[0], ids[1], vals[0])
	}
	if err != nil {
		s.msg = err.Error()
		return cmd
	}
	s.msg = "saved"
	return s.reloadCmd()
}

func (s *TopicsModel) updateConfirm(msg tea.Msg) tea.Cmd {
	cmd := s.confirm.Update(msg)
	if !s.confirm.Done {
		return cmd
	}
	accepted := s.confirm.Accepted
	unit, topic := s.pending[0], s.pending[1]
	s.confirm, s.pending = nil, [2]string{}
	if !accepted {
		s.msg = "cancelled"
		return cmd
	}
	if err := services.DeleteTopic(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), unit, topic, true); err != nil {
		s.msg = err.Error()
		return cmd
	}
	s.msg = "deleted " + topic
	return s.reloadCmd()
}

func (s *TopicsModel) reloadCmd() tea.Cmd {
	return func() tea.Msg {
		if err := s.Reload(); err != nil {
			s.msg = err.Error()
		}
		return nil
	}
}

// View renders the title, table and any active dialog.
func (s *TopicsModel) View(width, height int) string {
	s.SetSize(width, height)
	var b strings.Builder
	if s.form != nil {
		return s.form.View()
	}
	b.WriteString(styles.Header(s.title()) + "\n")
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

// Help lists topics keys.
func (s *TopicsModel) Help() string {
	return "x status · f filter · u all units · a add · r rename · m move · D delete"
}

func nextStatusFilter(cur string) string {
	switch cur {
	case "":
		return "unread"
	case "unread":
		return "pending"
	case "pending":
		return "read"
	default:
		return ""
	}
}

func orAll(s string) string {
	if s == "" {
		return "all"
	}
	return s
}
