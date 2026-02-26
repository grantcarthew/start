package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/grantcarthew/start/internal/tui"
	"github.com/spf13/cobra"
)

// addAssetsInfoCommand adds the info subcommand to the assets command.
func addAssetsInfoCommand(parent *cobra.Command) {
	infoCmd := &cobra.Command{
		Use:   "info <query>...",
		Short: "Show asset details",
		Long: `Show detailed information about an asset.

Searches for the asset in the registry index and displays full details
including description, module path, tags, and installation status.
Multiple words are combined with AND logic.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runAssetsInfo,
	}

	parent.AddCommand(infoCmd)
}

// runAssetsInfo shows detailed information about an asset.
func runAssetsInfo(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")
	ctx := context.Background()
	flags := getFlags(cmd)

	// Create registry client
	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	// Fetch index
	prog := tui.NewProgress(cmd.ErrOrStderr(), flags.Quiet)
	defer prog.Done()

	prog.Update("Fetching index...")
	index, err := client.FetchIndex(ctx, resolveAssetsIndexPath())
	if err != nil {
		return fmt.Errorf("fetching index: %w", err)
	}
	prog.Done()

	// Search for matching assets
	results, err := assets.SearchIndex(index, query, nil)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return fmt.Errorf("no assets found matching %q", query)
	}

	// If multiple matches, show first one with a note
	selected := results[0]
	if len(results) > 1 && !flags.Quiet {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Showing first of %d matches. Use 'start assets search %s' to see all.\n\n", len(results), query)
	}

	// Check installation status
	installed, installedScope := checkIfInstalled(selected)

	// Print detailed info
	printAssetInfo(cmd.OutOrStdout(), selected, installed, installedScope, flags.Verbose)

	return nil
}

// checkIfInstalled checks if an asset is installed in the config.
func checkIfInstalled(asset assets.SearchResult) (bool, string) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return false, ""
	}

	if !paths.AnyExists() {
		return false, ""
	}

	// Load merged config
	dirs := paths.ForScope(config.ScopeMerged)
	loader := internalcue.NewLoader()
	cfg, err := loader.Load(dirs)
	if err != nil {
		return false, ""
	}

	// Load local config separately for scope detection
	var localCfg cue.Value
	if paths.LocalExists {
		if v, loadErr := loader.LoadSingle(paths.Local); loadErr == nil {
			localCfg = v
		}
	}

	// Check if asset exists in config
	installed := collectInstalledAssets(cfg.Value, paths, localCfg)
	for _, a := range installed {
		if a.Category == asset.Category && a.Name == asset.Name {
			return true, a.Scope
		}
	}

	return false, ""
}

// printAssetInfo prints detailed information about an asset.
func printAssetInfo(w io.Writer, asset assets.SearchResult, installed bool, scope string, verbose bool) {
	_, _ = fmt.Fprintln(w)
	_, _ = tui.CategoryColor(asset.Category).Fprint(w, asset.Category)
	_, _ = fmt.Fprintf(w, "/%s\n", asset.Name)
	printSeparator(w)

	_, _ = tui.ColorDim.Fprint(w, "Type:")
	_, _ = fmt.Fprintf(w, " %s\n", asset.Category)
	_, _ = tui.ColorDim.Fprint(w, "Module:")
	_, _ = fmt.Fprintf(w, " %s\n", asset.Entry.Module)

	if asset.Entry.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = tui.ColorDim.Fprint(w, "Description:")
		_, _ = fmt.Fprintf(w, " %s\n", asset.Entry.Description)
	}

	if len(asset.Entry.Tags) > 0 {
		_, _ = tui.ColorDim.Fprint(w, "Tags:")
		_, _ = fmt.Fprintf(w, " %s\n", strings.Join(asset.Entry.Tags, ", "))
	}

	_, _ = fmt.Fprintln(w)
	if installed {
		_, _ = tui.ColorInstalled.Fprint(w, "✓")
		_, _ = fmt.Fprintf(w, " Installed %s\n", tui.Annotate("%s", scope))
	} else {
		_, _ = fmt.Fprintln(w, "  Not installed")
	}

	if asset.Entry.Version != "" {
		_, _ = tui.ColorDim.Fprint(w, "Version:")
		_, _ = fmt.Fprintf(w, " %s\n", asset.Entry.Version)
	}

	printSeparator(w)

	if !installed {
		_, _ = fmt.Fprintf(w, "\nUse 'start assets add %s' to install.\n", asset.Name)
	}
}
