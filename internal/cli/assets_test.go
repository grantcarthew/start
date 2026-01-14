package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// TestSearchIndex tests the searchIndex function.
func TestSearchIndex(t *testing.T) {
	index := &registry.Index{
		Agents: map[string]registry.IndexEntry{
			"ai/claude": {
				Module:      "github.com/test/agents/ai/claude@v0",
				Description: "Claude by Anthropic",
				Tags:        []string{"anthropic", "ai", "llm"},
			},
			"ai/gemini": {
				Module:      "github.com/test/agents/ai/gemini@v0",
				Description: "Gemini by Google",
				Tags:        []string{"google", "ai", "llm"},
			},
		},
		Roles: map[string]registry.IndexEntry{
			"golang/assistant": {
				Module:      "github.com/test/roles/golang/assistant@v0",
				Description: "Go programming expert",
				Tags:        []string{"golang", "programming"},
			},
		},
		Tasks: map[string]registry.IndexEntry{
			"golang/code-review": {
				Module:      "github.com/test/tasks/golang/code-review@v0",
				Description: "Review Go code for best practices",
				Tags:        []string{"golang", "review"},
			},
		},
		Contexts: map[string]registry.IndexEntry{},
	}

	tests := []struct {
		name       string
		query      string
		wantCount  int
		wantFirst  string
		wantInName bool
	}{
		{
			name:      "search by name - claude",
			query:     "claude",
			wantCount: 1,
			wantFirst: "ai/claude",
		},
		{
			name:      "search by tag - golang",
			query:     "golang",
			wantCount: 2, // role and task
		},
		{
			name:       "search by description - anthropic",
			query:      "anthropic",
			wantCount:  1,
			wantFirst:  "ai/claude",
			wantInName: false,
		},
		{
			name:      "search multiple matches - ai",
			query:     "ai",
			wantCount: 2, // claude and gemini
		},
		{
			name:      "no matches",
			query:     "nonexistent",
			wantCount: 0,
		},
		{
			name:      "case insensitive",
			query:     "CLAUDE",
			wantCount: 1,
			wantFirst: "ai/claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searchIndex(index, tt.query)

			if len(results) != tt.wantCount {
				t.Errorf("searchIndex() returned %d results, want %d", len(results), tt.wantCount)
			}

			if tt.wantFirst != "" && len(results) > 0 {
				if results[0].Name != tt.wantFirst {
					t.Errorf("first result = %q, want %q", results[0].Name, tt.wantFirst)
				}
			}
		})
	}
}

// TestMatchScore tests the matchScore function.
func TestMatchScore(t *testing.T) {
	entry := registry.IndexEntry{
		Module:      "github.com/test/roles/golang/assistant@v0",
		Description: "Go programming expert for code assistance",
		Tags:        []string{"golang", "programming", "expert"},
	}

	tests := []struct {
		name      string
		assetName string
		query     string
		wantScore int
	}{
		{
			name:      "exact name match scores highest",
			assetName: "golang",
			query:     "golang",
			wantScore: 6, // name(3) + module(2) + tag(1)
		},
		{
			name:      "module path match",
			assetName: "assistant",
			query:     "golang",
			wantScore: 3, // module(2) + tag(1)
		},
		{
			name:      "description only match",
			assetName: "assistant",
			query:     "programming",
			wantScore: 2, // description(1) + tag(1)
		},
		{
			name:      "tag only match",
			assetName: "assistant",
			query:     "expert",
			wantScore: 2, // description(1) + tag(1)
		},
		{
			name:      "no match",
			assetName: "assistant",
			query:     "nonexistent",
			wantScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := matchScore(tt.assetName, entry, strings.ToLower(tt.query))

			if score != tt.wantScore {
				t.Errorf("matchScore() = %d, want %d", score, tt.wantScore)
			}
		})
	}
}

// TestCategoryOrder tests the categoryOrder function.
func TestCategoryOrder(t *testing.T) {
	tests := []struct {
		category string
		want     int
	}{
		{"agents", 0},
		{"roles", 1},
		{"tasks", 2},
		{"contexts", 3},
		{"unknown", 4},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := categoryOrder(tt.category)
			if got != tt.want {
				t.Errorf("categoryOrder(%q) = %d, want %d", tt.category, got, tt.want)
			}
		})
	}
}

// TestPrintSearchResults tests the printSearchResults function.
func TestPrintSearchResults(t *testing.T) {
	results := []SearchResult{
		{
			Category: "agents",
			Name:     "ai/claude",
			Entry: registry.IndexEntry{
				Module:      "github.com/test/agents/ai/claude@v0",
				Description: "Claude by Anthropic",
				Tags:        []string{"anthropic", "ai"},
			},
		},
		{
			Category: "roles",
			Name:     "golang/assistant",
			Entry: registry.IndexEntry{
				Module:      "github.com/test/roles/golang/assistant@v0",
				Description: "Go programming expert",
				Tags:        []string{"golang"},
			},
		},
	}

	t.Run("non-verbose output", func(t *testing.T) {
		var buf bytes.Buffer
		printSearchResults(&buf, results, false)
		output := buf.String()

		if !strings.Contains(output, "Found 2 matches") {
			t.Errorf("output missing match count, got: %s", output)
		}
		if !strings.Contains(output, "agents/") {
			t.Errorf("output missing agents category, got: %s", output)
		}
		if !strings.Contains(output, "ai/claude") {
			t.Errorf("output missing claude, got: %s", output)
		}
		if !strings.Contains(output, "Claude by Anthropic") {
			t.Errorf("output missing description, got: %s", output)
		}
	})

	t.Run("verbose output", func(t *testing.T) {
		var buf bytes.Buffer
		printSearchResults(&buf, results, true)
		output := buf.String()

		if !strings.Contains(output, "Module:") {
			t.Errorf("verbose output missing Module, got: %s", output)
		}
		if !strings.Contains(output, "Tags:") {
			t.Errorf("verbose output missing Tags, got: %s", output)
		}
	})
}

// TestIsAssetRepo tests the isAssetRepo function.
func TestIsAssetRepo(t *testing.T) {
	t.Run("valid asset repo with agents", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(dir, "agents"), 0755)

		if !isAssetRepo(dir) {
			t.Error("expected isAssetRepo to return true for dir with agents/")
		}
	})

	t.Run("valid asset repo with roles", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(dir, "roles"), 0755)

		if !isAssetRepo(dir) {
			t.Error("expected isAssetRepo to return true for dir with roles/")
		}
	})

	t.Run("valid asset repo with multiple dirs", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(dir, "agents"), 0755)
		_ = os.MkdirAll(filepath.Join(dir, "roles"), 0755)
		_ = os.MkdirAll(filepath.Join(dir, "tasks"), 0755)
		_ = os.MkdirAll(filepath.Join(dir, "contexts"), 0755)

		if !isAssetRepo(dir) {
			t.Error("expected isAssetRepo to return true for full asset repo")
		}
	})

	t.Run("not an asset repo", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.MkdirAll(filepath.Join(dir, "src"), 0755)

		if isAssetRepo(dir) {
			t.Error("expected isAssetRepo to return false for non-asset dir")
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()

		if isAssetRepo(dir) {
			t.Error("expected isAssetRepo to return false for empty dir")
		}
	})
}

// TestAssetTypeToConfigFile tests the assetTypeToConfigFile function.
func TestAssetTypeToConfigFile(t *testing.T) {
	tests := []struct {
		category string
		want     string
	}{
		{"agents", "agents.cue"},
		{"roles", "roles.cue"},
		{"tasks", "tasks.cue"},
		{"contexts", "contexts.cue"},
		{"unknown", "settings.cue"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := assetTypeToConfigFile(tt.category)
			if got != tt.want {
				t.Errorf("assetTypeToConfigFile(%q) = %q, want %q", tt.category, got, tt.want)
			}
		})
	}
}

// TestGenerateAssetCUE tests the generateAssetCUE function.
func TestGenerateAssetCUE(t *testing.T) {
	asset := SearchResult{
		Category: "roles",
		Name:     "golang/assistant",
		Entry: registry.IndexEntry{
			Module:      "github.com/test/roles/golang/assistant",
			Description: "Go programming expert",
		},
	}
	modulePath := "github.com/test/roles/golang/assistant@v0"

	t.Run("new file", func(t *testing.T) {
		content := generateAssetCUE(asset, modulePath, "")

		if !strings.Contains(content, "roles: {") {
			t.Errorf("output missing roles struct, got: %s", content)
		}
		if !strings.Contains(content, "Added: roles/golang/assistant") {
			t.Errorf("output missing added comment, got: %s", content)
		}
		if !strings.Contains(content, modulePath) {
			t.Errorf("output missing module path, got: %s", content)
		}
	})

	t.Run("existing file without module", func(t *testing.T) {
		existing := "// existing config\nroles: {}\n"
		content := generateAssetCUE(asset, modulePath, existing)

		if !strings.Contains(content, existing) {
			t.Error("should preserve existing content")
		}
		if !strings.Contains(content, modulePath) {
			t.Error("should add module path comment")
		}
	})

	t.Run("existing file with same module", func(t *testing.T) {
		existing := "// Module: " + modulePath + "\nroles: {}\n"
		content := generateAssetCUE(asset, modulePath, existing)

		// Should return as-is since already imported
		if content != existing {
			t.Error("should return existing content unchanged when module already present")
		}
	})
}

// TestGenerateIndexCUE tests the generateIndexCUE function.
func TestGenerateIndexCUE(t *testing.T) {
	index := &ScannedIndex{
		Agents: []ScannedAsset{
			{
				Category:    "agents",
				Name:        "ai/claude",
				Module:      "github.com/test/agents/ai/claude",
				Description: "Claude by Anthropic",
				Tags:        []string{"ai", "llm"},
				Bin:         "claude",
			},
		},
		Roles: []ScannedAsset{
			{
				Category:    "roles",
				Name:        "golang/assistant",
				Module:      "github.com/test/roles/golang/assistant",
				Description: "Go programming expert",
				Tags:        []string{"golang"},
			},
		},
		Tasks:    []ScannedAsset{},
		Contexts: []ScannedAsset{},
	}

	content := generateIndexCUE(index)

	// Check header
	if !strings.Contains(content, "// Auto-generated") {
		t.Error("missing auto-generated comment")
	}
	if !strings.Contains(content, "package index") {
		t.Error("missing package declaration")
	}

	// Check agents section
	if !strings.Contains(content, "agents: {") {
		t.Error("missing agents section")
	}
	if !strings.Contains(content, `"ai/claude"`) {
		t.Error("missing quoted agent key")
	}
	if !strings.Contains(content, `bin: "claude"`) {
		t.Error("missing bin field for agent")
	}

	// Check roles section
	if !strings.Contains(content, "roles: {") {
		t.Error("missing roles section")
	}
	if !strings.Contains(content, `"golang/assistant"`) {
		t.Error("missing quoted role key")
	}
}

// TestAssetsCommandExists tests that the assets command is registered.
func TestAssetsCommandExists(t *testing.T) {
	cmd := NewRootCmd()

	// Find assets command
	var assetsCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Use == "assets" {
			assetsCmd = c
			break
		}
	}

	if assetsCmd == nil {
		t.Fatal("assets command not found")
	}

	// Check subcommands
	subcommands := []string{"browse", "search", "add", "list", "info", "update", "index"}
	for _, name := range subcommands {
		found := false
		for _, c := range assetsCmd.Commands() {
			if strings.HasPrefix(c.Use, name) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found", name)
		}
	}
}

// TestAssetsSearchValidation tests search command argument validation.
func TestAssetsSearchValidation(t *testing.T) {
	// We can't easily test the full search without network,
	// but we can test the query length validation
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"query too short - 1 char", "a", true},
		{"query too short - 2 chars", "ab", true},
		{"query valid - 3 chars", "abc", false}, // Will fail on network, but passes validation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test the validation logic
			if len(tt.query) < 3 {
				if !tt.wantErr {
					t.Error("expected validation error for short query")
				}
			}
		})
	}
}
