// Package exporter renders the academic status report from live database
// rows — never hand-maintained text. Text mode is a 72-column phone-friendly
// summary (see docs/11); JSON mode is the stable integration schema (v1:
// additive changes only).
package exporter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"zlanpiko/internal/analytics"
	"zlanpiko/internal/app"
	"zlanpiko/internal/domain"
	"zlanpiko/internal/services"
	"zlanpiko/internal/tasks"
)

// lineWidth is the phone-friendly wrap width (docs/11).
const lineWidth = 72

// attentionLimit caps the attention section (remainder counted).
const attentionLimit = 30

// TextReport renders the plain-text status report. unitID scopes to one unit
// ("" = overall). now is the generation time (local display).
func TextReport(db *sql.DB, unitID string, now time.Time) (string, error) {
	ucs, err := analytics.UnitCoverages(db, false, now)
	if err != nil {
		return "", err
	}
	if unitID != "" {
		filtered := ucs[:0:0]
		for _, uc := range ucs {
			if uc.UnitID == unitID {
				filtered = append(filtered, uc)
			}
		}
		if len(filtered) == 0 {
			return "", fmt.Errorf("exporter: unit %q not found", unitID)
		}
		ucs = filtered
	}
	sum, err := analytics.Summarize(db, ucs, now)
	if err != nil {
		return "", err
	}
	upcoming, err := analytics.Upcoming(db, now, 14)
	if err != nil {
		return "", err
	}
	if unitID != "" {
		upcoming = filterUnit(upcoming, unitID)
	}
	overdue, err := tasks.ListTasks(db, tasks.Filter{UnitID: unitID, Status: domain.TaskOverdue}, now)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("ACADEMIC STATUS REPORT\n")
	b.WriteString(fmt.Sprintf("Generated: %s (%s %s)\n", now.Format("2006-01-02 15:04"), app.Name, app.Version))
	b.WriteString("\nUNITS\n" + strings.Repeat("-", 28) + "\n")
	if len(ucs) == 0 {
		b.WriteString("None\n")
	}
	for _, uc := range ucs {
		att, reason, err := analytics.AssessUnit(db, uc, now)
		if err != nil {
			return "", err
		}
		name := uc.Name
		if uc.Code != "" {
			name += " [" + uc.Code + "]"
		}
		b.WriteString(wrap(fmt.Sprintf("%s — %d topics (R%d/P%d/U%d) %s · %d active · %s",
			name, uc.Total, uc.Read, uc.Pending, uc.Unread, uc.CoverageText(),
			uc.ActiveTasks, att)) + "\n")
		if reason != "" && att != analytics.AttentionOnTrack {
			b.WriteString(wrap("  ("+reason+")") + "\n")
		}
	}
	b.WriteString("\nTOPICS NEEDING ATTENTION\n" + strings.Repeat("-", 28) + "\n")
	attention, err := topicsNeedingAttention(db, ucs, now)
	if err != nil {
		return "", err
	}
	if len(attention) == 0 {
		b.WriteString("None\n")
	}
	for i, line := range attention {
		if i >= attentionLimit {
			b.WriteString(fmt.Sprintf("…and %d more\n", len(attention)-i))
			break
		}
		b.WriteString(wrap(fmt.Sprintf("%d. %s", i+1, line)) + "\n")
	}
	b.WriteString("\nUPCOMING DEADLINES (14 days)\n" + strings.Repeat("-", 28) + "\n")
	if len(upcoming) == 0 {
		b.WriteString("None\n")
	}
	for _, t := range upcoming {
		b.WriteString(wrap(fmt.Sprintf("%s - %s (%s)", dueDate(t.DueAt), t.Title, unitName(ucs, t.UnitID))) + "\n")
	}
	b.WriteString("\nOVERDUE ITEMS\n" + strings.Repeat("-", 28) + "\n")
	if len(overdue) == 0 {
		b.WriteString("None\n")
	}
	for _, t := range overdue {
		b.WriteString(wrap(fmt.Sprintf("%s - %s (%s)", dueDate(t.DueAt), t.Title, unitName(ucs, t.UnitID))) + "\n")
	}
	b.WriteString("\nGENERAL SUMMARY\n" + strings.Repeat("-", 28) + "\n")
	b.WriteString(fmt.Sprintf("Total units: %d\n", sum.Units))
	b.WriteString(fmt.Sprintf("Total topics: %d\nRead: %d\nPending: %d\nUnread: %d\nOverall reading coverage: %s\n",
		sum.TotalTopics, sum.Read, sum.Pending, sum.Unread, sum.CoverageText()))
	return b.String(), nil
}

func filterUnit(list []domain.Task, unitID string) []domain.Task {
	out := []domain.Task{}
	for _, t := range list {
		if t.UnitID == unitID {
			out = append(out, t)
		}
	}
	return out
}

func unitName(ucs []analytics.UnitCoverage, unitID string) string {
	for _, uc := range ucs {
		if uc.UnitID == unitID {
			return uc.Name
		}
	}
	return unitID
}

func dueDate(due *time.Time) string {
	if due == nil {
		return "-"
	}
	return due.Local().Format("2006-01-02")
}

// topicsNeedingAttention lists unread/pending topics in units with upcoming
// assessments, most urgent first (exam proximity, then status).
func topicsNeedingAttention(db *sql.DB, ucs []analytics.UnitCoverage, now time.Time) ([]string, error) {
	type candidate struct {
		line string
		urg  int // lower = more urgent
	}
	var out []candidate
	for _, uc := range ucs {
		exam, err := analytics.NextAssessment(db, uc.UnitID, now)
		if err != nil {
			return nil, err
		}
		if exam == nil {
			continue
		}
		topics, err := services.ListTopics(db, uc.UnitID, "")
		if err != nil {
			return nil, err
		}
		for _, t := range topics {
			att, _ := analytics.AssessTopic(t.ReadingStatus, t.Priority, exam.DueAt, now)
			if att == analytics.AttentionOnTrack || att == analytics.AttentionCompleted {
				continue
			}
			urg := 2
			if att == analytics.AttentionAtRisk || att == analytics.AttentionOverdue {
				urg = 0
			} else if att == analytics.AttentionReview {
				urg = 3
			} else {
				urg = 1
			}
			out = append(out, candidate{
				line: fmt.Sprintf("%s (%s) - %s, exam %s", t.Name, uc.Name, att, dueDate(exam.DueAt)),
				urg:  urg,
			})
		}
	}
	// Stable order: urgency, then line text.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && (out[j].urg < out[j-1].urg ||
			(out[j].urg == out[j-1].urg && out[j].line < out[j-1].line)); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	lines := make([]string, 0, len(out))
	for _, c := range out {
		lines = append(lines, c.line)
	}
	return lines, nil
}

// wrap folds text to lineWidth columns on word boundaries.
func wrap(s string) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	col := 0
	for i, w := range words {
		if i > 0 && col+1+len(w) > lineWidth {
			b.WriteString("\n")
			col = 0
		} else if i > 0 {
			b.WriteString(" ")
			col++
		}
		b.WriteString(w)
		col += len(w)
	}
	return b.String()
}

// JSON schema v1 (additive changes only).

// JSONExport renders the stable integration document.
func JSONExport(db *sql.DB, unitID string, now time.Time) ([]byte, error) {
	ucs, err := analytics.UnitCoverages(db, false, now)
	if err != nil {
		return nil, err
	}
	if unitID != "" {
		filtered := ucs[:0:0]
		for _, uc := range ucs {
			if uc.UnitID == unitID {
				filtered = append(filtered, uc)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("exporter: unit %q not found", unitID)
		}
		ucs = filtered
	}
	type unitRow struct {
		analytics.UnitCoverage
		Attention string `json:"attention"`
	}
	units := make([]unitRow, 0, len(ucs))
	var topics []domain.Topic
	var allTasks []domain.Task
	for _, uc := range ucs {
		att, _, err := analytics.AssessUnit(db, uc, now)
		if err != nil {
			return nil, err
		}
		units = append(units, unitRow{UnitCoverage: uc, Attention: string(att)})
		ts, err := services.ListTopics(db, uc.UnitID, "")
		if err != nil {
			return nil, err
		}
		topics = append(topics, ts...)
		tl, err := tasks.ListTasks(db, tasks.Filter{UnitID: uc.UnitID}, now)
		if err != nil {
			return nil, err
		}
		allTasks = append(allTasks, tl...)
	}
	if topics == nil {
		topics = []domain.Topic{}
	}
	if allTasks == nil {
		allTasks = []domain.Task{}
	}
	sum, err := analytics.Summarize(db, ucs, now)
	if err != nil {
		return nil, err
	}
	doc := map[string]any{
		"app":          map[string]any{"name": app.Name, "version": app.Version},
		"generated_at": now.UTC().Format(time.RFC3339),
		"units":        units,
		"topics":       topics,
		"tasks":        allTasks,
		"summary":      sum,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
