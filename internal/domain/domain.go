// Package domain holds entity shapes and pure validation rules.
//
// It imports stdlib only and performs no I/O. All status vocabularies,
// validation limits and the overdue rule live here so TUI, CLI, timeline,
// analytics and reports share exactly one definition (see docs/04, docs/09).
package domain

import (
	"fmt"
	"time"
)

// Vocabulary for units, topics and tasks.

type UnitStatus string

const (
	UnitActive   UnitStatus = "active"
	UnitArchived UnitStatus = "archived"
)

type ReadingStatus string

const (
	ReadingUnread  ReadingStatus = "unread"
	ReadingPending ReadingStatus = "pending"
	ReadingRead    ReadingStatus = "read"
)

// Any manual transition between reading statuses is allowed; in particular,
// nothing but an explicit user action may change the status (see docs/01 FR-T3).
var readingTransitions = map[ReadingStatus][]ReadingStatus{
	ReadingUnread:  {ReadingPending, ReadingRead},
	ReadingPending: {ReadingUnread, ReadingRead},
	ReadingRead:    {ReadingUnread, ReadingPending},
}

// CanTransitionReading reports whether a manual status change is permitted.
func CanTransitionReading(from, to ReadingStatus) bool {
	if from == to {
		return true
	}
	for _, next := range readingTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

type TaskStatus string

const (
	TaskNotStarted TaskStatus = "not_started"
	TaskInProgress TaskStatus = "in_progress"
	TaskCompleted  TaskStatus = "completed"
	TaskSubmitted  TaskStatus = "submitted"
)

// TaskOverdue is a derived pseudo-status for display and filtering only.
// It is never stored (see docs/09).
const TaskOverdue = "overdue"

type TaskKind string

const (
	TaskAssignment  TaskKind = "assignment"
	TaskCoursework  TaskKind = "coursework"
	TaskTest        TaskKind = "test"
	TaskExamination TaskKind = "examination"
	TaskProject     TaskKind = "project"
	TaskOther       TaskKind = "other"
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

// Parse helpers reject anything outside the vocabularies.

func ParseUnitStatus(s string) (UnitStatus, error) {
	switch UnitStatus(s) {
	case UnitActive, UnitArchived:
		return UnitStatus(s), nil
	default:
		return "", fmt.Errorf("invalid unit status %q: want active|archived", s)
	}
}

func ParseReadingStatus(s string) (ReadingStatus, error) {
	switch ReadingStatus(s) {
	case ReadingUnread, ReadingPending, ReadingRead:
		return ReadingStatus(s), nil
	default:
		return "", fmt.Errorf("invalid reading status %q: want unread|pending|read", s)
	}
}

func ParseTaskStatus(s string) (TaskStatus, error) {
	switch TaskStatus(s) {
	case TaskNotStarted, TaskInProgress, TaskCompleted, TaskSubmitted:
		return TaskStatus(s), nil
	default:
		return "", fmt.Errorf("invalid task status %q: want not_started|in_progress|completed|submitted", s)
	}
}

func ParseTaskKind(s string) (TaskKind, error) {
	switch TaskKind(s) {
	case TaskAssignment, TaskCoursework, TaskTest, TaskExamination, TaskProject, TaskOther:
		return TaskKind(s), nil
	default:
		return "", fmt.Errorf("invalid task kind %q", s)
	}
}

func ParsePriority(s string) (Priority, error) {
	switch Priority(s) {
	case PriorityLow, PriorityNormal, PriorityHigh:
		return Priority(s), nil
	default:
		return "", fmt.Errorf("invalid priority %q: want low|normal|high", s)
	}
}

// IsOverdue derives overdue state from deadline and stored status.
// A nil dueAt means no deadline, which can never be overdue.
func IsOverdue(status TaskStatus, dueAt *time.Time, now time.Time) bool {
	if dueAt == nil {
		return false
	}
	if status == TaskCompleted || status == TaskSubmitted {
		return false
	}
	return dueAt.Before(now)
}

// Entities. IDs are stable strings (unit-001, topic-001 per unit, ...);
// services assign them in Phase 3. Timestamps are UTC.

type Unit struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	Description string     `json:"description"`
	Colour      string     `json:"colour"`
	Status      UnitStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Validate checks a unit's field values (uniqueness is a storage concern).
func (u Unit) Validate() error {
	if len(u.Name) < 1 || len(u.Name) > 120 {
		return fmt.Errorf("unit name must be 1-120 characters, got %d", len(u.Name))
	}
	if _, err := ParseUnitStatus(string(u.Status)); err != nil {
		return err
	}
	return nil
}

type Topic struct {
	UnitID        string        `json:"unit_id"`
	ID            string        `json:"id"` // per-unit sequence, e.g. topic-001
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	ReadingStatus ReadingStatus `json:"reading_status"`
	Priority      Priority      `json:"priority"`
	Notes         string        `json:"notes"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// Validate checks a topic's field values.
func (t Topic) Validate() error {
	if len(t.Name) < 1 || len(t.Name) > 120 {
		return fmt.Errorf("topic name must be 1-120 characters, got %d", len(t.Name))
	}
	if _, err := ParseReadingStatus(string(t.ReadingStatus)); err != nil {
		return err
	}
	if _, err := ParsePriority(string(t.Priority)); err != nil {
		return err
	}
	return nil
}

type Task struct {
	UnitID      string     `json:"unit_id"`
	ID          string     `json:"id"` // per-unit sequence, e.g. task-001
	Title       string     `json:"title"`
	Kind        TaskKind   `json:"kind"`
	Status      TaskStatus `json:"status"`
	DueAt       *time.Time `json:"due_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Priority    Priority   `json:"priority"`
	Description string     `json:"description"`
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Validate checks a task's field values.
func (t Task) Validate() error {
	// Assumption: task titles allow up to 200 chars (spec fixes 120 only for
	// unit/topic names). Recorded here; easy to change in one place.
	if len(t.Title) < 1 || len(t.Title) > 200 {
		return fmt.Errorf("task title must be 1-200 characters, got %d", len(t.Title))
	}
	if _, err := ParseTaskKind(string(t.Kind)); err != nil {
		return err
	}
	if _, err := ParseTaskStatus(string(t.Status)); err != nil {
		return err
	}
	if _, err := ParsePriority(string(t.Priority)); err != nil {
		return err
	}
	return nil
}

// Overdue reports the derived overdue state of the task at time now.
func (t Task) Overdue(now time.Time) bool {
	return IsOverdue(t.Status, t.DueAt, now)
}
