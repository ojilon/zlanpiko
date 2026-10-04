// Package gui is the Wails-bound service surface for zlanpiko-gui.
//
// It is intentionally thin: every method resolves DTOs from the existing
// services (units, topics, tasks, timeline, analytics, filesystem, importer,
// exporter, backup, config) and returns them as JSON-friendly structs.
// Business rules stay in those packages; TypeScript renders and never
// re-derives (see docs/cs/01, docs/cs/02).
package gui

import (
	"time"

	"zlanpiko/internal/domain"
)

// GuiErrorCode classifies user-facing failures from bound methods.
type GuiErrorCode string

const (
	ErrValidation GuiErrorCode = "VALIDATION"
	ErrNotFound   GuiErrorCode = "NOT_FOUND"
	ErrConflict   GuiErrorCode = "CONFLICT"
	ErrStorage    GuiErrorCode = "STORAGE"
	ErrDB         GuiErrorCode = "DB"
	ErrBackup     GuiErrorCode = "BACKUP"
	ErrCancelled  GuiErrorCode = "CANCELLED"
)

// GuiError is the JSON error shape the frontend renders as toast/inline hint.
// It mirrors frontend/src/types.ts GuiError; keep both in sync (dto_test.go).
type GuiError struct {
	Code    GuiErrorCode `json:"code"`
	Message string       `json:"message"`
	Hint    string       `json:"hint,omitempty"`
}

func (e *GuiError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Hint == "" {
		return string(e.Code) + ": " + e.Message
	}
	return string(e.Code) + ": " + e.Message + " (" + e.Hint + ")"
}

func fail(code GuiErrorCode, msg, hint string) *GuiError {
	return &GuiError{Code: code, Message: msg, Hint: hint}
}

// TaskDTO is one task ready to render: raw enums plus pre-formatted
// deadline strings and the derived distance/progress values. The overdue
// flag is derived (never stored); dateless tasks carry nil DueISO and
// Distance "no date" (see docs/cs/02, docs/cs/05).
type TaskDTO struct {
	ID          string            `json:"id"`
	UnitID      string            `json:"unit_id"`
	UnitName    string            `json:"unit_name"`
	Title       string            `json:"title"`
	Kind        domain.TaskKind   `json:"kind"`
	Status      domain.TaskStatus `json:"status"`
	Overdue     bool              `json:"overdue"`
	Priority    domain.Priority   `json:"priority"`
	DueISO      *time.Time        `json:"due_iso,omitempty"`
	DueDisplay  string            `json:"due_display"`
	Distance    string            `json:"distance"`
	DaysLeft    *int              `json:"days_left,omitempty"`
	ProgressPct *float64          `json:"progress_pct,omitempty"`
}

// UnitCardDTO is one unit's dashboard card: coverage counts, the "—" rule
// for empty units, task roll-up and the attention label with its reason.
type UnitCardDTO struct {
	UnitID         string  `json:"unit_id"`
	Name           string  `json:"name"`
	Code           string  `json:"code"`
	Total          int     `json:"total"`
	Read           int     `json:"read"`
	Pending        int     `json:"pending"`
	Unread         int     `json:"unread"`
	Coverage       float64 `json:"coverage"`
	CoverageText   string  `json:"coverage_text"`
	HasTopics      bool    `json:"has_topics"`
	ActiveTasks    int     `json:"active_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	NextDueDisplay string  `json:"next_due_display"`
	Attention      string  `json:"attention"`
	Reason         string  `json:"reason,omitempty"`
}

// DayDTO is one calendar day lane in a week view.
type DayDTO struct {
	Date    string    `json:"date"`
	Display string    `json:"display"`
	Items   []TaskDTO `json:"items"`
}

// WeekDTO is one ISO week (Monday–Sunday) plus the overdue anchor section.
// Weeks are computed by internal/timeline; this struct only reshapes.
type WeekDTO struct {
	Year    int       `json:"year"`
	Week    int       `json:"week"`
	Title   string    `json:"title"`
	Monday  string    `json:"monday"`
	Days    []DayDTO  `json:"days"`
	Overdue []TaskDTO `json:"overdue"`
}

// SummaryDTO is the overall academic roll-up for the header/status line.
type SummaryDTO struct {
	Units          int     `json:"units"`
	TotalTopics    int     `json:"total_topics"`
	Read           int     `json:"read"`
	Pending        int     `json:"pending"`
	Unread         int     `json:"unread"`
	Coverage       float64 `json:"coverage"`
	CoverageText   string  `json:"coverage_text"`
	HasTopics      bool    `json:"has_topics"`
	ActiveTasks    int     `json:"active_tasks"`
	CompletedTasks int     `json:"completed_tasks"`
	OverdueTasks   int     `json:"overdue_tasks"`
}

// DashboardDTO boots the whole home view in one call: header summary,
// per-unit cards, attention-relevant lists and the current week rail.
type DashboardDTO struct {
	Version  string        `json:"version"`
	Summary  SummaryDTO    `json:"summary"`
	Units    []UnitCardDTO `json:"units"`
	Upcoming []TaskDTO     `json:"upcoming"`
	Overdue  []TaskDTO     `json:"overdue"`
	Week     WeekDTO       `json:"week"`
}
