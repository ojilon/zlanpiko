package installer

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keyType(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func TestListDrivesOnMachine(t *testing.T) {
	drives, err := ListDrives()
	if err != nil {
		t.Fatal(err)
	}
	if len(drives) == 0 {
		t.Fatal("expected at least one drive")
	}
	for _, d := range drives {
		if len(d.Letter) != 2 || d.Letter[1] != ':' {
			t.Errorf("bad drive letter %q", d.Letter)
		}
		if d.Kind == "" {
			t.Errorf("drive %s missing kind", d.Letter)
		}
	}
}

func TestListRootDirsPagesAtFifty(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 60; i++ {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("dir-%02d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	shown, total, err := ListRootDirs(root, folderPageSize)
	if err != nil {
		t.Fatal(err)
	}
	if total != 60 || len(shown) != 50 {
		t.Errorf("shown=%d total=%d, want 50/60", len(shown), total)
	}
	if shown[0] != "dir-00" {
		t.Errorf("not sorted: %q", shown[0])
	}
}

func TestResolveTypedChoice(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "Dev", "Projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveTypedChoice(root, "Dev")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "Dev") {
		t.Errorf("got %q", got)
	}
	got, err = ResolveTypedChoice(root, "Dev/Projects")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "Dev", "Projects") {
		t.Errorf("got %q", got)
	}
	for _, bad := range []string{"", "Nope", `C:\Windows`, `..\escape`, "file-missing/deep"} {
		if _, err := ResolveTypedChoice(root, bad); err == nil {
			t.Errorf("input %q should fail", bad)
		}
	}
}

func TestAskDirDefaultTypedAndChooseRefusal(t *testing.T) {
	var out bytes.Buffer
	ask := func(input string) (string, error) {
		return askDir(bufio.NewReader(strings.NewReader(input)), strings.NewReader(input), &out, "Label", `D:\Def`, "Zlanpiko")
	}
	got, err := ask("\n")
	if err != nil || got != `D:\Def` {
		t.Errorf("default = %q, %v", got, err)
	}
	out.Reset()
	got, err = ask("D:\\Mine\n")
	if err != nil || got != `D:\Mine` {
		t.Errorf("typed = %q, %v", got, err)
	}
	// "c" without a terminal must fail cleanly, not hang.
	out.Reset()
	if _, err := ask("c\n"); err == nil {
		t.Error("expected terminal error for chooser on pipes")
	}
}

func TestSharedReaderKeepsPipedAnswers(t *testing.T) {
	// Regression: one shared reader must serve consecutive prompts —
	// fresh readers per prompt discard buffered piped answers.
	in := "first\nsecond\n"
	reader := bufio.NewReader(strings.NewReader(in))
	var out bytes.Buffer
	a, err := askDir(reader, strings.NewReader(in), &out, "One", "d1", "Zlanpiko")
	if err != nil {
		t.Fatal(err)
	}
	b, err := askDir(reader, strings.NewReader(in), &out, "Two", "d2", "Zlanpiko")
	if err != nil {
		t.Fatal(err)
	}
	if a != "first" || b != "second" {
		t.Errorf("answers = %q, %q", a, b)
	}
}

func driveKeyModel(t *testing.T) *Picker {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"Dev", "Work"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p := newPicker("Test", "Zlanpiko", []Drive{{Letter: "T", Kind: "Local disk", Ready: true}})
	p.driveRootOverride = root // test seam: redirect drive root
	return p
}

func TestPickerDriveFolderConfirm(t *testing.T) {
	p := driveKeyModel(t)
	updated, _ := p.Update(keyType(tea.KeyEnter)) // select drive -> folders
	p = updated.(*Picker)
	if p.step != stepFolder {
		t.Fatalf("step = %v, want folder", p.step)
	}
	if len(p.folders) != 2 {
		t.Fatalf("folders = %v", p.folders)
	}
	updated, _ = p.Update(keyType(tea.KeyEnter)) // select first folder
	p = updated.(*Picker)
	if p.step != stepConfirm {
		t.Fatalf("step = %v, want confirm", p.step)
	}
	updated, _ = p.Update(keyRunes("y"))
	p = updated.(*Picker)
	if !p.done || p.aborted {
		t.Fatalf("picker should be done: %+v", p)
	}
	if base := filepath.Base(p.parent); base != "Dev" {
		t.Errorf("parent = %q", p.parent)
	}
	if got := p.finalPath(); filepath.Base(got) != "Zlanpiko" {
		t.Errorf("final = %q", got)
	}
}

func TestPickerRootAndManualAndBack(t *testing.T) {
	p := driveKeyModel(t)
	updated, _ := p.Update(keyType(tea.KeyEnter))
	p = updated.(*Picker)
	updated, _ = p.Update(keyRunes("r")) // drive root
	p = updated.(*Picker)
	if p.step != stepConfirm || p.parent == "" {
		t.Fatalf("root should go to confirm: %+v", p)
	}
	updated, _ = p.Update(keyRunes("n")) // back to folders
	p = updated.(*Picker)
	if p.step != stepFolder {
		t.Fatalf("n should go back: %v", p.step)
	}
	updated, _ = p.Update(keyRunes("t")) // manual mode
	p = updated.(*Picker)
	if p.step != stepManual {
		t.Fatalf("t should open manual: %v", p.step)
	}
	for _, r := range "Work" {
		updated, _ = p.Update(keyRunes(string(r)))
		p = updated.(*Picker)
	}
	updated, _ = p.Update(keyType(tea.KeyEnter))
	p = updated.(*Picker)
	if p.step != stepConfirm || filepath.Base(p.parent) != "Work" {
		t.Fatalf("manual should confirm Work: %+v", p)
	}
	updated, _ = p.Update(keyType(tea.KeyEsc)) // back to folders
	p = updated.(*Picker)
	if p.step != stepFolder {
		t.Fatalf("esc should go back: %v", p.step)
	}
	updated, _ = p.Update(keyRunes("q")) // abort
	p = updated.(*Picker)
	if !p.aborted {
		t.Error("q should abort")
	}
}
