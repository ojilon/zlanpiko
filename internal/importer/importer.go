// Package importer implements bulk file import with a preview-first flow:
// Preview plans every copy, skip and collision; Execute carries the plan out
// per file with a journal, continuing past individual failures (see docs/11).
package importer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zlanpiko/internal/domain"
	"zlanpiko/internal/filesystem"
)

// DefaultMaxBytes is the default per-file size cap (override via Options).
const DefaultMaxBytes = 100 * 1024 * 1024

// skippedExt are never imported (reported, not copied).
var skippedExt = map[string]bool{
	".exe": true, ".msi": true, ".bat": true, ".cmd": true, ".com": true,
	".scr": true, ".ps1": true, ".dll": true,
}

// CollisionPolicy decides destination-name conflicts.
type CollisionPolicy int

const (
	// CollisionRename (default) copies as name-2.ext, name-3.ext, ...
	CollisionRename CollisionPolicy = iota
	// CollisionSkip leaves the destination untouched.
	CollisionSkip
	// CollisionOverwrite replaces the destination (explicit consent only).
	CollisionOverwrite
)

// ParseCollisionPolicy parses skip|rename|overwrite.
func ParseCollisionPolicy(s string) (CollisionPolicy, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "rename":
		return CollisionRename, nil
	case "skip":
		return CollisionSkip, nil
	case "overwrite":
		return CollisionOverwrite, nil
	default:
		return CollisionRename, fmt.Errorf("invalid collision policy %q (want skip|rename|overwrite)", s)
	}
}

// Options tunes an import.
type Options struct {
	// Dest is the destination: "" (inbox/<date>), a unit id, unit/topic or
	// unit/task reference, or a plain root-relative directory.
	Dest string
	// Recursive descends into source directories (non-recursive by default).
	Recursive bool
	// IncludeLarge imports files over MaxBytes.
	IncludeLarge bool
	// MaxBytes caps per-file size (0 = DefaultMaxBytes).
	MaxBytes int64
	// Collision resolves name conflicts.
	Collision CollisionPolicy
}

// Planned is one source file's fate.
type Planned struct {
	Src     string `json:"src"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	DestRel string `json:"dest_rel"` // slash-separated, collision-resolved
	Action  string `json:"action"`   // copy | skip
	Reason  string `json:"reason"`   // why skipped, or collision note
	Renamed bool   `json:"renamed"`
}

// Plan is the full import plan: executing it changes nothing.
type Plan struct {
	Source  string    `json:"source"`
	DestDir string    `json:"dest_dir"`
	UnitID  string    `json:"unit_id,omitempty"`
	TopicID string    `json:"topic_id,omitempty"`
	TaskID  string    `json:"task_id,omitempty"`
	Plans   []Planned `json:"plans"`
}

// Copies counts actionable plans.
func (p *Plan) Copies() int {
	n := 0
	for _, pl := range p.Plans {
		if pl.Action == "copy" {
			n++
		}
	}
	return n
}

// destTarget resolves Options.Dest to an absolute dir plus record links.
func destTarget(db *sql.DB, root, dest string) (absDir, unitID, topicID, taskID string, err error) {
	if dest == "" {
		dest = "inbox/" + time.Now().Format("2006-01-02")
	}
	parts := strings.Split(filepath.ToSlash(dest), "/")
	resolveDir := func(rel string) (string, error) {
		abs, err := filesystem.Join(root, filepath.FromSlash(rel))
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	if len(parts) == 1 {
		var id string
		uerr := db.QueryRow(`SELECT id FROM units WHERE id = ?`, parts[0]).Scan(&id)
		if uerr == nil {
			abs, err := filesystem.UnitDir(root, id)
			return abs, id, "", "", err
		}
		// Top-level plain areas are addressable bare; anything else must be
		// a unit id (typos become errors, not surprise directories).
		if parts[0] == "inbox" || parts[0] == "exports" {
			abs, err := resolveDir(parts[0])
			return abs, "", "", "", err
		}
		return "", "", "", "", fmt.Errorf("importer: unknown destination %q (want unit id, unit/topic|task, or directory)", dest)
	}
	if len(parts) == 2 {
		second := parts[1]
		switch {
		case strings.HasPrefix(second, "topic-"):
			var id string
			if err := db.QueryRow(`SELECT id FROM topics WHERE unit_id = ? AND id = ?`, parts[0], second).Scan(&id); err != nil {
				return "", "", "", "", fmt.Errorf("importer: unknown destination %q", dest)
			}
			abs, err := filesystem.TopicDir(root, parts[0], second)
			return abs, parts[0], second, "", err
		case strings.HasPrefix(second, "task-"):
			var id string
			if err := db.QueryRow(`SELECT id FROM tasks WHERE unit_id = ? AND id = ?`, parts[0], second).Scan(&id); err != nil {
				return "", "", "", "", fmt.Errorf("importer: unknown destination %q", dest)
			}
			var kind string
			if err := db.QueryRow(`SELECT kind FROM tasks WHERE unit_id = ? AND id = ?`, parts[0], second).Scan(&kind); err != nil {
				return "", "", "", "", fmt.Errorf("importer: unknown destination %q", dest)
			}
			k, err := domain.ParseTaskKind(kind)
			if err != nil {
				return "", "", "", "", err
			}
			abs, err := filesystem.TaskDir(root, parts[0], k, second)
			return abs, parts[0], "", second, err
		}
	}
	abs, err := resolveDir(dest)
	if err != nil {
		return "", "", "", "", fmt.Errorf("importer: bad destination: %w", err)
	}
	return abs, "", "", "", nil
}

// Preview scans srcAbs and plans the import without copying anything.
func Preview(db *sql.DB, root, srcAbs string, opts Options) (*Plan, error) {
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	absDir, unitID, topicID, taskID, err := destTarget(db, root, opts.Dest)
	if err != nil {
		return nil, err
	}
	st, err := os.Lstat(srcAbs)
	if err != nil {
		return nil, fmt.Errorf("importer: read source: %w", err)
	}
	var sources []string
	if !st.IsDir() {
		if st.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("importer: source is a link, refusing")
		}
		sources = []string{srcAbs}
	} else {
		var walk func(dir string) error
		walk = func(dir string) error {
			ents, err := os.ReadDir(dir)
			if err != nil {
				return err
			}
			for _, e := range ents {
				full := filepath.Join(dir, e.Name())
				if e.IsDir() {
					if opts.Recursive {
						if err := walk(full); err != nil {
							return err
						}
					}
					continue
				}
				sources = append(sources, full)
			}
			return nil
		}
		if err := walk(srcAbs); err != nil {
			return nil, fmt.Errorf("importer: scan source dir: %w", err)
		}
	}
	occupied := map[string]bool{}
	if rows, err := db.Query(`SELECT rel_path FROM files`); err == nil {
		for rows.Next() {
			var rel string
			if serr := rows.Scan(&rel); serr == nil {
				occupied[filepath.ToSlash(rel)] = true
			}
		}
		rows.Close()
	}
	pv := &Plan{Source: srcAbs, DestDir: slashRel(root, absDir), UnitID: unitID, TopicID: topicID, TaskID: taskID, Plans: []Planned{}}
	for _, src := range sources {
		pv.Plans = append(pv.Plans, planOne(src, root, absDir, occupied, opts, maxBytes))
	}
	return pv, nil
}

// planOne decides one source file's fate, reserving its destination name.
func planOne(src, root, absDir string, occupied map[string]bool, opts Options, maxBytes int64) Planned {
	name := filepath.Base(src)
	pl := Planned{Src: src, Name: name}
	st, err := os.Lstat(src)
	if err != nil {
		pl.Action, pl.Reason = "skip", "unreadable: "+err.Error()
		return pl
	}
	if st.Mode()&os.ModeSymlink != 0 {
		pl.Action, pl.Reason = "skip", "link (not followed)"
		return pl
	}
	if strings.HasPrefix(name, ".") {
		pl.Action, pl.Reason = "skip", "hidden file"
		return pl
	}
	if skippedExt[strings.ToLower(filepath.Ext(name))] {
		pl.Action, pl.Reason = "skip", "executable type"
		return pl
	}
	pl.Size = st.Size()
	if !opts.IncludeLarge && st.Size() > maxBytes {
		pl.Action, pl.Reason = "skip", fmt.Sprintf("over size cap (%d bytes)", maxBytes)
		return pl
	}
	destRel := filepath.ToSlash(filepath.Join(slashRel(root, absDir), name))
	if occupied[destRel] || fileExists(filepath.Join(absDir, name)) {
		switch opts.Collision {
		case CollisionSkip:
			pl.Action, pl.Reason = "skip", "name taken at destination"
			pl.DestRel = destRel
			return pl
		case CollisionOverwrite:
			pl.Action, pl.Reason = "copy", "will overwrite confirmed destination"
			pl.DestRel = destRel
			occupied[destRel] = true
			return pl
		default: // rename
			alt := freeName(absDir, slashRel(root, absDir), name, occupied)
			pl.Action, pl.Reason, pl.DestRel, pl.Renamed = "copy", "renamed (name taken)", alt, true
			occupied[alt] = true
			return pl
		}
	}
	pl.Action, pl.DestRel = "copy", destRel
	occupied[destRel] = true
	return pl
}

// freeName finds name-2.ext, name-3.ext, ... free on disk and in the plan.
func freeName(absDir, destSlash, name string, occupied map[string]bool) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, i, ext)
		rel := destSlash + "/" + candidate
		if occupied[rel] || fileExists(filepath.Join(absDir, candidate)) {
			continue
		}
		return rel
	}
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func slashRel(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// Report journals one Execute run.
type FileResult struct {
	Src     string `json:"src"`
	DestRel string `json:"dest_rel"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
	Renamed bool   `json:"renamed"`
	Error   string `json:"error,omitempty"`
}

// Report tallies Execute outcomes.
type Report struct {
	Copied  int          `json:"copied"`
	Skipped int          `json:"skipped"`
	Failed  int          `json:"failed"`
	Results []FileResult `json:"results"`
}

// Execute carries out a Preview: insert the file row, copy via temp+rename,
// then record the hash. Failures are journaled per file without aborting the
// run; ctx cancellation stops between files.
func Execute(ctx context.Context, db *sql.DB, root string, pv *Plan, logger *slog.Logger) (*Report, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	report := &Report{Results: []FileResult{}}
	for _, pl := range pv.Plans {
		if err := ctx.Err(); err != nil {
			return report, fmt.Errorf("importer: cancelled: %w", err)
		}
		if pl.Action != "copy" {
			report.Skipped++
			report.Results = append(report.Results, FileResult{Src: pl.Src, DestRel: pl.DestRel, Error: "skipped: " + pl.Reason})
			continue
		}
		res := FileResult{Src: pl.Src, DestRel: pl.DestRel, Renamed: pl.Renamed}
		if err := copyOne(db, root, pl, pv, &res); err != nil {
			res.Error = err.Error()
			report.Failed++
			logger.Warn("import failed", "src", pl.Src, "err", err)
		} else {
			report.Copied++
			logger.Info("imported", "src", pl.Src, "dest", pl.DestRel)
		}
		report.Results = append(report.Results, res)
	}
	return report, nil
}

// copyOne imports a single planned file.
func copyOne(db *sql.DB, root string, pl Planned, pv *Plan, res *FileResult) error {
	destAbs, err := filesystem.Join(root, filepath.FromSlash(pl.DestRel))
	if err != nil {
		return err
	}
	destDir := filepath.Dir(destAbs)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create destination dir: %w", err)
	}
	in, err := os.Open(pl.Src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()
	tmp, err := os.CreateTemp(destDir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("stage destination: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(tmp, hash), in)
	if err != nil {
		tmp.Close()
		return fmt.Errorf("copy: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close staged file: %w", err)
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	now := time.Now().UTC().Truncate(time.Second).Format(time.RFC3339)
	var unitID, topicID, taskID any
	if pv.UnitID != "" {
		unitID = pv.UnitID
	}
	if pv.TopicID != "" {
		topicID = pv.TopicID
	}
	if pv.TaskID != "" {
		taskID = pv.TaskID
	}
	if strings.Contains(pl.Reason, "overwrite") {
		if _, err := db.Exec(`DELETE FROM files WHERE rel_path = ?`, pl.DestRel); err != nil {
			return fmt.Errorf("clear overwritten row: %w", err)
		}
		if err := os.Remove(destAbs); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("clear overwritten file: %w", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO files(unit_id, topic_id, task_id, rel_path, size, sha256, missing, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		unitID, topicID, taskID, pl.DestRel, size, sum, now, now); err != nil {
		return fmt.Errorf("record file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	if err := os.Rename(tmpName, destAbs); err != nil {
		_, _ = db.Exec(`DELETE FROM files WHERE rel_path = ?`, pl.DestRel)
		return fmt.Errorf("publish: %w", err)
	}
	res.Size, res.SHA256 = size, sum
	return nil
}
