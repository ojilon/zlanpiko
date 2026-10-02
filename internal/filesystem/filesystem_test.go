package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"zlanpiko/internal/domain"
)

func TestJoinRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	for _, rel := range [][]string{
		{".."},
		{"units", "..", "..", "evil"},
		{`C:\Windows`},
		{`\\server\share`},
		{"units", "CON"},
		{""},
	} {
		if _, err := Join(root, rel...); err == nil {
			t.Errorf("Join(%q) should fail", rel)
		}
	}
	got, err := Join(root, "units", "unit-001")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "units", "unit-001") {
		t.Errorf("Join = %q", got)
	}
}

func TestWriteJSONIsAtomicAndReadable(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "units", "unit-001", "unit.json")
	if err := WriteJSON(path, map[string]string{"id": "unit-001"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{\n  \"id\": \"unit-001\"\n}\n" {
		t.Errorf("unexpected sidecar content %q", raw)
	}
	leftovers, _ := filepath.Glob(filepath.Join(root, "units", "unit-001", ".tmp-*"))
	if len(leftovers) != 0 {
		t.Errorf("temp files left behind: %v", leftovers)
	}
}

func TestRecordDirsAndSkeletons(t *testing.T) {
	root := t.TempDir()
	if err := EnsureUnitSkeleton(root, "unit-001"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureTopicSkeleton(root, "unit-001", "topic-001"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureTaskSkeleton(root, "unit-001", domain.TaskAssignment, "task-001"); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{
		"units/unit-001/topics",
		"units/unit-001/notes",
		"units/unit-001/topics/topic-001/reading",
		"units/unit-001/topics/topic-001/summaries",
		"units/unit-001/assignments/task-001/submissions",
	} {
		if st, err := os.Stat(filepath.Join(root, filepath.FromSlash(dir))); err != nil || !st.IsDir() {
			t.Errorf("skeleton dir %s missing: %v", dir, err)
		}
	}
	if _, err := KindGroup("bogus"); err == nil {
		t.Error("expected error for unknown kind")
	}
}

func TestMoveDirCollisionSafe(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "a")
	dst := filepath.Join(root, "b")
	if err := EnsureDir(src); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDir(dst); err != nil {
		t.Fatal(err)
	}
	if err := MoveDir(src, dst); err == nil {
		t.Error("expected collision error when dst exists")
	}
	if st, err := os.Stat(src); err != nil || !st.IsDir() {
		t.Errorf("src must survive a refused move: %v", err)
	}
}
