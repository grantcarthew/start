package assets

import (
	"sort"
	"strings"

	"github.com/grantcarthew/start/internal/registry"
)

// SearchResult holds a matched index entry with its category and name.
type SearchResult struct {
	Category   string // "agents", "roles", "tasks", "contexts"
	Name       string
	Entry      registry.IndexEntry
	MatchScore int // Higher = better match
}

// SearchIndex searches all categories in the index for matching entries.
func SearchIndex(index *registry.Index, query string) []SearchResult {
	if index == nil {
		return nil
	}

	var results []SearchResult
	queryLower := strings.ToLower(query)

	// Search each category
	results = append(results, searchCategory("agents", index.Agents, queryLower)...)
	results = append(results, searchCategory("roles", index.Roles, queryLower)...)
	results = append(results, searchCategory("tasks", index.Tasks, queryLower)...)
	results = append(results, searchCategory("contexts", index.Contexts, queryLower)...)

	// Sort by match score (descending), then by category, then by name
	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchScore != results[j].MatchScore {
			return results[i].MatchScore > results[j].MatchScore
		}
		if results[i].Category != results[j].Category {
			return categoryOrder(results[i].Category) < categoryOrder(results[j].Category)
		}
		return results[i].Name < results[j].Name
	})

	return results
}

// searchCategory searches a single category map for matching entries.
func searchCategory(category string, entries map[string]registry.IndexEntry, queryLower string) []SearchResult {
	var results []SearchResult

	for name, entry := range entries {
		score := matchScore(name, entry, queryLower)
		if score > 0 {
			results = append(results, SearchResult{
				Category:   category,
				Name:       name,
				Entry:      entry,
				MatchScore: score,
			})
		}
	}

	return results
}

// matchScore calculates how well an entry matches the query.
// Returns 0 if no match, higher values for better matches.
// Priority: name (3) > path (2) > description (1) > tags (1)
func matchScore(name string, entry registry.IndexEntry, queryLower string) int {
	score := 0
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(entry.Description)
	moduleLower := strings.ToLower(entry.Module)

	// Name match (highest priority)
	if strings.Contains(nameLower, queryLower) {
		score += 3
	}

	// Module path match
	if strings.Contains(moduleLower, queryLower) {
		score += 2
	}

	// Description match
	if strings.Contains(descLower, queryLower) {
		score += 1
	}

	// Tags match
	for _, tag := range entry.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			score += 1
			break // Only count tag match once
		}
	}

	return score
}

// categoryOrder returns the display order for a category.
func categoryOrder(category string) int {
	switch category {
	case "agents":
		return 0
	case "roles":
		return 1
	case "tasks":
		return 2
	case "contexts":
		return 3
	default:
		return 4
	}
}
