package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTimeRoundTrip(t *testing.T) {
	now := Now()
	if now.Nanosecond() != 0 {
		t.Errorf("Now() should be second-truncated, got %v", now)
	}
	s := FormatTime(now)
	back, err := ParseTime(s)
	if err != nil {
		t.Fatal(err)
	}
	if !back.Equal(now) {
		t.Errorf("round trip: got %v, want %v", back, now)
	}
	if _, err := ParseTime("not-a-time"); err == nil {
		t.Error("expected error for bad timestamp")
	}
}

func TestConfirmRequiredError(t *testing.T) {
	err := &ConfirmRequiredError{Entity: "unit unit-001", Counts: ChildCounts{Topics: 2}}
	msg := err.Error()
	for _, want := range []string{"unit unit-001", "2 topic(s)", "archive"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q missing %q", msg, want)
		}
	}
}

func TestConfirmEmpty(t *testing.T) {
	if !(ChildCounts{}).Empty() {
		t.Error("zero counts should be empty")
	}
	if (ChildCounts{Files: 1}).Empty() {
		t.Error("non-zero counts should not be empty")
	}
}

func TestStructJSONTags(t *testing.T) {
	// Sidecars and JSON exports marshal these structs; fields must use
	// explicit snake_case tags.
	topic := Topic{UnitID: "unit-001", ID: "topic-001", Name: "T",
		ReadingStatus: ReadingUnread, Priority: PriorityNormal, CreatedAt: time.Now()}
	raw, err := json.Marshal(topic)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"unit_id"`, `"reading_status"`, `"created_at"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("topic JSON missing %s: %s", want, raw)
		}
	}
	due := time.Now()
	task := Task{UnitID: "unit-001", ID: "task-001", Title: "T",
		Kind: TaskAssignment, Status: TaskNotStarted, DueAt: &due, Priority: PriorityHigh}
	raw, err = json.Marshal(task)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"due_at"`, `"completed_at"`, `"kind"`} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("task JSON missing %s: %s", want, raw)
		}
	}
}
