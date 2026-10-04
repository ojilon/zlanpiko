package gui

import (
	"strings"
	"testing"
)

func TestRunCommandNav(t *testing.T) {
	_, api := openPhaseB(t)
	for cmd, view := range map[string]string{
		"/dashboard": "Dashboard",
		"/units":     "Units",
		"/calendar":  "Calendar",
		"/analytics": "Analytics",
		"/settings":  "Settings",
		"timeline":   "Timeline", // slash optional, like the TUI
	} {
		r, err := api.RunCommand(cmd)
		if err != nil {
			t.Fatalf("%s: %v", cmd, err)
		}
		if r.Kind != "navigate" || r.View != view {
			t.Fatalf("%s -> %+v, want navigate %s", cmd, r, view)
		}
	}
	r, _ := api.RunCommand("/timeline next")
	if r.WeekOffset == nil || *r.WeekOffset != 1 {
		t.Fatalf("timeline next -> %+v", r)
	}
	r, _ = api.RunCommand("/timeline 2026-W41")
	if r.WeekOffset == nil {
		t.Fatalf("absolute week must resolve an offset: %+v", r)
	}
	r, _ = api.RunCommand("/help")
	if r.Kind != "message" || !strings.Contains(r.Message, "/tasks deadline") {
		t.Fatalf("help wrong: %+v", r)
	}
	r, _ = api.RunCommand("/unitz")
	if r.Kind != "error" || !strings.Contains(r.Hint, "/units") {
		t.Fatalf("typo recovery wrong: %+v", r)
	}
	r, _ = api.RunCommand("/version")
	if r.Kind != "message" || !strings.Contains(r.Message, "zlanpiko") {
		t.Fatalf("version wrong: %+v", r)
	}
}

func TestRunCommandMutations(t *testing.T) {
	_, api := openPhaseB(t)
	r, err := api.RunCommand("/topics status unit-001 topic-001 read")
	if err != nil {
		t.Fatalf("topic status: %v", err)
	}
	if r.Kind != "refresh" {
		t.Fatalf("topic status -> %+v", r)
	}
	r, err = api.RunCommand("/topics status unit-001 topic-001 bogus")
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if r.Kind != "error" {
		t.Fatalf("bad status must be an error result: %+v", r)
	}
	r, err = api.RunCommand("/tasks deadline unit-001 task-001 2026-10-09")
	if err != nil {
		t.Fatalf("deadline: %v", err)
	}
	if r.Kind != "refresh" || !strings.Contains(r.Message, "2026-10-09") {
		t.Fatalf("deadline -> %+v", r)
	}
	d, err := api.GetTaskDetail("unit-001", "task-001")
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if d.Task.DueDisplay != "2026-10-09 00:00" {
		t.Fatalf("deadline not moved: %+v", d.Task)
	}
	r, _ = api.RunCommand("/tasks status unit-001 task-001 completed")
	if r.Kind != "refresh" {
		t.Fatalf("task status -> %+v", r)
	}
	r, _ = api.RunCommand("/units list")
	if r.Kind != "message" || !strings.Contains(r.Message, "Periodicity") {
		t.Fatalf("units list wrong: %+v", r)
	}
}
