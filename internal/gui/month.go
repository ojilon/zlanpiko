package gui

import (
	"fmt"
	"regexp"
	"time"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
	"zlanpiko/internal/timeline"
)

// MonthDayDTO is one calendar grid cell. Leading/trailing filler days carry
// InMonth=false but real counts for continuity. Level is the worst same-day
// signal: bad (an overdue-anchored item originally due that day), warn (open
// item due within 48h), info (items present), ok (completed items only),
// none (empty). The month-wide overdue total rides on MonthDTO.OverdueCount
// (docs/cs/06).
type MonthDayDTO struct {
	Date    string `json:"date"`
	Display string `json:"display"`
	Count   int    `json:"count"`
	Level   string `json:"level"`
	InMonth bool   `json:"in_month"`
}

// MonthDTO is a Monday-start 6×7 grid covering the requested month.
type MonthDTO struct {
	Year         int           `json:"year"`
	Month        int           `json:"month"`
	Title        string        `json:"title"`
	Days         []MonthDayDTO `json:"days"`
	OverdueCount int           `json:"overdue_count"`
}

// monthCells returns the 42 Monday-start grid dates for a month.
func monthCells(year, month int) []time.Time {
	first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	wd := int(first.Weekday())
	if wd == 0 {
		wd = 7
	}
	start := first.AddDate(0, 0, -(wd - 1))
	out := make([]time.Time, 0, 42)
	for i := 0; i < 42; i++ {
		out = append(out, start.AddDate(0, 0, i))
	}
	return out
}

// GetMonth buckets the ISO weeks overlapping the month via timeline.Build
// (no new calendar math; see docs/cs/06). Month 1–12, year ≥ 1.
func (g *GuiApi) GetMonth(year, month int) (MonthDTO, error) {
	now := g.now()
	if year < 1 || month < 1 || month > 12 {
		return MonthDTO{}, fail(ErrValidation, fmt.Sprintf("invalid month %d-%02d", year, month), "want YYYY-MM, e.g. 2026-10")
	}
	cells := monthCells(year, month)
	type mark struct {
		soon bool
		done bool
		bad  bool
	}
	byDay := make(map[string][]mark)
	addMark := func(k string, mk mark) { byDay[k] = append(byDay[k], mk) }
	seenWeek := map[string]bool{}
	seenOverdue := map[string]bool{}
	for _, c := range cells {
		y, w := timeline.WeekOf(c)
		key := fmt.Sprintf("%d-W%02d", y, w)
		if seenWeek[key] {
			continue
		}
		seenWeek[key] = true
		built, err := timeline.Build(g.ctx.DB, y, w, now)
		if err != nil {
			return MonthDTO{}, fail(ErrDB, "build week", err.Error())
		}
		for _, d := range built.Days {
			k := d.Date.Format("2006-01-02")
			for _, it := range d.Items {
				t := it.Task
				open := t.Status != domain.TaskCompleted && t.Status != domain.TaskSubmitted
				soon := false
				if open && t.DueAt != nil && t.DueAt.Sub(now) <= 48*time.Hour {
					soon = true
				}
				addMark(k, mark{soon: soon, done: !open})
			}
		}
		// Overdue-anchored items keep a red mark on their original day so
		// the calendar shows deadline density (docs/cs/06). They repeat in
		// every week's anchor section, so dedupe by task identity.
		for _, it := range built.Overdue {
			if it.Task.DueAt == nil {
				continue
			}
			id := it.Task.UnitID + "/" + it.Task.ID
			if seenOverdue[id] {
				continue
			}
			seenOverdue[id] = true
			addMark(it.Task.DueAt.Local().Format("2006-01-02"), mark{bad: true})
		}
	}
	overdue, err := g.weekOverdueCount(now)
	if err != nil {
		return MonthDTO{}, err
	}
	days := make([]MonthDayDTO, 0, 42)
	for _, c := range cells {
		k := c.Format("2006-01-02")
		items := byDay[k]
		level := "none"
		for _, it := range items {
			if it.bad {
				level = "bad"
				break
			}
			if it.soon {
				level = "warn"
				break
			}
		}
		if level == "none" && len(items) > 0 {
			level = "info"
			onlyDone := true
			for _, it := range items {
				if !it.done {
					onlyDone = false
					break
				}
			}
			if onlyDone {
				level = "ok"
			}
		}
		days = append(days, MonthDayDTO{
			Date:    k,
			Display: c.Format("02"),
			Count:   len(items),
			Level:   level,
			InMonth: int(c.Month()) == month,
		})
	}
	return MonthDTO{
		Year:         year,
		Month:        month,
		Title:        time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local).Format("January 2006"),
		Days:         days,
		OverdueCount: overdue,
	}, nil
}

// weekOverdueCount counts currently overdue open tasks.
func (g *GuiApi) weekOverdueCount(now time.Time) (int, error) {
	overdue, err := tasks.ListTasks(g.ctx.DB, tasks.Filter{Status: domain.TaskOverdue}, now)
	if err != nil {
		return 0, fail(ErrDB, "read overdue", err.Error())
	}
	return len(overdue), nil
}

// DayItemsDTO is one day's agenda: its week-bucket items plus any
// overdue-anchored items originally due that date.
type DayItemsDTO struct {
	Date    string    `json:"date"`
	Display string    `json:"display"`
	Items   []TaskDTO `json:"items"`
	Overdue []TaskDTO `json:"overdue"`
}

// dayDateRe validates YYYY-MM-DD for GetDayItems.
var dayDateRe = func() *regexp.Regexp { return regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`) }()

// GetDayItems returns a single day's agenda, keeping week/day bucketing in
// Go so calendar clicks need no date arithmetic in TypeScript (docs/cs/06).
func (g *GuiApi) GetDayItems(dateISO string) (DayItemsDTO, error) {
	now := g.now()
	m := dayDateRe.FindStringSubmatch(dateISO)
	if m == nil {
		return DayItemsDTO{}, fail(ErrValidation, "invalid date "+dateISO, "want YYYY-MM-DD, e.g. 2026-10-06")
	}
	day := time.Date(atoi(m[1]), time.Month(atoi(m[2])), atoi(m[3]), 0, 0, 0, 0, time.Local)
	if day.Format("2006-01-02") != dateISO {
		return DayItemsDTO{}, fail(ErrValidation, "invalid date "+dateISO, "want a real calendar day")
	}
	y, w := timeline.WeekOf(day)
	built, err := timeline.Build(g.ctx.DB, y, w, now)
	if err != nil {
		return DayItemsDTO{}, fail(ErrDB, "build week", err.Error())
	}
	names, err := g.unitNames()
	if err != nil {
		return DayItemsDTO{}, fail(ErrDB, "read unit names", err.Error())
	}
	out := DayItemsDTO{
		Date:    dateISO,
		Display: day.Format("Mon 2006-01-02"),
		Items:   []TaskDTO{},
		Overdue: []TaskDTO{},
	}
	for _, d := range built.Days {
		if d.Date.Format("2006-01-02") != dateISO {
			continue
		}
		for _, it := range d.Items {
			out.Items = append(out.Items, toTaskDTO(it.Task, names[it.Task.UnitID], now))
		}
	}
	for _, it := range built.Overdue {
		if it.Task.DueAt != nil && it.Task.DueAt.Local().Format("2006-01-02") == dateISO {
			out.Overdue = append(out.Overdue, toTaskDTO(it.Task, names[it.Task.UnitID], now))
		}
	}
	return out, nil
}

// atoi parses a validated decimal field (regex guarantees digits).
func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// GetMonthOffset returns the month N months from the current one (negative
// allowed), keeping month arithmetic in Go (docs/cs/06).
func (g *GuiApi) GetMonthOffset(offset int) (MonthDTO, error) {
	now := g.now()
	base := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, offset, 0)
	return g.GetMonth(base.Year(), int(base.Month()))
}
