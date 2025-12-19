package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

// Version information set via ldflags at build time
var (
	cliVersion = "dev"
	repoURL    = "https://github.com/grantcarthew/start"
)

// versionTemplate is the custom version output format per DR-033
var versionTemplate = fmt.Sprintf(`start version %s
%s
%s/issues/new
`, cliVersion, repoURL, repoURL)

// NewRootCmd creates a new root command instance with all subcommands attached.
// This factory function ensures tests get isolated command instances.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "AI agent CLI orchestrator",
		Long: `start is a command-line orchestrator for AI agents built on CUE.
It manages prompt composition, context injection, and workflow automation.`,
		Version: cliVersion,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Debug implies verbose
			if flagDebug {
				flagVerbose = true
			}
			// Validate and resolve directory flag
			if flagDirectory != "" {
				dir, err := resolveDirectory(flagDirectory)
				if err != nil {
					return err
				}
				flagDirectory = dir
			}
			return nil
		},
	}

	// Custom version template
	cmd.SetVersionTemplate(versionTemplate)

	// Add persistent flags
	cmd.PersistentFlags().StringVarP(&flagAgent, "agent", "a", "", "Override agent selection")
	cmd.PersistentFlags().StringVarP(&flagRole, "role", "r", "", "Override role (system prompt)")
	cmd.PersistentFlags().StringVarP(&flagModel, "model", "m", "", "Override model selection")
	cmd.PersistentFlags().StringSliceVarP(&flagContext, "context", "c", nil, "Select contexts by tag")
	cmd.PersistentFlags().StringVarP(&flagDirectory, "directory", "d", "", "Working directory for context detection")
	cmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Preview execution without launching agent")
	cmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress output")
	cmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Detailed output")
	cmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Debug output (implies --verbose)")

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

// resolveDirectory expands and validates the directory path.
func resolveDirectory(path string) (string, error) {
	// Expand tilde
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expanding ~: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Convert to absolute path
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	// Verify directory exists
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory not found: %s", abs)
		}
		return "", fmt.Errorf("accessing directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}

	return abs, nil
}
