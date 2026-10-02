package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"zlanpiko/internal/app"
	"zlanpiko/internal/exporter"
)

const exportHelp = `Usage:
  zlanpiko export [--unit ID] [--format txt|json] [--out FILE]

Renders the academic status report from live rows (default: text to
stdout). --out writes a file instead (created, never overwritten).
`

func runExport(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags(args)
	if err != nil {
		return usageError(stderr, "%v\n%s", err, exportHelp)
	}
	if err := rejectUnknown(f, "unit", "format", "out"); err != nil {
		return usageError(stderr, "%v", err)
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if format != "text" && format != "json" {
		format = "text"
	}
	now := time.Now()
	var report string
	if format == "json" {
		raw, err := exporter.JSONExport(ctx.DB, f.get("unit"), now)
		if err != nil {
			return runtimeError(stderr, err)
		}
		report = string(raw)
	} else {
		report, err = exporter.TextReport(ctx.DB, f.get("unit"), now)
		if err != nil {
			return runtimeError(stderr, err)
		}
	}
	if out := f.get("out"); out != "" {
		if _, err := os.Lstat(out); err == nil {
			return runtimeError(stderr, fmt.Errorf("refusing to overwrite %s", out))
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return runtimeError(stderr, err)
		}
		if err := os.WriteFile(out, []byte(report), 0o644); err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "wrote %s\n", out)
		return ExitOK
	}
	fmt.Fprint(stdout, report)
	return ExitOK
}
