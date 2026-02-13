package cli

import (
	"fmt"
	"io"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/spf13/cobra"
)

// ShowResult holds the result of preparing show output.
type ShowResult struct {
	ItemType   string   // "Agent", "Role", "Context", "Task"
	Name       string   // Item name (when showing specific item)
	Content    string   // Formatted content
	AllNames   []string // All available items of this type
	ShowReason string   // Why this item is shown (e.g., "first in config", "default")
}

// addShowCommand adds the show command and its subcommands to the parent command.
func addShowCommand(parent *cobra.Command) {
	showCmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{"view"},
		GroupID: "commands",
		Short:   "Display resolved configuration content",
		Long:  `Display resolved configuration content after UTD processing and config merging.`,
		RunE:  runShow,
	}

	showRoleCmd := &cobra.Command{
		Use:     "role [name]",
		Aliases: []string{"roles"},
		Short:   "Display resolved role content",
		Long:    `Display resolved role content after UTD processing.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowRole,
	}

	showContextCmd := &cobra.Command{
		Use:     "context [name]",
		Aliases: []string{"contexts"},
		Short:   "Display resolved context content",
		Long:    `Display resolved context content after UTD processing.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowContext,
	}

	showAgentCmd := &cobra.Command{
		Use:     "agent [name]",
		Aliases: []string{"agents"},
		Short:   "Display agent configuration",
		Long:    `Display effective agent configuration after config merging.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowAgent,
	}

	showTaskCmd := &cobra.Command{
		Use:     "task [name]",
		Aliases: []string{"tasks"},
		Short:   "Display task template",
		Long:    `Display resolved task prompt template.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowTask,
	}

	// Add --scope flag to show command
	showCmd.PersistentFlags().String("scope", "", "Show from specific scope: global or local")

	// Add subcommands
	showCmd.AddCommand(showRoleCmd)
	showCmd.AddCommand(showContextCmd)
	showCmd.AddCommand(showAgentCmd)
	showCmd.AddCommand(showTaskCmd)

	// Add show to parent
	parent.AddCommand(showCmd)
}

// runShow displays all configuration (agents, roles, contexts, tasks).
func runShow(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	if len(args) > 0 {
		return unknownCommandError("start show", args[0])
	}

	w := cmd.OutOrStdout()
	scope, _ := cmd.Flags().GetString("scope")

	type section struct {
		category string
		names    []string
	}

	var sections []section

	if result, err := prepareShowAgent("", scope); err == nil {
		sections = append(sections, section{"agents", result.AllNames})
	}
	if result, err := prepareShowRole("", scope); err == nil {
		sections = append(sections, section{"roles", result.AllNames})
	}
	if result, err := prepareShowContext("", scope); err == nil {
		sections = append(sections, section{"contexts", result.AllNames})
	}
	if result, err := prepareShowTask("", scope); err == nil {
		sections = append(sections, section{"tasks", result.AllNames})
	}

	for _, s := range sections {
		_, _ = categoryColor(s.category).Fprint(w, s.category)
		_, _ = fmt.Fprintln(w, "/")
		for _, name := range s.names {
			_, _ = fmt.Fprintf(w, "  %s\n", name)
		}
		_, _ = fmt.Fprintln(w)
	}

	return nil
}

// runShowRole displays resolved role content.
func runShowRole(cmd *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	scope, _ := cmd.Flags().GetString("scope")
	result, err := prepareShowRole(name, scope)
	if err != nil {
		return err
	}

	printPreview(cmd.OutOrStdout(), result)
	return nil
}

// runShowContext displays resolved context content.
func runShowContext(cmd *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	scope, _ := cmd.Flags().GetString("scope")
	result, err := prepareShowContext(name, scope)
	if err != nil {
		return err
	}

	printPreview(cmd.OutOrStdout(), result)
	return nil
}

// runShowAgent displays agent configuration.
func runShowAgent(cmd *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	scope, _ := cmd.Flags().GetString("scope")
	result, err := prepareShowAgent(name, scope)
	if err != nil {
		return err
	}

	printPreview(cmd.OutOrStdout(), result)
	return nil
}

// runShowTask displays task template.
func runShowTask(cmd *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	scope, _ := cmd.Flags().GetString("scope")
	result, err := prepareShowTask(name, scope)
	if err != nil {
		return err
	}

	printPreview(cmd.OutOrStdout(), result)
	return nil
}

// prepareShowRole prepares show output for a role.
func prepareShowRole(name, scope string) (ShowResult, error) {
	cfg, err := loadConfig(scope)
	if err != nil {
		return ShowResult{}, err
	}

	roles := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyRoles))
	if !roles.Exists() {
		return ShowResult{}, fmt.Errorf("no roles defined in configuration")
	}

	// Collect all role names in config order
	var allNames []string
	iter, err := roles.Fields()
	if err != nil {
		return ShowResult{}, fmt.Errorf("reading roles: %w", err)
	}
	for iter.Next() {
		allNames = append(allNames, iter.Selector().Unquoted())
	}
	if len(allNames) == 0 {
		return ShowResult{}, fmt.Errorf("no roles defined in configuration")
	}

	// Determine which role to show and why
	showReason := ""
	if name == "" {
		name = allNames[0]
		showReason = "first in config"
	}

	resolvedName := name
	role := roles.LookupPath(cue.MakePath(cue.Str(name)))
	if !role.Exists() {
		// Try substring match
		var matches []string
		for _, roleName := range allNames {
			if strings.Contains(roleName, name) {
				matches = append(matches, roleName)
			}
		}

		switch len(matches) {
		case 0:
			return ShowResult{}, fmt.Errorf("role %q not found", name)
		case 1:
			resolvedName = matches[0]
			role = roles.LookupPath(cue.MakePath(cue.Str(resolvedName)))
		default:
			return ShowResult{}, fmt.Errorf("ambiguous role name %q matches: %s", name, strings.Join(matches, ", "))
		}
	}

	content := formatShowContent(role, "role")

	return ShowResult{
		ItemType:   "Role",
		Name:       resolvedName,
		Content:    content,
		AllNames:   allNames,
		ShowReason: showReason,
	}, nil
}

// prepareShowContext prepares show output for context(s).
func prepareShowContext(name, scope string) (ShowResult, error) {
	cfg, err := loadConfig(scope)
	if err != nil {
		return ShowResult{}, err
	}

	contexts := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyContexts))
	if !contexts.Exists() {
		return ShowResult{}, fmt.Errorf("no contexts defined in configuration")
	}

	// Collect all context names in config order
	var allNames []string
	showReason := ""

	iter, err := contexts.Fields()
	if err != nil {
		return ShowResult{}, fmt.Errorf("reading contexts: %w", err)
	}
	for iter.Next() {
		allNames = append(allNames, iter.Selector().Unquoted())
	}

	if len(allNames) == 0 {
		return ShowResult{}, fmt.Errorf("no contexts defined in configuration")
	}

	// If no name specified, show first context
	if name == "" {
		name = allNames[0]
		showReason = "first in config"
	}

	// Show single context
	resolvedName := name
	ctx := contexts.LookupPath(cue.MakePath(cue.Str(name)))
	if !ctx.Exists() {
		// Try substring match
		var matches []string
		for _, ctxName := range allNames {
			if strings.Contains(ctxName, name) {
				matches = append(matches, ctxName)
			}
		}

		switch len(matches) {
		case 0:
			return ShowResult{}, fmt.Errorf("context %q not found", name)
		case 1:
			resolvedName = matches[0]
			ctx = contexts.LookupPath(cue.MakePath(cue.Str(resolvedName)))
		default:
			return ShowResult{}, fmt.Errorf("ambiguous context name %q matches: %s", name, strings.Join(matches, ", "))
		}
	}

	content := formatShowContent(ctx, "context")

	return ShowResult{
		ItemType:   "Context",
		Name:       resolvedName,
		Content:    content,
		AllNames:   allNames,
		ShowReason: showReason,
	}, nil
}

// prepareShowAgent prepares show output for an agent.
func prepareShowAgent(name, scope string) (ShowResult, error) {
	cfg, err := loadConfig(scope)
	if err != nil {
		return ShowResult{}, err
	}

	agents := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyAgents))
	if !agents.Exists() {
		return ShowResult{}, fmt.Errorf("no agents defined in configuration")
	}

	// Collect all agent names in config order
	var allNames []string
	iter, err := agents.Fields()
	if err != nil {
		return ShowResult{}, fmt.Errorf("reading agents: %w", err)
	}
	for iter.Next() {
		allNames = append(allNames, iter.Selector().Unquoted())
	}
	if len(allNames) == 0 {
		return ShowResult{}, fmt.Errorf("no agents defined in configuration")
	}

	// Determine which agent to show and why
	showReason := ""
	if name == "" {
		name = allNames[0]
		showReason = "first in config"
	}

	resolvedName := name
	agent := agents.LookupPath(cue.MakePath(cue.Str(name)))
	if !agent.Exists() {
		// Try substring match
		var matches []string
		for _, agentName := range allNames {
			if strings.Contains(agentName, name) {
				matches = append(matches, agentName)
			}
		}

		switch len(matches) {
		case 0:
			return ShowResult{}, fmt.Errorf("agent %q not found", name)
		case 1:
			resolvedName = matches[0]
			agent = agents.LookupPath(cue.MakePath(cue.Str(resolvedName)))
		default:
			return ShowResult{}, fmt.Errorf("ambiguous agent name %q matches: %s", name, strings.Join(matches, ", "))
		}
	}

	content := formatShowContent(agent, "agent")

	return ShowResult{
		ItemType:   "Agent",
		Name:       resolvedName,
		Content:    content,
		AllNames:   allNames,
		ShowReason: showReason,
	}, nil
}

// prepareShowTask prepares show output for a task.
func prepareShowTask(name, scope string) (ShowResult, error) {
	cfg, err := loadConfig(scope)
	if err != nil {
		return ShowResult{}, err
	}

	tasks := cfg.Value.LookupPath(cue.ParsePath(internalcue.KeyTasks))
	if !tasks.Exists() {
		return ShowResult{}, fmt.Errorf("no tasks defined in configuration")
	}

	// Collect all task names in config order
	var allNames []string
	iter, err := tasks.Fields()
	if err != nil {
		return ShowResult{}, fmt.Errorf("reading tasks: %w", err)
	}
	for iter.Next() {
		allNames = append(allNames, iter.Selector().Unquoted())
	}
	if len(allNames) == 0 {
		return ShowResult{}, fmt.Errorf("no tasks defined in configuration")
	}

	// Determine which task to show and why
	showReason := ""
	if name == "" {
		name = allNames[0]
		showReason = "first in config"
	}

	// Try exact match first
	resolvedName := name
	task := tasks.LookupPath(cue.MakePath(cue.Str(name)))
	if !task.Exists() {
		// Try substring match (per DR-015)
		var matches []string
		for _, taskName := range allNames {
			if strings.Contains(taskName, name) {
				matches = append(matches, taskName)
			}
		}

		switch len(matches) {
		case 0:
			return ShowResult{}, fmt.Errorf("task %q not found", name)
		case 1:
			resolvedName = matches[0]
			task = tasks.LookupPath(cue.MakePath(cue.Str(resolvedName)))
		default:
			return ShowResult{}, fmt.Errorf("ambiguous task name %q matches: %s", name, strings.Join(matches, ", "))
		}
	}

	content := formatShowContent(task, "task")

	return ShowResult{
		ItemType:   "Task",
		Name:       resolvedName,
		Content:    content,
		AllNames:   allNames,
		ShowReason: showReason,
	}, nil
}

// loadConfig loads CUE configuration based on scope.
func loadConfig(scope string) (internalcue.LoadResult, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return internalcue.LoadResult{}, fmt.Errorf("resolving config paths: %w", err)
	}

	s := config.ParseScope(scope)
	dirs := paths.ForScope(s)

	if len(dirs) == 0 {
		switch s {
		case config.ScopeGlobal:
			return internalcue.LoadResult{}, fmt.Errorf("no global configuration found at %s", paths.Global)
		case config.ScopeLocal:
			return internalcue.LoadResult{}, fmt.Errorf("no local configuration found at %s", paths.Local)
		default:
			return internalcue.LoadResult{}, fmt.Errorf("no configuration found (checked %s and %s)", paths.Global, paths.Local)
		}
	}

	loader := internalcue.NewLoader()
	return loader.Load(dirs)
}

// formatShowContent formats a CUE value for display.
func formatShowContent(v cue.Value, itemType string) string {
	var sb strings.Builder

	label := colorDim.Sprint

	switch itemType {
	case "agent":
		if desc := v.LookupPath(cue.ParsePath("description")); desc.Exists() {
			if s, err := desc.String(); err == nil {
				sb.WriteString(fmt.Sprintf("%s %s\n\n", label("Description:"), s))
			}
		}
		if bin := v.LookupPath(cue.ParsePath("bin")); bin.Exists() {
			if s, err := bin.String(); err == nil {
				sb.WriteString(fmt.Sprintf("%s %s\n", label("Binary:"), s))
			}
		}
		if cmd := v.LookupPath(cue.ParsePath("command")); cmd.Exists() {
			if s, err := cmd.String(); err == nil {
				sb.WriteString(fmt.Sprintf("%s %s\n", label("Command:"), s))
			}
		}
		if dm := v.LookupPath(cue.ParsePath("default_model")); dm.Exists() {
			if s, err := dm.String(); err == nil {
				sb.WriteString(fmt.Sprintf("%s %s\n", label("Default Model:"), s))
			}
		}
		if models := v.LookupPath(cue.ParsePath("models")); models.Exists() {
			sb.WriteString(fmt.Sprintf("\n%s\n", label("Models:")))
			iter, err := models.Fields()
			if err == nil {
				for iter.Next() {
					modelName := iter.Selector().Unquoted()
					if s, err := iter.Value().String(); err == nil {
						sb.WriteString(fmt.Sprintf("  %s %s\n", label(modelName+":"), s))
					}
				}
			}
		}
		if tags := v.LookupPath(cue.ParsePath("tags")); tags.Exists() {
			iter, err := tags.List()
			if err == nil {
				var tagList []string
				for iter.Next() {
					if s, err := iter.Value().String(); err == nil {
						tagList = append(tagList, s)
					}
				}
				if len(tagList) > 0 {
					sb.WriteString(fmt.Sprintf("\n%s %s\n", label("Tags:"), strings.Join(tagList, ", ")))
				}
			}
		}

	case "role", "context", "task":
		if desc := v.LookupPath(cue.ParsePath("description")); desc.Exists() {
			if s, err := desc.String(); err == nil {
				sb.WriteString(fmt.Sprintf("%s %s\n\n", label("Description:"), s))
			}
		}

		if file := v.LookupPath(cue.ParsePath("file")); file.Exists() {
			if s, err := file.String(); err == nil {
				sb.WriteString(fmt.Sprintf("%s %s\n", label("File:"), s))
			}
		}
		if cmd := v.LookupPath(cue.ParsePath("command")); cmd.Exists() {
			if s, err := cmd.String(); err == nil {
				sb.WriteString(fmt.Sprintf("%s %s\n", label("Command:"), s))
			}
		}
		if prompt := v.LookupPath(cue.ParsePath("prompt")); prompt.Exists() {
			if s, err := prompt.String(); err == nil {
				sb.WriteString(fmt.Sprintf("\n%s %s\n", label("Prompt:"), s))
			}
		}

		if itemType == "context" {
			if req := v.LookupPath(cue.ParsePath("required")); req.Exists() {
				if b, err := req.Bool(); err == nil {
					sb.WriteString(fmt.Sprintf("%s %t\n", label("Required:"), b))
				}
			}
			if def := v.LookupPath(cue.ParsePath("default")); def.Exists() {
				if b, err := def.Bool(); err == nil {
					sb.WriteString(fmt.Sprintf("%s %t\n", label("Default:"), b))
				}
			}
			if tags := v.LookupPath(cue.ParsePath("tags")); tags.Exists() {
				iter, err := tags.List()
				if err == nil {
					var tagList []string
					for iter.Next() {
						if s, err := iter.Value().String(); err == nil {
							tagList = append(tagList, s)
						}
					}
					if len(tagList) > 0 {
						sb.WriteString(fmt.Sprintf("%s %s\n", label("Tags:"), strings.Join(tagList, ", ")))
					}
				}
			}
		}

		if itemType == "task" {
			if role := v.LookupPath(cue.ParsePath("role")); role.Exists() {
				if s, err := role.String(); err == nil {
					sb.WriteString(fmt.Sprintf("%s %s\n", label("Role:"), s))
				}
			}
		}
	}

	return sb.String()
}

// itemTypeToCategory maps ShowResult.ItemType to the category string for categoryColor().
func itemTypeToCategory(itemType string) string {
	switch itemType {
	case "Agent":
		return "agents"
	case "Role":
		return "roles"
	case "Context":
		return "contexts"
	case "Task":
		return "tasks"
	default:
		return ""
	}
}

// printPreview writes the ShowResult to the given writer.
func printPreview(w io.Writer, r ShowResult) {
	cat := itemTypeToCategory(r.ItemType)
	_, _ = fmt.Fprintln(w)

	// Show list of all items if available (for agent/role)
	if len(r.AllNames) > 0 {
		_, _ = categoryColor(cat).Fprint(w, r.ItemType+"s")
		_, _ = fmt.Fprintf(w, ": %s\n", strings.Join(r.AllNames, ", "))
		_, _ = fmt.Fprintln(w)
	}

	// Show which item and why
	_, _ = categoryColor(cat).Fprint(w, r.ItemType)
	_, _ = fmt.Fprintf(w, ": %s", r.Name)
	if r.ShowReason != "" {
		_, _ = fmt.Fprint(w, " ")
		_, _ = colorCyan.Fprint(w, "(")
		_, _ = colorDim.Fprint(w, r.ShowReason)
		_, _ = colorCyan.Fprint(w, ")")
	}
	_, _ = fmt.Fprintln(w)
	PrintSeparator(w)

	// Show full content
	_, _ = fmt.Fprint(w, r.Content)
	if !strings.HasSuffix(r.Content, "\n") {
		_, _ = fmt.Fprintln(w)
	}
	PrintSeparator(w)
}

