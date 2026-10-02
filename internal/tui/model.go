// Package tui implements the terminal interface: a root model with a tab
// strip, a dynamic content area, a status bar and a persistent command
// input (see docs/07). Screens live in screens/, widgets in components/,
// the theme in styles/. All data flows through the Phase 3-4 services;
// the model never touches SQL.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/app"
	"zlanpiko/internal/config"
	"zlanpiko/internal/tui/screens"
	"zlanpiko/internal/tui/styles"
)

// Model is the root Bubble Tea model.
type Model struct {
	ctx     *app.Context
	shared  *screens.Shared
	order   []screens.ScreenID
	views   map[screens.ScreenID]screens.Screen
	active  screens.ScreenID
	width   int
	height  int
	input   textinput.Model
	focused bool
	history []string
	histPos int // -1 = typing fresh input
	draft   string
	status  string
	counts  string
	help    bool
}

// NewModel builds the TUI over an opened app context.
func NewModel(ctx *app.Context) *Model {
	shared := &screens.Shared{Ctx: ctx, NavTo: screens.NoNav}
	views := map[screens.ScreenID]screens.Screen{
		screens.Dashboard: screens.NewDashboard(shared),
		screens.Units:     screens.NewUnits(shared),
		screens.Topics:    screens.NewTopics(shared),
		screens.Tasks:     screens.NewTasks(shared),
		screens.Timeline:  screens.NewTimeline(shared),
		screens.Files:     screens.NewFiles(shared),
		screens.Analytics: screens.NewAnalytics(shared),
		screens.Settings:  screens.NewSettings(shared),
	}
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "Type a command or /help"
	m := &Model{
		ctx: ctx, shared: shared, order: screens.Order, views: views,
		active: screens.Dashboard, input: input, histPos: -1,
	}
	return m
}

// Init applies the saved theme, loads every screen and the status bar.
func (m *Model) Init() tea.Cmd {
	if user, err := config.LoadUser(m.ctx.DataRoot); err == nil {
		styles.Build(user.Theme)
	}
	for _, id := range m.order {
		if err := m.views[id].Reload(); err != nil {
			m.status = err.Error()
		}
	}
	m.refreshCounts()
	return nil
}

// Active returns the current screen (for tests).
func (m *Model) Active() screens.ScreenID { return m.active }

// Status returns the current status message (for tests).
func (m *Model) Status() string { return m.status }

// InputValue returns the persistent input text (for tests).
func (m *Model) InputValue() string { return m.input.Value() }

// Update handles window, navigation, input and screen messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	// Forward anything else (e.g. async results) to the active screen.
	cmd := m.views[m.active].Update(msg)
	m.afterScreen()
	return m, cmd
}

func (m *Model) updateKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.help {
		m.help = false
		return m, nil
	}
	if m.focused {
		return m.updateInput(key)
	}
	switch key.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		return m, tea.Quit
	case "/":
		m.focused = true
		m.histPos = -1
		return m, m.input.Focus()
	case "?":
		m.help = true
		return m, nil
	}
	for _, id := range m.order {
		if key.String() == screens.Keys[id] {
			m.switchTo(id)
			return m, nil
		}
	}
	cmd := m.views[m.active].Update(key)
	m.afterScreen()
	return m, cmd
}

func (m *Model) updateInput(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.focused = false
		m.histPos = -1
		m.input.Blur()
		return m, nil
	case "enter":
		raw := m.input.Value()
		m.input.Reset()
		m.histPos = -1
		m.focused = false
		m.input.Blur()
		return m, m.dispatch(raw)
	case "up":
		m.historyUp()
		return m, nil
	case "down":
		m.historyDown()
		return m, nil
	default:
		m.histPos = -1
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(key)
		return m, cmd
	}
}

func (m *Model) historyUp() {
	if len(m.history) == 0 {
		return
	}
	if m.histPos == -1 {
		m.draft = m.input.Value()
		m.histPos = len(m.history) - 1
	} else if m.histPos > 0 {
		m.histPos--
	}
	m.input.SetValue(m.history[m.histPos])
}

func (m *Model) historyDown() {
	if m.histPos == -1 {
		return
	}
	if m.histPos < len(m.history)-1 {
		m.histPos++
		m.input.SetValue(m.history[m.histPos])
	} else {
		m.histPos = -1
		m.input.SetValue(m.draft)
	}
}

// switchTo activates a screen and reloads it.
func (m *Model) switchTo(id screens.ScreenID) {
	m.active = id
	if err := m.views[id].Reload(); err != nil {
		m.status = err.Error()
	} else if m.status == "" {
		m.status = ""
	}
	m.refreshCounts()
}

// afterScreen consumes navigation requests and flash messages, then
// refreshes the status bar (cheap local queries).
func (m *Model) afterScreen() {
	if m.shared.NavTo != screens.NoNav {
		id := m.shared.NavTo
		m.shared.NavTo = screens.NoNav
		m.switchTo(id)
	}
	if m.shared.Flash != "" {
		m.status = m.shared.Flash
		m.shared.Flash = ""
	}
	m.refreshCounts()
}

// refreshCounts rebuilds the status-bar summary.
func (m *Model) refreshCounts() {
	now := time.Now()
	ucs, err := analytics.UnitCoverages(m.ctx.DB, false, now)
	if err != nil {
		return
	}
	sum, err := analytics.Summarize(m.ctx.DB, ucs, now)
	if err != nil {
		return
	}
	m.counts = fmt.Sprintf("%d units · %d topics (R%d/P%d/U%d %s) · %d active · %d overdue",
		sum.Units, sum.TotalTopics, sum.Read, sum.Pending, sum.Unread,
		sum.CoverageText(), sum.ActiveTasks, sum.OverdueTasks)
}

// View renders header, tabs, content, help line, status and input.
func (m *Model) View() string {
	if m.width < 80 || m.height < 24 {
		return "zlanpiko needs at least 80x24 (current " +
			fmt.Sprintf("%dx%d", m.width, m.height) + "). Please enlarge the terminal."
	}
	var b strings.Builder
	now := time.Now()
	_, week := now.ISOWeek()
	b.WriteString(styles.Header("zlanpiko"))
	b.WriteString(styles.Muted(fmt.Sprintf("   %s · week %d · %s\n",
		now.Format("Mon 2006-01-02"), week, app.Version)))
	tabs := make([]string, 0, len(m.order))
	compact := m.width < 110
	for _, id := range m.order {
		label := "[" + screens.Keys[id] + "] " + screens.Titles[id]
		if compact {
			label = "[" + screens.Keys[id] + "]"
		}
		if id == m.active {
			tabs = append(tabs, styles.Selected(" "+label+" "))
		} else {
			tabs = append(tabs, styles.Muted(" "+label+" "))
		}
	}
	b.WriteString(strings.Join(tabs, "") + "\n")
	b.WriteString(styles.Muted(strings.Repeat("─", min(m.width, 200))) + "\n")
	contentH := m.height - 6
	if m.help {
		b.WriteString(helpView())
	} else {
		b.WriteString(m.views[m.active].View(m.width, contentH))
	}
	b.WriteString("\n" + styles.Muted(m.views[m.active].Help()) + "\n")
	status := m.counts
	if m.status != "" {
		status = m.status
	}
	b.WriteString(styles.Status("Status: "+status) + "\n")
	prompt := m.input.View()
	if !m.focused {
		prompt = styles.Muted("> " + m.input.Value() + "  (/ to type)")
	}
	b.WriteString(prompt)
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func helpView() string {
	rows := [][]string{
		{"1–8", "switch screens"},
		{"/", "focus command input"},
		{"esc", "blur input / cancel dialog"},
		{"?", "this help"},
		{"q", "quit"},
		{"/help", "command list"},
		{"/quit", "quit"},
		{"a/r/e/c/s", "add · rename · edit · complete · submit (per screen)"},
		{"x/f/u", "topics/tasks: status · filter · all units"},
		{"D", "delete (with confirmation)"},
	}
	var b strings.Builder
	b.WriteString("Keys\n")
	for _, r := range rows {
		b.WriteString("  " + r[0] + "  " + r[1] + "\n")
	}
	b.WriteString("\n" + tuiCommandHelp + "\n")
	return b.String()
}
