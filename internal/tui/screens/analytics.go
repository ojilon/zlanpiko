package screens

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/tui/styles"
)

// AnalyticsModel shows coverage tables with attention labels and the overall
// summary. Every number is reproducible from live rows via the formulas in
// docs/10 (printed under the table).
type AnalyticsModel struct {
	shared *Shared
	table  table.Model
	msg    string
}

// NewAnalytics builds the analytics screen.
func NewAnalytics(shared *Shared) *AnalyticsModel {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "UNIT", Width: 22}, {Title: "COVERAGE", Width: 9},
			{Title: "R/P/U", Width: 11}, {Title: "TASKS", Width: 7},
			{Title: "ATTENTION", Width: 11},
		}),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	return &AnalyticsModel{shared: shared, table: t}
}

func (s *AnalyticsModel) ID() ScreenID { return Analytics }

func (s *AnalyticsModel) SetSize(width, height int) {
	s.table.SetWidth(width)
	s.table.SetHeight(max(height-5, 3))
	s.table.SetStyles(styles.TableStyles())
}

// Reload rebuilds the coverage table with attention labels.
func (s *AnalyticsModel) Reload() error {
	now := time.Now()
	ucs, err := analytics.UnitCoverages(s.shared.Ctx.DB, false, now)
	if err != nil {
		return err
	}
	rows := make([]table.Row, 0, len(ucs))
	for _, uc := range ucs {
		att, _, err := analytics.AssessUnit(s.shared.Ctx.DB, uc, now)
		if err != nil {
			return err
		}
		rows = append(rows, table.Row{
			uc.Name, uc.CoverageText(),
			fmt.Sprintf("%d/%d/%d", uc.Read, uc.Pending, uc.Unread),
			fmt.Sprintf("%d/%d", uc.ActiveTasks, uc.CompletedTasks),
			attentionText(att),
		})
	}
	s.table.SetRows(rows)
	return nil
}

// Update scrolls the table (read-only screen).
func (s *AnalyticsModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.table, cmd = s.table.Update(msg)
	return cmd
}

// View renders summary, table and formula footnotes.
func (s *AnalyticsModel) View(width, height int) string {
	s.SetSize(width, height)
	var b strings.Builder
	now := time.Now()
	ucs, _ := analytics.UnitCoverages(s.shared.Ctx.DB, false, now)
	sum, _ := analytics.Summarize(s.shared.Ctx.DB, ucs, now)
	b.WriteString(styles.Header("Analytics") + styles.Muted(fmt.Sprintf(
		"   %d units · coverage %s · %d active · %d overdue\n",
		sum.Units, sum.CoverageText(), sum.ActiveTasks, sum.OverdueTasks)))
	b.WriteString(s.table.View() + "\n")
	b.WriteString(styles.Muted("coverage = read ÷ total · R/P/U = read/pending/unread · tasks = active/completed\n"))
	b.WriteString(styles.Muted("labels: overdue, at-risk, attention, on-track, completed (planning indicators, see docs/10)"))
	if s.msg != "" {
		b.WriteString("\n" + styles.Muted(s.msg))
	}
	return b.String()
}

// Help notes the screen is read-only.
func (s *AnalyticsModel) Help() string { return "read-only · ↑↓ scroll" }

func attentionText(a analytics.Attention) string {
	switch a {
	case analytics.AttentionOverdue:
		return styles.Error(string(a))
	case analytics.AttentionAtRisk:
		return styles.Warn(string(a))
	case analytics.AttentionWatch:
		return styles.Warn(string(a))
	case analytics.AttentionCompleted:
		return styles.OK(string(a))
	default:
		return string(a)
	}
}
