// Package filesystem owns every file operation under the data root.
// Callers pass root-relative intents; this package resolves, validates and
// executes them. Phase 3 covers record folders and sidecars only; moves,
// renames, imports and deletes of user content arrive in Phase 4
// (see docs/06, docs/16).
package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"zlanpiko/internal/domain"
)

// reservedNames are Windows device names that must never become path segments.
var reservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// Join resolves rel segments under root, rejecting anything that escapes it
// (absolute segments, "..", drive letters, UNC) or uses reserved names.
// Stored rel_paths use "/" separators; conversion happens here, once.
func Join(root string, rel ...string) (string, error) {
	if len(rel) == 0 {
		return "", fmt.Errorf("filesystem: no path segments")
	}
	root = filepath.Clean(root)
	for _, seg := range rel {
		if seg == "" {
			return "", fmt.Errorf("filesystem: empty path segment")
		}
		if filepath.IsAbs(seg) || strings.Contains(seg, ":") ||
			strings.HasPrefix(seg, `\\`) {
			return "", fmt.Errorf("filesystem: forbidden segment %q", seg)
		}
		stem := strings.ToUpper(strings.SplitN(filepath.Base(seg), ".", 2)[0])
		if reservedNames[stem] {
			return "", fmt.Errorf("filesystem: reserved name %q", seg)
		}
	}
	joined := filepath.Join(append([]string{root}, rel...)...)
	relToRoot, err := filepath.Rel(root, joined)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("filesystem: path escapes data root: %q", filepath.Join(rel...))
	}
	return joined, nil
}

// Exists reports whether path exists (follows symlinks).
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir creates a directory and parents (no-op when present).
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("filesystem: create dir %s: %w", path, err)
	}
	return nil
}

// WriteJSON atomically writes v as indented JSON (temp file + rename, so a
// crash never leaves a half-written sidecar).
func WriteJSON(path string, v any) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("filesystem: encode %s: %w", path, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return fmt.Errorf("filesystem: stage %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename
	if _, err := tmp.Write(append(raw, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("filesystem: write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("filesystem: close %s: %w", path, err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("filesystem: chmod %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("filesystem: publish %s: %w", path, err)
	}
	return nil
}

// RemoveDir removes a record directory tree. The caller must already hold an
// explicit confirmation; missing directories are not an error.
func RemoveDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("filesystem: remove %s: %w", path, err)
	}
	return nil
}

// MoveDir renames src to dst, creating dst's parent. An existing dst is a
// collision error, never an overwrite.
func MoveDir(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("filesystem: destination exists: %s", dst)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("filesystem: inspect destination %s: %w", dst, err)
	}
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("filesystem: move %s: %w", src, err)
	}
	return nil
}

// Record folder layout (see docs/06). Folder names are stable IDs.

// UnitDir is units/<unitID>/.
func UnitDir(root, unitID string) (string, error) {
	return Join(root, "units", unitID)
}

// TopicDir is units/<unitID>/topics/<topicID>/.
func TopicDir(root, unitID, topicID string) (string, error) {
	return Join(root, "units", unitID, "topics", topicID)
}

// KindGroup maps a task kind to its folder group (docs/06).
func KindGroup(kind domain.TaskKind) (string, error) {
	switch kind {
	case domain.TaskAssignment:
		return "assignments", nil
	case domain.TaskCoursework:
		return "coursework", nil
	case domain.TaskTest:
		return "tests", nil
	case domain.TaskExamination:
		return "examinations", nil
	case domain.TaskProject:
		return "projects", nil
	case domain.TaskOther:
		return "other", nil
	default:
		return "", fmt.Errorf("filesystem: unknown task kind %q", kind)
	}
}

// TaskDir is units/<unitID>/<group>/<taskID>/.
func TaskDir(root, unitID string, kind domain.TaskKind, taskID string) (string, error) {
	group, err := KindGroup(kind)
	if err != nil {
		return "", err
	}
	return Join(root, "units", unitID, group, taskID)
}

// EnsureUnitSkeleton creates a unit's folders: topics/ and notes/.
func EnsureUnitSkeleton(root, unitID string) error {
	base, err := UnitDir(root, unitID)
	if err != nil {
		return err
	}
	for _, dir := range []string{base, filepath.Join(base, "topics"), filepath.Join(base, "notes")} {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// EnsureTopicSkeleton creates reading/, research/ and summaries/.
func EnsureTopicSkeleton(root, unitID, topicID string) error {
	base, err := TopicDir(root, unitID, topicID)
	if err != nil {
		return err
	}
	for _, dir := range []string{base,
		filepath.Join(base, "reading"),
		filepath.Join(base, "research"),
		filepath.Join(base, "summaries")} {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// EnsureTaskSkeleton creates materials/, drafts/ and submissions/.
func EnsureTaskSkeleton(root, unitID string, kind domain.TaskKind, taskID string) error {
	base, err := TaskDir(root, unitID, kind, taskID)
	if err != nil {
		return err
	}
	for _, dir := range []string{base,
		filepath.Join(base, "materials"),
		filepath.Join(base, "drafts"),
		filepath.Join(base, "submissions")} {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}
	return nil
}

// Sidecar paths (mirrors of DB rows, see docs/04).

// UnitSidecar is units/<unitID>/unit.json.
func UnitSidecar(root, unitID string) (string, error) {
	base, err := UnitDir(root, unitID)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "unit.json"), nil
}

// TopicSidecar is units/<unitID>/topics/<topicID>/topic.json.
func TopicSidecar(root, unitID, topicID string) (string, error) {
	base, err := TopicDir(root, unitID, topicID)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "topic.json"), nil
}

// TaskSidecar is units/<unitID>/<group>/<taskID>/task.json.
func TaskSidecar(root, unitID string, kind domain.TaskKind, taskID string) (string, error) {
	base, err := TaskDir(root, unitID, kind, taskID)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "task.json"), nil
}
