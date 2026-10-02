package services

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/filesystem"
)

func scanTopic(row *sql.Row) (domain.Topic, error) {
	var t domain.Topic
	var status, priority, created, updated string
	err := row.Scan(&t.UnitID, &t.ID, &t.Name, &t.Description, &status, &priority, &t.Notes, &created, &updated)
	if err != nil {
		return domain.Topic{}, err
	}
	if t.ReadingStatus, err = domain.ParseReadingStatus(status); err != nil {
		return domain.Topic{}, err
	}
	if t.Priority, err = domain.ParsePriority(priority); err != nil {
		return domain.Topic{}, err
	}
	if t.CreatedAt, err = domain.ParseTime(created); err != nil {
		return domain.Topic{}, err
	}
	if t.UpdatedAt, err = domain.ParseTime(updated); err != nil {
		return domain.Topic{}, err
	}
	return t, nil
}

const topicColumns = `unit_id, id, name, description, reading_status, priority, notes, created_at, updated_at`

// GetTopic returns one topic.
func GetTopic(db *sql.DB, unitID, topicID string) (domain.Topic, error) {
	t, err := scanTopic(db.QueryRow(`SELECT `+topicColumns+` FROM topics WHERE unit_id = ? AND id = ?`, unitID, topicID))
	if err == sql.ErrNoRows {
		return domain.Topic{}, fmt.Errorf("topic %q not found in unit %q", topicID, unitID)
	}
	return t, err
}

// ListTopics returns a unit's topics ordered by name, optionally filtered.
func ListTopics(db *sql.DB, unitID, statusFilter string) ([]domain.Topic, error) {
	query := `SELECT ` + topicColumns + ` FROM topics WHERE unit_id = ?`
	args := []any{unitID}
	if statusFilter != "" {
		if _, err := domain.ParseReadingStatus(statusFilter); err != nil {
			return nil, fmt.Errorf("services: %w", err)
		}
		query += ` AND reading_status = ?`
		args = append(args, statusFilter)
	}
	query += ` ORDER BY name`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Topic{}
	for rows.Next() {
		var t domain.Topic
		var status, priority, created, updated string
		if err := rows.Scan(&t.UnitID, &t.ID, &t.Name, &t.Description, &status, &priority, &t.Notes, &created, &updated); err != nil {
			return nil, err
		}
		var err error
		if t.ReadingStatus, err = domain.ParseReadingStatus(status); err != nil {
			return nil, err
		}
		if t.Priority, err = domain.ParsePriority(priority); err != nil {
			return nil, err
		}
		if t.CreatedAt, err = domain.ParseTime(created); err != nil {
			return nil, err
		}
		if t.UpdatedAt, err = domain.ParseTime(updated); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func topicNameTaken(db *sql.DB, unitID, name, exceptID string) (bool, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM topics WHERE unit_id = ? AND name = ? AND id <> ?`, unitID, name, exceptID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// CreateTopic validates, inserts, then creates the topic folders + sidecar.
// Topics cannot be added to archived units (unarchive first).
func CreateTopic(db *sql.DB, root string, logger *slog.Logger, unitID, name, priority string) (domain.Topic, error) {
	logger = orDiscard(logger)
	u, err := GetUnit(db, unitID)
	if err != nil {
		return domain.Topic{}, err
	}
	if u.Status == domain.UnitArchived {
		return domain.Topic{}, fmt.Errorf("services: unit %q is archived; unarchive it before adding topics", unitID)
	}
	if priority == "" {
		priority = string(domain.PriorityNormal)
	}
	id, err := nextPrefixedID(db, `SELECT id FROM topics WHERE unit_id = ?`, "topic", unitID)
	if err != nil {
		return domain.Topic{}, fmt.Errorf("services: allocate topic id: %w", err)
	}
	now := domain.Now()
	t := domain.Topic{UnitID: unitID, ID: id, Name: strings.TrimSpace(name),
		ReadingStatus: domain.ReadingUnread, CreatedAt: now, UpdatedAt: now}
	if t.Priority, err = domain.ParsePriority(priority); err != nil {
		return domain.Topic{}, fmt.Errorf("services: %w", err)
	}
	if err := t.Validate(); err != nil {
		return domain.Topic{}, fmt.Errorf("services: %w", err)
	}
	if taken, err := topicNameTaken(db, unitID, t.Name, ""); err != nil {
		return domain.Topic{}, err
	} else if taken {
		return domain.Topic{}, fmt.Errorf("services: topic %q already exists in unit %q", t.Name, unitID)
	}
	_, err = db.Exec(`INSERT INTO topics(unit_id, id, name, description, reading_status, priority, notes, created_at, updated_at)
		VALUES (?, ?, ?, '', ?, ?, '', ?, ?)`,
		t.UnitID, t.ID, t.Name, string(t.ReadingStatus), string(t.Priority),
		domain.FormatTime(now), domain.FormatTime(now))
	if err != nil {
		return domain.Topic{}, fmt.Errorf("services: create topic: %w", err)
	}
	if ferr := refreshTopicSidecar(root, t); ferr != nil {
		return t, ferr
	}
	logger.Info("topic created", "unit", unitID, "id", t.ID, "name", t.Name)
	return t, nil
}

// refreshTopicSidecar ensures the topic folders and rewrites topic.json.
func refreshTopicSidecar(root string, t domain.Topic) error {
	if err := filesystem.EnsureTopicSkeleton(root, t.UnitID, t.ID); err != nil {
		return fmt.Errorf("services: topic folders for %s/%s: %w", t.UnitID, t.ID, err)
	}
	path, err := filesystem.TopicSidecar(root, t.UnitID, t.ID)
	if err != nil {
		return err
	}
	if err := filesystem.WriteJSON(path, t); err != nil {
		return fmt.Errorf("services: topic sidecar for %s/%s: %w", t.UnitID, t.ID, err)
	}
	return nil
}

// touchTopic updates mutable topic fields and refreshes the sidecar.
func touchTopic(db *sql.DB, root string, t domain.Topic) (domain.Topic, error) {
	t.UpdatedAt = domain.Now()
	if err := t.Validate(); err != nil {
		return domain.Topic{}, fmt.Errorf("services: %w", err)
	}
	if _, err := db.Exec(`UPDATE topics SET name = ?, description = ?, reading_status = ?, priority = ?, notes = ?, updated_at = ?
		WHERE unit_id = ? AND id = ?`,
		t.Name, t.Description, string(t.ReadingStatus), string(t.Priority), t.Notes,
		domain.FormatTime(t.UpdatedAt), t.UnitID, t.ID); err != nil {
		return domain.Topic{}, fmt.Errorf("services: update topic: %w", err)
	}
	if ferr := refreshTopicSidecar(root, t); ferr != nil {
		return t, ferr
	}
	return t, nil
}

// RenameTopic changes a topic's name within its unit.
func RenameTopic(db *sql.DB, root string, logger *slog.Logger, unitID, topicID, newName string) (domain.Topic, error) {
	logger = orDiscard(logger)
	t, err := GetTopic(db, unitID, topicID)
	if err != nil {
		return domain.Topic{}, err
	}
	t.Name = strings.TrimSpace(newName)
	if taken, err := topicNameTaken(db, unitID, t.Name, topicID); err != nil {
		return domain.Topic{}, err
	} else if taken {
		return domain.Topic{}, fmt.Errorf("services: topic %q already exists in unit %q", t.Name, unitID)
	}
	t, err = touchTopic(db, root, t)
	if err != nil {
		return domain.Topic{}, err
	}
	logger.Info("topic renamed", "unit", unitID, "id", topicID, "name", t.Name)
	return t, nil
}

// SetTopicStatus performs a manual reading-status transition. Opening folders
// or files must never call this (see docs/01 FR-T3).
func SetTopicStatus(db *sql.DB, root string, logger *slog.Logger, unitID, topicID, status string) (domain.Topic, error) {
	logger = orDiscard(logger)
	t, err := GetTopic(db, unitID, topicID)
	if err != nil {
		return domain.Topic{}, err
	}
	next, err := domain.ParseReadingStatus(status)
	if err != nil {
		return domain.Topic{}, fmt.Errorf("services: %w", err)
	}
	if !domain.CanTransitionReading(t.ReadingStatus, next) {
		return domain.Topic{}, fmt.Errorf("services: cannot transition topic from %s to %s", t.ReadingStatus, next)
	}
	t.ReadingStatus = next
	t, err = touchTopic(db, root, t)
	if err != nil {
		return domain.Topic{}, err
	}
	logger.Info("topic status changed", "unit", unitID, "id", topicID, "status", next)
	return t, nil
}

// MoveTopic moves a topic to another active unit, keeping its local ID when
// free there and allocating a fresh one on collision. Folders and file links
// follow; the sidecar is rewritten at the new location.
func MoveTopic(db *sql.DB, root string, logger *slog.Logger, unitID, topicID, toUnitID string) (domain.Topic, error) {
	logger = orDiscard(logger)
	t, err := GetTopic(db, unitID, topicID)
	if err != nil {
		return domain.Topic{}, err
	}
	if unitID == toUnitID {
		return t, nil
	}
	dest, err := GetUnit(db, toUnitID)
	if err != nil {
		return domain.Topic{}, err
	}
	if dest.Status == domain.UnitArchived {
		return domain.Topic{}, fmt.Errorf("services: unit %q is archived", toUnitID)
	}
	newID := t.ID
	if _, err := GetTopic(db, toUnitID, t.ID); err == nil {
		newID, err = nextPrefixedID(db, `SELECT id FROM topics WHERE unit_id = ?`, "topic", toUnitID)
		if err != nil {
			return domain.Topic{}, fmt.Errorf("services: allocate topic id: %w", err)
		}
	}
	oldDir, err := filesystem.TopicDir(root, unitID, topicID)
	if err != nil {
		return domain.Topic{}, err
	}
	newDir, err := filesystem.TopicDir(root, toUnitID, newID)
	if err != nil {
		return domain.Topic{}, err
	}
	tx, err := db.Begin()
	if err != nil {
		return domain.Topic{}, fmt.Errorf("services: move topic: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE topics SET unit_id = ?, id = ?, updated_at = ? WHERE unit_id = ? AND id = ?`,
		toUnitID, newID, domain.FormatTime(domain.Now()), unitID, topicID); err != nil {
		return domain.Topic{}, fmt.Errorf("services: move topic: %w", err)
	}
	if _, err := tx.Exec(`UPDATE files SET unit_id = ?, topic_id = ? WHERE unit_id = ? AND topic_id = ?`,
		toUnitID, newID, unitID, topicID); err != nil {
		return domain.Topic{}, fmt.Errorf("services: move topic files: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Topic{}, fmt.Errorf("services: move topic: %w", err)
	}
	t.UnitID, t.ID = toUnitID, newID
	if filesystem.Exists(oldDir) {
		if merr := filesystem.MoveDir(oldDir, newDir); merr != nil {
			return t, fmt.Errorf("services: topic moved in database but folder move failed: %w", merr)
		}
	}
	if ferr := refreshTopicSidecar(root, t); ferr != nil {
		return t, ferr
	}
	logger.Info("topic moved", "from", unitID, "to", toUnitID, "id", newID)
	return t, nil
}

// topicChildCounts counts file links for the delete guard.
func topicChildCounts(db *sql.DB, unitID, topicID string) (domain.ChildCounts, error) {
	var c domain.ChildCounts
	if err := db.QueryRow(`SELECT COUNT(*) FROM files WHERE unit_id = ? AND topic_id = ?`, unitID, topicID).Scan(&c.Files); err != nil {
		return domain.ChildCounts{}, fmt.Errorf("services: count attachments: %w", err)
	}
	return c, nil
}

// DeleteTopic removes a topic and its folder; linked files need confirm=true.
func DeleteTopic(db *sql.DB, root string, logger *slog.Logger, unitID, topicID string, confirm bool) error {
	logger = orDiscard(logger)
	if _, err := GetTopic(db, unitID, topicID); err != nil {
		return err
	}
	counts, err := topicChildCounts(db, unitID, topicID)
	if err != nil {
		return err
	}
	if !counts.Empty() && !confirm {
		return &domain.ConfirmRequiredError{Entity: fmt.Sprintf("topic %s/%s", unitID, topicID), Counts: counts}
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("services: delete topic: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM topics WHERE unit_id = ? AND id = ?`, unitID, topicID); err != nil {
		return fmt.Errorf("services: delete topic: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM files WHERE unit_id = ? AND topic_id = ?`, unitID, topicID); err != nil {
		return fmt.Errorf("services: delete topic files: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("services: delete topic: %w", err)
	}
	dir, err := filesystem.TopicDir(root, unitID, topicID)
	if err != nil {
		return err
	}
	if err := filesystem.RemoveDir(dir); err != nil {
		return fmt.Errorf("services: topic deleted from database but folder removal failed: %w", err)
	}
	logger.Info("topic deleted", "unit", unitID, "id", topicID)
	return nil
}
