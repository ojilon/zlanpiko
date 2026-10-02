package domain

import (
	"strings"
	"testing"
	"time"
)

func TestParseValidVocabularies(t *testing.T) {
	if _, err := ParseUnitStatus("archived"); err != nil {
		t.Error(err)
	}
	if _, err := ParseReadingStatus("pending"); err != nil {
		t.Error(err)
	}
	if _, err := ParseTaskStatus("submitted"); err != nil {
		t.Error(err)
	}
	if _, err := ParseTaskKind("examination"); err != nil {
		t.Error(err)
	}
	if _, err := ParsePriority("high"); err != nil {
		t.Error(err)
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	cases := []func(string) (any, error){
		func(s string) (any, error) { return ParseUnitStatus(s) },
		func(s string) (any, error) { return ParseReadingStatus(s) },
		func(s string) (any, error) { return ParseTaskStatus(s) },
		func(s string) (any, error) { return ParseTaskKind(s) },
		func(s string) (any, error) { return ParsePriority(s) },
	}
	for i, parse := range cases {
		if _, err := parse("bogus"); err == nil {
			t.Errorf("case %d: expected error for invalid value", i)
		}
		if _, err := parse(""); err == nil {
			t.Errorf("case %d: expected error for empty value", i)
		}
	}
}

func TestTaskOverdueIsNotAStoredStatus(t *testing.T) {
	if _, err := ParseTaskStatus(TaskOverdue); err == nil {
		t.Error("overdue must not parse as a stored task status")
	}
}

func TestReadingTransitionsAreManualAndFree(t *testing.T) {
	all := []ReadingStatus{ReadingUnread, ReadingPending, ReadingRead}
	for _, from := range all {
		for _, to := range all {
			if !CanTransitionReading(from, to) {
				t.Errorf("expected transition %s -> %s to be allowed", from, to)
			}
		}
	}
	if CanTransitionReading("bogus", ReadingRead) {
		t.Error("unknown status must not transition")
	}
}

func TestValidateNames(t *testing.T) {
	if err := (Unit{Name: "Linear Algebra", Status: UnitActive}).Validate(); err != nil {
		t.Error(err)
	}
	if err := (Unit{Name: "", Status: UnitActive}).Validate(); err == nil {
		t.Error("expected error for empty unit name")
	}
	if err := (Unit{Name: strings.Repeat("x", 121), Status: UnitActive}).Validate(); err == nil {
		t.Error("expected error for overlong unit name")
	}
	if err := (Topic{Name: "Eigenvalues", ReadingStatus: ReadingUnread, Priority: PriorityNormal}).Validate(); err != nil {
		t.Error(err)
	}
	if err := (Task{Title: "Problem set", Kind: TaskAssignment, Status: TaskInProgress, Priority: PriorityHigh}).Validate(); err != nil {
		t.Error(err)
	}
	if err := (Task{Title: "", Kind: TaskAssignment, Status: TaskInProgress}).Validate(); err == nil {
		t.Error("expected error for empty task title")
	}
}

func TestIsOverdueDerived(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	cases := []struct {
		name   string
		status TaskStatus
		dueAt  *time.Time
		want   bool
	}{
		{"past due open", TaskInProgress, &past, true},
		{"past due completed", TaskCompleted, &past, false},
		{"past due submitted", TaskSubmitted, &past, false},
		{"future due open", TaskNotStarted, &future, false},
		{"no deadline", TaskInProgress, nil, false},
	}
	for _, c := range cases {
		if got := IsOverdue(c.status, c.dueAt, now); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
