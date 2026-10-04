package gui

import (
	"zlanpiko/internal/domain"
	"zlanpiko/internal/services"
	"zlanpiko/internal/tasks"
)

// TopicDTO is one topic row for lists and status changes.
type TopicDTO struct {
	UnitID   string               `json:"unit_id"`
	UnitName string               `json:"unit_name"`
	ID       string               `json:"id"`
	Name     string               `json:"name"`
	Status   domain.ReadingStatus `json:"status"`
	Priority domain.Priority      `json:"priority"`
}

// SetTopicStatus transitions a topic's reading status. Only explicit user
// action changes status (docs/04 FR-T3); anything may transition to
// anything, validated by domain.ParseReadingStatus.
func (g *GuiApi) SetTopicStatus(unitID, topicID, status string) (TopicDTO, error) {
	if _, err := domain.ParseReadingStatus(status); err != nil {
		return TopicDTO{}, fail(ErrValidation, err.Error(), "want unread|pending|read")
	}
	t, err := services.SetTopicStatus(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, topicID, status)
	if err != nil {
		return TopicDTO{}, fail(ErrNotFound, "no such topic "+unitID+"/"+topicID, err.Error())
	}
	names, err := g.unitNames()
	if err != nil {
		return TopicDTO{}, fail(ErrDB, "read unit names", err.Error())
	}
	return TopicDTO{
		UnitID:   t.UnitID,
		UnitName: names[t.UnitID],
		ID:       t.ID,
		Name:     t.Name,
		Status:   t.ReadingStatus,
		Priority: t.Priority,
	}, nil
}

// MoveDeadline reparses a deadline string and updates the task. Accepted
// formats are the CLI's (YYYY-MM-DD or YYYY-MM-DDTHH:MM, local); every
// affected view recomputes from the new value (docs/09).
func (g *GuiApi) MoveDeadline(unitID, taskID, due string) (TaskDetailDTO, error) {
	at, err := tasks.ParseDue(due)
	if err != nil {
		return TaskDetailDTO{}, fail(ErrValidation, err.Error(), "")
	}
	if _, err := tasks.UpdateTask(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, taskID, tasks.Update{
		SetDue: true,
		Due:    at,
	}); err != nil {
		return TaskDetailDTO{}, fail(ErrNotFound, "no such task "+unitID+"/"+taskID, err.Error())
	}
	return g.GetTaskDetail(unitID, taskID)
}

// SetTaskStatus changes a task's workflow status (overdue stays derived and
// can never be set). Completed/submitted timestamps are maintained by
// tasks.UpdateTask.
func (g *GuiApi) SetTaskStatus(unitID, taskID, status string) (TaskDetailDTO, error) {
	if _, err := domain.ParseTaskStatus(status); err != nil {
		return TaskDetailDTO{}, fail(ErrValidation, err.Error(), "want not_started|in_progress|completed|submitted")
	}
	if _, err := tasks.UpdateTask(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, taskID, tasks.Update{
		Status: &status,
	}); err != nil {
		return TaskDetailDTO{}, fail(ErrNotFound, "no such task "+unitID+"/"+taskID, err.Error())
	}
	return g.GetTaskDetail(unitID, taskID)
}
