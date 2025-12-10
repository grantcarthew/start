package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "start",
	Short: "AI agent CLI orchestrator",
	Long: `start is a command-line orchestrator for AI agents built on CUE.
It manages prompt composition, context injection, and workflow automation.`,
}

func Execute() error {
	return rootCmd.Execute()
}
