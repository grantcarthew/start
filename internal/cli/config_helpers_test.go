package cli

import (
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
