package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesRoundTrip(t *testing.T) {
	ctx, api := openPhaseB(t)

	top, err := api.ListFiles("")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	found := false
	for _, f := range top {
		if f.Name == "units" && f.IsDir {
			found = true
		}
	}
	if !found {
		t.Fatalf("root listing misses units/: %+v", top)
	}
	units, err := api.ListFiles("units")
	if err != nil {
		t.Fatalf("ListFiles units: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("units listing = %d, want 2", len(units))
	}

	made, err := api.MakeDir("scratch")
	if err != nil {
		t.Fatalf("MakeDir: %v", err)
	}
	if !made.IsDir {
		t.Fatalf("mkdir not a dir: %+v", made)
	}
	renamed, err := api.RenameFile("scratch", "scratch2")
	if err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if renamed.Rel != "scratch2" {
		t.Fatalf("rename rel %q", renamed.Rel)
	}
	if _, err := api.DeleteFile("scratch2", false); err != nil {
		// Empty dir may need recursive on some platforms; retry so.
		if _, err2 := api.DeleteFile("scratch2", true); err2 != nil {
			t.Fatalf("DeleteFile: %v / %v", err, err2)
		}
	}

	hits, err := api.SearchFiles("unit.json")
	if err != nil {
		t.Fatalf("SearchFiles: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("search for unit.json found nothing")
	}
	_ = ctx
}

func TestImportPreviewConfirm(t *testing.T) {
	_, api := openPhaseB(t)
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("seed src: %v", err)
	}
	pv, err := api.PreviewImport(src, "", false)
	if err != nil {
		t.Fatalf("PreviewImport: %v", err)
	}
	if pv.Copies != 1 || len(pv.Plans) != 1 || pv.Token == "" {
		t.Fatalf("plan wrong: %+v", pv)
	}
	rep, err := api.ConfirmImport(pv.Token)
	if err != nil {
		t.Fatalf("ConfirmImport: %v", err)
	}
	if rep.Copied != 1 {
		t.Fatalf("report wrong: %+v", rep)
	}
	if _, err := api.ConfirmImport(pv.Token); err == nil {
		t.Fatalf("plan reuse must fail")
	} else if gerr, ok := err.(*GuiError); !ok || gerr.Code != ErrNotFound {
		t.Fatalf("reuse error wrong: %v", err)
	}
}

func TestReportAndBackup(t *testing.T) {
	_, api := openPhaseB(t)
	rep, err := api.GetReport("")
	if err != nil {
		t.Fatalf("GetReport: %v", err)
	}
	if !strings.Contains(rep.Text, "Periodicity") || !strings.Contains(rep.JSON, "Periodicity") {
		t.Fatalf("report misses fixture unit")
	}

	// NOTE: Restore is not exercised here: it replaces the database file
	// under the open test handle. The CLI suite covers restore end to end.
	created, err := api.CreateBackup(false)
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if !created.Readable || created.Size <= 0 {
		t.Fatalf("backup wrong: %+v", created)
	}
	list, err := api.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("no backups listed")
	}
	if _, err := api.VerifyBackup(created.Path); err != nil {
		t.Fatalf("VerifyBackup: %v", err)
	}

	cfg, err := api.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if cfg.DataRoot == "" || !strings.Contains(cfg.Version, "zlanpiko") {
		t.Fatalf("config wrong: %+v", cfg)
	}
	if _, err := api.SetDataRoot(filepath.Join(cfg.DataRoot, "nope")); err == nil {
		t.Fatalf("missing dir must fail validation")
	}
}
