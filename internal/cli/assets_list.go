package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/grantcarthew/start/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

// NOTE(design): This file shares registry client creation, index fetching, and config
// loading patterns with assets_add.go, assets_search.go, assets_update.go, and
// assets_index.go. This duplication is accepted - each command uses the results
// differently and a shared helper would couple them for modest line savings.

// InstalledAsset represents an installed asset with version info.
type InstalledAsset struct {
	Category     string // "agents", "roles", "tasks", "contexts"
	Name         string
	InstalledVer string // Current installed version
	LatestVer    string // Latest available version
	UpdateAvail  bool   // True if update is available
	Scope        string // "global" or "local"
	Origin       string // Registry module path (for updates)
	ConfigFile   string // Path to config file containing this asset
}

// addAssetsListCommand adds the list subcommand to the assets command.
func addAssetsListCommand(parent *cobra.Command) {
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed assets",
		Long: `List installed registry assets with update status.

Shows all assets installed via the registry with their current version
and whether updates are available.`,
		Args: cobra.NoArgs,
		RunE: runAssetsList,
	}

	listCmd.Flags().String("type", "", "Filter by type (agents, roles, tasks, contexts)")

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
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No configuration found. Run 'start' to set up.")
		return nil
	}

	// Load merged config
	dirs := paths.ForScope(config.ScopeMerged)
	loader := internalcue.NewLoader()
	cfg, err := loader.Load(dirs)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Load local config separately for scope detection
	var localCfg cue.Value
	if paths.LocalExists {
		if v, loadErr := loader.LoadSingle(paths.Local); loadErr == nil {
			localCfg = v
		}
	}

	// Collect installed assets from config
	installed := collectInstalledAssets(cfg.Value, paths, localCfg)

	if len(installed) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No assets installed from registry.")
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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No %s installed from registry.\n", assetType)
			return nil
		}
	}

	// Check for updates if verbose
	flags := getFlags(cmd)
	if flags.Verbose {
		client, err := registry.NewClient()
		if err == nil {
			checkForUpdates(ctx, client, installed)
		}
	}

	// Print results
	printInstalledAssets(cmd.OutOrStdout(), installed, flags.Verbose)

	return nil
}

// collectInstalledAssets extracts installed assets from the config.
func collectInstalledAssets(v cue.Value, paths config.Paths, localCfg cue.Value) []InstalledAsset {
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
			assetVal := iter.Value()

			// Extract origin field (registry provenance)
			var origin string
			if originVal := assetVal.LookupPath(cue.ParsePath("origin")); originVal.Exists() {
				origin, _ = originVal.String()
			}

			// Only include assets with origin (from registry)
			if origin == "" {
				continue
			}

			installedVer := assets.VersionFromOrigin(origin)

			scope, configFile := determineScopeAndFile(localCfg, paths, cat, name)
			asset := InstalledAsset{
				Category:     cat,
				Name:         name,
				InstalledVer: installedVer,
				Scope:        scope,
				Origin:       origin,
				ConfigFile:   configFile,
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

// determineScopeAndFile determines whether an asset is from global or local config
// and returns the path to the config file.
func determineScopeAndFile(localCfg cue.Value, paths config.Paths, category, name string) (scope, configFile string) {
	configFileName := assetTypeToConfigFile(category)

	// Check local first (takes precedence)
	if paths.LocalExists && assets.AssetExists(localCfg, category, name) {
		return "local", filepath.Join(paths.Local, configFileName)
	}

	// Default to global. Assets from collectInstalledAssets came from CUE evaluation
	// of these same files, so this fallback is for informational display purposes.
	return "global", filepath.Join(paths.Global, configFileName)
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
			installed[i].UpdateAvail = semver.Compare(entry.Version, installed[i].InstalledVer) > 0
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
	_, _ = fmt.Fprintln(w, "Installed assets:")
	_, _ = fmt.Fprintln(w)

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

		_, _ = tui.CategoryColor(cat).Fprint(w, cat)
		_, _ = fmt.Fprintln(w, "/")
		for _, a := range assets {
			if verbose && a.LatestVer != "" {
				_, _ = fmt.Fprintf(w, "  %-25s ", a.Name)
				if a.UpdateAvail {
					_, _ = fmt.Fprint(w, tui.Annotate("update available: %s", a.LatestVer))
				} else {
					_, _ = fmt.Fprint(w, tui.Annotate("latest"))
				}
				_, _ = fmt.Fprintln(w)
			} else {
				scopeIndicator := ""
				if verbose {
					scopeIndicator = fmt.Sprintf(" [%s]", a.Scope)
				}
				if a.InstalledVer != "" {
					_, _ = fmt.Fprintf(w, "  %-25s %s%s\n", a.Name, tui.Annotate("%s", a.InstalledVer), scopeIndicator)
				} else {
					_, _ = fmt.Fprintf(w, "  %s%s\n", a.Name, scopeIndicator)
				}
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}

// Helper functions

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

// assetTypeToConfigFile returns the config file name for an asset type.
func assetTypeToConfigFile(category string) string {
	switch category {
	case "agents":
		return "agents.cue"
	case "roles":
		return "roles.cue"
	case "tasks":
		return "tasks.cue"
	case "contexts":
		return "contexts.cue"
	default:
		return "settings.cue"
	}
}
