package screens

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/domain"
	"zlanpiko/internal/tasks"
	"zlanpiko/internal/tui/styles"
)

// Dashboard is a read-only overview: counts, per-unit coverage, upcoming
// deadlines and overdue items (see docs/07).
type DashboardModel struct {
	shared   *Shared
	viewport viewport.Model
	content  string
	width    int
}

// NewDashboard builds the dashboard.
func NewDashboard(shared *Shared) *DashboardModel {
	return &DashboardModel{shared: shared, viewport: viewport.New(80, 20)}
}

func (d *DashboardModel) ID() ScreenID { return Dashboard }

func (d *DashboardModel) SetSize(width, height int) {
	d.width = width
	d.viewport.Width = width
	d.viewport.Height = height
	d.viewport.SetContent(d.content)
}

// Reload recomputes every section from live rows.
func (d *DashboardModel) Reload() error {
	now := time.Now()
	ucs, err := analytics.UnitCoverages(d.shared.Ctx.DB, false, now)
	if err != nil {
		return err
	}
	sum, err := analytics.Summarize(d.shared.Ctx.DB, ucs, now)
	if err != nil {
		return err
	}
	upcoming, err := analytics.Upcoming(d.shared.Ctx.DB, now, 14)
	if err != nil {
		return err
	}
	overdue, err := tasks.ListTasks(d.shared.Ctx.DB, tasks.Filter{Status: domain.TaskOverdue}, now)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(styles.Header("Academic overview"))
	b.WriteString(fmt.Sprintf("   %d units · %d topics · %d active tasks\n\n",
		sum.Units, sum.TotalTopics, sum.ActiveTasks))
	b.WriteString(fmt.Sprintf("Topics  read %d · pending %d · unread %d   coverage %s\n\n",
		sum.Read, sum.Pending, sum.Unread, sum.CoverageText()))
	b.WriteString(styles.Header("Units"))
	b.WriteString("\n")
	barW := d.width - 56
	if barW < 10 {
		barW = 10
	}
	if len(ucs) == 0 {
		b.WriteString(styles.Muted("no units yet — press 2, then a to add one\n"))
	}
	for _, uc := range ucs {
		name := uc.Name
		if uc.Code != "" {
			name += " [" + uc.Code + "]"
		}
		frac := 0.0
		if uc.HasTopics {
			frac = uc.Coverage / 100
		}
		b.WriteString(fmt.Sprintf("%-28.28s %s %6s  %d active\n", name,
			styles.Bar(frac, barW), uc.CoverageText(), uc.ActiveTasks))
	}
	b.WriteString("\n" + styles.Header("Needs attention") + "\n")
	attention := 0
	for _, uc := range ucs {
		att, reason, err := analytics.AssessUnit(d.shared.Ctx.DB, uc, now)
		if err != nil {
			return err
		}
		if att == analytics.AttentionOnTrack || att == analytics.AttentionCompleted || att == analytics.AttentionNone {
			continue
		}
		attention++
		marker := styles.Warn("•")
		if att == analytics.AttentionOverdue {
			marker = styles.Error("!")
		}
		line := fmt.Sprintf("%s  %s — %s", marker, uc.Name, att)
		if reason != "" {
			line += styles.Muted(" (" + reason + ")")
		}
		b.WriteString(line + "\n")
	}
	if attention == 0 {
		b.WriteString(styles.OK("all clear") + "\n")
	}
	b.WriteString("\n" + styles.Header("Overdue") + "\n")
	if len(overdue) == 0 {
		b.WriteString(styles.OK("none") + "\n")
	}
	for _, t := range overdue {
		b.WriteString(fmt.Sprintf("%s  %s  %s\n", styles.Error("!"), t.Title, dueText(t.DueAt)))
	}
	b.WriteString("\n" + styles.Header("Upcoming (14 days)") + "\n")
	if len(upcoming) == 0 {
		b.WriteString(styles.Muted("none") + "\n")
	}
	for i, t := range upcoming {
		if i >= 8 {
			b.WriteString(styles.Muted(fmt.Sprintf("…and %d more", len(upcoming)-i)) + "\n")
			break
		}
		b.WriteString(fmt.Sprintf("  %s  %s\n", t.Title, dueText(t.DueAt)))
	}
	d.content = b.String()
	d.viewport.SetContent(d.content)
	return nil
}

func dueText(due *time.Time) string {
	if due == nil {
		return "-"
	}
	return due.Local().Format("2006-01-02 15:04")
}

// Update scrolls the viewport.
func (d *DashboardModel) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	return cmd
}

// View renders the dashboard.
func (d *DashboardModel) View(width, height int) string {
	d.SetSize(width, height)
	return d.viewport.View()
}

// Help lists dashboard keys.
func (d *DashboardModel) Help() string { return "↑↓ scroll" }
