package app

import (
	"strings"
	"testing"
)

func TestInfoContainsMetadata(t *testing.T) {
	info := Info()
	for _, want := range []string{Name, Version, "schema 1"} {
		if !strings.Contains(info, want) {
			t.Errorf("Info() = %q, want it to contain %q", info, want)
		}
	}
}

func TestVersionDefaultNonEmpty(t *testing.T) {
	if Version == "" {
		t.Error("Version must not be empty")
	}
}
