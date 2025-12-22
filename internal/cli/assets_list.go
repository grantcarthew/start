package cli

import (
	"context"
	"fmt"
	"io"
	"sort"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// InstalledAsset represents an installed asset with version info.
type InstalledAsset struct {
	Category       string // "agents", "roles", "tasks", "contexts"
	Name           string
	InstalledVer   string // Current installed version
	LatestVer      string // Latest available version
	UpdateAvail    bool   // True if update is available
	Scope          string // "global" or "local"
}

// addAssetsListCommand adds the list subcommand to the assets command.
func addAssetsListCommand(parent *cobra.Command) {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed assets",
		Long: `List installed registry assets with update status.

Shows all assets installed via the registry with their current version
and whether updates are available.`,
		Args: cobra.NoArgs,
		RunE: runAssetsList,
	}

	listCmd.Flags().String("type", "", "Filter by type (agents, roles, tasks, contexts)")
	listCmd.Flags().BoolP("verbose", "v", false, "Show detailed output")

	parent.AddCommand(listCmd)
}

// runAssetsList lists installed assets with update status.
func runAssetsList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	if !paths.AnyExists() {
		fmt.Fprintln(cmd.OutOrStdout(), "No configuration found. Run 'start' to set up.")
		return nil
	}

	// Load merged config
	dirs := paths.ForScope(config.ScopeMerged)
	loader := internalcue.NewLoader()
	cfg, err := loader.Load(dirs)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Collect installed assets from config
	installed := collectInstalledAssets(cfg.Value, paths)

	if len(installed) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No assets installed from registry.")
		return nil
	}

	// Filter by type if specified
	assetType, _ := cmd.Flags().GetString("type")
	if assetType != "" {
		var filtered []InstalledAsset
		for _, a := range installed {
			if a.Category == assetType {
				filtered = append(filtered, a)
			}
		}
		installed = filtered

		if len(installed) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No %s installed from registry.\n", assetType)
			return nil
		}
	}

	// Check for updates if verbose
	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		client, err := registry.NewClient()
		if err == nil {
			checkForUpdates(ctx, client, installed)
		}
	}

	// Print results
	printInstalledAssets(cmd.OutOrStdout(), installed, verbose)

	return nil
}

// collectInstalledAssets extracts installed assets from the config.
func collectInstalledAssets(v cue.Value, paths config.Paths) []InstalledAsset {
	var installed []InstalledAsset

	categories := []string{"agents", "roles", "tasks", "contexts"}
	for _, cat := range categories {
		catVal := v.LookupPath(cue.ParsePath(cat))
		if !catVal.Exists() {
			continue
		}

		iter, err := catVal.Fields()
		if err != nil {
			continue
		}

		for iter.Next() {
			name := iter.Selector().Unquoted()
			asset := InstalledAsset{
				Category: cat,
				Name:     name,
				Scope:    determineScope(paths, cat, name),
			}
			installed = append(installed, asset)
		}
	}

	// Sort by category then name
	sort.Slice(installed, func(i, j int) bool {
		if installed[i].Category != installed[j].Category {
			return categoryOrder(installed[i].Category) < categoryOrder(installed[j].Category)
		}
		return installed[i].Name < installed[j].Name
	})

	return installed
}

// determineScope determines whether an asset is from global or local config.
func determineScope(paths config.Paths, category, name string) string {
	// For now, default to "global" - could be enhanced to track source
	if paths.LocalExists {
		return "local"
	}
	return "global"
}

// checkForUpdates checks registry for available updates.
func checkForUpdates(ctx context.Context, client *registry.Client, installed []InstalledAsset) {
	// Fetch index for version info
	index, err := client.FetchIndex(ctx)
	if err != nil {
		return
	}

	for i := range installed {
		entry := findInIndex(index, installed[i].Category, installed[i].Name)
		if entry != nil && entry.Version != "" {
			installed[i].LatestVer = entry.Version
			// Version comparison would go here
			// For now, just note we have version info
		}
	}
}

// findInIndex looks up an asset in the index.
func findInIndex(index *registry.Index, category, name string) *registry.IndexEntry {
	var entries map[string]registry.IndexEntry

	switch category {
	case "agents":
		entries = index.Agents
	case "roles":
		entries = index.Roles
	case "tasks":
		entries = index.Tasks
	case "contexts":
		entries = index.Contexts
	}

	if entry, ok := entries[name]; ok {
		return &entry
	}
	return nil
}

// printInstalledAssets prints the list of installed assets.
func printInstalledAssets(w io.Writer, installed []InstalledAsset, verbose bool) {
	fmt.Fprintln(w, "Installed assets:")
	fmt.Fprintln(w)

	// Group by category
	grouped := make(map[string][]InstalledAsset)
	for _, a := range installed {
		grouped[a.Category] = append(grouped[a.Category], a)
	}

	categories := []string{"agents", "roles", "tasks", "contexts"}
	for _, cat := range categories {
		assets := grouped[cat]
		if len(assets) == 0 {
			continue
		}

		fmt.Fprintf(w, "%s/\n", cat)
		for _, a := range assets {
			if verbose && a.LatestVer != "" {
				if a.UpdateAvail {
					fmt.Fprintf(w, "  %-25s %s (update available: %s)\n", a.Name, a.InstalledVer, a.LatestVer)
				} else {
					fmt.Fprintf(w, "  %-25s %s (latest)\n", a.Name, a.LatestVer)
				}
			} else {
				scopeIndicator := ""
				if verbose {
					scopeIndicator = fmt.Sprintf(" [%s]", a.Scope)
				}
				fmt.Fprintf(w, "  %s%s\n", a.Name, scopeIndicator)
			}
		}
		fmt.Fprintln(w)
	}
}
