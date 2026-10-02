package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// makeShortcut creates a Start-Menu entry via the stock Windows Script Host
// (no extra dependencies). Failures are advisory: setup continues.
func makeShortcut(opts Options, installDir string, out, errOut io.Writer) error {
	menuDir := opts.StartMenuDir
	if menuDir == "" {
		appData, err := os.UserConfigDir() // %AppData%
		if err != nil {
			return fmt.Errorf("locate Start Menu: %w", err)
		}
		menuDir = filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Zlanpiko")
	}
	if err := os.MkdirAll(menuDir, 0o755); err != nil {
		return err
	}
	link := filepath.Join(menuDir, "zlanpiko.lnk")
	exe := filepath.Join(installDir, "zlanpiko.exe")
	script := "Set sh = CreateObject(\"WScript.Shell\")\n" +
		"Set lnk = sh.CreateShortcut(\"" + vbsEscape(link) + "\")\n" +
		"lnk.TargetPath = \"" + vbsEscape(exe) + "\"\n" +
		"lnk.WorkingDirectory = \"" + vbsEscape(installDir) + "\"\n" +
		"lnk.Description = \"zlanpiko\"\n" +
		"lnk.Save\n"
	tmp, err := os.CreateTemp("", "zlanpiko-shortcut-*.vbs")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(script); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	if opts.SkipOS {
		fmt.Fprintf(out, "(skip) shortcut %s -> %s\n", link, exe)
		return nil
	}
	cmd := exec.Command("cscript", "//Nologo", "//E:vbscript", tmpName)
	if bout, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cscript failed: %w (%s)", err, strings.TrimSpace(string(bout)))
	}
	fmt.Fprintf(out, "Start-Menu shortcut created.\n")
	return nil
}

func vbsEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `""`)
}

// pathScript returns the PowerShell script that idempotently prepends dir to
// the *user* PATH (registry + broadcast handled by .NET, no truncation like
// setx). Pure function: unit-tested, executed only outside SkipOS.
func pathScript(dir string) string {
	return "$d = '" + psEscape(dir) + "'; " +
		"$cur = [Environment]::GetEnvironmentVariable('Path','User'); " +
		"if ($cur -eq $null) { $cur = '' }; " +
		"$parts = $cur -split ';' | Where-Object { $_ -ne '' }; " +
		"if ($parts -notcontains $d) { " +
		"[Environment]::SetEnvironmentVariable('Path', ($d + ';' + $cur).Trim(';'), 'User'); " +
		"Write-Output 'added' } else { Write-Output 'present' }"
}

func psEscape(s string) string {
	return strings.ReplaceAll(s, `'`, `''`)
}

// ensurePath adds installDir to the user PATH (idempotent).
func ensurePath(opts Options, installDir string, out io.Writer) error {
	if opts.SkipOS {
		fmt.Fprintf(out, "(skip) PATH += %s\n", installDir)
		return nil
	}
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", pathScript(installDir))
	bout, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("powershell PATH update failed: %w (%s)", err, strings.TrimSpace(string(bout)))
	}
	switch strings.TrimSpace(string(bout)) {
	case "added":
		fmt.Fprintf(out, "Added to your PATH (restart the terminal to use it).\n")
	default:
		fmt.Fprintf(out, "Already on your PATH.\n")
	}
	return nil
}
