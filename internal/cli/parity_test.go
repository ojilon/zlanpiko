package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
	"zlanpiko/internal/timeline"
)

func TestAnalyticsParityWithLibrary(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "Maths")
	mustOK(t, ctx, "units", "add", "--name", "Empty")
	mustOK(t, ctx, "topics", "add", "--unit", "unit-001", "--name", "T1")
	mustOK(t, ctx, "topics", "add", "--unit", "unit-001", "--name", "T2")
	mustOK(t, ctx, "topics", "add", "--unit", "unit-001", "--name", "T3")
	mustOK(t, ctx, "topics", "add", "--unit", "unit-001", "--name", "T4")
	mustOK(t, ctx, "topics", "status", "--unit", "unit-001", "--topic", "topic-001", "--status", "read")
	mustOK(t, ctx, "topics", "status", "--unit", "unit-001", "--topic", "topic-002", "--status", "read")
	mustOK(t, ctx, "topics", "status", "--unit", "unit-001", "--topic", "topic-003", "--status", "pending")
	mustOK(t, ctx, "tasks", "add", "--unit", "unit-001", "--title", "Old",
		"--kind", "assignment", "--due", "2020-01-01")
	mustOK(t, ctx, "tasks", "add", "--unit", "unit-001", "--title", "Soon",
		"--kind", "test", "--due", "2099-01-01")

	now := time.Now()
	ucs, err := analytics.UnitCoverages(ctx.DB, false, now)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := analytics.Summarize(ctx.DB, ucs, now)
	if err != nil {
		t.Fatal(err)
	}
	out := mustOK(t, ctx, "analytics", "--format", "json")
	var decoded struct {
		Units []struct {
			UnitID   string  `json:"unit_id"`
			Coverage float64 `json:"coverage"`
		} `json:"units"`
		Summary struct {
			Coverage     float64 `json:"coverage"`
			OverdueTasks int     `json:"overdue_tasks"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("analytics json: %v\n%s", err, out)
	}
	if decoded.Summary.Coverage != sum.Coverage {
		t.Errorf("overall coverage: cli %v vs library %v", decoded.Summary.Coverage, sum.Coverage)
	}
	if decoded.Summary.OverdueTasks != sum.OverdueTasks || sum.OverdueTasks != 1 {
		t.Errorf("overdue: cli %d vs library %d", decoded.Summary.OverdueTasks, sum.OverdueTasks)
	}
	byUnit := map[string]float64{}
	for _, u := range decoded.Units {
		byUnit[u.UnitID] = u.Coverage
	}
	for _, uc := range ucs {
		if byUnit[uc.UnitID] != uc.Coverage {
			t.Errorf("unit %s coverage: cli %v vs library %v", uc.UnitID, byUnit[uc.UnitID], uc.Coverage)
		}
	}
	if byUnit["unit-001"] != 50.0 {
		t.Errorf("unit-001 coverage = %v, want 50", byUnit["unit-001"])
	}
	// Empty unit reports no topics: dash in text, zero coverage + has_topics false in json.
	out = mustOK(t, ctx, "analytics", "--unit", "unit-002")
	if !strings.Contains(out, "—") {
		t.Errorf("empty unit should show —:\n%s", out)
	}
	// Overdue filter agrees between tasks list and analytics.
	overdue, err := tasks.ListTasks(ctx.DB, tasks.Filter{Status: domain.TaskOverdue}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(overdue) != sum.OverdueTasks {
		t.Errorf("tasks list overdue %d vs summary %d", len(overdue), sum.OverdueTasks)
	}
}

func TestTimelineCommands(t *testing.T) {
	ctx := testContext(t)
	mustOK(t, ctx, "units", "add", "--name", "U")
	mustOK(t, ctx, "tasks", "add", "--unit", "unit-001", "--title", "Dated",
		"--kind", "assignment", "--due", "2026-10-07T09:00")
	out := mustOK(t, ctx, "timeline", "--week", "2026-W41")
	if !strings.Contains(out, "Week 41") || !strings.Contains(out, "Dated") {
		t.Errorf("timeline output:\n%s", out)
	}
	out = mustOK(t, ctx, "timeline", "--week", "2026-W41", "--format", "json")
	if !strings.Contains(out, `"week": 41`) || !strings.Contains(out, "Dated") {
		t.Errorf("timeline json:\n%s", out)
	}
	// Navigation and validation.
	y, w := timeline.WeekOf(time.Now())
	ey, ew := timeline.Offset(y, w, 1)
	out = mustOK(t, ctx, "timeline", "--offset", "1")
	if !strings.Contains(out, fmt.Sprintf("Week %d", ew)) || !strings.Contains(out, fmt.Sprintf("%d-", ey)) {
		t.Errorf("offset output:\n%s", out)
	}
	for _, args := range [][]string{
		{"timeline", "--week", "tomorrow"},
		{"timeline", "--week", "2026-W41", "--offset", "1"},
		{"timeline", "--days", "x"},
	} {
		if code, _, _ := runArgs(t, ctx, args...); code != ExitUsage {
			t.Errorf("%v exit = %d, want %d", args, code, ExitUsage)
		}
	}
}
