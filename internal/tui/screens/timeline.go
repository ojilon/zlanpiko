package screens

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
	"zlanpiko/internal/timeline"
	"zlanpiko/internal/tui/components"
	"zlanpiko/internal/tui/styles"
)

// TimelineModel is a scrollable week view with an item detail overlay.
// n/p move weeks, t returns to today, enter opens details with inline
// deadline editing (see docs/07, docs/10).
type TimelineModel struct {
	shared  *Shared
	table   table.Model
	week    *timeline.Week
	items   []timeline.Item // parallel to table rows
	year    int
	wn      int
	detail  *timeline.Item
	form    *components.Form
	msg     string
	setWeek bool // year/wn explicitly set (command or t key)
}

// NewTimeline builds the timeline screen at the current week.
func NewTimeline(shared *Shared) *TimelineModel {
	y, w := timeline.WeekOf(time.Now())
	t := table.New(
		table.WithColumns(fullTimelineColumns()),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	return &TimelineModel{shared: shared, table: t, year: y, wn: w}
}

func fullTimelineColumns() []table.Column {
	return []table.Column{
		{Title: "DAY", Width: 12}, {Title: "TIME", Width: 6},
		{Title: "TITLE", Width: 28}, {Title: "UNIT", Width: 10},
		{Title: "KIND", Width: 11}, {Title: "STATUS", Width: 12},
	}
}

func compactTimelineColumns() []table.Column {
	return []table.Column{
		{Title: "DAY", Width: 12}, {Title: "TIME", Width: 6},
		{Title: "TITLE", Width: 30}, {Title: "STATUS", Width: 12},
	}
}

func (s *TimelineModel) ID() ScreenID { return Timeline }

// SetWeek jumps to an ISO week (used by /timeline).
func (s *TimelineModel) SetWeek(year, week int) {
	s.year, s.wn, s.setWeek = year, week, true
}

// GoToday returns to the current week.
func (s *TimelineModel) GoToday() {
	s.year, s.wn = timeline.WeekOf(time.Now())
}

// Shift moves the view by delta weeks (used by /timeline next|prev).
func (s *TimelineModel) Shift(delta int) {
	s.year, s.wn = timeline.Offset(s.year, s.wn, delta)
}

func (s *TimelineModel) SetSize(width, height int) {
	if width < 90 {
		s.table.SetColumns(compactTimelineColumns())
	} else {
		s.table.SetColumns(fullTimelineColumns())
	}
	s.table.SetWidth(width)
	s.table.SetHeight(max(height-3, 3))
	s.table.SetStyles(styles.TableStyles())
}

// Reload rebuilds the week view.
func (s *TimelineModel) Reload() error {
	w, err := timeline.Build(s.shared.Ctx.DB, s.year, s.wn, time.Now())
	if err != nil {
		return err
	}
	s.week = w
	s.items = nil
	rows := []table.Row{}
	compact := s.table.Columns() != nil && len(s.table.Columns()) == 4
	for _, it := range w.Overdue {
		s.items = append(s.items, it)
		rows = append(rows, s.row("OVERDUE", it, compact))
	}
	for _, d := range w.Days {
		label := d.Date.Format("Mon 01-02")
		for i, it := range d.Items {
			day := ""
			if i == 0 {
				day = label
			}
			s.items = append(s.items, it)
			rows = append(rows, s.row(day, it, compact))
		}
	}
	s.table.SetRows(rows)
	return nil
}

func (s *TimelineModel) row(day string, it timeline.Item, compact bool) table.Row {
	t := it.Task
	st := string(t.Status)
	if it.Overdue {
		st = "overdue"
	} else if t.Status == domain.TaskCompleted || t.Status == domain.TaskSubmitted {
		st = string(t.Status) + " ✓"
	}
	when := "-"
	if t.DueAt != nil {
		when = t.DueAt.Local().Format("15:04")
	}
	if compact {
		return table.Row{day, when, t.Title, st}
	}
	return table.Row{day, when, t.Title, t.UnitID, string(t.Kind), st}
}

func (s *TimelineModel) selected() *timeline.Item {
	if len(s.items) == 0 {
		return nil
	}
	i := s.table.Cursor()
	if i < 0 || i >= len(s.items) {
		return nil
	}
	return &s.items[i]
}

// Update handles navigation, detail overlay and inline edits.
func (s *TimelineModel) Update(msg tea.Msg) tea.Cmd {
	if s.form != nil {
		return s.updateForm(msg)
	}
	if s.detail != nil {
		return s.updateDetail(msg)
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "n":
		s.year, s.wn = timeline.Offset(s.year, s.wn, 1)
		return s.reloadCmd()
	case "p":
		s.year, s.wn = timeline.Offset(s.year, s.wn, -1)
		return s.reloadCmd()
	case "t":
		s.GoToday()
		return s.reloadCmd()
	case "enter":
		if s.selected() != nil {
			it := *s.selected()
			s.detail = &it
		}
		return nil
	}
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	return cmd
}

func (s *TimelineModel) updateDetail(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	t := s.detail.Task
	switch key.String() {
	case "esc":
		s.detail = nil
		return nil
	case "e":
		due := ""
		if t.DueAt != nil {
			due = t.DueAt.Local().Format("2006-01-02T15:04")
		}
		s.form = components.NewForm("Edit deadline", []components.Field{
			{Label: "Due (empty = none)", Initial: due},
		})
		return nil
	case "c":
		status := string(domain.TaskCompleted)
		if _, err := tasks.CompleteTask(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared), t.UnitID, t.ID, status); err != nil {
			s.msg = err.Error()
			return nil
		}
		s.detail = nil
		s.msg = "completed " + t.ID
		return s.reloadCmd()
	case "g":
		s.shared.TopicsUnit = t.UnitID
		s.shared.NavTo = Topics
		s.detail = nil
		return nil
	}
	return nil
}

func (s *TimelineModel) updateForm(msg tea.Msg) tea.Cmd {
	cmd := s.form.Update(msg)
	if s.form.Cancelled {
		s.form = nil
		return cmd
	}
	if !s.form.Done {
		return cmd
	}
	vals := s.form.Values()
	s.form = nil
	due, err := tasks.ParseDue(valAt(vals, 0))
	if err != nil {
		s.msg = err.Error()
		return cmd
	}
	t := s.detail.Task
	updated, err := tasks.UpdateTask(s.shared.Ctx.DB, rootOf(s.shared), logOf(s.shared),
		t.UnitID, t.ID, tasks.Update{SetDue: true, Due: due})
	if err != nil {
		s.msg = err.Error()
		return cmd
	}
	s.detail = &timeline.Item{Task: updated, Overdue: updated.Overdue(time.Now())}
	s.msg = "deadline updated"
	return s.reloadCmd()
}

func (s *TimelineModel) reloadCmd() tea.Cmd {
	return func() tea.Msg {
		if err := s.Reload(); err != nil {
			s.msg = err.Error()
		}
		return nil
	}
}

// View renders the week table or the detail overlay.
func (s *TimelineModel) View(width, height int) string {
	s.SetSize(width, height)
	var b strings.Builder
	if s.form != nil {
		return s.form.View()
	}
	if s.detail != nil {
		return s.detailView()
	}
	b.WriteString(styles.Header(timeline.Title(s.year, s.wn)) + styles.Muted("  (n/p weeks · t today)") + "\n")
	b.WriteString(s.table.View())
	if s.msg != "" {
		b.WriteString("\n" + styles.Muted(s.msg))
	}
	return b.String()
}

func (s *TimelineModel) detailView() string {
	t := s.detail.Task
	var b strings.Builder
	b.WriteString(styles.Header(t.Title) + "\n\n")
	b.WriteString(fmt.Sprintf("Unit     %s\nKind     %s\nStatus   %s\nDue      %s\nPriority %s\n",
		t.UnitID, t.Kind, statusText(t), dueText(t.DueAt), t.Priority))
	if t.Description != "" {
		b.WriteString("About    " + t.Description + "\n")
	}
	b.WriteString("\n" + styles.Muted("e edit deadline · c complete · g unit topics · esc back"))
	if s.msg != "" {
		b.WriteString("\n" + styles.Muted(s.msg))
	}
	return b.String()
}

func statusText(t domain.Task) string {
	if t.Overdue(time.Now()) {
		return "overdue"
	}
	return string(t.Status)
}

// Help lists timeline keys.
func (s *TimelineModel) Help() string {
	return "n/p weeks · t today · enter detail · e due · c complete · g unit"
}
