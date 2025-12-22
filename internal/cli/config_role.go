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

// Role add command flags
var (
	roleName        string
	roleDescription string
	roleFile        string
	roleCommand     string
	rolePrompt      string
	roleTags        []string
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
	}

	addConfigRoleListCommand(roleCmd)
	addConfigRoleAddCommand(roleCmd)
	addConfigRoleInfoCommand(roleCmd)
	addConfigRoleEditCommand(roleCmd)
	addConfigRoleRemoveCommand(roleCmd)
	addConfigRoleDefaultCommand(roleCmd)

	parent.AddCommand(roleCmd)
}

// addConfigRoleListCommand adds the list subcommand.
func addConfigRoleListCommand(parent *cobra.Command) {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all roles",
		Long: `List all configured roles.

Shows roles from both global and local configuration.
Use --local to show only local roles.`,
		RunE: runConfigRoleList,
	}

	parent.AddCommand(listCmd)
}

// runConfigRoleList lists all configured roles.
func runConfigRoleList(cmd *cobra.Command, _ []string) error {
	roles, err := loadRolesForScope(configLocal)
	if err != nil {
		return err
	}

	if len(roles) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No roles configured.")
		return nil
	}

	// Get default role
	defaultRole := ""
	if cfg, err := loadConfigForScope(configLocal); err == nil {
		defaultRole = getDefaultRoleFromConfig(cfg)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "Roles:")
	fmt.Fprintln(w)

	// Sort role names for consistent output
	var names []string
	for name := range roles {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		role := roles[name]
		marker := "  "
		if name == defaultRole {
			marker = "* "
		}
		if role.Description != "" {
			fmt.Fprintf(w, "%s%s - %s (%s)\n", marker, name, role.Description, role.Source)
		} else {
			fmt.Fprintf(w, "%s%s (%s)\n", marker, name, role.Source)
		}
	}

	if defaultRole != "" {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "* = default role\n")
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
		RunE: runConfigRoleAdd,
	}

	addCmd.Flags().StringVar(&roleName, "name", "", "Role name (identifier)")
	addCmd.Flags().StringVar(&roleDescription, "description", "", "Description")
	addCmd.Flags().StringVar(&roleFile, "file", "", "Path to role prompt file")
	addCmd.Flags().StringVar(&roleCommand, "command", "", "Command to generate prompt")
	addCmd.Flags().StringVar(&rolePrompt, "prompt", "", "Inline prompt text")
	addCmd.Flags().StringSliceVar(&roleTags, "tag", nil, "Tags")

	parent.AddCommand(addCmd)
}

// runConfigRoleAdd adds a new role configuration.
func runConfigRoleAdd(cmd *cobra.Command, _ []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	// Check if interactive
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	// Collect values
	name := roleName
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

	description := roleDescription
	if description == "" && isTTY {
		var err error
		description, err = promptString(stdout, stdin, "Description (optional)", "")
		if err != nil {
			return err
		}
	}

	// Content source: must have exactly one of file, command, or prompt
	file := roleFile
	command := roleCommand
	prompt := rolePrompt

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
		fmt.Fprintln(stdout, "  3. Inline prompt")
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
			prompt, err = promptString(stdout, stdin, "Prompt text", "")
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
	role := RoleConfig{
		Name:        name,
		Description: description,
		File:        file,
		Command:     command,
		Prompt:      prompt,
		Tags:        roleTags,
	}

	// Determine target directory
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	var scopeName string
	if configLocal {
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
	existingRoles, err := loadRolesFromDir(configDir)
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
	if err := writeRolesFile(rolePath, existingRoles); err != nil {
		return fmt.Errorf("writing roles file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		fmt.Fprintf(stdout, "Added role %q to %s config\n", name, scopeName)
		fmt.Fprintf(stdout, "Config: %s\n", rolePath)
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

	roles, err := loadRolesForScope(configLocal)
	if err != nil {
		return err
	}

	role, exists := roles[name]
	if !exists {
		return fmt.Errorf("role %q not found", name)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Role: %s\n", name)
	fmt.Fprintln(w, strings.Repeat("─", 40))
	fmt.Fprintf(w, "Source: %s\n", role.Source)

	if role.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", role.Description)
	}
	if role.File != "" {
		fmt.Fprintf(w, "File: %s\n", role.File)
	}
	if role.Command != "" {
		fmt.Fprintf(w, "Command: %s\n", role.Command)
	}
	if role.Prompt != "" {
		fmt.Fprintf(w, "Prompt: %s\n", truncatePrompt(role.Prompt, 100))
	}
	if len(role.Tags) > 0 {
		fmt.Fprintf(w, "Tags: %s\n", strings.Join(role.Tags, ", "))
	}

	return nil
}

// addConfigRoleEditCommand adds the edit subcommand.
func addConfigRoleEditCommand(parent *cobra.Command) {
	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit role configuration",
		Long: `Edit role configuration.

Without a name, opens the roles.cue file in $EDITOR.
With a name, provides interactive prompts to modify the role.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigRoleEdit,
	}

	parent.AddCommand(editCmd)
}

// runConfigRoleEdit edits a role configuration.
func runConfigRoleEdit(cmd *cobra.Command, args []string) error {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	if configLocal {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

	rolePath := filepath.Join(configDir, "roles.cue")

	// No name: open file in editor
	if len(args) == 0 {
		return openInEditor(rolePath)
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

	// Load existing roles
	roles, err := loadRolesFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading roles: %w", err)
	}

	role, exists := roles[name]
	if !exists {
		return fmt.Errorf("role %q not found in %s config", name, scopeString(configLocal))
	}

	// Prompt for each field with current value as default
	fmt.Fprintf(stdout, "Editing role %q (press Enter to keep current value)\n\n", name)

	newDescription, err := promptString(stdout, stdin, "Description", role.Description)
	if err != nil {
		return err
	}

	// For content source, show current and allow change
	fmt.Fprintln(stdout, "\nCurrent content source:")
	if role.File != "" {
		fmt.Fprintf(stdout, "  File: %s\n", role.File)
	}
	if role.Command != "" {
		fmt.Fprintf(stdout, "  Command: %s\n", role.Command)
	}
	if role.Prompt != "" {
		fmt.Fprintf(stdout, "  Prompt: %s\n", truncatePrompt(role.Prompt, 50))
	}

	fmt.Fprint(stdout, "Keep current? [Y/n] ")
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

		fmt.Fprintln(stdout, "\nNew content source:")
		fmt.Fprintln(stdout, "  1. File path")
		fmt.Fprintln(stdout, "  2. Command")
		fmt.Fprintln(stdout, "  3. Inline prompt")
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
			newPrompt, err = promptString(stdout, stdin, "Prompt text", "")
			if err != nil {
				return err
			}
		}
	}

	// Update role
	role.Description = newDescription
	role.File = newFile
	role.Command = newCommand
	role.Prompt = newPrompt
	roles[name] = role

	// Write updated file
	if err := writeRolesFile(rolePath, roles); err != nil {
		return fmt.Errorf("writing roles file: %w", err)
	}

	fmt.Fprintf(stdout, "\nUpdated role %q\n", name)
	return nil
}

// addConfigRoleRemoveCommand adds the remove subcommand.
func addConfigRoleRemoveCommand(parent *cobra.Command) {
	removeCmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a role",
		Long: `Remove a role configuration.

Removes the specified role from the configuration file.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigRoleRemove,
	}

	parent.AddCommand(removeCmd)
}

// runConfigRoleRemove removes a role configuration.
func runConfigRoleRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	if configLocal {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

	// Load existing roles
	roles, err := loadRolesFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading roles: %w", err)
	}

	if _, exists := roles[name]; !exists {
		return fmt.Errorf("role %q not found in %s config", name, scopeString(configLocal))
	}

	// Confirm removal
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	if isTTY {
		fmt.Fprintf(stdout, "Remove role %q from %s config? [y/N] ", name, scopeString(configLocal))
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

	// Remove role
	delete(roles, name)

	// Write updated file
	rolePath := filepath.Join(configDir, "roles.cue")
	if err := writeRolesFile(rolePath, roles); err != nil {
		return fmt.Errorf("writing roles file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		fmt.Fprintf(stdout, "Removed role %q\n", name)
	}

	return nil
}

// addConfigRoleDefaultCommand adds the default subcommand.
func addConfigRoleDefaultCommand(parent *cobra.Command) {
	defaultCmd := &cobra.Command{
		Use:   "default [name]",
		Short: "Set or show default role",
		Long: `Set or show the default role.

Without a name, shows the current default role.
With a name, sets that role as the default.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigRoleDefault,
	}

	parent.AddCommand(defaultCmd)
}

// runConfigRoleDefault sets or shows the default role.
func runConfigRoleDefault(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	if configLocal {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

	// Show current default
	if len(args) == 0 {
		cfg, err := loadConfigForScope(configLocal)
		if err != nil {
			fmt.Fprintln(stdout, "No default role set.")
			return nil
		}
		defaultRole := getDefaultRoleFromConfig(cfg)
		if defaultRole == "" {
			fmt.Fprintln(stdout, "No default role set.")
		} else {
			fmt.Fprintf(stdout, "Default role: %s\n", defaultRole)
		}
		return nil
	}

	// Set default
	name := args[0]

	// Verify role exists
	roles, err := loadRolesForScope(configLocal)
	if err != nil {
		return err
	}
	if _, exists := roles[name]; !exists {
		return fmt.Errorf("role %q not found", name)
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Update settings in settings.cue
	configPath := filepath.Join(configDir, "settings.cue")
	if err := writeDefaultRoleSetting(configPath, name); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		fmt.Fprintf(stdout, "Set default role to %q\n", name)
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
	Source      string // "global" or "local" - for display only
}

// loadRolesForScope loads roles from the appropriate scope.
func loadRolesForScope(localOnly bool) (map[string]RoleConfig, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return nil, fmt.Errorf("resolving config paths: %w", err)
	}

	roles := make(map[string]RoleConfig)

	if localOnly {
		if paths.LocalExists {
			localRoles, err := loadRolesFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, role := range localRoles {
				role.Source = "local"
				roles[name] = role
			}
		}
	} else {
		if paths.GlobalExists {
			globalRoles, err := loadRolesFromDir(paths.Global)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, role := range globalRoles {
				role.Source = "global"
				roles[name] = role
			}
		}
		if paths.LocalExists {
			localRoles, err := loadRolesFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, role := range localRoles {
				role.Source = "local"
				roles[name] = role
			}
		}
	}

	return roles, nil
}

// loadRolesFromDir loads roles from a specific directory.
func loadRolesFromDir(dir string) (map[string]RoleConfig, error) {
	roles := make(map[string]RoleConfig)

	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(dir)
	if err != nil {
		// If no CUE files exist, return empty map (not an error)
		if strings.Contains(err.Error(), "no CUE files") {
			return roles, nil
		}
		return roles, err
	}

	rolesVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyRoles))
	if !rolesVal.Exists() {
		return roles, nil
	}

	iter, err := rolesVal.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating roles: %w", err)
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

		roles[name] = role
	}

	return roles, nil
}

// writeRolesFile writes the roles configuration to a file.
func writeRolesFile(path string, roles map[string]RoleConfig) error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your role configuration\n\n")
	sb.WriteString("roles: {\n")

	// Sort role names for consistent output
	var names []string
	for name := range roles {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		role := roles[name]
		sb.WriteString(fmt.Sprintf("\t%q: {\n", name))

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

		sb.WriteString("\t}\n")
	}

	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// getDefaultRoleFromConfig extracts default_role from config value.
func getDefaultRoleFromConfig(cfg cue.Value) string {
	val := cfg.LookupPath(cue.ParsePath("settings.default_role"))
	if val.Exists() {
		s, _ := val.String()
		return s
	}
	return ""
}

// writeDefaultRoleSetting writes or updates the default_role setting.
func writeDefaultRoleSetting(path string, roleName string) error {
	var sb strings.Builder
	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your settings\n\n")
	sb.WriteString("settings: {\n")
	sb.WriteString(fmt.Sprintf("\tdefault_role: %q\n", roleName))
	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// truncatePrompt truncates a prompt for display.
func truncatePrompt(s string, max int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
