package cli

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"zlanpiko/internal/app"
	"zlanpiko/internal/timeline"
)

const timelineHelp = `Usage:
  zlanpiko timeline [--week YYYY-Www | --offset N] [--format text|json]

Shows one ISO week of deadlines (default: this week). Overdue incomplete
items anchor to a leading OVERDUE section; dateless tasks are excluded.
`

func runTimeline(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags(args)
	if err != nil {
		return usageError(stderr, "%v\n%s", err, timelineHelp)
	}
	if err := rejectUnknown(f, "week", "offset", "format"); err != nil {
		return usageError(stderr, "%v", err)
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	now := time.Now()
	year, week := timeline.WeekOf(now)
	if f.has("week") && f.has("offset") {
		return usageError(stderr, "--week and --offset exclude each other")
	}
	if f.has("week") {
		year, week, err = timeline.ParseWeek(f.get("week"))
		if err != nil {
			return usageError(stderr, "%v", err)
		}
	}
	if f.has("offset") {
		delta, err := strconv.Atoi(f.get("offset"))
		if err != nil {
			return usageError(stderr, "invalid --offset %q", f.get("offset"))
		}
		year, week = timeline.Offset(year, week, delta)
	}
	w, err := timeline.Build(ctx.DB, year, week, now)
	if err != nil {
		return runtimeError(stderr, err)
	}
	if format == "json" {
		return encode(stdout, stderr, w)
	}
	fmt.Fprintln(stdout, timeline.Title(year, week))
	if len(w.Overdue) > 0 {
		fmt.Fprintln(stdout, "OVERDUE")
		for _, it := range w.Overdue {
			fmt.Fprintf(stdout, "  ! %s (%s, %s, due %s)\n",
				it.Task.Title, it.Task.UnitID, it.Task.Kind, formatDue(it.Task.DueAt))
		}
	}
	shown := 0
	for _, d := range w.Days {
		if len(d.Items) == 0 {
			continue
		}
		shown++
		fmt.Fprintln(stdout, d.Date.Format("Mon 2006-01-02"))
		for _, it := range d.Items {
			mark := " "
			if it.Overdue {
				mark = "!"
			}
			fmt.Fprintf(stdout, "  %s %s  %s (%s, %s, %s)\n", mark,
				it.Task.DueAt.Local().Format("15:04"), it.Task.Title,
				it.Task.UnitID, it.Task.Kind, displayStatus(it.Task, now))
		}
	}
	if shown == 0 && len(w.Overdue) == 0 {
		fmt.Fprintln(stdout, "no deadlines this week")
	}
	return ExitOK
}
