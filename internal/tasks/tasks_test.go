package tasks

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"zlanpiko/internal/database"
	"zlanpiko/migrations"
)

func setupDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	root := t.TempDir()
	db, err := database.Open(database.Path(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(db, migrations.FS, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO units(id, name, created_at, updated_at)
		VALUES ('unit-001', 'U', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	return db, root
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func utc(tb testing.TB, s string) *time.Time {
	tb.Helper()
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		tb.Fatal(err)
	}
	return &t
}

func TestParseDue(t *testing.T) {
	if got, err := ParseDue(""); err != nil || got != nil {
		t.Errorf("empty: got %v, %v", got, err)
	}
	got, err := ParseDue("2026-10-09")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := time.ParseInLocation("2006-01-02", "2026-10-09", time.Local)
	if !got.Equal(want.UTC().Truncate(time.Second)) {
		t.Errorf("got %v, want %v", got, want.UTC())
	}
	// The space form users type in the GUI drawer must work too.
	spaced, err := ParseDue("2026-10-06 07:00")
	if err != nil {
		t.Fatalf("space form: %v", err)
	}
	wantT, _ := time.ParseInLocation("2006-01-02 15:04", "2026-10-06 07:00", time.Local)
	if !spaced.Equal(wantT.UTC().Truncate(time.Second)) {
		t.Errorf("space form: got %v, want %v", spaced, wantT.UTC())
	}
	if _, err := ParseDue("09/10/2026"); err == nil {
		t.Error("expected error for ambiguous format")
	}
	if _, err := ParseDue("bogus"); err == nil {
		t.Error("expected error for garbage")
	}
}

func TestTaskLifecycleWithDeadline(t *testing.T) {
	db, root := setupDB(t)
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	past := now.Add(-48 * time.Hour)
	tk, err := CreateTask(db, root, discardLogger(), "unit-001", "Problem set", "assignment", &past, "high", "")
	if err != nil {
		t.Fatal(err)
	}
	if tk.ID != "task-001" {
		t.Errorf("id = %q", tk.ID)
	}
	if !tk.Overdue(now) {
		t.Error("past-due open task should be overdue")
	}
	overdue, err := ListTasks(db, Filter{Status: "overdue"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(overdue) != 1 {
		t.Errorf("overdue list = %d, want 1", len(overdue))
	}
	done, err := CompleteTask(db, root, discardLogger(), "unit-001", "task-001", "submitted")
	if err != nil {
		t.Fatal(err)
	}
	if done.CompletedAt == nil || done.Overdue(now) {
		t.Errorf("submitted task should have completed_at and not be overdue: %+v", done)
	}
	// Folder + sidecar exist under the kind group.
	if _, err := os.Stat(filepath.Join(root, "units", "unit-001", "assignments", "task-001", "task.json")); err != nil {
		t.Errorf("task sidecar missing: %v", err)
	}
}

func TestTaskEditAndKindMove(t *testing.T) {
	db, root := setupDB(t)
	future := utc(t, "2026-10-20T12:00:00Z")
	tk, err := CreateTask(db, root, discardLogger(), "unit-001", "Midterm", "test", future, "", "")
	if err != nil {
		t.Fatal(err)
	}
	newTitle, newDue := "Final exam", utc(t, "2026-12-01T09:00:00Z")
	kind := "examination"
	updated, err := UpdateTask(db, root, discardLogger(), "unit-001", tk.ID, Update{
		Title: &newTitle, Kind: &kind, SetDue: true, Due: newDue,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != newTitle || !updated.DueAt.Equal(*newDue) {
		t.Errorf("unexpected update %+v", updated)
	}
	if _, err := os.Stat(filepath.Join(root, "units", "unit-001", "examinations", tk.ID, "task.json")); err != nil {
		t.Errorf("sidecar should follow kind move: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "units", "unit-001", "tests", tk.ID)); !os.IsNotExist(err) {
		t.Errorf("old kind folder should be gone: %v", err)
	}
	// Clearing the deadline.
	cleared, err := UpdateTask(db, root, discardLogger(), "unit-001", tk.ID, Update{SetDue: true})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.DueAt != nil {
		t.Errorf("due should be cleared: %v", cleared.DueAt)
	}
}

func TestTaskDeleteGuard(t *testing.T) {
	db, root := setupDB(t)
	if _, err := CreateTask(db, root, discardLogger(), "unit-001", "T", "", nil, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := DeleteTask(db, root, discardLogger(), "unit-001", "task-001", false); err != nil {
		t.Fatal(err) // no attachments: no confirmation needed
	}
	if _, err := GetTask(db, "unit-001", "task-001"); err == nil {
		t.Error("task should be gone")
	}
}
