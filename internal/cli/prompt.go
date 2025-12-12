package cli

import (
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt [text]",
	Short: "Launch AI agent with custom prompt",
	Long: `Launch AI agent with a custom prompt and only required contexts.

Default contexts are excluded to keep the prompt focused.
Use -c default to explicitly include default contexts.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPrompt,
}

func init() {
	rootCmd.AddCommand(promptCmd)
}

// runPrompt executes the prompt command.
func runPrompt(cmd *cobra.Command, args []string) error {
	customText := ""
	if len(args) > 0 {
		customText = args[0]
	}

	// Per DR-014: required contexts only, no defaults
	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: false,
		Tags:            flagContext,
	}

	return executeStart(cmd.OutOrStdout(), cmd.ErrOrStderr(), selection, customText)
}
