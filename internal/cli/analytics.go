package cli

import (
	"fmt"
	"io"
	"time"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/app"
	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
)

// taskListOverdue counts a unit's derived-overdue tasks.
func taskListOverdue(ctx *app.Context, unit string, now time.Time) (int, error) {
	list, err := tasks.ListTasks(ctx.DB, tasks.Filter{UnitID: unit, Status: domain.TaskOverdue}, now)
	if err != nil {
		return 0, err
	}
	return len(list), nil
}

const analyticsHelp = `Usage:
  zlanpiko analytics [--unit ID] [--format text|json]

Reading coverage (read/total) with attention labels. Empty units show —,
never a fake percentage. Labels are planning indicators (see docs/10).
`

func runAnalytics(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags(args)
	if err != nil {
		return usageError(stderr, "%v\n%s", err, analyticsHelp)
	}
	if err := rejectUnknown(f, "unit", "format"); err != nil {
		return usageError(stderr, "%v", err)
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	now := time.Now()
	ucs, err := analytics.UnitCoverages(ctx.DB, false, now)
	if err != nil {
		return runtimeError(stderr, err)
	}
	if unit := f.get("unit"); unit != "" {
		filtered := ucs[:0:0]
		for _, uc := range ucs {
			if uc.UnitID == unit {
				filtered = append(filtered, uc)
			}
		}
		if len(filtered) == 0 {
			return runtimeError(stderr, fmt.Errorf("unit %q not found", unit))
		}
		ucs = filtered
	}
	type row struct {
		analytics.UnitCoverage
		Attention string `json:"attention"`
	}
	rows := make([]row, 0, len(ucs))
	for _, uc := range ucs {
		att, _, err := analytics.AssessUnit(ctx.DB, uc, now)
		if err != nil {
			return runtimeError(stderr, err)
		}
		rows = append(rows, row{UnitCoverage: uc, Attention: string(att)})
	}
	sum, err := analytics.Summarize(ctx.DB, ucs, now)
	if err != nil {
		return runtimeError(stderr, err)
	}
	if f.get("unit") != "" {
		// Summarize counts overdue globally; narrow it to the unit.
		narrow, err := taskListOverdue(ctx, f.get("unit"), now)
		if err != nil {
			return runtimeError(stderr, err)
		}
		sum.OverdueTasks = narrow
	}
	if format == "json" {
		return encode(stdout, stderr, map[string]any{"units": rows, "summary": sum})
	}
	table := make([][]string, 0, len(rows))
	for _, r := range rows {
		table = append(table, []string{
			r.Name, r.CoverageText(),
			fmt.Sprintf("%d/%d/%d", r.Read, r.Pending, r.Unread),
			fmt.Sprintf("%d/%d", r.ActiveTasks, r.CompletedTasks),
			r.Attention,
		})
	}
	printTable(stdout, []string{"UNIT", "COVERAGE", "R/P/U", "TASKS", "ATTENTION"}, table)
	fmt.Fprintf(stdout, "overall %s · %d units · %d active · %d overdue\n",
		sum.CoverageText(), sum.Units, sum.ActiveTasks, sum.OverdueTasks)
	return ExitOK
}
