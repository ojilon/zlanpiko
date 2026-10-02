package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewJSONWritesRecord(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, Options{})
	l.Info("hello", "k", "v")
	out := buf.String()
	for _, want := range []string{`"msg":"hello"`, `"k":"v"`} {
		if !strings.Contains(out, want) {
			t.Errorf("log output = %q, want it to contain %q", out, want)
		}
	}
}

func TestDebugSuppressedAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, Options{})
	l.Debug("quiet")
	if buf.Len() != 0 {
		t.Errorf("expected no output at info level, got %q", buf.String())
	}

	buf.Reset()
	l = New(&buf, Options{Debug: true})
	l.Debug("loud")
	if !strings.Contains(buf.String(), "loud") {
		t.Errorf("expected debug output when enabled, got %q", buf.String())
	}
}
