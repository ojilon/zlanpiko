// Package tasks implements the assignment/coursework/test/examination use
// cases: the only place that writes task records. Deadlines are task
// attributes; overdue is derived at read time, never stored (see docs/09).
package tasks

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/filesystem"
)

func orDiscard(l *slog.Logger) *slog.Logger {
	if l == nil {
		return slog.New(slog.DiscardHandler)
	}
	return l
}

// ParseDue parses a deadline: YYYY-MM-DD, YYYY-MM-DDTHH:MM or
// YYYY-MM-DD HH:MM (local time), or full RFC 3339. The space form exists
// because people type it and every other form rejects it with a confusing
// error (see GUI drawer). Empty input means no deadline (nil, nil).
func ParseDue(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	layouts := []string{"2006-01-02", "2006-01-02T15:04", "2006-01-02 15:04", time.RFC3339}
	var lastErr error
	for _, layout := range layouts {
		var t time.Time
		var err error
		if layout == time.RFC3339 {
			t, err = time.Parse(layout, s)
		} else {
			t, err = time.ParseInLocation(layout, s, time.Local)
		}
		if err == nil {
			utc := t.UTC().Truncate(time.Second)
			return &utc, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("invalid deadline %q (want YYYY-MM-DD, YYYY-MM-DDTHH:MM or YYYY-MM-DD HH:MM): %w", s, lastErr)
}

const taskColumns = `unit_id, id, title, kind, status, due_at, completed_at, priority, description, notes, created_at, updated_at`

func scanTask(row *sql.Row) (domain.Task, error) {
	var t domain.Task
	if err := scanInto(row, &t); err != nil {
		return domain.Task{}, err
	}
	return t, nil
}

// rowScanner covers *sql.Row and *sql.Rows for shared scan logic.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanInto(row rowScanner, t *domain.Task) error {
	var kind, status, priority, created, updated string
	var dueAt, completedAt sql.NullString
	err := row.Scan(&t.UnitID, &t.ID, &t.Title, &kind, &status, &dueAt, &completedAt, &priority, &t.Description, &t.Notes, &created, &updated)
	if err != nil {
		return err
	}
	if t.Kind, err = domain.ParseTaskKind(kind); err != nil {
		return err
	}
	if t.Status, err = domain.ParseTaskStatus(status); err != nil {
		return err
	}
	if t.Priority, err = domain.ParsePriority(priority); err != nil {
		return err
	}
	if dueAt.Valid {
		if t.DueAt, err = parseNullableTime(dueAt.String); err != nil {
			return err
		}
	}
	if completedAt.Valid {
		if t.CompletedAt, err = parseNullableTime(completedAt.String); err != nil {
			return err
		}
	}
	if t.CreatedAt, err = domain.ParseTime(created); err != nil {
		return err
	}
	if t.UpdatedAt, err = domain.ParseTime(updated); err != nil {
		return err
	}
	return nil
}

func parseNullableTime(s string) (*time.Time, error) {
	t, err := domain.ParseTime(s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func formatNullable(t *time.Time) any {
	if t == nil {
		return nil
	}
	return domain.FormatTime(*t)
}

// GetTask returns one task.
func GetTask(db *sql.DB, unitID, taskID string) (domain.Task, error) {
	t, err := scanTask(db.QueryRow(`SELECT `+taskColumns+` FROM tasks WHERE unit_id = ? AND id = ?`, unitID, taskID))
	if err == sql.ErrNoRows {
		return domain.Task{}, fmt.Errorf("task %q not found in unit %q", taskID, unitID)
	}
	return t, err
}

func nextTaskID(db *sql.DB, unitID string) (string, error) {
	rows, err := db.Query(`SELECT id FROM tasks WHERE unit_id = ?`, unitID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	max := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		if n, cerr := strconv.Atoi(strings.TrimPrefix(id, "task-")); cerr == nil && n > max {
			max = n
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("task-%03d", max+1), nil
}

// unitMustBeActive rejects writes against missing or archived units.
func unitMustBeActive(db *sql.DB, unitID string) error {
	var status string
	err := db.QueryRow(`SELECT status FROM units WHERE id = ?`, unitID).Scan(&status)
	if err == sql.ErrNoRows {
		return fmt.Errorf("unit %q not found", unitID)
	}
	if err != nil {
		return err
	}
	if status == string(domain.UnitArchived) {
		return fmt.Errorf("tasks: unit %q is archived; unarchive it before changing tasks", unitID)
	}
	return nil
}

// CreateTask validates, inserts, then creates the task folders + sidecar.
func CreateTask(db *sql.DB, root string, logger *slog.Logger, unitID, title, kind string, dueAt *time.Time, priority, description string) (domain.Task, error) {
	logger = orDiscard(logger)
	if err := unitMustBeActive(db, unitID); err != nil {
		return domain.Task{}, err
	}
	if priority == "" {
		priority = string(domain.PriorityNormal)
	}
	if kind == "" {
		kind = string(domain.TaskAssignment)
	}
	id, err := nextTaskID(db, unitID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("tasks: allocate task id: %w", err)
	}
	now := domain.Now()
	t := domain.Task{UnitID: unitID, ID: id, Title: strings.TrimSpace(title),
		Status: domain.TaskNotStarted, DueAt: dueAt,
		Description: description, CreatedAt: now, UpdatedAt: now}
	if t.Kind, err = domain.ParseTaskKind(kind); err != nil {
		return domain.Task{}, fmt.Errorf("tasks: %w", err)
	}
	if t.Priority, err = domain.ParsePriority(priority); err != nil {
		return domain.Task{}, fmt.Errorf("tasks: %w", err)
	}
	if err := t.Validate(); err != nil {
		return domain.Task{}, fmt.Errorf("tasks: %w", err)
	}
	_, err = db.Exec(`INSERT INTO tasks(unit_id, id, title, kind, status, due_at, completed_at, priority, description, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?, '', ?, ?)`,
		t.UnitID, t.ID, t.Title, string(t.Kind), string(t.Status), formatNullable(t.DueAt),
		string(t.Priority), t.Description, domain.FormatTime(now), domain.FormatTime(now))
	if err != nil {
		return domain.Task{}, fmt.Errorf("tasks: create task: %w", err)
	}
	if ferr := refreshTaskSidecar(root, t); ferr != nil {
		return t, ferr
	}
	logger.Info("task created", "unit", unitID, "id", t.ID, "title", t.Title)
	return t, nil
}

// refreshTaskSidecar ensures the task folders and rewrites task.json.
func refreshTaskSidecar(root string, t domain.Task) error {
	if err := filesystem.EnsureTaskSkeleton(root, t.UnitID, t.Kind, t.ID); err != nil {
		return fmt.Errorf("tasks: task folders for %s/%s: %w", t.UnitID, t.ID, err)
	}
	path, err := filesystem.TaskSidecar(root, t.UnitID, t.Kind, t.ID)
	if err != nil {
		return err
	}
	if err := filesystem.WriteJSON(path, t); err != nil {
		return fmt.Errorf("tasks: task sidecar for %s/%s: %w", t.UnitID, t.ID, err)
	}
	return nil
}

// Update carries optional task edits; nil means "leave unchanged".
// SetDue distinguishes "leave" (false) from "set or clear" (true + Due).
type Update struct {
	Title       *string
	Kind        *string
	Status      *string
	Priority    *string
	Description *string
	Notes       *string
	SetDue      bool
	Due         *time.Time
}

// UpdateTask applies edits, maintains completed_at, moves folders on kind
// change and refreshes the sidecar.
func UpdateTask(db *sql.DB, root string, logger *slog.Logger, unitID, taskID string, up Update) (domain.Task, error) {
	logger = orDiscard(logger)
	if err := unitMustBeActive(db, unitID); err != nil {
		return domain.Task{}, err
	}
	t, err := GetTask(db, unitID, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	oldKind := t.Kind
	if up.Title != nil {
		t.Title = strings.TrimSpace(*up.Title)
	}
	if up.Kind != nil {
		if t.Kind, err = domain.ParseTaskKind(*up.Kind); err != nil {
			return domain.Task{}, fmt.Errorf("tasks: %w", err)
		}
	}
	if up.Status != nil {
		if t.Status, err = domain.ParseTaskStatus(*up.Status); err != nil {
			return domain.Task{}, fmt.Errorf("tasks: %w", err)
		}
	}
	if up.Priority != nil {
		if t.Priority, err = domain.ParsePriority(*up.Priority); err != nil {
			return domain.Task{}, fmt.Errorf("tasks: %w", err)
		}
	}
	if up.Description != nil {
		t.Description = *up.Description
	}
	if up.Notes != nil {
		t.Notes = *up.Notes
	}
	if up.SetDue {
		t.DueAt = up.Due
	}
	if err := t.Validate(); err != nil {
		return domain.Task{}, fmt.Errorf("tasks: %w", err)
	}
	// completed_at follows stored status, never edited directly.
	if t.Status == domain.TaskCompleted || t.Status == domain.TaskSubmitted {
		if t.CompletedAt == nil {
			now := domain.Now()
			t.CompletedAt = &now
		}
	} else {
		t.CompletedAt = nil
	}
	t.UpdatedAt = domain.Now()
	if _, err := db.Exec(`UPDATE tasks SET title = ?, kind = ?, status = ?, due_at = ?, completed_at = ?,
		priority = ?, description = ?, notes = ?, updated_at = ? WHERE unit_id = ? AND id = ?`,
		t.Title, string(t.Kind), string(t.Status), formatNullable(t.DueAt), formatNullable(t.CompletedAt),
		string(t.Priority), t.Description, t.Notes, domain.FormatTime(t.UpdatedAt), unitID, taskID); err != nil {
		return domain.Task{}, fmt.Errorf("tasks: update task: %w", err)
	}
	if t.Kind != oldKind {
		oldDir, err := filesystem.TaskDir(root, unitID, oldKind, taskID)
		if err != nil {
			return domain.Task{}, err
		}
		newDir, err := filesystem.TaskDir(root, unitID, t.Kind, taskID)
		if err != nil {
			return domain.Task{}, err
		}
		if filesystem.Exists(oldDir) {
			if merr := filesystem.MoveDir(oldDir, newDir); merr != nil {
				return t, fmt.Errorf("tasks: task updated in database but folder move failed: %w", merr)
			}
		}
	}
	if ferr := refreshTaskSidecar(root, t); ferr != nil {
		return t, ferr
	}
	logger.Info("task updated", "unit", unitID, "id", taskID)
	return t, nil
}

// CompleteTask is shorthand for UpdateTask with a terminal stored status.
func CompleteTask(db *sql.DB, root string, logger *slog.Logger, unitID, taskID, status string) (domain.Task, error) {
	if status != string(domain.TaskCompleted) && status != string(domain.TaskSubmitted) {
		return domain.Task{}, fmt.Errorf("tasks: %q is not a completion status (want completed|submitted)", status)
	}
	return UpdateTask(db, root, logger, unitID, taskID, Update{Status: &status})
}

// taskFileCounts counts linked files for the delete guard.
func taskFileCounts(db *sql.DB, unitID, taskID string) (domain.ChildCounts, error) {
	var c domain.ChildCounts
	if err := db.QueryRow(`SELECT COUNT(*) FROM files WHERE unit_id = ? AND task_id = ?`, unitID, taskID).Scan(&c.Files); err != nil {
		return domain.ChildCounts{}, fmt.Errorf("tasks: count attachments: %w", err)
	}
	return c, nil
}

// DeleteTask removes a task, its file rows and its folder.
// Linked files require confirm=true.
func DeleteTask(db *sql.DB, root string, logger *slog.Logger, unitID, taskID string, confirm bool) error {
	logger = orDiscard(logger)
	t, err := GetTask(db, unitID, taskID)
	if err != nil {
		return err
	}
	counts, err := taskFileCounts(db, unitID, taskID)
	if err != nil {
		return err
	}
	if !counts.Empty() && !confirm {
		return &domain.ConfirmRequiredError{Entity: fmt.Sprintf("task %s/%s", unitID, taskID), Counts: counts}
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("tasks: delete task: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM tasks WHERE unit_id = ? AND id = ?`, unitID, taskID); err != nil {
		return fmt.Errorf("tasks: delete task: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM files WHERE unit_id = ? AND task_id = ?`, unitID, taskID); err != nil {
		return fmt.Errorf("tasks: delete task files: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tasks: delete task: %w", err)
	}
	dir, err := filesystem.TaskDir(root, unitID, t.Kind, taskID)
	if err != nil {
		return err
	}
	if err := filesystem.RemoveDir(dir); err != nil {
		return fmt.Errorf("tasks: task deleted from database but folder removal failed: %w", err)
	}
	logger.Info("task deleted", "unit", unitID, "id", taskID)
	return nil
}

// NextOpen returns the nearest open task of the given kinds due at or
// after now, or nil when there is none.
func NextOpen(db *sql.DB, unitID string, kinds []domain.TaskKind, now time.Time) (*domain.Task, error) {
	if len(kinds) == 0 {
		return nil, nil
	}
	placeholders := ""
	args := []any{unitID}
	for i, k := range kinds {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, string(k))
	}
	args = append(args, domain.FormatTime(now))
	row := db.QueryRow(`SELECT `+taskColumns+` FROM tasks
		WHERE unit_id = ? AND kind IN (`+placeholders+`)
		AND status IN ('not_started','in_progress') AND due_at IS NOT NULL AND due_at >= ?
		ORDER BY due_at LIMIT 1`, args...)
	t, err := scanTask(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// NextDeadline returns the nearest open due date in the unit (any kind) at
// or after now, or nil when there is none.
func NextDeadline(db *sql.DB, unitID string, now time.Time) (*time.Time, error) {
	var due sql.NullString
	err := db.QueryRow(`SELECT due_at FROM tasks
		WHERE unit_id = ? AND status IN ('not_started','in_progress')
		AND due_at IS NOT NULL AND due_at >= ? ORDER BY due_at LIMIT 1`,
		unitID, domain.FormatTime(now)).Scan(&due)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t, err := parseNullableTime(due.String)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// (derived) or "" for all. DueBefore/DueAfter bound due_at (nil = unbounded).
type Filter struct {
	UnitID    string
	Kind      string
	Status    string
	DueBefore *time.Time
	DueAfter  *time.Time
}

// ListTasks returns matching tasks ordered by due_at (dateless last).
// now anchors the derived overdue filter.
func ListTasks(db *sql.DB, f Filter, now time.Time) ([]domain.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks WHERE 1 = 1`
	var args []any
	if f.UnitID != "" {
		query += ` AND unit_id = ?`
		args = append(args, f.UnitID)
	}
	if f.Kind != "" {
		if _, err := domain.ParseTaskKind(f.Kind); err != nil {
			return nil, fmt.Errorf("tasks: %w", err)
		}
		query += ` AND kind = ?`
		args = append(args, f.Kind)
	}
	status := f.Status
	overdueOnly := status == domain.TaskOverdue
	if status != "" && !overdueOnly {
		if _, err := domain.ParseTaskStatus(status); err != nil {
			return nil, fmt.Errorf("tasks: %w", err)
		}
		query += ` AND status = ?`
		args = append(args, status)
	}
	if f.DueBefore != nil {
		query += ` AND due_at IS NOT NULL AND due_at < ?`
		args = append(args, domain.FormatTime(*f.DueBefore))
	}
	if f.DueAfter != nil {
		query += ` AND due_at IS NOT NULL AND due_at >= ?`
		args = append(args, domain.FormatTime(*f.DueAfter))
	}
	query += ` ORDER BY due_at IS NULL, due_at`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Task
	for rows.Next() {
		var t domain.Task
		if err := scanInto(rows, &t); err != nil {
			return nil, err
		}
		if overdueOnly && !t.Overdue(now) {
			continue
		}
		out = append(out, t)
	}
	if out == nil {
		out = []domain.Task{}
	}
	return out, rows.Err()
}
