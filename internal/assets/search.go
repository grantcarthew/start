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

// ParseSearchTerms splits an input string into unique, lowercased search terms.
// It splits on both whitespace and commas, removes empty strings and duplicates.
// Returns nil if no valid terms remain.
func ParseSearchTerms(input string) []string {
	normalized := strings.ReplaceAll(input, ",", " ")
	parts := strings.Fields(normalized)

	seen := make(map[string]bool, len(parts))
	var terms []string
	for _, p := range parts {
		lower := strings.ToLower(p)
		if !seen[lower] {
			seen[lower] = true
			terms = append(terms, lower)
		}
	}
	return terms
}

// SearchIndex searches all categories in the index for matching entries.
// The query is split into terms (by whitespace and commas) and all terms
// must match for an entry to be included (AND semantics).
func SearchIndex(index *registry.Index, query string) []SearchResult {
	if index == nil {
		return nil
	}

	terms := ParseSearchTerms(query)
	if len(terms) == 0 {
		return nil
	}

	var results []SearchResult

	// Search each category
	results = append(results, searchCategory("agents", index.Agents, terms)...)
	results = append(results, searchCategory("roles", index.Roles, terms)...)
	results = append(results, searchCategory("tasks", index.Tasks, terms)...)
	results = append(results, searchCategory("contexts", index.Contexts, terms)...)

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
func searchCategory(category string, entries map[string]registry.IndexEntry, terms []string) []SearchResult {
	var results []SearchResult

	for name, entry := range entries {
		score := matchScoreTerms(name, entry, terms)
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

// matchScoreTerms calculates how well an entry matches ALL search terms.
// Returns 0 if any term fails to match at least one field (AND semantics).
// Score is the sum of per-term field scores.
// Field weights: name (3) > description (1) > tags (1)
func matchScoreTerms(name string, entry registry.IndexEntry, terms []string) int {
	if len(terms) == 0 {
		return 0
	}

	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(entry.Description)

	totalScore := 0
	for _, term := range terms {
		termScore := 0

		if strings.Contains(nameLower, term) {
			termScore += 3
		}
		if strings.Contains(descLower, term) {
			termScore += 1
		}
		for _, tag := range entry.Tags {
			if strings.Contains(strings.ToLower(tag), term) {
				termScore += 1
				break
			}
		}

		if termScore == 0 {
			return 0 // AND: every term must match something
		}
		totalScore += termScore
	}

	return totalScore
}

// SearchCategoryEntries searches a single category's registry entries with scoring.
// The query is split into terms and all must match (AND semantics).
// Returns results sorted by score descending, then by name ascending.
func SearchCategoryEntries(category string, entries map[string]registry.IndexEntry, query string) []SearchResult {
	terms := ParseSearchTerms(query)
	if len(terms) == 0 {
		return nil
	}
	results := searchCategory(category, entries, terms)

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
// description/tags into IndexEntry structs, applies scoring, and returns
// scored results. The query is split into terms with AND semantics.
func SearchInstalledConfig(cfg cue.Value, cueKey, category, query string) []SearchResult {
	catVal := cfg.LookupPath(cue.ParsePath(cueKey))
	if !catVal.Exists() {
		return nil
	}

	iter, err := catVal.Fields()
	if err != nil {
		return nil
	}

	terms := ParseSearchTerms(query)
	if len(terms) == 0 {
		return nil
	}

	var results []SearchResult

	for iter.Next() {
		name := iter.Selector().Unquoted()
		entry := extractIndexEntryFromCUE(iter.Value())
		score := matchScoreTerms(name, entry, terms)
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
