package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/tui"
	"github.com/spf13/cobra"
)

// addConfigTaskCommand adds the task subcommand group to the config command.
func addConfigTaskCommand(parent *cobra.Command) {
	taskCmd := &cobra.Command{
		Use:     "task",
		Aliases: []string{"tasks"},
		Short:   "Manage task configuration",
		Long: `Manage reusable tasks.

Tasks define reusable workflows that can be executed with 'start task <name>'.
Each task specifies a prompt and optionally a role to use.`,
		RunE: runConfigTask,
	}

	addConfigTaskListCommand(taskCmd)
	addConfigTaskAddCommand(taskCmd)
	addConfigTaskInfoCommand(taskCmd)
	addConfigTaskEditCommand(taskCmd)
	addConfigTaskRemoveCommand(taskCmd)

	parent.AddCommand(taskCmd)
}

// runConfigTask runs list by default, handles help subcommand.
func runConfigTask(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	if len(args) > 0 {
		return unknownCommandError("start config task", args[0])
	}
	return runConfigTaskList(cmd, args)
}

// addConfigTaskListCommand adds the list subcommand.
func addConfigTaskListCommand(parent *cobra.Command) {
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all tasks",
		Long: `List all configured tasks.

Shows tasks from both global and local configuration.
Use --local to show only local tasks.`,
		RunE: runConfigTaskList,
	}

	parent.AddCommand(listCmd)
}

// runConfigTaskList lists all configured tasks.
func runConfigTaskList(cmd *cobra.Command, _ []string) error {
	local := getFlags(cmd).Local
	tasks, order, err := loadTasksForScope(local)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No tasks configured.")
		return nil
	}

	w := cmd.OutOrStdout()
	_, _ = tui.ColorTasks.Fprint(w, "tasks")
	_, _ = fmt.Fprintln(w, "/")
	_, _ = fmt.Fprintln(w)

	for _, name := range order {
		task := tasks[name]
		source := task.Source
		if task.Origin != "" {
			source += ", registry"
		}
		if task.Description != "" {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = tui.ColorDim.Fprint(w, "- "+task.Description+" ")
			_, _ = fmt.Fprintln(w, tui.Annotate("%s", source))
		} else {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = fmt.Fprintln(w, tui.Annotate("%s", source))
		}
	}

	return nil
}

// addConfigTaskAddCommand adds the add subcommand.
func addConfigTaskAddCommand(parent *cobra.Command) {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new task",
		Long: `Add a new task configuration.

Run interactively to be prompted for values.

A task must have exactly one content source: file, command, or prompt.

Examples:
  start config task add
  start config task add --local`,
		Args: noArgsOrHelp,
		RunE: runConfigTaskAdd,
	}

	parent.AddCommand(addCmd)
}

// runConfigTaskAdd adds a new task configuration.
func runConfigTaskAdd(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	stdin := cmd.InOrStdin()
	if !isTerminal(stdin) {
		return fmt.Errorf("interactive add requires a terminal")
	}
	return configTaskAdd(stdin, cmd.OutOrStdout(), getFlags(cmd).Local)
}

// configTaskAdd is the inner add logic for tasks.
func configTaskAdd(stdin io.Reader, stdout io.Writer, local bool) error {
	// Collect values
	name, err := promptString(stdout, stdin, "Task name", "")
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("task name is required")
	}

	description, err := promptString(stdout, stdin, "Description (optional)", "")
	if err != nil {
		return err
	}

	// Content source
	file, command, prompt, err := promptContentSource(stdout, stdin, "3", "")
	if err != nil {
		return err
	}

	// Validate content source
	sourceCount := 0
	if file != "" {
		sourceCount++
	}
	if command != "" {
		sourceCount++
	}
	if prompt != "" {
		sourceCount++
	}

	if sourceCount == 0 {
		return fmt.Errorf("must specify one of: file, command, or prompt")
	}
	if sourceCount > 1 {
		return fmt.Errorf("specify only one of: file, command, or prompt")
	}

	// Role (optional)
	role, err := promptString(stdout, stdin, "Role (optional)", "")
	if err != nil {
		return err
	}

	// Build task struct (tags are empty on add; use edit to set them)
	task := TaskConfig{
		Name:        name,
		Description: description,
		File:        file,
		Command:     command,
		Prompt:      prompt,
		Role:        role,
	}

	// Determine target directory
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(local)
	scopeName := scopeString(local)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Load existing tasks from target directory
	existingTasks, _, err := loadTasksFromDir(configDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("loading existing tasks: %w", err)
	}

	// Check for duplicate
	if _, exists := existingTasks[name]; exists {
		return fmt.Errorf("task %q already exists in %s config", name, scopeName)
	}

	// Add new task
	existingTasks[name] = task

	// Write tasks file
	taskPath := filepath.Join(configDir, "tasks.cue")
	if err := writeTasksFile(taskPath, existingTasks); err != nil {
		return fmt.Errorf("writing tasks file: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "Added task %q to %s config\n", name, scopeName)
	_, _ = fmt.Fprintf(stdout, "Config: %s\n", taskPath)

	return nil
}

// addConfigTaskInfoCommand adds the info subcommand.
func addConfigTaskInfoCommand(parent *cobra.Command) {
	infoCmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show task details",
		Long: `Show detailed information about a task.

Displays all configuration fields for the specified task.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigTaskInfo,
	}

	parent.AddCommand(infoCmd)
}

// runConfigTaskInfo shows detailed information about a task.
func runConfigTaskInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	local := getFlags(cmd).Local

	tasks, _, err := loadTasksForScope(local)
	if err != nil {
		return err
	}

	resolvedName, task, err := resolveInstalledName(tasks, "task", name)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(w)
	_, _ = tui.ColorTasks.Fprint(w, "tasks")
	_, _ = fmt.Fprintf(w, "/%s\n", resolvedName)
	printSeparator(w)

	_, _ = tui.ColorDim.Fprint(w, "Source:")
	_, _ = fmt.Fprintf(w, " %s\n", task.Source)
	if task.Origin != "" {
		_, _ = tui.ColorDim.Fprint(w, "Origin:")
		_, _ = fmt.Fprintf(w, " %s\n", task.Origin)
	}
	if task.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = tui.ColorDim.Fprint(w, "Description:")
		_, _ = fmt.Fprintf(w, " %s\n", task.Description)
	}
	if task.File != "" {
		_, _ = tui.ColorDim.Fprint(w, "File:")
		_, _ = fmt.Fprintf(w, " %s\n", task.File)
	}
	if task.Command != "" {
		_, _ = tui.ColorDim.Fprint(w, "Command:")
		_, _ = fmt.Fprintf(w, " %s\n", task.Command)
	}
	if task.Prompt != "" {
		_, _ = tui.ColorDim.Fprint(w, "Prompt:")
		_, _ = fmt.Fprintf(w, " %s\n", truncatePrompt(task.Prompt, 100))
	}
	if task.Role != "" {
		_, _ = tui.ColorDim.Fprint(w, "Role:")
		_, _ = fmt.Fprintf(w, " %s\n", task.Role)
	}
	if len(task.Tags) > 0 {
		_, _ = tui.ColorDim.Fprint(w, "Tags:")
		_, _ = fmt.Fprintf(w, " %s\n", strings.Join(task.Tags, ", "))
	}
	printSeparator(w)

	return nil
}

// addConfigTaskEditCommand adds the edit subcommand.
func addConfigTaskEditCommand(parent *cobra.Command) {
	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit task configuration",
		Long: `Edit task configuration.

Without a name, opens the tasks.cue file in $EDITOR.
With a name, prompts interactively for values.

Examples:
  start config task edit
  start config task edit review`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigTaskEdit,
	}

	parent.AddCommand(editCmd)
}

// runConfigTaskEdit edits a task configuration.
func runConfigTaskEdit(cmd *cobra.Command, args []string) error {
	local := getFlags(cmd).Local
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}
	configDir := paths.Dir(local)
	taskPath := filepath.Join(configDir, "tasks.cue")

	// No name: open file in editor
	if len(args) == 0 {
		return openInEditor(taskPath)
	}

	stdin := cmd.InOrStdin()
	if !isTerminal(stdin) {
		return fmt.Errorf("interactive edit requires a terminal")
	}
	return configTaskEdit(stdin, cmd.OutOrStdout(), local, args[0])
}

// configTaskEdit is the inner edit logic for tasks.
func configTaskEdit(stdin io.Reader, stdout io.Writer, local bool, name string) error {
	// Determine target directory
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}
	configDir := paths.Dir(local)
	taskPath := filepath.Join(configDir, "tasks.cue")

	// Load existing tasks
	tasks, _, err := loadTasksFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	resolvedName, task, err := resolveInstalledName(tasks, "task", name)
	if err != nil {
		return err
	}

	// Prompt for each field with current value as default
	_, _ = fmt.Fprintf(stdout, "Editing task %q %s\n\n", resolvedName, tui.Annotate("press Enter to keep current value"))

	newDescription, err := promptString(stdout, stdin, "Description", task.Description)
	if err != nil {
		return err
	}

	// For content source, show current and allow change
	_, _ = fmt.Fprintln(stdout, "\nCurrent content source:")
	if task.File != "" {
		_, _ = fmt.Fprintf(stdout, "  File: %s\n", task.File)
	}
	if task.Command != "" {
		_, _ = fmt.Fprintf(stdout, "  Command: %s\n", task.Command)
	}
	if task.Prompt != "" {
		_, _ = fmt.Fprintf(stdout, "  Prompt: %s\n", truncatePrompt(task.Prompt, 50))
	}

	_, _ = fmt.Fprintf(stdout, "Keep current? %s ", tui.Bracket("Y/n"))
	reader := bufio.NewReader(stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))

	newFile := task.File
	newCommand := task.Command
	newPrompt := task.Prompt

	if input == "n" || input == "no" {
		newFile, newCommand, newPrompt, err = promptContentSource(stdout, stdin, "3", task.Prompt)
		if err != nil {
			return err
		}
	}

	// Role
	newRole, err := promptString(stdout, stdin, "Role", task.Role)
	if err != nil {
		return err
	}

	// Prompt for tags
	_, _ = fmt.Fprintln(stdout)
	newTags, err := promptTags(stdout, stdin, task.Tags)
	if err != nil {
		return err
	}

	// Update task
	task.Description = newDescription
	task.File = newFile
	task.Command = newCommand
	task.Prompt = newPrompt
	task.Role = newRole
	task.Tags = newTags
	tasks[resolvedName] = task

	// Write updated file
	if err := writeTasksFile(taskPath, tasks); err != nil {
		return fmt.Errorf("writing tasks file: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "\nUpdated task %q\n", resolvedName)
	return nil
}

// addConfigTaskRemoveCommand adds the remove subcommand.
func addConfigTaskRemoveCommand(parent *cobra.Command) {
	removeCmd := &cobra.Command{
		Use:     "remove <name>...",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove one or more tasks",
		Long: `Remove one or more task configurations.

Removes the specified tasks from the configuration file.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runConfigTaskRemove,
	}

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(removeCmd)
}

// runConfigTaskRemove removes one or more task configurations.
func runConfigTaskRemove(cmd *cobra.Command, args []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(local)

	// Load existing tasks
	tasks, _, err := loadTasksFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	skipConfirm, _ := cmd.Flags().GetBool("yes")

	resolvedNames, err := resolveRemoveNames(tasks, "task", args, skipConfirm, stdout, stdin)
	if err != nil {
		return err
	}
	if resolvedNames == nil {
		return nil // user cancelled
	}

	// Confirm removal unless --yes flag is set
	if !skipConfirm {
		confirmed, err := confirmMultiRemoval(stdout, stdin, "task", resolvedNames, local)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	// Remove all tasks
	for _, name := range resolvedNames {
		delete(tasks, name)
	}

	// Write updated file once
	taskPath := filepath.Join(configDir, "tasks.cue")
	if err := writeTasksFile(taskPath, tasks); err != nil {
		return fmt.Errorf("writing tasks file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		for _, name := range resolvedNames {
			_, _ = fmt.Fprintf(stdout, "Removed task %q\n", name)
		}
	}

	return nil
}

// TaskConfig represents a task configuration for editing.
type TaskConfig struct {
	Name        string
	Description string
	File        string
	Command     string
	Prompt      string
	Role        string
	Tags        []string
	Source      string // "global" or "local" - for display only
	Origin      string // Registry module path when installed from registry
}

// loadTasksForScope loads tasks from the appropriate scope.
// Returns the tasks map, names in definition order, and any error.
func loadTasksForScope(localOnly bool) (map[string]TaskConfig, []string, error) {
	return loadForScope(localOnly, loadTasksFromDir, func(t *TaskConfig, s string) { t.Source = s })
}

// loadTasksFromDir loads tasks from a specific directory.
// Returns the tasks map, names in definition order, and any error.
func loadTasksFromDir(dir string) (map[string]TaskConfig, []string, error) {
	tasks := make(map[string]TaskConfig)
	var order []string

	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(dir)
	if err != nil {
		// If no CUE files exist, return empty map (not an error)
		if errors.Is(err, internalcue.ErrNoCUEFiles) {
			return tasks, order, nil
		}
		return tasks, order, err
	}

	tasksVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyTasks))
	if !tasksVal.Exists() {
		return tasks, order, nil
	}

	iter, err := tasksVal.Fields()
	if err != nil {
		return nil, nil, fmt.Errorf("iterating tasks: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		val := iter.Value()

		task := TaskConfig{Name: name}

		if v := val.LookupPath(cue.ParsePath("description")); v.Exists() {
			task.Description, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("file")); v.Exists() {
			task.File, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("command")); v.Exists() {
			task.Command, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("prompt")); v.Exists() {
			task.Prompt, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("role")); v.Exists() {
			task.Role, _ = v.String()
		}

		task.Tags = extractTags(val)

		// Load origin (registry provenance)
		if v := val.LookupPath(cue.ParsePath("origin")); v.Exists() {
			task.Origin, _ = v.String()
		}

		tasks[name] = task
		order = append(order, name)
	}

	return tasks, order, nil
}

// writeTasksFile writes the tasks configuration to a file.
// Tasks are not order-dependent, so names are sorted alphabetically for consistent output.
func writeTasksFile(path string, tasks map[string]TaskConfig) error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your task configuration\n\n")
	sb.WriteString("tasks: {\n")

	// Sort task names for consistent output
	var names []string
	for name := range tasks {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		task := tasks[name]
		sb.WriteString(fmt.Sprintf("\t%q: {\n", name))

		// Write origin first if present (registry provenance)
		if task.Origin != "" {
			sb.WriteString(fmt.Sprintf("\t\torigin: %q\n", task.Origin))
		}
		if task.Description != "" {
			sb.WriteString(fmt.Sprintf("\t\tdescription: %q\n", task.Description))
		}
		if task.File != "" {
			sb.WriteString(fmt.Sprintf("\t\tfile: %q\n", task.File))
		}
		if task.Command != "" {
			sb.WriteString(fmt.Sprintf("\t\tcommand: %q\n", task.Command))
		}
		writeCUEPrompt(&sb, task.Prompt)
		if task.Role != "" {
			sb.WriteString(fmt.Sprintf("\t\trole: %q\n", task.Role))
		}
		writeCUETags(&sb, task.Tags)

		sb.WriteString("\t}\n")
	}

	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
