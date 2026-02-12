package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// addAssetsSearchCommand adds the search subcommand to the assets command.
func addAssetsSearchCommand(parent *cobra.Command) {
	searchCmd := &cobra.Command{
		Use:     "search <query>...",
		Aliases: []string{"find"},
		Short:   "Search registry for assets",
		Long: `Search the asset registry index by keyword.

Searches asset names, descriptions, and tags. Multiple words are combined
with AND logic - all terms must match. Terms can be space-separated or
comma-separated. Total query must be at least 3 characters.
Terms support regex patterns (e.g. ^home, expert$, go.*review).
Results are grouped by type (agents, roles, tasks, contexts).`,
		Args: cobra.MinimumNArgs(1),
		RunE: runAssetsSearch,
	}

	searchCmd.Flags().BoolP("verbose", "v", false, "Show tags and module paths")

	parent.AddCommand(searchCmd)
}

// runAssetsSearch searches the registry index for matching assets.
func runAssetsSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	terms := assets.ParseSearchPatterns(query)
	totalLen := 0
	for _, t := range terms {
		totalLen += len(t)
	}
	if totalLen < 3 {
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
	results, err := assets.SearchIndex(index, query)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No matches found for %q\n", query)
		return nil
	}

	// Collect installed asset names for marking in output
	installed := collectInstalledNames()

	// Print results
	verbose, _ := cmd.Flags().GetBool("verbose")
	printSearchResults(cmd.OutOrStdout(), results, verbose, installed)

	return nil
}

// collectInstalledNames returns a set of "category/name" keys for installed assets.
func collectInstalledNames() map[string]bool {
	paths, err := config.ResolvePaths("")
	if err != nil || !paths.AnyExists() {
		return nil
	}

	dirs := paths.ForScope(config.ScopeMerged)
	loader := internalcue.NewLoader()
	cfg, err := loader.Load(dirs)
	if err != nil {
		return nil
	}

	installedAssets := collectInstalledAssets(cfg.Value, paths)
	names := make(map[string]bool, len(installedAssets))
	for _, a := range installedAssets {
		names[a.Category+"/"+a.Name] = true
	}
	return names
}

// printSearchResults prints search results grouped by category.
// installed is an optional set of "category/name" keys for marking installed assets.
func printSearchResults(w io.Writer, results []assets.SearchResult, verbose bool, installed map[string]bool) {
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

		_, _ = categoryColor(cat).Fprint(w, cat)
		_, _ = fmt.Fprintln(w, "/")
		for _, r := range catResults {
			marker := ""
			if installed[r.Category+"/"+r.Name] {
				marker = " " + colorInstalled.Sprint("*")
			}

			if verbose {
				_, _ = fmt.Fprintf(w, "  %-25s %s%s\n", r.Name, colorDim.Sprint(r.Entry.Description), marker)
				_, _ = fmt.Fprintf(w, "    Module: %s\n", colorDim.Sprint(r.Entry.Module))
				if len(r.Entry.Tags) > 0 {
					_, _ = fmt.Fprintf(w, "    Tags: %s\n", colorDim.Sprint(strings.Join(r.Entry.Tags, ", ")))
				}
			} else {
				_, _ = fmt.Fprintf(w, "  %-25s %s%s\n", r.Name, colorDim.Sprint(r.Entry.Description), marker)
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}
