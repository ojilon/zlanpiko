package services

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zlanpiko/internal/domain"
)

func TestUnitLifecycle(t *testing.T) {
	db, root := setupDB(t)
	u, err := CreateUnit(db, root, discardLogger(), "Linear Algebra", "MATH201", "blue", "")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID != "unit-001" || u.Status != domain.UnitActive {
		t.Errorf("unexpected unit %+v", u)
	}
	if _, err := CreateUnit(db, root, discardLogger(), "Linear Algebra", "", "", ""); err == nil {
		t.Error("expected duplicate-name error")
	}
	got, err := GetUnit(db, "unit-001")
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != "MATH201" {
		t.Errorf("code = %q", got.Code)
	}
	renamed, err := RenameUnit(db, root, discardLogger(), "unit-001", "Advanced Linear Algebra")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Advanced Linear Algebra" {
		t.Errorf("name = %q", renamed.Name)
	}
	// Sidecar mirrors the row.
	raw, err := os.ReadFile(filepath.Join(root, "units", "unit-001", "unit.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Advanced Linear Algebra") {
		t.Errorf("sidecar stale: %s", raw)
	}
}

func TestUnitArchiveAndDeleteGuards(t *testing.T) {
	db, root := setupDB(t)
	if _, err := CreateUnit(db, root, discardLogger(), "Physics", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTopic(db, root, discardLogger(), "unit-001", "Mechanics", ""); err != nil {
		t.Fatal(err)
	}
	arch, err := SetUnitArchived(db, root, discardLogger(), "unit-001", true)
	if err != nil {
		t.Fatal(err)
	}
	if arch.Status != domain.UnitArchived {
		t.Errorf("status = %s", arch.Status)
	}
	if _, err := CreateTopic(db, root, discardLogger(), "unit-001", "Optics", ""); err == nil {
		t.Error("expected refusal to add topic to archived unit")
	}
	if err := DeleteUnit(db, root, discardLogger(), "unit-001", false); err == nil {
		t.Error("expected confirm error")
	} else {
		var cerr *domain.ConfirmRequiredError
		if !errors.As(err, &cerr) {
			t.Errorf("expected ConfirmRequiredError, got %T", err)
		}
	}
	if _, err := SetUnitArchived(db, root, discardLogger(), "unit-001", false); err != nil {
		t.Fatal(err)
	}
	if err := DeleteUnit(db, root, discardLogger(), "unit-001", true); err != nil {
		t.Fatal(err)
	}
	if _, err := GetUnit(db, "unit-001"); err == nil {
		t.Error("unit should be gone")
	}
	if _, err := os.Stat(filepath.Join(root, "units", "unit-001")); !os.IsNotExist(err) {
		t.Errorf("unit folder should be gone: %v", err)
	}
	if units, err := ListUnits(db, false); err != nil || len(units) != 0 {
		t.Errorf("list = %v, %v", units, err)
	}
}

func TestUnitIDNeverReused(t *testing.T) {
	db, root := setupDB(t)
	if _, err := CreateUnit(db, root, discardLogger(), "A", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateUnit(db, root, discardLogger(), "B", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := DeleteUnit(db, root, discardLogger(), "unit-001", true); err != nil {
		t.Fatal(err)
	}
	c, err := CreateUnit(db, root, discardLogger(), "C", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != "unit-003" {
		t.Errorf("id = %q, want unit-003 (no reuse)", c.ID)
	}
}
