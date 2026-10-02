package filesystem

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// scanAreas are the user-content subtrees covered by Scan.
var scanAreas = []string{"units", "inbox"}

// sidecarNames are app-managed JSON files, never reported as unindexed.
var sidecarNames = map[string]bool{
	"unit.json": true, "topic.json": true, "task.json": true, "user.json": true,
}

// ScanReport describes filesystem/database divergence.
type ScanReport struct {
	// Missing lists DB-tracked files absent on disk (flags now set in the DB).
	Missing []string `json:"missing"`
	// Unindexed lists on-disk files with no DB row (import candidates).
	Unindexed []string `json:"unindexed"`
}

// Scan walks the user-content areas, refreshes the files.missing flags and
// reports divergence. It writes only the missing flags — unindexed files are
// reported, never auto-linked (see docs/06).
func Scan(root string, db *sql.DB) (ScanReport, error) {
	report := ScanReport{Missing: []string{}, Unindexed: []string{}}
	onDisk := map[string]bool{}
	for _, area := range scanAreas {
		base := filepath.Join(root, area)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || sidecarNames[d.Name()] {
				return nil
			}
			onDisk[slashOf(root, path)] = true
			return nil
		})
		if err != nil {
			return report, fmt.Errorf("filesystem: scan %s: %w", area, err)
		}
	}
	rows, err := db.Query(`SELECT rel_path, missing FROM files`)
	if err != nil {
		return report, fmt.Errorf("filesystem: scan tracked files: %w", err)
	}
	type tracked struct {
		rel     string
		missing int
	}
	var all []tracked
	for rows.Next() {
		var tr tracked
		if err := rows.Scan(&tr.rel, &tr.missing); err != nil {
			rows.Close()
			return report, fmt.Errorf("filesystem: scan tracked files: %w", err)
		}
		all = append(all, tr)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return report, fmt.Errorf("filesystem: scan tracked files: %w", err)
	}
	rows.Close() // before any UPDATE: single-connection database
	trackedSet := map[string]bool{}
	for _, tr := range all {
		slash := filepath.ToSlash(tr.rel)
		trackedSet[slash] = true
		present := onDisk[slash]
		wantFlag := 0
		if !present {
			wantFlag = 1
			report.Missing = append(report.Missing, slash)
		}
		if wantFlag != tr.missing {
			if _, err := db.Exec(`UPDATE files SET missing = ? WHERE rel_path = ?`, wantFlag, tr.rel); err != nil {
				return report, fmt.Errorf("filesystem: flag %s: %w", tr.rel, err)
			}
		}
	}
	for rel := range onDisk {
		if !trackedSet[rel] {
			report.Unindexed = append(report.Unindexed, rel)
		}
	}
	return report, nil
}
