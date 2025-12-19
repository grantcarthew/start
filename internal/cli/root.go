package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// NewRootCmd creates a new root command instance with all subcommands attached.
// This factory function ensures tests get isolated command instances.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "AI agent CLI orchestrator",
		Long: `start is a command-line orchestrator for AI agents built on CUE.
It manages prompt composition, context injection, and workflow automation.`,
	}

	// Add persistent flags
	cmd.PersistentFlags().StringVarP(&flagAgent, "agent", "a", "", "Override agent selection")
	cmd.PersistentFlags().StringVarP(&flagRole, "role", "r", "", "Override role (system prompt)")
	cmd.PersistentFlags().StringVarP(&flagModel, "model", "m", "", "Override model selection")
	cmd.PersistentFlags().StringSliceVarP(&flagContext, "context", "c", nil, "Select contexts by tag")
	cmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Preview execution without launching agent")
	cmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress output")
	cmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Detailed output")

	// Set RunE on root command for `start` execution
	cmd.RunE = runStart

	// Add subcommands
	addShowCommand(cmd)
	addPromptCommand(cmd)
	addTaskCommand(cmd)
	addAssetsCommand(cmd)
	addConfigCommand(cmd)
	addDoctorCommand(cmd)
	addCompletionCommand(cmd)

	return cmd
}

// Execute runs the root command. This is the main entry point for the CLI.
func Execute() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("start does not support Windows (see DR-006 for platform scope)")
	}
	return NewRootCmd().Execute()
}
