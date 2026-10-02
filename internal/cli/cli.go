// Package cli implements the standalone command tree (see docs/08).
// Handlers are thin: parse flags, call service functions, render output.
// Business rules live in services/tasks; this package owns exit codes,
// help text and the text/json formats.
package cli

import (
	"fmt"
	"io"
	"strings"

	"zlanpiko/internal/app"
)

// Exit codes: 0 success, 1 runtime error, 2 usage error.
const (
	ExitOK      = 0
	ExitRuntime = 1
	ExitUsage   = 2
)

const topHelp = `zlanpiko — local-first academic manager

Usage:
  zlanpiko [command] [flags] [--data-root PATH]

Commands:
  units    Manage course units      (list add rename archive unarchive delete show)
  topics   Manage topics            (list add status rename move delete)
  tasks    Manage assignments etc.  (list add edit complete submit delete deadlines)
  timeline Show the weekly timeline  (--week YYYY-Www | --offset N)
  analytics Show coverage + attention [--unit ID]
  files    Manage files             (list import move rename delete open search)
  .        Import the working directory (preview first, needs --yes)
  help     Show this help
  version  Show version information

Global flags:
  --data-root PATH   Use this data root (else $ZLANPIKO_DATA or the install pointer)
  --format text|json Output format for list/show commands (default text)

Run 'zlanpiko <command> --help' for subcommand details.
`

// Run executes args against ctx. Data goes to stdout; errors to stderr.
func Run(ctx *app.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, topHelp)
		return ExitOK
	}
	switch args[0] {
	case "units":
		return runUnits(ctx, args[1:], stdout, stderr)
	case "topics":
		return runTopics(ctx, args[1:], stdout, stderr)
	case "tasks":
		return runTasks(ctx, args[1:], stdout, stderr)
	case "timeline":
		return runTimeline(ctx, args[1:], stdout, stderr)
	case "analytics":
		return runAnalytics(ctx, args[1:], stdout, stderr)
	case "files":
		return runFiles(ctx, args[1:], stdout, stderr)
	case ".":
		return runDotImport(ctx, args[1:], stdout, stderr)
	case "help", "--help", "-h", "/help":
		fmt.Fprint(stdout, topHelp)
		return ExitOK
	default:
		fmt.Fprintf(stderr, "zlanpiko: unknown command %q (try: zlanpiko help)\n", args[0])
		return ExitUsage
	}
}

// flags is the parsed flag set for one invocation.
type flags struct {
	vals    map[string]string
	present map[string]bool
}

func (f flags) get(key string) string {
	return f.vals[key]
}

func (f flags) has(key string) bool {
	return f.present[key]
}

func (f flags) yes() bool {
	return f.present["yes"]
}

// parseFlags parses --key value, --key=value and bare --bool flags.
// Positional arguments are rejected: the grammar is flags-only.
func parseFlags(args []string) (flags, error) {
	f := flags{vals: map[string]string{}, present: map[string]bool{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				return f, fmt.Errorf("did you mean -%s? flags use double dashes", a)
			}
			return f, fmt.Errorf("unexpected argument %q (flags only, e.g. --unit %s)", a, a)
		}
		key := strings.TrimPrefix(a, "--")
		if key == "" {
			return f, fmt.Errorf("empty flag '--'")
		}
		if j := strings.Index(key, "="); j >= 0 {
			f.vals[key[:j]] = key[j+1:]
			f.present[key[:j]] = true
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			f.vals[key] = args[i+1]
			f.present[key] = true
			i++
		} else {
			f.vals[key] = "true"
			f.present[key] = true
		}
	}
	return f, nil
}

// parseMixed parses flags like parseFlags but also collects positional
// arguments (used by the files commands, whose grammar mixes both).
func parseMixed(args []string) (pos []string, f flags, err error) {
	f = flags{vals: map[string]string{}, present: map[string]bool{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			if strings.HasPrefix(a, "-") && len(a) > 1 {
				return nil, f, fmt.Errorf("did you mean -%s? flags use double dashes", a)
			}
			pos = append(pos, a)
			continue
		}
		key := strings.TrimPrefix(a, "--")
		if key == "" {
			return nil, f, fmt.Errorf("empty flag '--'")
		}
		if j := strings.Index(key, "="); j >= 0 {
			f.vals[key[:j]] = key[j+1:]
			f.present[key[:j]] = true
			continue
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			// A value-looking token after a known boolean flag belongs to
			// the positionals: only --recursive/--yes/--include-large are
			// bare booleans in the files grammar.
			if key == "recursive" || key == "yes" || key == "include-large" {
				f.vals[key] = "true"
				f.present[key] = true
				continue
			}
			f.vals[key] = args[i+1]
			f.present[key] = true
			i++
		} else {
			f.vals[key] = "true"
			f.present[key] = true
		}
	}
	return pos, f, nil
}

// rejectUnknown errors on flags outside the allowed set.
func rejectUnknown(f flags, allowed ...string) error {
	ok := map[string]bool{}
	for _, a := range allowed {
		ok[a] = true
	}
	for key := range f.present {
		if !ok[key] {
			return fmt.Errorf("unknown flag --%s", key)
		}
	}
	return nil
}

// need returns a required flag or a usage error.
func need(f flags, key string) (string, error) {
	if v := f.get(key); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("missing required --%s", key)
}

// usageError prints a usage message to stderr.
func usageError(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "zlanpiko: "+format+"\n", args...)
	return ExitUsage
}

// runtimeError prints a service error to stderr.
func runtimeError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "zlanpiko: %v\n", err)
	return ExitRuntime
}
