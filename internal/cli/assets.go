package cli

import (
	"github.com/spf13/cobra"
)

// Asset repository constants
const (
	// DefaultAssetRepoURL is the default GitHub repository for browsing assets.
	DefaultAssetRepoURL = "https://github.com/grantcarthew/start-assets"
)

// Assets command flags
var (
	assetsLocal   bool   // --local flag for add command
	assetsVerbose bool   // --verbose flag for search, info, list
	assetsDryRun  bool   // --dry-run flag for update
	assetsForce   bool   // --force flag for update
	assetsType    string // --type flag for list
)

// addAssetsCommand adds the assets command group and its subcommands to the parent.
func addAssetsCommand(parent *cobra.Command) {
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage assets from the CUE registry",
		Long: `Manage assets (roles, tasks, contexts, agents) from the CUE Central Registry.

Assets are CUE modules that define reusable AI agent configurations.
Use these commands to discover, install, and update assets.`,
	}

	// Add subcommands
	addAssetsBrowseCommand(assetsCmd)
	addAssetsSearchCommand(assetsCmd)
	addAssetsAddCommand(assetsCmd)
	addAssetsListCommand(assetsCmd)
	addAssetsInfoCommand(assetsCmd)
	addAssetsUpdateCommand(assetsCmd)
	addAssetsIndexCommand(assetsCmd)

	parent.AddCommand(assetsCmd)
}
