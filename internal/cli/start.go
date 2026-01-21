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
	NoColor   bool
	Local     bool
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
// Format: [DEBUG] <category>: <message>
func debugf(flags *Flags, category, format string, args ...interface{}) {
	if flags.Debug {
		_, _ = fmt.Fprintf(os.Stderr, "[DEBUG] %s: "+format+"\n", append([]interface{}{category}, args...)...)
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
		debugf(flags, "config", "Working directory (from --directory): %s", workingDir)
	} else {
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		debugf(flags, "config", "Working directory (from pwd): %s", workingDir)
	}

	// Load configuration (uses workingDir for local config lookup)
	cfg, err := loadMergedConfigFromDirWithDebug(workingDir, flags)
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	// Select agent
	agentName := flags.Agent
	if agentName == "" {
		agentName = orchestration.GetDefaultAgent(cfg.Value)
		debugf(flags, "agent", "Selected %q (config default)", agentName)
	} else {
		debugf(flags, "agent", "Selected %q (--agent flag)", agentName)
	}
	if agentName == "" {
		return nil, fmt.Errorf("no agent configured")
	}

	agent, err := orchestration.ExtractAgent(cfg.Value, agentName)
	if err != nil {
		return nil, fmt.Errorf("loading agent: %w", err)
	}
	debugf(flags, "agent", "Binary: %s", agent.Bin)
	debugf(flags, "agent", "Command template: %s", agent.Command)

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

	debugf(flags, "context", "Selection: required=%t, defaults=%t, tags=%v",
		selection.IncludeRequired, selection.IncludeDefaults, selection.Tags)

	// Compose prompt with role
	result, err := env.Composer.ComposeWithRole(env.Cfg.Value, selection, flags.Role, customText, "")
	if err != nil {
		return fmt.Errorf("composing prompt: %w", err)
	}

	debugf(flags, "role", "Selected %q", result.RoleName)
	for _, ctx := range result.Contexts {
		debugf(flags, "context", "Including %q", ctx.Name)
	}
	debugf(flags, "compose", "Role: %d bytes", len(result.Role))
	debugf(flags, "compose", "Prompt: %d bytes (%d contexts)", len(result.Prompt), len(result.Contexts))

	// Print warnings
	printWarnings(flags, stderr, result.Warnings)

	// Determine effective model and its source
	model, modelSource := resolveModel(flags.Model, env.Agent.DefaultModel)
	if model != "" {
		debugf(flags, "agent", "Model: %s (%s)", model, modelSource)
	} else {
		debugf(flags, "agent", "Model: agent default (none specified)")
	}

	// Build execution config
	execConfig := orchestration.ExecuteConfig{
		Agent:      env.Agent,
		Model:      flags.Model,
		Role:       result.Role,
		RoleFile:   result.RoleFile,
		Prompt:     result.Prompt,
		WorkingDir: env.WorkingDir,
		DryRun:     flags.DryRun,
	}

	// Build command and validate before proceeding
	cmdStr, err := env.Executor.BuildCommand(execConfig)
	if err != nil {
		return err
	}
	debugf(flags, "exec", "Final command: %s", cmdStr)

	if flags.DryRun {
		debugf(flags, "exec", "Dry-run mode, skipping execution")
		return executeDryRun(stdout, env.Executor, execConfig, result, env.Agent, model, modelSource)
	}

	// Print execution info
	if !flags.Quiet {
		printExecutionInfo(stdout, env.Agent, model, modelSource, result)
	}

	debugf(flags, "exec", "Executing agent (process replacement)")
	// Execute agent (replaces current process) - command already validated
	return env.Executor.ExecuteCommand(cmdStr, execConfig)
}

// resolveModel determines the effective model and its source.
// Returns the model name and source ("--model" or "config").
// If both are empty, returns empty strings.
func resolveModel(flagModel, configModel string) (model, source string) {
	if flagModel != "" {
		return flagModel, "--model"
	}
	if configModel != "" {
		return configModel, "config"
	}
	return "", ""
}

// printWarnings prints warnings to stderr if not in quiet mode.
func printWarnings(flags *Flags, stderr io.Writer, warnings []string) {
	if flags.Quiet {
		return
	}
	for _, w := range warnings {
		PrintWarning(stderr, "%s", w)
	}
}

// executeDryRun handles --dry-run mode.
func executeDryRun(w io.Writer, executor *orchestration.Executor, cfg orchestration.ExecuteConfig, result orchestration.ComposeResult, agent orchestration.Agent, model, modelSource string) error {
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
	printDryRunSummary(w, agent, model, modelSource, result, dir)

	return nil
}

// printExecutionInfo prints the execution summary.
func printExecutionInfo(w io.Writer, agent orchestration.Agent, model, modelSource string, result orchestration.ComposeResult) {
	PrintHeader(w, "Starting AI Agent")
	PrintSeparator(w)

	_, _ = fmt.Fprintf(w, "Agent: %s\n", agent.Name)
	if model != "" {
		_, _ = fmt.Fprintf(w, "Model: %s (via %s)\n", model, modelSource)
	} else {
		_, _ = fmt.Fprintf(w, "Model: -\n")
	}
	_, _ = fmt.Fprintln(w)

	PrintContextTable(w, result.Contexts)

	PrintRoleTable(w, result.RoleName, result.RoleFile, result.Role != "")

	_, _ = fmt.Fprintf(w, "Starting %s - awaiting response...\n", agent.Name)
}

// printDryRunSummary prints the dry-run summary per DR-016.
func printDryRunSummary(w io.Writer, agent orchestration.Agent, model, modelSource string, result orchestration.ComposeResult, dir string) {
	PrintHeader(w, "Dry Run - Agent Not Executed")
	PrintSeparator(w)

	_, _ = fmt.Fprintf(w, "Agent: %s\n", agent.Name)
	if model != "" {
		_, _ = fmt.Fprintf(w, "Model: %s (via %s)\n", model, modelSource)
	} else {
		_, _ = fmt.Fprintf(w, "Model: -\n")
	}
	_, _ = fmt.Fprintln(w)

	PrintContextTable(w, result.Contexts)

	PrintRoleTable(w, result.RoleName, result.RoleFile, result.Role != "")

	// Show role preview
	if result.Role != "" {
		printContentPreview(w, "Role", result.Role, 5)
		_, _ = fmt.Fprintln(w)
	}

	// Show prompt preview
	if result.Prompt != "" {
		printContentPreview(w, "Prompt", result.Prompt, 5)
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintf(w, "Files: %s/\n", dir)
	_, _ = fmt.Fprintln(w, "  role.md")
	_, _ = fmt.Fprintln(w, "  prompt.md")
	_, _ = fmt.Fprintln(w, "  command.txt")
}

// printContentPreview prints content with a header showing line count only when truncated.
func printContentPreview(w io.Writer, label, text string, maxLines int) {
	lines := strings.Split(text, "\n")
	truncated := len(lines) > maxLines

	if truncated {
		_, _ = fmt.Fprintf(w, "%s (%d lines):\n", label, maxLines)
	} else {
		_, _ = fmt.Fprintf(w, "%s:\n", label)
	}

	shown := maxLines
	if len(lines) < shown {
		shown = len(lines)
	}
	for i := 0; i < shown; i++ {
		_, _ = fmt.Fprintf(w, "  %s\n", lines[i])
	}
	if truncated {
		_, _ = fmt.Fprintf(w, "  ... (%d more lines)\n", len(lines)-maxLines)
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

// loadMergedConfigFromDirWithDebug loads configuration with debug logging.
func loadMergedConfigFromDirWithDebug(workingDir string, flags *Flags) (internalcue.LoadResult, error) {
	// Resolve paths first for debug output
	paths, err := config.ResolvePaths(workingDir)
	if err != nil {
		return internalcue.LoadResult{}, fmt.Errorf("resolving config paths: %w", err)
	}

	debugf(flags, "config", "Global: %s (exists: %t)", paths.Global, paths.GlobalExists)
	debugf(flags, "config", "Local: %s (exists: %t)", paths.Local, paths.LocalExists)

	// Load using the standard function
	result, err := loadMergedConfigWithIO(os.Stdout, os.Stderr, os.Stdin, workingDir)
	if err != nil {
		return result, err
	}

	// Log what was loaded
	var loaded []string
	if result.GlobalLoaded {
		loaded = append(loaded, "global")
	}
	if result.LocalLoaded {
		loaded = append(loaded, "local")
	}
	if len(loaded) > 0 {
		debugf(flags, "config", "Loaded from: %s", strings.Join(loaded, ", "))
	}

	return result, nil
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

	_, _ = fmt.Fprintln(stdout)
	return nil
}
