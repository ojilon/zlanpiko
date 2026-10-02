package cli

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"zlanpiko/internal/app"
	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
)

const tasksHelp = `Usage:
  zlanpiko tasks list [--unit ID] [--kind K] [--status S|overdue] [--format text|json]
  zlanpiko tasks add --unit ID --title TITLE --kind K [--due YYYY-MM-DD[THH:MM]] [--priority P]
  zlanpiko tasks edit --unit ID --task ID [--title T] [--kind K] [--status S]
                     [--priority P] [--description D] [--notes N] [--due DATE] [--due= to clear]
  zlanpiko tasks complete|submit --unit ID --task ID
  zlanpiko tasks delete --unit ID --task ID [--yes]
  zlanpiko tasks deadlines [--days N] [--unit ID] [--format text|json]
`

func taskRows(list []domain.Task, now time.Time, withUnit bool) [][]string {
	rows := make([][]string, 0, len(list))
	for _, t := range list {
		row := []string{t.ID, t.Title, string(t.Kind), displayStatus(t, now), formatDue(t.DueAt), string(t.Priority)}
		if withUnit {
			row = append([]string{t.UnitID}, row...)
		}
		rows = append(rows, row)
	}
	return rows
}

func taskHeaders(withUnit bool) []string {
	if withUnit {
		return []string{"UNIT", "ID", "TITLE", "KIND", "STATUS", "DUE", "PRIORITY"}
	}
	return []string{"ID", "TITLE", "KIND", "STATUS", "DUE", "PRIORITY"}
}

func runTasks(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, tasksHelp)
		return ExitUsage
	}
	f, err := parseFlags(args[1:])
	if err != nil {
		return usageError(stderr, "%v\n%s", err, tasksHelp)
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	now := time.Now()
	switch args[0] {
	case "list":
		list, err := tasks.ListTasks(ctx.DB, tasks.Filter{
			UnitID: f.get("unit"), Kind: f.get("kind"), Status: f.get("status"),
		}, now)
		if err != nil {
			return runtimeError(stderr, err)
		}
		withUnit := f.get("unit") == ""
		if format == "json" {
			return encode(stdout, stderr, list)
		}
		printTable(stdout, taskHeaders(withUnit), taskRows(list, now, withUnit))
		return ExitOK
	case "add":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		title, err := need(f, "title")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		var due *time.Time
		if f.has("due") {
			if due, err = tasks.ParseDue(f.get("due")); err != nil {
				return usageError(stderr, "%v", err)
			}
		}
		t, err := tasks.CreateTask(ctx.DB, ctx.DataRoot, ctx.Logger,
			unit, title, f.get("kind"), due, f.get("priority"), f.get("description"))
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "created %s/%s (%s)\n", t.UnitID, t.ID, t.Title)
		return ExitOK
	case "edit":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		task, err := need(f, "task")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		var up tasks.Update
		if f.has("title") {
			v := f.get("title")
			up.Title = &v
		}
		if f.has("kind") {
			v := f.get("kind")
			up.Kind = &v
		}
		if f.has("status") {
			v := f.get("status")
			up.Status = &v
		}
		if f.has("priority") {
			v := f.get("priority")
			up.Priority = &v
		}
		if f.has("description") {
			v := f.get("description")
			up.Description = &v
		}
		if f.has("notes") {
			v := f.get("notes")
			up.Notes = &v
		}
		if f.has("due") {
			up.SetDue = true
			if f.get("due") != "" {
				due, err := tasks.ParseDue(f.get("due"))
				if err != nil {
					return usageError(stderr, "%v", err)
				}
				up.Due = due
			}
		}
		t, err := tasks.UpdateTask(ctx.DB, ctx.DataRoot, ctx.Logger, unit, task, up)
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "updated %s/%s\n", t.UnitID, t.ID)
		return ExitOK
	case "complete", "submit":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		task, err := need(f, "task")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		status := string(domain.TaskCompleted)
		if args[0] == "submit" {
			status = string(domain.TaskSubmitted)
		}
		t, err := tasks.CompleteTask(ctx.DB, ctx.DataRoot, ctx.Logger, unit, task, status)
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "%s/%s is now %s\n", t.UnitID, t.ID, t.Status)
		return ExitOK
	case "delete":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		task, err := need(f, "task")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		if err := tasks.DeleteTask(ctx.DB, ctx.DataRoot, ctx.Logger, unit, task, f.yes()); err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "deleted %s/%s\n", unit, task)
		return ExitOK
	case "deadlines":
		days := 14
		if f.has("days") {
			days, err = strconv.Atoi(f.get("days"))
			if err != nil || days < 0 {
				return usageError(stderr, "invalid --days %q", f.get("days"))
			}
		}
		horizon := now.AddDate(0, 0, days)
		list, err := tasks.ListTasks(ctx.DB, tasks.Filter{
			UnitID:    f.get("unit"),
			DueBefore: &horizon,
		}, now)
		if err != nil {
			return runtimeError(stderr, err)
		}
		overdue, upcoming := []domain.Task{}, []domain.Task{}
		for _, t := range list {
			switch {
			case t.Overdue(now):
				overdue = append(overdue, t)
			case t.Status == domain.TaskCompleted || t.Status == domain.TaskSubmitted:
				continue // history, not upcoming work
			default:
				upcoming = append(upcoming, t)
			}
		}
		withUnit := f.get("unit") == ""
		if format == "json" {
			return encode(stdout, stderr, map[string]any{"overdue": overdue, "upcoming": upcoming})
		}
		if len(overdue) > 0 {
			fmt.Fprintln(stdout, "OVERDUE")
			printTable(stdout, taskHeaders(withUnit), taskRows(overdue, now, withUnit))
		}
		if len(upcoming) > 0 {
			fmt.Fprintln(stdout, "UPCOMING")
			printTable(stdout, taskHeaders(withUnit), taskRows(upcoming, now, withUnit))
		}
		if len(overdue) == 0 && len(upcoming) == 0 {
			fmt.Fprintln(stdout, "no deadlines")
		}
		return ExitOK
	default:
		return usageError(stderr, "unknown tasks command %q\n%s", args[0], tasksHelp)
	}
}
