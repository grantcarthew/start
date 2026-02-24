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

// addConfigRoleCommand adds the role subcommand group to the config command.
func addConfigRoleCommand(parent *cobra.Command) {
	roleCmd := &cobra.Command{
		Use:     "role",
		Aliases: []string{"roles"},
		Short:   "Manage role configuration",
		Long: `Manage AI agent roles (system prompts).

Roles define the behavior and expertise of AI agents.
Each role specifies a prompt via inline text, file reference, or command.`,
		RunE: runConfigRole,
	}

	addConfigRoleListCommand(roleCmd)
	addConfigRoleAddCommand(roleCmd)
	addConfigRoleInfoCommand(roleCmd)
	addConfigRoleEditCommand(roleCmd)
	addConfigRoleRemoveCommand(roleCmd)
	addConfigRoleOrderCommand(roleCmd)

	parent.AddCommand(roleCmd)
}

// runConfigRole runs list by default, handles help subcommand.
func runConfigRole(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	if len(args) > 0 {
		return unknownCommandError("start config role", args[0])
	}
	return runConfigRoleList(cmd, args)
}

// addConfigRoleListCommand adds the list subcommand.
func addConfigRoleListCommand(parent *cobra.Command) {
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all roles",
		Long: `List all configured roles.

Shows roles from both global and local configuration.
Use --local to show only local roles.`,
		RunE: runConfigRoleList,
	}

	parent.AddCommand(listCmd)
}

// runConfigRoleList lists all configured roles.
func runConfigRoleList(cmd *cobra.Command, _ []string) error {
	local := getFlags(cmd).Local
	roles, order, err := loadRolesForScope(local)
	if err != nil {
		return err
	}

	if len(roles) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No roles configured.")
		return nil
	}

	w := cmd.OutOrStdout()
	_, _ = tui.ColorRoles.Fprint(w, "roles")
	_, _ = fmt.Fprintln(w, "/")
	_, _ = fmt.Fprintln(w)

	for _, name := range order {
		role := roles[name]
		source := role.Source
		if role.Origin != "" {
			source += ", registry"
		}
		if role.Description != "" {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = tui.ColorDim.Fprint(w, "- "+role.Description+" ")
			_, _ = fmt.Fprintln(w, tui.Annotate("%s", source))
		} else {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = fmt.Fprintln(w, tui.Annotate("%s", source))
		}
	}

	return nil
}

// addConfigRoleAddCommand adds the add subcommand.
func addConfigRoleAddCommand(parent *cobra.Command) {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new role",
		Long: `Add a new role configuration.

Run interactively to be prompted for values.

A role must have exactly one content source: file, command, or prompt.

Examples:
  start config role add
  start config role add --local`,
		Args: noArgsOrHelp,
		RunE: runConfigRoleAdd,
	}

	parent.AddCommand(addCmd)
}

// runConfigRoleAdd adds a new role configuration.
func runConfigRoleAdd(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	stdin := cmd.InOrStdin()
	if !isTerminal(stdin) {
		return fmt.Errorf("interactive add requires a terminal")
	}
	return configRoleAdd(stdin, cmd.OutOrStdout(), getFlags(cmd).Local)
}

// configRoleAdd is the inner add logic for roles.
func configRoleAdd(stdin io.Reader, stdout io.Writer, local bool) error {
	name, err := promptString(stdout, stdin, "Role name", "")
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("role name is required")
	}

	description, err := promptString(stdout, stdin, "Description (optional)", "")
	if err != nil {
		return err
	}

	// Content source: must have exactly one of file, command, or prompt
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

	// Build role struct (tags and optional are empty on add; use edit to set them)
	role := RoleConfig{
		Name:        name,
		Description: description,
		File:        file,
		Command:     command,
		Prompt:      prompt,
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

	// Load existing roles from target directory
	existingRoles, existingOrder, err := loadRolesFromDir(configDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("loading existing roles: %w", err)
	}

	// Check for duplicate
	if _, exists := existingRoles[name]; exists {
		return fmt.Errorf("role %q already exists in %s config", name, scopeName)
	}

	// Add new role
	existingRoles[name] = role

	// Write roles file
	rolePath := filepath.Join(configDir, "roles.cue")
	if err := writeRolesFile(rolePath, existingRoles, append(existingOrder, name)); err != nil {
		return fmt.Errorf("writing roles file: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "Added role %q to %s config\n", name, scopeName)
	_, _ = fmt.Fprintf(stdout, "Config: %s\n", rolePath)

	return nil
}

// addConfigRoleInfoCommand adds the info subcommand.
func addConfigRoleInfoCommand(parent *cobra.Command) {
	infoCmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show role details",
		Long: `Show detailed information about a role.

Displays all configuration fields for the specified role.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigRoleInfo,
	}

	parent.AddCommand(infoCmd)
}

// runConfigRoleInfo shows detailed information about a role.
func runConfigRoleInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	local := getFlags(cmd).Local

	roles, _, err := loadRolesForScope(local)
	if err != nil {
		return err
	}

	resolvedName, role, err := resolveInstalledName(roles, "role", name)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(w)
	_, _ = tui.ColorRoles.Fprint(w, "roles")
	_, _ = fmt.Fprintf(w, "/%s\n", resolvedName)
	printSeparator(w)

	_, _ = tui.ColorDim.Fprint(w, "Source:")
	_, _ = fmt.Fprintf(w, " %s\n", role.Source)
	if role.Origin != "" {
		_, _ = tui.ColorDim.Fprint(w, "Origin:")
		_, _ = fmt.Fprintf(w, " %s\n", role.Origin)
	}
	if role.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = tui.ColorDim.Fprint(w, "Description:")
		_, _ = fmt.Fprintf(w, " %s\n", role.Description)
	}
	if role.File != "" {
		_, _ = tui.ColorDim.Fprint(w, "File:")
		_, _ = fmt.Fprintf(w, " %s\n", role.File)
	}
	if role.Command != "" {
		_, _ = tui.ColorDim.Fprint(w, "Command:")
		_, _ = fmt.Fprintf(w, " %s\n", role.Command)
	}
	if role.Prompt != "" {
		_, _ = tui.ColorDim.Fprint(w, "Prompt:")
		_, _ = fmt.Fprintf(w, " %s\n", truncatePrompt(role.Prompt, 100))
	}
	if len(role.Tags) > 0 {
		_, _ = tui.ColorDim.Fprint(w, "Tags:")
		_, _ = fmt.Fprintf(w, " %s\n", strings.Join(role.Tags, ", "))
	}
	printSeparator(w)

	return nil
}

// addConfigRoleEditCommand adds the edit subcommand.
func addConfigRoleEditCommand(parent *cobra.Command) {
	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit role configuration",
		Long: `Edit role configuration.

Without a name, opens the roles.cue file in $EDITOR.
With a name, prompts interactively for values.

Examples:
  start config role edit
  start config role edit go-expert`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigRoleEdit,
	}

	parent.AddCommand(editCmd)
}

// runConfigRoleEdit edits a role configuration.
func runConfigRoleEdit(cmd *cobra.Command, args []string) error {
	local := getFlags(cmd).Local
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}
	configDir := paths.Dir(local)
	rolePath := filepath.Join(configDir, "roles.cue")

	// No name: open file in editor
	if len(args) == 0 {
		return openInEditor(rolePath)
	}

	stdin := cmd.InOrStdin()
	if !isTerminal(stdin) {
		return fmt.Errorf("interactive edit requires a terminal")
	}
	return configRoleEdit(stdin, cmd.OutOrStdout(), local, args[0])
}

// configRoleEdit is the inner edit logic for roles.
func configRoleEdit(stdin io.Reader, stdout io.Writer, local bool, name string) error {
	// Determine target directory
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}
	configDir := paths.Dir(local)
	rolePath := filepath.Join(configDir, "roles.cue")

	// Load existing roles
	roles, order, err := loadRolesFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading roles: %w", err)
	}

	resolvedName, role, err := resolveInstalledName(roles, "role", name)
	if err != nil {
		return err
	}

	// Prompt for each field with current value as default
	_, _ = fmt.Fprintf(stdout, "Editing role %q %s\n\n", resolvedName, tui.Annotate("press Enter to keep current value"))

	newDescription, err := promptString(stdout, stdin, "Description", role.Description)
	if err != nil {
		return err
	}

	// For content source, show current and allow change
	_, _ = fmt.Fprintln(stdout, "\nCurrent content source:")
	if role.File != "" {
		_, _ = fmt.Fprintf(stdout, "  File: %s\n", role.File)
	}
	if role.Command != "" {
		_, _ = fmt.Fprintf(stdout, "  Command: %s\n", role.Command)
	}
	if role.Prompt != "" {
		_, _ = fmt.Fprintf(stdout, "  Prompt: %s\n", truncatePrompt(role.Prompt, 50))
	}

	_, _ = fmt.Fprintf(stdout, "Keep current? %s ", tui.Bracket("Y/n"))
	reader := bufio.NewReader(stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))

	newFile := role.File
	newCommand := role.Command
	newPrompt := role.Prompt

	if input == "n" || input == "no" {
		newFile, newCommand, newPrompt, err = promptContentSource(stdout, stdin, "1", role.Prompt)
		if err != nil {
			return err
		}
	}

	// Prompt for tags
	_, _ = fmt.Fprintln(stdout)
	newTags, err := promptTags(stdout, stdin, role.Tags)
	if err != nil {
		return err
	}

	// Update role
	role.Description = newDescription
	role.File = newFile
	role.Command = newCommand
	role.Prompt = newPrompt
	role.Tags = newTags
	roles[resolvedName] = role

	// Write updated file
	if err := writeRolesFile(rolePath, roles, order); err != nil {
		return fmt.Errorf("writing roles file: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "\nUpdated role %q\n", resolvedName)
	return nil
}

// addConfigRoleRemoveCommand adds the remove subcommand.
func addConfigRoleRemoveCommand(parent *cobra.Command) {
	removeCmd := &cobra.Command{
		Use:     "remove <name>...",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove one or more roles",
		Long: `Remove one or more role configurations.

Removes the specified roles from the configuration file.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runConfigRoleRemove,
	}

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(removeCmd)
}

// runConfigRoleRemove removes one or more role configurations.
func runConfigRoleRemove(cmd *cobra.Command, args []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(local)

	// Load existing roles
	roles, order, err := loadRolesFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading roles: %w", err)
	}

	skipConfirm, _ := cmd.Flags().GetBool("yes")

	resolvedNames, err := resolveRemoveNames(roles, "role", args, skipConfirm, stdout, stdin)
	if err != nil {
		return err
	}
	if resolvedNames == nil {
		return nil // user cancelled
	}

	// Confirm removal unless --yes flag is set
	if !skipConfirm {
		confirmed, err := confirmMultiRemoval(stdout, stdin, "role", resolvedNames, local)
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

	// Remove all roles and their order entries
	for _, name := range resolvedNames {
		delete(roles, name)
	}
	newOrder := make([]string, 0, len(order))
	for _, n := range order {
		if !toRemove[n] {
			newOrder = append(newOrder, n)
		}
	}

	// Write updated file once
	rolePath := filepath.Join(configDir, "roles.cue")
	if err := writeRolesFile(rolePath, roles, newOrder); err != nil {
		return fmt.Errorf("writing roles file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		for _, name := range resolvedNames {
			_, _ = fmt.Fprintf(stdout, "Removed role %q\n", name)
		}
	}

	return nil
}

