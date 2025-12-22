package cli

import (
	"github.com/spf13/cobra"
)

// addConfigCommand adds the config command group and its subcommands to the parent.
func addConfigCommand(parent *cobra.Command) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage start configuration",
		Long: `Manage configuration for agents, roles, contexts, and tasks.

Configuration can be stored globally (~/.config/start/) or locally (./.start/).
Use --local to target project-specific configuration.`,
	}

	// Add persistent flags to config command (applies to all subcommands)
	configCmd.PersistentFlags().Bool("local", false, "Target local config (./.start/) instead of global")

	// Add entity subcommand groups
	addConfigAgentCommand(configCmd)
	addConfigRoleCommand(configCmd)
	addConfigContextCommand(configCmd)
	addConfigTaskCommand(configCmd)
	addConfigSettingsCommand(configCmd)

	parent.AddCommand(configCmd)
}
