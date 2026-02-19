package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveInstalledName(t *testing.T) {
	items := map[string]string{
		"cwd/dotai/create-role":      "Create a role",
		"golang/review/architecture": "Architecture review",
		"golang/review/code":         "Code review",
		"confluence/read-doc":        "Read Confluence doc",
		"gitlab/review-pipeline":     "Review pipeline",
	}

	tests := []struct {
		name    string
		query   string
		wantKey string
		wantVal string
		wantErr string
	}{
		{
			name:    "exact match",
			query:   "cwd/dotai/create-role",
			wantKey: "cwd/dotai/create-role",
			wantVal: "Create a role",
		},
		{
			name:    "unique substring",
			query:   "create-role",
			wantKey: "cwd/dotai/create-role",
			wantVal: "Create a role",
		},
		{
			name:    "unique substring read-doc",
			query:   "read-doc",
			wantKey: "confluence/read-doc",
			wantVal: "Read Confluence doc",
		},
		{
			name:    "unique regex anchor",
			query:   "^confluence",
			wantKey: "confluence/read-doc",
			wantVal: "Read Confluence doc",
		},
		{
			name:    "unique substring pipeline",
			query:   "pipeline",
			wantKey: "gitlab/review-pipeline",
			wantVal: "Review pipeline",
		},
		{
			name:    "case insensitive match",
			query:   "CREATE-ROLE",
			wantKey: "cwd/dotai/create-role",
			wantVal: "Create a role",
		},
		{
			name:    "ambiguous match",
			query:   "review",
			wantErr: "ambiguous",
		},
		{
			name:    "no match",
			query:   "nonexistent",
			wantErr: "not found",
		},
		{
			name:    "regex dot matches separator",
			query:   "golang.review.architecture",
			wantKey: "golang/review/architecture",
			wantVal: "Architecture review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val, err := resolveInstalledName(items, "task", tt.query)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
			if val != tt.wantVal {
				t.Errorf("val = %q, want %q", val, tt.wantVal)
			}
		})
	}
}

func TestResolveInstalledName_AmbiguousListsMatches(t *testing.T) {
	items := map[string]string{
		"golang/review/architecture": "Architecture review",
		"golang/review/code":         "Code review",
	}

	_, _, err := resolveInstalledName(items, "task", "review")
	if err == nil {
		t.Fatal("expected error for ambiguous match")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "golang/review/architecture") {
		t.Errorf("error should list 'golang/review/architecture': %s", errMsg)
	}
	if !strings.Contains(errMsg, "golang/review/code") {
		t.Errorf("error should list 'golang/review/code': %s", errMsg)
	}
}

func TestResolveInstalledName_EmptyMap(t *testing.T) {
	items := map[string]string{}

	_, _, err := resolveInstalledName(items, "agent", "anything")
	if err == nil {
		t.Fatal("expected error for empty map")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestResolveAllMatchingNames(t *testing.T) {
	items := map[string]string{
		"golang/review/architecture": "Architecture review",
		"golang/review/code":         "Code review",
		"golang/review/security":     "Security review",
		"confluence/read-doc":        "Read Confluence doc",
	}

	t.Run("exact match returns one", func(t *testing.T) {
		names, err := resolveAllMatchingNames(items, "task", "confluence/read-doc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 1 || names[0] != "confluence/read-doc" {
			t.Errorf("got %v, want [confluence/read-doc]", names)
		}
	})

	t.Run("ambiguous query returns all matches", func(t *testing.T) {
		names, err := resolveAllMatchingNames(items, "task", "golang/review")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 3 {
			t.Errorf("got %d matches, want 3: %v", len(names), names)
		}
	})

	t.Run("unique substring returns one", func(t *testing.T) {
		names, err := resolveAllMatchingNames(items, "task", "read-doc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 1 || names[0] != "confluence/read-doc" {
			t.Errorf("got %v, want [confluence/read-doc]", names)
		}
	})

	t.Run("no match returns error", func(t *testing.T) {
		_, err := resolveAllMatchingNames(items, "task", "nonexistent")
		if err == nil {
			t.Fatal("expected error for no match")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}

func TestPromptSelectFromList(t *testing.T) {
	names := []string{"golang/review/architecture", "golang/review/code", "golang/review/security"}

	t.Run("empty input cancels", func(t *testing.T) {
		w := &bytes.Buffer{}
		selected, err := promptSelectFromList(w, strings.NewReader("\n"), "task", "golang/review", names)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if selected != nil {
			t.Errorf("expected nil (cancelled), got %v", selected)
		}
	})

	t.Run("all selects all", func(t *testing.T) {
		w := &bytes.Buffer{}
		selected, err := promptSelectFromList(w, strings.NewReader("all\n"), "task", "golang/review", names)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 3 {
			t.Errorf("got %v, want all 3", selected)
		}
	})

	t.Run("comma-separated numbers", func(t *testing.T) {
		w := &bytes.Buffer{}
		selected, err := promptSelectFromList(w, strings.NewReader("1,3\n"), "task", "golang/review", names)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 2 {
			t.Fatalf("got %v, want 2 entries", selected)
		}
		if selected[0] != "golang/review/architecture" || selected[1] != "golang/review/security" {
			t.Errorf("got %v", selected)
		}
	})

	t.Run("range selection", func(t *testing.T) {
		w := &bytes.Buffer{}
		selected, err := promptSelectFromList(w, strings.NewReader("1-2\n"), "task", "golang/review", names)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(selected) != 2 {
			t.Fatalf("got %v, want 2 entries", selected)
		}
		if selected[0] != "golang/review/architecture" || selected[1] != "golang/review/code" {
			t.Errorf("got %v", selected)
		}
	})

	t.Run("invalid number returns error", func(t *testing.T) {
		w := &bytes.Buffer{}
		_, err := promptSelectFromList(w, strings.NewReader("99\n"), "task", "golang/review", names)
		if err == nil {
			t.Fatal("expected error for out-of-range number")
		}
	})

	t.Run("reversed range returns error", func(t *testing.T) {
		w := &bytes.Buffer{}
		_, err := promptSelectFromList(w, strings.NewReader("2-1\n"), "task", "golang/review", names)
		if err == nil {
			t.Fatal("expected error for reversed range")
		}
	})
}

func TestConfirmMultiRemoval_SingleItem(t *testing.T) {
	w := &bytes.Buffer{}
	// Non-TTY reader returns error, so just test the single-item prompt format
	// via the non-TTY path returning the expected error.
	_, err := confirmMultiRemoval(w, strings.NewReader(""), "task", []string{"my-task"}, false)
	if err == nil {
		t.Fatal("expected non-TTY error")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("expected '--yes' hint in error, got: %v", err)
	}
}

func TestConfirmMultiRemoval_MultipleItems_NonTTY(t *testing.T) {
	w := &bytes.Buffer{}
	_, err := confirmMultiRemoval(w, strings.NewReader(""), "role", []string{"role-a", "role-b"}, false)
	if err == nil {
		t.Fatal("expected non-TTY error")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("expected '--yes' hint in error, got: %v", err)
	}
}
