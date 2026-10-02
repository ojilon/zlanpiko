package cli

import (
	"fmt"
	"io"

	"zlanpiko/internal/app"
	"zlanpiko/internal/services"
)

const topicsHelp = `Usage:
  zlanpiko topics list --unit ID [--status unread|pending|read] [--format text|json]
  zlanpiko topics add --unit ID --name NAME [--priority low|normal|high]
  zlanpiko topics status --unit ID --topic ID --status unread|pending|read
  zlanpiko topics rename --unit ID --topic ID --name NAME
  zlanpiko topics move --unit ID --topic ID --to UNIT
  zlanpiko topics delete --unit ID --topic ID [--yes]
`

func runTopics(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, topicsHelp)
		return ExitUsage
	}
	f, err := parseFlags(args[1:])
	if err != nil {
		return usageError(stderr, "%v\n%s", err, topicsHelp)
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	switch args[0] {
	case "list":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		topics, err := services.ListTopics(ctx.DB, unit, f.get("status"))
		if err != nil {
			return runtimeError(stderr, err)
		}
		if format == "json" {
			return encode(stdout, stderr, topics)
		}
		rows := make([][]string, 0, len(topics))
		for _, t := range topics {
			rows = append(rows, []string{t.ID, t.Name, string(t.ReadingStatus), string(t.Priority)})
		}
		printTable(stdout, []string{"ID", "NAME", "STATUS", "PRIORITY"}, rows)
		return ExitOK
	case "add":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		name, err := need(f, "name")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		t, err := services.CreateTopic(ctx.DB, ctx.DataRoot, ctx.Logger, unit, name, f.get("priority"))
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "created %s/%s (%s)\n", t.UnitID, t.ID, t.Name)
		return ExitOK
	case "status":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		topic, err := need(f, "topic")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		status, err := need(f, "status")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		t, err := services.SetTopicStatus(ctx.DB, ctx.DataRoot, ctx.Logger, unit, topic, status)
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "%s/%s is now %s\n", t.UnitID, t.ID, t.ReadingStatus)
		return ExitOK
	case "rename":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		topic, err := need(f, "topic")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		name, err := need(f, "name")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		t, err := services.RenameTopic(ctx.DB, ctx.DataRoot, ctx.Logger, unit, topic, name)
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "renamed %s/%s to %q\n", t.UnitID, t.ID, t.Name)
		return ExitOK
	case "move":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		topic, err := need(f, "topic")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		to, err := need(f, "to")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		t, err := services.MoveTopic(ctx.DB, ctx.DataRoot, ctx.Logger, unit, topic, to)
		if err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "moved to %s/%s\n", t.UnitID, t.ID)
		return ExitOK
	case "delete":
		unit, err := need(f, "unit")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		topic, err := need(f, "topic")
		if err != nil {
			return usageError(stderr, "%v", err)
		}
		if err := services.DeleteTopic(ctx.DB, ctx.DataRoot, ctx.Logger, unit, topic, f.yes()); err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "deleted %s/%s\n", unit, topic)
		return ExitOK
	default:
		return usageError(stderr, "unknown topics command %q\n%s", args[0], topicsHelp)
	}
}
