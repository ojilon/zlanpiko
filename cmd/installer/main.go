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
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `zlanpiko-installer — set up zlanpiko on this machine.

Usage:
  zlanpiko-installer [--install-dir DIR] [--data-dir DIR] [--yes]
                     [--no-path] [--no-shortcut]

How it works:
  1. You choose an install directory (the program) and a data directory
     (your academic records). Existing data is preserved, never wiped.
  2. For each directory you may press Enter (default), type a full path,
     or press c for the interactive chooser:
       - pick a drive from the list (type and free space shown),
       - pick one of the first 50 folders, or press r for the drive root,
       - or press t and type a folder name (more than 50? just type it)
         or a deeper subpath like Dev\Projects — it must already exist.
     The installer then creates the app folder (Zlanpiko / AcademicData)
     inside your choice and shows it for confirmation (y/n).
  3. The installer initialises the database, saves the configuration,
     offers a Start-Menu shortcut and a user-PATH entry, and prints a
     completion summary. Safe to run again for updates.

Examples:
  zlanpiko-installer
  zlanpiko-installer --install-dir D:\Tools\Zlanpiko --data-dir D:\AcademicData --yes
  zlanpiko-installer --data-dir E:\School --yes --no-shortcut

Flags:
`)
		fs.PrintDefaults()
	}
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
