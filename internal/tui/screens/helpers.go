package screens

import (
	"log/slog"
)

// rootOf returns the data root for service calls.
func rootOf(s *Shared) string { return s.Ctx.DataRoot }

// logOf returns the context logger for service calls.
func logOf(s *Shared) *slog.Logger { return s.Ctx.Logger }

// valAt returns vals[i] or "" when short (optional form fields).
func valAt(vals []string, i int) string {
	if i < len(vals) {
		return vals[i]
	}
	return ""
}
