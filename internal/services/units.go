// Package services implements the unit and topic use cases: the only place
// that writes those records. Fronts (TUI, CLI) call these functions; they
// never touch the database or filesystem directly (see docs/02).
package services

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/filesystem"
)

// orDiscard tolerates a nil logger (tests may pass nil).
func orDiscard(l *slog.Logger) *slog.Logger {
	if l == nil {
		return slog.New(slog.DiscardHandler)
	}
	return l
}

// isUniqueErr reports SQLite unique-constraint violations.
func isUniqueErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// nextPrefixedID allocates prefix-NNN (never reused: max suffix + 1).
func nextPrefixedID(db *sql.DB, query string, prefix string, args ...any) (string, error) {
	rows, err := db.Query(query, args...)
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
		if n, cerr := strconv.Atoi(strings.TrimPrefix(id, prefix+"-")); cerr == nil && n > max {
			max = n
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%03d", prefix, max+1), nil
}

func scanUnit(row *sql.Row) (domain.Unit, error) {
	var u domain.Unit
	var code, colour sql.NullString
	var status, created, updated string
	err := row.Scan(&u.ID, &u.Name, &code, &u.Description, &colour, &status, &created, &updated)
	if err != nil {
		return domain.Unit{}, err
	}
	u.Code, u.Colour = code.String, colour.String
	st, err := domain.ParseUnitStatus(status)
	if err != nil {
		return domain.Unit{}, err
	}
	u.Status = st
	if u.CreatedAt, err = domain.ParseTime(created); err != nil {
		return domain.Unit{}, err
	}
	if u.UpdatedAt, err = domain.ParseTime(updated); err != nil {
		return domain.Unit{}, err
	}
	return u, nil
}

// GetUnit returns one unit by ID.
func GetUnit(db *sql.DB, id string) (domain.Unit, error) {
	u, err := scanUnit(db.QueryRow(`SELECT id, name, code, description, colour, status, created_at, updated_at FROM units WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return domain.Unit{}, fmt.Errorf("unit %q not found", id)
	}
	return u, err
}

// ListUnits returns units ordered by name.
func ListUnits(db *sql.DB, includeArchived bool) ([]domain.Unit, error) {
	query := `SELECT id, name, code, description, colour, status, created_at, updated_at FROM units`
	if !includeArchived {
		query += ` WHERE status = 'active'`
	}
	query += ` ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Unit{}
	for rows.Next() {
		var u domain.Unit
		var code, colour sql.NullString
		var status, created, updated string
		if err := rows.Scan(&u.ID, &u.Name, &code, &u.Description, &colour, &status, &created, &updated); err != nil {
			return nil, err
		}
		u.Code, u.Colour = code.String, colour.String
		st, err := domain.ParseUnitStatus(status)
		if err != nil {
			return nil, err
		}
		u.Status = st
		if u.CreatedAt, err = domain.ParseTime(created); err != nil {
			return nil, err
		}
		if u.UpdatedAt, err = domain.ParseTime(updated); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UnitSummary is a unit plus attachment counts for list views.
type UnitSummary struct {
	domain.Unit
	Topics int `json:"topics"`
	Tasks  int `json:"tasks"`
}

// ListUnitSummaries returns units with topic/task counts.
func ListUnitSummaries(db *sql.DB, includeArchived bool) ([]UnitSummary, error) {
	units, err := ListUnits(db, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]UnitSummary, 0, len(units))
	for _, u := range units {
		var topics, tasks int
		if err := db.QueryRow(`SELECT COUNT(*) FROM topics WHERE unit_id = ?`, u.ID).Scan(&topics); err != nil {
			return nil, err
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE unit_id = ?`, u.ID).Scan(&tasks); err != nil {
			return nil, err
		}
		out = append(out, UnitSummary{Unit: u, Topics: topics, Tasks: tasks})
	}
	return out, nil
}

// CreateUnit validates, inserts, then creates the unit folder skeleton and
// sidecar. Directory failures are returned alongside the created unit so the
// caller sees the partial state (DB row first, per docs/16).
func CreateUnit(db *sql.DB, root string, logger *slog.Logger, name, code, colour, description string) (domain.Unit, error) {
	logger = orDiscard(logger)
	id, err := nextPrefixedID(db, `SELECT id FROM units`, "unit")
	if err != nil {
		return domain.Unit{}, fmt.Errorf("services: allocate unit id: %w", err)
	}
	now := domain.Now()
	u := domain.Unit{ID: id, Name: strings.TrimSpace(name), Code: strings.TrimSpace(code),
		Description: description, Colour: strings.TrimSpace(colour),
		Status: domain.UnitActive, CreatedAt: now, UpdatedAt: now}
	if err := u.Validate(); err != nil {
		return domain.Unit{}, fmt.Errorf("services: %w", err)
	}
	_, err = db.Exec(`INSERT INTO units(id, name, code, description, colour, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Name, nullIfEmpty(u.Code), u.Description, nullIfEmpty(u.Colour),
		string(u.Status), domain.FormatTime(now), domain.FormatTime(now))
	if err != nil {
		if isUniqueErr(err) {
			return domain.Unit{}, fmt.Errorf("services: unit name %q already exists", u.Name)
		}
		return domain.Unit{}, fmt.Errorf("services: create unit: %w", err)
	}
	if ferr := refreshUnitSidecar(db, root, u); ferr != nil {
		return u, ferr
	}
	logger.Info("unit created", "id", u.ID, "name", u.Name)
	return u, nil
}

// refreshUnitSidecar ensures the unit folders and rewrites unit.json.
func refreshUnitSidecar(db *sql.DB, root string, u domain.Unit) error {
	_ = db // rows are already committed; sidecar mirrors the passed unit
	if err := filesystem.EnsureUnitSkeleton(root, u.ID); err != nil {
		return fmt.Errorf("services: unit folders for %s: %w", u.ID, err)
	}
	path, err := filesystem.UnitSidecar(root, u.ID)
	if err != nil {
		return err
	}
	if err := filesystem.WriteJSON(path, u); err != nil {
		return fmt.Errorf("services: unit sidecar for %s: %w", u.ID, err)
	}
	return nil
}

// RenameUnit changes a unit's name (folders use stable IDs, so nothing moves).
func RenameUnit(db *sql.DB, root string, logger *slog.Logger, id, newName string) (domain.Unit, error) {
	logger = orDiscard(logger)
	u, err := GetUnit(db, id)
	if err != nil {
		return domain.Unit{}, err
	}
	u.Name = strings.TrimSpace(newName)
	u.UpdatedAt = domain.Now()
	if err := u.Validate(); err != nil {
		return domain.Unit{}, fmt.Errorf("services: %w", err)
	}
	res, err := db.Exec(`UPDATE units SET name = ?, updated_at = ? WHERE id = ? AND status = 'active'`,
		u.Name, domain.FormatTime(u.UpdatedAt), id)
	if err != nil {
		if isUniqueErr(err) {
			return domain.Unit{}, fmt.Errorf("services: unit name %q already exists", u.Name)
		}
		return domain.Unit{}, fmt.Errorf("services: rename unit: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return domain.Unit{}, fmt.Errorf("services: rename unit: %w", err)
	}
	if n == 0 {
		if isUnique, uerr := unitNameTaken(db, u.Name, id); uerr == nil && isUnique {
			return domain.Unit{}, fmt.Errorf("services: unit name %q already exists", u.Name)
		}
		return domain.Unit{}, fmt.Errorf("unit %q not found or is archived", id)
	}
	if ferr := refreshUnitSidecar(db, root, u); ferr != nil {
		return u, ferr
	}
	logger.Info("unit renamed", "id", id, "name", u.Name)
	return u, nil
}

func unitNameTaken(db *sql.DB, name, exceptID string) (bool, error) {
	var id string
	err := db.QueryRow(`SELECT id FROM units WHERE name = ? AND status = 'active' AND id <> ?`, name, exceptID).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// SetUnitArchived archives (true) or unarchives (false) a unit.
func SetUnitArchived(db *sql.DB, root string, logger *slog.Logger, id string, archived bool) (domain.Unit, error) {
	logger = orDiscard(logger)
	u, err := GetUnit(db, id)
	if err != nil {
		return domain.Unit{}, err
	}
	if !archived && u.Status == domain.UnitActive {
		return u, nil
	}
	if archived && u.Status == domain.UnitArchived {
		return u, nil
	}
	if !archived {
		if taken, err := unitNameTaken(db, u.Name, id); err != nil {
			return domain.Unit{}, err
		} else if taken {
			return domain.Unit{}, fmt.Errorf("services: cannot unarchive: active unit named %q already exists", u.Name)
		}
		u.Status = domain.UnitActive
	} else {
		u.Status = domain.UnitArchived
	}
	u.UpdatedAt = domain.Now()
	if _, err := db.Exec(`UPDATE units SET status = ?, updated_at = ? WHERE id = ?`,
		string(u.Status), domain.FormatTime(u.UpdatedAt), id); err != nil {
		return domain.Unit{}, fmt.Errorf("services: archive unit: %w", err)
	}
	if ferr := refreshUnitSidecar(db, root, u); ferr != nil {
		return u, ferr
	}
	logger.Info("unit archive changed", "id", id, "archived", archived)
	return u, nil
}

// unitChildCounts tallies attached records for the delete guard.
func unitChildCounts(db *sql.DB, id string) (domain.ChildCounts, error) {
	var c domain.ChildCounts
	for _, q := range []struct {
		query string
		dest  *int
	}{
		{`SELECT COUNT(*) FROM topics WHERE unit_id = ?`, &c.Topics},
		{`SELECT COUNT(*) FROM tasks WHERE unit_id = ?`, &c.Tasks},
		{`SELECT COUNT(*) FROM files WHERE unit_id = ?`, &c.Files},
	} {
		if err := db.QueryRow(q.query, id).Scan(q.dest); err != nil {
			return domain.ChildCounts{}, fmt.Errorf("services: count attachments: %w", err)
		}
	}
	return c, nil
}

// DeleteUnit removes a unit, its records (cascaded) and its folder.
// Records with attachments require confirm=true; archiving is suggested
// by the confirm error message (see docs/01 FR-U4).
func DeleteUnit(db *sql.DB, root string, logger *slog.Logger, id string, confirm bool) error {
	logger = orDiscard(logger)
	if _, err := GetUnit(db, id); err != nil {
		return err
	}
	counts, err := unitChildCounts(db, id)
	if err != nil {
		return err
	}
	if !counts.Empty() && !confirm {
		return &domain.ConfirmRequiredError{Entity: "unit " + id, Counts: counts}
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("services: delete unit: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM units WHERE id = ?`, id); err != nil {
		return fmt.Errorf("services: delete unit: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("services: delete unit: %w", err)
	}
	dir, err := filesystem.UnitDir(root, id)
	if err != nil {
		return err
	}
	// Missing folders are fine (RemoveAll is a no-op); a removal failure
	// after the DB commit is reported, never hidden.
	if err := filesystem.RemoveDir(dir); err != nil {
		return fmt.Errorf("services: unit %s deleted from database but folder removal failed: %w", id, err)
	}
	logger.Info("unit deleted", "id", id)
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
