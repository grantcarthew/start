package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/fatih/color"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// IsSilentError returns true if the error should not be printed to stderr.
// Used by main.go to suppress output for errors that only set the exit code.
func IsSilentError(err error) bool {
	type silent interface {
		Silent() bool
	}
	if s, ok := err.(silent); ok {
		return s.Silent()
	}
	return false
}

// Build-time variables set via ldflags
var (
	cliVersion = "dev"
	commit     = "unknown"
	buildDate  = "unknown"
	repoURL    = "https://github.com/grantcarthew/start"
)

// versionTemplate is the custom version output format per DR-033
var versionTemplate = fmt.Sprintf(`start version %s
%s
%s/issues/new
`, cliVersion, repoURL, repoURL)

// NewRootCmd creates a new root command instance with all subcommands attached.
// This factory function ensures tests get isolated command instances with their own Flags.
func NewRootCmd() *cobra.Command {
	// Create flags scoped to this command instance
	flags := &Flags{}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "AI agent CLI orchestrator",
		Long: `start is a command-line orchestrator for AI agents built on CUE.
It manages prompt composition, context injection, and workflow automation.`,
		Version: cliVersion,
		// SilenceUsage prevents usage from being printed on RunE errors.
		// Usage is still shown for flag/argument parsing errors.
		SilenceUsage: true,
		// SilenceErrors prevents Cobra from printing errors - we handle them
		// ourselves in main.go with colored output.
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Store flags in context for access by all commands
			ctx := context.WithValue(cmd.Context(), flagsKey{}, flags)
			cmd.SetContext(ctx)

			// Apply --no-color flag to disable colors globally
			if flags.NoColor {
				color.NoColor = true
			}

			// Debug implies verbose
			if flags.Debug {
				flags.Verbose = true
			}
			// Validate and resolve directory flag
			if flags.Directory != "" {
				dir, err := resolveDirectory(flags.Directory)
				if err != nil {
					return err
				}
				flags.Directory = dir
			}
			return nil
		},
	}

	// Custom version template
	cmd.SetVersionTemplate(versionTemplate)

	// Add persistent flags bound to this instance's Flags struct
	cmd.PersistentFlags().StringVarP(&flags.Agent, "agent", "a", "", "Override agent selection")
	cmd.PersistentFlags().StringVarP(&flags.Role, "role", "r", "", "Override role (config name or file path)")
	cmd.PersistentFlags().StringVarP(&flags.Model, "model", "m", "", "Override model selection")
	cmd.PersistentFlags().StringSliceVarP(&flags.Context, "context", "c", nil, "Select contexts (tags or file paths)")
	cmd.PersistentFlags().StringVarP(&flags.Directory, "directory", "d", "", "Working directory for context detection")
	cmd.PersistentFlags().BoolVar(&flags.DryRun, "dry-run", false, "Preview execution without launching agent")
	cmd.PersistentFlags().BoolVarP(&flags.Quiet, "quiet", "q", false, "Suppress output")
	cmd.PersistentFlags().BoolVar(&flags.Verbose, "verbose", false, "Detailed output")
	cmd.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "Debug output (implies --verbose)")
	cmd.PersistentFlags().BoolVar(&flags.NoColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().BoolVarP(&flags.Local, "local", "l", false, "Target local config (./.start/) instead of global")
	cmd.PersistentFlags().BoolVar(&flags.NoRole, "no-role", false, "Skip role assignment")
	cmd.MarkFlagsMutuallyExclusive("role", "no-role")

	// Set RunE on root command for `start` execution
	cmd.RunE = runStart

	// Define command groups for help output
	cmd.AddGroup(
		&cobra.Group{ID: "commands", Title: "Commands:"},
		&cobra.Group{ID: "utilities", Title: "Utilities:"},
	)

	// Add subcommands
	addShowCommand(cmd)
	addPromptCommand(cmd)
	addTaskCommand(cmd)
	addAssetsCommand(cmd)
	addConfigCommand(cmd)
	addSearchCommand(cmd)
	addDoctorCommand(cmd)
	addCompletionCommand(cmd)

	// Move built-in help command into utilities group
	cmd.InitDefaultHelpCmd()
	for _, c := range cmd.Commands() {
		if c.Name() == "help" {
			c.GroupID = "utilities"
			break
		}
	}

	return cmd
}

// Execute runs the root command. This is the main entry point for the CLI.
func Execute() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("start does not support Windows (see DR-006 for platform scope)")
	}
	return NewRootCmd().Execute()
}

// checkHelpArg checks if the first argument is "help" and shows help if so.
// Returns true if help was shown, false otherwise.
// Use this in parent commands that have both RunE and subcommands.
func checkHelpArg(cmd *cobra.Command, args []string) (bool, error) {
	if len(args) > 0 && args[0] == "help" {
		return true, cmd.Help()
	}
	return false, nil
}

// unknownCommandError returns a formatted error for unknown subcommands.
func unknownCommandError(cmdPath, arg string) error {
	return fmt.Errorf("unknown command %q for %q\nRun '%s --help' for usage", arg, cmdPath, cmdPath)
}

// resolveDirectory expands and validates the directory path.
func resolveDirectory(path string) (string, error) {
	abs, err := orchestration.ExpandFilePath(path)
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

// isTerminal reports whether r is connected to a terminal.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
