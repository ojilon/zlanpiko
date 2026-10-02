// Package timeline owns all week/day date arithmetic and projects task rows
// into weekly buckets. Nothing else in the codebase computes weeks, day
// boundaries or timeline sorting (see docs/10).
//
// Weeks are ISO weeks in local time. Overdue incomplete items anchor to a
// leading Overdue section (not their original day) so they stay visible;
// completed items render in their day, dimmed by the fronts. Dateless tasks
// never appear in buckets (see docs/09).
package timeline

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
)

// Item is one task placed on the timeline.
type Item struct {
	Task    domain.Task `json:"task"`
	Overdue bool        `json:"overdue"`
}

// Day is one local-midnight-to-midnight bucket.
type Day struct {
	Date  time.Time `json:"date"`
	Items []Item    `json:"items"`
}

// Week is a built Monday–Sunday view plus the overdue anchor section.
type Week struct {
	Year    int       `json:"year"`
	Week    int       `json:"week"`
	Monday  time.Time `json:"monday"`
	Days    []Day     `json:"days"`
	Overdue []Item    `json:"overdue"`
}

// WeekOf returns the ISO year/week of t.
func WeekOf(t time.Time) (year, week int) {
	return t.ISOWeek()
}

// MondayOf returns the local-midnight Monday of an ISO week.
func MondayOf(year, week int) time.Time {
	// Jan 4 is always in ISO week 1; rewind to its Monday, then advance.
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)
	wd := int(jan4.Weekday())
	if wd == 0 {
		wd = 7 // Sunday -> 7
	}
	monday := time.Date(jan4.Year(), jan4.Month(), jan4.Day()-(wd-1), 0, 0, 0, 0, time.Local)
	return monday.AddDate(0, 0, (week-1)*7)
}

// SundayOf returns the local-midnight Sunday (start of day) of an ISO week.
func SundayOf(year, week int) time.Time {
	return MondayOf(year, week).AddDate(0, 0, 6)
}

// Offset moves year/week by delta weeks (negative allowed).
func Offset(year, week, delta int) (int, int) {
	return MondayOf(year, week).AddDate(0, 0, delta*7).ISOWeek()
}

var isoWeekRe = regexp.MustCompile(`^(\d{4})-W(\d{1,2})$`)

// ParseWeek parses YYYY-Www (e.g. 2026-W41) for --week and /timeline.
func ParseWeek(s string) (int, int, error) {
	m := isoWeekRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, fmt.Errorf("invalid week %q (want YYYY-Www, e.g. 2026-W41)", s)
	}
	year, _ := strconv.Atoi(m[1])
	week, _ := strconv.Atoi(m[2])
	if week < 1 || week > 53 {
		return 0, 0, fmt.Errorf("invalid week %q (week 1-53)", s)
	}
	return year, week, nil
}

// Title renders "Week 41 · 2026-09-29 – 2026-10-05".
func Title(year, week int) string {
	mon, sun := MondayOf(year, week), SundayOf(year, week)
	return "Week " + itoa(week) + " · " + mon.Format("2006-01-02") + " – " + sun.Format("2006-01-02")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// Build projects tasks into the requested week at time now.
func Build(db *sql.DB, year, week int, now time.Time) (*Week, error) {
	mon := MondayOf(year, week)
	end := mon.AddDate(0, 0, 7) // exclusive upper bound
	w := &Week{Year: year, Week: week, Monday: mon, Days: make([]Day, 0, 7), Overdue: []Item{}}
	for i := 0; i < 7; i++ {
		w.Days = append(w.Days, Day{Date: mon.AddDate(0, 0, i), Items: []Item{}})
	}
	list, err := tasks.ListTasks(db, tasks.Filter{DueAfter: &mon, DueBefore: &end}, now)
	if err != nil {
		return nil, err
	}
	for _, t := range list {
		if t.DueAt == nil {
			continue
		}
		if t.Overdue(now) {
			w.Overdue = append(w.Overdue, Item{Task: t, Overdue: true})
			continue
		}
		local := t.DueAt.Local()
		dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
		idx := int(dayStart.Sub(mon).Hours() / 24)
		if idx < 0 || idx > 6 {
			continue
		}
		w.Days[idx].Items = append(w.Days[idx].Items, Item{Task: t})
	}
	// Overdue items with older due dates never entered the window query.
	older, err := tasks.ListTasks(db, tasks.Filter{Status: domain.TaskOverdue}, now)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, it := range w.Overdue {
		seen[it.Task.UnitID+"/"+it.Task.ID] = true
	}
	for _, t := range older {
		if !seen[t.UnitID+"/"+t.ID] {
			w.Overdue = append(w.Overdue, Item{Task: t, Overdue: true})
		}
	}
	sortItems(w.Overdue)
	for i := range w.Days {
		sortItems(w.Days[i].Items)
	}
	return w, nil
}

// sortItems orders by due time, then priority (high first), then title.
func sortItems(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i].Task, items[j].Task
		if a.DueAt != nil && b.DueAt != nil && !a.DueAt.Equal(*b.DueAt) {
			return a.DueAt.Before(*b.DueAt)
		}
		if domain.PriorityRank(a.Priority) != domain.PriorityRank(b.Priority) {
			return domain.PriorityRank(a.Priority) < domain.PriorityRank(b.Priority)
		}
		return a.Title < b.Title
	})
}
