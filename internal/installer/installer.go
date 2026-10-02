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
		var err error
		installDir, err = askPath(in, out, "Install directory", installDir)
		if err != nil {
			return err
		}
		dataDir, err = askPath(in, out, "Academic data directory", dataDir)
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

func askPath(in io.Reader, out io.Writer, label, def string) (string, error) {
	fmt.Fprintf(out, "%s [%s]: ", label, def)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", fmt.Errorf("installer: no input (re-run with explicit directories)")
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
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
