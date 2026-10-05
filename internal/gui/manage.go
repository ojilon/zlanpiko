package gui

import (
	"zlanpiko/internal/domain"
	"zlanpiko/internal/services"
	"zlanpiko/internal/tasks"
)

// UnitDetailDTO is the unit workspace: its card plus topics and tasks.
// It backs the Units view detail panel (docs/cs/04, docs/cs/05).
type UnitDetailDTO struct {
	Card   UnitCardDTO `json:"card"`
	Topics []TopicDTO  `json:"topics"`
	Tasks  []TaskDTO   `json:"tasks"`
}

// unitCardOf recomputes one unit's card (used after mutations so the UI
// updates without a second round-trip; docs/cs/01).
func (g *GuiApi) unitCardOf(unitID string) (UnitCardDTO, error) {
	cards, err := g.GetUnitCards()
	if err != nil {
		return UnitCardDTO{}, err
	}
	for _, c := range cards {
		if c.UnitID == unitID {
			return c, nil
		}
	}
	// Archived units are excluded from cards; report them plainly.
	u, err := services.GetUnit(g.ctx.DB, unitID)
	if err != nil {
		return UnitCardDTO{}, fail(ErrNotFound, "no such unit "+unitID, "")
	}
	return UnitCardDTO{UnitID: u.ID, Name: u.Name, Code: u.Code, Attention: "—", CoverageText: "—"}, nil
}

// GetUnitDetail returns a unit's card, topics and tasks.
func (g *GuiApi) GetUnitDetail(unitID string) (UnitDetailDTO, error) {
	now := g.now()
	card, err := g.unitCardOf(unitID)
	if err != nil {
		return UnitDetailDTO{}, err
	}
	topics, err := g.ListTopics(unitID, "")
	if err != nil {
		return UnitDetailDTO{}, err
	}
	list, err := tasks.ListTasks(g.ctx.DB, tasks.Filter{UnitID: unitID}, now)
	if err != nil {
		return UnitDetailDTO{}, fail(ErrDB, "list unit tasks", err.Error())
	}
	out := make([]TaskDTO, 0, len(list))
	for _, t := range list {
		out = append(out, toTaskDTO(t, card.Name, now))
	}
	return UnitDetailDTO{Card: card, Topics: topics, Tasks: out}, nil
}

// CreateUnit creates a course unit (name 1–120 chars; code optional).
func (g *GuiApi) CreateUnit(name, code string) (UnitCardDTO, error) {
	u, err := services.CreateUnit(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, name, code, "", "")
	if err != nil {
		return UnitCardDTO{}, fail(ErrValidation, "cannot create unit", err.Error())
	}
	return g.unitCardOf(u.ID)
}

// RenameUnit renames a unit. Folder names use stable IDs, so references
// never break on rename (docs/06).
func (g *GuiApi) RenameUnit(unitID, newName string) (UnitCardDTO, error) {
	u, err := services.RenameUnit(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, newName)
	if err != nil {
		return UnitCardDTO{}, fail(ErrValidation, "cannot rename unit "+unitID, err.Error())
	}
	return g.unitCardOf(u.ID)
}

// SetUnitArchived archives (or restores) a unit. Archiving preserves
// history; deletion is the destructive path (docs/07).
func (g *GuiApi) SetUnitArchived(unitID string, archived bool) (UnitCardDTO, error) {
	u, err := services.SetUnitArchived(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, archived)
	if err != nil {
		return UnitCardDTO{}, fail(ErrNotFound, "no such unit "+unitID, err.Error())
	}
	if archived {
		return UnitCardDTO{UnitID: u.ID, Name: u.Name, Code: u.Code, Attention: "—", CoverageText: "—"}, nil
	}
	return g.unitCardOf(u.ID)
}

// DeleteUnit deletes a unit after explicit confirmation. Associated records
// and files decide the outcome in services (safe deletion, docs/07).
func (g *GuiApi) DeleteUnit(unitID string, confirm bool) (string, error) {
	if !confirm {
		return "", fail(ErrValidation, "delete needs confirmation", "tick the confirm box — deletion cannot be undone")
	}
	if err := services.DeleteUnit(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, true); err != nil {
		return "", fail(ErrConflict, "cannot delete unit "+unitID, err.Error())
	}
	return "deleted " + unitID, nil
}

// ListTopics returns a unit's topics, optionally filtered by reading status
// ("" = all). Unknown units are NOT_FOUND, not empty.
func (g *GuiApi) ListTopics(unitID, statusFilter string) ([]TopicDTO, error) {
	if _, err := services.GetUnit(g.ctx.DB, unitID); err != nil {
		return nil, fail(ErrNotFound, "no such unit "+unitID, "")
	}
	if statusFilter != "" {
		if _, err := domain.ParseReadingStatus(statusFilter); err != nil {
			return nil, fail(ErrValidation, err.Error(), "want unread|pending|read")
		}
	}
	list, err := services.ListTopics(g.ctx.DB, unitID, statusFilter)
	if err != nil {
		return nil, fail(ErrDB, "list topics", err.Error())
	}
	names, err := g.unitNames()
	if err != nil {
		return nil, fail(ErrDB, "read unit names", err.Error())
	}
	out := make([]TopicDTO, 0, len(list))
	for _, t := range list {
		out = append(out, TopicDTO{
			UnitID:   t.UnitID,
			UnitName: names[t.UnitID],
			ID:       t.ID,
			Name:     t.Name,
			Status:   t.ReadingStatus,
			Priority: t.Priority,
		})
	}
	return out, nil
}

// CreateTopic adds a topic to an active unit.
func (g *GuiApi) CreateTopic(unitID, name, priority string) (TopicDTO, error) {
	if priority == "" {
		priority = string(domain.PriorityNormal)
	}
	t, err := services.CreateTopic(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, name, priority)
	if err != nil {
		return TopicDTO{}, fail(ErrValidation, "cannot create topic", err.Error())
	}
	names, _ := g.unitNames()
	return TopicDTO{
		UnitID:   t.UnitID,
		UnitName: names[t.UnitID],
		ID:       t.ID,
		Name:     t.Name,
		Status:   t.ReadingStatus,
		Priority: t.Priority,
	}, nil
}

// DeleteTopic removes a topic after explicit confirmation.
func (g *GuiApi) DeleteTopic(unitID, topicID string, confirm bool) (string, error) {
	if !confirm {
		return "", fail(ErrValidation, "delete needs confirmation", "tick the confirm box — deletion cannot be undone")
	}
	if err := services.DeleteTopic(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, topicID, true); err != nil {
		return "", fail(ErrConflict, "cannot delete topic "+unitID+"/"+topicID, err.Error())
	}
	return "deleted " + unitID + "/" + topicID, nil
}

// TaskFilterDTO filters the consolidated task list. Empty means all;
// status accepts a stored status or the derived "overdue".
type TaskFilterDTO struct {
	UnitID string `json:"unit_id"`
	Status string `json:"status"`
	Kind   string `json:"kind"`
}

// ListTasksFlat returns tasks across units (newest-due ordering comes from
// tasks.ListTasks; dateless last).
func (g *GuiApi) ListTasksFlat(f TaskFilterDTO) ([]TaskDTO, error) {
	if f.Status != "" {
		if f.Status == domain.TaskOverdue {
			// valid derived pseudo-status
		} else if _, err := domain.ParseTaskStatus(f.Status); err != nil {
			return nil, fail(ErrValidation, err.Error(), "want not_started|in_progress|completed|submitted|overdue")
		}
	}
	if f.Kind != "" {
		if _, err := domain.ParseTaskKind(f.Kind); err != nil {
			return nil, fail(ErrValidation, err.Error(), "")
		}
	}
	if f.UnitID != "" {
		if _, err := services.GetUnit(g.ctx.DB, f.UnitID); err != nil {
			return nil, fail(ErrNotFound, "no such unit "+f.UnitID, "")
		}
	}
	now := g.now()
	list, err := tasks.ListTasks(g.ctx.DB, tasks.Filter{UnitID: f.UnitID, Status: f.Status, Kind: f.Kind}, now)
	if err != nil {
		return nil, fail(ErrDB, "list tasks", err.Error())
	}
	names, err := g.unitNames()
	if err != nil {
		return nil, fail(ErrDB, "read unit names", err.Error())
	}
	out := make([]TaskDTO, 0, len(list))
	for _, t := range list {
		out = append(out, toTaskDTO(t, names[t.UnitID], now))
	}
	return out, nil
}

// CreateTaskInput creates an assignment etc. under an active unit. Due
// accepts the CLI formats ("" = dateless).
func (g *GuiApi) CreateTask(unitID, title, kind, due, priority string) (TaskDetailDTO, error) {
	if kind == "" {
		kind = string(domain.TaskAssignment)
	}
	if _, err := domain.ParseTaskKind(kind); err != nil {
		return TaskDetailDTO{}, fail(ErrValidation, err.Error(), "")
	}
	if priority == "" {
		priority = string(domain.PriorityNormal)
	}
	at, err := tasks.ParseDue(due)
	if err != nil {
		return TaskDetailDTO{}, fail(ErrValidation, err.Error(), "")
	}
	t, err := tasks.CreateTask(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, title, kind, at, priority, "")
	if err != nil {
		return TaskDetailDTO{}, fail(ErrValidation, "cannot create task", err.Error())
	}
	return g.GetTaskDetail(t.UnitID, t.ID)
}

// DeleteTask removes a task after explicit confirmation.
func (g *GuiApi) DeleteTask(unitID, taskID string, confirm bool) (string, error) {
	if !confirm {
		return "", fail(ErrValidation, "delete needs confirmation", "tick the confirm box — deletion cannot be undone")
	}
	if err := tasks.DeleteTask(g.ctx.DB, g.ctx.DataRoot, g.ctx.Logger, unitID, taskID, true); err != nil {
		return "", fail(ErrConflict, "cannot delete task "+unitID+"/"+taskID, err.Error())
	}
	return "deleted " + unitID + "/" + taskID, nil
}
