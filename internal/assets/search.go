package assets

import (
	"fmt"
	"regexp"
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
// Use ParseSearchPatterns instead when terms will be compiled as regex patterns.
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

// ParseSearchPatterns splits an input string into unique search patterns,
// preserving original case. Deduplication is case-insensitive but the
// first occurrence's casing is kept. This avoids corrupting case-sensitive
// regex escape sequences like \S, \D, \W, and \B.
// Returns nil if no valid patterns remain.
func ParseSearchPatterns(input string) []string {
	normalized := strings.ReplaceAll(input, ",", " ")
	parts := strings.Fields(normalized)

	seen := make(map[string]bool, len(parts))
	var patterns []string
	for _, p := range parts {
		lower := strings.ToLower(p)
		if !seen[lower] {
			seen[lower] = true
			patterns = append(patterns, p)
		}
	}
	return patterns
}

// CompileSearchTerms compiles search terms into case-insensitive regular expressions.
// Each term is treated as a regex pattern, allowing operators like ^ $ . + * etc.
// Returns an error if any term contains an invalid regex pattern.
func CompileSearchTerms(terms []string) ([]*regexp.Regexp, error) {
	patterns := make([]*regexp.Regexp, len(terms))
	for i, term := range terms {
		re, err := regexp.Compile("(?i)" + term)
		if err != nil {
			return nil, fmt.Errorf("invalid search pattern %q: %w", term, err)
		}
		patterns[i] = re
	}
	return patterns, nil
}

// SearchIndex searches all categories in the index for matching entries.
// The query is split into terms (by whitespace and commas) and all terms
// must match for an entry to be included (AND semantics).
// Terms are treated as regex patterns for flexible matching.
func SearchIndex(index *registry.Index, query string) ([]SearchResult, error) {
	if index == nil {
		return nil, nil
	}

	terms := ParseSearchPatterns(query)
	if len(terms) == 0 {
		return nil, nil
	}

	patterns, err := CompileSearchTerms(terms)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	// Search each category
	results = append(results, searchCategory("agents", index.Agents, patterns)...)
	results = append(results, searchCategory("roles", index.Roles, patterns)...)
	results = append(results, searchCategory("tasks", index.Tasks, patterns)...)
	results = append(results, searchCategory("contexts", index.Contexts, patterns)...)

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

	return results, nil
}

// searchCategory searches a single category map for matching entries.
func searchCategory(category string, entries map[string]registry.IndexEntry, patterns []*regexp.Regexp) []SearchResult {
	var results []SearchResult

	for name, entry := range entries {
		score := matchScorePatterns(name, entry, patterns)
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

// matchScorePatterns calculates how well an entry matches ALL search patterns.
// Returns 0 if any pattern fails to match at least one field (AND semantics).
// Score is the sum of per-pattern field scores.
// Field weights: name (3) > description (1) > tags (1)
// Patterns must be compiled with (?i) for case-insensitive matching.
func matchScorePatterns(name string, entry registry.IndexEntry, patterns []*regexp.Regexp) int {
	if len(patterns) == 0 {
		return 0
	}

	totalScore := 0
	for _, pattern := range patterns {
		termScore := 0

		if pattern.MatchString(name) {
			termScore += 3
		}
		if pattern.MatchString(entry.Description) {
			termScore += 1
		}
		for _, tag := range entry.Tags {
			if pattern.MatchString(tag) {
				termScore += 1
				break
			}
		}

		if termScore == 0 {
			return 0 // AND: every pattern must match something
		}
		totalScore += termScore
	}

	return totalScore
}

// SearchCategoryEntries searches a single category's registry entries with scoring.
// The query is split into terms and all must match (AND semantics).
// Terms are treated as regex patterns for flexible matching.
// Returns results sorted by score descending, then by name ascending.
func SearchCategoryEntries(category string, entries map[string]registry.IndexEntry, query string) ([]SearchResult, error) {
	terms := ParseSearchPatterns(query)
	if len(terms) == 0 {
		return nil, nil
	}

	patterns, err := CompileSearchTerms(terms)
	if err != nil {
		return nil, err
	}

	results := searchCategory(category, entries, patterns)

	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchScore != results[j].MatchScore {
			return results[i].MatchScore > results[j].MatchScore
		}
		return results[i].Name < results[j].Name
	})

	return results, nil
}

// SearchInstalledConfig searches installed CUE config entries for a category.
// It iterates entries under the given CUE key (e.g. "agents"), extracts
// description/tags into IndexEntry structs, applies scoring, and returns
// scored results. The query is split into terms with AND semantics.
// Terms are treated as regex patterns for flexible matching.
func SearchInstalledConfig(cfg cue.Value, cueKey, category, query string) ([]SearchResult, error) {
	catVal := cfg.LookupPath(cue.ParsePath(cueKey))
	if !catVal.Exists() {
		return nil, nil
	}

	iter, err := catVal.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating %s fields: %w", cueKey, err)
	}

	terms := ParseSearchPatterns(query)
	if len(terms) == 0 {
		return nil, nil
	}

	patterns, err := CompileSearchTerms(terms)
	if err != nil {
		return nil, err
	}

	var results []SearchResult

	for iter.Next() {
		name := iter.Selector().Unquoted()
		entry := extractIndexEntryFromCUE(iter.Value())
		score := matchScorePatterns(name, entry, patterns)
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

	return results, nil
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
