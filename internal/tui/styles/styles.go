// Package styles owns the TUI visual identity: a light-blue/grey theme with
// restrained status colours (see docs/07). All screens read styles through
// these constructors so a theme change rebuilds everything in one place.
//
// Modes: "auto" (terminal-adapted), "16" (forced 16-colour profile),
// "none" (no ANSI escapes at all; NO_COLOR is also honoured by the renderer).
package styles

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	renderer *lipgloss.Renderer
	// Plain disables all ANSI styling (layout is unchanged).
	Plain bool

	header      lipgloss.Style
	accent      lipgloss.Style
	muted       lipgloss.Style
	selected    lipgloss.Style
	errStyle    lipgloss.Style
	okStyle     lipgloss.Style
	warnStyle   lipgloss.Style
	box         lipgloss.Style
	statusStyle lipgloss.Style
	inputStyle  lipgloss.Style
)

// Build rebuilds the theme for mode auto|16|none.
func Build(mode string) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "none":
		Plain = true
		renderer = lipgloss.DefaultRenderer()
	case "16":
		Plain = os.Getenv("NO_COLOR") != ""
		renderer = lipgloss.NewRenderer(os.Stdout, termenv.WithProfile(termenv.ANSI))
	default: // auto
		Plain = os.Getenv("NO_COLOR") != ""
		renderer = lipgloss.DefaultRenderer()
	}
	new_ := renderer.NewStyle
	header = new_().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#1a4fa0", Dark: "#7aa2f7"})
	accent = new_().Foreground(lipgloss.AdaptiveColor{Light: "#1a4fa0", Dark: "#7aa2f7"})
	muted = new_().Foreground(lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#565f89"})
	selected = new_().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#16161e"}).
		Background(lipgloss.AdaptiveColor{Light: "#1a4fa0", Dark: "#7aa2f7"})
	errStyle = new_().Foreground(lipgloss.AdaptiveColor{Light: "#b91c1c", Dark: "#f7768e"})
	okStyle = new_().Foreground(lipgloss.AdaptiveColor{Light: "#15803d", Dark: "#9ece6a"})
	warnStyle = new_().Foreground(lipgloss.AdaptiveColor{Light: "#b45309", Dark: "#e0af68"})
	box = new_().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.AdaptiveColor{Light: "#9ca3af", Dark: "#565f89"}).Padding(0, 1)
	statusStyle = new_().Foreground(lipgloss.AdaptiveColor{Light: "#374151", Dark: "#a9b1d6"})
	inputStyle = new_().Foreground(lipgloss.AdaptiveColor{Light: "#111827", Dark: "#c0caf5"})
}

func init() {
	Build("auto")
}

func render(st lipgloss.Style, s string) string {
	if Plain {
		return s
	}
	return st.Render(s)
}

// Header renders top-level headings.
func Header(s string) string { return render(header, s) }

// Accent renders selections and key hints.
func Accent(s string) string { return render(accent, s) }

// Muted renders inactive text and borders.
func Muted(s string) string { return render(muted, s) }

// Selected renders the active tab/row marker.
func Selected(s string) string { return render(selected, s) }

// Error renders failures.
func Error(s string) string { return render(errStyle, s) }

// OK renders successes.
func OK(s string) string { return render(okStyle, s) }

// Warn renders warnings and attention markers.
func Warn(s string) string { return render(warnStyle, s) }

// Box draws a subtle rounded border around content.
func Box(s string) string {
	if Plain {
		return s
	}
	return box.Render(s)
}

// Status renders the status bar line.
func Status(s string) string { return render(statusStyle, s) }

// Input renders the command prompt marker.
func Input(s string) string { return render(inputStyle, s) }

// TableStyles returns the shared bubbles table styling. In Plain mode the
// selection uses reverse video (no colour) instead of the accent background.
func TableStyles() table.Styles {
	st := table.DefaultStyles()
	if !Plain {
		st.Selected = selected
		st.Header = header
		return st
	}
	plain := lipgloss.NewStyle()
	st.Header = plain.Bold(true)
	st.Selected = plain.Reverse(true)
	st.Cell = plain
	return st
}

// Bar renders a compact progress bar (█/░) usable without colour.
func Bar(frac float64, width int) string {
	if width < 1 {
		width = 1
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac*float64(width) + 0.5)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	if Plain {
		return bar
	}
	if frac >= 1 {
		return okStyle.Render(bar)
	}
	return accent.Render(bar)
}
