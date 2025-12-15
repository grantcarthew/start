package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIndex_ValidIndex(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid index CUE file
	indexCUE := `
package index

agents: {
	"ai/claude": {
		module:      "github.com/test/claude@v0"
		description: "Claude AI"
		bin:         "claude"
	}
	"ai/gemini": {
		module:      "github.com/test/gemini@v0"
		description: "Gemini AI"
		bin:         "gemini"
	}
}

tasks: {
	"golang/review": {
		module:      "github.com/test/review@v0"
		description: "Code review"
		tags:        ["golang", "review"]
	}
}

roles: {
	"dev/expert": {
		module: "github.com/test/expert@v0"
	}
}

contexts: {
	"env/local": {
		module: "github.com/test/local@v0"
	}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.cue"), []byte(indexCUE), 0644); err != nil {
		t.Fatalf("writing test index: %v", err)
	}

	index, err := LoadIndex(tmpDir, nil)
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Verify agents
	if len(index.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(index.Agents))
	}

	claude, ok := index.Agents["ai/claude"]
	if !ok {
		t.Error("missing ai/claude agent")
	} else {
		if claude.Module != "github.com/test/claude@v0" {
			t.Errorf("wrong module: %s", claude.Module)
		}
		if claude.Bin != "claude" {
			t.Errorf("wrong bin: %s", claude.Bin)
		}
	}

	// Verify tasks
	if len(index.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(index.Tasks))
	}

	review, ok := index.Tasks["golang/review"]
	if !ok {
		t.Error("missing golang/review task")
	} else {
		if len(review.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(review.Tags))
		}
	}

	// Verify roles
	if len(index.Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(index.Roles))
	}

	// Verify contexts
	if len(index.Contexts) != 1 {
		t.Errorf("expected 1 context, got %d", len(index.Contexts))
	}
}

func TestLoadIndex_EmptyCategories(t *testing.T) {
	tmpDir := t.TempDir()

	// Index with only agents
	indexCUE := `
package index

agents: {
	"ai/test": {
		module: "github.com/test/test@v0"
		bin:    "test"
	}
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.cue"), []byte(indexCUE), 0644); err != nil {
		t.Fatalf("writing test index: %v", err)
	}

	index, err := LoadIndex(tmpDir, nil)
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	if len(index.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(index.Agents))
	}
	if len(index.Tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(index.Tasks))
	}
	if len(index.Roles) != 0 {
		t.Errorf("expected 0 roles, got %d", len(index.Roles))
	}
	if len(index.Contexts) != 0 {
		t.Errorf("expected 0 contexts, got %d", len(index.Contexts))
	}
}

func TestLoadIndex_InvalidCUE(t *testing.T) {
	tmpDir := t.TempDir()

	// Invalid CUE syntax
	indexCUE := `
package index

agents: {
	"ai/test": {
		module: "github.com/test/test@v0"
		// Missing closing brace
`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.cue"), []byte(indexCUE), 0644); err != nil {
		t.Fatalf("writing test index: %v", err)
	}

	_, err := LoadIndex(tmpDir, nil)
	if err == nil {
		t.Error("expected error for invalid CUE, got nil")
	}
}

func TestLoadIndex_NonexistentDir(t *testing.T) {
	_, err := LoadIndex("/nonexistent/directory/12345", nil)
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}

func TestLoadIndex_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadIndex(tmpDir, nil)
	if err == nil {
		t.Error("expected error for empty directory, got nil")
	}
}
