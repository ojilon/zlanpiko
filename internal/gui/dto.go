// Package gui is the Wails-bound service surface for zlanpiko-gui.
//
// It is intentionally thin: every method resolves DTOs from the existing
// services (units, topics, tasks, timeline, analytics, filesystem, importer,
// exporter, backup, config) and returns them as JSON-friendly structs.
// Business rules stay in those packages; TypeScript renders and never
// re-derives (see docs/cs/01, docs/cs/02).
package gui

// GuiErrorCode classifies user-facing failures from bound methods.
type GuiErrorCode string

const (
	ErrValidation GuiErrorCode = "VALIDATION"
	ErrNotFound   GuiErrorCode = "NOT_FOUND"
	ErrConflict   GuiErrorCode = "CONFLICT"
	ErrStorage    GuiErrorCode = "STORAGE"
	ErrDB         GuiErrorCode = "DB"
	ErrBackup     GuiErrorCode = "BACKUP"
	ErrCancelled  GuiErrorCode = "CANCELLED"
)

// GuiError is the JSON error shape the frontend renders as toast/inline hint.
// It mirrors frontend/src/types.ts GuiError; keep both in sync (dto_test.go).
type GuiError struct {
	Code    GuiErrorCode `json:"code"`
	Message string       `json:"message"`
	Hint    string       `json:"hint,omitempty"`
}

func (e *GuiError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Hint == "" {
		return string(e.Code) + ": " + e.Message
	}
	return string(e.Code) + ": " + e.Message + " (" + e.Hint + ")"
}

func fail(code GuiErrorCode, msg, hint string) *GuiError {
	return &GuiError{Code: code, Message: msg, Hint: hint}
}

// DashboardDTO is the Phase A placeholder. Phase B expands it to the full
// home model (unit cards, attention, overdue, next-14-days; see docs/cs/05).
type DashboardDTO struct {
	Units       int    `json:"units"`
	Topics      int    `json:"topics"`
	ActiveTasks int    `json:"active_tasks"`
	Overdue     int    `json:"overdue"`
	Version     string `json:"version"`
}
