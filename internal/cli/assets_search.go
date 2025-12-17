package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// SearchResult holds a matched index entry with its category and name.
type SearchResult struct {
	Category    string // "agents", "roles", "tasks", "contexts"
	Name        string
	Entry       registry.IndexEntry
	MatchScore  int // Higher = better match
}

// addAssetsSearchCommand adds the search subcommand to the assets command.
func addAssetsSearchCommand(parent *cobra.Command) {
	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search registry for assets",
		Long: `Search the asset registry index by keyword.

Searches asset names, descriptions, and tags. Query must be at least 3 characters.
Results are grouped by type (agents, roles, tasks, contexts).`,
		Args: cobra.ExactArgs(1),
		RunE: runAssetsSearch,
	}

	searchCmd.Flags().BoolVarP(&assetsVerbose, "verbose", "v", false, "Show tags and module paths")

	parent.AddCommand(searchCmd)
}

// runAssetsSearch searches the registry index for matching assets.
func runAssetsSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	if len(query) < 3 {
		return fmt.Errorf("query must be at least 3 characters")
	}

	ctx := context.Background()

	// Fetch index from registry
	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	index, err := client.FetchIndex(ctx)
	if err != nil {
		return fmt.Errorf("fetching index: %w", err)
	}

	// Search index
	results := searchIndex(index, query)

	if len(results) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No matches found for %q\n", query)
		return nil
	}

	// Print results
	printSearchResults(cmd.OutOrStdout(), results, assetsVerbose)

	return nil
}

// searchIndex searches all categories in the index for matching entries.
func searchIndex(index *registry.Index, query string) []SearchResult {
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

// printSearchResults prints search results grouped by category.
func printSearchResults(w io.Writer, results []SearchResult, verbose bool) {
	fmt.Fprintf(w, "Found %d matches:\n\n", len(results))

	// Group by category for display
	grouped := make(map[string][]SearchResult)
	for _, r := range results {
		grouped[r.Category] = append(grouped[r.Category], r)
	}

	// Print in category order
	categories := []string{"agents", "roles", "tasks", "contexts"}
	for _, cat := range categories {
		catResults := grouped[cat]
		if len(catResults) == 0 {
			continue
		}

		fmt.Fprintf(w, "%s/\n", cat)
		for _, r := range catResults {
			if verbose {
				fmt.Fprintf(w, "  %-25s %s\n", r.Name, r.Entry.Description)
				fmt.Fprintf(w, "    Module: %s\n", r.Entry.Module)
				if len(r.Entry.Tags) > 0 {
					fmt.Fprintf(w, "    Tags: %s\n", strings.Join(r.Entry.Tags, ", "))
				}
			} else {
				fmt.Fprintf(w, "  %-25s %s\n", r.Name, r.Entry.Description)
			}
		}
		fmt.Fprintln(w)
	}
}
