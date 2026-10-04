// Package installer implements the first-run and reinstall setup behind
// zlanpiko-installer.exe: directory selection, skeleton + database init,
// existing-install adoption, Start-Menu shortcut and user-PATH entry
// (see docs/12). All OS integration points are injectable or skippable so
// tests never touch the real machine.
package installer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zlanpiko/internal/app"
	"zlanpiko/internal/backup"
	"zlanpiko/internal/config"
)

// Options tunes one installer run.
type Options struct {
	InstallDir string // default %LocalAppData%\Zlanpiko
	DataDir    string // default %USERPROFILE%\AcademicData
	Yes        bool   // skip confirmations (scripts/tests)
	AddPath    bool   // offer/perform user-PATH entry (default true)
	Shortcut   bool   // create Start-Menu shortcut (default true)
	ExePath    string // zlanpiko.exe to install (default: beside this exe)
	GuiExePath string // zlanpiko-gui.exe to install (default: beside this exe)
	NoGUI      bool   // skip the desktop GUI even when present
	// OS integration overrides (tests):
	StartMenuDir string // shortcut parent (default user Start Menu\Programs)
	SkipOS       bool   // skip PATH+shortcut execution (report only)
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
}

// defaultInstallDir is %LocalAppData%\Zlanpiko (os.UserCacheDir on Windows).
func defaultInstallDir() string {
	if local, err := os.UserCacheDir(); err == nil {
		return filepath.Join(local, "Zlanpiko")
	}
	return filepath.Join(".", "Zlanpiko")
}

// defaultDataDir is %USERPROFILE%\AcademicData.
func defaultDataDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "AcademicData")
	}
	return filepath.Join(".", "AcademicData")
}

// Run executes the setup flow. It is idempotent and preserves data.
func Run(opts Options) error {
	out, errOut := opts.Stdout, opts.Stderr
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	in := opts.Stdin
	if in == nil {
		in = os.Stdin
	}
	fmt.Fprintf(out, "%s\n%s\n\n", app.Name, app.Info())
	installDir := opts.InstallDir
	dataDir := opts.DataDir
	if installDir == "" {
		installDir = defaultInstallDir()
	}
	if dataDir == "" {
		dataDir = defaultDataDir()
	}
	// One shared reader: fresh bufio.Readers per prompt would discard
	// buffered typeahead and break piped answers.
	reader := bufio.NewReader(in)
	wantGUI := false
	if !opts.Yes {
		var err error
		installDir, err = askDir(reader, in, out, "Install directory", installDir, "Zlanpiko")
		if err != nil {
			return err
		}
		dataDir, err = askDir(reader, in, out, "Academic data directory", dataDir, "AcademicData")
		if err != nil {
			return err
		}
		wantGUI = askGUI(reader, out, opts)
	} else {
		_, wantGUI = guiWanted(opts)
	}
	existing := detectExisting(dataDir)
	if existing != "" {
		fmt.Fprintf(out, "Existing data found at %s — it will be preserved and adopted.\n", existing)
	}
	fmt.Fprintf(out, "Installing to %s\nData root    %s\n", installDir, dataDir)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("installer: create install dir: %w", err)
	}
	if existing != "" {
		// Pre-update safety backup before touching program files: abort
		// the update when the backup fails (docs/cs/09).
		if err := preUpdateBackup(out, dataDir); err != nil {
			return err
		}
	}
	if err := installExe(out, opts.ExePath, installDir); err != nil {
		fmt.Fprintf(out, "note: %v\n", err)
	}
	if wantGUI {
		if err := installGuiExe(out, opts.GuiExePath, installDir); err != nil {
			fmt.Fprintf(out, "note: %v\n", err)
		}
	}
	ctx, err := app.Open(app.OpenOptions{DataRoot: dataDir})
	if err != nil {
		return fmt.Errorf("installer: initialise data: %w", err)
	}
	ctx.Close()
	if err := config.SavePointer(&config.Pointer{DataRoot: dataDir, InstallDir: installDir}); err != nil {
		return fmt.Errorf("installer: save pointer: %w", err)
	}
	fmt.Fprintf(out, "Data initialised (schema v%d).\n", app.SchemaVersion)
	if opts.Shortcut {
		if err := makeShortcut(opts, installDir, out, errOut); err != nil {
			fmt.Fprintf(out, "note: shortcut skipped: %v\n", err)
		}
	}
	if opts.AddPath {
		if err := ensurePath(opts, installDir, out); err != nil {
			fmt.Fprintf(out, "note: PATH entry skipped: %v\n", err)
		}
	}
	fmt.Fprintf(out, "\nDone. %s is ready.\n", app.Name)
	fmt.Fprintf(out, "Restart your terminal before using %s from any directory.\n", app.ExeName)
	if wantGUI {
		fmt.Fprintf(out, "Desktop GUI installed as zlanpiko-gui.exe (frameless window).\n")
	}
	return nil
}

// askDir resolves one directory: Enter keeps the default, "c" opens the
// interactive drive/folder chooser (which appends appFolder), and any other
// input is used as the literal path. in is the raw input for the TTY check;
// answers come from the shared reader.
func askDir(reader *bufio.Reader, in io.Reader, out io.Writer, label, def, appFolder string) (string, error) {
	fmt.Fprintf(out, "%s\n  [%s]\n  Enter = default · c = choose drive/folder · or type a path: ", label, def)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", fmt.Errorf("installer: no input (re-run with --yes or explicit directories)")
	}
	switch choice := strings.TrimSpace(line); {
	case choice == "":
		return def, nil
	case choice == "c" || choice == "C":
		f, ok := in.(*os.File)
		if !ok || !isTerminalFile(f) {
			return "", fmt.Errorf("installer: choosing needs a terminal (pass an explicit path instead)")
		}
		parent, err := PickParent(label, appFolder, in, out)
		if err != nil {
			return "", err
		}
		return filepath.Join(parent, appFolder), nil
	default:
		return choice, nil
	}
}

// isTerminalFile reports whether f is a character device (real terminal).
func isTerminalFile(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// detectExisting reports a data root that already holds a database.
func detectExisting(dataDir string) string {
	if _, err := os.Lstat(filepath.Join(dataDir, "database", "zlanpiko.db")); err == nil {
		return dataDir
	}
	return ""
}

// preUpdateBackup writes a db-only safety backup into
// <dataDir>/backups/pre-update-<UTC stamp>.zip. It opens (and migrates) the
// existing database first, so a corrupt store aborts before exes move.
func preUpdateBackup(out io.Writer, dataDir string) error {
	ctx, err := app.Open(app.OpenOptions{DataRoot: dataDir})
	if err != nil {
		return fmt.Errorf("installer: existing data unreadable, update aborted: %w", err)
	}
	defer ctx.Close()
	stamp := time.Now().UTC().Format("20060102-150405")
	dest := filepath.Join(dataDir, "backups", "pre-update-"+stamp+".zip")
	if _, err := backup.Create(ctx.DB, dataDir, dest, false, nil); err != nil {
		return fmt.Errorf("installer: pre-update backup failed, update aborted: %w", err)
	}
	fmt.Fprintf(out, "Pre-update backup: %s\n", dest)
	return nil
}

// guiCandidate locates zlanpiko-gui.exe beside the installer (or the
// explicit path) and reports whether it exists.
func guiCandidate(guiExePath string) (string, bool) {
	if guiExePath != "" {
		if _, err := os.Stat(guiExePath); err == nil {
			return guiExePath, true
		}
		return guiExePath, false
	}
	self, err := os.Executable()
	if err != nil {
		return "", false
	}
	p := filepath.Join(filepath.Dir(self), "zlanpiko-gui.exe")
	if _, err := os.Stat(p); err == nil {
		return p, true
	}
	return "", false
}

// guiWanted reports whether a GUI payload is available and desired under
// --yes (present beside the installer, not disabled).
func guiWanted(opts Options) (path string, ok bool) {
	if opts.NoGUI {
		return "", false
	}
	return guiCandidate(opts.GuiExePath)
}

// askGUI confirms the desktop GUI payload interactively (shared reader).
func askGUI(reader *bufio.Reader, out io.Writer, opts Options) bool {
	path, present := guiWanted(opts)
	if !present {
		fmt.Fprintf(out, "note: zlanpiko-gui.exe not found beside installer; GUI skipped\n")
		return false
	}
	fmt.Fprintf(out, "Install desktop GUI (%s)? [Y/n]: ", filepath.Base(path))
	line, _ := reader.ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "" || answer == "y" || answer == "yes"
}

// installGuiExe copies the desktop GUI executable (see installExe).
func installGuiExe(out io.Writer, guiExePath, installDir string) error {
	path, _ := guiCandidate(guiExePath)
	if path == "" {
		return fmt.Errorf("desktop GUI not found beside installer; skipping GUI (main install continues)")
	}
	return copyExe(out, path, installDir, "zlanpiko-gui.exe")
}

// installExe copies the main executable next to the installer (when the
// release package layout is present) into the install directory.
func installExe(out io.Writer, exePath, installDir string) error {
	if exePath == "" {
		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot locate installed executable: %w", err)
		}
		exePath = filepath.Join(filepath.Dir(self), app.ExeName)
	}
	return copyExe(out, exePath, installDir, app.ExeName)
}

// copyExe installs one executable with one-generation rollback: the
// previous install becomes name.prev.exe (docs/cs/09).
func copyExe(out io.Writer, srcPath, installDir, name string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("executable not found (%s); configuration continues without copying", srcPath)
	}
	defer src.Close()
	dstPath := filepath.Join(installDir, name)
	if _, err := os.Lstat(dstPath); err == nil {
		prev := dstPath + ".prev.exe"
		_ = os.Remove(prev)
		if err := os.Rename(dstPath, prev); err != nil {
			return fmt.Errorf("rotate previous executable: %w", err)
		}
		fmt.Fprintf(out, "Kept previous as %s\n", prev)
	}
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("copy executable: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy executable: %w", err)
	}
	fmt.Fprintf(out, "Copied %s\n", dstPath)
	return nil
}
