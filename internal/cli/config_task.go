// NOTE(design): This file shares structural patterns with config_agent.go,
// config_role.go, and config_context.go (CUE field extraction, scope-aware loading,
// interactive prompting, CUE file generation). This duplication is accepted - each
// entity has distinct fields and behaviours that make a generic abstraction more
// complex than the repetition it would eliminate.
package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
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
	tasks, err := loadTasksForScope(local)
	if err != nil {
		return err
	}

	if len(tasks) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No tasks configured.")
		return nil
	}

	w := cmd.OutOrStdout()
	_, _ = colorTasks.Fprint(w, "tasks")
	_, _ = fmt.Fprintln(w, "/")
	_, _ = fmt.Fprintln(w)

	// Sort task names for consistent output
	var names []string
	for name := range tasks {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		task := tasks[name]
		source := task.Source
		if task.Origin != "" {
			source += ", registry"
		}
		if task.Description != "" {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = colorDim.Fprint(w, "- "+task.Description+" ")
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, source)
			_, _ = colorCyan.Fprintln(w, ")")
		} else {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, source)
			_, _ = colorCyan.Fprintln(w, ")")
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

Provide task details via flags or run interactively to be prompted for values.

A task must have exactly one content source: file, command, or prompt.

Examples:
  start config task add
  start config task add --name review --prompt "Review this code for bugs"
  start config task add --name commit --file ~/.config/start/tasks/commit.md --role git-expert`,
		Args: cobra.NoArgs,
		RunE: runConfigTaskAdd,
	}

	addCmd.Flags().String("name", "", "Task name (identifier)")
	addCmd.Flags().String("description", "", "Description")
	addCmd.Flags().String("file", "", "Path to task prompt file")
	addCmd.Flags().String("command", "", "Command to generate prompt")
	addCmd.Flags().String("prompt", "", "Inline prompt text")
	addCmd.Flags().String("role", "", "Role to use for this task")
	addCmd.Flags().StringSlice("tag", nil, "Tags")

	parent.AddCommand(addCmd)
}

// runConfigTaskAdd adds a new task configuration.
func runConfigTaskAdd(cmd *cobra.Command, _ []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	// Check if interactive - only prompt for optional fields if no flags provided
	isTTY := isTerminal(stdin)
	// If any flags are set, skip prompts for optional fields
	hasFlags := anyFlagChanged(cmd, "name", "description", "file", "command", "prompt", "role", "tag")
	interactive := isTTY && !hasFlags

	// Collect values
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		if !isTTY {
			return fmt.Errorf("--name is required (run interactively or provide flag)")
		}
		var err error
		name, err = promptString(stdout, stdin, "Task name", "")
		if err != nil {
			return err
		}
	}
	if name == "" {
		return fmt.Errorf("task name is required")
	}

	description, _ := cmd.Flags().GetString("description")
	if description == "" && interactive {
		var err error
		description, err = promptString(stdout, stdin, "Description (optional)", "")
		if err != nil {
			return err
		}
	}

	// Content source
	file, _ := cmd.Flags().GetString("file")
	command, _ := cmd.Flags().GetString("command")
	prompt, _ := cmd.Flags().GetString("prompt")

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

	if sourceCount == 0 && interactive {
		_, _ = fmt.Fprintf(stdout, "\nContent source %s%s%s:\n", colorCyan.Sprint("("), colorDim.Sprint("choose one"), colorCyan.Sprint(")"))
		_, _ = fmt.Fprintln(stdout, "  1. File path")
		_, _ = fmt.Fprintln(stdout, "  2. Command")
		_, _ = fmt.Fprintln(stdout, "  3. Inline prompt")
		_, _ = fmt.Fprintf(stdout, "Choice %s%s%s: ", colorCyan.Sprint("["), colorDim.Sprint("3"), colorCyan.Sprint("]"))

		reader := bufio.NewReader(stdin)
		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		choice = strings.TrimSpace(choice)
		if choice == "" {
			choice = "3" // Default to inline prompt for tasks
		}

		switch choice {
		case "1":
			file, err = promptString(stdout, stdin, "File path", "")
			if err != nil {
				return err
			}
		case "2":
			command, err = promptString(stdout, stdin, "Command", "")
			if err != nil {
				return err
			}
		case "3":
			prompt, err = promptText(stdout, stdin, "Prompt text", "")
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid choice: %s", choice)
		}
	}

	// Validate content source
	sourceCount = 0
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
		return fmt.Errorf("must specify one of: --file, --command, or --prompt")
	}
	if sourceCount > 1 {
		return fmt.Errorf("specify only one of: --file, --command, or --prompt")
	}

	// Role (optional)
	role, _ := cmd.Flags().GetString("role")
	if role == "" && interactive {
		var err error
		role, err = promptString(stdout, stdin, "Role (optional)", "")
		if err != nil {
			return err
		}
	}

	// Build task struct
	tags, _ := cmd.Flags().GetStringSlice("tag")
	task := TaskConfig{
		Name:        name,
		Description: description,
		File:        file,
		Command:     command,
		Prompt:      prompt,
		Role:        role,
		Tags:        tags,
	}

	// Determine target directory
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	var scopeName string
	if local {
		configDir = paths.Local
		scopeName = "local"
	} else {
		configDir = paths.Global
		scopeName = "global"
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Load existing tasks from target directory
	existingTasks, err := loadTasksFromDir(configDir)
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

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Added task %q to %s config\n", name, scopeName)
		_, _ = fmt.Fprintf(stdout, "Config: %s\n", taskPath)
	}

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

	tasks, err := loadTasksForScope(local)
	if err != nil {
		return err
	}

	resolvedName, task, err := resolveInstalledName(tasks, "task", name)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(w)
	_, _ = colorTasks.Fprint(w, "tasks")
	_, _ = fmt.Fprintf(w, "/%s\n", resolvedName)
	printSeparator(w)

	_, _ = colorDim.Fprint(w, "Source:")
	_, _ = fmt.Fprintf(w, " %s\n", task.Source)
	if task.Origin != "" {
		_, _ = colorDim.Fprint(w, "Origin:")
		_, _ = fmt.Fprintf(w, " %s\n", task.Origin)
	}
	if task.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = colorDim.Fprint(w, "Description:")
		_, _ = fmt.Fprintf(w, " %s\n", task.Description)
	}
	if task.File != "" {
		_, _ = colorDim.Fprint(w, "File:")
		_, _ = fmt.Fprintf(w, " %s\n", task.File)
	}
	if task.Command != "" {
		_, _ = colorDim.Fprint(w, "Command:")
		_, _ = fmt.Fprintf(w, " %s\n", task.Command)
	}
	if task.Prompt != "" {
		_, _ = colorDim.Fprint(w, "Prompt:")
		_, _ = fmt.Fprintf(w, " %s\n", truncatePrompt(task.Prompt, 100))
	}
	if task.Role != "" {
		_, _ = colorDim.Fprint(w, "Role:")
		_, _ = fmt.Fprintf(w, " %s\n", task.Role)
	}
	if len(task.Tags) > 0 {
		_, _ = colorDim.Fprint(w, "Tags:")
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
With a name and flags, updates only the specified fields.
With a name and no flags in a terminal, provides interactive prompts.

Examples:
  start config task edit
  start config task edit review --prompt "Review this code for bugs"
  start config task edit commit --role git-expert`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigTaskEdit,
	}

	editCmd.Flags().String("description", "", "Description")
	editCmd.Flags().String("file", "", "Path to task prompt file")
	editCmd.Flags().String("command", "", "Command to generate prompt")
	editCmd.Flags().String("prompt", "", "Inline prompt text")
	editCmd.Flags().String("role", "", "Role to use for this task")
	editCmd.Flags().StringSlice("tag", nil, "Tags")

	parent.AddCommand(editCmd)
}

// runConfigTaskEdit edits a task configuration.
func runConfigTaskEdit(cmd *cobra.Command, args []string) error {
	local := getFlags(cmd).Local
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	if local {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

	taskPath := filepath.Join(configDir, "tasks.cue")

	// No name: open file in editor
	if len(args) == 0 {
		return openInEditor(taskPath)
	}

	// Named edit
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	// Load existing tasks
	tasks, err := loadTasksFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	resolvedName, task, err := resolveInstalledName(tasks, "task", name)
	if err != nil {
		return err
	}

	// Check if any edit flags are provided
	hasEditFlags := anyFlagChanged(cmd, "description", "file", "command", "prompt", "role", "tag")

	if hasEditFlags {
		// Non-interactive flag-based update
		if cmd.Flags().Changed("description") {
			task.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("file") {
			task.File, _ = cmd.Flags().GetString("file")
		}
		if cmd.Flags().Changed("command") {
			task.Command, _ = cmd.Flags().GetString("command")
		}
		if cmd.Flags().Changed("prompt") {
			task.Prompt, _ = cmd.Flags().GetString("prompt")
		}
		if cmd.Flags().Changed("role") {
			task.Role, _ = cmd.Flags().GetString("role")
		}
		if cmd.Flags().Changed("tag") {
			task.Tags, _ = cmd.Flags().GetStringSlice("tag")
		}

		tasks[resolvedName] = task

		if err := writeTasksFile(taskPath, tasks); err != nil {
			return fmt.Errorf("writing tasks file: %w", err)
		}

		flags := getFlags(cmd)
		if !flags.Quiet {
			_, _ = fmt.Fprintf(stdout, "Updated task %q\n", resolvedName)
		}
		return nil
	}

	// No flags: require TTY for interactive editing
	isTTY := isTerminal(stdin)
	if !isTTY {
		return fmt.Errorf("interactive editing requires a terminal")
	}

	// Prompt for each field with current value as default
	_, _ = fmt.Fprintf(stdout, "Editing task %q %s%s%s\n\n", resolvedName, colorCyan.Sprint("("), colorDim.Sprint("press Enter to keep current value"), colorCyan.Sprint(")"))

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

	_, _ = fmt.Fprintf(stdout, "Keep current? %s%s%s ", colorCyan.Sprint("["), colorDim.Sprint("Y/n"), colorCyan.Sprint("]"))
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
		newFile = ""
		newCommand = ""
		newPrompt = ""

		_, _ = fmt.Fprintln(stdout, "\nNew content source:")
		_, _ = fmt.Fprintln(stdout, "  1. File path")
		_, _ = fmt.Fprintln(stdout, "  2. Command")
		_, _ = fmt.Fprintln(stdout, "  3. Inline prompt")
		_, _ = fmt.Fprintf(stdout, "Choice %s%s%s: ", colorCyan.Sprint("["), colorDim.Sprint("3"), colorCyan.Sprint("]"))

		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		choice = strings.TrimSpace(choice)
		if choice == "" {
			choice = "3"
		}

		switch choice {
		case "1":
			newFile, err = promptString(stdout, stdin, "File path", "")
			if err != nil {
				return err
			}
		case "2":
			newCommand, err = promptString(stdout, stdin, "Command", "")
			if err != nil {
				return err
			}
		case "3":
			newPrompt, err = promptText(stdout, stdin, "Prompt text", task.Prompt)
			if err != nil {
				return err
			}
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
		Use:     "remove <name>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a task",
		Long: `Remove a task configuration.

Removes the specified task from the configuration file.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigTaskRemove,
	}

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(removeCmd)
}

// runConfigTaskRemove removes a task configuration.
func runConfigTaskRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	if local {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

	// Load existing tasks
	tasks, err := loadTasksFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	resolvedName, _, err := resolveInstalledName(tasks, "task", name)
	if err != nil {
		return err
	}

	// Confirm removal unless --yes flag is set
	skipConfirm, _ := cmd.Flags().GetBool("yes")
	if !skipConfirm {
		isTTY := isTerminal(stdin)

		if isTTY {
			_, _ = fmt.Fprintf(stdout, "Remove task %q from %s config? %s%s%s ", resolvedName, scopeString(local), colorCyan.Sprint("["), colorDim.Sprint("y/N"), colorCyan.Sprint("]"))
			reader := bufio.NewReader(stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "y" && input != "yes" {
				_, _ = fmt.Fprintln(stdout, "Cancelled.")
				return nil
			}
		}
	}

	// Remove task
	delete(tasks, resolvedName)

	// Write updated file
	taskPath := filepath.Join(configDir, "tasks.cue")
	if err := writeTasksFile(taskPath, tasks); err != nil {
		return fmt.Errorf("writing tasks file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Removed task %q\n", resolvedName)
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
func loadTasksForScope(localOnly bool) (map[string]TaskConfig, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return nil, fmt.Errorf("resolving config paths: %w", err)
	}

	tasks := make(map[string]TaskConfig)

	if localOnly {
		if paths.LocalExists {
			localTasks, err := loadTasksFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, task := range localTasks {
				task.Source = "local"
				tasks[name] = task
			}
		}
	} else {
		if paths.GlobalExists {
			globalTasks, err := loadTasksFromDir(paths.Global)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, task := range globalTasks {
				task.Source = "global"
				tasks[name] = task
			}
		}
		if paths.LocalExists {
			localTasks, err := loadTasksFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, task := range localTasks {
				task.Source = "local"
				tasks[name] = task
			}
		}
	}

	return tasks, nil
}

// loadTasksFromDir loads tasks from a specific directory.
func loadTasksFromDir(dir string) (map[string]TaskConfig, error) {
	tasks := make(map[string]TaskConfig)

	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(dir)
	if err != nil {
		// If no CUE files exist, return empty map (not an error)
		if errors.Is(err, internalcue.ErrNoCUEFiles) {
			return tasks, nil
		}
		return tasks, err
	}

	tasksVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyTasks))
	if !tasksVal.Exists() {
		return tasks, nil
	}

	iter, err := tasksVal.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating tasks: %w", err)
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

		// Load tags
		if tagsVal := val.LookupPath(cue.ParsePath("tags")); tagsVal.Exists() {
			tagIter, err := tagsVal.List()
			if err == nil {
				for tagIter.Next() {
					if s, err := tagIter.Value().String(); err == nil {
						task.Tags = append(task.Tags, s)
					}
				}
			}
		}

		// Load origin (registry provenance)
		if v := val.LookupPath(cue.ParsePath("origin")); v.Exists() {
			task.Origin, _ = v.String()
		}

		tasks[name] = task
	}

	return tasks, nil
}

// writeTasksFile writes the tasks configuration to a file.
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
		if task.Prompt != "" {
			if strings.Contains(task.Prompt, "\n") || len(task.Prompt) > 80 {
				sb.WriteString("\t\tprompt: \"\"\"\n")
				for _, line := range strings.Split(task.Prompt, "\n") {
					sb.WriteString(fmt.Sprintf("\t\t\t%s\n", line))
				}
				sb.WriteString("\t\t\t\"\"\"\n")
			} else {
				sb.WriteString(fmt.Sprintf("\t\tprompt: %q\n", task.Prompt))
			}
		}
		if task.Role != "" {
			sb.WriteString(fmt.Sprintf("\t\trole: %q\n", task.Role))
		}
		if len(task.Tags) > 0 {
			sb.WriteString("\t\ttags: [")
			for i, tag := range task.Tags {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%q", tag))
			}
			sb.WriteString("]\n")
		}

		sb.WriteString("\t}\n")
	}

	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
