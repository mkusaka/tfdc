package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestSpinner_NonTTY_PrintsLines(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)

	s.Start("starting")
	s.Update("step 1")
	s.Update("step 2")
	s.Update("step 2") // duplicate, should not produce output
	s.Stop()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), buf.String())
	}
	if lines[0] != "starting" {
		t.Fatalf("expected first line 'starting', got %q", lines[0])
	}
	if lines[1] != "step 1" {
		t.Fatalf("expected second line 'step 1', got %q", lines[1])
	}
	if lines[2] != "step 2" {
		t.Fatalf("expected third line 'step 2', got %q", lines[2])
	}
}

func TestSpinner_NonTTY_StopBeforeStart(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	s.Stop() // should not panic
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestSpinner_NonTTY_DoubleStop(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	s.Start("hello")
	s.Stop()
	s.Stop() // should not panic
}

func TestSpinner_NonTTY_DoubleStart(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	s.Start("first")
	s.Start("second") // should be ignored
	s.Stop()

	if !strings.Contains(buf.String(), "first") {
		t.Fatalf("expected 'first' in output, got %q", buf.String())
	}
	if strings.Contains(buf.String(), "second") {
		t.Fatalf("did not expect 'second' in output, got %q", buf.String())
	}
}

func TestSpinner_NonTTY_UpdateBeforeStart(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	s.Update("before start") // should not produce output
	if buf.Len() != 0 {
		t.Fatalf("expected no output before start, got %q", buf.String())
	}
}

func TestIsTerminal_NonFile(t *testing.T) {
	var buf bytes.Buffer
	if isTerminal(&buf) {
		t.Fatalf("bytes.Buffer should not be detected as terminal")
	}
}

func TestSpinner_NonTTY_RapidUpdates(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	s.Start("init")
	for i := 0; i < 100; i++ {
		s.Update("msg")
	}
	s.Stop()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// Should have exactly 2 lines: "init" and "msg" (duplicates suppressed)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (duplicates suppressed), got %d: %q", len(lines), buf.String())
	}
}

func TestNew_ReturnsFalseForNonTTY(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	if s.isTTY {
		t.Fatalf("expected isTTY to be false for bytes.Buffer")
	}
}

func TestSpinner_NonTTY_StopIsFast(t *testing.T) {
	var buf bytes.Buffer
	s := New(&buf)
	s.Start("test")

	start := time.Now()
	s.Stop()
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Fatalf("Stop took too long for non-TTY spinner: %v", elapsed)
	}
}
