// Command academic is the single executable for the TUI and CLI.
//
// Phase 1: supports `help` and `version` only. With no arguments it prints
// help (the interactive TUI arrives in Phase 5, the full command tree in
// Phase 7 — see docs/08). Exit codes: 0 success, 2 usage error.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"zlanpiko/internal/app"
)

const usage = `Academic Manager (%s)

Usage:
  academic [command]

Commands:
  help       Show this help
  version    Show version information

Full topic, task, timeline, file, export and backup commands arrive
with later phases (see docs/08). With no arguments this help is shown;
the interactive TUI arrives in Phase 5.
`

// normalize accepts help, --help, -h and /help spellings.
func normalize(arg string) string {
	return strings.TrimLeft(arg, "-/")
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stdout, usage, app.Version)
		return 0
	}
	switch normalize(args[0]) {
	case "help", "h":
		fmt.Fprintf(stdout, usage, app.Version)
		return 0
	case "version", "v":
		fmt.Fprintf(stdout, "%s\n", app.Info())
		return 0
	default:
		fmt.Fprintf(stderr, "academic: unknown command %q (try: academic help)\n", args[0])
		return 2
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
