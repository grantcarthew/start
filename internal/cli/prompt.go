package cli

import (
	"fmt"

	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/spf13/cobra"
)

// addPromptCommand adds the prompt command to the parent command.
func addPromptCommand(parent *cobra.Command) {
	promptCmd := &cobra.Command{
		Use:     "prompt [text]",
		GroupID: "commands",
		Short:   "Launch AI agent with a custom prompt",
		Long: `Launch AI agent with a custom prompt and only required contexts.

The argument can be inline text or a file path (starting with ./, /, or ~).
Default contexts are excluded to keep the prompt focused.
Use -c default to include contexts configured with default: true.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runPrompt,
	}
	parent.AddCommand(promptCmd)
}

// runPrompt executes the prompt command.
func runPrompt(cmd *cobra.Command, args []string) error {
	customText := ""
	if len(args) > 0 {
		arg := args[0]
		// Check if argument is a file path (per DR-038)
		if orchestration.IsFilePath(arg) {
			content, err := orchestration.ReadFilePath(arg)
			if err != nil {
				return fmt.Errorf("reading prompt file %q: %w", arg, err)
			}
			customText = content
		} else {
			customText = arg
		}
	}

	flags := getFlags(cmd)

	// Per DR-014: required contexts only, no defaults
	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: false,
		Tags:            flags.Context,
	}

	return executeStart(cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd.InOrStdin(), flags, selection, customText)
}
