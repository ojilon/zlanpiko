package timeline

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

func seedTask(t *testing.T, db *sql.DB, unit, id, title, kind, status, due string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO units(id, name, created_at, updated_at)
		VALUES (?, 'U', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')
		ON CONFLICT(id) DO NOTHING`, unit); err != nil {
		t.Fatal(err)
	}
	dueVal := "NULL"
	if due != "" {
		dueVal = "'" + due + "'"
	}
	if _, err := db.Exec(`INSERT INTO tasks(unit_id, id, title, kind, status, due_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, `+dueVal+`, '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`,
		unit, id, title, kind, status); err != nil {
		t.Fatal(err)
	}
}

func TestMondayOfKnownWeeks(t *testing.T) {
	// 2026-10-05 is a Monday (ISO week 41).
	mon := MondayOf(2026, 41)
	if mon.Format("2006-01-02") != "2026-10-05" {
		t.Errorf("MondayOf(2026,41) = %s", mon.Format("2006-01-02"))
	}
	if got := SundayOf(2026, 41).Format("2006-01-02"); got != "2026-10-11" {
		t.Errorf("SundayOf = %s", got)
	}
	y, w := WeekOf(mon.AddDate(0, 0, 3))
	if y != 2026 || w != 41 {
		t.Errorf("WeekOf mid-week = %d-W%d", y, w)
	}
}

func TestOffsetAcrossYearBoundary(t *testing.T) {
	y, w := Offset(2026, 41, 1)
	if y != 2026 || w != 42 {
		t.Errorf("+1 = %d-W%d", y, w)
	}
	y, w = Offset(2026, 1, -1)
	if y != 2025 || w != 52 {
		t.Errorf("-1 over boundary = %d-W%d", y, w)
	}
	y, w = Offset(2026, 41, 0)
	if y != 2026 || w != 41 {
		t.Errorf("0 = %d-W%d", y, w)
	}
}

func TestTitleFormat(t *testing.T) {
	if got := Title(2026, 41); got != "Week 41 · 2026-10-05 – 2026-10-11" {
		t.Errorf("title = %q", got)
	}
}

func TestBuildBucketsSortsAndAnchorsOverdue(t *testing.T) {
	db := setupDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC) // Tuesday of week 41
	seedTask(t, db, "unit-001", "task-001", "Z last", "assignment", "not_started", "2026-10-07T09:00:00Z")
	seedTask(t, db, "unit-001", "task-002", "A first", "assignment", "not_started", "2026-10-07T09:00:00Z")
	seedTask(t, db, "unit-001", "task-003", "High pri", "test", "in_progress", "2026-10-07T08:00:00Z")
	seedTask(t, db, "unit-001", "task-004", "Old overdue", "assignment", "not_started", "2026-09-01T00:00:00Z")
	seedTask(t, db, "unit-001", "task-005", "Week overdue", "assignment", "not_started", "2026-10-05T00:00:00Z")
	seedTask(t, db, "unit-001", "task-006", "Done", "assignment", "completed", "2026-10-07T09:00:00Z")
	seedTask(t, db, "unit-001", "task-007", "Dateless", "assignment", "not_started", "")

	w, err := Build(db, 2026, 41, now)
	if err != nil {
		t.Fatal(err)
	}
	// Wednesday (index 2) holds the due items: earliest first, then priority.
	wed := w.Days[2]
	if len(wed.Items) != 4 { // 3 open + 1 completed; dateless excluded
		t.Fatalf("wednesday items = %d, want 4", len(wed.Items))
	}
	if wed.Items[0].Task.ID != "task-003" { // earliest due
		t.Errorf("first = %s", wed.Items[0].Task.ID)
	}
	if wed.Items[1].Task.ID != "task-002" || wed.Items[2].Task.ID != "task-006" || wed.Items[3].Task.ID != "task-001" {
		t.Errorf("tie order wrong: %s, %s, %s", wed.Items[1].Task.ID, wed.Items[2].Task.ID, wed.Items[3].Task.ID)
	}
	// Overdue section: old + in-week overdue, each once.
	if len(w.Overdue) != 2 {
		t.Fatalf("overdue = %d, want 2", len(w.Overdue))
	}
	seen := map[string]int{}
	for _, it := range w.Overdue {
		seen[it.Task.ID]++
		if !it.Overdue {
			t.Errorf("overdue flag unset on %s", it.Task.ID)
		}
	}
	if seen["task-004"] != 1 || seen["task-005"] != 1 {
		t.Errorf("overdue ids wrong: %v", seen)
	}
	// Other days empty.
	for i, d := range w.Days {
		if i != 0 && i != 2 && len(d.Items) != 0 {
			t.Errorf("day %d should be empty: %+v", i, d.Items)
		}
	}
	if len(w.Days[0].Items) != 0 { // Monday: task-005 is overdue -> anchored, not in day
		t.Errorf("monday should be empty (overdue anchored): %+v", w.Days[0].Items)
	}
}
