package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withIsolatedConfigDir points the pointer file at a temp dir.
func withIsolatedConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("APPDATA", dir) // os.UserConfigDir reads %APPDATA% on Windows
}

func TestPointerRoundTrip(t *testing.T) {
	withIsolatedConfigDir(t)
	want := &Pointer{DataRoot: `D:\ZlanpikoData`, InstallDir: `C:\Program Files\zlanpiko`}
	if err := SavePointer(want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPointer()
	if err != nil {
		t.Fatal(err)
	}
	if *got != *want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveDataRootPrecedence(t *testing.T) {
	withIsolatedConfigDir(t)
	if err := SavePointer(&Pointer{DataRoot: `P:\pointer`}); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvDataRoot, `P:\env`)

	if got, _ := ResolveDataRoot(`P:\explicit`); got != `P:\explicit` {
		t.Errorf("explicit: got %q", got)
	}
	if got, _ := ResolveDataRoot(""); got != `P:\env` {
		t.Errorf("env: got %q", got)
	}
	t.Setenv(EnvDataRoot, "")
	if got, _ := ResolveDataRoot(""); got != `P:\pointer` {
		t.Errorf("pointer: got %q", got)
	}
}

func TestResolveDataRootMissingIsError(t *testing.T) {
	withIsolatedConfigDir(t) // no pointer file here
	t.Setenv(EnvDataRoot, "")
	if _, err := ResolveDataRoot(""); err == nil {
		t.Error("expected error when no data root is configured")
	}
}

func TestLoadPointerMalformedIsError(t *testing.T) {
	withIsolatedConfigDir(t)
	path, err := PointerPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPointer(); err == nil {
		t.Error("expected error for malformed pointer JSON")
	}
}

func TestUserPrefsDefaultsAndRoundTrip(t *testing.T) {
	root := t.TempDir()
	u, err := LoadUser(root) // absent file -> defaults, not an error
	if err != nil {
		t.Fatal(err)
	}
	if u != DefaultUser() {
		t.Errorf("got %+v, want defaults %+v", u, DefaultUser())
	}
	want := User{Theme: "none", LastScreen: "timeline"}
	if err := SaveUser(root, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadUser(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadUserMalformedIsError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(UserPath(root), []byte("{nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadUser(root); err == nil {
		t.Error("expected error for malformed user JSON")
	}
}
