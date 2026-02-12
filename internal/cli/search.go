package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// searchSection groups search results under a labelled section.
type searchSection struct {
	Label         string // Section name (e.g. "local", "global", "registry")
	Path          string // Optional path shown in parentheses with cyan colour
	Results       []assets.SearchResult
	ShowInstalled bool // Only true for registry section
}

// addSearchCommand adds the top-level search command.
func addSearchCommand(parent *cobra.Command) {
	searchCmd := &cobra.Command{
		Use:   "search <query>...",
		Aliases: []string{"find"},
		Short: "Search configs and registry for assets",
		Long: `Search local config, global config, and the asset registry by keyword.

Searches asset names, descriptions, and tags. Multiple words are combined
with AND logic - all terms must match. Terms can be space-separated or
comma-separated. Total query must be at least 3 characters.
Terms support regex patterns (e.g. ^home, expert$, go.*review).
Results are grouped by source (local, global, registry) and category.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}

	searchCmd.Flags().BoolP("verbose", "v", false, "Show tags and module paths")

	parent.AddCommand(searchCmd)
}

// runSearch searches local config, global config, and the registry.
func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	terms := assets.ParseSearchPatterns(query)
	totalLen := 0
	for _, t := range terms {
		totalLen += len(t)
	}
	if totalLen < 3 {
		return fmt.Errorf("query must be at least 3 characters")
	}

	// Validate regex patterns before searching
	if _, err := assets.CompileSearchTerms(terms); err != nil {
		return err
	}

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

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

	// Search local config
	if paths.LocalExists {
		cfg, err := loader.LoadSingle(paths.Local)
		if err == nil {
			var results []assets.SearchResult
			for _, cat := range categories {
				catResults, err := assets.SearchInstalledConfig(cfg, cat.cueKey, cat.category, query)
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

	// Search global config
	if paths.GlobalExists {
		cfg, err := loader.LoadSingle(paths.Global)
		if err == nil {
			var results []assets.SearchResult
			for _, cat := range categories {
				catResults, err := assets.SearchInstalledConfig(cfg, cat.cueKey, cat.category, query)
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

	// Search registry (graceful fallback if unavailable)
	var registryErr error
	ctx := context.Background()
	client, err := registry.NewClient()
	if err != nil {
		registryErr = err
	} else {
		index, err := client.FetchIndex(ctx)
		if err != nil {
			registryErr = err
		} else {
			results, err := assets.SearchIndex(index, query)
			if err != nil {
				return err
			}
			if len(results) > 0 {
				sections = append(sections, searchSection{
					Label:         "registry",
					Results:       results,
					ShowInstalled: true,
				})
			}
		}
	}

	if len(sections) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No matches found for %q\n", query)
		if registryErr != nil {
			PrintWarning(cmd.ErrOrStderr(), "registry unavailable: %v\n", registryErr)
		}
		return nil
	}

	installed := collectInstalledNames()
	verbose, _ := cmd.Flags().GetBool("verbose")
	printSearchSections(cmd.OutOrStdout(), sections, verbose, installed)

	if registryErr != nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		PrintWarning(cmd.ErrOrStderr(), "registry unavailable: %v\n", registryErr)
	}

	return nil
}

// printSearchSections prints search results grouped by section and category.
func printSearchSections(w io.Writer, sections []searchSection, verbose bool, installed map[string]bool) {
	for i, section := range sections {
		if len(section.Results) == 0 {
			continue
		}

		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		if section.Path != "" {
			_, _ = fmt.Fprintf(w, "%s %s%s%s\n", section.Label, colorCyan.Sprint("("), colorDim.Sprint(section.Path), colorCyan.Sprint(")"))
		} else {
			_, _ = fmt.Fprintln(w, section.Label)
		}

		// Group results by category
		grouped := make(map[string][]assets.SearchResult)
		for _, r := range section.Results {
			grouped[r.Category] = append(grouped[r.Category], r)
		}

		// Print in category order
		categories := []string{"agents", "roles", "tasks", "contexts"}
		for _, cat := range categories {
			catResults := grouped[cat]
			if len(catResults) == 0 {
				continue
			}

			_, _ = fmt.Fprint(w, "  ")
			_, _ = categoryColor(cat).Fprint(w, cat)
			_, _ = fmt.Fprintln(w, "/")

			for _, r := range catResults {
				marker := ""
				if section.ShowInstalled && installed[r.Category+"/"+r.Name] {
					marker = " " + colorInstalled.Sprint("*")
				}

				_, _ = fmt.Fprintf(w, "    %-25s %s%s\n", r.Name, colorDim.Sprint(r.Entry.Description), marker)
				if verbose {
					if r.Entry.Module != "" {
						_, _ = fmt.Fprintf(w, "      Module: %s\n", colorDim.Sprint(r.Entry.Module))
					}
					if len(r.Entry.Tags) > 0 {
						_, _ = fmt.Fprintf(w, "      Tags: %s\n", colorDim.Sprint(strings.Join(r.Entry.Tags, ", ")))
					}
				}
			}
		}
	}
}

// shortenHome replaces the home directory prefix with ~.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
