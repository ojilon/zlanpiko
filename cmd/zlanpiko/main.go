// Command zlanpiko is the single executable for the TUI and CLI.
//
// help and version run without storage; every other command resolves the
// data root (--data-root, $ZLANPIKO_DATA or the install pointer), opens it
// and dispatches to the CLI. With no arguments it launches the interactive
// TUI on a terminal, or prints help when piped. Exit codes: 0 success,
// 1 runtime error, 2 usage error.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/app"
	"zlanpiko/internal/cli"
	"zlanpiko/internal/logging"
	"zlanpiko/internal/tui"
)

const usage = `zlanpiko (%s)

Usage:
  zlanpiko [command] [flags] [--data-root PATH]

With no arguments the interactive TUI starts (on a terminal).

Commands:
  units      Manage course units
  topics     Manage topics and reading progress
  tasks      Manage assignments, tests and deadlines
  timeline   Show the weekly timeline
  analytics  Show coverage and attention
  files      Manage files (list import move rename delete open search)
  .          Import the working directory (preview first, needs --yes)
  help       Show this help
  version    Show version information

More commands (export, backup) arrive with later phases
(see docs/08).
`

// normalize accepts help, --help, -h and /help spellings.
func normalize(arg string) string {
	return strings.TrimLeft(arg, "-/")
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runTUI(nil, stdout, stderr)
	}
	switch normalize(args[0]) {
	case "help", "h":
		fmt.Fprintf(stdout, usage, app.Version)
		return 0
	case "version", "v":
		fmt.Fprintf(stdout, "%s\n", app.Info())
		return 0
	default:
		return runWithStorage(args, stdout, stderr)
	}
}

// runTUI launches the interactive interface, or prints help when stdout is
// not a terminal (script-friendly: no hanging on pipes).
func runTUI(dataRoot *string, stdout, stderr io.Writer) int {
	outFile, ok := stdout.(*os.File)
	if !ok || !cli.IsTerminal(outFile) {
		fmt.Fprintf(stdout, usage, app.Version)
		return 0
	}
	var explicit string
	if dataRoot != nil {
		explicit = *dataRoot
	}
	ctx, err := app.Open(app.OpenOptions{
		DataRoot: explicit,
		Logger:   logging.New(stderr, logging.Options{Text: true}),
	})
	if err != nil {
		fmt.Fprintf(stderr, "zlanpiko: %v\n", err)
		return 1
	}
	defer ctx.Close()
	if _, err := tea.NewProgram(tui.NewModel(ctx), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(stderr, "zlanpiko: TUI error: %v\n", err)
		return 1
	}
	return 0
}

// extractDataRoot pulls --data-root (both spellings) out of args.
func extractDataRoot(args []string) (rest []string, dataRoot string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--data-root" && i+1 < len(args) {
			dataRoot = args[i+1]
			i++
			continue
		}
		if v, ok := strings.CutPrefix(a, "--data-root="); ok {
			dataRoot = v
			continue
		}
		rest = append(rest, a)
	}
	return rest, dataRoot
}

func runWithStorage(args []string, stdout, stderr io.Writer) int {
	rest, dataRoot := extractDataRoot(args)
	ctx, err := app.Open(app.OpenOptions{
		DataRoot: dataRoot,
		Logger:   logging.New(stderr, logging.Options{Text: true}),
	})
	if err != nil {
		fmt.Fprintf(stderr, "zlanpiko: %v\n", err)
		return 1
	}
	defer ctx.Close()
	return cli.Run(ctx, rest, stdout, stderr)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
