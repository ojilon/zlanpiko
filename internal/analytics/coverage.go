// Package analytics computes reading coverage and work summaries from live
// database rows. Nothing here is stored; everything is recomputed on read,
// and every formula is shared by the dashboard, CLI, timeline and reports
// (see docs/10).
//
// The one coverage formula: coverage = read / total × 100. An empty unit
// (total = 0) reports HasTopics = false and displays as "—", never 0% or
// 100%. Attention labels and timeline buckets arrive in Phase 6.
package analytics

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
)

// UnitCoverage is one unit's reading and work summary.
type UnitCoverage struct {
	UnitID         string            `json:"unit_id"`
	Name           string            `json:"name"`
	Code           string            `json:"code"`
	Status         domain.UnitStatus `json:"status"`
	Total          int               `json:"total"`
	Read           int               `json:"read"`
	Pending        int               `json:"pending"`
	Unread         int               `json:"unread"`
	Coverage       float64           `json:"coverage"`
	HasTopics      bool              `json:"has_topics"`
	ActiveTasks    int               `json:"active_tasks"`
	CompletedTasks int               `json:"completed_tasks"`
	NextDue        *time.Time        `json:"next_due,omitempty"`
}

// CoverageText renders the percentage or "—" for empty units.
func (u UnitCoverage) CoverageText() string {
	if !u.HasTopics {
		return "—"
	}
	return formatPct(u.Coverage)
}

// formatPct renders one-decimal percentages, trimming ".0" (40%, 33.3%).
func formatPct(f float64) string {
	s := fmt.Sprintf("%.1f", f)
	return strings.TrimSuffix(s, ".0") + "%"
}

// UnitCoverages returns per-unit summaries ordered by name.
func UnitCoverages(db *sql.DB, includeArchived bool, now time.Time) ([]UnitCoverage, error) {
	query := `SELECT id, name, code, status FROM units`
	if !includeArchived {
		query += ` WHERE status = 'active'`
	}
	query += ` ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	out := []UnitCoverage{}
	for rows.Next() {
		var uc UnitCoverage
		var code sql.NullString
		var status string
		if err := rows.Scan(&uc.UnitID, &uc.Name, &code, &status); err != nil {
			rows.Close()
			return nil, err
		}
		uc.Code = code.String
		uc.Status, err = domain.ParseUnitStatus(status)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, uc)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close() // before fillCoverage: single-connection database
	for i := range out {
		if err := fillCoverage(db, &out[i], now); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func fillCoverage(db *sql.DB, uc *UnitCoverage, now time.Time) error {
	counts := map[domain.ReadingStatus]*int{
		domain.ReadingRead: &uc.Read, domain.ReadingPending: &uc.Pending, domain.ReadingUnread: &uc.Unread,
	}
	for status, dest := range counts {
		if err := db.QueryRow(`SELECT COUNT(*) FROM topics WHERE unit_id = ? AND reading_status = ?`,
			uc.UnitID, string(status)).Scan(dest); err != nil {
			return err
		}
	}
	uc.Total = uc.Read + uc.Pending + uc.Unread
	uc.HasTopics = uc.Total > 0
	if uc.HasTopics {
		uc.Coverage = float64(uc.Read) / float64(uc.Total) * 100
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE unit_id = ? AND status IN ('not_started','in_progress')`,
		uc.UnitID).Scan(&uc.ActiveTasks); err != nil {
		return err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE unit_id = ? AND status IN ('completed','submitted')`,
		uc.UnitID).Scan(&uc.CompletedTasks); err != nil {
		return err
	}
	var due sql.NullString
	err := db.QueryRow(`SELECT due_at FROM tasks WHERE unit_id = ? AND due_at IS NOT NULL
		AND status IN ('not_started','in_progress') ORDER BY due_at LIMIT 1`, uc.UnitID).Scan(&due)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if t, err := domain.ParseTime(due.String); err == nil {
		uc.NextDue = &t
	}
	return nil
}

// Summary is the overall academic roll-up.
type Summary struct {
	Units          int     `json:"units"`
	TotalTopics    int     `json:"total_topics"`
	Read           int     `json:"read"`
	Pending        int     `json:"pending"`
	Unread         int     `json:"unread"`
	Coverage       float64 `json:"coverage"`
	HasTopics      bool    `json:"has_topics"`
	ActiveTasks    int     `json:"active_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	OverdueTasks   int     `json:"overdue_tasks"`
}

// CoverageText renders the overall percentage or "—" with no topics.
func (s Summary) CoverageText() string {
	if !s.HasTopics {
		return "—"
	}
	return formatPct(s.Coverage)
}

// Summarize rolls per-unit coverages plus task states into a Summary.
// Overdue is derived with now (see docs/09).
func Summarize(db *sql.DB, ucs []UnitCoverage, now time.Time) (Summary, error) {
	var s Summary
	s.Units = len(ucs)
	for _, uc := range ucs {
		s.TotalTopics += uc.Total
		s.Read += uc.Read
		s.Pending += uc.Pending
		s.Unread += uc.Unread
		s.ActiveTasks += uc.ActiveTasks
		s.CompletedTasks += uc.CompletedTasks
	}
	s.HasTopics = s.TotalTopics > 0
	if s.HasTopics {
		s.Coverage = float64(s.Read) / float64(s.TotalTopics) * 100
	}
	overdue, err := tasks.ListTasks(db, tasks.Filter{Status: domain.TaskOverdue}, now)
	if err != nil {
		return Summary{}, err
	}
	s.OverdueTasks = len(overdue)
	return s, nil
}

// Upcoming returns open tasks due within days (ordered by due date).
func Upcoming(db *sql.DB, now time.Time, days int) ([]domain.Task, error) {
	horizon := now.AddDate(0, 0, days)
	list, err := tasks.ListTasks(db, tasks.Filter{DueAfter: &now, DueBefore: &horizon}, now)
	if err != nil {
		return nil, err
	}
	out := []domain.Task{}
	for _, t := range list {
		if t.Status == domain.TaskCompleted || t.Status == domain.TaskSubmitted {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}
