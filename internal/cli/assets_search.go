package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/cache"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/grantcarthew/start/internal/tui"
	"github.com/spf13/cobra"
)

// NOTE(design): This file shares registry client creation, index fetching, and config
// loading patterns with assets_add.go, assets_list.go, assets_update.go, and
// assets_index.go. This duplication is accepted - each command uses the results
// differently and a shared helper would couple them for modest line savings.

// addAssetsSearchCommand adds the search subcommand to the assets command.
func addAssetsSearchCommand(parent *cobra.Command) {
	searchCmd := &cobra.Command{
		Use:     "search [query]...",
		Aliases: []string{"find"},
		Short:   "Search registry for assets",
		Long: `Search the asset registry index by keyword.

Searches asset names, descriptions, and tags. Multiple words are combined
with AND logic - all terms must match. Terms can be space-separated or
comma-separated. Total query must be at least 3 characters.
Terms support regex patterns (e.g. '^home', 'expert$', 'go.*review').
Results are grouped by type (agents, roles, contexts, tasks).

Use --tag to filter by tags. Tags can be used alone or combined with a query.

Use 'start search' to also include local and global config in results.`,
		Args: cobra.MinimumNArgs(0),
		RunE: runAssetsSearch,
	}
	searchCmd.Flags().StringSlice("tag", nil, "Filter by tags (comma-separated)")

	parent.AddCommand(searchCmd)
}

// runAssetsSearch searches the registry index for matching assets.
func runAssetsSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	tagFlags, _ := cmd.Flags().GetStringSlice("tag")
	tags := assets.ParseSearchTerms(strings.Join(tagFlags, ","))

	terms := assets.ParseSearchPatterns(query)
	if err := assets.ValidateSearchQuery(terms, tags); err != nil {
		w := cmd.OutOrStdout()
		stdin := cmd.InOrStdin()
		if !isTerminal(stdin) {
			return err
		}
		if query != "" {
			_, _ = fmt.Fprintln(w, "Query must be at least 3 characters")
		}
		input, promptErr := promptSearchQuery(w, stdin)
		if promptErr != nil {
			return promptErr
		}
		if input == "" {
			return nil
		}
		query = input
		terms = assets.ParseSearchPatterns(query)
	}

	ctx := context.Background()
	flags := getFlags(cmd)

	// Fetch index from registry
	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	prog := tui.NewProgress(cmd.ErrOrStderr(), flags.Quiet)
	defer prog.Done()

	prog.Update("Fetching index...")
	index, indexVersion, err := client.FetchIndex(ctx, resolveAssetsIndexPath())
	if err != nil {
		return fmt.Errorf("fetching index: %w", err)
	}
	_ = cache.WriteIndex(indexVersion)
	prog.Done()

	// Search index
	results, err := assets.SearchIndex(index, query, tags)
	if err != nil {
		return err
	}

	displayQuery := query
	if displayQuery == "" && len(tags) > 0 {
		displayQuery = "--tag " + strings.Join(tags, ",")
	}

	if len(results) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No matches found for %q\n", displayQuery)
		return nil
	}

	// Collect installed asset names for marking in output
	installed := collectInstalledNames()

	// Print results
	printSearchResults(cmd.OutOrStdout(), results, flags.Verbose, installed)

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

	var localCfg cue.Value
	if paths.LocalExists {
		if v, loadErr := loader.LoadSingle(paths.Local); loadErr == nil {
			localCfg = v
		}
	}

	installedAssets := collectInstalledAssets(cfg.Value, paths, localCfg)
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
	categories := []string{"agents", "roles", "contexts", "tasks"}
	for _, cat := range categories {
		catResults := grouped[cat]
		if len(catResults) == 0 {
			continue
		}

		_, _ = tui.CategoryColor(cat).Fprint(w, cat)
		_, _ = fmt.Fprintln(w, "/")
		for _, r := range catResults {
			marker := "  "
			if installed[r.Category+"/"+r.Name] {
				marker = tui.ColorInstalled.Sprint("★") + " "
			}

			if verbose {
				_, _ = fmt.Fprintf(w, "  %s%-25s %s\n", marker, r.Name, tui.ColorDim.Sprint(r.Entry.Description))
				_, _ = fmt.Fprintf(w, "      Module: %s\n", tui.ColorDim.Sprint(r.Entry.Module))
				if len(r.Entry.Tags) > 0 {
					_, _ = fmt.Fprintf(w, "      Tags: %s\n", tui.ColorDim.Sprint(strings.Join(r.Entry.Tags, ", ")))
				}
			} else {
				_, _ = fmt.Fprintf(w, "  %s%-25s %s\n", marker, r.Name, tui.ColorDim.Sprint(r.Entry.Description))
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}
