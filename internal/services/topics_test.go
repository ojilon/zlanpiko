package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"zlanpiko/internal/domain"
)

func TestTopicStatusCycle(t *testing.T) {
	db, root := setupDB(t)
	if _, err := CreateUnit(db, root, discardLogger(), "Maths", "", "", ""); err != nil {
		t.Fatal(err)
	}
	tp, err := CreateTopic(db, root, discardLogger(), "unit-001", "Algebra", "high")
	if err != nil {
		t.Fatal(err)
	}
	if tp.ReadingStatus != domain.ReadingUnread || tp.Priority != domain.PriorityHigh {
		t.Errorf("unexpected topic %+v", tp)
	}
	for _, want := range []domain.ReadingStatus{domain.ReadingPending, domain.ReadingRead, domain.ReadingPending} {
		updated, err := SetTopicStatus(db, root, discardLogger(), "unit-001", "topic-001", string(want))
		if err != nil {
			t.Fatal(err)
		}
		if updated.ReadingStatus != want {
			t.Errorf("status = %s, want %s", updated.ReadingStatus, want)
		}
	}
	if _, err := SetTopicStatus(db, root, discardLogger(), "unit-001", "topic-001", "bogus"); err == nil {
		t.Error("expected error for invalid status")
	}
	topics, err := ListTopics(db, "unit-001", "pending")
	if err != nil {
		t.Fatal(err)
	}
	if len(topics) != 1 {
		t.Errorf("filtered list = %d topics, want 1", len(topics))
	}
}

func TestTopicRenameAndMove(t *testing.T) {
	db, root := setupDB(t)
	if _, err := CreateUnit(db, root, discardLogger(), "U1", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateUnit(db, root, discardLogger(), "U2", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTopic(db, root, discardLogger(), "unit-001", "Shared", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTopic(db, root, discardLogger(), "unit-002", "Shared", ""); err != nil {
		t.Fatal(err)
	}
	// Duplicate name refused within the same unit.
	if _, err := CreateTopic(db, root, discardLogger(), "unit-001", "Shared", ""); err == nil {
		t.Error("expected duplicate topic-name error")
	}
	moved, err := MoveTopic(db, root, discardLogger(), "unit-001", "topic-001", "unit-002")
	if err != nil {
		t.Fatal(err)
	}
	if moved.UnitID != "unit-002" {
		t.Errorf("unit = %s", moved.UnitID)
	}
	if moved.ID == "topic-001" {
		t.Errorf("expected fresh id on collision, kept %s", moved.ID)
	}
	if _, err := os.Stat(filepath.Join(root, "units", "unit-002", "topics", moved.ID, "topic.json")); err != nil {
		t.Errorf("moved sidecar missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "units", "unit-001", "topics", "topic-001")); !os.IsNotExist(err) {
		t.Errorf("old folder should be gone: %v", err)
	}
	renamed, err := RenameTopic(db, root, discardLogger(), "unit-002", moved.ID, "Renamed")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "Renamed" {
		t.Errorf("name = %q", renamed.Name)
	}
}

func TestTopicDeleteGuardWithFileLink(t *testing.T) {
	db, root := setupDB(t)
	if _, err := CreateUnit(db, root, discardLogger(), "U", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTopic(db, root, discardLogger(), "unit-001", "T", ""); err != nil {
		t.Fatal(err)
	}
	// Simulate a linked file row (attachments arrive in Phase 4).
	if _, err := db.Exec(`INSERT INTO files(unit_id, topic_id, rel_path, created_at, updated_at)
		VALUES ('unit-001', 'topic-001', 'units/unit-001/topics/topic-001/reading/x.pdf', '2026-10-02T00:00:00Z', '2026-10-02T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := DeleteTopic(db, root, discardLogger(), "unit-001", "topic-001", false); err == nil {
		t.Error("expected confirm error with file link")
	} else {
		var cerr *domain.ConfirmRequiredError
		if !errors.As(err, &cerr) || cerr.Counts.Files != 1 {
			t.Errorf("expected file-count confirm error, got %v", err)
		}
	}
	if err := DeleteTopic(db, root, discardLogger(), "unit-001", "topic-001", true); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&n); err != nil || n != 0 {
		t.Errorf("linked file rows should cascade: n=%d err=%v", n, err)
	}
}
