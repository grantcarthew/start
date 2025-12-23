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

// flagsKey is the context key for storing Flags.
type flagsKey struct{}

// Flags holds all CLI flag values. Each command instance gets its own Flags,
// enabling parallel test execution without shared state.
type Flags struct {
	Agent     string
	Role      string
	Model     string
	Context   []string
	Directory string
	DryRun    bool
	Quiet     bool
	Verbose   bool
	Debug     bool
}

// getFlags retrieves Flags from the command context.
func getFlags(cmd *cobra.Command) *Flags {
	if f, ok := cmd.Context().Value(flagsKey{}).(*Flags); ok {
		return f
	}
	// Fallback for commands without context (shouldn't happen in normal use)
	return &Flags{}
}

// debugf prints debug output if debug mode is enabled.
func debugf(flags *Flags, format string, args ...interface{}) {
	if flags.Debug {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// ExecutionEnv holds the common execution environment for start and task commands.
type ExecutionEnv struct {
	Cfg        internalcue.LoadResult
	WorkingDir string
	Agent      orchestration.Agent
	Composer   *orchestration.Composer
	Executor   *orchestration.Executor
}

// prepareExecutionEnv prepares the common execution environment.
// This is shared between executeStart and executeTask to avoid code duplication.
func prepareExecutionEnv(flags *Flags) (*ExecutionEnv, error) {
	// Determine working directory
	var workingDir string
	var err error
	if flags.Directory != "" {
		workingDir = flags.Directory
		debugf(flags, "Working directory (from --directory): %s", workingDir)
	} else {
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		debugf(flags, "Working directory (from pwd): %s", workingDir)
	}

	// Load configuration (uses workingDir for local config lookup)
	cfg, err := loadMergedConfigFromDir(workingDir)
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	// Select agent
	agentName := flags.Agent
	if agentName == "" {
		agentName = orchestration.GetDefaultAgent(cfg.Value)
		debugf(flags, "Agent (from config default): %s", agentName)
	} else {
		debugf(flags, "Agent (from --agent flag): %s", agentName)
	}
	if agentName == "" {
		return nil, fmt.Errorf("no agent configured")
	}

	agent, err := orchestration.ExtractAgent(cfg.Value, agentName)
	if err != nil {
		return nil, fmt.Errorf("loading agent: %w", err)
	}
	debugf(flags, "Agent binary: %s", agent.Bin)
	debugf(flags, "Agent command template: %s", agent.Command)

	// Create shell runner and template processor
	shellRunner := shell.NewRunner()
	processor := orchestration.NewTemplateProcessor(nil, shellRunner, workingDir)
	composer := orchestration.NewComposer(processor, workingDir)
	executor := orchestration.NewExecutor(workingDir)

	return &ExecutionEnv{
		Cfg:        cfg,
		WorkingDir: workingDir,
		Agent:      agent,
		Composer:   composer,
		Executor:   executor,
	}, nil
}

// runStart executes the start command (root command with no subcommand).
func runStart(cmd *cobra.Command, args []string) error {
	flags := getFlags(cmd)
	return executeStart(cmd.OutOrStdout(), cmd.ErrOrStderr(), flags, orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: true,
		Tags:            flags.Context,
	}, "")
}

// executeStart is the shared execution logic for start commands.
func executeStart(stdout, stderr io.Writer, flags *Flags, selection orchestration.ContextSelection, customText string) error {
	env, err := prepareExecutionEnv(flags)
	if err != nil {
		return err
	}

	debugf(flags, "Context selection: required=%t, defaults=%t, tags=%v",
		selection.IncludeRequired, selection.IncludeDefaults, selection.Tags)

	// Compose prompt with role
	result, err := env.Composer.ComposeWithRole(env.Cfg.Value, selection, flags.Role, customText, "")
	if err != nil {
		return fmt.Errorf("composing prompt: %w", err)
	}

	debugf(flags, "Role resolved: %s", result.RoleName)
	for _, ctx := range result.Contexts {
		debugf(flags, "Context included: %s", ctx.Name)
	}

	// Print warnings
	printWarnings(flags, stderr, result.Warnings)

	// Determine effective model
	model := flags.Model
	if model == "" {
		model = env.Agent.DefaultModel
	}
	debugf(flags, "Model: %s", model)

	// Build execution config
	execConfig := orchestration.ExecuteConfig{
		Agent:      env.Agent,
		Model:      flags.Model,
		Role:       result.Role,
		Prompt:     result.Prompt,
		WorkingDir: env.WorkingDir,
		DryRun:     flags.DryRun,
	}

	// Build and log final command
	if flags.Debug {
		cmdStr, err := env.Executor.BuildCommand(execConfig)
		if err == nil {
			debugf(flags, "Final command: %s", cmdStr)
		}
	}

	if flags.DryRun {
		return executeDryRun(stdout, env.Executor, execConfig, result, env.Agent)
	}

	// Print execution info
	if !flags.Quiet {
		printExecutionInfo(stdout, env.Agent, flags.Model, result)
	}

	// Execute agent (replaces current process)
	return env.Executor.Execute(execConfig)
}

// printWarnings prints warnings to stderr if not in quiet mode.
func printWarnings(flags *Flags, stderr io.Writer, warnings []string) {
	if flags.Quiet {
		return
	}
	for _, w := range warnings {
		fmt.Fprintf(stderr, "Warning: %s\n", w)
	}
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
	return loadMergedConfigFromDir("")
}

// loadMergedConfigFromDir loads configuration using the specified working directory
// for local config resolution. If workingDir is empty, uses current directory.
func loadMergedConfigFromDir(workingDir string) (internalcue.LoadResult, error) {
	return loadMergedConfigWithIO(os.Stdout, os.Stderr, os.Stdin, workingDir)
}

// loadMergedConfigWithIO loads configuration with custom I/O streams.
func loadMergedConfigWithIO(stdout, stderr io.Writer, stdin io.Reader, workingDir string) (internalcue.LoadResult, error) {
	paths, err := config.ResolvePaths(workingDir)
	if err != nil {
		return internalcue.LoadResult{}, fmt.Errorf("resolving config paths: %w", err)
	}

	// Validate existing configuration
	validation := config.ValidateConfig(paths)

	// If no valid config exists, trigger auto-setup
	if !validation.AnyValid() {
		// Check if there are validation errors (config exists but is invalid)
		if validation.HasErrors() {
			// Report the first error with full details
			if validation.GlobalError != nil {
				return internalcue.LoadResult{}, fmt.Errorf("%s", validation.GlobalError.DetailedError())
			}
			if validation.LocalError != nil {
				return internalcue.LoadResult{}, fmt.Errorf("%s", validation.LocalError.DetailedError())
			}
		}

		// No config at all - trigger auto-setup
		if err := runAutoSetup(stdout, stderr, stdin); err != nil {
			return internalcue.LoadResult{}, err
		}
		// Re-resolve and validate after auto-setup
		paths, err = config.ResolvePaths(workingDir)
		if err != nil {
			return internalcue.LoadResult{}, fmt.Errorf("resolving config paths: %w", err)
		}
		validation = config.ValidateConfig(paths)
		if !validation.AnyValid() {
			return internalcue.LoadResult{}, fmt.Errorf("auto-setup did not create valid configuration")
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
