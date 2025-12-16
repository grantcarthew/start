package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/shell"
	"github.com/grantcarthew/start/internal/temp"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Command flags
var (
	flagAgent   string
	flagRole    string
	flagModel   string
	flagContext []string
	flagDryRun  bool
	flagQuiet   bool
	flagVerbose bool
)

func init() {
	// Add flags to root command
	rootCmd.PersistentFlags().StringVarP(&flagAgent, "agent", "a", "", "Override agent selection")
	rootCmd.PersistentFlags().StringVarP(&flagRole, "role", "r", "", "Override role (system prompt)")
	rootCmd.PersistentFlags().StringVarP(&flagModel, "model", "m", "", "Override model selection")
	rootCmd.PersistentFlags().StringSliceVarP(&flagContext, "context", "c", nil, "Select contexts by tag")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Preview execution without launching agent")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress output")
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Detailed output")

	// Set RunE on root command for `start` execution
	rootCmd.RunE = runStart
}

// runStart executes the start command (root command with no subcommand).
func runStart(cmd *cobra.Command, args []string) error {
	return executeStart(cmd.OutOrStdout(), cmd.ErrOrStderr(), orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: true,
		Tags:            flagContext,
	}, "")
}

// executeStart is the shared execution logic for start commands.
func executeStart(stdout, stderr io.Writer, selection orchestration.ContextSelection, customText string) error {
	// Load configuration
	cfg, err := loadMergedConfig()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Get working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Select agent
	agentName := flagAgent
	if agentName == "" {
		agentName = orchestration.GetDefaultAgent(cfg.Value)
	}
	if agentName == "" {
		return fmt.Errorf("no agent configured")
	}

	agent, err := orchestration.ExtractAgent(cfg.Value, agentName)
	if err != nil {
		return fmt.Errorf("loading agent: %w", err)
	}

	// Select role
	roleName := flagRole

	// Create shell runner and template processor
	shellRunner := shell.NewRunner()
	processor := orchestration.NewTemplateProcessor(nil, shellRunner, workingDir)
	composer := orchestration.NewComposer(processor, workingDir)

	// Compose prompt with role
	result, err := composer.ComposeWithRole(cfg.Value, selection, roleName, customText, "")
	if err != nil {
		return fmt.Errorf("composing prompt: %w", err)
	}

	// Print warnings
	for _, w := range result.Warnings {
		if !flagQuiet {
			fmt.Fprintf(stderr, "Warning: %s\n", w)
		}
	}

	// Build execution config
	execConfig := orchestration.ExecuteConfig{
		Agent:      agent,
		Model:      flagModel,
		Role:       result.Role,
		Prompt:     result.Prompt,
		WorkingDir: workingDir,
		DryRun:     flagDryRun,
	}

	executor := orchestration.NewExecutor(workingDir)

	if flagDryRun {
		return executeDryRun(stdout, executor, execConfig, result, agent)
	}

	// Print execution info
	if !flagQuiet {
		printExecutionInfo(stdout, agent, flagModel, result)
	}

	// Execute agent (replaces current process)
	return executor.Execute(execConfig)
}

// executeDryRun handles --dry-run mode.
func executeDryRun(w io.Writer, executor *orchestration.Executor, cfg orchestration.ExecuteConfig, result orchestration.ComposeResult, agent orchestration.Agent) error {
	// Build command string
	cmdStr, err := executor.BuildCommand(cfg)
	if err != nil {
		return fmt.Errorf("building command: %w", err)
	}

	// Create temp directory
	tempMgr := temp.NewDryRunManager()
	dir, err := tempMgr.DryRunDir()
	if err != nil {
		return fmt.Errorf("creating dry-run directory: %w", err)
	}

	// Get context names
	var contextNames []string
	for _, ctx := range result.Contexts {
		contextNames = append(contextNames, ctx.Name)
	}

	// Generate command file content
	cmdContent := orchestration.GenerateDryRunCommand(agent, cfg.Model, result.RoleName, contextNames, cfg.WorkingDir, cmdStr)

	// Write files
	if err := tempMgr.WriteDryRunFiles(dir, result.Role, result.Prompt, cmdContent); err != nil {
		return fmt.Errorf("writing dry-run files: %w", err)
	}

	// Print summary
	printDryRunSummary(w, agent, cfg.Model, result, dir)

	return nil
}

// printExecutionInfo prints the execution summary.
func printExecutionInfo(w io.Writer, agent orchestration.Agent, model string, result orchestration.ComposeResult) {
	fmt.Fprintln(w, "Starting AI Agent")
	fmt.Fprintln(w, strings.Repeat("─", 79))

	modelStr := model
	if modelStr == "" {
		modelStr = agent.DefaultModel
	}
	fmt.Fprintf(w, "Agent: %s (model: %s)\n", agent.Name, modelStr)
	fmt.Fprintln(w)

	if len(result.Contexts) > 0 {
		fmt.Fprintln(w, "Context documents:")
		for _, ctx := range result.Contexts {
			marker := "✓"
			fmt.Fprintf(w, "  %s %s\n", marker, ctx.Name)
		}
		fmt.Fprintln(w)
	}

	if result.RoleName != "" {
		fmt.Fprintf(w, "Role: %s\n", result.RoleName)
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "Executing...")
}

// printDryRunSummary prints the dry-run summary per DR-016.
func printDryRunSummary(w io.Writer, agent orchestration.Agent, model string, result orchestration.ComposeResult, dir string) {
	fmt.Fprintln(w, "Dry Run - Agent Not Executed")
	fmt.Fprintln(w, strings.Repeat("─", 79))

	modelStr := model
	if modelStr == "" {
		modelStr = agent.DefaultModel
	}
	fmt.Fprintf(w, "Agent: %s (model: %s)\n", agent.Name, modelStr)
	fmt.Fprintf(w, "Role: %s\n", result.RoleName)

	var contextNames []string
	for _, ctx := range result.Contexts {
		contextNames = append(contextNames, ctx.Name)
	}
	fmt.Fprintf(w, "Contexts: %s\n", strings.Join(contextNames, ", "))
	fmt.Fprintln(w)

	// Show 5-line preview of role
	if result.Role != "" {
		fmt.Fprintln(w, "Role (5 lines):")
		printPreviewLines(w, result.Role, 5)
		fmt.Fprintln(w)
	}

	// Show 5-line preview of prompt
	if result.Prompt != "" {
		fmt.Fprintln(w, "Prompt (5 lines):")
		printPreviewLines(w, result.Prompt, 5)
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Files: %s/\n", dir)
	fmt.Fprintln(w, "  role.md")
	fmt.Fprintln(w, "  prompt.md")
	fmt.Fprintln(w, "  command.txt")
}

// printPreviewLines prints up to n lines of text with indentation.
func printPreviewLines(w io.Writer, text string, n int) {
	lines := strings.Split(text, "\n")
	shown := n
	if len(lines) < shown {
		shown = len(lines)
	}
	for i := 0; i < shown; i++ {
		fmt.Fprintf(w, "  %s\n", lines[i])
	}
	if len(lines) > n {
		fmt.Fprintf(w, "  ... (%d more lines)\n", len(lines)-n)
	}
}

// loadMergedConfig loads and merges global and local configuration.
// If no configuration exists, it triggers auto-setup.
func loadMergedConfig() (internalcue.LoadResult, error) {
	return loadMergedConfigWithIO(os.Stdout, os.Stderr, os.Stdin)
}

// loadMergedConfigWithIO loads configuration with custom I/O streams.
func loadMergedConfigWithIO(stdout, stderr io.Writer, stdin io.Reader) (internalcue.LoadResult, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return internalcue.LoadResult{}, fmt.Errorf("resolving config paths: %w", err)
	}

	// Trigger auto-setup if no config exists
	if !paths.AnyExists() {
		if err := runAutoSetup(stdout, stderr, stdin); err != nil {
			return internalcue.LoadResult{}, err
		}
		// Re-resolve paths after auto-setup
		paths, err = config.ResolvePaths("")
		if err != nil {
			return internalcue.LoadResult{}, fmt.Errorf("resolving config paths: %w", err)
		}
	}

	dirs := paths.ForScope(config.ScopeMerged)
	loader := internalcue.NewLoader()
	return loader.Load(dirs)
}

// runAutoSetup runs the auto-setup flow.
func runAutoSetup(stdout, stderr io.Writer, stdin io.Reader) error {
	// Check if stdin is a TTY
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	autoSetup := orchestration.NewAutoSetup(stdout, stderr, stdin, isTTY)
	ctx := context.Background()

	_, err := autoSetup.Run(ctx)
	if err != nil {
		return fmt.Errorf("auto-setup failed: %w", err)
	}

	fmt.Fprintln(stdout)
	return nil
}
