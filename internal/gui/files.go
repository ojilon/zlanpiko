package gui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"zlanpiko/internal/filesystem"
	"zlanpiko/internal/importer"
)

// FileDTO is one file or directory row. Rel is slash-separated and
// root-relative; the frontend never builds absolute paths (docs/cs/02).
type FileDTO struct {
	Name       string `json:"name"`
	Rel        string `json:"rel"`
	IsDir      bool   `json:"is_dir"`
	Size       int64  `json:"size"`
	SizeText   string `json:"size_text"`
	ModDisplay string `json:"mod_display"`
}

func toFileDTO(i filesystem.Info) FileDTO {
	return FileDTO{
		Name:       i.Name,
		Rel:        i.Rel,
		IsDir:      i.IsDir,
		Size:       i.Size,
		SizeText:   humanBytes(i.Size, i.IsDir),
		ModDisplay: i.ModTime.Local().Format("2006-01-02 15:04"),
	}
}

func humanBytes(size int64, isDir bool) string {
	if isDir {
		return "—"
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func fileErr(op, rel string, err error) *GuiError {
	return fail(ErrStorage, op+" "+rel+": "+err.Error(), "")
}

// ListFiles lists one academic directory ("" = data-root top). Traversal
// outside the root is rejected by the filesystem package.
func (g *GuiApi) ListFiles(rel string) ([]FileDTO, error) {
	infos, err := filesystem.List(g.ctx.DataRoot, rel)
	if err != nil {
		return nil, fileErr("list", rel, err)
	}
	out := make([]FileDTO, 0, len(infos))
	for _, i := range infos {
		out = append(out, toFileDTO(i))
	}
	return out, nil
}

// SearchFiles ranks academic files by name (capped at 50).
func (g *GuiApi) SearchFiles(query string) ([]FileDTO, error) {
	infos, err := filesystem.Search(g.ctx.DataRoot, query, 50)
	if err != nil {
		return nil, fileErr("search", query, err)
	}
	out := make([]FileDTO, 0, len(infos))
	for _, i := range infos {
		out = append(out, toFileDTO(i))
	}
	return out, nil
}

// OpenFile opens a file with the OS default application.
func (g *GuiApi) OpenFile(rel string) (string, error) {
	if err := filesystem.Open(g.ctx.DataRoot, rel); err != nil {
		return "", fileErr("open", rel, err)
	}
	return "opened " + rel, nil
}

// MakeDir creates one academic directory.
func (g *GuiApi) MakeDir(rel string) (FileDTO, error) {
	if err := filesystem.Mkdir(g.ctx.DataRoot, rel); err != nil {
		return FileDTO{}, fileErr("mkdir", rel, err)
	}
	st, err := filesystem.Stat(g.ctx.DataRoot, rel)
	if err != nil {
		return FileDTO{}, fileErr("stat", rel, err)
	}
	return toFileDTO(st), nil
}

// RenameFile renames within its directory (collisions are refused, never
// silent overwrites).
func (g *GuiApi) RenameFile(rel, newName string) (FileDTO, error) {
	if err := filesystem.Rename(g.ctx.DataRoot, rel, newName); err != nil {
		return FileDTO{}, fileErr("rename", rel, err)
	}
	// Resolve the new rel for the response: same parent, new leaf.
	parent := rel
	for i := len(rel) - 1; i >= 0; i-- {
		if rel[i] == '/' {
			parent = rel[:i]
			break
		}
	}
	nrel := newName
	if parent != rel {
		nrel = parent + "/" + newName
	}
	st, err := filesystem.Stat(g.ctx.DataRoot, nrel)
	if err != nil {
		return FileDTO{}, fileErr("stat", nrel, err)
	}
	return toFileDTO(st), nil
}

// MoveFile moves a file into another academic directory.
func (g *GuiApi) MoveFile(srcRel, dstDirRel string) (FileDTO, error) {
	if err := filesystem.Move(g.ctx.DataRoot, srcRel, dstDirRel); err != nil {
		return FileDTO{}, fileErr("move", srcRel, err)
	}
	name := srcRel
	for i := len(srcRel) - 1; i >= 0; i-- {
		if srcRel[i] == '/' {
			name = srcRel[i+1:]
			break
		}
	}
	nrel := name
	if dstDirRel != "" {
		nrel = dstDirRel + "/" + name
	}
	st, err := filesystem.Stat(g.ctx.DataRoot, nrel)
	if err != nil {
		return FileDTO{}, fileErr("stat", nrel, err)
	}
	return toFileDTO(st), nil
}

// DeleteFile removes one file or (with recursive) directory. The frontend
// confirms first; there is no trash recovery.
func (g *GuiApi) DeleteFile(rel string, recursive bool) (string, error) {
	if err := filesystem.Delete(g.ctx.DataRoot, rel, recursive); err != nil {
		return "", fileErr("delete", rel, err)
	}
	return "deleted " + rel, nil
}

// ImportPlannedDTO is one source file's fate in a preview.
type ImportPlannedDTO struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	DestRel string `json:"dest_rel"`
	Action  string `json:"action"`
	Reason  string `json:"reason"`
	Renamed bool   `json:"renamed"`
}

// ImportPlanDTO is a frozen preview: executing it changes nothing until
// ConfirmImport(token) runs. Plans expire with the process.
type ImportPlanDTO struct {
	Token   string             `json:"token"`
	Source  string             `json:"source"`
	DestDir string             `json:"dest_dir"`
	Copies  int                `json:"copies"`
	Plans   []ImportPlannedDTO `json:"plans"`
}

// ImportReportDTO tallies a confirmed import.
type ImportReportDTO struct {
	Copied  int `json:"copied"`
	Skipped int `json:"skipped"`
	Failed  int `json:"failed"`
}

var (
	planMu    sync.Mutex
	planSeq   int
	planCache = map[string]*importer.Plan{}
)

// PreviewImport plans a bulk import without touching anything: source is an
// absolute outside path, dest follows importer rules ("" = inbox/<date>,
// unit id, unit/topic or unit/task reference, or root-relative directory).
// Like the CLI `.` flow it never silently imports (docs/cs/02).
func (g *GuiApi) PreviewImport(srcAbs, dest string, recursive bool) (ImportPlanDTO, error) {
	if srcAbs == "" {
		return ImportPlanDTO{}, fail(ErrValidation, "empty source path", "pick a folder or file to import")
	}
	pv, err := importer.Preview(g.ctx.DB, g.ctx.DataRoot, srcAbs, importer.Options{
		Dest:      dest,
		Recursive: recursive,
		Collision: importer.CollisionRename,
	})
	if err != nil {
		return ImportPlanDTO{}, fail(ErrValidation, "cannot preview "+srcAbs, err.Error())
	}
	planMu.Lock()
	planSeq++
	token := fmt.Sprintf("plan-%d-%d", time.Now().Unix(), planSeq)
	planCache[token] = pv
	planMu.Unlock()
	plans := make([]ImportPlannedDTO, 0, len(pv.Plans))
	for _, p := range pv.Plans {
		plans = append(plans, ImportPlannedDTO{
			Name:    p.Name,
			Size:    p.Size,
			DestRel: p.DestRel,
			Action:  p.Action,
			Reason:  p.Reason,
			Renamed: p.Renamed,
		})
	}
	return ImportPlanDTO{
		Token:   token,
		Source:  pv.Source,
		DestDir: pv.DestDir,
		Copies:  pv.Copies(),
		Plans:   plans,
	}, nil
}

// ConfirmImport executes a previewed plan. Unknown or already-used tokens
// are rejected; plans are single-use.
func (g *GuiApi) ConfirmImport(token string) (ImportReportDTO, error) {
	planMu.Lock()
	pv, ok := planCache[token]
	if ok {
		delete(planCache, token)
	}
	planMu.Unlock()
	if !ok {
		return ImportReportDTO{}, fail(ErrNotFound, "unknown import plan", "preview again before confirming")
	}
	rep, err := importer.Execute(context.Background(), g.ctx.DB, g.ctx.DataRoot, pv, g.ctx.Logger)
	if err != nil {
		return ImportReportDTO{}, fail(ErrStorage, "import failed", err.Error())
	}
	return ImportReportDTO{Copied: rep.Copied, Skipped: rep.Skipped, Failed: rep.Failed}, nil
}
