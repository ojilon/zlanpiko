// Command zlanpiko-gui is the Wails desktop window for zlanpiko.
//
// It lives at the repository root (rather than cmd/gui) because `wails build`
// compiles the main package in the project root. Plain Go builds work too:
//
//	go build -o dist/zlanpiko-gui.exe .
//
// The window resolves the same data root as the CLI/TUI (--data-root,
// $ZLANPIKO_DATA or the install pointer), opens the same SQLite database
// (schema v1, migrations shared) and serves the embedded Vite build from
// dist-frontend/. It never invents storage: unconfigured + no flag is a
// startup error, never an implicit second database (see docs/cs/01,
// docs/cs/09).
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"

	"zlanpiko/internal/app"
	"zlanpiko/internal/gui"
	"zlanpiko/internal/logging"
)

// frontendFS embeds the production frontend. Rebuild it first (see
// docs/cs/10); the .gitkeep in dist-frontend keeps plain `go build ./...`
// working before the first frontend build.
//
//go:embed all:dist-frontend
var frontendFS embed.FS

// assets returns the embedded subtree Wails must serve (index.html at root).
func assets() (fs.FS, error) {
	return fs.Sub(frontendFS, "dist-frontend")
}

func run(args []string) int {
	fs := flag.NewFlagSet("zlanpiko-gui", flag.ContinueOnError)
	dataRoot := fs.String("data-root", "", "use this data root")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, err := app.Open(app.OpenOptions{
		DataRoot: *dataRoot,
		Logger:   logging.New(os.Stderr, logging.Options{Text: true}),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "zlanpiko-gui: %v\n", err)
		return 1
	}
	defer ctx.Close()

	web, err := assets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "zlanpiko-gui: frontend assets: %v\n", err)
		return 1
	}

	api := gui.NewGuiApi(ctx)
	if err := wails.Run(&options.App{
		Title:     "zlanpiko",
		Width:     1280,
		Height:    800,
		MinWidth:  960,
		MinHeight: 640,
		Frameless: true,
		Assets:    web,
		Bind:      []any{api},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "zlanpiko-gui: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:]))
}
