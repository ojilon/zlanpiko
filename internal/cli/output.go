package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"zlanpiko/internal/domain"
)

// outputFormat resolves --format (default text).
func outputFormat(f flags) (string, error) {
	format := f.get("format")
	if format == "" {
		return "text", nil
	}
	if format != "text" && format != "json" {
		return "", fmt.Errorf("invalid --format %q (want text|json)", format)
	}
	return format, nil
}

// printTable renders aligned columns (headers + rows of equal length).
func printTable(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, joinCells(headers))
	for _, r := range rows {
		fmt.Fprintln(tw, joinCells(r))
	}
	tw.Flush()
}

func joinCells(cells []string) string {
	out := ""
	for i, c := range cells {
		if i > 0 {
			out += "\t"
		}
		out += c
		if c == "" {
			out += "-"
		}
	}
	return out
}

// printJSON renders v as indented JSON.
func printJSON(w io.Writer, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(raw))
	return nil
}

// displayStatus shows the derived overdue pseudo-status when applicable.
func displayStatus(t domain.Task, now time.Time) string {
	if t.Overdue(now) {
		return domain.TaskOverdue
	}
	return string(t.Status)
}

// formatDue renders a deadline in local time, "-" when dateless.
func formatDue(due *time.Time) string {
	if due == nil {
		return "-"
	}
	return due.Local().Format("2006-01-02 15:04")
}
