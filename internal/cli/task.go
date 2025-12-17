package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"cuelang.org/go/cue"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/shell"
	"github.com/grantcarthew/start/internal/temp"
	"github.com/spf13/cobra"
)

// addTaskCommand adds the task command to the parent command.
func addTaskCommand(parent *cobra.Command) {
	taskCmd := &cobra.Command{
		Use:   "task [name] [instructions]",
		Short: "Run a predefined task",
		Long: `Run a predefined task with optional instructions.

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

	return executeTask(cmd.OutOrStdout(), cmd.ErrOrStderr(), taskName, instructions)
}

// executeTask handles task execution.
func executeTask(stdout, stderr io.Writer, taskName, instructions string) error {
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

	// Create shell runner and template processor
	shellRunner := shell.NewRunner()
	processor := orchestration.NewTemplateProcessor(nil, shellRunner, workingDir)
	composer := orchestration.NewComposer(processor, workingDir)

	// Resolve task
	taskResult, err := composer.ResolveTask(cfg.Value, taskName, instructions)
	if err != nil {
		return fmt.Errorf("resolving task: %w", err)
	}

	// Get task's role if specified, else use flag or default
	roleName := flagRole
	if roleName == "" {
		roleName = orchestration.GetTaskRole(cfg.Value, taskName)
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

	// Per DR-015: required contexts only for tasks
	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: false,
		Tags:            flagContext,
	}

	// Compose contexts and resolve role
	composeResult, err := composer.ComposeWithRole(cfg.Value, selection, roleName, taskResult.Content, "")
	if err != nil {
		return fmt.Errorf("composing prompt: %w", err)
	}

	// Print warnings
	for _, w := range taskResult.Warnings {
		if !flagQuiet {
			fmt.Fprintf(stderr, "Warning: %s\n", w)
		}
	}
	for _, w := range composeResult.Warnings {
		if !flagQuiet {
			fmt.Fprintf(stderr, "Warning: %s\n", w)
		}
	}

	// Build execution config
	execConfig := orchestration.ExecuteConfig{
		Agent:      agent,
		Model:      flagModel,
		Role:       composeResult.Role,
		Prompt:     composeResult.Prompt,
		WorkingDir: workingDir,
		DryRun:     flagDryRun,
	}

	executor := orchestration.NewExecutor(workingDir)

	if flagDryRun {
		return executeTaskDryRun(stdout, executor, execConfig, composeResult, agent, taskName, instructions)
	}

	// Print execution info
	if !flagQuiet {
		printTaskExecutionInfo(stdout, agent, flagModel, composeResult, taskName, instructions, taskResult)
	}

	// Execute agent (replaces current process)
	return executor.Execute(execConfig)
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
	fmt.Fprintf(w, "Starting Task: %s\n", taskName)
	fmt.Fprintln(w, strings.Repeat("─", 79))

	modelStr := model
	if modelStr == "" {
		modelStr = agent.DefaultModel
	}
	fmt.Fprintf(w, "Agent: %s (model: %s)\n", agent.Name, modelStr)
	fmt.Fprintln(w)

	if len(result.Contexts) > 0 {
		fmt.Fprintln(w, "Context documents (required only):")
		for _, ctx := range result.Contexts {
			fmt.Fprintf(w, "  ✓ %s\n", ctx.Name)
		}
		fmt.Fprintln(w)
	}

	if result.RoleName != "" {
		fmt.Fprintf(w, "Role: %s\n", result.RoleName)
	}

	if taskResult.CommandExecuted {
		fmt.Fprintln(w, "Command: executed")
	}

	if instructions != "" {
		fmt.Fprintf(w, "Instructions: %s\n", instructions)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Executing...")
}

// printTaskDryRunSummary prints the task dry-run summary.
func printTaskDryRunSummary(w io.Writer, agent orchestration.Agent, model string, result orchestration.ComposeResult, dir, taskName, instructions string) {
	fmt.Fprintf(w, "Dry Run - Task: %s\n", taskName)
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

	if instructions != "" {
		fmt.Fprintf(w, "Instructions: %s\n", instructions)
	}
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

// findTask attempts to find a task by exact name or prefix match.
func findTask(cfg internalcue.LoadResult, name string) (string, error) {
	tasks := cfg.Value.LookupPath(cue.ParsePath("tasks"))
	if !tasks.Exists() {
		return "", fmt.Errorf("no tasks defined")
	}

	// Try exact match first
	if tasks.LookupPath(cue.MakePath(cue.Str(name))).Exists() {
		return name, nil
	}

	// Try prefix match
	var matches []string
	iter, err := tasks.Fields()
	if err != nil {
		return "", fmt.Errorf("reading tasks: %w", err)
	}

	for iter.Next() {
		taskName := iter.Selector().Unquoted()
		if strings.HasPrefix(taskName, name) {
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
