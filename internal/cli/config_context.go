package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// addConfigContextCommand adds the context subcommand group to the config command.
func addConfigContextCommand(parent *cobra.Command) {
	contextCmd := &cobra.Command{
		Use:     "context",
		Aliases: []string{"contexts"},
		Short:   "Manage context configuration",
		Long: `Manage context documents.

Contexts provide additional information injected into prompts.
Each context specifies content via inline text, file reference, or command.`,
		RunE: runConfigContext,
	}

	addConfigContextListCommand(contextCmd)
	addConfigContextAddCommand(contextCmd)
	addConfigContextInfoCommand(contextCmd)
	addConfigContextEditCommand(contextCmd)
	addConfigContextRemoveCommand(contextCmd)

	parent.AddCommand(contextCmd)
}

// runConfigContext runs list by default, handles help subcommand.
func runConfigContext(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	if len(args) > 0 {
		return unknownCommandError("start config context", args[0])
	}
	return runConfigContextList(cmd, args)
}

// addConfigContextListCommand adds the list subcommand.
func addConfigContextListCommand(parent *cobra.Command) {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all contexts",
		Long: `List all configured contexts.

Shows contexts from both global and local configuration.
Use --local to show only local contexts.`,
		RunE: runConfigContextList,
	}

	parent.AddCommand(listCmd)
}

// runConfigContextList lists all configured contexts.
func runConfigContextList(cmd *cobra.Command, _ []string) error {
	local, _ := cmd.Flags().GetBool("local")
	contexts, err := loadContextsForScope(local)
	if err != nil {
		return err
	}

	if len(contexts) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No contexts configured.")
		return nil
	}

	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "Contexts:")
	fmt.Fprintln(w)

	// Sort context names for consistent output
	var names []string
	for name := range contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		ctx := contexts[name]
		markers := ""
		if ctx.Required {
			markers += "R"
		}
		if ctx.Default {
			markers += "D"
		}
		if markers != "" {
			markers = "[" + markers + "] "
		}

		source := ctx.Source
		if ctx.Origin != "" {
			source += ", registry"
		}
		if ctx.Description != "" {
			fmt.Fprintf(w, "  %s%s - %s (%s)\n", markers, name, ctx.Description, source)
		} else {
			fmt.Fprintf(w, "  %s%s (%s)\n", markers, name, source)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "R = required, D = default")

	return nil
}

// addConfigContextAddCommand adds the add subcommand.
func addConfigContextAddCommand(parent *cobra.Command) {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new context",
		Long: `Add a new context configuration.

Provide context details via flags or run interactively to be prompted for values.

A context must have exactly one content source: file, command, or prompt.

Examples:
  start config context add
  start config context add --name project --file PROJECT.md --required
  start config context add --local --name readme --file README.md --default`,
		RunE: runConfigContextAdd,
	}

	addCmd.Flags().String("name", "", "Context name (identifier)")
	addCmd.Flags().String("description", "", "Description")
	addCmd.Flags().String("file", "", "Path to context file")
	addCmd.Flags().String("command", "", "Command to generate content")
	addCmd.Flags().String("prompt", "", "Inline content text")
	addCmd.Flags().Bool("required", false, "Always include this context")
	addCmd.Flags().Bool("default", false, "Include by default")
	addCmd.Flags().StringSlice("tag", nil, "Tags")

	parent.AddCommand(addCmd)
}

// runConfigContextAdd adds a new context configuration.
func runConfigContextAdd(cmd *cobra.Command, _ []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local, _ := cmd.Flags().GetBool("local")

	// Check if interactive
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	// Get flag values
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		if !isTTY {
			return fmt.Errorf("--name is required (run interactively or provide flag)")
		}
		var err error
		name, err = promptString(stdout, stdin, "Context name", "")
		if err != nil {
			return err
		}
	}
	if name == "" {
		return fmt.Errorf("context name is required")
	}

	description, _ := cmd.Flags().GetString("description")
	if description == "" && isTTY {
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

	if sourceCount == 0 && isTTY {
		fmt.Fprintln(stdout, "\nContent source (choose one):")
		fmt.Fprintln(stdout, "  1. File path")
		fmt.Fprintln(stdout, "  2. Command")
		fmt.Fprintln(stdout, "  3. Inline content")
		fmt.Fprint(stdout, "Choice [1]: ")

		reader := bufio.NewReader(stdin)
		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		choice = strings.TrimSpace(choice)
		if choice == "" {
			choice = "1"
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
			prompt, err = promptString(stdout, stdin, "Content text", "")
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

	// Ask about required/default if interactive
	required, _ := cmd.Flags().GetBool("required")
	isDefault, _ := cmd.Flags().GetBool("default")
	if isTTY && !required && !isDefault {
		fmt.Fprint(stdout, "Required (always include)? [y/N] ")
		reader := bufio.NewReader(stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		required = input == "y" || input == "yes"

		if !required {
			fmt.Fprint(stdout, "Default (include by default)? [y/N] ")
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			input = strings.TrimSpace(strings.ToLower(input))
			isDefault = input == "y" || input == "yes"
		}
	}

	// Build context struct
	tags, _ := cmd.Flags().GetStringSlice("tag")
	ctx := ContextConfig{
		Name:        name,
		Description: description,
		File:        file,
		Command:     command,
		Prompt:      prompt,
		Required:    required,
		Default:     isDefault,
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

	// Load existing contexts from target directory
	existingContexts, err := loadContextsFromDir(configDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("loading existing contexts: %w", err)
	}

	// Check for duplicate
	if _, exists := existingContexts[name]; exists {
		return fmt.Errorf("context %q already exists in %s config", name, scopeName)
	}

	// Add new context
	existingContexts[name] = ctx

	// Write contexts file
	contextPath := filepath.Join(configDir, "contexts.cue")
	if err := writeContextsFile(contextPath, existingContexts); err != nil {
		return fmt.Errorf("writing contexts file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		fmt.Fprintf(stdout, "Added context %q to %s config\n", name, scopeName)
		fmt.Fprintf(stdout, "Config: %s\n", contextPath)
	}

	return nil
}

// addConfigContextInfoCommand adds the info subcommand.
func addConfigContextInfoCommand(parent *cobra.Command) {
	infoCmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show context details",
		Long: `Show detailed information about a context.

Displays all configuration fields for the specified context.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigContextInfo,
	}

	parent.AddCommand(infoCmd)
}

// runConfigContextInfo shows detailed information about a context.
func runConfigContextInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	local, _ := cmd.Flags().GetBool("local")

	contexts, err := loadContextsForScope(local)
	if err != nil {
		return err
	}

	ctx, exists := contexts[name]
	if !exists {
		return fmt.Errorf("context %q not found", name)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Context: %s\n", name)
	fmt.Fprintln(w, strings.Repeat("─", 40))
	fmt.Fprintf(w, "Source: %s\n", ctx.Source)

	if ctx.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", ctx.Description)
	}
	if ctx.File != "" {
		fmt.Fprintf(w, "File: %s\n", ctx.File)
	}
	if ctx.Command != "" {
		fmt.Fprintf(w, "Command: %s\n", ctx.Command)
	}
	if ctx.Prompt != "" {
		fmt.Fprintf(w, "Prompt: %s\n", truncatePrompt(ctx.Prompt, 100))
	}
	fmt.Fprintf(w, "Required: %t\n", ctx.Required)
	fmt.Fprintf(w, "Default: %t\n", ctx.Default)
	if len(ctx.Tags) > 0 {
		fmt.Fprintf(w, "Tags: %s\n", strings.Join(ctx.Tags, ", "))
	}

	return nil
}

// addConfigContextEditCommand adds the edit subcommand.
func addConfigContextEditCommand(parent *cobra.Command) {
	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit context configuration",
		Long: `Edit context configuration.

Without a name, opens the contexts.cue file in $EDITOR.
With a name, provides interactive prompts to modify the context.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigContextEdit,
	}

	parent.AddCommand(editCmd)
}

// runConfigContextEdit edits a context configuration.
func runConfigContextEdit(cmd *cobra.Command, args []string) error {
	local, _ := cmd.Flags().GetBool("local")
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

	contextPath := filepath.Join(configDir, "contexts.cue")

	// No name: open file in editor
	if len(args) == 0 {
		return openInEditor(contextPath)
	}

	// Named edit: interactive modification
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	// Check if interactive
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}
	if !isTTY {
		return fmt.Errorf("interactive editing requires a terminal")
	}

	// Load existing contexts
	contexts, err := loadContextsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading contexts: %w", err)
	}

	ctx, exists := contexts[name]
	if !exists {
		return fmt.Errorf("context %q not found in %s config", name, scopeString(local))
	}

	// Prompt for each field with current value as default
	fmt.Fprintf(stdout, "Editing context %q (press Enter to keep current value)\n\n", name)

	newDescription, err := promptString(stdout, stdin, "Description", ctx.Description)
	if err != nil {
		return err
	}

	// For content source, show current and allow change
	fmt.Fprintln(stdout, "\nCurrent content source:")
	if ctx.File != "" {
		fmt.Fprintf(stdout, "  File: %s\n", ctx.File)
	}
	if ctx.Command != "" {
		fmt.Fprintf(stdout, "  Command: %s\n", ctx.Command)
	}
	if ctx.Prompt != "" {
		fmt.Fprintf(stdout, "  Prompt: %s\n", truncatePrompt(ctx.Prompt, 50))
	}

	fmt.Fprint(stdout, "Keep current? [Y/n] ")
	reader := bufio.NewReader(stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))

	newFile := ctx.File
	newCommand := ctx.Command
	newPrompt := ctx.Prompt

	if input == "n" || input == "no" {
		newFile = ""
		newCommand = ""
		newPrompt = ""

		fmt.Fprintln(stdout, "\nNew content source:")
		fmt.Fprintln(stdout, "  1. File path")
		fmt.Fprintln(stdout, "  2. Command")
		fmt.Fprintln(stdout, "  3. Inline content")
		fmt.Fprint(stdout, "Choice [1]: ")

		choice, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		choice = strings.TrimSpace(choice)
		if choice == "" {
			choice = "1"
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
			newPrompt, err = promptString(stdout, stdin, "Content text", "")
			if err != nil {
				return err
			}
		}
	}

	// Required/default flags
	fmt.Fprintf(stdout, "\nRequired (currently %t)? [y/N] ", ctx.Required)
	input, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	newRequired := input == "y" || input == "yes"

	newDefault := ctx.Default
	if !newRequired {
		fmt.Fprintf(stdout, "Default (currently %t)? [y/N] ", ctx.Default)
		input, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		newDefault = input == "y" || input == "yes"
	}

	// Update context
	ctx.Description = newDescription
	ctx.File = newFile
	ctx.Command = newCommand
	ctx.Prompt = newPrompt
	ctx.Required = newRequired
	ctx.Default = newDefault
	contexts[name] = ctx

	// Write updated file
	if err := writeContextsFile(contextPath, contexts); err != nil {
		return fmt.Errorf("writing contexts file: %w", err)
	}

	fmt.Fprintf(stdout, "\nUpdated context %q\n", name)
	return nil
}

// addConfigContextRemoveCommand adds the remove subcommand.
func addConfigContextRemoveCommand(parent *cobra.Command) {
	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a context",
		Long: `Remove a context configuration.

Removes the specified context from the configuration file.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigContextRemove,
	}

	parent.AddCommand(removeCmd)
}

// runConfigContextRemove removes a context configuration.
func runConfigContextRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local, _ := cmd.Flags().GetBool("local")

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

	// Load existing contexts
	contexts, err := loadContextsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading contexts: %w", err)
	}

	if _, exists := contexts[name]; !exists {
		return fmt.Errorf("context %q not found in %s config", name, scopeString(local))
	}

	// Confirm removal
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	if isTTY {
		fmt.Fprintf(stdout, "Remove context %q from %s config? [y/N] ", name, scopeString(local))
		reader := bufio.NewReader(stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Fprintln(stdout, "Cancelled.")
			return nil
		}
	}

	// Remove context
	delete(contexts, name)

	// Write updated file
	contextPath := filepath.Join(configDir, "contexts.cue")
	if err := writeContextsFile(contextPath, contexts); err != nil {
		return fmt.Errorf("writing contexts file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		fmt.Fprintf(stdout, "Removed context %q\n", name)
	}

	return nil
}

// ContextConfig represents a context configuration for editing.
type ContextConfig struct {
	Name        string
	Description string
	File        string
	Command     string
	Prompt      string
	Required    bool
	Default     bool
	Tags        []string
	Source      string // "global" or "local" - for display only
	Origin      string // Registry module path when installed from registry
}

// loadContextsForScope loads contexts from the appropriate scope.
func loadContextsForScope(localOnly bool) (map[string]ContextConfig, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return nil, fmt.Errorf("resolving config paths: %w", err)
	}

	contexts := make(map[string]ContextConfig)

	if localOnly {
		if paths.LocalExists {
			localContexts, err := loadContextsFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, ctx := range localContexts {
				ctx.Source = "local"
				contexts[name] = ctx
			}
		}
	} else {
		if paths.GlobalExists {
			globalContexts, err := loadContextsFromDir(paths.Global)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, ctx := range globalContexts {
				ctx.Source = "global"
				contexts[name] = ctx
			}
		}
		if paths.LocalExists {
			localContexts, err := loadContextsFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, ctx := range localContexts {
				ctx.Source = "local"
				contexts[name] = ctx
			}
		}
	}

	return contexts, nil
}

// loadContextsFromDir loads contexts from a specific directory.
func loadContextsFromDir(dir string) (map[string]ContextConfig, error) {
	contexts := make(map[string]ContextConfig)

	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(dir)
	if err != nil {
		// If no CUE files exist, return empty map (not an error)
		if strings.Contains(err.Error(), "no CUE files") {
			return contexts, nil
		}
		return contexts, err
	}

	contextsVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyContexts))
	if !contextsVal.Exists() {
		return contexts, nil
	}

	iter, err := contextsVal.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating contexts: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		val := iter.Value()

		ctx := ContextConfig{Name: name}

		if v := val.LookupPath(cue.ParsePath("description")); v.Exists() {
			ctx.Description, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("file")); v.Exists() {
			ctx.File, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("command")); v.Exists() {
			ctx.Command, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("prompt")); v.Exists() {
			ctx.Prompt, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("required")); v.Exists() {
			ctx.Required, _ = v.Bool()
		}
		if v := val.LookupPath(cue.ParsePath("default")); v.Exists() {
			ctx.Default, _ = v.Bool()
		}

		// Load tags
		if tagsVal := val.LookupPath(cue.ParsePath("tags")); tagsVal.Exists() {
			tagIter, err := tagsVal.List()
			if err == nil {
				for tagIter.Next() {
					if s, err := tagIter.Value().String(); err == nil {
						ctx.Tags = append(ctx.Tags, s)
					}
				}
			}
		}

		// Load origin (registry provenance)
		if v := val.LookupPath(cue.ParsePath("origin")); v.Exists() {
			ctx.Origin, _ = v.String()
		}

		contexts[name] = ctx
	}

	return contexts, nil
}

// writeContextsFile writes the contexts configuration to a file.
func writeContextsFile(path string, contexts map[string]ContextConfig) error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your context configuration\n\n")
	sb.WriteString("contexts: {\n")

	// Sort context names for consistent output
	var names []string
	for name := range contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		ctx := contexts[name]
		sb.WriteString(fmt.Sprintf("\t%q: {\n", name))

		// Write origin first if present (registry provenance)
		if ctx.Origin != "" {
			sb.WriteString(fmt.Sprintf("\t\torigin: %q\n", ctx.Origin))
		}
		if ctx.Description != "" {
			sb.WriteString(fmt.Sprintf("\t\tdescription: %q\n", ctx.Description))
		}
		if ctx.File != "" {
			sb.WriteString(fmt.Sprintf("\t\tfile: %q\n", ctx.File))
		}
		if ctx.Command != "" {
			sb.WriteString(fmt.Sprintf("\t\tcommand: %q\n", ctx.Command))
		}
		if ctx.Prompt != "" {
			if strings.Contains(ctx.Prompt, "\n") || len(ctx.Prompt) > 80 {
				sb.WriteString("\t\tprompt: \"\"\"\n")
				for _, line := range strings.Split(ctx.Prompt, "\n") {
					sb.WriteString(fmt.Sprintf("\t\t\t%s\n", line))
				}
				sb.WriteString("\t\t\t\"\"\"\n")
			} else {
				sb.WriteString(fmt.Sprintf("\t\tprompt: %q\n", ctx.Prompt))
			}
		}
		if ctx.Required {
			sb.WriteString("\t\trequired: true\n")
		}
		if ctx.Default {
			sb.WriteString("\t\tdefault: true\n")
		}
		if len(ctx.Tags) > 0 {
			sb.WriteString("\t\ttags: [")
			for i, tag := range ctx.Tags {
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
