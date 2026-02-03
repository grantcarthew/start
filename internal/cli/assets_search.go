package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// addAssetsSearchCommand adds the search subcommand to the assets command.
func addAssetsSearchCommand(parent *cobra.Command) {
	searchCmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"find"},
		Short:   "Search registry for assets",
		Long: `Search the asset registry index by keyword.

Searches asset names, descriptions, and tags. Query must be at least 3 characters.
Results are grouped by type (agents, roles, tasks, contexts).`,
		Args: cobra.ExactArgs(1),
		RunE: runAssetsSearch,
	}

	searchCmd.Flags().BoolP("verbose", "v", false, "Show tags and module paths")

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
	results := assets.SearchIndex(index, query)

	if len(results) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No matches found for %q\n", query)
		return nil
	}

	// Print results
	verbose, _ := cmd.Flags().GetBool("verbose")
	printSearchResults(cmd.OutOrStdout(), results, verbose)

	return nil
}

// printSearchResults prints search results grouped by category.
func printSearchResults(w io.Writer, results []assets.SearchResult, verbose bool) {
	_, _ = fmt.Fprintf(w, "Found %d matches:\n\n", len(results))

	// Group by category for display
	grouped := make(map[string][]assets.SearchResult)
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

		_, _ = fmt.Fprintf(w, "%s/\n", cat)
		for _, r := range catResults {
			if verbose {
				_, _ = fmt.Fprintf(w, "  %-25s %s\n", r.Name, r.Entry.Description)
				_, _ = fmt.Fprintf(w, "    Module: %s\n", r.Entry.Module)
				if len(r.Entry.Tags) > 0 {
					_, _ = fmt.Fprintf(w, "    Tags: %s\n", strings.Join(r.Entry.Tags, ", "))
				}
			} else {
				_, _ = fmt.Fprintf(w, "  %-25s %s\n", r.Name, r.Entry.Description)
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}
