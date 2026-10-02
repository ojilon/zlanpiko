// Command zlanpiko-installer performs first-time setup and updates:
// install location, data directory, database init, Start-Menu shortcut and
// user-PATH entry. Safe to re-run; existing data is never wiped.
package main

import (
	"flag"
	"fmt"
	"os"

	"zlanpiko/internal/installer"
)

func main() {
	fs := flag.NewFlagSet("zlanpiko-installer", flag.ContinueOnError)
	installDir := fs.String("install-dir", "", "installation directory (default %LocalAppData%\\Zlanpiko)")
	dataDir := fs.String("data-dir", "", "academic data directory (default %USERPROFILE%\\AcademicData)")
	yes := fs.Bool("yes", false, "non-interactive: accept defaults/confirmations")
	noPath := fs.Bool("no-path", false, "skip user-PATH entry")
	noShortcut := fs.Bool("no-shortcut", false, "skip Start-Menu shortcut")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	opts := installer.Options{
		InstallDir: *installDir,
		DataDir:    *dataDir,
		Yes:        *yes,
		AddPath:    !*noPath,
		Shortcut:   !*noShortcut,
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}
	if err := installer.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "zlanpiko-installer: %v\n", err)
		os.Exit(1)
	}
}
