// NOTE(design): This file shares structural patterns with config_agent.go,
// config_context.go, and config_task.go (CUE field extraction, scope-aware loading,
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
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	_, _ = colorRoles.Fprint(w, "roles")
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
			_, _ = colorDim.Fprint(w, "- "+role.Description+" ")
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

// addConfigRoleAddCommand adds the add subcommand.
func addConfigRoleAddCommand(parent *cobra.Command) {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new role",
		Long: `Add a new role configuration.

Provide role details via flags or run interactively to be prompted for values.

A role must have exactly one content source: file, command, or prompt.

Examples:
  start config role add
  start config role add --name go-expert --file ~/.config/start/roles/go-expert.md
  start config role add --name reviewer --prompt "You are a code reviewer..."`,
		Args: cobra.NoArgs,
		RunE: runConfigRoleAdd,
	}

	addCmd.Flags().String("name", "", "Role name (identifier)")
	addCmd.Flags().String("description", "", "Description")
	addCmd.Flags().String("file", "", "Path to role prompt file")
	addCmd.Flags().String("command", "", "Command to generate prompt")
	addCmd.Flags().String("prompt", "", "Inline prompt text")
	addCmd.Flags().StringSlice("tag", nil, "Tags")
	addCmd.Flags().Bool("optional", false, "Skip gracefully when file is missing")

	parent.AddCommand(addCmd)
}

// runConfigRoleAdd adds a new role configuration.
func runConfigRoleAdd(cmd *cobra.Command, _ []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	// Check if interactive - only prompt for optional fields if no flags provided
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}
	// If any flags are set, skip prompts for optional fields
	hasFlags := anyFlagChanged(cmd, "name", "description", "file", "command", "prompt", "tag", "optional")
	interactive := isTTY && !hasFlags

	// Get flag values
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		if !isTTY {
			return fmt.Errorf("--name is required (run interactively or provide flag)")
		}
		var err error
		name, err = promptString(stdout, stdin, "Role name", "")
		if err != nil {
			return err
		}
	}
	if name == "" {
		return fmt.Errorf("role name is required")
	}

	description, _ := cmd.Flags().GetString("description")
	if description == "" && interactive {
		var err error
		description, err = promptString(stdout, stdin, "Description (optional)", "")
		if err != nil {
			return err
		}
	}

	// Content source: must have exactly one of file, command, or prompt
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
		_, _ = fmt.Fprintf(stdout, "Choice %s%s%s: ", colorCyan.Sprint("["), colorDim.Sprint("1"), colorCyan.Sprint("]"))

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

	// Build role struct
	tags, _ := cmd.Flags().GetStringSlice("tag")
	optional, _ := cmd.Flags().GetBool("optional")
	role := RoleConfig{
		Name:        name,
		Description: description,
		File:        file,
		Command:     command,
		Prompt:      prompt,
		Tags:        tags,
		Optional:    optional,
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

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Added role %q to %s config\n", name, scopeName)
		_, _ = fmt.Fprintf(stdout, "Config: %s\n", rolePath)
	}

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
	_, _ = colorRoles.Fprint(w, "roles")
	_, _ = fmt.Fprintf(w, "/%s\n", resolvedName)
	printSeparator(w)

	_, _ = colorDim.Fprint(w, "Source:")
	_, _ = fmt.Fprintf(w, " %s\n", role.Source)
	if role.Origin != "" {
		_, _ = colorDim.Fprint(w, "Origin:")
		_, _ = fmt.Fprintf(w, " %s\n", role.Origin)
	}
	if role.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = colorDim.Fprint(w, "Description:")
		_, _ = fmt.Fprintf(w, " %s\n", role.Description)
	}
	if role.File != "" {
		_, _ = colorDim.Fprint(w, "File:")
		_, _ = fmt.Fprintf(w, " %s\n", role.File)
	}
	if role.Command != "" {
		_, _ = colorDim.Fprint(w, "Command:")
		_, _ = fmt.Fprintf(w, " %s\n", role.Command)
	}
	if role.Prompt != "" {
		_, _ = colorDim.Fprint(w, "Prompt:")
		_, _ = fmt.Fprintf(w, " %s\n", truncatePrompt(role.Prompt, 100))
	}
	if len(role.Tags) > 0 {
		_, _ = colorDim.Fprint(w, "Tags:")
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
With a name and flags, updates only the specified fields.
With a name and no flags in a terminal, provides interactive prompts.

Examples:
  start config role edit
  start config role edit go-expert --description "Go programming expert"
  start config role edit reviewer --prompt "You are a code reviewer"`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigRoleEdit,
	}

	editCmd.Flags().String("description", "", "Description")
	editCmd.Flags().String("file", "", "Path to role prompt file")
	editCmd.Flags().String("command", "", "Command to generate prompt")
	editCmd.Flags().String("prompt", "", "Inline prompt text")
	editCmd.Flags().StringSlice("tag", nil, "Tags")
	editCmd.Flags().Bool("optional", false, "Skip gracefully when file is missing")

	parent.AddCommand(editCmd)
}

// runConfigRoleEdit edits a role configuration.
func runConfigRoleEdit(cmd *cobra.Command, args []string) error {
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

	rolePath := filepath.Join(configDir, "roles.cue")

	// No name: open file in editor
	if len(args) == 0 {
		return openInEditor(rolePath)
	}

	// Named edit
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	// Load existing roles
	roles, order, err := loadRolesFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading roles: %w", err)
	}

	resolvedName, role, err := resolveInstalledName(roles, "role", name)
	if err != nil {
		return err
	}

	// Check if any edit flags are provided
	hasEditFlags := anyFlagChanged(cmd, "description", "file", "command", "prompt", "tag", "optional")

	if hasEditFlags {
		// Non-interactive flag-based update
		if cmd.Flags().Changed("description") {
			role.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("file") {
			role.File, _ = cmd.Flags().GetString("file")
		}
		if cmd.Flags().Changed("command") {
			role.Command, _ = cmd.Flags().GetString("command")
		}
		if cmd.Flags().Changed("prompt") {
			role.Prompt, _ = cmd.Flags().GetString("prompt")
		}
		if cmd.Flags().Changed("tag") {
			role.Tags, _ = cmd.Flags().GetStringSlice("tag")
		}
		if cmd.Flags().Changed("optional") {
			role.Optional, _ = cmd.Flags().GetBool("optional")
		}

		roles[resolvedName] = role

		if err := writeRolesFile(rolePath, roles, order); err != nil {
			return fmt.Errorf("writing roles file: %w", err)
		}

		flags := getFlags(cmd)
		if !flags.Quiet {
			_, _ = fmt.Fprintf(stdout, "Updated role %q\n", resolvedName)
		}
		return nil
	}

	// No flags: require TTY for interactive editing
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}
	if !isTTY {
		return fmt.Errorf("interactive editing requires a terminal")
	}

	// Prompt for each field with current value as default
	_, _ = fmt.Fprintf(stdout, "Editing role %q %s%s%s\n\n", resolvedName, colorCyan.Sprint("("), colorDim.Sprint("press Enter to keep current value"), colorCyan.Sprint(")"))

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

	_, _ = fmt.Fprintf(stdout, "Keep current? %s%s%s ", colorCyan.Sprint("["), colorDim.Sprint("Y/n"), colorCyan.Sprint("]"))
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
		// Clear existing and prompt for new
		newFile = ""
		newCommand = ""
		newPrompt = ""

		_, _ = fmt.Fprintln(stdout, "\nNew content source:")
		_, _ = fmt.Fprintln(stdout, "  1. File path")
		_, _ = fmt.Fprintln(stdout, "  2. Command")
		_, _ = fmt.Fprintln(stdout, "  3. Inline prompt")
		_, _ = fmt.Fprintf(stdout, "Choice %s%s%s: ", colorCyan.Sprint("["), colorDim.Sprint("1"), colorCyan.Sprint("]"))

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
			newPrompt, err = promptText(stdout, stdin, "Prompt text", role.Prompt)
			if err != nil {
				return err
			}
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
		Use:     "remove <name>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a role",
		Long: `Remove a role configuration.

Removes the specified role from the configuration file.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigRoleRemove,
	}

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(removeCmd)
}

// runConfigRoleRemove removes a role configuration.
func runConfigRoleRemove(cmd *cobra.Command, args []string) error {
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

	// Load existing roles
	roles, order, err := loadRolesFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading roles: %w", err)
	}

	resolvedName, _, err := resolveInstalledName(roles, "role", name)
	if err != nil {
		return err
	}

	// Confirm removal unless --yes flag is set
	skipConfirm, _ := cmd.Flags().GetBool("yes")
	if !skipConfirm {
		isTTY := false
		if f, ok := stdin.(*os.File); ok {
			isTTY = term.IsTerminal(int(f.Fd()))
		}

		if isTTY {
			_, _ = fmt.Fprintf(stdout, "Remove role %q from %s config? %s%s%s ", resolvedName, scopeString(local), colorCyan.Sprint("["), colorDim.Sprint("y/N"), colorCyan.Sprint("]"))
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

	// Remove role and its order entry
	delete(roles, resolvedName)
	newOrder := make([]string, 0, len(order))
	for _, n := range order {
		if n != resolvedName {
			newOrder = append(newOrder, n)
		}
	}

	// Write updated file
	rolePath := filepath.Join(configDir, "roles.cue")
	if err := writeRolesFile(rolePath, roles, newOrder); err != nil {
		return fmt.Errorf("writing roles file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Removed role %q\n", resolvedName)
	}

	return nil
}

// RoleConfig represents a role configuration for editing.
type RoleConfig struct {
	Name        string
	Description string
	File        string
	Command     string
	Prompt      string
	Tags        []string
	Optional    bool   // If true, skip gracefully when file is missing
	Source      string // "global" or "local" - for display only
	Origin      string // Registry module path when installed from registry
}

// loadRolesForScope loads roles from the appropriate scope.
// Returns the roles map, names in definition order, and any error.
// Order: global roles first (in definition order), then local roles (in definition order).
// Local roles override global roles with the same name but retain their global position.
func loadRolesForScope(localOnly bool) (map[string]RoleConfig, []string, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return nil, nil, fmt.Errorf("resolving config paths: %w", err)
	}

	roles := make(map[string]RoleConfig)
	var order []string
	seen := make(map[string]bool)

	if localOnly {
		if paths.LocalExists {
			localRoles, localOrder, err := loadRolesFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, nil, err
			}
			for _, name := range localOrder {
				role := localRoles[name]
				role.Source = "local"
				roles[name] = role
				order = append(order, name)
			}
		}
	} else {
		if paths.GlobalExists {
			globalRoles, globalOrder, err := loadRolesFromDir(paths.Global)
			if err != nil && !os.IsNotExist(err) {
				return nil, nil, err
			}
			for _, name := range globalOrder {
				role := globalRoles[name]
				role.Source = "global"
				roles[name] = role
				order = append(order, name)
				seen[name] = true
			}
		}
		if paths.LocalExists {
			localRoles, localOrder, err := loadRolesFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, nil, err
			}
			for _, name := range localOrder {
				role := localRoles[name]
				role.Source = "local"
				roles[name] = role
				// Only add to order if not already present from global
				if !seen[name] {
					order = append(order, name)
				}
			}
		}
	}

	return roles, order, nil
}

// loadRolesFromDir loads roles from a specific directory.
// Returns the roles map, names in definition order, and any error.
func loadRolesFromDir(dir string) (map[string]RoleConfig, []string, error) {
	roles := make(map[string]RoleConfig)
	var order []string

	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(dir)
	if err != nil {
		// If no CUE files exist, return empty map (not an error)
		if errors.Is(err, internalcue.ErrNoCUEFiles) {
			return roles, order, nil
		}
		return roles, order, err
	}

	rolesVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyRoles))
	if !rolesVal.Exists() {
		return roles, order, nil
	}

	iter, err := rolesVal.Fields()
	if err != nil {
		return nil, nil, fmt.Errorf("iterating roles: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		val := iter.Value()

		role := RoleConfig{Name: name}

		if v := val.LookupPath(cue.ParsePath("description")); v.Exists() {
			role.Description, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("file")); v.Exists() {
			role.File, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("command")); v.Exists() {
			role.Command, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("prompt")); v.Exists() {
			role.Prompt, _ = v.String()
		}

		// Load tags
		if tagsVal := val.LookupPath(cue.ParsePath("tags")); tagsVal.Exists() {
			tagIter, err := tagsVal.List()
			if err == nil {
				for tagIter.Next() {
					if s, err := tagIter.Value().String(); err == nil {
						role.Tags = append(role.Tags, s)
					}
				}
			}
		}

		// Load origin (registry provenance)
		if v := val.LookupPath(cue.ParsePath("origin")); v.Exists() {
			role.Origin, _ = v.String()
		}

		// Load optional field
		if v := val.LookupPath(cue.ParsePath("optional")); v.Exists() {
			role.Optional, _ = v.Bool()
		}

		roles[name] = role
		order = append(order, name)
	}

	return roles, order, nil
}

// writeRolesFile writes the roles configuration to a file.
// Fields are written in the provided order.
func writeRolesFile(path string, roles map[string]RoleConfig, order []string) error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your role configuration\n\n")
	sb.WriteString("roles: {\n")

	for _, name := range order {
		role := roles[name]
		sb.WriteString(fmt.Sprintf("\t%q: {\n", name))

		// Write origin first if present (registry provenance)
		if role.Origin != "" {
			sb.WriteString(fmt.Sprintf("\t\torigin: %q\n", role.Origin))
		}
		if role.Description != "" {
			sb.WriteString(fmt.Sprintf("\t\tdescription: %q\n", role.Description))
		}
		if role.File != "" {
			sb.WriteString(fmt.Sprintf("\t\tfile: %q\n", role.File))
		}
		if role.Command != "" {
			sb.WriteString(fmt.Sprintf("\t\tcommand: %q\n", role.Command))
		}
		if role.Prompt != "" {
			// Use multi-line string for long prompts
			if strings.Contains(role.Prompt, "\n") || len(role.Prompt) > 80 {
				sb.WriteString("\t\tprompt: \"\"\"\n")
				for _, line := range strings.Split(role.Prompt, "\n") {
					sb.WriteString(fmt.Sprintf("\t\t\t%s\n", line))
				}
				sb.WriteString("\t\t\t\"\"\"\n")
			} else {
				sb.WriteString(fmt.Sprintf("\t\tprompt: %q\n", role.Prompt))
			}
		}
		if len(role.Tags) > 0 {
			sb.WriteString("\t\ttags: [")
			for i, tag := range role.Tags {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%q", tag))
			}
			sb.WriteString("]\n")
		}
		if role.Optional {
			sb.WriteString("\t\toptional: true\n")
		}

		sb.WriteString("\t}\n")
	}

	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
