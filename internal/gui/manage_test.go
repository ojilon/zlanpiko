package gui

import (
	"testing"
)

func TestUnitManage(t *testing.T) {
	_, api := openPhaseB(t)

	if _, err := api.CreateUnit("", "X"); err == nil {
		t.Fatalf("empty unit name must fail")
	}
	made, err := api.CreateUnit("Algebra", "M:101")
	if err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}
	if made.Name != "Algebra" || made.CoverageText != "—" {
		t.Fatalf("new unit card wrong: %+v", made)
	}
	ren, err := api.RenameUnit(made.UnitID, "Linear Algebra")
	if err != nil {
		t.Fatalf("RenameUnit: %v", err)
	}
	if ren.Name != "Linear Algebra" {
		t.Fatalf("rename wrong: %+v", ren)
	}
	arch, err := api.SetUnitArchived(made.UnitID, true)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if arch.Attention != "—" {
		t.Fatalf("archived card wrong: %+v", arch)
	}
	if _, err := api.SetUnitArchived(made.UnitID, false); err != nil {
		t.Fatalf("unarchive: %v", err)
	}

	d, err := api.GetUnitDetail("unit-001")
	if err != nil {
		t.Fatalf("GetUnitDetail: %v", err)
	}
	if len(d.Topics) != 5 || len(d.Tasks) != 2 {
		t.Fatalf("detail counts wrong: %d topics %d tasks", len(d.Topics), len(d.Tasks))
	}
	if _, err := api.GetUnitDetail("unit-999"); err == nil {
		t.Fatalf("unknown unit must fail")
	}

	if _, err := api.DeleteUnit(made.UnitID, false); err == nil {
		t.Fatalf("unconfirmed delete must fail")
	}
	if _, err := api.DeleteUnit(made.UnitID, true); err != nil {
		t.Fatalf("delete empty unit: %v", err)
	}
	if _, err := api.GetUnitDetail(made.UnitID); err == nil {
		t.Fatalf("deleted unit must be gone")
	}
}

func TestTopicManage(t *testing.T) {
	_, api := openPhaseB(t)

	list, err := api.ListTopics("unit-001", "read")
	if err != nil {
		t.Fatalf("ListTopics: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("read filter = %d, want 1", len(list))
	}
	if _, err := api.ListTopics("unit-999", ""); err == nil {
		t.Fatalf("unknown unit must fail")
	}
	if _, err := api.ListTopics("unit-001", "bogus"); err == nil {
		t.Fatalf("bad status filter must fail")
	}
	made, err := api.CreateTopic("unit-001", "new topic", "")
	if err != nil {
		t.Fatalf("CreateTopic: %v", err)
	}
	if made.Status != "unread" || made.UnitName != "Periodicity" {
		t.Fatalf("topic wrong: %+v", made)
	}
	if _, err := api.CreateTopic("unit-001", "x", "bogus"); err == nil {
		t.Fatalf("bad priority must fail")
	}
	if _, err := api.DeleteTopic("unit-001", made.ID, false); err == nil {
		t.Fatalf("unconfirmed topic delete must fail")
	}
	if _, err := api.DeleteTopic("unit-001", made.ID, true); err != nil {
		t.Fatalf("delete topic: %v", err)
	}
}

func TestTaskManage(t *testing.T) {
	_, api := openPhaseB(t)

	all, err := api.ListTasksFlat(TaskFilterDTO{})
	if err != nil {
		t.Fatalf("ListTasksFlat: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("all tasks = %d, want 3", len(all))
	}
	over, err := api.ListTasksFlat(TaskFilterDTO{Status: "overdue"})
	if err != nil {
		t.Fatalf("overdue filter: %v", err)
	}
	if len(over) != 1 {
		t.Fatalf("overdue = %d, want 1", len(over))
	}
	if _, err := api.ListTasksFlat(TaskFilterDTO{Status: "bogus"}); err == nil {
		t.Fatalf("bad status must fail")
	}
	unit, err := api.ListTasksFlat(TaskFilterDTO{UnitID: "unit-002"})
	if err != nil || len(unit) != 1 {
		t.Fatalf("unit filter wrong: %+v %v", unit, err)
	}

	made, err := api.CreateTask("unit-001", "lab writeup", "coursework", "2026-10-12", "high")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if made.Task.DueDisplay != "2026-10-12 00:00" {
		t.Fatalf("task due wrong: %+v", made.Task)
	}
	if _, err := api.CreateTask("unit-001", "x", "bogus-kind", "", ""); err == nil {
		t.Fatalf("bad kind must fail")
	}
	if _, err := api.CreateTask("unit-001", "x", "other", "not-a-date", ""); err == nil {
		t.Fatalf("bad due must fail")
	}
	if _, err := api.DeleteTask("unit-001", made.Task.ID, false); err == nil {
		t.Fatalf("unconfirmed task delete must fail")
	}
	if _, err := api.DeleteTask("unit-001", made.Task.ID, true); err != nil {
		t.Fatalf("delete task: %v", err)
	}
}
