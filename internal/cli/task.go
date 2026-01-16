package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/grantcarthew/start/internal/temp"
	"github.com/spf13/cobra"
)

// addTaskCommand adds the task command to the parent command.
func addTaskCommand(parent *cobra.Command) {
	taskCmd := &cobra.Command{
		Use:   "task [name] [instructions]",
		Short: "Run a predefined task",
		Long: `Run a predefined task with optional instructions.

The name can be a config task name or a file path (starting with ./, /, or ~).
Tasks are reusable workflows defined in configuration.
Instructions are passed to the task template via the {{.Instructions}} placeholder.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runTask,
	}
	parent.AddCommand(taskCmd)
}

// runTask executes the task command.
func runTask(cmd *cobra.Command, args []string) error {
	taskName := args[0]
	instructions := ""
	if len(args) > 1 {
		instructions = args[1]
	}

	flags := getFlags(cmd)
	return executeTask(cmd.OutOrStdout(), cmd.ErrOrStderr(), flags, taskName, instructions)
}

// executeTask handles task execution.
func executeTask(stdout, stderr io.Writer, flags *Flags, taskName, instructions string) error {
	env, err := prepareExecutionEnv(flags)
	if err != nil {
		return err
	}

	debugf(flags, "task", "Searching for task %q", taskName)

	// Check if taskName is a file path (per DR-038)
	var taskResult orchestration.ProcessResult
	var resolvedName string
	roleName := flags.Role
	if orchestration.IsFilePath(taskName) {
		debugf(flags, "task", "Detected file path, reading file")
		content, err := orchestration.ReadFilePath(taskName)
		if err != nil {
			return fmt.Errorf("reading task file %q: %w", taskName, err)
		}
		// Process through template processor for {{.instructions}} support
		taskResult, err = env.Composer.ProcessContent(content, instructions)
		if err != nil {
			return fmt.Errorf("processing task file: %w", err)
		}
		taskResult.FileRead = true
		resolvedName = taskName // Display file path as task name
	} else {
		// Resolve task name (exact match or substring match)
		resolvedName, err = findTask(env.Cfg, taskName)
		if err != nil {
			// Per DR-015: if not found locally, try fetching from registry
			if isTaskNotFoundError(err, taskName) {
				debugf(flags, "task", "Not found locally, checking registry...")
				installed, installErr := tryInstallTaskFromRegistry(stdout, stderr, flags, taskName)
				if installErr != nil {
					// Registry install failed, return original error
					debugf(flags, "task", "Registry install failed: %v", installErr)
					return err
				}
				if installed {
					// Reload config and retry
					debugf(flags, "task", "Installed from registry, reloading config...")
					env, err = prepareExecutionEnv(flags)
					if err != nil {
						return err
					}
					resolvedName, err = findTask(env.Cfg, taskName)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			} else {
				return err
			}
		}

		if resolvedName != taskName {
			debugf(flags, "task", "Resolved to %q (substring match)", resolvedName)
		} else {
			debugf(flags, "task", "Resolved to %q (exact match)", resolvedName)
		}

		// Resolve task from config
		taskResult, err = env.Composer.ResolveTask(env.Cfg.Value, resolvedName, instructions)
		if err != nil {
			return fmt.Errorf("resolving task: %w", err)
		}

		// Get task's role if not specified via flag
		if roleName == "" {
			roleName = orchestration.GetTaskRole(env.Cfg.Value, resolvedName)
			if roleName != "" {
				debugf(flags, "role", "Selected %q (from task)", roleName)
			}
		}
	}

	if taskResult.CommandExecuted {
		debugf(flags, "task", "UTD source: command (executed)")
	} else if taskResult.FileRead {
		debugf(flags, "task", "UTD source: file")
	} else {
		debugf(flags, "task", "UTD source: prompt")
	}

	if instructions != "" {
		debugf(flags, "task", "Instructions: %s", instructions)
	}

	// Log role source if specified via flag
	if flags.Role != "" {
		debugf(flags, "role", "Selected %q (--role flag)", flags.Role)
	}

	// Per DR-015: required contexts only for tasks
	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: false,
		Tags:            flags.Context,
	}

	debugf(flags, "context", "Selection: required=%t, defaults=%t, tags=%v",
		selection.IncludeRequired, selection.IncludeDefaults, selection.Tags)

	// Compose contexts and resolve role
	composeResult, err := env.Composer.ComposeWithRole(env.Cfg.Value, selection, roleName, taskResult.Content, "")
	if err != nil {
		return fmt.Errorf("composing prompt: %w", err)
	}

	for _, ctx := range composeResult.Contexts {
		debugf(flags, "context", "Including %q", ctx.Name)
	}
	debugf(flags, "compose", "Role: %d bytes", len(composeResult.Role))
	debugf(flags, "compose", "Prompt: %d bytes (%d contexts)", len(composeResult.Prompt), len(composeResult.Contexts))

	// Print warnings
	printWarnings(flags, stderr, taskResult.Warnings)
	printWarnings(flags, stderr, composeResult.Warnings)

	// Build execution config
	execConfig := orchestration.ExecuteConfig{
		Agent:      env.Agent,
		Model:      flags.Model,
		Role:       composeResult.Role,
		Prompt:     composeResult.Prompt,
		WorkingDir: env.WorkingDir,
		DryRun:     flags.DryRun,
	}

	// Build and log final command
	if flags.Debug {
		cmdStr, err := env.Executor.BuildCommand(execConfig)
		if err == nil {
			debugf(flags, "exec", "Final command: %s", cmdStr)
		}
	}

	if flags.DryRun {
		debugf(flags, "exec", "Dry-run mode, skipping execution")
		return executeTaskDryRun(stdout, env.Executor, execConfig, composeResult, env.Agent, resolvedName, instructions)
	}

	// Print execution info
	if !flags.Quiet {
		printTaskExecutionInfo(stdout, env.Agent, flags.Model, composeResult, resolvedName, instructions, taskResult)
	}

	debugf(flags, "exec", "Executing agent (process replacement)")
	// Execute agent (replaces current process)
	return env.Executor.Execute(execConfig)
}

// executeTaskDryRun handles --dry-run mode for tasks.
func executeTaskDryRun(w io.Writer, executor *orchestration.Executor, cfg orchestration.ExecuteConfig, result orchestration.ComposeResult, agent orchestration.Agent, taskName, instructions string) error {
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
	printTaskDryRunSummary(w, agent, cfg.Model, result, dir, taskName, instructions)

	return nil
}

// printTaskExecutionInfo prints the task execution summary.
func printTaskExecutionInfo(w io.Writer, agent orchestration.Agent, model string, result orchestration.ComposeResult, taskName, instructions string, taskResult orchestration.ProcessResult) {
	PrintHeader(w, fmt.Sprintf("Starting Task: %s", taskName))
	PrintSeparator(w)

	modelStr := model
	if modelStr == "" {
		modelStr = agent.DefaultModel
	}
	if modelStr == "" {
		modelStr = "-"
	}
	_, _ = fmt.Fprintf(w, "Agent: %s\n", agent.Name)
	_, _ = fmt.Fprintf(w, "Model: %s\n", modelStr)
	_, _ = fmt.Fprintln(w)

	PrintContextTable(w, result.Contexts)

	if result.RoleName != "" {
		_, _ = fmt.Fprintf(w, "Role: %s\n", result.RoleName)
	}

	if taskResult.CommandExecuted {
		_, _ = fmt.Fprintln(w, "Command: executed")
	}

	if instructions != "" {
		_, _ = fmt.Fprintf(w, "Instructions: %s\n", instructions)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Starting %s - awaiting response...\n", agent.Name)
}

// printTaskDryRunSummary prints the task dry-run summary.
func printTaskDryRunSummary(w io.Writer, agent orchestration.Agent, model string, result orchestration.ComposeResult, dir, taskName, instructions string) {
	PrintHeader(w, fmt.Sprintf("Dry Run - Task: %s", taskName))
	PrintSeparator(w)

	modelStr := model
	if modelStr == "" {
		modelStr = agent.DefaultModel
	}
	if modelStr == "" {
		modelStr = "-"
	}
	_, _ = fmt.Fprintf(w, "Agent: %s\n", agent.Name)
	_, _ = fmt.Fprintf(w, "Model: %s\n", modelStr)
	_, _ = fmt.Fprintf(w, "Role: %s\n", result.RoleName)
	_, _ = fmt.Fprintln(w)

	PrintContextTable(w, result.Contexts)

	if instructions != "" {
		_, _ = fmt.Fprintf(w, "Instructions: %s\n", instructions)
		_, _ = fmt.Fprintln(w)
	}

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

// findTask attempts to find a task by exact name or prefix match.
func findTask(cfg internalcue.LoadResult, name string) (string, error) {
	tasks := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyTasks))
	if !tasks.Exists() {
		return "", fmt.Errorf("no tasks defined")
	}

	// Try exact match first
	if tasks.LookupPath(cue.MakePath(cue.Str(name))).Exists() {
		return name, nil
	}

	// Try substring match
	var matches []string
	iter, err := tasks.Fields()
	if err != nil {
		return "", fmt.Errorf("reading tasks: %w", err)
	}

	for iter.Next() {
		taskName := iter.Selector().Unquoted()
		if strings.Contains(taskName, name) {
			matches = append(matches, taskName)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("task %q not found", name)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous task prefix %q matches: %s", name, strings.Join(matches, ", "))
	}
}

// isTaskNotFoundError checks if the error is a "task not found" error.
// This includes both "task X not found" and "no tasks defined".
func isTaskNotFoundError(err error, name string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == fmt.Sprintf("task %q not found", name) || msg == "no tasks defined"
}

// tryInstallTaskFromRegistry searches the registry for an exact task match and installs it.
// Returns (true, nil) if installed successfully, (false, nil) if not found, (false, error) on failure.
func tryInstallTaskFromRegistry(stdout, stderr io.Writer, flags *Flags, taskName string) (bool, error) {
	ctx := context.Background()

	// Create registry client
	client, err := registry.NewClient()
	if err != nil {
		return false, fmt.Errorf("creating registry client: %w", err)
	}

	// Fetch index
	if !flags.Quiet {
		_, _ = fmt.Fprintln(stdout, "Task not found locally, checking registry...")
	}
	index, err := client.FetchIndex(ctx)
	if err != nil {
		return false, fmt.Errorf("fetching registry index: %w", err)
	}

	// Search for exact match in tasks category
	result := findExactTaskInRegistry(index, taskName)
	if result == nil {
		debugf(flags, "task", "No exact match found in registry for %q", taskName)
		return false, nil
	}

	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Found in registry: %s/%s\n", result.Category, result.Name)
		_, _ = fmt.Fprintln(stdout, "Installing...")
	}

	// Install the task using the same logic as assets add
	paths, err := config.ResolvePaths("")
	if err != nil {
		return false, fmt.Errorf("resolving config paths: %w", err)
	}

	// Always install to global config for auto-install
	configDir := paths.Global

	// Fetch the actual asset module from registry
	modulePath := result.Entry.Module
	if !strings.Contains(modulePath, "@") {
		modulePath += "@v0"
	}

	// Resolve to canonical version
	resolvedPath, err := client.ResolveLatestVersion(ctx, modulePath)
	if err != nil {
		return false, fmt.Errorf("resolving asset version: %w", err)
	}

	fetchResult, err := client.Fetch(ctx, resolvedPath)
	if err != nil {
		return false, fmt.Errorf("fetching asset module: %w", err)
	}

	// Extract asset content from fetched module
	originPath := modulePath
	if idx := strings.Index(originPath, "@"); idx != -1 {
		originPath = originPath[:idx]
	}
	assetContent, err := extractAssetContent(fetchResult.SourceDir, *result, client.Registry(), originPath)
	if err != nil {
		return false, fmt.Errorf("extracting asset content: %w", err)
	}

	// Determine the config file
	configFile := assetTypeToConfigFile(result.Category)
	configPath := configDir + "/" + configFile

	// Write the asset to config
	if err := writeAssetToConfig(configPath, *result, assetContent, modulePath); err != nil {
		return false, fmt.Errorf("writing config: %w", err)
	}

	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Installed %s/%s to global config\n\n", result.Category, result.Name)
	}

	return true, nil
}

// findExactTaskInRegistry searches for an exact task match in the registry index.
// Supports both "name" and "category/name" formats (e.g., "code-review" or "golang/code-review").
func findExactTaskInRegistry(index *registry.Index, taskName string) *SearchResult {
	// Check if taskName includes a path prefix (e.g., "golang/code-review")
	for name, entry := range index.Tasks {
		// Exact match on full name (e.g., "golang/code-review" matches "golang/code-review")
		if name == taskName {
			return &SearchResult{
				Category: "tasks",
				Name:     name,
				Entry:    entry,
			}
		}
		// Also match if the short name matches (e.g., "code-review" matches "golang/code-review")
		shortName := getAssetKey(name)
		if shortName == taskName {
			return &SearchResult{
				Category: "tasks",
				Name:     name,
				Entry:    entry,
			}
		}
	}
	return nil
}
