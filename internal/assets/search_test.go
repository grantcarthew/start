package assets

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grantcarthew/start/internal/registry"
)

// TestSearchIndex tests the SearchIndex function.
func TestSearchIndex(t *testing.T) {
	t.Parallel()

	index := &registry.Index{
		Agents: map[string]registry.IndexEntry{
			"ai/claude": {
				Module:      "github.com/test/agents/ai/claude@v0",
				Description: "Anthropic Claude AI",
				Tags:        []string{"ai", "llm"},
			},
		},
		Roles: map[string]registry.IndexEntry{
			"golang/assistant": {
				Module:      "github.com/test/roles/golang/assistant@v0",
				Description: "Go programming expert",
				Tags:        []string{"golang", "programming"},
			},
			"golang/code-review": {
				Module:      "github.com/test/roles/golang/code-review@v0",
				Description: "Review Go code for quality",
				Tags:        []string{"golang", "review"},
			},
		},
		Tasks: map[string]registry.IndexEntry{
			"start/commit": {
				Module:      "github.com/test/tasks/start/commit@v0",
				Description: "Create git commit",
				Tags:        []string{"git", "commit"},
			},
		},
		Contexts: map[string]registry.IndexEntry{
			"cwd/agents-md": {
				Module:      "github.com/test/contexts/cwd/agents-md@v0",
				Description: "Read AGENTS.md file",
				Tags:        []string{"repository", "guidelines"},
			},
		},
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantFirst string // category/name of first result
	}{
		{
			name:      "find by exact name",
			query:     "claude",
			wantCount: 1,
			wantFirst: "agents/ai/claude",
		},
		{
			name:      "find by partial name",
			query:     "golang",
			wantCount: 2,
			wantFirst: "roles/golang/assistant", // or golang/code-review, both valid
		},
		{
			name:      "find by description",
			query:     "programming",
			wantCount: 1,
			wantFirst: "roles/golang/assistant",
		},
		{
			name:      "find by tag",
			query:     "commit",
			wantCount: 1,
			wantFirst: "tasks/start/commit",
		},
		{
			name:      "no matches",
			query:     "nonexistent",
			wantCount: 0,
		},
		{
			name:      "multiple matches",
			query:     "review",
			wantCount: 1,
			wantFirst: "roles/golang/code-review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SearchIndex(index, tt.query)

			if len(results) != tt.wantCount {
				t.Errorf("SearchIndex() returned %d results, want %d", len(results), tt.wantCount)
			}

			if tt.wantCount > 0 && tt.wantFirst != "" {
				first := results[0].Category + "/" + results[0].Name
				if first != tt.wantFirst {
					// For "golang" query, either result is valid
					if tt.query == "golang" {
						validResults := map[string]bool{
							"roles/golang/assistant":   true,
							"roles/golang/code-review": true,
						}
						if !validResults[first] {
							t.Errorf("SearchIndex() first result = %q, want one of golang/assistant or golang/code-review", first)
						}
					} else {
						t.Errorf("SearchIndex() first result = %q, want %q", first, tt.wantFirst)
					}
				}
			}
		})
	}
}

// TestMatchScore tests the matchScore function.
func TestMatchScore(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

// TestSearchCategory tests the searchCategory function.
func TestSearchCategory(t *testing.T) {
	t.Parallel()

	entries := map[string]registry.IndexEntry{
		"golang/assistant": {
			Module:      "github.com/test/roles/golang/assistant@v0",
			Description: "Go programming expert",
			Tags:        []string{"golang", "programming"},
		},
		"python/assistant": {
			Module:      "github.com/test/roles/python/assistant@v0",
			Description: "Python programming expert",
			Tags:        []string{"python", "programming"},
		},
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{
			name:      "find golang",
			query:     "golang",
			wantCount: 1,
		},
		{
			name:      "find programming (both match)",
			query:     "programming",
			wantCount: 2,
		},
		{
			name:      "no match",
			query:     "javascript",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := searchCategory("roles", entries, tt.query)

			if len(results) != tt.wantCount {
				t.Errorf("searchCategory() returned %d results, want %d", len(results), tt.wantCount)
			}

			// Verify all results are from the correct category
			for _, r := range results {
				if r.Category != "roles" {
					t.Errorf("searchCategory() returned result with category %q, want %q", r.Category, "roles")
				}
			}
		})
	}
}

// TestSearchIndex_NilIndex documents Bug: SearchIndex panics on nil *registry.Index.
// SearchIndex accesses index.Agents, index.Roles, etc. without a nil check.
func TestSearchIndex_NilIndex(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SearchIndex(nil, ...) panicked: %v", r)
		}
	}()

	results := SearchIndex(nil, "test")
	if len(results) != 0 {
		t.Errorf("SearchIndex(nil, ...) returned %d results, want 0", len(results))
	}
}

// TestSearchIndex_EmptyIndex tests SearchIndex with an empty (non-nil) index.
func TestSearchIndex_EmptyIndex(t *testing.T) {
	t.Parallel()

	index := &registry.Index{}
	results := SearchIndex(index, "test")

	if len(results) != 0 {
		t.Errorf("SearchIndex(empty, ...) returned %d results, want 0", len(results))
	}
}

// TestSearchIndex_NilMaps tests SearchIndex with non-nil Index but nil category maps.
func TestSearchIndex_NilMaps(t *testing.T) {
	t.Parallel()

	index := &registry.Index{
		Agents:   nil,
		Roles:    nil,
		Tasks:    nil,
		Contexts: nil,
	}
	results := SearchIndex(index, "test")

	if len(results) != 0 {
		t.Errorf("SearchIndex(nil maps, ...) returned %d results, want 0", len(results))
	}
}

// TestSearchResultOrdering tests that results are ordered correctly.
func TestSearchResultOrdering(t *testing.T) {
	t.Parallel()

	index := &registry.Index{
		Roles: map[string]registry.IndexEntry{
			"golang/assistant": {
				Module:      "github.com/test/roles/golang/assistant@v0",
				Description: "Go programming expert",
				Tags:        []string{"golang"},
			},
			"golang/code-review": {
				Module:      "github.com/test/roles/golang/code-review@v0",
				Description: "Review code quality",
				Tags:        []string{"golang"},
			},
		},
		Tasks: map[string]registry.IndexEntry{
			"golang/test": {
				Module:      "github.com/test/tasks/golang/test@v0",
				Description: "Run Go tests",
				Tags:        []string{"golang", "testing"},
			},
		},
	}

	results := SearchIndex(index, "golang")

	// Should have 3 results
	if len(results) != 3 {
		t.Fatalf("SearchIndex() returned %d results, want 3", len(results))
	}

	// Results should be ordered by score (descending), then category, then name
	// All have "golang" in name and module, so scores should be similar
	// Check that categories are in order (roles before tasks)
	categoryOrder := make([]string, len(results))
	for i, r := range results {
		categoryOrder[i] = r.Category
	}

	// At least verify that results are grouped by category
	var seenTasks bool
	for _, cat := range categoryOrder {
		if cat == "tasks" {
			seenTasks = true
		}
		if seenTasks && cat == "roles" {
			t.Error("SearchIndex() results not properly ordered: tasks before roles")
		}
	}
}

// TestSearchCategoryEntries tests the exported SearchCategoryEntries function.
func TestSearchCategoryEntries(t *testing.T) {
	t.Parallel()

	entries := map[string]registry.IndexEntry{
		"golang/assistant": {
			Module:      "github.com/test/roles/golang/assistant@v0",
			Description: "Go programming expert",
			Tags:        []string{"golang", "programming"},
		},
		"python/assistant": {
			Module:      "github.com/test/roles/python/assistant@v0",
			Description: "Python programming expert",
			Tags:        []string{"python", "programming"},
		},
		"rust/assistant": {
			Module:      "github.com/test/roles/rust/assistant@v0",
			Description: "Rust programming expert",
			Tags:        []string{"rust", "systems"},
		},
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantFirst string // expected first result name
	}{
		{
			name:      "find golang",
			query:     "golang",
			wantCount: 1,
			wantFirst: "golang/assistant",
		},
		{
			name:      "find programming matches multiple",
			query:     "programming",
			wantCount: 3, // golang, python, rust all have "programming" in description
		},
		{
			name:      "no match",
			query:     "javascript",
			wantCount: 0,
		},
		{
			name:      "results sorted by score then name",
			query:     "assistant",
			wantCount: 3,
			wantFirst: "golang/assistant", // all same score, alphabetical
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SearchCategoryEntries("roles", entries, tt.query)

			if len(results) != tt.wantCount {
				t.Errorf("SearchCategoryEntries() returned %d results, want %d", len(results), tt.wantCount)
			}

			if tt.wantFirst != "" && len(results) > 0 {
				if results[0].Name != tt.wantFirst {
					t.Errorf("SearchCategoryEntries() first result = %q, want %q", results[0].Name, tt.wantFirst)
				}
			}

			// Verify all results have correct category
			for _, r := range results {
				if r.Category != "roles" {
					t.Errorf("result %q has category %q, want %q", r.Name, r.Category, "roles")
				}
			}
		})
	}
}

// TestSearchInstalledConfig tests searching installed CUE config values.
func TestSearchInstalledConfig(t *testing.T) {
	t.Parallel()

	cctx := cuecontext.New()
	cfg := cctx.CompileString(`{
		agents: {
			claude: {
				description: "Anthropic Claude AI assistant"
				tags: ["ai", "llm"]
				origin: "github.com/test/agents/claude@v0"
			}
			"gemini-non-interactive": {
				description: "Google Gemini non-interactive mode"
				tags: ["ai", "google"]
			}
		}
		roles: {
			"golang/assistant": {
				description: "Go programming expert"
				tags: ["golang", "programming"]
			}
		}
	}`)

	tests := []struct {
		name      string
		cueKey    string
		category  string
		query     string
		wantCount int
		wantFirst string
	}{
		{
			name:      "find agent by name substring",
			cueKey:    "agents",
			category:  "agents",
			query:     "claude",
			wantCount: 1,
			wantFirst: "claude",
		},
		{
			name:      "find agent by tag",
			cueKey:    "agents",
			category:  "agents",
			query:     "ai",
			wantCount: 2,
		},
		{
			name:      "find agent by description",
			cueKey:    "agents",
			category:  "agents",
			query:     "google",
			wantCount: 1,
			wantFirst: "gemini-non-interactive",
		},
		{
			name:      "find role by name",
			cueKey:    "roles",
			category:  "roles",
			query:     "golang",
			wantCount: 1,
			wantFirst: "golang/assistant",
		},
		{
			name:      "no matches",
			cueKey:    "agents",
			category:  "agents",
			query:     "nonexistent",
			wantCount: 0,
		},
		{
			name:      "missing category returns nil",
			cueKey:    "tasks",
			category:  "tasks",
			query:     "anything",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SearchInstalledConfig(cfg, tt.cueKey, tt.category, tt.query)

			if len(results) != tt.wantCount {
				t.Errorf("SearchInstalledConfig() returned %d results, want %d", len(results), tt.wantCount)
			}

			if tt.wantFirst != "" && len(results) > 0 {
				if results[0].Name != tt.wantFirst {
					t.Errorf("SearchInstalledConfig() first result = %q, want %q", results[0].Name, tt.wantFirst)
				}
			}

			// Verify all results have correct category
			for _, r := range results {
				if r.Category != tt.category {
					t.Errorf("result %q has category %q, want %q", r.Name, r.Category, tt.category)
				}
			}
		})
	}
}

// TestExtractIndexEntryFromCUE tests field extraction from CUE values.
func TestExtractIndexEntryFromCUE(t *testing.T) {
	t.Parallel()

	cctx := cuecontext.New()

	tests := []struct {
		name            string
		cueStr          string
		wantDescription string
		wantTags        []string
		wantModule      string
	}{
		{
			name: "full entry",
			cueStr: `{
				description: "Go programming expert"
				tags: ["golang", "programming"]
				origin: "github.com/test/roles/golang@v0"
			}`,
			wantDescription: "Go programming expert",
			wantTags:        []string{"golang", "programming"},
			wantModule:      "github.com/test/roles/golang@v0",
		},
		{
			name: "description only",
			cueStr: `{
				description: "Simple entry"
			}`,
			wantDescription: "Simple entry",
			wantTags:        nil,
			wantModule:      "",
		},
		{
			name: "empty struct",
			cueStr: `{
				prompt: "some prompt"
			}`,
			wantDescription: "",
			wantTags:        nil,
			wantModule:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := cctx.CompileString(tt.cueStr)
			entry := extractIndexEntryFromCUE(v)

			if entry.Description != tt.wantDescription {
				t.Errorf("Description = %q, want %q", entry.Description, tt.wantDescription)
			}

			if len(entry.Tags) != len(tt.wantTags) {
				t.Errorf("Tags = %v, want %v", entry.Tags, tt.wantTags)
			} else {
				for i, tag := range entry.Tags {
					if tag != tt.wantTags[i] {
						t.Errorf("Tags[%d] = %q, want %q", i, tag, tt.wantTags[i])
					}
				}
			}

			if entry.Module != tt.wantModule {
				t.Errorf("Module = %q, want %q", entry.Module, tt.wantModule)
			}
		})
	}
}
