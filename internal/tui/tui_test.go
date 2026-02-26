package tui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestCategoryColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  interface{} // expected *color.Color pointer
	}{
		{"agents", ColorAgents},
		{"AGENTS", ColorAgents},
		{"roles", ColorRoles},
		{"Roles", ColorRoles},
		{"contexts", ColorContexts},
		{"CONTEXTS", ColorContexts},
		{"tasks", ColorTasks},
		{"Tasks", ColorTasks},
		{"unknown", ColorDim},
		{"", ColorDim},
		{"prompts", ColorDim}, // not in switch, falls to default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CategoryColor(tt.input)
			if got != tt.want {
				t.Errorf("CategoryColor(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAnnotate(t *testing.T) {
	t.Parallel()

	t.Run("plain string", func(t *testing.T) {
		result := Annotate("hello")
		if !strings.Contains(result, "hello") {
			t.Errorf("Annotate(%q) = %q, want to contain %q", "hello", result, "hello")
		}
		if !strings.Contains(result, "(") || !strings.Contains(result, ")") {
			t.Errorf("Annotate(%q) = %q, want parentheses", "hello", result)
		}
	})

	t.Run("formatted string", func(t *testing.T) {
		result := Annotate("count %d", 42)
		if !strings.Contains(result, "count 42") {
			t.Errorf("Annotate() = %q, want to contain %q", result, "count 42")
		}
		if !strings.Contains(result, "(") || !strings.Contains(result, ")") {
			t.Errorf("Annotate() = %q, want parentheses", result)
		}
	})

	t.Run("key=value format", func(t *testing.T) {
		result := Annotate("key=%s", "val")
		if !strings.Contains(result, "key=val") {
			t.Errorf("Annotate() = %q, want to contain %q", result, "key=val")
		}
	})
}

func TestBracket(t *testing.T) {
	t.Parallel()

	t.Run("plain string", func(t *testing.T) {
		result := Bracket("hello")
		if !strings.Contains(result, "hello") {
			t.Errorf("Bracket(%q) = %q, want to contain %q", "hello", result, "hello")
		}
		if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
			t.Errorf("Bracket(%q) = %q, want square brackets", "hello", result)
		}
	})

	t.Run("formatted string", func(t *testing.T) {
		result := Bracket("count %d", 42)
		if !strings.Contains(result, "count 42") {
			t.Errorf("Bracket() = %q, want to contain %q", result, "count 42")
		}
		if !strings.Contains(result, "[") || !strings.Contains(result, "]") {
			t.Errorf("Bracket() = %q, want square brackets", result)
		}
	})

	t.Run("key=value format", func(t *testing.T) {
		result := Bracket("key=%s", "val")
		if !strings.Contains(result, "key=val") {
			t.Errorf("Bracket() = %q, want to contain %q", result, "key=val")
		}
	})
}

func TestProgress_NonTTY(t *testing.T) {
	t.Parallel()
	// bytes.Buffer is not *os.File, so NewProgress treats it as non-TTY.
	// Update and Done must be no-ops.
	buf := &bytes.Buffer{}
	p := NewProgress(buf, false)

	p.Update("loading %d%%", 50)
	p.Update("loading %d%%", 100)
	p.Done()

	if buf.Len() != 0 {
		t.Errorf("Progress should be no-op for non-TTY writer, got %q", buf.String())
	}
}

func TestProgress_Quiet(t *testing.T) {
	t.Parallel()
	// Use a real *os.File so the writer would pass the type assertion;
	// quiet=true must still suppress all output.
	f, err := os.CreateTemp(t.TempDir(), "prog")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	p := NewProgress(f, true)
	p.Update("loading %d%%", 50)
	p.Done()

	n, _ := f.Seek(0, io.SeekEnd)
	if n != 0 {
		t.Errorf("Progress should be no-op when quiet, but %d bytes were written", n)
	}
}

func TestProgress_NonTTY_DoneOnEmpty(t *testing.T) {
	t.Parallel()
	// Calling Done without any Update should not write anything.
	buf := &bytes.Buffer{}
	p := NewProgress(buf, false)
	p.Done()

	if buf.Len() != 0 {
		t.Errorf("Done() on empty progress wrote output: %q", buf.String())
	}
}
