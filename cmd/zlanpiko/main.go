// Command zlanpiko is the single executable for the TUI and CLI.
//
// help and version run without storage; every other command resolves the
// data root (--data-root, $ZLANPIKO_DATA or the install pointer), opens it
// and dispatches to the CLI. With no arguments it launches the interactive
// TUI on a terminal, or prints help when piped. Exit codes: 0 success,
// 1 runtime error, 2 usage error.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/app"
	"zlanpiko/internal/cli"
	"zlanpiko/internal/config"
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
  config     Show/set configuration
  maintenance Verify consistency
  export     Academic status report
  backup     Backups (create list verify restore)
  files      Manage files (list import move rename delete open search)
  .          Import the working directory (preview first, needs --yes)
  help       Show this help
  version    Show version information
`

// normalize accepts help, --help, -h and /help spellings.
func normalize(arg string) string {
	return strings.TrimLeft(arg, "-/")
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runTUI(stdout, stderr)
	}
	switch normalize(args[0]) {
	case "help", "h":
		fmt.Fprintf(stdout, usage, app.Version)
		return 0
	case "version", "v":
		if hasJSONFlag(args[1:]) {
			fmt.Fprintf(stdout, "{\"name\":%q,\"version\":%q,\"commit\":%q,\"build_date\":%q,\"schema_version\":%d}\n",
				app.Name, app.Version, app.Commit, app.BuildDate, app.SchemaVersion)
			return 0
		}
		fmt.Fprintf(stdout, "%s\n", app.Info())
		return 0
	default:
		return runWithStorage(args, stdout, stderr)
	}
}

// runTUI launches the interactive interface, or prints help when stdout is
// not a terminal (script-friendly: no hanging on pipes).
func runTUI(stdout, stderr io.Writer) int {
	outFile, ok := stdout.(*os.File)
	if !ok || !cli.IsTerminal(outFile) {
		fmt.Fprintf(stdout, usage, app.Version)
		return 0
	}
	dataRoot, err := ensureConfigured(stdout, stderr, os.Stdin)
	if err != nil {
		fmt.Fprintf(stderr, "zlanpiko: %v\n", err)
		return 1
	}
	ctx, err := app.Open(app.OpenOptions{
		DataRoot: dataRoot,
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

// ensureConfigured resolves the data root, guiding first-run setup on a
// terminal when nothing is configured yet (see docs/12). It never invents
// storage on pipes: unconfigured + non-TTY is an error.
func ensureConfigured(stdout, stderr io.Writer, stdin *os.File) (string, error) {
	if root, err := config.ResolveDataRoot(""); err == nil {
		return root, nil
	} else if !cli.IsTerminal(stdin) {
		return "", err
	}
	_ = stderr
	def := filepath.Join(homeDir(), "AcademicData")
	root, err := promptDataRoot(stdin, stdout, def)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create data root: %w", err)
	}
	if err := config.SavePointer(&config.Pointer{DataRoot: root}); err != nil {
		return "", err
	}
	fmt.Fprintf(stdout, "Data root set to %s\n", root)
	return root, nil
}

// promptDataRoot asks for the data directory (empty = default).
func promptDataRoot(stdin io.Reader, stdout io.Writer, def string) (string, error) {
	fmt.Fprintf(stdout, "Welcome to %s! First, choose where your academic data lives.\n", app.Name)
	fmt.Fprintf(stdout, "Data directory [%s]: ", def)
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil && len(strings.TrimSpace(line)) == 0 {
		return "", fmt.Errorf("no input: set %s or pass --data-root", config.EnvDataRoot)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	if abs, err := filepath.Abs(line); err == nil {
		return abs, nil
	} else {
		return "", fmt.Errorf("bad path %q: %w", line, err)
	}
}

func homeDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

// hasJSONFlag reports --format json in either spelling.
func hasJSONFlag(args []string) bool {
	for i, a := range args {
		if a == "--format=json" {
			return true
		}
		if a == "--format" && i+1 < len(args) && args[i+1] == "json" {
			return true
		}
	}
	return false
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
	if dataRoot == "" {
		var err error
		dataRoot, err = ensureConfigured(stdout, stderr, os.Stdin)
		if err != nil {
			fmt.Fprintf(stderr, "zlanpiko: %v\n", err)
			return 1
		}
	}
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
