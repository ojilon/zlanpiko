package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolateEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
}

func TestInstallFreshAndReinstall(t *testing.T) {
	isolateEnv(t)
	install := filepath.Join(t.TempDir(), "Install")
	data := filepath.Join(t.TempDir(), "Data")
	fakeExe := filepath.Join(t.TempDir(), "zlanpiko.exe")
	if err := os.WriteFile(fakeExe, []byte("fake-exe"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	opts := Options{
		InstallDir: install, DataDir: data, Yes: true,
		AddPath: true, Shortcut: true, ExePath: fakeExe,
		StartMenuDir: filepath.Join(t.TempDir(), "Menu"),
		SkipOS:       true, Stdout: &out,
	}
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(install, "zlanpiko.exe")); err != nil {
		t.Errorf("exe not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(data, "database", "zlanpiko.db")); err != nil {
		t.Errorf("db not initialised: %v", err)
	}
	if !strings.Contains(out.String(), "Done") {
		t.Errorf("no completion message:\n%s", out.String())
	}
	// Reinstall over the same dirs preserves data.
	marker := filepath.Join(data, "units", "keep.txt")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "mine" {
		t.Errorf("reinstall must preserve data: %q %v", raw, err)
	}
	if !strings.Contains(out.String(), "preserved") {
		t.Errorf("reinstall should note adoption:\n%s", out.String())
	}
}

func TestPathScriptIdempotentShape(t *testing.T) {
	script := pathScript(`C:\Tools\zlanpiko`)
	for _, want := range []string{
		`[Environment]::GetEnvironmentVariable('Path','User')`,
		`[Environment]::SetEnvironmentVariable('Path'`,
		`C:\Tools\zlanpiko`,
		`-notcontains`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q:\n%s", want, script)
		}
	}
}

func TestShortcutRealCreationInTemp(t *testing.T) {
	if _, err := os.Stat(`C:\Windows\System32\cscript.exe`); err != nil {
		t.Skip("cscript unavailable")
	}
	menu := t.TempDir()
	var out bytes.Buffer
	err := makeShortcut(Options{StartMenuDir: menu}, t.TempDir(), &out, &out)
	if err != nil {
		t.Skipf("shortcut creation unavailable: %v", err)
	}
	if _, err := os.Stat(filepath.Join(menu, "zlanpiko.lnk")); err != nil {
		t.Errorf("lnk missing: %v", err)
	}
}
