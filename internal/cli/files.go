package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"zlanpiko/internal/app"
	"zlanpiko/internal/filesystem"
	"zlanpiko/internal/importer"
)

const filesHelp = `Usage:
  zlanpiko files list [PATH] [--format text|json]
  zlanpiko files import SRC --to DEST [--recursive] [--yes]
                  [--collision skip|rename|overwrite] [--include-large]
  zlanpiko files move SRC DST
  zlanpiko files rename PATH NEWNAME
  zlanpiko files delete PATH [--recursive] --yes
  zlanpiko files open PATH
  zlanpiko files search QUERY [--limit N]

Paths are root-relative (e.g. inbox, units/unit-001). SRC for import is an
external path. DEST is a unit id, unit/topic|task, or a directory
(default inbox/<today>). Imports preview first and copy nothing without
--yes (or an interactive confirmation on a TTY).
`

func runFiles(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, filesHelp)
		return ExitUsage
	}
	pos, f, err := parseMixed(args[1:])
	if err != nil {
		return usageError(stderr, "%v\n%s", err, filesHelp)
	}
	if names, ok := map[string][]string{
		"list":   {"format"},
		"import": {"to", "recursive", "yes", "collision", "include-large"},
		"move":   {},
		"rename": {},
		"delete": {"recursive", "yes"},
		"open":   {},
		"search": {"limit", "format"},
	}[args[0]]; ok {
		if err := rejectUnknown(f, names...); err != nil {
			return usageError(stderr, "%v", err)
		}
	}
	format, err := outputFormat(f)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	root := ctx.DataRoot
	switch args[0] {
	case "list":
		if len(pos) > 1 {
			return usageError(stderr, "list takes at most one PATH\n%s", filesHelp)
		}
		rel := ""
		if len(pos) == 1 {
			rel = pos[0]
		}
		info, err := filesystem.Stat(root, rel)
		if err != nil {
			return runtimeError(stderr, err)
		}
		if !info.IsDir {
			if format == "json" {
				return encode(stdout, stderr, info)
			}
			printTable(stdout, []string{"NAME", "SIZE", "MODIFIED"},
				[][]string{{info.Name, fmt.Sprint(info.Size), info.ModTime.Format("2006-01-02 15:04")}})
			return ExitOK
		}
		children, err := filesystem.List(root, rel)
		if err != nil {
			return runtimeError(stderr, err)
		}
		if format == "json" {
			return encode(stdout, stderr, children)
		}
		rows := make([][]string, 0, len(children))
		for _, c := range children {
			name := c.Name
			if c.IsDir {
				name += "/"
			}
			rows = append(rows, []string{name, fmt.Sprint(c.Size), c.ModTime.Format("2006-01-02 15:04")})
		}
		printTable(stdout, []string{"NAME", "SIZE", "MODIFIED"}, rows)
		return ExitOK
	case "import":
		if len(pos) != 1 {
			return usageError(stderr, "import needs exactly one SRC\n%s", filesHelp)
		}
		srcAbs, err := filepath.Abs(pos[0])
		if err != nil {
			return runtimeError(stderr, err)
		}
		return runImport(ctx, stdout, stderr, srcAbs, importOptions(f), f.yes())
	case "move":
		if len(pos) != 2 {
			return usageError(stderr, "move needs SRC and DST\n%s", filesHelp)
		}
		if err := filesystem.Move(root, pos[0], pos[1]); err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "moved %s to %s\n", pos[0], pos[1])
		return ExitOK
	case "rename":
		if len(pos) != 2 {
			return usageError(stderr, "rename needs PATH and NEWNAME\n%s", filesHelp)
		}
		if err := filesystem.Rename(root, pos[0], pos[1]); err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "renamed %s to %s\n", pos[0], pos[1])
		return ExitOK
	case "delete":
		if len(pos) != 1 {
			return usageError(stderr, "delete needs exactly one PATH\n%s", filesHelp)
		}
		if !f.yes() {
			return usageError(stderr, "delete requires --yes (and --recursive for directories)")
		}
		if err := filesystem.Delete(root, pos[0], f.has("recursive")); err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "deleted %s\n", pos[0])
		return ExitOK
	case "open":
		if len(pos) != 1 {
			return usageError(stderr, "open needs exactly one PATH\n%s", filesHelp)
		}
		if err := filesystem.Open(root, pos[0]); err != nil {
			return runtimeError(stderr, err)
		}
		fmt.Fprintf(stdout, "opened %s\n", pos[0])
		return ExitOK
	case "search":
		if len(pos) != 1 {
			return usageError(stderr, "search needs exactly one QUERY\n%s", filesHelp)
		}
		limit := 100
		if f.has("limit") {
			limit, err = strconv.Atoi(f.get("limit"))
			if err != nil || limit <= 0 {
				return usageError(stderr, "invalid --limit %q", f.get("limit"))
			}
		}
		hits, err := filesystem.Search(root, pos[0], limit)
		if err != nil {
			return runtimeError(stderr, err)
		}
		if format == "json" {
			return encode(stdout, stderr, hits)
		}
		rows := make([][]string, 0, len(hits))
		for _, h := range hits {
			rows = append(rows, []string{h.Rel, fmt.Sprint(h.Size)})
		}
		printTable(stdout, []string{"PATH", "SIZE"}, rows)
		return ExitOK
	default:
		return usageError(stderr, "unknown files command %q\n%s", args[0], filesHelp)
	}
}

// importOptions builds importer options from flags.
func importOptions(f flags) importer.Options {
	collision, _ := importer.ParseCollisionPolicy(f.get("collision"))
	return importer.Options{
		Dest:         f.get("to"),
		Recursive:    f.has("recursive"),
		IncludeLarge: f.has("include-large"),
		Collision:    collision,
	}
}

// runImport previews, confirms and executes an import. Without --yes it
// asks interactively on a TTY and refuses otherwise (never hangs, never
// silently imports — see docs/08, docs/16).
func runImport(ctx *app.Context, stdout, stderr io.Writer, srcAbs string, opts importer.Options, yes bool) int {
	if opts.Collision == importer.CollisionOverwrite && !yes {
		return usageError(stderr, "--collision overwrite needs explicit --yes")
	}
	pv, err := importer.Preview(ctx.DB, ctx.DataRoot, srcAbs, opts)
	if err != nil {
		return runtimeError(stderr, err)
	}
	printPreview(stdout, pv)
	if pv.Copies() == 0 {
		fmt.Fprintln(stdout, "nothing to import")
		return ExitOK
	}
	proceed, code := confirmImport(stdout, stderr, yes)
	if code != ExitOK {
		return code
	}
	if !proceed {
		fmt.Fprintln(stdout, "cancelled: nothing imported")
		return ExitOK
	}
	report, err := importer.Execute(context.Background(), ctx.DB, ctx.DataRoot, pv, ctx.Logger)
	if err != nil {
		return runtimeError(stderr, err)
	}
	fmt.Fprintf(stdout, "imported %d file(s), skipped %d, failed %d\n", report.Copied, report.Skipped, report.Failed)
	for _, r := range report.Results {
		if r.Error != "" && !strings.HasPrefix(r.Error, "skipped:") {
			fmt.Fprintf(stdout, "FAILED %s: %s\n", r.Src, r.Error)
		}
	}
	if report.Failed > 0 {
		return ExitRuntime
	}
	return ExitOK
}

// confirmImport resolves --yes or an interactive TTY confirmation.
func confirmImport(stdout, stderr io.Writer, yes bool) (bool, int) {
	if yes {
		return true, ExitOK
	}
	if stdinIsTerminal() {
		return askConfirm(stdout, "Import these files?"), ExitOK
	}
	fmt.Fprintln(stderr, "zlanpiko: refusing without confirmation (re-run with --yes)")
	return false, ExitRuntime
}

// stdinIsTerminal is a variable so tests can force the non-TTY path
// (a test binary inheriting a real console must never block on input).
var stdinIsTerminal = func() bool { return IsTerminal(os.Stdin) }

// printPreview renders the plan: copies, destination, collisions and skips.
func printPreview(stdout io.Writer, pv *importer.Plan) {
	fmt.Fprintf(stdout, "import %s\n", pv.Source)
	fmt.Fprintf(stdout, "destination: %s", pv.DestDir)
	if pv.UnitID != "" {
		fmt.Fprintf(stdout, " (unit %s", pv.UnitID)
		if pv.TopicID != "" {
			fmt.Fprintf(stdout, "/%s", pv.TopicID)
		}
		if pv.TaskID != "" {
			fmt.Fprintf(stdout, "/%s", pv.TaskID)
		}
		fmt.Fprint(stdout, ")")
	}
	fmt.Fprintln(stdout)
	rows := make([][]string, 0, len(pv.Plans))
	for _, pl := range pv.Plans {
		action := pl.Action
		if pl.Renamed {
			action = "copy (renamed)"
		}
		detail := pl.DestRel
		if pl.Action == "skip" {
			detail = pl.Reason
		} else if pl.Reason != "" && !pl.Renamed {
			detail += " [" + pl.Reason + "]"
		}
		rows = append(rows, []string{pl.Name, action, detail})
	}
	printTable(stdout, []string{"FILE", "ACTION", "DESTINATION/NOTE"}, rows)
}

// runDotImport implements `zlanpiko .`: context-aware import of the working
// directory with preview and confirmation (see docs/08).
func runDotImport(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	pos, f, err := parseMixed(args)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if err := rejectUnknown(f, "to", "recursive", "yes", "collision", "include-large"); err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(pos) != 0 {
		return usageError(stderr, "'.' takes no positional arguments (flags only)")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return runtimeError(stderr, err)
	}
	return runImport(ctx, stdout, stderr, cwd, importOptions(f), f.yes())
}

// IsTerminal reports whether f is a character device (interactive terminal).
func IsTerminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// askConfirm prompts on stdin; used only when attached to a TTY.
func askConfirm(stdout io.Writer, prompt string) bool {
	fmt.Fprintf(stdout, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
