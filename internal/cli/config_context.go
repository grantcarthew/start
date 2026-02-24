package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/grantcarthew/start/internal/config"
	"github.com/grantcarthew/start/internal/tui"
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
	_, _ = tui.ColorContexts.Fprint(w, "contexts")
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
			_, _ = tui.ColorDim.Fprint(w, "- "+ctx.Description+" ")
			_, _ = fmt.Fprint(w, tui.Annotate("%s", source))
		} else {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = fmt.Fprint(w, tui.Annotate("%s", source))
		}
		if ctx.Required {
			_, _ = fmt.Fprintf(w, " %s", tui.Bracket("required"))
		}
		if ctx.Default {
			_, _ = fmt.Fprintf(w, " %s", tui.Bracket("default"))
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

Run interactively to be prompted for values.

A context must have exactly one content source: file, command, or prompt.

Examples:
  start config context add
  start config context add --local`,
		Args: noArgsOrHelp,
		RunE: runConfigContextAdd,
	}

	parent.AddCommand(addCmd)
}

// runConfigContextAdd adds a new context configuration.
func runConfigContextAdd(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	stdin := cmd.InOrStdin()
	if !isTerminal(stdin) {
		return fmt.Errorf("interactive add requires a terminal")
	}
	return configContextAdd(stdin, cmd.OutOrStdout(), getFlags(cmd).Local)
}

// configContextAdd is the inner add logic for contexts.
func configContextAdd(stdin io.Reader, stdout io.Writer, local bool) error {
	name, err := promptString(stdout, stdin, "Context name", "")
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("context name is required")
	}

	description, err := promptString(stdout, stdin, "Description (optional)", "")
	if err != nil {
		return err
	}

	// Content source
	file, command, prompt, err := promptContentSource(stdout, stdin, "1", "")
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

	// Ask about required/default
	var required, isDefault bool
	{
		_, _ = fmt.Fprintf(stdout, "Required %s? %s ", tui.Annotate("always include"), tui.Bracket("y/N"))
		reader := bufio.NewReader(stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(strings.ToLower(input))
		required = input == "y" || input == "yes"

		if !required {
			_, _ = fmt.Fprintf(stdout, "Default %s? %s ", tui.Annotate("include by default"), tui.Bracket("y/N"))
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading input: %w", err)
			}
			input = strings.TrimSpace(strings.ToLower(input))
			isDefault = input == "y" || input == "yes"
		}
	}

	// Build context struct (tags are empty on add; use edit to set them)
	ctx := ContextConfig{
		Name:        name,
		Description: description,
		File:        file,
		Command:     command,
		Prompt:      prompt,
		Required:    required,
		Default:     isDefault,
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

	_, _ = fmt.Fprintf(stdout, "Added context %q to %s config\n", name, scopeName)
	_, _ = fmt.Fprintf(stdout, "Config: %s\n", contextPath)

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
	_, _ = tui.ColorContexts.Fprint(w, "contexts")
	_, _ = fmt.Fprintf(w, "/%s\n", resolvedName)
	printSeparator(w)

	_, _ = tui.ColorDim.Fprint(w, "Source:")
	_, _ = fmt.Fprintf(w, " %s\n", ctx.Source)
	if ctx.Origin != "" {
		_, _ = tui.ColorDim.Fprint(w, "Origin:")
		_, _ = fmt.Fprintf(w, " %s\n", ctx.Origin)
	}
	if ctx.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = tui.ColorDim.Fprint(w, "Description:")
		_, _ = fmt.Fprintf(w, " %s\n", ctx.Description)
	}
	if ctx.File != "" {
		_, _ = tui.ColorDim.Fprint(w, "File:")
		_, _ = fmt.Fprintf(w, " %s\n", ctx.File)
	}
	if ctx.Command != "" {
		_, _ = tui.ColorDim.Fprint(w, "Command:")
		_, _ = fmt.Fprintf(w, " %s\n", ctx.Command)
	}
	if ctx.Prompt != "" {
		_, _ = tui.ColorDim.Fprint(w, "Prompt:")
		_, _ = fmt.Fprintf(w, " %s\n", truncatePrompt(ctx.Prompt, 100))
	}
	_, _ = tui.ColorDim.Fprint(w, "Required:")
	_, _ = fmt.Fprintf(w, " %t\n", ctx.Required)
	_, _ = tui.ColorDim.Fprint(w, "Default:")
	_, _ = fmt.Fprintf(w, " %t\n", ctx.Default)
	if len(ctx.Tags) > 0 {
		_, _ = tui.ColorDim.Fprint(w, "Tags:")
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
With a name, prompts interactively for values.

Examples:
  start config context edit
  start config context edit project`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigContextEdit,
	}

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

	stdin := cmd.InOrStdin()
	if !isTerminal(stdin) {
		return fmt.Errorf("interactive edit requires a terminal")
	}
	return configContextEdit(stdin, cmd.OutOrStdout(), local, args[0])
}

// configContextEdit is the inner edit logic for contexts.
func configContextEdit(stdin io.Reader, stdout io.Writer, local bool, name string) error {
	// Determine target directory
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}
	configDir := paths.Dir(local)
	contextPath := filepath.Join(configDir, "contexts.cue")

	// Load existing contexts
	contexts, order, err := loadContextsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading contexts: %w", err)
	}

	resolvedName, ctx, err := resolveInstalledName(contexts, "context", name)
	if err != nil {
		return err
	}

	// Prompt for each field with current value as default
	_, _ = fmt.Fprintf(stdout, "Editing context %q %s\n\n", resolvedName, tui.Annotate("press Enter to keep current value"))

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

	_, _ = fmt.Fprintf(stdout, "Keep current? %s ", tui.Bracket("Y/n"))
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
	_, _ = fmt.Fprintf(stdout, "\nRequired %s? %s ", tui.Annotate("currently %t", ctx.Required), tui.Bracket("y/N"))
	input, err = reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	newRequired := input == "y" || input == "yes"

	newDefault := ctx.Default
	if !newRequired {
		_, _ = fmt.Fprintf(stdout, "Default %s? %s ", tui.Annotate("currently %t", ctx.Default), tui.Bracket("y/N"))
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
		Use:     "remove <name>...",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove one or more contexts",
		Long: `Remove one or more context configurations.

Removes the specified contexts from the configuration file.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runConfigContextRemove,
	}

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(removeCmd)
}

// runConfigContextRemove removes one or more context configurations.
func runConfigContextRemove(cmd *cobra.Command, args []string) error {
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

	skipConfirm, _ := cmd.Flags().GetBool("yes")

	resolvedNames, err := resolveRemoveNames(contexts, "context", args, skipConfirm, stdout, stdin)
	if err != nil {
		return err
	}
	if resolvedNames == nil {
		return nil // user cancelled
	}

	// Confirm removal unless --yes flag is set
	if !skipConfirm {
		confirmed, err := confirmMultiRemoval(stdout, stdin, "context", resolvedNames, local)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	// Build set of names to remove for order filtering
	toRemove := make(map[string]bool, len(resolvedNames))
	for _, name := range resolvedNames {
		toRemove[name] = true
	}

	// Remove all contexts and their order entries
	for _, name := range resolvedNames {
		delete(contexts, name)
	}
	newOrder := make([]string, 0, len(order))
	for _, n := range order {
		if !toRemove[n] {
			newOrder = append(newOrder, n)
		}
	}

	// Write updated file once
	contextPath := filepath.Join(configDir, "contexts.cue")
	if err := writeContextsFile(contextPath, contexts, newOrder); err != nil {
		return fmt.Errorf("writing contexts file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		for _, name := range resolvedNames {
			_, _ = fmt.Fprintf(stdout, "Removed context %q\n", name)
		}
	}

	return nil
}

