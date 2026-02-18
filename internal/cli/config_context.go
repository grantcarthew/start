package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/spf13/cobra"
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
	addConfigContextOrderCommand(contextCmd)

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
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all contexts",
		Long: `List all configured contexts.

Shows contexts from both global and local configuration.
Use --local to show only local contexts.`,
		RunE: runConfigContextList,
	}

	parent.AddCommand(listCmd)
}

// runConfigContextList lists all configured contexts.
func runConfigContextList(cmd *cobra.Command, _ []string) error {
	local := getFlags(cmd).Local
	contexts, order, err := loadContextsForScope(local)
	if err != nil {
		return err
	}

	if len(contexts) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No contexts configured.")
		return nil
	}

	w := cmd.OutOrStdout()
	_, _ = colorContexts.Fprint(w, "contexts")
	_, _ = fmt.Fprintln(w, "/")
	_, _ = fmt.Fprintln(w)

	for _, name := range order {
		ctx := contexts[name]
		source := ctx.Source
		if ctx.Origin != "" {
			source += ", registry"
		}
		if ctx.Description != "" {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = colorDim.Fprint(w, "- "+ctx.Description+" ")
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, source)
			_, _ = colorCyan.Fprint(w, ")")
		} else {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, source)
			_, _ = colorCyan.Fprint(w, ")")
		}
		if ctx.Required {
			_, _ = fmt.Fprint(w, " ")
			_, _ = colorCyan.Fprint(w, "[")
			_, _ = colorDim.Fprint(w, "required")
			_, _ = colorCyan.Fprint(w, "]")
		}
		if ctx.Default {
			_, _ = fmt.Fprint(w, " ")
			_, _ = colorCyan.Fprint(w, "[")
			_, _ = colorDim.Fprint(w, "default")
			_, _ = colorCyan.Fprint(w, "]")
		}
		_, _ = fmt.Fprintln(w)
	}

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
		Args: cobra.NoArgs,
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
	local := getFlags(cmd).Local

	// Check if interactive - only prompt for optional fields if no flags provided
	isTTY := isTerminal(stdin)
	// If any flags are set, skip prompts for optional fields
	hasFlags := anyFlagChanged(cmd, "name", "description", "file", "command", "prompt", "required", "default", "tag")
	interactive := isTTY && !hasFlags

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
		var err error
		file, command, prompt, err = promptContentSource(stdout, stdin, "1", "")
		if err != nil {
			return err
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
		_, _ = fmt.Fprintf(stdout, "Required %s%s%s? %s%s%s ", colorCyan.Sprint("("), colorDim.Sprint("always include"), colorCyan.Sprint(")"), colorCyan.Sprint("["), colorDim.Sprint("y/N"), colorCyan.Sprint("]"))
		reader := bufio.NewReader(stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		required = input == "y" || input == "yes"

		if !required {
			_, _ = fmt.Fprintf(stdout, "Default %s%s%s? %s%s%s ", colorCyan.Sprint("("), colorDim.Sprint("include by default"), colorCyan.Sprint(")"), colorCyan.Sprint("["), colorDim.Sprint("y/N"), colorCyan.Sprint("]"))
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

	configDir := paths.Dir(local)
	scopeName := scopeString(local)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Load existing contexts from target directory
	existingContexts, existingOrder, err := loadContextsFromDir(configDir)
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
	if err := writeContextsFile(contextPath, existingContexts, append(existingOrder, name)); err != nil {
		return fmt.Errorf("writing contexts file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Added context %q to %s config\n", name, scopeName)
		_, _ = fmt.Fprintf(stdout, "Config: %s\n", contextPath)
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
	local := getFlags(cmd).Local

	contexts, _, err := loadContextsForScope(local)
	if err != nil {
		return err
	}

	resolvedName, ctx, err := resolveInstalledName(contexts, "context", name)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(w)
	_, _ = colorContexts.Fprint(w, "contexts")
	_, _ = fmt.Fprintf(w, "/%s\n", resolvedName)
	printSeparator(w)

	_, _ = colorDim.Fprint(w, "Source:")
	_, _ = fmt.Fprintf(w, " %s\n", ctx.Source)
	if ctx.Origin != "" {
		_, _ = colorDim.Fprint(w, "Origin:")
		_, _ = fmt.Fprintf(w, " %s\n", ctx.Origin)
	}
	if ctx.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = colorDim.Fprint(w, "Description:")
		_, _ = fmt.Fprintf(w, " %s\n", ctx.Description)
	}
	if ctx.File != "" {
		_, _ = colorDim.Fprint(w, "File:")
		_, _ = fmt.Fprintf(w, " %s\n", ctx.File)
	}
	if ctx.Command != "" {
		_, _ = colorDim.Fprint(w, "Command:")
		_, _ = fmt.Fprintf(w, " %s\n", ctx.Command)
	}
	if ctx.Prompt != "" {
		_, _ = colorDim.Fprint(w, "Prompt:")
		_, _ = fmt.Fprintf(w, " %s\n", truncatePrompt(ctx.Prompt, 100))
	}
	_, _ = colorDim.Fprint(w, "Required:")
	_, _ = fmt.Fprintf(w, " %t\n", ctx.Required)
	_, _ = colorDim.Fprint(w, "Default:")
	_, _ = fmt.Fprintf(w, " %t\n", ctx.Default)
	if len(ctx.Tags) > 0 {
		_, _ = colorDim.Fprint(w, "Tags:")
		_, _ = fmt.Fprintf(w, " %s\n", strings.Join(ctx.Tags, ", "))
	}
	printSeparator(w)

	return nil
}

// addConfigContextEditCommand adds the edit subcommand.
func addConfigContextEditCommand(parent *cobra.Command) {
	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit context configuration",
		Long: `Edit context configuration.

Without a name, opens the contexts.cue file in $EDITOR.
With a name and flags, updates only the specified fields.
With a name and no flags in a terminal, provides interactive prompts.

Examples:
  start config context edit
  start config context edit project --file PROJECT.md
  start config context edit readme --required=true --default=false`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigContextEdit,
	}

	editCmd.Flags().String("description", "", "Description")
	editCmd.Flags().String("file", "", "Path to context file")
	editCmd.Flags().String("command", "", "Command to generate content")
	editCmd.Flags().String("prompt", "", "Inline content text")
	editCmd.Flags().Bool("required", false, "Always include this context")
	editCmd.Flags().Bool("default", false, "Include by default")
	editCmd.Flags().StringSlice("tag", nil, "Tags")

	parent.AddCommand(editCmd)
}

// runConfigContextEdit edits a context configuration.
func runConfigContextEdit(cmd *cobra.Command, args []string) error {
	local := getFlags(cmd).Local
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(local)

	contextPath := filepath.Join(configDir, "contexts.cue")

	// No name: open file in editor
	if len(args) == 0 {
		return openInEditor(contextPath)
	}

	// Named edit
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	// Load existing contexts
	contexts, order, err := loadContextsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading contexts: %w", err)
	}

	resolvedName, ctx, err := resolveInstalledName(contexts, "context", name)
	if err != nil {
		return err
	}

	// Check if any edit flags are provided
	hasEditFlags := anyFlagChanged(cmd, "description", "file", "command", "prompt", "required", "default", "tag")

	if hasEditFlags {
		// Non-interactive flag-based update
		if cmd.Flags().Changed("description") {
			ctx.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("file") {
			ctx.File, _ = cmd.Flags().GetString("file")
		}
		if cmd.Flags().Changed("command") {
			ctx.Command, _ = cmd.Flags().GetString("command")
		}
		if cmd.Flags().Changed("prompt") {
			ctx.Prompt, _ = cmd.Flags().GetString("prompt")
		}
		if cmd.Flags().Changed("required") {
			ctx.Required, _ = cmd.Flags().GetBool("required")
		}
		if cmd.Flags().Changed("default") {
			ctx.Default, _ = cmd.Flags().GetBool("default")
		}
		if cmd.Flags().Changed("tag") {
			ctx.Tags, _ = cmd.Flags().GetStringSlice("tag")
		}

		contexts[resolvedName] = ctx

		if err := writeContextsFile(contextPath, contexts, order); err != nil {
			return fmt.Errorf("writing contexts file: %w", err)
		}

		flags := getFlags(cmd)
		if !flags.Quiet {
			_, _ = fmt.Fprintf(stdout, "Updated context %q\n", resolvedName)
		}
		return nil
	}

	// No flags: require TTY for interactive editing
	isTTY := isTerminal(stdin)
	if !isTTY {
		return fmt.Errorf("interactive editing requires a terminal")
	}

	// Prompt for each field with current value as default
	_, _ = fmt.Fprintf(stdout, "Editing context %q %s%s%s\n\n", resolvedName, colorCyan.Sprint("("), colorDim.Sprint("press Enter to keep current value"), colorCyan.Sprint(")"))

	newDescription, err := promptString(stdout, stdin, "Description", ctx.Description)
	if err != nil {
		return err
	}

	// For content source, show current and allow change
	_, _ = fmt.Fprintln(stdout, "\nCurrent content source:")
	if ctx.File != "" {
		_, _ = fmt.Fprintf(stdout, "  File: %s\n", ctx.File)
	}
	if ctx.Command != "" {
		_, _ = fmt.Fprintf(stdout, "  Command: %s\n", ctx.Command)
	}
	if ctx.Prompt != "" {
		_, _ = fmt.Fprintf(stdout, "  Prompt: %s\n", truncatePrompt(ctx.Prompt, 50))
	}

	_, _ = fmt.Fprintf(stdout, "Keep current? %s%s%s ", colorCyan.Sprint("["), colorDim.Sprint("Y/n"), colorCyan.Sprint("]"))
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
		newFile, newCommand, newPrompt, err = promptContentSource(stdout, stdin, "1", ctx.Prompt)
		if err != nil {
			return err
		}
	}

	// Required/default flags
	_, _ = fmt.Fprintf(stdout, "\nRequired %s%s%s? %s%s%s ", colorCyan.Sprint("("), colorDim.Sprintf("currently %t", ctx.Required), colorCyan.Sprint(")"), colorCyan.Sprint("["), colorDim.Sprint("y/N"), colorCyan.Sprint("]"))
	input, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	newRequired := input == "y" || input == "yes"

	newDefault := ctx.Default
	if !newRequired {
		_, _ = fmt.Fprintf(stdout, "Default %s%s%s? %s%s%s ", colorCyan.Sprint("("), colorDim.Sprintf("currently %t", ctx.Default), colorCyan.Sprint(")"), colorCyan.Sprint("["), colorDim.Sprint("y/N"), colorCyan.Sprint("]"))
		input, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		newDefault = input == "y" || input == "yes"
	}

	// Prompt for tags
	_, _ = fmt.Fprintln(stdout)
	newTags, err := promptTags(stdout, stdin, ctx.Tags)
	if err != nil {
		return err
	}

	// Update context
	ctx.Description = newDescription
	ctx.File = newFile
	ctx.Command = newCommand
	ctx.Prompt = newPrompt
	ctx.Required = newRequired
	ctx.Default = newDefault
	ctx.Tags = newTags
	contexts[resolvedName] = ctx

	// Write updated file
	if err := writeContextsFile(contextPath, contexts, order); err != nil {
		return fmt.Errorf("writing contexts file: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "\nUpdated context %q\n", resolvedName)
	return nil
}

// addConfigContextRemoveCommand adds the remove subcommand.
func addConfigContextRemoveCommand(parent *cobra.Command) {
	removeCmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a context",
		Long: `Remove a context configuration.

Removes the specified context from the configuration file.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigContextRemove,
	}

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(removeCmd)
}

// runConfigContextRemove removes a context configuration.
func runConfigContextRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(local)

	// Load existing contexts
	contexts, order, err := loadContextsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading contexts: %w", err)
	}

	resolvedName, _, err := resolveInstalledName(contexts, "context", name)
	if err != nil {
		return err
	}

	// Confirm removal unless --yes flag is set
	skipConfirm, _ := cmd.Flags().GetBool("yes")
	if !skipConfirm {
		confirmed, err := confirmRemoval(stdout, stdin, "context", resolvedName, local)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	// Remove context and its order entry
	delete(contexts, resolvedName)
	newOrder := make([]string, 0, len(order))
	for _, n := range order {
		if n != resolvedName {
			newOrder = append(newOrder, n)
		}
	}

	// Write updated file
	contextPath := filepath.Join(configDir, "contexts.cue")
	if err := writeContextsFile(contextPath, contexts, newOrder); err != nil {
		return fmt.Errorf("writing contexts file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Removed context %q\n", resolvedName)
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
// Returns the contexts map, names in definition order, and any error.
func loadContextsForScope(localOnly bool) (map[string]ContextConfig, []string, error) {
	return loadForScope(localOnly, loadContextsFromDir, func(c *ContextConfig, s string) { c.Source = s })
}

// loadContextsFromDir loads contexts from a specific directory.
// Returns the contexts map, names in definition order, and any error.
func loadContextsFromDir(dir string) (map[string]ContextConfig, []string, error) {
	contexts := make(map[string]ContextConfig)
	var order []string

	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(dir)
	if err != nil {
		// If no CUE files exist, return empty map (not an error)
		if errors.Is(err, internalcue.ErrNoCUEFiles) {
			return contexts, order, nil
		}
		return contexts, order, err
	}

	contextsVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyContexts))
	if !contextsVal.Exists() {
		return contexts, order, nil
	}

	iter, err := contextsVal.Fields()
	if err != nil {
		return nil, nil, fmt.Errorf("iterating contexts: %w", err)
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

		ctx.Tags = extractTags(val)

		// Load origin (registry provenance)
		if v := val.LookupPath(cue.ParsePath("origin")); v.Exists() {
			ctx.Origin, _ = v.String()
		}

		contexts[name] = ctx
		order = append(order, name)
	}

	return contexts, order, nil
}

// writeContextsFile writes the contexts configuration to a file.
// Fields are written in the provided order.
func writeContextsFile(path string, contexts map[string]ContextConfig, order []string) error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your context configuration\n\n")
	sb.WriteString("contexts: {\n")

	for _, name := range order {
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
		writeCUEPrompt(&sb, ctx.Prompt)
		if ctx.Required {
			sb.WriteString("\t\trequired: true\n")
		}
		if ctx.Default {
			sb.WriteString("\t\tdefault: true\n")
		}
		writeCUETags(&sb, ctx.Tags)

		sb.WriteString("\t}\n")
	}

	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
