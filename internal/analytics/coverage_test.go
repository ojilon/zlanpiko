package analytics

import (
	"database/sql"
	"testing"
	"time"

	"zlanpiko/internal/database"
	"zlanpiko/migrations"
)

func setupDB(t *testing.T) *sql.DB {
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

func seedUnit(t *testing.T, db *sql.DB, id, name string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO units(id, name, created_at, updated_at) VALUES (?, ?, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`, id, name); err != nil {
		t.Fatal(err)
	}
}

func seedTopics(t *testing.T, db *sql.DB, unit string, read, pending, unread int) {
	t.Helper()
	n := 0
	add := func(status string, count int) {
		for i := 0; i < count; i++ {
			n++
			if _, err := db.Exec(`INSERT INTO topics(unit_id, id, name, reading_status, created_at, updated_at)
				VALUES (?, ?, ?, ?, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`,
				unit, "topic-"+itoa(n), "T", status); err != nil {
				t.Fatal(err)
			}
		}
	}
	add("read", read)
	add("pending", pending)
	add("unread", unread)
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestCoverageFormula(t *testing.T) {
	db := setupDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	seedUnit(t, db, "unit-001", "Maths")
	seedTopics(t, db, "unit-001", 8, 5, 7) // 8/20 = 40%
	ucs, err := UnitCoverages(db, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(ucs) != 1 {
		t.Fatalf("units = %d", len(ucs))
	}
	uc := ucs[0]
	if uc.Total != 20 || uc.Read != 8 || uc.Pending != 5 || uc.Unread != 7 {
		t.Errorf("counts = %+v", uc)
	}
	if uc.Coverage != 40.0 {
		t.Errorf("coverage = %v, want 40.0", uc.Coverage)
	}
	if uc.CoverageText() != "40%" {
		t.Errorf("text = %q", uc.CoverageText())
	}
	s, err := Summarize(db, ucs, now)
	if err != nil {
		t.Fatal(err)
	}
	if s.Coverage != 40.0 || s.CoverageText() != "40%" {
		t.Errorf("summary = %+v", s)
	}
}

func TestEmptyUnitShowsDash(t *testing.T) {
	db := setupDB(t)
	now := time.Now()
	seedUnit(t, db, "unit-001", "Empty")
	ucs, err := UnitCoverages(db, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if ucs[0].HasTopics || ucs[0].CoverageText() != "—" {
		t.Errorf("empty unit = %+v", ucs[0])
	}
	s, err := Summarize(db, ucs, now)
	if err != nil {
		t.Fatal(err)
	}
	if s.HasTopics || s.CoverageText() != "—" {
		t.Errorf("empty summary = %+v", s)
	}
}

func TestTaskCountsAndOverdue(t *testing.T) {
	db := setupDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	seedUnit(t, db, "unit-001", "U")
	seedTopics(t, db, "unit-001", 1, 0, 0)
	exec := func(title, status, due string) {
		t.Helper()
		dueVal := "NULL"
		if due != "" {
			dueVal = "'" + due + "'"
		}
		if _, err := db.Exec(`INSERT INTO tasks(unit_id, id, title, status, due_at, created_at, updated_at)
			VALUES ('unit-001', '` + title + `', '` + title + `', '` + status + `', ` + dueVal + `, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`); err != nil {
			t.Fatal(err)
		}
	}
	exec("task-001", "in_progress", "2026-10-01T00:00:00Z") // overdue
	exec("task-002", "completed", "2026-10-01T00:00:00Z")   // done, not overdue
	exec("task-003", "not_started", "2026-10-20T00:00:00Z") // upcoming
	ucs, err := UnitCoverages(db, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if ucs[0].ActiveTasks != 2 || ucs[0].CompletedTasks != 1 {
		t.Errorf("task counts = %+v", ucs[0])
	}
	if ucs[0].NextDue == nil || ucs[0].NextDue.Format("2006-01-02") != "2026-10-01" {
		t.Errorf("next due = %v", ucs[0].NextDue)
	}
	s, err := Summarize(db, ucs, now)
	if err != nil {
		t.Fatal(err)
	}
	if s.OverdueTasks != 1 {
		t.Errorf("overdue = %d", s.OverdueTasks)
	}
	up, err := Upcoming(db, now, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(up) != 1 || up[0].ID != "task-003" {
		t.Errorf("upcoming = %+v", up)
	}
}
