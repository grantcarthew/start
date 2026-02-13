package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	agents, err := loadAgentsForScope(local)
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
	_, _ = colorAgents.Fprint(w, "agents")
	_, _ = fmt.Fprintln(w, "/")
	_, _ = fmt.Fprintln(w)

	// Sort agent names for consistent output
	var names []string
	for name := range agents {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		agent := agents[name]
		marker := "  "
		if name == defaultAgent {
			marker = colorInstalled.Sprint("→") + " "
		}
		source := agent.Source
		if agent.Origin != "" {
			source += ", registry"
		}
		if agent.Description != "" {
			_, _ = fmt.Fprintf(w, "%s%s ", marker, name)
			_, _ = colorDim.Fprint(w, "- "+agent.Description+" ")
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, source)
			_, _ = colorCyan.Fprintln(w, ")")
		} else {
			_, _ = fmt.Fprintf(w, "%s%s ", marker, name)
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, source)
			_, _ = colorCyan.Fprintln(w, ")")
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
	isTTY := false
	if f, ok := stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}
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

	// Load existing agents from target directory
	existingAgents, err := loadAgentsFromDir(configDir)
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

	agents, err := loadAgentsForScope(local)
	if err != nil {
		return err
	}

	agent, exists := agents[name]
	if !exists {
		return fmt.Errorf("agent %q not found", name)
	}

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "Agent: %s\n", name)
	_, _ = fmt.Fprintln(w, strings.Repeat("─", 40))
	_, _ = fmt.Fprintf(w, "Source: %s\n", agent.Source)
	if agent.Bin != "" {
		_, _ = fmt.Fprintf(w, "Bin: %s\n", agent.Bin)
	}
	_, _ = fmt.Fprintf(w, "Command: %s\n", agent.Command)

	if agent.DefaultModel != "" {
		_, _ = fmt.Fprintf(w, "Default Model: %s\n", agent.DefaultModel)
	}
	if agent.Description != "" {
		_, _ = fmt.Fprintf(w, "Description: %s\n", agent.Description)
	}
	if len(agent.Tags) > 0 {
		_, _ = fmt.Fprintf(w, "Tags: %s\n", strings.Join(agent.Tags, ", "))
	}
	if len(agent.Models) > 0 {
		_, _ = fmt.Fprintln(w, "Models:")
		var aliases []string
		for alias := range agent.Models {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		for _, alias := range aliases {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", alias, agent.Models[alias])
		}
	}

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

	var configDir string
	if local {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

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
	agents, err := loadAgentsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}

	agent, exists := agents[name]
	if !exists {
		return fmt.Errorf("agent %q not found in %s config", name, scopeString(local))
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

		agents[name] = agent

		if err := writeAgentsFile(agentPath, agents); err != nil {
			return fmt.Errorf("writing agents file: %w", err)
		}

		flags := getFlags(cmd)
		if !flags.Quiet {
			_, _ = fmt.Fprintf(stdout, "Updated agent %q\n", name)
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
	_, _ = fmt.Fprintf(stdout, "Editing agent %q %s%s%s\n\n", name, colorCyan.Sprint("("), colorDim.Sprint("press Enter to keep current value"), colorCyan.Sprint(")"))

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
	agents[name] = agent

	// Write updated file
	if err := writeAgentsFile(agentPath, agents); err != nil {
		return fmt.Errorf("writing agents file: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "\nUpdated agent %q\n", name)
	return nil
}

// addConfigAgentRemoveCommand adds the remove subcommand.
func addConfigAgentRemoveCommand(parent *cobra.Command) {
	removeCmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove an agent",
		Long: `Remove an agent configuration.

Removes the specified agent from the configuration file.`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigAgentRemove,
	}

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	parent.AddCommand(removeCmd)
}

// runConfigAgentRemove removes an agent configuration.
func runConfigAgentRemove(cmd *cobra.Command, args []string) error {
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

	// Load existing agents
	agents, err := loadAgentsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}

	if _, exists := agents[name]; !exists {
		return fmt.Errorf("agent %q not found in %s config", name, scopeString(local))
	}

	// Confirm removal unless --yes flag is set
	skipConfirm, _ := cmd.Flags().GetBool("yes")
	if !skipConfirm {
		isTTY := false
		if f, ok := stdin.(*os.File); ok {
			isTTY = term.IsTerminal(int(f.Fd()))
		}

		if isTTY {
			_, _ = fmt.Fprintf(stdout, "Remove agent %q from %s config? [y/N] ", name, scopeString(local))
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

	// Remove agent
	delete(agents, name)

	// Write updated file
	agentPath := filepath.Join(configDir, "agents.cue")
	if err := writeAgentsFile(agentPath, agents); err != nil {
		return fmt.Errorf("writing agents file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Removed agent %q\n", name)
	}

	return nil
}

// addConfigAgentDefaultCommand adds the default subcommand.
func addConfigAgentDefaultCommand(parent *cobra.Command) {
	defaultCmd := &cobra.Command{
		Use:   "default [name]",
		Short: "Set or show default agent",
		Long: `Set or show the default agent.

Without a name, shows the current default agent.
With a name, sets that agent as the default.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigAgentDefault,
	}

	parent.AddCommand(defaultCmd)
}

// runConfigAgentDefault sets or shows the default agent.
func runConfigAgentDefault(cmd *cobra.Command, args []string) error {
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
	agents, err := loadAgentsForScope(local)
	if err != nil {
		return err
	}
	if _, exists := agents[name]; !exists {
		return fmt.Errorf("agent %q not found", name)
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Update settings in settings.cue
	configPath := filepath.Join(configDir, "settings.cue")
	if err := writeDefaultAgentSetting(configPath, name); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintf(stdout, "Set default agent to %q\n", name)
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
func loadAgentsForScope(localOnly bool) (map[string]AgentConfig, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return nil, fmt.Errorf("resolving config paths: %w", err)
	}

	agents := make(map[string]AgentConfig)

	if localOnly {
		// Local only
		if paths.LocalExists {
			localAgents, err := loadAgentsFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, agent := range localAgents {
				agent.Source = "local"
				agents[name] = agent
			}
		}
	} else {
		// Merged: global first, then local overrides
		if paths.GlobalExists {
			globalAgents, err := loadAgentsFromDir(paths.Global)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, agent := range globalAgents {
				agent.Source = "global"
				agents[name] = agent
			}
		}
		if paths.LocalExists {
			localAgents, err := loadAgentsFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for name, agent := range localAgents {
				agent.Source = "local"
				agents[name] = agent
			}
		}
	}

	return agents, nil
}

// loadAgentsFromDir loads agents from a specific directory.
func loadAgentsFromDir(dir string) (map[string]AgentConfig, error) {
	agents := make(map[string]AgentConfig)

	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(dir)
	if err != nil {
		// If no CUE files exist, return empty map (not an error)
		if strings.Contains(err.Error(), "no CUE files") {
			return agents, nil
		}
		return agents, err
	}

	agentsVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyAgents))
	if !agentsVal.Exists() {
		return agents, nil
	}

	iter, err := agentsVal.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating agents: %w", err)
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

		// Load tags
		if tagsVal := val.LookupPath(cue.ParsePath("tags")); tagsVal.Exists() {
			tagIter, err := tagsVal.List()
			if err == nil {
				for tagIter.Next() {
					if s, err := tagIter.Value().String(); err == nil {
						agent.Tags = append(agent.Tags, s)
					}
				}
			}
		}

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
	}

	return agents, nil
}

// writeAgentsFile writes the agents configuration to a file.
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
		if len(agent.Tags) > 0 {
			sb.WriteString("\t\ttags: [")
			for i, tag := range agent.Tags {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%q", tag))
			}
			sb.WriteString("]\n")
		}
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

// writeDefaultAgentSetting writes or updates the default_agent setting.
func writeDefaultAgentSetting(path string, agentName string) error {
	// Read existing settings if file exists
	settings := make(map[string]string)
	if data, err := os.ReadFile(path); err == nil {
		// Parse existing settings (simple approach)
		// For now, we just overwrite the file with the setting
		_ = data // Would parse existing settings here
	}

	settings["default_agent"] = agentName

	var sb strings.Builder
	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your settings\n\n")
	sb.WriteString("settings: {\n")

	// Sort keys for consistent output
	var keys []string
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("\tdefault_agent: %q\n", settings[k]))
	}

	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// promptString prompts for a string value with a default.
func promptString(w io.Writer, r io.Reader, label, defaultVal string) (string, error) {
	if defaultVal != "" {
		_, _ = fmt.Fprintf(w, "%s %s%s%s: ", label, colorCyan.Sprint("["), colorDim.Sprint(defaultVal), colorCyan.Sprint("]"))
	} else {
		_, _ = fmt.Fprintf(w, "%s: ", label)
	}

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

// promptDefaultModel prompts for a default model selection.
// When models are defined, displays a numbered list for selection.
// Falls back to free-text input when no models are defined.
func promptDefaultModel(w io.Writer, r io.Reader, current string, models map[string]string) (string, error) {
	if len(models) == 0 {
		return promptString(w, r, "Default model", current)
	}

	// Sort aliases for stable ordering
	aliases := make([]string, 0, len(models))
	for alias := range models {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	_, _ = fmt.Fprintln(w, "Default model:")
	for i, alias := range aliases {
		if alias == current {
			_, _ = fmt.Fprintf(w, "  %d. %s - %s %s%s%s\n", i+1, alias, colorDim.Sprint(models[alias]), colorCyan.Sprint("("), colorSuccess.Sprint("current"), colorCyan.Sprint(")"))
		} else {
			_, _ = fmt.Fprintf(w, "  %d. %s - %s\n", i+1, alias, colorDim.Sprint(models[alias]))
		}
	}

	_, _ = fmt.Fprintln(w)
	if current != "" {
		_, _ = fmt.Fprintf(w, "Select model %s%s%s: ", colorCyan.Sprint("("), colorDim.Sprintf("number, alias, or Enter to keep %q", current), colorCyan.Sprint(")"))
	} else {
		_, _ = fmt.Fprintf(w, "Select model %s%s%s: ", colorCyan.Sprint("("), colorDim.Sprint("number or alias"), colorCyan.Sprint(")"))
	}

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return current, nil
	}

	// Try parsing as number
	if choice, err := strconv.Atoi(input); err == nil {
		if choice >= 1 && choice <= len(aliases) {
			return aliases[choice-1], nil
		}
		return "", fmt.Errorf("invalid selection: %s (choose 1-%d)", input, len(aliases))
	}

	// Try matching by alias
	for _, alias := range aliases {
		if strings.EqualFold(alias, input) {
			return alias, nil
		}
	}

	return "", fmt.Errorf("invalid selection: %q is not a known model alias", input)
}

// promptTags prompts for editing a slice of tags.
// Shows current tags and allows: comma-separated input to replace, empty to clear, Enter to keep.
func promptTags(w io.Writer, r io.Reader, current []string) ([]string, error) {
	if len(current) > 0 {
		_, _ = fmt.Fprintf(w, "Current tags: %s%s%s\n", colorCyan.Sprint("["), colorDim.Sprint(strings.Join(current, ", ")), colorCyan.Sprint("]"))
	} else {
		_, _ = fmt.Fprintf(w, "Current tags: %s%s%s\n", colorCyan.Sprint("("), colorDim.Sprint("none"), colorCyan.Sprint(")"))
	}
	_, _ = fmt.Fprintf(w, "Tags %s%s%s: ", colorCyan.Sprint("("), colorDim.Sprint("comma-separated, - to clear, Enter to keep"), colorCyan.Sprint(")"))

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Enter keeps current
	if input == "" {
		return current, nil
	}

	// "-" clears tags
	if input == "-" {
		return nil, nil
	}

	// Parse comma-separated tags
	var tags []string
	for _, t := range strings.Split(input, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	return tags, nil
}

// promptModels prompts for editing a map of model aliases.
// Offers options: (k)eep, (c)lear, (e)dit.
func promptModels(w io.Writer, r io.Reader, current map[string]string) (map[string]string, error) {
	reader := bufio.NewReader(r)

	if len(current) > 0 {
		_, _ = fmt.Fprintln(w, "Current models:")
		var aliases []string
		for alias := range current {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		for _, alias := range aliases {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", alias, colorDim.Sprint(current[alias]))
		}
	} else {
		_, _ = fmt.Fprintf(w, "Current models: %s%s%s\n", colorCyan.Sprint("("), colorDim.Sprint("none"), colorCyan.Sprint(")"))
	}

	_, _ = fmt.Fprintf(w, "Models: %sk%seep, %sc%slear, %se%sdit %s%s%s: ",
		colorCyan.Sprint("("), colorCyan.Sprint(")"),
		colorCyan.Sprint("("), colorCyan.Sprint(")"),
		colorCyan.Sprint("("), colorCyan.Sprint(")"),
		colorCyan.Sprint("["), colorDim.Sprint("k"), colorCyan.Sprint("]"))
	choice, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	choice = strings.TrimSpace(strings.ToLower(choice))

	switch choice {
	case "", "k", "keep":
		return current, nil
	case "c", "clear":
		return nil, nil
	case "e", "edit":
		return promptModelsEdit(w, reader, current)
	default:
		return nil, fmt.Errorf("invalid choice: %s", choice)
	}
}

// promptModelsEdit handles the edit mode for models.
func promptModelsEdit(w io.Writer, reader *bufio.Reader, current map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	// Edit existing models
	if len(current) > 0 {
		_, _ = fmt.Fprintln(w, "Edit existing models (Enter to keep, - to delete):")
		var aliases []string
		for alias := range current {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)

		for _, alias := range aliases {
			currentVal := current[alias]
			_, _ = fmt.Fprintf(w, "  %s %s%s%s: ", alias, colorCyan.Sprint("["), colorDim.Sprint(currentVal), colorCyan.Sprint("]"))

			input, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("reading input: %w", err)
			}
			input = strings.TrimSpace(input)

			if input == "-" {
				// Delete this model
				continue
			}
			if input == "" {
				// Keep current value
				result[alias] = currentVal
			} else {
				// Update value
				result[alias] = input
			}
		}
	}

	// Add new models
	_, _ = fmt.Fprintln(w, "Add new models (alias=model-id, empty to finish):")
	for {
		_, _ = fmt.Fprint(w, "  > ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(input)

		if input == "" {
			break
		}

		parts := strings.SplitN(input, "=", 2)
		if len(parts) != 2 {
			_, _ = fmt.Fprintln(w, "  Invalid format. Use: alias=model-id")
			continue
		}

		alias := strings.TrimSpace(parts[0])
		modelID := strings.TrimSpace(parts[1])
		if alias == "" || modelID == "" {
			_, _ = fmt.Fprintln(w, "  Invalid format. Use: alias=model-id")
			continue
		}

		result[alias] = modelID
	}

	return result, nil
}

// openInEditor opens a file in the user's editor.
func openInEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// scopeString returns "local" or "global" based on the flag.
func scopeString(local bool) string {
	if local {
		return "local"
	}
	return "global"
}
