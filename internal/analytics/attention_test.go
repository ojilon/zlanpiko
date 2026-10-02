package analytics

import (
	"database/sql"
	"testing"
	"time"

	"zlanpiko/internal/database"
	"zlanpiko/internal/domain"
	"zlanpiko/migrations"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(database.Path(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(db, migrations.FS, 1); err != nil {
		t.Fatal(err)
	}
	return db
}

func addUnit(t *testing.T, db *sql.DB, id, name string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO units(id, name, created_at, updated_at) VALUES (?, ?, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`, id, name); err != nil {
		t.Fatal(err)
	}
}

func addTopic(t *testing.T, db *sql.DB, unit, id, status, priority string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO topics(unit_id, id, name, reading_status, priority, created_at, updated_at)
		VALUES (?, ?, 'T', ?, ?, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`, unit, id, status, priority); err != nil {
		t.Fatal(err)
	}
}

func addTask(t *testing.T, db *sql.DB, unit, id, kind, status, due string) {
	t.Helper()
	dueVal := "NULL"
	if due != "" {
		dueVal = "'" + due + "'"
	}
	if _, err := db.Exec(`INSERT INTO tasks(unit_id, id, title, kind, status, due_at, created_at, updated_at)
		VALUES (?, ?, 'T', ?, ?, `+dueVal+`, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`,
		unit, id, kind, status); err != nil {
		t.Fatal(err)
	}
}

func assess(t *testing.T, db *sql.DB, unit string, now time.Time) (Attention, string) {
	t.Helper()
	ucs, err := UnitCoverages(db, false, now)
	if err != nil {
		t.Fatal(err)
	}
	var uc UnitCoverage
	for _, u := range ucs {
		if u.UnitID == unit {
			uc = u
		}
	}
	a, reason, err := AssessUnit(db, uc, now)
	if err != nil {
		t.Fatal(err)
	}
	return a, reason
}

func TestAttentionLabels(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	empty := openDB(t)
	addUnit(t, empty, "unit-001", "Empty")
	if a, _ := assess(t, empty, "unit-001", now); a != AttentionNone {
		t.Errorf("empty = %q", a)
	}

	over := openDB(t)
	addUnit(t, over, "unit-001", "Over")
	addTopic(t, over, "unit-001", "topic-001", "read", "normal")
	addTask(t, over, "unit-001", "task-001", "assignment", "in_progress", "2026-10-01T00:00:00Z")
	if a, reason := assess(t, over, "unit-001", now); a != AttentionOverdue || reason == "" {
		t.Errorf("overdue = %q (%q)", a, reason)
	}

	riskyTopics := openDB(t)
	addUnit(t, riskyTopics, "unit-001", "Risky")
	addTopic(t, riskyTopics, "unit-001", "topic-001", "unread", "high")
	addTask(t, riskyTopics, "unit-001", "task-001", "examination", "not_started", "2026-10-08T00:00:00Z")
	if a, _ := assess(t, riskyTopics, "unit-001", now); a != AttentionOverdue {
		t.Errorf("unread-high-before-exam = %q, want Overdue", a)
	}

	done := openDB(t)
	addUnit(t, done, "unit-001", "Done")
	addTopic(t, done, "unit-001", "topic-001", "read", "normal")
	addTask(t, done, "unit-001", "task-001", "assignment", "submitted", "2026-10-01T00:00:00Z")
	if a, _ := assess(t, done, "unit-001", now); a != AttentionCompleted {
		t.Errorf("completed = %q", a)
	}

	risk := openDB(t)
	addUnit(t, risk, "unit-001", "Risk")
	addTopic(t, risk, "unit-001", "topic-001", "unread", "normal")
	addTopic(t, risk, "unit-001", "topic-002", "unread", "normal")
	addTask(t, risk, "unit-001", "task-001", "test", "not_started", "2026-10-11T00:00:00Z")
	if a, _ := assess(t, risk, "unit-001", now); a != AttentionAtRisk {
		t.Errorf("at-risk = %q", a)
	}

	watch := openDB(t)
	addUnit(t, watch, "unit-001", "Watch")
	addTopic(t, watch, "unit-001", "topic-001", "read", "normal")
	addTask(t, watch, "unit-001", "task-001", "assignment", "in_progress", "2026-10-16T00:00:00Z")
	if a, _ := assess(t, watch, "unit-001", now); a != AttentionWatch {
		t.Errorf("watch = %q", a)
	}

	calm := openDB(t)
	addUnit(t, calm, "unit-001", "Calm")
	addTopic(t, calm, "unit-001", "topic-001", "read", "normal")
	addTask(t, calm, "unit-001", "task-001", "assignment", "in_progress", "2026-11-20T00:00:00Z")
	if a, _ := assess(t, calm, "unit-001", now); a != AttentionOnTrack {
		t.Errorf("on-track = %q", a)
	}
}

func TestAssessTopicPure(t *testing.T) {
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	near := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	week := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		status   domain.ReadingStatus
		priority domain.Priority
		exam     *time.Time
		want     Attention
	}{
		{domain.ReadingUnread, domain.PriorityHigh, &near, AttentionAtRisk},
		{domain.ReadingUnread, domain.PriorityNormal, &near, AttentionAtRisk},
		{domain.ReadingUnread, domain.PriorityNormal, &week, AttentionWatch},
		{domain.ReadingUnread, domain.PriorityHigh, &week, AttentionAtRisk},
		{domain.ReadingUnread, domain.PriorityNormal, nil, AttentionOnTrack},
		{domain.ReadingPending, domain.PriorityNormal, &week, AttentionWatch},
		{domain.ReadingRead, domain.PriorityNormal, &week, AttentionReview},
		{domain.ReadingRead, domain.PriorityNormal, nil, AttentionOnTrack},
	}
	for i, c := range cases {
		if got, _ := AssessTopic(c.status, c.priority, c.exam, now); got != c.want {
			t.Errorf("case %d: got %q, want %q", i, got, c.want)
		}
	}
}
