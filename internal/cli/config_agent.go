package cli

import (
	"errors"
	"fmt"
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

// addConfigAgentCommand adds the agent subcommand group to the config command.
func addConfigAgentCommand(parent *cobra.Command) {
	agentCmd := &cobra.Command{
		Use:     "agent",
		Aliases: []string{"agents"},
		Short:   "Manage agent configuration",
		Long: `Manage AI agent configurations.

Agents define the AI CLI tools that start can use (e.g., claude, gemini, aider).
Each agent specifies a binary, command template, and available models.`,
		RunE: runConfigAgent,
	}

	addConfigAgentListCommand(agentCmd)
	addConfigAgentAddCommand(agentCmd)
	addConfigAgentInfoCommand(agentCmd)
	addConfigAgentEditCommand(agentCmd)
	addConfigAgentRemoveCommand(agentCmd)
	addConfigAgentDefaultCommand(agentCmd)

	parent.AddCommand(agentCmd)
}

// runConfigAgent runs list by default, handles help subcommand.
func runConfigAgent(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	if len(args) > 0 {
		return unknownCommandError("start config agent", args[0])
	}
	return runConfigAgentList(cmd, args)
}

// addConfigAgentListCommand adds the list subcommand.
func addConfigAgentListCommand(parent *cobra.Command) {
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all agents",
		Long: `List all configured agents.

Shows agents from both global and local configuration.
Use --local to show only local agents.`,
		RunE: runConfigAgentList,
	}

	parent.AddCommand(listCmd)
}

// runConfigAgentList lists all configured agents.
func runConfigAgentList(cmd *cobra.Command, _ []string) error {
	local := getFlags(cmd).Local
	agents, order, err := loadAgentsForScope(local)
	if err != nil {
		return err
	}

	if len(agents) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No agents configured.")
		return nil
	}

	// Get default agent
	defaultAgent := ""
	if cfg, err := loadConfigForScope(local); err == nil {
		defaultAgent = getDefaultAgentFromConfig(cfg)
	}

	w := cmd.OutOrStdout()
	_, _ = tui.ColorAgents.Fprint(w, "agents")
	_, _ = fmt.Fprintln(w, "/")
	_, _ = fmt.Fprintln(w)

	for _, name := range order {
		agent := agents[name]
		marker := "  "
		if name == defaultAgent {
			marker = tui.ColorInstalled.Sprint("→") + " "
		}
		source := agent.Source
		if agent.Origin != "" {
			source += ", registry"
		}
		if agent.Description != "" {
			_, _ = fmt.Fprintf(w, "%s%s ", marker, name)
			_, _ = tui.ColorDim.Fprint(w, "- "+agent.Description+" ")
			_, _ = fmt.Fprintln(w, tui.Annotate("%s", source))
		} else {
			_, _ = fmt.Fprintf(w, "%s%s ", marker, name)
			_, _ = fmt.Fprintln(w, tui.Annotate("%s", source))
		}
	}

	return nil
}

// addConfigAgentAddCommand adds the add subcommand.
func addConfigAgentAddCommand(parent *cobra.Command) {
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new agent",
		Long: `Add a new agent configuration.

Provide agent details via flags or run interactively to be prompted for values.

Examples:
  start config agent add
  start config agent add --name gemini --bin gemini --command 'gemini "{{.prompt}}"'
  start config agent add --local --name project-agent --bin claude --command 'claude "{{.prompt}}"'`,
		Args: cobra.NoArgs,
		RunE: runConfigAgentAdd,
	}

	addCmd.Flags().String("name", "", "Agent name (identifier)")
	addCmd.Flags().String("bin", "", "Binary executable name")
	addCmd.Flags().String("command", "", "Command template")
	addCmd.Flags().String("default-model", "", "Default model alias")
	addCmd.Flags().String("description", "", "Description")
	addCmd.Flags().StringSlice("model", nil, "Model mapping (alias=model-id)")
	addCmd.Flags().StringSlice("tag", nil, "Tags")

	parent.AddCommand(addCmd)
}

// runConfigAgentAdd adds a new agent configuration.
func runConfigAgentAdd(cmd *cobra.Command, _ []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	// Check if interactive - only prompt for optional fields if no flags provided
	isTTY := isTerminal(stdin)
	// If any flags are set, skip prompts for optional fields
	hasFlags := anyFlagChanged(cmd, "name", "bin", "command", "default-model", "description", "model", "tag")
	interactive := isTTY && !hasFlags

	// Get flag values
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		if !isTTY {
			return fmt.Errorf("--name is required (run interactively or provide flag)")
		}
		var err error
		name, err = promptString(stdout, stdin, "Agent name", "")
		if err != nil {
			return err
		}
	}
	if name == "" {
		return fmt.Errorf("agent name is required")
	}

	bin, _ := cmd.Flags().GetString("bin")
	if bin == "" && interactive {
		var err error
		bin, err = promptString(stdout, stdin, "Binary (optional)", "")
		if err != nil {
			return err
		}
	}

	command, _ := cmd.Flags().GetString("command")
	if command == "" {
		if !isTTY {
			return fmt.Errorf("--command is required (run interactively or provide flag)")
		}
		var err error
		defaultCmd := fmt.Sprintf("%s \"{{.prompt}}\"", bin)
		command, err = promptString(stdout, stdin, "Command template", defaultCmd)
		if err != nil {
			return err
		}
	}
	if command == "" {
		return fmt.Errorf("command template is required")
	}

	defaultModel, _ := cmd.Flags().GetString("default-model")
	if defaultModel == "" && interactive {
		var err error
		defaultModel, err = promptString(stdout, stdin, "Default model (optional)", "")
		if err != nil {
			return err
		}
	}

	description, _ := cmd.Flags().GetString("description")
	if description == "" && interactive {
		var err error
		description, err = promptString(stdout, stdin, "Description (optional)", "")
		if err != nil {
			return err
		}
	}

	// Parse models
	agentModels, _ := cmd.Flags().GetStringSlice("model")
	agentTags, _ := cmd.Flags().GetStringSlice("tag")
	models := make(map[string]string)
	for _, m := range agentModels {
		parts := strings.SplitN(m, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid model format %q (expected alias=model-id)", m)
		}
		models[parts[0]] = parts[1]
	}

	// Build agent struct
	agent := AgentConfig{
		Name:         name,
		Bin:          bin,
		Command:      command,
		DefaultModel: defaultModel,
		Description:  description,
		Models:       models,
		Tags:         agentTags,
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

	// Load existing agents from target directory
	existingAgents, _, err := loadAgentsFromDir(configDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("loading existing agents: %w", err)
	}

	// Check for duplicate
	if _, exists := existingAgents[name]; exists {
		return fmt.Errorf("agent %q already exists in %s config", name, scopeName)
	}

	// Add new agent
	existingAgents[name] = agent

	// Write agents file
	agentPath := filepath.Join(configDir, "agents.cue")
	if err := writeAgentsFile(agentPath, existingAgents); err != nil {
		return fmt.Errorf("writing agents file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Added agent %q to %s config\n", name, scopeName)
		_, _ = fmt.Fprintf(stdout, "Config: %s\n", agentPath)
	}

	return nil
}

// addConfigAgentInfoCommand adds the info subcommand.
func addConfigAgentInfoCommand(parent *cobra.Command) {
	infoCmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show agent details",
		Long: `Show detailed information about an agent.

Displays all configuration fields for the specified agent.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigAgentInfo,
	}

	parent.AddCommand(infoCmd)
}

// runConfigAgentInfo shows detailed information about an agent.
func runConfigAgentInfo(cmd *cobra.Command, args []string) error {
	name := args[0]
	local := getFlags(cmd).Local

	agents, _, err := loadAgentsForScope(local)
	if err != nil {
		return err
	}

	resolvedName, agent, err := resolveInstalledName(agents, "agent", name)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(w)
	_, _ = tui.ColorAgents.Fprint(w, "agents")
	_, _ = fmt.Fprintf(w, "/%s\n", resolvedName)
	printSeparator(w)

	_, _ = tui.ColorDim.Fprint(w, "Source:")
	_, _ = fmt.Fprintf(w, " %s\n", agent.Source)
	if agent.Origin != "" {
		_, _ = tui.ColorDim.Fprint(w, "Origin:")
		_, _ = fmt.Fprintf(w, " %s\n", agent.Origin)
	}
	if agent.Bin != "" {
		_, _ = tui.ColorDim.Fprint(w, "Bin:")
		_, _ = fmt.Fprintf(w, " %s\n", agent.Bin)
	}
	_, _ = tui.ColorDim.Fprint(w, "Command:")
	_, _ = fmt.Fprintf(w, " %s\n", agent.Command)

	if agent.DefaultModel != "" {
		_, _ = tui.ColorDim.Fprint(w, "Default Model:")
		_, _ = fmt.Fprintf(w, " %s\n", agent.DefaultModel)
	}
	if agent.Description != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = tui.ColorDim.Fprint(w, "Description:")
		_, _ = fmt.Fprintf(w, " %s\n", agent.Description)
	}
	if len(agent.Tags) > 0 {
		_, _ = tui.ColorDim.Fprint(w, "Tags:")
		_, _ = fmt.Fprintf(w, " %s\n", strings.Join(agent.Tags, ", "))
	}
	if len(agent.Models) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = tui.ColorDim.Fprintln(w, "Models:")
		var aliases []string
		for alias := range agent.Models {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		for _, alias := range aliases {
			_, _ = fmt.Fprintf(w, "  %s ", alias)
			_, _ = tui.ColorBlue.Fprint(w, "->")
			_, _ = fmt.Fprint(w, " ")
			_, _ = tui.ColorDim.Fprintf(w, "%s\n", agent.Models[alias])
		}
	}
	printSeparator(w)

	return nil
}

// addConfigAgentEditCommand adds the edit subcommand.
func addConfigAgentEditCommand(parent *cobra.Command) {
	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit agent configuration",
		Long: `Edit agent configuration.

Without a name, opens the agents.cue file in $EDITOR.
With a name and flags, updates only the specified fields.
With a name and no flags in a terminal, provides interactive prompts.

Examples:
  start config agent edit
  start config agent edit claude --bin claude-code
  start config agent edit gemini --default-model flash --tag ai,google`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigAgentEdit,
	}

	editCmd.Flags().String("bin", "", "Binary executable name")
	editCmd.Flags().String("command", "", "Command template")
	editCmd.Flags().String("default-model", "", "Default model alias")
	editCmd.Flags().String("description", "", "Description")
	editCmd.Flags().StringSlice("model", nil, "Model mapping (alias=model-id)")
	editCmd.Flags().StringSlice("tag", nil, "Tags")

	parent.AddCommand(editCmd)
}

// runConfigAgentEdit edits an agent configuration.
func runConfigAgentEdit(cmd *cobra.Command, args []string) error {
	local := getFlags(cmd).Local
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(local)

	agentPath := filepath.Join(configDir, "agents.cue")

	// No name: open file in editor
	if len(args) == 0 {
		return openInEditor(agentPath)
	}

	// Named edit
	name := args[0]
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	// Load existing agents
	agents, _, err := loadAgentsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}

	resolvedName, agent, err := resolveInstalledName(agents, "agent", name)
	if err != nil {
		return err
	}

	// Check if any edit flags are provided
	hasEditFlags := anyFlagChanged(cmd, "bin", "command", "default-model", "description", "model", "tag")

	if hasEditFlags {
		// Non-interactive flag-based update
		if cmd.Flags().Changed("bin") {
			agent.Bin, _ = cmd.Flags().GetString("bin")
		}
		if cmd.Flags().Changed("command") {
			agent.Command, _ = cmd.Flags().GetString("command")
		}
		if cmd.Flags().Changed("default-model") {
			agent.DefaultModel, _ = cmd.Flags().GetString("default-model")
		}
		if cmd.Flags().Changed("description") {
			agent.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("model") {
			// Replace models entirely when specified
			agentModels, _ := cmd.Flags().GetStringSlice("model")
			agent.Models = make(map[string]string)
			for _, m := range agentModels {
				parts := strings.SplitN(m, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid model format %q (expected alias=model-id)", m)
				}
				agent.Models[parts[0]] = parts[1]
			}
		}
		if cmd.Flags().Changed("tag") {
			agent.Tags, _ = cmd.Flags().GetStringSlice("tag")
		}

		agents[resolvedName] = agent

		if err := writeAgentsFile(agentPath, agents); err != nil {
			return fmt.Errorf("writing agents file: %w", err)
		}

		flags := getFlags(cmd)
		if !flags.Quiet {
			_, _ = fmt.Fprintf(stdout, "Updated agent %q\n", resolvedName)
		}
		return nil
	}

	// No flags: require TTY for interactive editing
	isTTY := isTerminal(stdin)
	if !isTTY {
		return fmt.Errorf("interactive editing requires a terminal")
	}

	// Prompt for each field with current value as default
	_, _ = fmt.Fprintf(stdout, "Editing agent %q %s\n\n", resolvedName, tui.Annotate("press Enter to keep current value"))

	newBin, err := promptString(stdout, stdin, "Binary", agent.Bin)
	if err != nil {
		return err
	}
	if newBin == "" {
		newBin = agent.Bin
	}

	newCommand, err := promptString(stdout, stdin, "Command template", agent.Command)
	if err != nil {
		return err
	}
	if newCommand == "" {
		newCommand = agent.Command
	}

	newDescription, err := promptString(stdout, stdin, "Description", agent.Description)
	if err != nil {
		return err
	}

	// Prompt for models before default model so the user can see available choices
	_, _ = fmt.Fprintln(stdout)
	newModels, err := promptModels(stdout, stdin, agent.Models)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(stdout)
	newDefaultModel, err := promptDefaultModel(stdout, stdin, agent.DefaultModel, newModels)
	if err != nil {
		return err
	}

	// Prompt for tags
	_, _ = fmt.Fprintln(stdout)
	newTags, err := promptTags(stdout, stdin, agent.Tags)
	if err != nil {
		return err
	}

	// Update agent
	agent.Bin = newBin
	agent.Command = newCommand
	agent.DefaultModel = newDefaultModel
	agent.Description = newDescription
	agent.Models = newModels
	agent.Tags = newTags
	agents[resolvedName] = agent

	// Write updated file
	if err := writeAgentsFile(agentPath, agents); err != nil {
		return fmt.Errorf("writing agents file: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "\nUpdated agent %q\n", resolvedName)
	return nil
}

// addConfigAgentRemoveCommand adds the remove subcommand.
func addConfigAgentRemoveCommand(parent *cobra.Command) {
	removeCmd := &cobra.Command{
		Use:     "remove <name>...",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove one or more agents",
		Long: `Remove one or more agent configurations.

Removes the specified agents from the configuration file.`,
		Args: cobra.MinimumNArgs(1),
		RunE: runConfigAgentRemove,
	}

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(removeCmd)
}

// runConfigAgentRemove removes one or more agent configurations.
func runConfigAgentRemove(cmd *cobra.Command, args []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(local)

	// Load existing agents
	agents, _, err := loadAgentsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}

	skipConfirm, _ := cmd.Flags().GetBool("yes")

	resolvedNames, err := resolveRemoveNames(agents, "agent", args, skipConfirm, stdout, stdin)
	if err != nil {
		return err
	}
	if resolvedNames == nil {
		return nil // user cancelled
	}

	// Confirm removal unless --yes flag is set
	if !skipConfirm {
		confirmed, err := confirmMultiRemoval(stdout, stdin, "agent", resolvedNames, local)
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	// Remove all agents
	for _, name := range resolvedNames {
		delete(agents, name)
	}

	// Write updated file once
	agentPath := filepath.Join(configDir, "agents.cue")
	if err := writeAgentsFile(agentPath, agents); err != nil {
		return fmt.Errorf("writing agents file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		for _, name := range resolvedNames {
			_, _ = fmt.Fprintf(stdout, "Removed agent %q\n", name)
		}
	}

	return nil
}

// addConfigAgentDefaultCommand adds the default subcommand.
func addConfigAgentDefaultCommand(parent *cobra.Command) {
	defaultCmd := &cobra.Command{
		Use:   "default [name]",
		Short: "Set, show, or unset default agent",
		Long: `Set, show, or unset the default agent.

Without a name, shows the current default agent.
With a name, sets that agent as the default.
With --unset, removes the default agent setting.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigAgentDefault,
	}

	defaultCmd.Flags().Bool("unset", false, "Remove the default agent setting")

	parent.AddCommand(defaultCmd)
}

// runConfigAgentDefault sets, shows, or unsets the default agent.
func runConfigAgentDefault(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local
	unset, _ := cmd.Flags().GetBool("unset")

	// Validate: --unset and name are mutually exclusive
	if unset && len(args) > 0 {
		return fmt.Errorf("cannot use --unset with an agent name")
	}

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(local)

	// Unset default
	if unset {
		settings, _ := loadSettingsFromDir(configDir)
		if settings == nil {
			settings = make(map[string]string)
		}
		delete(settings, "default_agent")

		settingsPath := filepath.Join(configDir, "settings.cue")
		if err := writeSettingsFile(settingsPath, settings); err != nil {
			return fmt.Errorf("writing settings file: %w", err)
		}

		flags := getFlags(cmd)
		if !flags.Quiet {
			_, _ = fmt.Fprintln(stdout, "Unset default agent")
		}
		return nil
	}

	// Show current default
	if len(args) == 0 {
		cfg, err := loadConfigForScope(local)
		if err != nil {
			_, _ = fmt.Fprintln(stdout, "No default agent set.")
			return nil
		}
		defaultAgent := getDefaultAgentFromConfig(cfg)
		if defaultAgent == "" {
			_, _ = fmt.Fprintln(stdout, "No default agent set.")
		} else {
			_, _ = fmt.Fprintf(stdout, "Default agent: %s\n", defaultAgent)
		}
		return nil
	}

	// Set default
	name := args[0]

	// Verify agent exists
	agents, _, err := loadAgentsForScope(local)
	if err != nil {
		return err
	}
	resolvedName, _, err := resolveInstalledName(agents, "agent", name)
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Load existing settings, set default_agent, write back
	settings, _ := loadSettingsFromDir(configDir)
	if settings == nil {
		settings = make(map[string]string)
	}
	settings["default_agent"] = resolvedName

	settingsPath := filepath.Join(configDir, "settings.cue")
	if err := writeSettingsFile(settingsPath, settings); err != nil {
		return fmt.Errorf("writing settings file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Set default agent to %q\n", resolvedName)
	}

	return nil
}

// AgentConfig represents an agent configuration for editing.
type AgentConfig struct {
	Name         string
	Bin          string
	Command      string
	DefaultModel string
	Description  string
	Models       map[string]string
	Tags         []string
	Source       string // "global" or "local" - for display only
	Origin       string // Registry module path when installed from registry
}

// loadAgentsForScope loads agents from the appropriate scope.
// Returns the agents map, names in definition order, and any error.
func loadAgentsForScope(localOnly bool) (map[string]AgentConfig, []string, error) {
	return loadForScope(localOnly, loadAgentsFromDir, func(a *AgentConfig, s string) { a.Source = s })
}

// loadAgentsFromDir loads agents from a specific directory.
// Returns the agents map, names in definition order, and any error.
func loadAgentsFromDir(dir string) (map[string]AgentConfig, []string, error) {
	agents := make(map[string]AgentConfig)
	var order []string

	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(dir)
	if err != nil {
		// If no CUE files exist, return empty map (not an error)
		if errors.Is(err, internalcue.ErrNoCUEFiles) {
			return agents, order, nil
		}
		return agents, order, err
	}

	agentsVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyAgents))
	if !agentsVal.Exists() {
		return agents, order, nil
	}

	iter, err := agentsVal.Fields()
	if err != nil {
		return nil, nil, fmt.Errorf("iterating agents: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		val := iter.Value()

		agent := AgentConfig{Name: name}

		if v := val.LookupPath(cue.ParsePath("bin")); v.Exists() {
			agent.Bin, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("command")); v.Exists() {
			agent.Command, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("default_model")); v.Exists() {
			agent.DefaultModel, _ = v.String()
		}
		if v := val.LookupPath(cue.ParsePath("description")); v.Exists() {
			agent.Description, _ = v.String()
		}

		agent.Tags = extractTags(val)

		// Load models
		if modelsVal := val.LookupPath(cue.ParsePath("models")); modelsVal.Exists() {
			agent.Models = make(map[string]string)
			modelIter, err := modelsVal.Fields()
			if err == nil {
				for modelIter.Next() {
					alias := modelIter.Selector().Unquoted()
					if s, err := modelIter.Value().String(); err == nil {
						agent.Models[alias] = s
					}
				}
			}
		}

		// Load origin (registry provenance)
		if v := val.LookupPath(cue.ParsePath("origin")); v.Exists() {
			agent.Origin, _ = v.String()
		}

		agents[name] = agent
		order = append(order, name)
	}

	return agents, order, nil
}

// writeAgentsFile writes the agents configuration to a file.
// Agents are not order-dependent, so names are sorted alphabetically for consistent output.
func writeAgentsFile(path string, agents map[string]AgentConfig) error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your agent configuration\n\n")
	sb.WriteString("agents: {\n")

	// Sort agent names for consistent output
	var names []string
	for name := range agents {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		agent := agents[name]
		sb.WriteString(fmt.Sprintf("\t%q: {\n", name))

		// Write origin first if present (registry provenance)
		if agent.Origin != "" {
			sb.WriteString(fmt.Sprintf("\t\torigin: %q\n", agent.Origin))
		}
		if agent.Bin != "" {
			sb.WriteString(fmt.Sprintf("\t\tbin:     %q\n", agent.Bin))
		}
		sb.WriteString(fmt.Sprintf("\t\tcommand: %q\n", agent.Command))

		if agent.DefaultModel != "" {
			sb.WriteString(fmt.Sprintf("\t\tdefault_model: %q\n", agent.DefaultModel))
		}
		if agent.Description != "" {
			sb.WriteString(fmt.Sprintf("\t\tdescription: %q\n", agent.Description))
		}
		writeCUETags(&sb, agent.Tags)
		if len(agent.Models) > 0 {
			sb.WriteString("\t\tmodels: {\n")
			var aliases []string
			for alias := range agent.Models {
				aliases = append(aliases, alias)
			}
			sort.Strings(aliases)
			for _, alias := range aliases {
				sb.WriteString(fmt.Sprintf("\t\t\t%q: %q\n", alias, agent.Models[alias]))
			}
			sb.WriteString("\t\t}\n")
		}

		sb.WriteString("\t}\n")
	}

	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// loadConfigForScope loads the settings.cue settings for the scope.
func loadConfigForScope(localOnly bool) (cue.Value, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return cue.Value{}, err
	}

	loader := internalcue.NewLoader()

	var dirs []string
	if localOnly {
		if paths.LocalExists {
			dirs = []string{paths.Local}
		}
	} else {
		dirs = paths.ForScope(config.ScopeMerged)
	}

	if len(dirs) == 0 {
		return cue.Value{}, fmt.Errorf("no config found")
	}

	result, err := loader.Load(dirs)
	if err != nil {
		return cue.Value{}, err
	}

	return result.Value, nil
}

// getDefaultAgentFromConfig extracts default_agent from config value.
func getDefaultAgentFromConfig(cfg cue.Value) string {
	val := cfg.LookupPath(cue.ParsePath("settings.default_agent"))
	if val.Exists() {
		s, _ := val.String()
		return s
	}
	return ""
}
