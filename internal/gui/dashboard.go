package gui

import (
	"fmt"
	"time"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/app"
	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
	"zlanpiko/internal/timeline"
)

// upcomingDays is the dashboard horizon: tasks due within this window
// (open only) feed the Upcoming list and the 14-day chart (docs/cs/05).
const upcomingDays = 14

// formatDue renders a local "2026-10-06 00:00" display string.
func formatDue(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}

// distance renders "today", "tomorrow", "in Nd", "in Nh" or "overdue by …"
// plus whole days left (negative = overdue). Dateless tasks report "no date".
func distance(now time.Time, due *time.Time) (string, *int) {
	if due == nil {
		return "no date", nil
	}
	d := due.Sub(now)
	days := int(d.Hours() / 24)
	if d < 0 {
		over := now.Sub(*due)
		od := int(over.Hours() / 24)
		if od < 1 {
			oh := int(over.Hours())
			if oh < 1 {
				return "overdue by minutes", &days
			}
			return fmt.Sprintf("overdue by %dh", oh), &days
		}
		return fmt.Sprintf("overdue by %dd", od), &days
	}
	y, m, day := now.Local().Date()
	ty, tm, tday := due.Local().Date()
	switch {
	case y == ty && m == tm && day == tday:
		return "today", &days
	case due.Local().Sub(time.Date(y, m, day, 0, 0, 0, 0, time.Local)) < 48*time.Hour:
		return "tomorrow", &days
	case d < 48*time.Hour:
		return fmt.Sprintf("in %dh", int(d.Hours())), &days
	default:
		return fmt.Sprintf("in %dd", days), &days
	}
}

// progress renders the created→due elapsed share, clamped to [0,100].
// Single-day (or inverted) ranges and dateless tasks report nil so the UI
// shows "due today"/"no date" instead of a misleading bar (docs/cs/05).
func progress(created time.Time, due *time.Time, now time.Time) *float64 {
	if due == nil || !due.After(created.Add(24*time.Hour)) {
		return nil
	}
	p := float64(now.Sub(created)) / float64(due.Sub(created)) * 100
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return &p
}

func toTaskDTO(t domain.Task, unitName string, now time.Time) TaskDTO {
	overdue := t.Overdue(now)
	d := TaskDTO{
		ID:       t.ID,
		UnitID:   t.UnitID,
		UnitName: unitName,
		Title:    t.Title,
		Kind:     t.Kind,
		Status:   t.Status,
		Overdue:  overdue,
		Priority: t.Priority,
	}
	// Finished work is never described with overdue language, even when its
	// due date lies in the past (docs/cs/05).
	if t.Status == domain.TaskCompleted || t.Status == domain.TaskSubmitted {
		d.Distance = "done"
	} else {
		dist, days := distance(now, t.DueAt)
		d.Distance = dist
		d.DaysLeft = days
	}
	if t.DueAt != nil {
		local := t.DueAt.Local()
		d.DueISO = &local
		d.DueDisplay = formatDue(t.DueAt)
		d.ProgressPct = progress(t.CreatedAt, t.DueAt, now)
	}
	return d
}

// unitNames maps unit IDs to display names for task attribution.
func (g *GuiApi) unitNames() (map[string]string, error) {
	ucs, err := analytics.UnitCoverages(g.ctx.DB, false, g.now())
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(ucs))
	for _, uc := range ucs {
		out[uc.UnitID] = uc.Name
	}
	return out, nil
}

// GetUnitCards returns per-unit dashboard cards ordered by name (the same
// order as analytics.UnitCoverages).
func (g *GuiApi) GetUnitCards() ([]UnitCardDTO, error) {
	now := g.now()
	ucs, err := analytics.UnitCoverages(g.ctx.DB, false, now)
	if err != nil {
		return nil, fail(ErrDB, "read unit coverages", err.Error())
	}
	out := make([]UnitCardDTO, 0, len(ucs))
	for _, uc := range ucs {
		label, reason, err := analytics.AssessUnit(g.ctx.DB, uc, now)
		if err != nil {
			return nil, fail(ErrDB, "assess unit "+uc.UnitID, err.Error())
		}
		nextDue := ""
		var nextDays *int
		if uc.NextDue != nil {
			nextDue = formatDue(uc.NextDue)
			_, days := distance(now, uc.NextDue)
			nextDays = days
		}
		out = append(out, UnitCardDTO{
			UnitID:         uc.UnitID,
			Name:           uc.Name,
			Code:           uc.Code,
			Total:          uc.Total,
			Read:           uc.Read,
			Pending:        uc.Pending,
			Unread:         uc.Unread,
			Coverage:       uc.Coverage,
			CoverageText:   uc.CoverageText(),
			HasTopics:      uc.HasTopics,
			ActiveTasks:    uc.ActiveTasks,
			CompletedTasks: uc.CompletedTasks,
			NextDueDisplay: nextDue,
			NextDueDays:    nextDays,
			Attention:      string(label),
			Reason:         reason,
		})
	}
	return out, nil
}

// GetWeek returns the requested ISO week reshaped for the rail/timeline.
// Year/week 0/0 means the current week.
func (g *GuiApi) GetWeek(year, week int) (WeekDTO, error) {
	now := g.now()
	if year == 0 && week == 0 {
		year, week = timeline.WeekOf(now)
	}
	if year < 1 || week < 1 || week > 53 {
		return WeekDTO{}, fail(ErrValidation, fmt.Sprintf("invalid week %d-W%02d", year, week), "want YYYY-Www, e.g. 2026-W41")
	}
	w, err := timeline.Build(g.ctx.DB, year, week, now)
	if err != nil {
		return WeekDTO{}, fail(ErrDB, "build week", err.Error())
	}
	names, err := g.unitNames()
	if err != nil {
		return WeekDTO{}, fail(ErrDB, "read unit names", err.Error())
	}
	dto := WeekDTO{
		Year:    w.Year,
		Week:    w.Week,
		Title:   timeline.Title(w.Year, w.Week),
		Monday:  w.Monday.Format("2006-01-02"),
		Days:    make([]DayDTO, 0, len(w.Days)),
		Overdue: make([]TaskDTO, 0, len(w.Overdue)),
	}
	for _, d := range w.Days {
		items := make([]TaskDTO, 0, len(d.Items))
		for _, it := range d.Items {
			items = append(items, toTaskDTO(it.Task, names[it.Task.UnitID], now))
		}
		dto.Days = append(dto.Days, DayDTO{
			Date:    d.Date.Format("2006-01-02"),
			Display: d.Date.Format("Mon 02"),
			Items:   items,
		})
	}
	for _, it := range w.Overdue {
		dto.Overdue = append(dto.Overdue, toTaskDTO(it.Task, names[it.Task.UnitID], now))
	}
	return dto, nil
}

// GetWeekOffset returns the week N weeks from the current one (negative
// allowed). It keeps ISO-week arithmetic in Go so the rail can span a
// fortnight without any date math in TypeScript (docs/cs/05).
func (g *GuiApi) GetWeekOffset(offset int) (WeekDTO, error) {
	now := g.now()
	y, w := timeline.WeekOf(now)
	if offset != 0 {
		y, w = timeline.Offset(y, w, offset)
	}
	return g.GetWeek(y, w)
}

// GetDashboard boots the home view in one call: summary, unit cards,
// 14-day upcoming, overdue and the current week rail.
func (g *GuiApi) GetDashboard() (DashboardDTO, error) {
	now := g.now()
	ucs, err := analytics.UnitCoverages(g.ctx.DB, false, now)
	if err != nil {
		return DashboardDTO{}, fail(ErrDB, "read unit coverages", err.Error())
	}
	sum, err := analytics.Summarize(g.ctx.DB, ucs, now)
	if err != nil {
		return DashboardDTO{}, fail(ErrDB, "summarize", err.Error())
	}
	cards, err := g.GetUnitCards()
	if err != nil {
		return DashboardDTO{}, err
	}
	upcoming, err := analytics.Upcoming(g.ctx.DB, now, upcomingDays)
	if err != nil {
		return DashboardDTO{}, fail(ErrDB, "read upcoming", err.Error())
	}
	overdueTasks, err := tasks.ListTasks(g.ctx.DB, tasks.Filter{Status: domain.TaskOverdue}, now)
	if err != nil {
		return DashboardDTO{}, fail(ErrDB, "read overdue", err.Error())
	}
	names := make(map[string]string, len(ucs))
	for _, uc := range ucs {
		names[uc.UnitID] = uc.Name
	}
	up := make([]TaskDTO, 0, len(upcoming))
	for _, t := range upcoming {
		up = append(up, toTaskDTO(t, names[t.UnitID], now))
	}
	over := make([]TaskDTO, 0, len(overdueTasks))
	for _, t := range overdueTasks {
		over = append(over, toTaskDTO(t, names[t.UnitID], now))
	}
	y, w := timeline.WeekOf(now)
	week, err := g.GetWeek(y, w)
	if err != nil {
		return DashboardDTO{}, err
	}
	return DashboardDTO{
		Version: app.Info(),
		Summary: SummaryDTO{
			Units:          sum.Units,
			TotalTopics:    sum.TotalTopics,
			Read:           sum.Read,
			Pending:        sum.Pending,
			Unread:         sum.Unread,
			Coverage:       sum.Coverage,
			CoverageText:   sum.CoverageText(),
			HasTopics:      sum.HasTopics,
			ActiveTasks:    sum.ActiveTasks,
			CompletedTasks: sum.CompletedTasks,
			OverdueTasks:   sum.OverdueTasks,
		},
		Units:    cards,
		Upcoming: up,
		Overdue:  over,
		Week:     week,
	}, nil
}
