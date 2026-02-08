package assets

import (
	"sort"
	"strings"

	"cuelang.org/go/cue"
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

// SearchCategoryEntries searches a single category's registry entries with scoring.
// Returns results sorted by score descending, then by name ascending.
func SearchCategoryEntries(category string, entries map[string]registry.IndexEntry, query string) []SearchResult {
	queryLower := strings.ToLower(query)
	results := searchCategory(category, entries, queryLower)

	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchScore != results[j].MatchScore {
			return results[i].MatchScore > results[j].MatchScore
		}
		return results[i].Name < results[j].Name
	})

	return results
}

// SearchInstalledConfig searches installed CUE config entries for a category.
// It iterates entries under the given CUE key (e.g. "agents"), extracts
// description/tags into IndexEntry structs, applies matchScore, and returns
// scored results.
func SearchInstalledConfig(cfg cue.Value, cueKey, category, query string) []SearchResult {
	catVal := cfg.LookupPath(cue.ParsePath(cueKey))
	if !catVal.Exists() {
		return nil
	}

	iter, err := catVal.Fields()
	if err != nil {
		return nil
	}

	queryLower := strings.ToLower(query)
	var results []SearchResult

	for iter.Next() {
		name := iter.Selector().Unquoted()
		entry := extractIndexEntryFromCUE(iter.Value())
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

	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchScore != results[j].MatchScore {
			return results[i].MatchScore > results[j].MatchScore
		}
		return results[i].Name < results[j].Name
	})

	return results
}

// extractIndexEntryFromCUE extracts description, tags, and origin from a CUE value
// into an IndexEntry for scoring.
func extractIndexEntryFromCUE(v cue.Value) registry.IndexEntry {
	var entry registry.IndexEntry

	if desc := v.LookupPath(cue.ParsePath("description")); desc.Exists() {
		entry.Description, _ = desc.String()
	}

	if tags := v.LookupPath(cue.ParsePath("tags")); tags.Exists() {
		tagIter, err := tags.List()
		if err == nil {
			for tagIter.Next() {
				if s, err := tagIter.Value().String(); err == nil {
					entry.Tags = append(entry.Tags, s)
				}
			}
		}
	}

	if origin := v.LookupPath(cue.ParsePath("origin")); origin.Exists() {
		entry.Module, _ = origin.String()
	}

	return entry
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
