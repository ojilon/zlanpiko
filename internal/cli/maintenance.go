package cli

import (
	"fmt"
	"io"

	"zlanpiko/internal/app"
	"zlanpiko/internal/maintenance"
)

const maintenanceHelp = `Usage:
  zlanpiko maintenance verify [--repair] [--format text|json]

Checks database integrity, missing/unindexed files, sidecar freshness and
leftover temp files. Verify is read-only; --repair rewrites stale sidecars
and removes temp files (file divergence always needs a human decision).
Exits 1 when issues remain after the run.
`

func runMaintenance(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "verify" {
		if len(args) == 0 {
			fmt.Fprint(stderr, maintenanceHelp)
		} else {
			fmt.Fprintf(stderr, "zlanpiko: unknown maintenance command %q\n%s", args[0], maintenanceHelp)
		}
		return ExitUsage
	}
	f, err := parseFlags(args[1:])
	if err != nil {
		return usageError(stderr, "%v\n%s", err, maintenanceHelp)
	}
	if err := rejectUnknown(f, "repair", "format"); err != nil {
		return usageError(stderr, "%v", err)
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	rep, err := maintenance.Verify(ctx.DB, ctx.DataRoot, ctx.Logger)
	if err != nil {
		return runtimeError(stderr, err)
	}
	if f.has("repair") && !rep.Clean() {
		if err := maintenance.Repair(ctx.DB, ctx.DataRoot, ctx.Logger, rep); err != nil {
			return runtimeError(stderr, err)
		}
		rep, err = maintenance.Verify(ctx.DB, ctx.DataRoot, ctx.Logger)
		if err != nil {
			return runtimeError(stderr, err)
		}
		rep.Repaired = true
	}
	if format == "json" {
		return encode(stdout, stderr, rep)
	}
	fmt.Fprintln(stdout, rep.Summary())
	for _, m := range rep.Missing {
		fmt.Fprintf(stdout, "  missing: %s\n", m)
	}
	for _, u := range rep.Unindexed {
		fmt.Fprintf(stdout, "  unindexed: %s\n", u)
	}
	for _, s := range rep.Sidecars {
		fmt.Fprintf(stdout, "  sidecar %s: %s/%s\n", s.Problem, s.UnitID, s.ID)
	}
	for _, t := range rep.TempFiles {
		fmt.Fprintf(stdout, "  temp: %s\n", t)
	}
	if !rep.Clean() {
		return ExitRuntime
	}
	return ExitOK
}
