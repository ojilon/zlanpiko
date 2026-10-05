package installer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Reinstalling over existing data writes a pre-update backup first, keeps
// the previous exes as *.prev.exe, and preserves user files.
func TestReinstallBacksUpAndRotates(t *testing.T) {
	isolateEnv(t)
	install := filepath.Join(t.TempDir(), "Install")
	data := filepath.Join(t.TempDir(), "Data")
	fakeExe := filepath.Join(t.TempDir(), "zlanpiko.exe")
	fakeGUI := filepath.Join(t.TempDir(), "zlanpiko-gui.exe")
	if err := os.WriteFile(fakeExe, []byte("fake-exe-v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fakeGUI, []byte("fake-gui-v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	run := func() string {
		var out bytes.Buffer
		opts := Options{
			InstallDir: install, DataDir: data, Yes: true,
			AddPath: false, Shortcut: false,
			ExePath: fakeExe, GuiExePath: fakeGUI,
			StartMenuDir: filepath.Join(t.TempDir(), "Menu"),
			SkipOS:       true, Stdout: &out,
		}
		if err := Run(opts); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	first := run()
	if !strings.Contains(first, "zlanpiko-gui.exe") {
		t.Fatalf("fresh install should copy the GUI:\n%s", first)
	}
	marker := filepath.Join(data, "units", "keep.txt")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fakeExe, []byte("fake-exe-v2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fakeGUI, []byte("fake-gui-v2"), 0o755); err != nil {
		t.Fatal(err)
	}
	second := run()
	if !strings.Contains(second, "Pre-update backup") {
		t.Fatalf("reinstall must back up first:\n%s", second)
	}
	matches, err := filepath.Glob(filepath.Join(data, "backups", "pre-update-*.zip"))
	if err != nil || len(matches) == 0 {
		t.Fatalf("no pre-update backup written: %v", matches)
	}
	for exe, want := range map[string]string{
		"zlanpiko.exe":     "fake-exe-v2",
		"zlanpiko-gui.exe": "fake-gui-v2",
	} {
		raw, err := os.ReadFile(filepath.Join(install, exe))
		if err != nil || string(raw) != want {
			t.Fatalf("%s not updated: %q %v", exe, raw, err)
		}
		prev, err := os.ReadFile(filepath.Join(install, exe+".prev.exe"))
		if err != nil {
			t.Fatalf("%s has no .prev rollback: %v", exe, err)
		}
		if !strings.HasSuffix(string(prev), "-v1") {
			t.Fatalf("%s.prev wrong generation: %q", exe, prev)
		}
	}
	if raw, err := os.ReadFile(marker); err != nil || string(raw) != "mine" {
		t.Fatalf("user data lost: %q %v", raw, err)
	}
}

// NoGUI skips the desktop payload while the main install proceeds.
func TestNoGUISkipsPayload(t *testing.T) {
	isolateEnv(t)
	install := filepath.Join(t.TempDir(), "Install")
	data := filepath.Join(t.TempDir(), "Data")
	fakeExe := filepath.Join(t.TempDir(), "zlanpiko.exe")
	fakeGUI := filepath.Join(t.TempDir(), "zlanpiko-gui.exe")
	if err := os.WriteFile(fakeExe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fakeGUI, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	opts := Options{
		InstallDir: install, DataDir: data, Yes: true, NoGUI: true,
		AddPath: false, Shortcut: false,
		ExePath: fakeExe, GuiExePath: fakeGUI,
		StartMenuDir: filepath.Join(t.TempDir(), "Menu"),
		SkipOS:       true, Stdout: &out,
	}
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(install, "zlanpiko-gui.exe")); !os.IsNotExist(err) {
		t.Fatalf("GUI should be skipped with NoGUI")
	}
	if _, err := os.Stat(filepath.Join(install, "zlanpiko.exe")); err != nil {
		t.Fatalf("main exe missing: %v", err)
	}
}
