package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/app"
	"zlanpiko/internal/domain"
	"zlanpiko/internal/services"
	"zlanpiko/internal/tasks"
	"zlanpiko/internal/timeline"
)

// fixedNow pins "now" for deterministic goldens. Times serialize with the
// machine's local UTC offset, so goldens are host-TZ specific; regenerate
// with UPDATE_GOLDEN=1 after a deliberate DTO change (see docs/cs/10).
var fixedNow = time.Date(2026, time.October, 4, 12, 0, 0, 0, time.Local)

// seedPhaseB builds the documented fixture: Periodicity (R1/P1/U3, one task
// due in 2d) plus an empty Ecology unit with one overdue task.
func seedPhaseB(t *testing.T, ctx *app.Context) {
	t.Helper()
	created := fixedNow.AddDate(0, 0, -10)
	duePres := fixedNow.AddDate(0, 0, 2)
	dueOver := fixedNow.AddDate(0, 0, -1)
	dueDone := fixedNow.AddDate(0, 0, -3)

	per, err := services.CreateUnit(ctx.DB, ctx.DataRoot, nil, "Periodicity", "SED:2103", "", "")
	if err != nil {
		t.Fatalf("create unit: %v", err)
	}
	eco, err := services.CreateUnit(ctx.DB, ctx.DataRoot, nil, "Ecology", "SBE:2101", "", "")
	if err != nil {
		t.Fatalf("create unit: %v", err)
	}
	statuses := []string{"read", "pending", "unread", "unread", "unread"}
	for i, st := range statuses {
		topic, err := services.CreateTopic(ctx.DB, ctx.DataRoot, nil, per.ID, "topic-"+string(rune('a'+i)), "normal")
		if err != nil {
			t.Fatalf("create topic: %v", err)
		}
		if _, err := services.SetTopicStatus(ctx.DB, ctx.DataRoot, nil, per.ID, topic.ID, st); err != nil {
			t.Fatalf("set topic status: %v", err)
		}
	}
	mkTask := func(unitID, title, kind string, due time.Time) domain.Task {
		task, err := tasks.CreateTask(ctx.DB, ctx.DataRoot, nil, unitID, title, kind, &due, "normal", "")
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		if _, err := ctx.DB.Exec(`UPDATE tasks SET created_at = ?, updated_at = ? WHERE unit_id = ? AND id = ?`,
			domain.FormatTime(created), domain.FormatTime(created), unitID, task.ID); err != nil {
			t.Fatalf("pin created_at: %v", err)
		}
		return task
	}
	mkTask(per.ID, "class presentation", "assignment", duePres)
	mkTask(eco.ID, "old lab report", "coursework", dueOver)
	done := mkTask(per.ID, "done quiz", "test", dueDone)
	if _, err := tasks.CompleteTask(ctx.DB, ctx.DataRoot, nil, per.ID, done.ID, "completed"); err != nil {
		t.Fatalf("complete task: %v", err)
	}
}

func openPhaseB(t *testing.T) (*app.Context, *GuiApi) {
	t.Helper()
	ctx, err := app.Open(app.OpenOptions{DataRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("app.Open: %v", err)
	}
	t.Cleanup(func() { ctx.Close() })
	seedPhaseB(t, ctx)
	api := NewGuiApi(ctx)
	api.now = func() time.Time { return fixedNow }
	return ctx, api
}

func golden(t *testing.T, name string, v any) {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	raw = append(raw, '\n')
	path := filepath.Join("testdata", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", name, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with UPDATE_GOLDEN=1 to create): %v", name, err)
	}
	if string(raw) != string(want) {
		t.Fatalf("golden %s mismatch; run with UPDATE_GOLDEN=1 after a deliberate change", name)
	}
}

func TestGetVersion(t *testing.T) {
	api := NewGuiApi(&app.Context{})
	v, err := api.GetVersion()
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if !strings.Contains(v, app.Name) {
		t.Fatalf("GetVersion %q does not mention %q", v, app.Name)
	}
}

func TestGetDashboardMatchesAnalytics(t *testing.T) {
	ctx, api := openPhaseB(t)

	d, err := api.GetDashboard()
	if err != nil {
		t.Fatalf("GetDashboard: %v", err)
	}

	// Parity with the single sources of truth (docs/cs/01: TS never re-derives).
	ucs, err := analytics.UnitCoverages(ctx.DB, false, fixedNow)
	if err != nil {
		t.Fatalf("UnitCoverages: %v", err)
	}
	sum, err := analytics.Summarize(ctx.DB, ucs, fixedNow)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if d.Summary.TotalTopics != sum.TotalTopics || d.Summary.Read != sum.Read ||
		d.Summary.Pending != sum.Pending || d.Summary.Unread != sum.Unread ||
		d.Summary.ActiveTasks != sum.ActiveTasks || d.Summary.OverdueTasks != sum.OverdueTasks {
		t.Fatalf("dashboard summary %+v != analytics %+v", d.Summary, sum)
	}
	if len(d.Units) != len(ucs) {
		t.Fatalf("dashboard has %d cards, analytics %d units", len(d.Units), len(ucs))
	}
	up, err := analytics.Upcoming(ctx.DB, fixedNow, upcomingDays)
	if err != nil {
		t.Fatalf("Upcoming: %v", err)
	}
	if len(d.Upcoming) != len(up) {
		t.Fatalf("dashboard upcoming %d != analytics %d", len(d.Upcoming), len(up))
	}
	// Fixture expectations: Periodicity 1/1/3 at 20%, Ecology empty "—".
	var per, eco *UnitCardDTO
	for i := range d.Units {
		switch d.Units[i].Name {
		case "Periodicity":
			per = &d.Units[i]
		case "Ecology":
			eco = &d.Units[i]
		}
	}
	if per == nil || eco == nil {
		t.Fatalf("missing unit cards: %+v", d.Units)
	}
	if per.Total != 5 || per.Read != 1 || per.Pending != 1 || per.Unread != 3 {
		t.Fatalf("Periodicity counts wrong: %+v", per)
	}
	if per.CoverageText != "20%" {
		t.Fatalf("Periodicity coverage %q, want 20%%", per.CoverageText)
	}
	if eco.HasTopics || eco.CoverageText != "—" {
		t.Fatalf("empty unit must report — : %+v", eco)
	}
	if len(d.Overdue) != 1 || d.Overdue[0].Title != "old lab report" {
		t.Fatalf("overdue wrong: %+v", d.Overdue)
	}
	if len(d.Upcoming) != 1 || d.Upcoming[0].Title != "class presentation" {
		t.Fatalf("upcoming wrong: %+v", d.Upcoming)
	}
	first := d.Upcoming[0]
	if first.Distance != "in 2d" || first.ProgressPct == nil {
		t.Fatalf("distance/progress wrong: %+v", first)
	}
	y, w := timeline.WeekOf(fixedNow)
	if d.Week.Year != y || d.Week.Week != w || d.Week.Title != timeline.Title(y, w) {
		t.Fatalf("dashboard week wrong: %+v", d.Week)
	}

	golden(t, "dashboard.json", d)
}

func TestGetWeekMatchesTimeline(t *testing.T) {
	ctx, api := openPhaseB(t)
	y, w := timeline.WeekOf(fixedNow)
	got, err := api.GetWeek(y, w)
	if err != nil {
		t.Fatalf("GetWeek: %v", err)
	}
	want, err := timeline.Build(ctx.DB, y, w, fixedNow)
	if err != nil {
		t.Fatalf("timeline.Build: %v", err)
	}
	if got.Title != timeline.Title(y, w) || len(got.Days) != 7 {
		t.Fatalf("week shape wrong: %+v", got)
	}
	n := 0
	for _, d := range want.Days {
		n += len(d.Items)
	}
	m := 0
	for _, d := range got.Days {
		m += len(d.Items)
	}
	if n != m || len(want.Overdue) != len(got.Overdue) {
		t.Fatalf("week items diverge: timeline day=%d over=%d, gui day=%d over=%d",
			n, len(want.Overdue), m, len(got.Overdue))
	}
	if _, err := api.GetWeek(2026, 99); err == nil {
		t.Fatalf("week 99 must fail validation")
	}
	prev, err := api.GetWeekOffset(-1)
	if err != nil {
		t.Fatalf("GetWeekOffset: %v", err)
	}
	py, pw := timeline.WeekOf(fixedNow)
	py, pw = timeline.Offset(py, pw, -1)
	if prev.Year != py || prev.Week != pw {
		t.Fatalf("offset week wrong: %+v", prev)
	}
	// Finished work never speaks overdue language.
	for _, d := range got.Days {
		for _, it := range d.Items {
			if it.Status == "completed" && it.Distance != "done" {
				t.Fatalf("completed task distance %q, want done", it.Distance)
			}
		}
	}
	golden(t, "week.json", got)
}

func TestGetMonthGrid(t *testing.T) {
	_, api := openPhaseB(t)
	m, err := api.GetMonth(2026, 10)
	if err != nil {
		t.Fatalf("GetMonth: %v", err)
	}
	if m.Title != "October 2026" || len(m.Days) != 42 {
		t.Fatalf("month shape wrong: title %q days %d", m.Title, len(m.Days))
	}
	// Grid starts on a Monday and covers Oct 1 in-month.
	if m.Days[0].Date != "2026-09-28" {
		t.Fatalf("grid starts %s, want 2026-09-28", m.Days[0].Date)
	}
	inMonth := 0
	counts := 0
	for _, d := range m.Days {
		if d.InMonth {
			inMonth++
		}
		counts += d.Count
	}
	if inMonth != 31 {
		t.Fatalf("in-month cells %d, want 31", inMonth)
	}
	// Oct 1 (done quiz) + Oct 3 (overdue report sits in anchor, but its
	// original day still counts) + Oct 6 is outside October grid? No: Oct 6
	// is in-month. Expect 3 counted day-items total.
	if counts != 3 {
		t.Fatalf("month item count %d, want 3", counts)
	}
	if m.OverdueCount != 1 {
		t.Fatalf("month overdue %d, want 1", m.OverdueCount)
	}
	if _, err := api.GetMonth(2026, 13); err == nil {
		t.Fatalf("month 13 must fail validation")
	}
	cur, err := api.GetMonthOffset(0)
	if err != nil {
		t.Fatalf("GetMonthOffset: %v", err)
	}
	if cur.Year != 2026 || cur.Month != 10 {
		t.Fatalf("current month wrong: %+v", cur)
	}
	golden(t, "month.json", m)
}

func TestGetAnalytics(t *testing.T) {
	_, api := openPhaseB(t)
	a, err := api.GetAnalytics()
	if err != nil {
		t.Fatalf("GetAnalytics: %v", err)
	}
	if a.Statuses.NotStarted != 2 || a.Statuses.Completed != 1 || a.Statuses.Overdue != 1 {
		t.Fatalf("status counts wrong: %+v", a.Statuses)
	}
	if len(a.Histogram) != 14 {
		t.Fatalf("histogram days %d, want 14", len(a.Histogram))
	}
	// Oct 6 column carries the presentation.
	found := false
	for _, h := range a.Histogram {
		if h.Date == "2026-10-06" {
			found = true
			if h.Count != 1 || h.Worst != "warn" {
				t.Fatalf("Oct 6 column wrong: %+v", h)
			}
		}
	}
	if !found {
		t.Fatalf("Oct 6 missing from histogram")
	}
	// NextDueDays drives table sorting without display parsing.
	var per *UnitCardDTO
	for i := range a.Units {
		if a.Units[i].Name == "Periodicity" {
			per = &a.Units[i]
		}
	}
	if per == nil || per.NextDueDays == nil || *per.NextDueDays != 2 {
		t.Fatalf("next due days wrong: %+v", per)
	}
	golden(t, "analytics.json", a)
}

func TestGetTaskDetail(t *testing.T) {
	_, api := openPhaseB(t)
	d, err := api.GetTaskDetail("unit-001", "task-001")
	if err != nil {
		t.Fatalf("GetTaskDetail: %v", err)
	}
	if d.Task.Title != "class presentation" || d.Task.UnitName != "Periodicity" {
		t.Fatalf("detail wrong: %+v", d.Task)
	}
	if _, err := api.GetTaskDetail("unit-001", "task-999"); err == nil {
		t.Fatalf("unknown task must fail")
	} else if gerr, ok := err.(*GuiError); !ok || gerr.Code != ErrNotFound {
		t.Fatalf("unknown task error wrong: %v", err)
	}
}

func TestGetDayItems(t *testing.T) {
	_, api := openPhaseB(t)
	// Oct 6: the presentation in its day bucket.
	day, err := api.GetDayItems("2026-10-06")
	if err != nil {
		t.Fatalf("GetDayItems: %v", err)
	}
	if len(day.Items) != 1 || day.Items[0].Title != "class presentation" {
		t.Fatalf("Oct 6 items wrong: %+v", day.Items)
	}
	// Oct 3: empty bucket, overdue-anchored report attributed to its day.
	over, err := api.GetDayItems("2026-10-03")
	if err != nil {
		t.Fatalf("GetDayItems: %v", err)
	}
	if len(over.Items) != 0 || len(over.Overdue) != 1 {
		t.Fatalf("Oct 3 wrong: %+v", over)
	}
	for _, bad := range []string{"2026-13-01", "2026-10-32", "tomorrow"} {
		if _, err := api.GetDayItems(bad); err == nil {
			t.Fatalf("date %q must fail validation", bad)
		}
	}
}

func TestGuiErrorText(t *testing.T) {
	e := fail(ErrValidation, "bad status", "want unread|pending|read")
	if !strings.Contains(e.Error(), "VALIDATION") || !strings.Contains(e.Error(), "bad status") {
		t.Fatalf("unexpected error text %q", e.Error())
	}
}
