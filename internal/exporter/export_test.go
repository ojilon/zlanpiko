package exporter

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"zlanpiko/internal/database"
	"zlanpiko/internal/services"
	"zlanpiko/internal/tasks"
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

func TestTextReportSections(t *testing.T) {
	db, root := setupDBRoot(t)
	u, err := services.CreateUnit(db, root, nil, "Algebra", "MATH201", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := services.CreateTopic(db, root, nil, u.ID, "Eigen", "high"); err != nil {
		t.Fatal(err)
	}
	due := time.Date(2026, 10, 9, 23, 59, 0, 0, time.UTC)
	if _, err := tasks.CreateTask(db, root, nil, u.ID, "Problem set", "assignment", &due, "high", ""); err != nil {
		t.Fatal(err)
	}
	dueLocal := due.Local().Format("2006-01-02")
	now := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	report, err := TextReport(db, "", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"ACADEMIC STATUS REPORT", "UNITS", "Algebra", "MATH201",
		"TOPICS NEEDING ATTENTION", "UPCOMING DEADLINES", dueLocal + " - Problem set",
		"OVERDUE ITEMS", "None", "GENERAL SUMMARY", "Total units: 1",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report missing %q:\n%s", want, report)
		}
	}
	// Every line fits the phone width.
	for _, line := range strings.Split(report, "\n") {
		if len([]rune(line)) > lineWidth {
			t.Errorf("line exceeds %d cols: %q", lineWidth, line)
		}
	}
}

func TestTextReportScopedAndEmpty(t *testing.T) {
	db, _ := setupDBRoot(t)
	now := time.Now()
	report, err := TextReport(db, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(report, "Total units: 0") {
		t.Errorf("empty report:\n%s", report)
	}
	if _, err := TextReport(db, "unit-999", now); err == nil {
		t.Error("expected error for unknown unit scope")
	}
}

func TestJSONExportStable(t *testing.T) {
	db, root := setupDBRoot(t)
	u, err := services.CreateUnit(db, root, nil, "U", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := services.CreateTopic(db, root, nil, u.ID, "T", ""); err != nil {
		t.Fatal(err)
	}
	raw, err := JSONExport(db, "", time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"generated_at"`, `"units"`, `"topics"`, `"tasks"`, `"summary"`,
		`"attention"`, `"unit-001"`,
	} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("json missing %s:\n%s", want, raw)
		}
	}
}

func setupDBRoot(t *testing.T) (*sql.DB, string) {
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
	return db, root
}
