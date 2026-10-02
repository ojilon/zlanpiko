package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/windows"
)

// folderPageSize caps the interactive folder listing (docs: first fifty).
const folderPageSize = 50

// Drive is one logical drive with its type and free space.
type Drive struct {
	Letter string // e.g. "D:"
	Kind   string // Local disk, Removable, Network, ...
	FreeGB float64
	Ready  bool
}

// ListDrives enumerates logical drives with type and free space.
// Unready drives (empty DVD etc.) are listed with Ready=false.
func ListDrives() ([]Drive, error) {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil, fmt.Errorf("installer: list drives: %w", err)
	}
	var out []Drive
	for i := uint(0); i < 26; i++ {
		if mask&(1<<i) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		d := Drive{Letter: strings.TrimSuffix(root, `\`)}
		d.Kind = driveKind(root)
		var free, total, totalFree uint64
		p, _ := windows.UTF16PtrFromString(root)
		if err := windows.GetDiskFreeSpaceEx(p, &free, &total, &totalFree); err == nil {
			d.Ready = true
			d.FreeGB = float64(free) / (1 << 30)
		}
		out = append(out, d)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("installer: no drives found")
	}
	return out, nil
}

func driveKind(root string) string {
	p, _ := windows.UTF16PtrFromString(root)
	switch windows.GetDriveType(p) {
	case windows.DRIVE_REMOVABLE:
		return "Removable"
	case windows.DRIVE_FIXED:
		return "Local disk"
	case windows.DRIVE_REMOTE:
		return "Network"
	case windows.DRIVE_CDROM:
		return "DVD"
	case windows.DRIVE_RAMDISK:
		return "RAM disk"
	default:
		return "Unknown"
	}
}

// ListRootDirs returns up to limit directory names directly under root
// (sorted, non-recursive) plus the total directory count, so the picker can
// say "showing 50 of 132".
func ListRootDirs(root string, limit int) (shown []string, total int, err error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, 0, fmt.Errorf("installer: list %s: %w", root, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "$") {
			continue // hidden/system clutter stays out of the picker
		}
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	total = len(names)
	if limit > 0 && len(names) > limit {
		names = names[:limit]
	}
	return names, total, nil
}

// ResolveTypedChoice validates a manually typed folder: either a name
// present in the drive root or a deeper relative subpath. Every level must
// exist (the installer creates only the final app folder, never parents).
// Absolute paths and ".." escapes are rejected.
func ResolveTypedChoice(driveRoot, input string) (string, error) {
	trimmed := strings.TrimSpace(strings.Trim(input, `"/`))
	if trimmed == "" {
		return "", fmt.Errorf("empty folder name")
	}
	rel := filepath.FromSlash(strings.ReplaceAll(trimmed, "/", `\`))
	if filepath.IsAbs(input) || filepath.IsAbs(rel) || strings.HasPrefix(rel, `\\`) {
		return "", fmt.Errorf("use a folder name or relative subpath, not an absolute path")
	}
	abs := filepath.Join(driveRoot, rel)
	if relToRoot, err := filepath.Rel(driveRoot, abs); err != nil ||
		relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes the drive: %q", input)
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("folder %q not found on %s", trimmed, driveRoot)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%q is not a folder", trimmed)
	}
	return abs, nil
}
