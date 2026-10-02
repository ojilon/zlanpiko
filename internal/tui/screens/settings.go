package screens

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/app"
	"zlanpiko/internal/config"
	"zlanpiko/internal/database"
	"zlanpiko/internal/tui/styles"
)

// Settings shows configuration, storage locations, theme control, database
// verification and the version (see docs/07).
type SettingsModel struct {
	shared *Shared
	lines  string
	msg    string
	theme  string
}

// NewSettings builds the settings screen.
func NewSettings(shared *Shared) *SettingsModel {
	return &SettingsModel{shared: shared}
}

func (s *SettingsModel) ID() ScreenID { return Settings }

func (s *SettingsModel) SetSize(_, _ int) {}

// Reload rebuilds the settings content.
func (s *SettingsModel) Reload() error {
	user, err := config.LoadUser(rootOf(s.shared))
	if err != nil {
		return err
	}
	s.theme = user.Theme
	ucs, err := analytics.UnitCoverages(s.shared.Ctx.DB, true, time.Now())
	if err != nil {
		return err
	}
	units, archived := 0, 0
	for _, uc := range ucs {
		if uc.Status == "archived" {
			archived++
		} else {
			units++
		}
	}
	var b strings.Builder
	b.WriteString(styles.Header("Settings") + "\n\n")
	b.WriteString(fmt.Sprintf("Application   %s\n", app.Info()))
	b.WriteString(fmt.Sprintf("Data root     %s\n", rootOf(s.shared)))
	b.WriteString(fmt.Sprintf("Database      %s\n", database.Path(rootOf(s.shared))))
	b.WriteString(fmt.Sprintf("Theme         %s  (t cycles auto → 16 → none)\n", s.theme))
	b.WriteString(fmt.Sprintf("Units         %d active, %d archived\n", units, archived))
	b.WriteString("\n" + styles.Header("Keys") + "\n")
	b.WriteString("1–6 screens · / command · esc blur · ? help · q quit\n")
	b.WriteString("\n" + styles.Header("Maintenance") + "\n")
	b.WriteString("v verify database · exports and backups arrive in Phase 8\n")
	s.lines = b.String()
	return nil
}

// Update handles theme cycling and verification.
func (s *SettingsModel) Update(msg tea.Msg) tea.Cmd {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch key.String() {
	case "t":
		next := map[string]string{"auto": "16", "16": "none", "none": "auto"}[s.theme]
		if next == "" {
			next = "auto"
		}
		if err := config.SaveUser(rootOf(s.shared), config.User{Theme: next}); err != nil {
			s.msg = err.Error()
			return nil
		}
		s.theme = next
		styles.Build(next)
		s.msg = "theme: " + next
		return s.reloadCmd()
	case "v":
		if err := database.Verify(s.shared.Ctx.DB); err != nil {
			s.msg = err.Error()
			return nil
		}
		s.msg = "database ok"
		return s.reloadCmd()
	}
	return nil
}

func (s *SettingsModel) reloadCmd() tea.Cmd {
	return func() tea.Msg {
		if err := s.Reload(); err != nil {
			s.msg = err.Error()
		}
		return nil
	}
}

// View renders settings.
func (s *SettingsModel) View(_, _ int) string {
	var b strings.Builder
	b.WriteString(s.lines)
	if s.msg != "" {
		b.WriteString("\n" + styles.Muted(s.msg))
	}
	return b.String()
}

// Help lists settings keys.
func (s *SettingsModel) Help() string { return "t theme · v verify database" }
