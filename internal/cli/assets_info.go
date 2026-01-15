package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// addAssetsInfoCommand adds the info subcommand to the assets command.
func addAssetsInfoCommand(parent *cobra.Command) {
	infoCmd := &cobra.Command{
		Use:   "info <query>",
		Short: "Show asset details",
		Long: `Show detailed information about an asset.

Searches for the asset in the registry index and displays full details
including description, module path, tags, and installation status.`,
		Args: cobra.ExactArgs(1),
		RunE: runAssetsInfo,
	}

	infoCmd.Flags().BoolP("verbose", "v", false, "Show additional details")

	parent.AddCommand(infoCmd)
}

// runAssetsInfo shows detailed information about an asset.
func runAssetsInfo(cmd *cobra.Command, args []string) error {
	query := args[0]
	ctx := context.Background()
	flags := getFlags(cmd)

	// Create registry client
	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	// Fetch index
	if !flags.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Fetching index...")
	}
	index, err := client.FetchIndex(ctx)
	if err != nil {
		return fmt.Errorf("fetching index: %w", err)
	}

	// Search for matching assets
	results := searchIndex(index, query)

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
	verbose, _ := cmd.Flags().GetBool("verbose")
	printAssetInfo(cmd.OutOrStdout(), selected, installed, installedScope, verbose)

	return nil
}

// checkIfInstalled checks if an asset is installed in the config.
func checkIfInstalled(asset SearchResult) (bool, string) {
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

	// Check if asset exists in config
	installed := collectInstalledAssets(cfg.Value, paths)
	for _, a := range installed {
		if a.Category == asset.Category && a.Name == asset.Name {
			return true, a.Scope
		}
	}

	return false, ""
}

// printAssetInfo prints detailed information about an asset.
func printAssetInfo(w io.Writer, asset SearchResult, installed bool, scope string, verbose bool) {
	_, _ = fmt.Fprintf(w, "Asset: %s/%s\n", asset.Category, asset.Name)
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 60))

	_, _ = fmt.Fprintf(w, "Type: %s\n", asset.Category)
	_, _ = fmt.Fprintf(w, "Module: %s\n", asset.Entry.Module)
	_, _ = fmt.Fprintln(w)

	if asset.Entry.Description != "" {
		_, _ = fmt.Fprintln(w, "Description:")
		_, _ = fmt.Fprintf(w, "  %s\n", asset.Entry.Description)
		_, _ = fmt.Fprintln(w)
	}

	if len(asset.Entry.Tags) > 0 {
		_, _ = fmt.Fprintf(w, "Tags: %s\n", strings.Join(asset.Entry.Tags, ", "))
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintln(w, "Status:")
	if installed {
		_, _ = fmt.Fprintf(w, "  Installed: Yes (%s)\n", scope)
	} else {
		_, _ = fmt.Fprintln(w, "  Installed: No")
	}

	if asset.Entry.Version != "" {
		_, _ = fmt.Fprintf(w, "  Version: %s\n", asset.Entry.Version)
	}

	_, _ = fmt.Fprintln(w)

	if !installed {
		_, _ = fmt.Fprintf(w, "Use 'start assets add %s' to install.\n", asset.Name)
	}
}
