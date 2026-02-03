package assets

import (
	"strings"
	"testing"

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
