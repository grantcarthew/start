package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/grantcarthew/start/internal/temp"
	"github.com/spf13/cobra"
)

// TaskSource indicates where a task comes from.
type TaskSource string

const (
	TaskSourceInstalled TaskSource = "installed"
	TaskSourceRegistry  TaskSource = "registry"
)

// TaskMatch represents a task found during resolution.
type TaskMatch struct {
	Name   string              // Task name (e.g., "golang/debug")
	Source TaskSource          // Where the task comes from
	Entry  registry.IndexEntry // Registry entry (only set if Source == TaskSourceRegistry)
}

// maxTaskResults is the maximum number of tasks to display in interactive selection.
const maxTaskResults = 20

// addTaskCommand adds the task command to the parent command.
func addTaskCommand(parent *cobra.Command) {
	taskCmd := &cobra.Command{
		Use:     "task [name] [instructions]",
		Aliases: []string{"tasks"},
		GroupID: "commands",
		Short:   "Run a predefined task",
		Long: `Run a predefined task with optional instructions.

The name can be a config task name or a file path (starting with ./, /, or ~).
Tasks are reusable workflows defined in configuration.
Instructions are passed to the task template via the {{.instructions}} placeholder.`,
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
	return executeTask(cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd.InOrStdin(), flags, taskName, instructions)
}

// executeTask handles task execution.
func executeTask(stdout, stderr io.Writer, stdin io.Reader, flags *Flags, taskName, instructions string) error {
	// Phase 1: Load config
	cfg, workingDir, err := loadExecutionConfig(flags)
	if err != nil {
		return err
	}

	// Phase 2: Resolve asset flags (agent, role, context)
	r := newResolver(cfg, flags, stdout, stdin)

	agentName := flags.Agent
	if agentName != "" {
		agentName, err = r.resolveAgent(agentName)
		if err != nil {
			return err
		}
	}

	var roleName string
	if !flags.NoRole {
		roleName = flags.Role
		if roleName != "" {
			roleName, err = r.resolveRole(roleName)
			if err != nil {
				return err
			}
		}
	}

	contextTags := flags.Context
	if len(contextTags) > 0 {
		contextTags = r.resolveContexts(contextTags)
	}

	// Track if we need a config reload (from flag resolution installs)
	flagInstalled := r.didInstall

	// If flag resolution installed assets, reload config
	if flagInstalled {
		if err := r.reloadConfig(workingDir); err != nil {
			return err
		}
		cfg = r.cfg
	}

	// Phase 3: Build execution environment with resolved agent
	env, err := buildExecutionEnv(cfg, workingDir, agentName, flags, stdout, stdin)
	if err != nil {
		return err
	}

	// Resolve --model flag against agent's models map
	resolvedModel := flags.Model
	if resolvedModel != "" {
		resolvedModel = r.resolveModelName(resolvedModel, env.Agent)
	}

	debugf(flags, dbgTask, "Searching for task %q", taskName)

	// Check if taskName is a file path (per DR-038)
	var taskResult orchestration.ProcessResult
	var resolvedName string
	if orchestration.IsFilePath(taskName) {
		debugf(flags, dbgTask, "Detected file path, reading file")
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
		// Per DR-015: Unified task resolution across installed config and registry

		// Step 1: Check for exact match in installed config (fast path, no registry fetch)
		if hasExactInstalledTask(env.Cfg, taskName) {
			debugf(flags, dbgTask, "Exact match found in installed config")
			resolvedName = taskName
		} else {
			// Step 2: No exact match - fetch registry and combine results
			debugf(flags, dbgTask, "No exact installed match, fetching registry index...")
			if !flags.Quiet {
				_, _ = fmt.Fprintf(stdout, "Task not found in configuration\n")
			}

			index, client, err := r.ensureIndex()
			if err != nil {
				return fmt.Errorf("fetching registry index: %w", err)
			}
			if index == nil {
				return fmt.Errorf("task %q not found and registry is unavailable", taskName)
			}

			// Check for exact match in registry
			exactRegistry, err := findExactTaskInRegistry(index, taskName)
			if err != nil {
				return err
			}
			if exactRegistry != nil {
				debugf(flags, dbgTask, "Exact match found in registry: %s", exactRegistry.Name)
				if !flags.Quiet {
					_, _ = fmt.Fprintf(stdout, "Installing %s from registry...\n", exactRegistry.Name)
				}
				env, err = installTaskAndReloadEnv(stdout, stdin, flags, client, index, *exactRegistry, workingDir, agentName)
				if err != nil {
					return err
				}
				resolvedName = exactRegistry.Name
			} else {
				// Step 3: Regex match across installed + registry
				installedMatches, err := findInstalledTasks(env.Cfg, taskName)
				if err != nil {
					return err
				}
				registryMatches, err := findRegistryTasks(index, taskName)
				if err != nil {
					return err
				}
				allMatches := mergeTaskMatches(installedMatches, registryMatches)

				debugf(flags, dbgTask, "Found %d installed matches, %d registry matches, %d total",
					len(installedMatches), len(registryMatches), len(allMatches))

				switch len(allMatches) {
				case 0:
					return fmt.Errorf("task %q not found", taskName)
				case 1:
					// Single match - use it
					match := allMatches[0]
					if match.Source == TaskSourceRegistry {
						if !flags.Quiet {
							_, _ = fmt.Fprintf(stdout, "Installing %s from registry...\n", match.Name)
						}
						result := assets.SearchResult{
							Category: "tasks",
							Name:     match.Name,
							Entry:    match.Entry,
						}
						env, err = installTaskAndReloadEnv(stdout, stdin, flags, client, index, result, workingDir, agentName)
						if err != nil {
							return err
						}
					}
					resolvedName = match.Name
				default:
					// Multiple matches - interactive selection
					debugf(flags, dbgTask, "Multiple matches, prompting for selection")
					isTTY := isTerminal(stdin)
					if !isTTY {
						var names []string
						for _, m := range allMatches {
							names = append(names, m.Name)
						}
						return fmt.Errorf("ambiguous task %q matches: %s\nSpecify exact name or run interactively", taskName, strings.Join(names, ", "))
					}
					reader := bufio.NewReader(stdin)
					selected, err := promptTaskSelection(stdout, reader, allMatches, taskName)
					if err != nil {
						return err
					}

					if selected.Source == TaskSourceRegistry {
						if !flags.Quiet {
							_, _ = fmt.Fprintf(stdout, "Installing %s from registry...\n", selected.Name)
						}
						result := assets.SearchResult{
							Category: "tasks",
							Name:     selected.Name,
							Entry:    selected.Entry,
						}
						env, err = installTaskAndReloadEnv(stdout, stdin, flags, client, index, result, workingDir, agentName)
						if err != nil {
							return err
						}
					}
					resolvedName = selected.Name
				}
			}
		}

		if resolvedName != taskName {
			debugf(flags, dbgTask, "Resolved to %q", resolvedName)
		} else {
			debugf(flags, dbgTask, "Resolved to %q (exact match)", resolvedName)
		}

		// Resolve task from config
		taskResult, err = env.Composer.ResolveTask(env.Cfg.Value, resolvedName, instructions)
		if err != nil {
			return fmt.Errorf("resolving task: %w", err)
		}

		// Get task's role if not specified via flag and --no-role not set
		if !flags.NoRole && roleName == "" {
			roleName = orchestration.GetTaskRole(env.Cfg.Value, resolvedName)
			if roleName != "" {
				// If the task's role is not installed, resolve through three-tier
				// search which may auto-install from registry (same as --role flag).
				if !hasExactInstalled(env.Cfg.Value, internalcue.KeyRoles, roleName) {
					beforeInstall := r.didInstall
					roleName, err = r.resolveRole(roleName)
					if err != nil {
						return err
					}
					if r.didInstall && !beforeInstall {
						env, err = reloadEnv(workingDir, agentName, flags, stdout, stdin)
						if err != nil {
							return err
						}
					}
				}
				debugf(flags, dbgRole, "Selected %q (from task)", roleName)
			}
		}
	}

	if taskResult.CommandExecuted {
		debugf(flags, dbgTask, "UTD source: command (executed)")
	} else if taskResult.FileRead {
		debugf(flags, dbgTask, "UTD source: file")
	} else {
		debugf(flags, dbgTask, "UTD source: prompt")
	}

	if instructions != "" {
		debugf(flags, dbgTask, "Instructions: %s", instructions)
	}

	// Log role source if specified via flag
	if flags.Role != "" {
		debugf(flags, dbgRole, "Selected %q (--role flag)", flags.Role)
	}

	// Per DR-015: required contexts only for tasks
	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: false,
		Tags:            contextTags,
	}

	debugf(flags, dbgContext, "Selection: required=%t, defaults=%t, tags=%v",
		selection.IncludeRequired, selection.IncludeDefaults, selection.Tags)

	// Compose contexts and resolve role
	var composeResult orchestration.ComposeResult
	var composeErr error
	if flags.NoRole {
		debugf(flags, dbgRole, "Skipping role (--no-role)")
		composeResult, composeErr = env.Composer.Compose(env.Cfg.Value, selection, taskResult.Content, "")
	} else {
		composeResult, composeErr = env.Composer.ComposeWithRole(env.Cfg.Value, selection, roleName, taskResult.Content, "")
	}
	if composeErr != nil {
		// Show UI with role resolutions before returning error
		if !flags.Quiet && len(composeResult.RoleResolutions) > 0 {
			printComposeError(stdout, env.Agent, composeResult)
		}
		return fmt.Errorf("composing prompt: %w", composeErr)
	}

	for _, ctx := range composeResult.Contexts {
		debugf(flags, dbgContext, "Including %q", ctx.Name)
	}
	debugf(flags, dbgCompose, "Role: %d bytes", len(composeResult.Role))
	debugf(flags, dbgCompose, "Prompt: %d bytes (%d contexts)", len(composeResult.Prompt), len(composeResult.Contexts))

	// Print warnings
	printWarnings(flags, stderr, taskResult.Warnings)
	printWarnings(flags, stderr, composeResult.Warnings)

	// Determine effective model and its source
	model, modelSource := resolveModel(resolvedModel, env.Agent.DefaultModel)
	if model != "" {
		debugf(flags, dbgTask, "Model: %s (%s)", model, modelSource)
	} else {
		debugf(flags, dbgTask, "Model: agent default (none specified)")
	}

	// Build execution config
	execConfig := orchestration.ExecuteConfig{
		Agent:      env.Agent,
		Model:      resolvedModel,
		Role:       composeResult.Role,
		RoleFile:   composeResult.RoleFile,
		Prompt:     composeResult.Prompt,
		WorkingDir: env.WorkingDir,
		DryRun:     flags.DryRun,
	}

	// Build and log final command
	if flags.Debug {
		cmdStr, err := env.Executor.BuildCommand(execConfig)
		if err == nil {
			debugf(flags, dbgExec, "Final command: %s", cmdStr)
		}
	}

	if flags.DryRun {
		debugf(flags, dbgExec, "Dry-run mode, skipping execution")
		return executeTaskDryRun(stdout, env.Executor, execConfig, composeResult, env.Agent, model, modelSource, resolvedName, instructions)
	}

	// Print execution info
	if !flags.Quiet {
		printTaskExecutionInfo(stdout, env.Agent, model, modelSource, composeResult, resolvedName, instructions, taskResult)
	}

	debugf(flags, dbgExec, "Executing agent (process replacement)")
	// Execute agent (replaces current process)
	return env.Executor.Execute(execConfig)
}

// executeTaskDryRun handles --dry-run mode for tasks.
func executeTaskDryRun(w io.Writer, executor *orchestration.Executor, cfg orchestration.ExecuteConfig, result orchestration.ComposeResult, agent orchestration.Agent, model, modelSource, taskName, instructions string) error {
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
	printTaskDryRunSummary(w, agent, model, modelSource, result, dir, taskName, instructions)

	return nil
}

// printTaskExecutionInfo prints the task execution summary.
func printTaskExecutionInfo(w io.Writer, agent orchestration.Agent, model, modelSource string, result orchestration.ComposeResult, taskName, instructions string, taskResult orchestration.ProcessResult) {
	printHeader(w, fmt.Sprintf("Starting Task: %s", taskName))
	printSeparator(w)

	printAgentModel(w, agent, model, modelSource)
	printContextTable(w, result.Contexts, result.Selection)
	printRoleTable(w, result.RoleResolutions)

	if taskResult.CommandExecuted {
		_, _ = fmt.Fprintln(w, "Command: executed")
	}

	if instructions != "" {
		_, _ = fmt.Fprintf(w, "Instructions:\n%s\n", instructions)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Starting %s - awaiting response...\n", agent.Name)
}

// printTaskDryRunSummary prints the task dry-run summary.
func printTaskDryRunSummary(w io.Writer, agent orchestration.Agent, model, modelSource string, result orchestration.ComposeResult, dir, taskName, instructions string) {
	printHeader(w, fmt.Sprintf("Dry Run - Task: %s", taskName))
	printSeparator(w)

	printAgentModel(w, agent, model, modelSource)
	printContextTable(w, result.Contexts, result.Selection)
	printRoleTable(w, result.RoleResolutions)

	if instructions != "" {
		_, _ = fmt.Fprintf(w, "Instructions:\n%s\n", instructions)
		_, _ = fmt.Fprintln(w)
	}

	// Show role preview
	if result.Role != "" {
		printContentPreview(w, "Role", colorRoles, result.Role, 5)
		_, _ = fmt.Fprintln(w)
	}

	// Show prompt preview
	if result.Prompt != "" {
		printContentPreview(w, "Prompt", colorPrompts, result.Prompt, 5)
		_, _ = fmt.Fprintln(w)
	}

	_, _ = colorDim.Fprint(w, "Files:")
	_, _ = fmt.Fprintf(w, " %s/\n", dir)
	_, _ = fmt.Fprintln(w, "  role.md")
	_, _ = fmt.Fprintln(w, "  prompt.md")
	_, _ = fmt.Fprintln(w, "  command.txt")
}

// hasExactInstalledTask checks if an exact task name exists in the config.
func hasExactInstalledTask(cfg internalcue.LoadResult, name string) bool {
	tasks := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyTasks))
	if !tasks.Exists() {
		return false
	}
	return tasks.LookupPath(cue.MakePath(cue.Str(name))).Exists()
}

// findInstalledTasks finds tasks in the config that match the search term.
// Uses the scoring system to match against name, description, and tags.
// Multiple terms (space or comma separated) use AND logic - all must match.
func findInstalledTasks(cfg internalcue.LoadResult, searchTerm string) ([]TaskMatch, error) {
	results, err := assets.SearchInstalledConfig(cfg.Value, internalcue.KeyTasks, "tasks", searchTerm)
	if err != nil {
		return nil, err
	}
	var matches []TaskMatch
	for _, r := range results {
		matches = append(matches, TaskMatch{
			Name:   r.Name,
			Source: TaskSourceInstalled,
		})
	}
	return matches, nil
}

// findRegistryTasks finds tasks in the registry that match the search term.
// Uses the scoring system to match against name, description, and tags.
// Multiple terms (space or comma separated) use AND logic - all must match.
func findRegistryTasks(index *registry.Index, searchTerm string) ([]TaskMatch, error) {
	results, err := assets.SearchCategoryEntries("tasks", index.Tasks, searchTerm)
	if err != nil {
		return nil, err
	}
	var matches []TaskMatch
	for _, r := range results {
		matches = append(matches, TaskMatch{
			Name:   r.Name,
			Source: TaskSourceRegistry,
			Entry:  r.Entry,
		})
	}
	return matches, nil
}

// mergeTaskMatches combines installed and registry matches, deduplicating by name.
// Installed tasks take precedence over registry tasks with the same name.
// Results are sorted alphabetically by name.
func mergeTaskMatches(installed, registry []TaskMatch) []TaskMatch {
	// Build map of installed task names for deduplication
	installedNames := make(map[string]bool)
	for _, m := range installed {
		installedNames[m.Name] = true
	}

	// Start with installed matches
	merged := make([]TaskMatch, len(installed))
	copy(merged, installed)

	// Add registry matches that aren't already installed
	for _, m := range registry {
		if !installedNames[m.Name] {
			merged = append(merged, m)
		}
	}

	// Sort alphabetically by name
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Name < merged[j].Name
	})

	return merged
}

// promptTaskSelection prompts the user to select a task from multiple matches.
// The caller is responsible for TTY detection; this function assumes interactive input.
// Returns the selected TaskMatch or an error if selection fails.
func promptTaskSelection(w io.Writer, reader *bufio.Reader, matches []TaskMatch, searchTerm string) (TaskMatch, error) {
	totalCount := len(matches)
	displayCount := totalCount
	truncated := false
	if displayCount > maxTaskResults {
		displayCount = maxTaskResults
		truncated = true
	}

	_, _ = fmt.Fprintf(w, "Found %d tasks matching %q:\n\n", totalCount, searchTerm)

	// Find the longest task name for alignment
	maxNameLen := 0
	for i := 0; i < displayCount; i++ {
		if len(matches[i].Name) > maxNameLen {
			maxNameLen = len(matches[i].Name)
		}
	}

	// Display matches with source labels
	for i := 0; i < displayCount; i++ {
		m := matches[i]
		padding := strings.Repeat(" ", maxNameLen-len(m.Name)+2)
		_, _ = fmt.Fprintf(w, "  %2d. %s%s%s\n", i+1, m.Name, padding, m.Source)
	}

	if truncated {
		_, _ = fmt.Fprintf(w, "\nShowing %d of %d matches. Refine search for more specific results.\n", displayCount, totalCount)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Select %s%s%s: ", colorCyan.Sprint("("), colorDim.Sprintf("1-%d", displayCount), colorCyan.Sprint(")"))

	input, err := reader.ReadString('\n')
	if err != nil {
		return TaskMatch{}, fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Try parsing as number
	if choice, err := strconv.Atoi(input); err == nil {
		if choice >= 1 && choice <= displayCount {
			return matches[choice-1], nil
		}
		return TaskMatch{}, fmt.Errorf("invalid selection: %s (choose 1-%d)", input, displayCount)
	}

	// Try matching by name (exact or substring) within displayed matches
	inputLower := strings.ToLower(input)
	for i := 0; i < displayCount; i++ {
		if strings.ToLower(matches[i].Name) == inputLower {
			return matches[i], nil
		}
	}

	// Try substring match on input
	var subMatches []TaskMatch
	for i := 0; i < displayCount; i++ {
		if strings.Contains(strings.ToLower(matches[i].Name), inputLower) {
			subMatches = append(subMatches, matches[i])
		}
	}
	if len(subMatches) == 1 {
		return subMatches[0], nil
	}

	return TaskMatch{}, fmt.Errorf("invalid selection: %s", input)
}

// reloadEnv reloads configuration and rebuilds the execution environment after an asset install.
func reloadEnv(workingDir, agentName string, flags *Flags, stdout io.Writer, stdin io.Reader) (*ExecutionEnv, error) {
	reloadedCfg, err := loadMergedConfigFromDirWithDebug(workingDir, flags)
	if err != nil {
		return nil, fmt.Errorf("reloading configuration: %w", err)
	}
	return buildExecutionEnv(reloadedCfg, workingDir, agentName, flags, stdout, stdin)
}

// installTaskAndReloadEnv installs a task from the registry and reloads the execution environment.
func installTaskAndReloadEnv(stdout io.Writer, stdin io.Reader, flags *Flags, client *registry.Client, index *registry.Index, result assets.SearchResult, workingDir, agentName string) (*ExecutionEnv, error) {
	if err := installTaskFromRegistry(stdout, flags, client, index, result); err != nil {
		return nil, err
	}
	return reloadEnv(workingDir, agentName, flags, stdout, stdin)
}

// installTaskFromRegistry installs a task from the registry using a pre-fetched client and result.
func installTaskFromRegistry(stdout io.Writer, flags *Flags, client *registry.Client, index *registry.Index, result assets.SearchResult) error {
	ctx := context.Background()

	// Install the task using the assets package
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	// Always install to global config for auto-install
	configDir := paths.Global

	// Install the asset
	if err := assets.InstallAsset(ctx, client, index, result, configDir); err != nil {
		return err
	}

	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Installed %s to global config\n\n", result.Name)
	}

	return nil
}

// findExactTaskInRegistry searches for an exact task match in the registry index.
// Supports both full name (e.g., "golang/code-review") and short name (e.g., "code-review").
// Returns an error if multiple entries share the same short name.
func findExactTaskInRegistry(index *registry.Index, taskName string) (*assets.SearchResult, error) {
	// Full name match is always unambiguous
	if entry, ok := index.Tasks[taskName]; ok {
		return &assets.SearchResult{
			Category: "tasks",
			Name:     taskName,
			Entry:    entry,
		}, nil
	}

	// Short name match: collect all matches to detect ambiguity
	var matches []string
	for name := range index.Tasks {
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			if name[idx+1:] == taskName {
				matches = append(matches, name)
			}
		}
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return &assets.SearchResult{
			Category: "tasks",
			Name:     matches[0],
			Entry:    index.Tasks[matches[0]],
		}, nil
	default:
		sort.Strings(matches)
		return nil, fmt.Errorf("ambiguous task name %q matches multiple entries: %s", taskName, strings.Join(matches, ", "))
	}
}
