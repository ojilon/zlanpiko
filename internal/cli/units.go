package cli

import (
	"fmt"
	"io"

	"zlanpiko/internal/app"
	"zlanpiko/internal/services"
)

const unitsHelp = `Usage:
  zlanpiko units list [--archived] [--format text|json]
  zlanpiko units add --name NAME [--code CODE] [--colour C] [--description D]
  zlanpiko units rename --unit ID --name NAME
  zlanpiko units archive|unarchive --unit ID
  zlanpiko units delete --unit ID [--yes]
  zlanpiko units show --unit ID [--format text|json]
`

func runUnits(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, unitsHelp)
		return ExitUsage
	}
	f, err := parseFlags(args[1:])
	if err != nil {
		return usageError(stderr, "%v\n%s", err, unitsHelp)
	}
	if names, ok := map[string][]string{
		"list":      {"archived", "format"},
		"add":       {"name", "code", "colour", "description"},
		"rename":    {"unit", "name"},
		"archive":   {"unit"},
		"unarchive": {"unit"},
		"delete":    {"unit", "yes"},
		"show":      {"unit", "format"},
	}[args[0]]; ok {
		if err := rejectUnknown(f, names...); err != nil {
			return usageError(stderr, "%v", err)
		}
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	switch args[0] {
	case "list":
		sums, err := services.ListUnitSummaries(ctx.DB, f.has("archived"))
		if err != nil {
			return runtimeError(stderr, err)
		}
		if format == "json" {
			return encode(stdout, stderr, sums)
		}
		rows := make([][]string, 0, len(sums))
		for _, s := range sums {
			rows = append(rows, []string{s.ID, s.Name, s.Code, string(s.Status),
				fmt.Sprint(s.Topics), fmt.Sprint(s.Tasks)})
		}
		printTable(stdout, []string{"ID", "NAME", "CODE", "STATUS", "TOPICS", "TASKS"}, rows)
		return ExitOK
	case "add":
		name, err := need(f, "name")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		u, err := services.CreateUnit(ctx.DB, ctx.DataRoot, ctx.Logger,
			name, f.get("code"), f.get("colour"), f.get("description"))
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "created %s (%s)\n", u.ID, u.Name)
		return ExitOK
	case "rename":
		id, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		name, err := need(f, "name")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		u, err := services.RenameUnit(ctx.DB, ctx.DataRoot, ctx.Logger, id, name)
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "renamed %s to %q\n", u.ID, u.Name)
		return ExitOK
	case "archive", "unarchive":
		id, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		u, err := services.SetUnitArchived(ctx.DB, ctx.DataRoot, ctx.Logger, id, args[0] == "archive")
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "%s is now %s\n", u.ID, u.Status)
		return ExitOK
	case "delete":
		id, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		if err := services.DeleteUnit(ctx.DB, ctx.DataRoot, ctx.Logger, id, f.yes()); err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "deleted %s\n", id)
		return ExitOK
	case "show":
		id, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		sums, err := services.ListUnitSummaries(ctx.DB, true)
		if err != nil {
			return runtimeError(stderr, err)
		}
		for _, s := range sums {
			if s.ID != id {
				continue
			}
			if format == "json" {
				return encode(stdout, stderr, s)
			}
			printTable(stdout,
				[]string{"ID", "NAME", "CODE", "STATUS", "TOPICS", "TASKS"},
				[][]string{{s.ID, s.Name, s.Code, string(s.Status), fmt.Sprint(s.Topics), fmt.Sprint(s.Tasks)}})
			return ExitOK
		}
		return runtimeError(stderr, fmt.Errorf("unit %q not found", id))
	default:
		return usageError(stderr, "unknown units command %q\n%s", args[0], unitsHelp)
	}
}

func encode(stdout, stderr io.Writer, v any) int {
	if err := printJSON(stdout, v); err != nil {
		return runtimeError(stderr, err)
	}
	return ExitOK
}
