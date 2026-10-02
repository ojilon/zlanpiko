// Package screens implements the TUI screens. Each screen loads its data
// through the Phase 3-4 services, renders with Lipgloss via the styles
// package, and talks to the root model through Shared (filters, navigation
// requests and flash messages). Screens never touch SQL directly.
package screens

import (
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/app"
)

// ScreenID identifies a screen; order matches the tab strip.
type ScreenID int

const (
	Dashboard ScreenID = iota
	Units
	Topics
	Tasks
	Timeline
	Files
	Analytics
	Settings
)

// Order is the tab-strip sequence.
var Order = []ScreenID{Dashboard, Units, Topics, Tasks, Timeline, Files, Analytics, Settings}

// Titles maps screens to tab labels.
var Titles = map[ScreenID]string{
	Dashboard: "Dashboard",
	Units:     "Units",
	Topics:    "Topics",
	Tasks:     "Tasks",
	Timeline:  "Timeline",
	Files:     "Files",
	Analytics: "Analytics",
	Settings:  "Settings",
}

// Keys maps screens to tab-switch digits.
var Keys = map[ScreenID]string{
	Dashboard: "1", Units: "2", Topics: "3", Tasks: "4", Timeline: "5",
	Files: "6", Analytics: "7", Settings: "8",
}

// NoNav clears navigation requests.
const NoNav = ScreenID(-1)

// Shared carries cross-screen state owned by the root model.
type Shared struct {
	Ctx *app.Context
	// TopicsUnit / TasksUnit filter those screens ("" = all).
	TopicsUnit  string
	TasksUnit   string
	TasksStatus string
	// NavTo requests a screen switch (NoNav = none).
	NavTo ScreenID
	// Flash is a one-shot status-bar message consumed by the root model.
	Flash string
}

// Screen is one tab's content area.
type Screen interface {
	ID() ScreenID
	Reload() error
	Update(msg tea.Msg) tea.Cmd
	View(width, height int) string
	Help() string
	SetSize(width, height int)
}
