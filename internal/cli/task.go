package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/grantcarthew/start/internal/temp"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
		Short:   "Run a predefined task",
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
	return executeTask(cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd.InOrStdin(), flags, taskName, instructions)
}

// executeTask handles task execution.
func executeTask(stdout, stderr io.Writer, stdin io.Reader, flags *Flags, taskName, instructions string) error {
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
		// Per DR-015: Unified task resolution across installed config and registry

		// Step 1: Check for exact match in installed config (fast path, no registry fetch)
		if hasExactInstalledTask(env.Cfg, taskName) {
			debugf(flags, "task", "Exact match found in installed config")
			resolvedName = taskName
		} else {
			// Step 2: No exact match - fetch registry and combine results
			debugf(flags, "task", "No exact installed match, fetching registry index...")

			ctx := context.Background()
			client, err := registry.NewClient()
			if err != nil {
				return fmt.Errorf("creating registry client: %w", err)
			}

			index, err := client.FetchIndex(ctx)
			if err != nil {
				return fmt.Errorf("fetching registry index: %w", err)
			}

			// Check for exact match in registry
			exactRegistry := findExactTaskInRegistry(index, taskName)
			if exactRegistry != nil {
				debugf(flags, "task", "Exact match found in registry: %s", exactRegistry.Name)
				// Install and run
				if !flags.Quiet {
					_, _ = fmt.Fprintf(stdout, "Installing %s from registry...\n", exactRegistry.Name)
				}
				if err := installTaskFromRegistry(stdout, flags, client, *exactRegistry); err != nil {
					return err
				}
				// Reload config
				env, err = prepareExecutionEnv(flags)
				if err != nil {
					return err
				}
				resolvedName = exactRegistry.Name
			} else {
				// Step 3: Substring match across installed + registry
				installedMatches := findInstalledTasks(env.Cfg, taskName)
				registryMatches := findRegistryTasks(index, taskName)
				allMatches := mergeTaskMatches(installedMatches, registryMatches)

				debugf(flags, "task", "Found %d installed matches, %d registry matches, %d total",
					len(installedMatches), len(registryMatches), len(allMatches))

				switch len(allMatches) {
				case 0:
					return fmt.Errorf("task %q not found", taskName)
				case 1:
					// Single match - use it
					match := allMatches[0]
					if match.Source == TaskSourceRegistry {
						// Install from registry first
						if !flags.Quiet {
							_, _ = fmt.Fprintf(stdout, "Installing %s from registry...\n", match.Name)
						}
						result := &SearchResult{
							Category: "tasks",
							Name:     match.Name,
							Entry:    match.Entry,
						}
						if err := installTaskFromRegistry(stdout, flags, client, *result); err != nil {
							return err
						}
						// Reload config
						env, err = prepareExecutionEnv(flags)
						if err != nil {
							return err
						}
					}
					resolvedName = match.Name
				default:
					// Multiple matches - interactive selection
					debugf(flags, "task", "Multiple matches, prompting for selection")
					selected, err := promptTaskSelection(stdout, stdin, allMatches, taskName)
					if err != nil {
						return err
					}

					if selected.Source == TaskSourceRegistry {
						// Install from registry first
						if !flags.Quiet {
							_, _ = fmt.Fprintf(stdout, "Installing %s from registry...\n", selected.Name)
						}
						result := &SearchResult{
							Category: "tasks",
							Name:     selected.Name,
							Entry:    selected.Entry,
						}
						if err := installTaskFromRegistry(stdout, flags, client, *result); err != nil {
							return err
						}
						// Reload config
						env, err = prepareExecutionEnv(flags)
						if err != nil {
							return err
						}
					}
					resolvedName = selected.Name
				}
			}
		}

		if resolvedName != taskName {
			debugf(flags, "task", "Resolved to %q", resolvedName)
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
	composeResult, composeErr := env.Composer.ComposeWithRole(env.Cfg.Value, selection, roleName, taskResult.Content, "")
	if composeErr != nil {
		// Show UI with role resolutions before returning error
		if !flags.Quiet && len(composeResult.RoleResolutions) > 0 {
			printComposeError(stdout, env.Agent, composeResult)
		}
		return fmt.Errorf("composing prompt: %w", composeErr)
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
		RoleFile:   composeResult.RoleFile,
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

	PrintRoleTable(w, result.RoleResolutions)

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
	_, _ = fmt.Fprintln(w)

	PrintContextTable(w, result.Contexts)

	PrintRoleTable(w, result.RoleResolutions)

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

// hasExactInstalledTask checks if an exact task name exists in the config.
func hasExactInstalledTask(cfg internalcue.LoadResult, name string) bool {
	tasks := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyTasks))
	if !tasks.Exists() {
		return false
	}
	return tasks.LookupPath(cue.MakePath(cue.Str(name))).Exists()
}

// findInstalledTasks finds tasks in the config that match the search term (substring match).
func findInstalledTasks(cfg internalcue.LoadResult, searchTerm string) []TaskMatch {
	var matches []TaskMatch

	tasks := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyTasks))
	if !tasks.Exists() {
		return matches
	}

	iter, err := tasks.Fields()
	if err != nil {
		return matches
	}

	for iter.Next() {
		taskName := iter.Selector().Unquoted()
		if strings.Contains(taskName, searchTerm) {
			matches = append(matches, TaskMatch{
				Name:   taskName,
				Source: TaskSourceInstalled,
			})
		}
	}

	return matches
}

// findRegistryTasks finds tasks in the registry that match the search term (substring match).
func findRegistryTasks(index *registry.Index, searchTerm string) []TaskMatch {
	var matches []TaskMatch

	for name, entry := range index.Tasks {
		if strings.Contains(name, searchTerm) {
			matches = append(matches, TaskMatch{
				Name:   name,
				Source: TaskSourceRegistry,
				Entry:  entry,
			})
		}
	}

	return matches
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

// findTask attempts to find a task by exact name or substring match (installed only).
// Returns (resolvedName, nil, nil) for exact/unique match.
// Returns ("", matches, nil) when multiple tasks match (caller should prompt for selection).
// Returns ("", nil, error) for actual errors (no tasks defined, task not found, etc.).
// Note: This function is kept for backward compatibility with tests.
func findTask(cfg internalcue.LoadResult, name string) (string, []string, error) {
	tasks := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyTasks))
	if !tasks.Exists() {
		return "", nil, fmt.Errorf("no tasks defined")
	}

	// Try exact match first
	if tasks.LookupPath(cue.MakePath(cue.Str(name))).Exists() {
		return name, nil, nil
	}

	// Try substring match
	var matches []string
	iter, err := tasks.Fields()
	if err != nil {
		return "", nil, fmt.Errorf("reading tasks: %w", err)
	}

	for iter.Next() {
		taskName := iter.Selector().Unquoted()
		if strings.Contains(taskName, name) {
			matches = append(matches, taskName)
		}
	}

	switch len(matches) {
	case 0:
		return "", nil, fmt.Errorf("task %q not found", name)
	case 1:
		return matches[0], nil, nil
	default:
		// Multiple matches - return them for interactive selection
		return "", matches, nil
	}
}

// promptTaskSelection prompts the user to select a task from multiple matches.
// Returns the selected TaskMatch or an error if selection fails.
func promptTaskSelection(w io.Writer, r io.Reader, matches []TaskMatch, searchTerm string) (TaskMatch, error) {
	// Check if stdin is a TTY
	isTTY := false
	if f, ok := r.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	if !isTTY {
		var names []string
		for _, m := range matches {
			names = append(names, m.Name)
		}
		return TaskMatch{}, fmt.Errorf("ambiguous task %q matches: %s\nSpecify exact name or run interactively", searchTerm, strings.Join(names, ", "))
	}

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
	_, _ = fmt.Fprintf(w, "Select (1-%d): ", displayCount)

	reader := bufio.NewReader(r)
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

// installTaskFromRegistry installs a task from the registry using a pre-fetched client and result.
func installTaskFromRegistry(stdout io.Writer, flags *Flags, client *registry.Client, result SearchResult) error {
	ctx := context.Background()

	// Install the task using the same logic as assets add
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
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
		return fmt.Errorf("resolving asset version: %w", err)
	}

	fetchResult, err := client.Fetch(ctx, resolvedPath)
	if err != nil {
		return fmt.Errorf("fetching asset module: %w", err)
	}

	// Extract asset content from fetched module
	originPath := modulePath
	if idx := strings.Index(originPath, "@"); idx != -1 {
		originPath = originPath[:idx]
	}
	assetContent, err := extractAssetContent(fetchResult.SourceDir, result, client.Registry(), originPath)
	if err != nil {
		return fmt.Errorf("extracting asset content: %w", err)
	}

	// Determine the config file
	configFile := assetTypeToConfigFile(result.Category)
	configPath := configDir + "/" + configFile

	// Write the asset to config
	if err := writeAssetToConfig(configPath, result, assetContent, modulePath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Installed %s to global config\n\n", result.Name)
	}

	return nil
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
