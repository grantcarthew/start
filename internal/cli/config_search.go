package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/spf13/cobra"
)

// addConfigSearchCommand adds the search subcommand to the config command group.
func addConfigSearchCommand(parent *cobra.Command) {
	searchCmd := &cobra.Command{
		Use:     "search [query]...",
		Aliases: []string{"find"},
		Short:   "Search installed config for assets",
		Long: `Search local and global config for installed assets by keyword.

Searches asset names, descriptions, and tags. Multiple words are combined
with AND logic - all terms must match. Terms can be space-separated or
comma-separated. Total query must be at least 3 characters.
Terms support regex patterns (e.g. '^home', 'expert$', 'go.*review').
Results are grouped by scope (local, global) and category.

Use --local to search only project-local config (./.start/).
Use --tag to filter by tags. Tags can be used alone or combined with a query.

Use 'start search' to also include the asset registry in results.`,
		Args: cobra.MinimumNArgs(0),
		RunE: runConfigSearch,
	}
	searchCmd.Flags().StringSlice("tag", nil, "Filter by tags (comma-separated)")

	parent.AddCommand(searchCmd)
}

// runConfigSearch searches local and global config, excluding the registry.
func runConfigSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	tagFlags, _ := cmd.Flags().GetStringSlice("tag")
	tags := assets.ParseSearchTerms(strings.Join(tagFlags, ","))

	terms := assets.ParseSearchPatterns(query)
	if err := assets.ValidateSearchQuery(terms, tags); err != nil {
		return err
	}

	if len(terms) > 0 {
		if _, err := assets.CompileSearchTerms(terms); err != nil {
			return err
		}
	}

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	flags := getFlags(cmd)
	loader := internalcue.NewLoader()
	categories := []struct {
		cueKey   string
		category string
	}{
		{internalcue.KeyAgents, "agents"},
		{internalcue.KeyRoles, "roles"},
		{internalcue.KeyTasks, "tasks"},
		{internalcue.KeyContexts, "contexts"},
	}

	var sections []searchSection
	stderr := cmd.ErrOrStderr()

	// Search local config (.start/)
	if paths.LocalExists {
		cfg, err := loader.LoadSingle(paths.Local)
		if err != nil && !errors.Is(err, internalcue.ErrNoCUEFiles) {
			printWarning(stderr, "failed to load local config: %s", err)
		} else if err == nil {
			var results []assets.SearchResult
			for _, cat := range categories {
				catResults, err := assets.SearchInstalledConfig(cfg, cat.cueKey, cat.category, query, tags)
				if err != nil {
					return err
				}
				results = append(results, catResults...)
			}
			if len(results) > 0 {
				sections = append(sections, searchSection{
					Label:   "local",
					Path:    "./.start",
					Results: results,
				})
			}
		}
	}

	// Search global config unless --local flag is set
	if !flags.Local && paths.GlobalExists {
		cfg, err := loader.LoadSingle(paths.Global)
		if err != nil && !errors.Is(err, internalcue.ErrNoCUEFiles) {
			printWarning(stderr, "failed to load global config: %s", err)
		} else if err == nil {
			var results []assets.SearchResult
			for _, cat := range categories {
				catResults, err := assets.SearchInstalledConfig(cfg, cat.cueKey, cat.category, query, tags)
				if err != nil {
					return err
				}
				results = append(results, catResults...)
			}
			if len(results) > 0 {
				sections = append(sections, searchSection{
					Label:   "global",
					Path:    shortenHome(paths.Global),
					Results: results,
				})
			}
		}
	}

	displayQuery := query
	if displayQuery == "" && len(tags) > 0 {
		displayQuery = "--tag " + strings.Join(tags, ",")
	}

	if len(sections) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No matches found for %q\n", displayQuery)
		return nil
	}

	printSearchSections(cmd.OutOrStdout(), sections, flags.Verbose, nil)
	return nil
}
