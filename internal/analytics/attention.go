package analytics

import (
	"database/sql"
	"fmt"
	"time"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
)

// Attention planning indicators (see docs/10). These describe workload risk
// from deadlines and reading flags only — never performance predictions.
type Attention string

const (
	AttentionNone      Attention = "—"
	AttentionOnTrack   Attention = "On Track"
	AttentionWatch     Attention = "Attention"
	AttentionAtRisk    Attention = "At Risk"
	AttentionOverdue   Attention = "Overdue"
	AttentionCompleted Attention = "Completed"
	// AttentionReview is topic-level only: read, but an assessment is near.
	AttentionReview Attention = "Review"
)

// Rule thresholds (docs/10). Single place to change them.
const (
	examOverdueDays = 3
	examRiskDays    = 7
	deadlineDays    = 14
	riskCoverage    = 50.0
	watchCoverage   = 70.0
)

// withinDays reports whether t falls in [now, now+days].
func withinDays(t *time.Time, now time.Time, days int) bool {
	if t == nil || t.Before(now) {
		return false
	}
	return t.Sub(now) <= time.Duration(days)*24*time.Hour
}

// NextAssessment returns the unit's nearest open test/examination due at
// or after now, or nil when there is none.
func NextAssessment(db *sql.DB, unitID string, now time.Time) (*domain.Task, error) {
	return tasks.NextOpen(db, unitID, []domain.TaskKind{domain.TaskTest, domain.TaskExamination}, now)
}

// AssessUnit evaluates one unit's attention label with its reason.
// Rules (docs/10), evaluated in order:
// Overdue (overdue task, or unread high-priority topic just before an exam),
// Completed, At Risk, Attention, On Track. Empty units report None.
// Note: docs/10's "half thresholds" clause is intentionally not implemented
// (unquantifiable); the 14-day/70% rule covers the same cases (see D11).
func AssessUnit(db *sql.DB, uc UnitCoverage, now time.Time) (Attention, string, error) {
	if !uc.HasTopics {
		return AttentionNone, "No topics yet", nil
	}
	overdue, err := tasks.ListTasks(db, tasks.Filter{UnitID: uc.UnitID, Status: domain.TaskOverdue}, now)
	if err != nil {
		return "", "", err
	}
	if len(overdue) > 0 {
		return AttentionOverdue, fmt.Sprintf("%d overdue task(s)", len(overdue)), nil
	}
	var unreadHigh int
	if err := db.QueryRow(`SELECT COUNT(*) FROM topics
		WHERE unit_id = ? AND reading_status = 'unread' AND priority = 'high'`, uc.UnitID).Scan(&unreadHigh); err != nil {
		return "", "", err
	}
	exam, err := NextAssessment(db, uc.UnitID, now)
	if err != nil {
		return "", "", err
	}
	var examDue *time.Time
	if exam != nil {
		examDue = exam.DueAt
	}
	if unreadHigh > 0 && withinDays(examDue, now, examOverdueDays) {
		return AttentionOverdue, fmt.Sprintf("unread high-priority topic(s) before %s", examDue.Local().Format("2006-01-02")), nil
	}
	if uc.ActiveTasks == 0 && uc.Coverage == 100 {
		return AttentionCompleted, "all read, nothing open", nil
	}
	if withinDays(examDue, now, examRiskDays) && uc.Coverage < riskCoverage {
		return AttentionAtRisk, fmt.Sprintf("exam %s at %s", examDue.Local().Format("2006-01-02"), uc.CoverageText()), nil
	}
	deadline, err := tasks.NextDeadline(db, uc.UnitID, now)
	if err != nil {
		return "", "", err
	}
	if withinDays(deadline, now, deadlineDays) && (uc.Coverage < watchCoverage || uc.ActiveTasks > 0) {
		return AttentionWatch, fmt.Sprintf("deadline %s at %s", deadline.Local().Format("2006-01-02"), uc.CoverageText()), nil
	}
	return AttentionOnTrack, "", nil
}

// AssessTopic derives a topic's indicator from its own status/priority and
// the unit's nearest assessment (pure function, no I/O). High priority
// escalates one step near an assessment.
func AssessTopic(status domain.ReadingStatus, priority domain.Priority, examDue *time.Time, now time.Time) (Attention, string) {
	hot := priority == domain.PriorityHigh
	switch status {
	case domain.ReadingRead:
		if withinDays(examDue, now, examRiskDays) {
			return AttentionReview, "assessment " + examDue.Local().Format("2006-01-02")
		}
		return AttentionOnTrack, ""
	case domain.ReadingUnread:
		if withinDays(examDue, now, examOverdueDays) {
			return AttentionAtRisk, "unread before " + examDue.Local().Format("2006-01-02")
		}
		if withinDays(examDue, now, examRiskDays) {
			if hot {
				return AttentionAtRisk, "unread, high priority"
			}
			return AttentionWatch, "unread, assessment near"
		}
		return AttentionOnTrack, ""
	default: // pending
		if withinDays(examDue, now, examRiskDays) {
			if hot {
				return AttentionAtRisk, "high priority, assessment near"
			}
			return AttentionWatch, "assessment near"
		}
		return AttentionOnTrack, ""
	}
}
