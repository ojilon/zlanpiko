package gui

import (
	"zlanpiko/internal/tasks"
)

// TaskDetailDTO is the shared task drawer model (docs/cs/06): the timeline
// TaskDTO plus the long-form fields. Mutations arrive in Phase D; this view
// is read-only.
type TaskDetailDTO struct {
	Task             TaskDTO `json:"task"`
	Description      string  `json:"description"`
	Notes            string  `json:"notes"`
	CreatedDisplay   string  `json:"created_display"`
	CompletedDisplay string  `json:"completed_display"`
}

// GetTaskDetail returns one task for the drawer. Unknown IDs are NOT_FOUND
// with the unit-qualified hint the CLI prints.
func (g *GuiApi) GetTaskDetail(unitID, taskID string) (TaskDetailDTO, error) {
	now := g.now()
	t, err := tasks.GetTask(g.ctx.DB, unitID, taskID)
	if err != nil {
		return TaskDetailDTO{}, fail(ErrNotFound, "no such task "+unitID+"/"+taskID, "check the unit and task IDs")
	}
	names, err := g.unitNames()
	if err != nil {
		return TaskDetailDTO{}, fail(ErrDB, "read unit names", err.Error())
	}
	d := TaskDetailDTO{
		Task:           toTaskDTO(t, names[t.UnitID], now),
		Description:    t.Description,
		Notes:          t.Notes,
		CreatedDisplay: t.CreatedAt.Local().Format("2006-01-02 15:04"),
	}
	if t.CompletedAt != nil {
		d.CompletedDisplay = t.CompletedAt.Local().Format("2006-01-02 15:04")
	}
	return d, nil
}
