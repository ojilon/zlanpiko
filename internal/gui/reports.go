package gui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"zlanpiko/internal/app"
	"zlanpiko/internal/backup"
	"zlanpiko/internal/config"
	"zlanpiko/internal/exporter"
)

// ReportDTO is an academic status report: human-readable text for the phone
// plus structured JSON for integrations (docs/11, docs/cs/05 export parity).
type ReportDTO struct {
	Text string `json:"text"`
	JSON string `json:"json"`
}

// GetReport renders the current database records (unitID "" = overall).
func (g *GuiApi) GetReport(unitID string) (ReportDTO, error) {
	now := g.now()
	text, err := exporter.TextReport(g.ctx.DB, unitID, now)
	if err != nil {
		return ReportDTO{}, fail(ErrDB, "render report", err.Error())
	}
	raw, err := exporter.JSONExport(g.ctx.DB, unitID, now)
	if err != nil {
		return ReportDTO{}, fail(ErrDB, "render report JSON", err.Error())
	}
	return ReportDTO{Text: text, JSON: string(raw)}, nil
}

// BackupDTO is one backup file with its manifest facts.
type BackupDTO struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Created  string `json:"created"`
	Files    int    `json:"files"`
	Size     int64  `json:"size"`
	SizeText string `json:"size_text"`
	Readable bool   `json:"readable"`
}

func toBackupDTO(root string, b backup.BackupInfo) BackupDTO {
	name := b.Path
	if i := strings.LastIndex(name, "\\"); i >= 0 {
		name = name[i+1:]
	}
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return BackupDTO{
		Path:     b.Path,
		Name:     name,
		Kind:     b.Kind,
		Created:  b.Created.Local().Format("2006-01-02 15:04"),
		Files:    b.Files,
		Size:     b.Size,
		SizeText: humanBytes(b.Size, false),
		Readable: b.Readable,
	}
}

// ListBackups enumerates backups/*.zip newest first.
func (g *GuiApi) ListBackups() ([]BackupDTO, error) {
	list, err := backup.List(g.ctx.DataRoot)
	if err != nil {
		return nil, fail(ErrBackup, "list backups", err.Error())
	}
	out := make([]BackupDTO, 0, len(list))
	for _, b := range list {
		out = append(out, toBackupDTO(g.ctx.DataRoot, b))
	}
	return out, nil
}

// CreateBackup writes a timestamped zip (full includes academic files).
// The destination name is generated up front so the response describes the
// exact file written.
func (g *GuiApi) CreateBackup(full bool) (BackupDTO, error) {
	kind := "db"
	if full {
		kind = "full"
	}
	out := backup.DefaultName(g.ctx.DataRoot, kind, time.Now())
	if _, err := backup.Create(g.ctx.DB, g.ctx.DataRoot, out, full, g.ctx.Logger); err != nil {
		return BackupDTO{}, fail(ErrBackup, "create backup", err.Error())
	}
	st, err := os.Stat(out)
	if err != nil {
		return BackupDTO{}, fail(ErrBackup, "stat new backup", err.Error())
	}
	m, err := backup.Verify(out)
	if err != nil {
		return BackupDTO{}, fail(ErrBackup, "verify new backup", err.Error())
	}
	return BackupDTO{
		Path:     out,
		Name:     filepath.Base(out),
		Kind:     kind,
		Created:  m.CreatedAt.Local().Format("2006-01-02 15:04"),
		Files:    len(m.Files),
		Size:     st.Size(),
		SizeText: humanBytes(st.Size(), false),
		Readable: true,
	}, nil
}

// VerifyBackup re-checks every hash in a backup file.
func (g *GuiApi) VerifyBackup(path string) (BackupDTO, error) {
	m, err := backup.Verify(path)
	if err != nil {
		return BackupDTO{}, fail(ErrBackup, "verify "+path, err.Error())
	}
	list, err := backup.List(g.ctx.DataRoot)
	if err == nil {
		for _, b := range list {
			if b.Path == path {
				return toBackupDTO(g.ctx.DataRoot, b), nil
			}
		}
	}
	return BackupDTO{
		Path:     path,
		Name:     filepath.Base(path),
		Kind:     m.Kind,
		Created:  m.CreatedAt.Local().Format("2006-01-02 15:04"),
		SizeText: humanBytes(m.DBSize, false),
		Readable: true,
	}, nil
}

// RestoreBackup restores a backup. Overwrite=false refuses a non-empty
// target; overwrite=true takes a safety backup first (same rule as the CLI).
func (g *GuiApi) RestoreBackup(path string, overwrite bool) (string, error) {
	if !overwrite {
		if err := backup.Restore(g.ctx.DataRoot, path, false, g.ctx.Logger); err != nil {
			return "", fail(ErrBackup, "restore refused", err.Error())
		}
		return "restored " + filepath.Base(path), nil
	}
	if _, err := backup.Create(g.ctx.DB, g.ctx.DataRoot, "", false, g.ctx.Logger); err != nil {
		return "", fail(ErrBackup, "safety backup failed; restore aborted", err.Error())
	}
	if err := backup.Restore(g.ctx.DataRoot, path, true, g.ctx.Logger); err != nil {
		return "", fail(ErrBackup, "restore failed", err.Error())
	}
	return "restored " + filepath.Base(path) + " (safety backup taken)", nil
}

// ConfigDTO surfaces storage + build identity for Settings.
type ConfigDTO struct {
	DataRoot      string `json:"data_root"`
	Version       string `json:"version"`
	SchemaVersion int    `json:"schema_version"`
}

// GetConfig returns the data root and build identity.
func (g *GuiApi) GetConfig() (ConfigDTO, error) {
	return ConfigDTO{
		DataRoot:      g.ctx.DataRoot,
		Version:       app.Info(),
		SchemaVersion: app.SchemaVersion,
	}, nil
}

// SetDataRoot validates a directory, saves it as the configured storage and
// asks for a restart: the running process keeps its open database (no live
// re-open, no half-migrated state).
func (g *GuiApi) SetDataRoot(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fail(ErrValidation, "empty storage path", "pick an existing directory")
	}
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return "", fail(ErrValidation, "not a directory: "+path, "pick an existing local folder")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fail(ErrValidation, "bad storage path "+path, err.Error())
	}
	if err := config.SavePointer(&config.Pointer{DataRoot: abs}); err != nil {
		return "", fail(ErrStorage, "save storage pointer", err.Error())
	}
	return "storage set to " + abs + " — restart the app to use it", nil
}
