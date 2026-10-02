// Package logging bootstraps the application's structured logging.
//
// Convention: services and fronts take a *slog.Logger (never the global
// default); user-facing messages stay short while diagnostic detail goes to
// the log. CLI writes logs to a file, never stdout (see docs/08).
package logging

import (
	"io"
	"log/slog"
)

// Options tunes the logger.
type Options struct {
	// Debug enables debug-level records; default is info and above.
	Debug bool
	// Text selects human-readable text output; default is JSON.
	Text bool
}

// New returns a structured logger writing to w.
func New(w io.Writer, o Options) *slog.Logger {
	level := slog.LevelInfo
	if o.Debug {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}
	if o.Text {
		return slog.New(slog.NewTextHandler(w, opts))
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}
