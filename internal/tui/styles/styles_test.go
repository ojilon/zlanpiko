package styles

import (
	"strings"
	"testing"
)

func TestModes(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	Build("none")
	if Plain != true {
		t.Error("none mode must set Plain")
	}
	if got := Header("x"); got != "x" {
		t.Errorf("plain mode must not style: %q", got)
	}
	Build("auto")
	if Plain {
		t.Error("auto mode without NO_COLOR must style")
	}
	if got := Header("x"); !strings.Contains(got, "x") {
		t.Errorf("styled header lost text: %q", got)
	}
	Build("16")
	if Plain {
		t.Error("16 mode without NO_COLOR must style")
	}
	Build("auto")
}

func TestBarBounds(t *testing.T) {
	Build("none")
	if got := Bar(-1, 4); got != "░░░░" {
		t.Errorf("clamped low = %q", got)
	}
	if got := Bar(2, 4); got != "████" {
		t.Errorf("clamped high = %q", got)
	}
	if got := Bar(0.5, 4); got != "██░░" {
		t.Errorf("half bar = %q", got)
	}
	if got := Bar(0.4, 10); len([]rune(got)) != 10 {
		t.Errorf("width wrong: %q", got)
	}
	Build("auto")
}
