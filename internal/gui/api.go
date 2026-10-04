package gui

import (
	"time"

	"zlanpiko/internal/app"
)

// GuiApi is the single struct bound to the Wails runtime. All frontend calls
// go through its methods (see docs/cs/02 for the full catalogue).
type GuiApi struct {
	ctx *app.Context
	// now supplies the current time; tests pin it for deterministic goldens.
	now func() time.Time
}

// NewGuiApi wires the shared application context (data root, db, logger).
// The caller owns ctx; GuiApi never closes the database itself.
func NewGuiApi(ctx *app.Context) *GuiApi {
	return &GuiApi{ctx: ctx, now: time.Now}
}

// GetVersion returns the build identification string (titlebar + settings).
func (g *GuiApi) GetVersion() (string, error) {
	return app.Info(), nil
}
