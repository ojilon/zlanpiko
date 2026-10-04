package gui

import (
	"zlanpiko/internal/app"
)

// GuiApi is the single struct bound to the Wails runtime. All frontend calls
// go through its methods (see docs/cs/02 for the full catalogue).
type GuiApi struct {
	ctx *app.Context
}

// NewGuiApi wires the shared application context (data root, db, logger).
// The caller owns ctx; GuiApi never closes the database itself.
func NewGuiApi(ctx *app.Context) *GuiApi {
	return &GuiApi{ctx: ctx}
}

// GetVersion returns the build identification string (titlebar + settings).
func (g *GuiApi) GetVersion() (string, error) {
	return app.Info(), nil
}

// GetDashboard is a Phase A placeholder returning zero counts plus the
// version. Phase B replaces the body with live analytics rows.
func (g *GuiApi) GetDashboard() (DashboardDTO, error) {
	_ = g.ctx
	return DashboardDTO{Version: app.Info()}, nil
}
