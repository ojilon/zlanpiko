// Projections for the GUI analytics view (docs/cs/07). Like coverage.go,
// everything here is recomputed on read from live rows; the frontend draws
// shapes only and never re-derives.
package analytics

import (
	"database/sql"
	"time"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
)

// StatusCounts tallies tasks by stored status plus the derived overdue
// pseudo-status (never stored; docs/09).
type StatusCounts struct {
	NotStarted int `json:"not_started"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Submitted  int `json:"submitted"`
	Overdue    int `json:"overdue"`
}

// CountByStatus returns one StatusCounts snapshot at now.
func CountByStatus(db *sql.DB, now time.Time) (StatusCounts, error) {
	var c StatusCounts
	row := func(status string, dest *int) error {
		return db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE status = ?`, status).Scan(dest)
	}
	for status, dest := range map[string]*int{
		string(domain.TaskNotStarted): &c.NotStarted,
		string(domain.TaskInProgress): &c.InProgress,
		string(domain.TaskCompleted):  &c.Completed,
		string(domain.TaskSubmitted):  &c.Submitted,
	} {
		if err := row(status, dest); err != nil {
			return StatusCounts{}, err
		}
	}
	overdue, err := tasks.ListTasks(db, tasks.Filter{Status: domain.TaskOverdue}, now)
	if err != nil {
		return StatusCounts{}, err
	}
	c.Overdue = len(overdue)
	return c, nil
}

// HistDay is one due-date column: open-item count plus the worst signal
// (bad = overdue-anchored item originally due that day, warn = open item,
// none = nothing). Completed items do not move the needle.
type HistDay struct {
	Date    string `json:"date"`
	Display string `json:"display"`
	Count   int    `json:"count"`
	Worst   string `json:"worst"`
}

// DueHistogram returns days consecutive day columns starting today (local).
func DueHistogram(db *sql.DB, now time.Time, days int) ([]HistDay, error) {
	if days < 1 || days > 60 {
		days = 14
	}
	horizon := now.AddDate(0, 0, days)
	list, err := tasks.ListTasks(db, tasks.Filter{DueAfter: &now, DueBefore: &horizon}, now)
	if err != nil {
		return nil, err
	}
	overdue, err := tasks.ListTasks(db, tasks.Filter{Status: domain.TaskOverdue}, now)
	if err != nil {
		return nil, err
	}
	type mark struct {
		count int
		bad   bool
	}
	byDay := map[string]*mark{}
	touch := func(k string) *mark {
		m, ok := byDay[k]
		if !ok {
			m = &mark{}
			byDay[k] = m
		}
		return m
	}
	for _, t := range list {
		if t.DueAt == nil {
			continue
		}
		if t.Status == domain.TaskCompleted || t.Status == domain.TaskSubmitted {
			continue
		}
		touch(t.DueAt.Local().Format("2006-01-02")).count++
	}
	for _, t := range overdue {
		if t.DueAt == nil {
			continue
		}
		k := t.DueAt.Local().Format("2006-01-02")
		// Only mark in-horizon days; older overdue stays in the anchor lane.
		if k >= now.Local().Format("2006-01-02") {
			touch(k).bad = true
		}
	}
	out := make([]HistDay, 0, days)
	base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	for i := 0; i < days; i++ {
		d := base.AddDate(0, 0, i)
		k := d.Format("2006-01-02")
		h := HistDay{Date: k, Display: d.Format("Mon 02"), Count: 0, Worst: "none"}
		if m, ok := byDay[k]; ok {
			h.Count = m.count
			switch {
			case m.bad:
				h.Worst = "bad"
			case m.count > 0:
				h.Worst = "warn"
			}
		}
		out = append(out, h)
	}
	return out, nil
}
