// Package app holds process-wide application metadata.
//
// Version, Commit and BuildDate are overridden at build time with -ldflags
// (see configs/build.json); the defaults describe a local dev build.
// The canonical identity lives in configs/app.json — Phase 2 wires this
// package to read it at startup instead of relying on these defaults.
package app

import "fmt"

const (
	// Name is the user-visible application name (see configs/app.json).
	Name = "zlanpiko"
	// ExeName is the main executable file name.
	ExeName = "zlanpiko.exe"
	// InstallerExeName is the installer executable file name.
	InstallerExeName = "zlanpiko-installer.exe"
	// ID is the application identifier.
	ID = "com.local.zlanpiko"
	// SchemaVersion is the current SQLite schema version (see docs/05).
	SchemaVersion = 1
)

var (
	// Version is the semantic version, e.g. "0.1.0" (dev builds: "0.1.0-dev").
	Version = "0.1.0-dev"
	// Commit is the short git SHA embedded at build time ("none" if unknown).
	Commit = "none"
	// BuildDate is the build timestamp in RFC 3339 ("unknown" if not set).
	BuildDate = "unknown"
)

// Info returns a one-line build identification string.
func Info() string {
	return fmt.Sprintf("%s %s (commit %s, built %s, schema %d)",
		Name, Version, Commit, BuildDate, SchemaVersion)
}
