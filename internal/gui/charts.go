package gui

import (
	"zlanpiko/internal/analytics"
)

// AnalyticsDTO powers the Analytics view (docs/cs/07): the shared summary
// and unit cards plus task-status counts and the 14-day due histogram. Every
// number is Go-computed; the frontend draws SVG shapes only.
type AnalyticsDTO struct {
	Summary   SummaryDTO             `json:"summary"`
	Units     []UnitCardDTO          `json:"units"`
	Statuses  analytics.StatusCounts `json:"statuses"`
	Histogram []analytics.HistDay    `json:"histogram"`
}

// GetAnalytics returns the full analytics model in one call.
func (g *GuiApi) GetAnalytics() (AnalyticsDTO, error) {
	now := g.now()
	ucs, err := analytics.UnitCoverages(g.ctx.DB, false, now)
	if err != nil {
		return AnalyticsDTO{}, fail(ErrDB, "read unit coverages", err.Error())
	}
	sum, err := analytics.Summarize(g.ctx.DB, ucs, now)
	if err != nil {
		return AnalyticsDTO{}, fail(ErrDB, "summarize", err.Error())
	}
	cards, err := g.GetUnitCards()
	if err != nil {
		return AnalyticsDTO{}, err
	}
	counts, err := analytics.CountByStatus(g.ctx.DB, now)
	if err != nil {
		return AnalyticsDTO{}, fail(ErrDB, "count statuses", err.Error())
	}
	hist, err := analytics.DueHistogram(g.ctx.DB, now, 14)
	if err != nil {
		return AnalyticsDTO{}, fail(ErrDB, "build histogram", err.Error())
	}
	return AnalyticsDTO{
		Summary: SummaryDTO{
			Units:          sum.Units,
			TotalTopics:    sum.TotalTopics,
			Read:           sum.Read,
			Pending:        sum.Pending,
			Unread:         sum.Unread,
			Coverage:       sum.Coverage,
			CoverageText:   sum.CoverageText(),
			HasTopics:      sum.HasTopics,
			ActiveTasks:    sum.ActiveTasks,
			CompletedTasks: sum.CompletedTasks,
			OverdueTasks:   sum.OverdueTasks,
		},
		Units:     cards,
		Statuses:  counts,
		Histogram: hist,
	}, nil
}
