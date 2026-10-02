// Package maintenance implements the consistency verify/repair flow behind
// `maintenance verify [--repair]`: database integrity, filesystem/database
// divergence (missing/unindexed files), sidecar freshness, and leftover temp
// files. Verify is read-only; Repair fixes what verify reports (see docs/05,
// docs/06, docs/16).
package maintenance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zlanpiko/internal/database"
	"zlanpiko/internal/filesystem"
	"zlanpiko/internal/services"
	"zlanpiko/internal/tasks"
)

// SidecarIssue is one missing or stale sidecar.
type SidecarIssue struct {
	Kind    string `json:"kind"` // unit | topic | task
	UnitID  string `json:"unit_id"`
	ID      string `json:"id"`
	Problem string `json:"problem"` // missing | stale | unreadable
}

// Report is the verify outcome.
type Report struct {
	Integrity string         `json:"integrity"`
	Missing   []string       `json:"missing"`
	Unindexed []string       `json:"unindexed"`
	Sidecars  []SidecarIssue `json:"sidecars"`
	TempFiles []string       `json:"temp_files"`
	Repaired  bool           `json:"repaired"`
	CheckedAt time.Time      `json:"checked_at"`
}

// Clean reports whether verify found nothing to fix.
func (r *Report) Clean() bool {
	return len(r.Missing) == 0 && len(r.Unindexed) == 0 &&
		len(r.Sidecars) == 0 && len(r.TempFiles) == 0
}

// Summary renders a one-line human result.
func (r *Report) Summary() string {
	if r.Clean() {
		return "verify: ok"
	}
	return fmt.Sprintf("verify: %d missing file(s), %d unindexed, %d sidecar issue(s), %d temp file(s)",
		len(r.Missing), len(r.Unindexed), len(r.Sidecars), len(r.TempFiles))
}

// Verify runs every read-only check. It updates files.missing flags as a
// side effect (that is the flag's purpose); everything else is reported.
func Verify(db *sql.DB, root string, logger *slog.Logger) (*Report, error) {
	logger = orDiscard(logger)
	rep := &Report{Missing: []string{}, Unindexed: []string{},
		Sidecars: []SidecarIssue{}, TempFiles: []string{}, CheckedAt: time.Now()}
	if err := database.Verify(db); err != nil {
		return nil, err
	}
	rep.Integrity = "ok"
	scan, err := filesystem.Scan(root, db)
	if err != nil {
		return nil, err
	}
	rep.Missing, rep.Unindexed = scan.Missing, scan.Unindexed
	sidecars, err := checkSidecars(db, root)
	if err != nil {
		return nil, err
	}
	rep.Sidecars = sidecars
	temps, err := findTempFiles(root)
	if err != nil {
		return nil, err
	}
	rep.TempFiles = temps
	logger.Info("verify complete", "missing", len(rep.Missing), "unindexed", len(rep.Unindexed),
		"sidecars", len(rep.Sidecars), "temp", len(rep.TempFiles))
	return rep, nil
}

// Repair fixes sidecar issues (rewrite from DB rows) and removes temp files.
// Missing/unindexed files need user decisions and are never auto-fixed.
func Repair(db *sql.DB, root string, logger *slog.Logger, rep *Report) error {
	logger = orDiscard(logger)
	for _, issue := range rep.Sidecars {
		if err := repairSidecar(db, root, issue); err != nil {
			return err
		}
		logger.Info("sidecar repaired", "kind", issue.Kind, "unit", issue.UnitID, "id", issue.ID)
	}
	for _, tmp := range rep.TempFiles {
		abs, err := filesystem.Join(root, filepath.FromSlash(tmp))
		if err != nil {
			return err
		}
		if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("maintenance: remove temp %s: %w", tmp, err)
		}
	}
	rep.Repaired = true
	rep.Sidecars = []SidecarIssue{}
	rep.TempFiles = []string{}
	return nil
}

func orDiscard(l *slog.Logger) *slog.Logger {
	if l == nil {
		return slog.New(slog.DiscardHandler)
	}
	return l
}

// checkSidecars compares every record's sidecar with its row.
func checkSidecars(db *sql.DB, root string) ([]SidecarIssue, error) {
	var out []SidecarIssue
	units, err := services.ListUnits(db, true)
	if err != nil {
		return nil, err
	}
	for _, u := range units {
		path, err := filesystem.UnitSidecar(root, u.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, checkOne(path, "unit", u.ID, u.ID, u.UpdatedAt)...)
		topics, err := services.ListTopics(db, u.ID, "")
		if err != nil {
			return nil, err
		}
		for _, t := range topics {
			tpath, err := filesystem.TopicSidecar(root, u.ID, t.ID)
			if err != nil {
				return nil, err
			}
			out = append(out, checkOne(tpath, "topic", u.ID, t.ID, t.UpdatedAt)...)
		}
		tasks, err := tasks.ListTasks(db, tasks.Filter{UnitID: u.ID}, time.Now())
		if err != nil {
			return nil, err
		}
		for _, tk := range tasks {
			kpath, err := filesystem.TaskSidecar(root, u.ID, tk.Kind, tk.ID)
			if err != nil {
				return nil, err
			}
			out = append(out, checkOne(kpath, "task", u.ID, tk.ID, tk.UpdatedAt)...)
		}
	}
	if out == nil {
		out = []SidecarIssue{}
	}
	return out, nil
}

// checkOne compares a sidecar's updated_at with the row's.
func checkOne(path, kind, unitID, id string, rowUpdated time.Time) []SidecarIssue {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []SidecarIssue{{Kind: kind, UnitID: unitID, ID: id, Problem: "missing"}}
		}
		return []SidecarIssue{{Kind: kind, UnitID: unitID, ID: id, Problem: "unreadable"}}
	}
	var probe struct {
		UpdatedAt time.Time `json:"updated_at"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil || probe.UpdatedAt.IsZero() {
		return []SidecarIssue{{Kind: kind, UnitID: unitID, ID: id, Problem: "unreadable"}}
	}
	if !probe.UpdatedAt.Equal(rowUpdated.UTC().Truncate(time.Second)) {
		return []SidecarIssue{{Kind: kind, UnitID: unitID, ID: id, Problem: "stale"}}
	}
	return nil
}

// repairSidecar rewrites one sidecar from its DB row (ensuring folders).
func repairSidecar(db *sql.DB, root string, issue SidecarIssue) error {
	switch issue.Kind {
	case "unit":
		u, err := services.GetUnit(db, issue.UnitID)
		if err != nil {
			return err
		}
		if err := filesystem.EnsureUnitSkeleton(root, u.ID); err != nil {
			return err
		}
		path, err := filesystem.UnitSidecar(root, u.ID)
		if err != nil {
			return err
		}
		return filesystem.WriteJSON(path, u)
	case "topic":
		t, err := services.GetTopic(db, issue.UnitID, issue.ID)
		if err != nil {
			return err
		}
		if err := filesystem.EnsureTopicSkeleton(root, t.UnitID, t.ID); err != nil {
			return err
		}
		path, err := filesystem.TopicSidecar(root, t.UnitID, t.ID)
		if err != nil {
			return err
		}
		return filesystem.WriteJSON(path, t)
	case "task":
		tk, err := tasks.GetTask(db, issue.UnitID, issue.ID)
		if err != nil {
			return err
		}
		if err := filesystem.EnsureTaskSkeleton(root, tk.UnitID, tk.Kind, tk.ID); err != nil {
			return err
		}
		path, err := filesystem.TaskSidecar(root, tk.UnitID, tk.Kind, tk.ID)
		if err != nil {
			return err
		}
		return filesystem.WriteJSON(path, tk)
	default:
		return fmt.Errorf("maintenance: unknown sidecar kind %q", issue.Kind)
	}
}

// findTempFiles lists leftover .tmp-* staging files (never inside database/).
func findTempFiles(root string) ([]string, error) {
	var out []string
	for _, area := range []string{"units", "inbox", "exports", "backups", "config", "reports"} {
		base := filepath.Join(root, area)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasPrefix(d.Name(), ".tmp-") {
				rel, rerr := filepath.Rel(root, path)
				if rerr != nil {
					return rerr
				}
				out = append(out, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("maintenance: scan temp files: %w", err)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}
