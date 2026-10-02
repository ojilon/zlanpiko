package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Info describes one file or directory (JSON-tagged for CLI output).
type Info struct {
	Name    string    `json:"name"`
	Rel     string    `json:"rel"` // slash-separated, root-relative
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// resolve maps a root-relative rel ("" = root) to an absolute path plus its
// slash-normalized rel form.
func resolve(root, rel string) (abs, slash string, err error) {
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." {
		rel = ""
	}
	if rel == "" {
		return filepath.Clean(root), "", nil
	}
	abs, err = Join(root, rel)
	if err != nil {
		return "", "", err
	}
	back, rerr := filepath.Rel(filepath.Clean(root), abs)
	if rerr != nil {
		return "", "", fmt.Errorf("filesystem: resolve %q: %w", rel, rerr)
	}
	return abs, filepath.ToSlash(back), nil
}

// protectedTop are first-level entries Delete/Move targets must never remove.
var protectedTop = map[string]bool{
	"database": true, "config": true, "backups": true, "exports": true,
	"reports": true, "units": true, "inbox": true,
}

// isProtected reports paths Delete must refuse: the root, first-level
// skeleton dirs, and unit record roots (deleted via services only).
func isProtected(slashRel string) bool {
	if slashRel == "" {
		return true
	}
	top := slashRel
	if i := strings.Index(top, "/"); i >= 0 {
		top = top[:i]
	}
	if protectedTop[top] {
		rest := strings.TrimPrefix(slashRel, top)
		rest = strings.TrimPrefix(rest, "/")
		// Refuse the skeleton dir itself and, under units/, the unit roots.
		if rest == "" {
			return true
		}
		if top == "units" && !strings.Contains(rest, "/") {
			return true
		}
	}
	return false
}

// WriteFile creates a new file with content, atomically. An existing file is
// a collision error, never an overwrite (see docs/16).
func WriteFile(root, rel string, content []byte) error {
	abs, _, err := resolve(root, rel)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(abs); err == nil {
		return fmt.Errorf("filesystem: file exists: %s", rel)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("filesystem: inspect %s: %w", rel, err)
	}
	if err := EnsureDir(filepath.Dir(abs)); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".tmp-*")
	if err != nil {
		return fmt.Errorf("filesystem: stage %s: %w", rel, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("filesystem: write %s: %w", rel, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("filesystem: close %s: %w", rel, err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("filesystem: chmod %s: %w", rel, err)
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return fmt.Errorf("filesystem: publish %s: %w", rel, err)
	}
	return nil
}

// Mkdir creates a directory (parents included; idempotent).
func Mkdir(root, rel string) error {
	abs, _, err := resolve(root, rel)
	if err != nil {
		return err
	}
	if err := EnsureDir(abs); err != nil {
		return err
	}
	var st os.FileInfo
	if st, err = os.Stat(abs); err != nil {
		return fmt.Errorf("filesystem: mkdir %s: %w", rel, err)
	}
	if !st.IsDir() {
		return fmt.Errorf("filesystem: %s exists and is not a directory", rel)
	}
	return nil
}

// Rename renames a file or directory within its parent. newName is a plain
// name (separators rejected); an occupied name is a collision error.
func Rename(root, rel, newName string) error {
	if newName == "" || strings.ContainsAny(newName, `/\`) {
		return fmt.Errorf("filesystem: invalid new name %q", newName)
	}
	if _, err := Join(root, newName); err != nil {
		return fmt.Errorf("filesystem: invalid new name %q: %w", newName, err)
	}
	abs, _, err := resolve(root, rel)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(abs); err != nil {
		return fmt.Errorf("filesystem: rename %s: %w", rel, err)
	}
	dst := filepath.Join(filepath.Dir(abs), newName)
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("filesystem: name taken: %s", newName)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("filesystem: inspect %s: %w", newName, err)
	}
	if err := os.Rename(abs, dst); err != nil {
		return fmt.Errorf("filesystem: rename %s: %w", rel, err)
	}
	return nil
}

// Move relocates a file or directory into an existing destination directory,
// keeping its basename. The destination must exist; name collisions fail.
func Move(root, srcRel, dstDirRel string) error {
	srcAbs, _, err := resolve(root, srcRel)
	if err != nil {
		return err
	}
	dstAbs, _, err := resolve(root, dstDirRel)
	if err != nil {
		return err
	}
	if _, err := os.Stat(srcAbs); err != nil {
		return fmt.Errorf("filesystem: move %s: %w", srcRel, err)
	}
	dstSt, err := os.Stat(dstAbs)
	if err != nil {
		return fmt.Errorf("filesystem: move into %s: %w", dstDirRel, err)
	}
	if !dstSt.IsDir() {
		return fmt.Errorf("filesystem: destination is not a directory: %s", dstDirRel)
	}
	final := filepath.Join(dstAbs, filepath.Base(srcAbs))
	if _, err := os.Lstat(final); err == nil {
		return fmt.Errorf("filesystem: destination name taken: %s", filepath.Base(srcAbs))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("filesystem: inspect destination: %w", err)
	}
	// Never relocate a protected subtree (skeleton dirs, unit record roots):
	// those move through the services/tasks paths. Moving *into* a
	// protected directory (e.g. inbox) is normal and allowed.
	srcSlash := slashOf(root, srcAbs)
	if isProtected(srcSlash) {
		return fmt.Errorf("filesystem: refusing to move protected path %q (use services for record folders)", srcRel)
	}
	if err := os.Rename(srcAbs, final); err != nil {
		return fmt.Errorf("filesystem: move %s: %w", srcRel, err)
	}
	return nil
}

// slashOf renders abs as a root-relative slash path ("" for root itself).
func slashOf(root, abs string) string {
	rel, err := filepath.Rel(filepath.Clean(root), abs)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

// Delete removes a file, or (with recursive=true) a directory tree.
// Protected paths (root, skeleton dirs, unit record roots) are refused;
// record subtrees must go through the services/tasks delete paths.
func Delete(root, rel string, recursive bool) error {
	abs, slash, err := resolve(root, rel)
	if err != nil {
		return err
	}
	if isProtected(slash) {
		return fmt.Errorf("filesystem: refusing to delete protected path %q", rel)
	}
	st, err := os.Lstat(abs)
	if err != nil {
		return fmt.Errorf("filesystem: delete %s: %w", rel, err)
	}
	if st.IsDir() && !recursive {
		return fmt.Errorf("filesystem: %s is a directory (use --recursive with explicit confirmation)", rel)
	}
	if err := os.RemoveAll(abs); err != nil {
		return fmt.Errorf("filesystem: delete %s: %w", rel, err)
	}
	return nil
}

// Stat returns metadata for one file or directory.
func Stat(root, rel string) (Info, error) {
	abs, slash, err := resolve(root, rel)
	if err != nil {
		return Info{}, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return Info{}, fmt.Errorf("filesystem: stat %s: %w", rel, err)
	}
	name := st.Name()
	if slash == "" {
		name = filepath.Base(filepath.Clean(root))
	}
	size := st.Size()
	if st.IsDir() {
		size = 0
	}
	return Info{Name: name, Rel: slash, IsDir: st.IsDir(), Size: size, ModTime: st.ModTime()}, nil
}

// List returns a directory's children sorted by name (case-insensitive).
func List(root, rel string) ([]Info, error) {
	abs, _, err := resolve(root, rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, fmt.Errorf("filesystem: list %s: %w", rel, err)
	}
	out := make([]Info, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return nil, fmt.Errorf("filesystem: list %s: %w", rel, err)
		}
		childRel := filepath.ToSlash(filepath.Join(slashOf(root, abs), e.Name()))
		size := info.Size()
		if info.IsDir() {
			size = 0
		}
		out = append(out, Info{Name: e.Name(), Rel: childRel, IsDir: info.IsDir(), Size: size, ModTime: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// Search matches filenames (case-insensitive substring) under units/ and
// inbox/, skipping app-managed areas. At most limit hits (limit <= 0 means 100).
func Search(root, query string, limit int) ([]Info, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("filesystem: empty search query")
	}
	if limit <= 0 {
		limit = 100
	}
	q := strings.ToLower(query)
	var out []Info
	for _, area := range []string{"units", "inbox"} {
		base := filepath.Join(root, area)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil || len(out) >= limit {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.Contains(strings.ToLower(d.Name()), q) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil // racing delete: skip, don't fail the search
			}
			out = append(out, Info{Name: d.Name(), Rel: slashOf(root, path),
				Size: info.Size(), ModTime: info.ModTime()})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("filesystem: search %s: %w", area, err)
		}
	}
	return out, nil
}

// Open launches a file with the OS default application (Windows). Only files
// inside the data root can be opened, and only files (not directories).
func Open(root, rel string) error {
	abs, _, err := resolve(root, rel)
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("filesystem: open %s: %w", rel, err)
	}
	if st.IsDir() {
		return fmt.Errorf("filesystem: cannot open directory %s (files only)", rel)
	}
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", abs)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("filesystem: open %s: %w", rel, err)
	}
	return nil
}
