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

	"zlanpiko/internal/app"
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
	if !opts.Yes {
		// One shared reader: fresh bufio.Readers per prompt would discard
		// buffered typeahead and break piped answers.
		reader := bufio.NewReader(in)
		var err error
		installDir, err = askDir(reader, in, out, "Install directory", installDir, "Zlanpiko")
		if err != nil {
			return err
		}
		dataDir, err = askDir(reader, in, out, "Academic data directory", dataDir, "AcademicData")
		if err != nil {
			return err
		}
	}
	existing := detectExisting(dataDir)
	if existing != "" {
		fmt.Fprintf(out, "Existing data found at %s — it will be preserved and adopted.\n", existing)
	}
	fmt.Fprintf(out, "Installing to %s\nData root    %s\n", installDir, dataDir)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("installer: create install dir: %w", err)
	}
	if err := installExe(out, opts.ExePath, installDir); err != nil {
		fmt.Fprintf(out, "note: %v\n", err)
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
	src, err := os.Open(exePath)
	if err != nil {
		return fmt.Errorf("main executable not found beside installer (%s); configuration continues without copying", exePath)
	}
	defer src.Close()
	dstPath := filepath.Join(installDir, app.ExeName)
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
