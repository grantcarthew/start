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
	ListOnly   bool     // True when showing list without content
	// Context-specific fields
	DefaultContexts  []string // Contexts with default: true
	RequiredContexts []string // Contexts with required: true
	AllTags          []string // All unique tags across items
}

// addShowCommand adds the show command and its subcommands to the parent command.
func addShowCommand(parent *cobra.Command) {
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Display resolved configuration content",
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

	// Show agents
	if result, err := prepareShowAgent("", scope); err == nil {
		fmt.Fprintln(w, "Agents:")
		for _, name := range result.AllNames {
			fmt.Fprintf(w, "  %s\n", name)
		}
		fmt.Fprintln(w)
	}

	// Show roles
	if result, err := prepareShowRole("", scope); err == nil {
		fmt.Fprintln(w, "Roles:")
		for _, name := range result.AllNames {
			fmt.Fprintf(w, "  %s\n", name)
		}
		fmt.Fprintln(w)
	}

	// Show contexts
	if result, err := prepareShowContext("", scope); err == nil {
		fmt.Fprintln(w, "Contexts:")
		for _, name := range result.AllNames {
			fmt.Fprintf(w, "  %s\n", name)
		}
		if len(result.RequiredContexts) > 0 {
			fmt.Fprintf(w, "  Required: %v\n", result.RequiredContexts)
		}
		if len(result.DefaultContexts) > 0 {
			fmt.Fprintf(w, "  Default: %v\n", result.DefaultContexts)
		}
		fmt.Fprintln(w)
	}

	// Show tasks
	if result, err := prepareShowTask("", scope); err == nil {
		fmt.Fprintln(w, "Tasks:")
		for _, name := range result.AllNames {
			fmt.Fprintf(w, "  %s\n", name)
		}
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

	role := roles.LookupPath(cue.MakePath(cue.Str(name)))
	if !role.Exists() {
		return ShowResult{}, fmt.Errorf("role %q not found", name)
	}

	content := formatShowContent(role, "role")

	return ShowResult{
		ItemType:   "Role",
		Name:       name,
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

	// Collect all context info
	var allNames []string
	var defaultContexts []string
	var requiredContexts []string
	tagSet := make(map[string]bool)

	iter, err := contexts.Fields()
	if err != nil {
		return ShowResult{}, fmt.Errorf("reading contexts: %w", err)
	}
	for iter.Next() {
		ctxName := iter.Selector().Unquoted()
		ctx := iter.Value()
		allNames = append(allNames, ctxName)

		// Check default flag
		if def := ctx.LookupPath(cue.ParsePath("default")); def.Exists() {
			if b, err := def.Bool(); err == nil && b {
				defaultContexts = append(defaultContexts, ctxName)
			}
		}

		// Check required flag
		if req := ctx.LookupPath(cue.ParsePath("required")); req.Exists() {
			if b, err := req.Bool(); err == nil && b {
				requiredContexts = append(requiredContexts, ctxName)
			}
		}

		// Collect tags
		if tags := ctx.LookupPath(cue.ParsePath("tags")); tags.Exists() {
			tagIter, err := tags.List()
			if err == nil {
				for tagIter.Next() {
					if s, err := tagIter.Value().String(); err == nil {
						tagSet[s] = true
					}
				}
			}
		}
	}

	if len(allNames) == 0 {
		return ShowResult{}, fmt.Errorf("no contexts defined in configuration")
	}

	// Convert tag set to sorted slice
	var allTags []string
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}

	// If no name specified, return list only
	if name == "" {
		return ShowResult{
			ItemType:         "Context",
			AllNames:         allNames,
			DefaultContexts:  defaultContexts,
			RequiredContexts: requiredContexts,
			AllTags:          allTags,
			ListOnly:         true,
		}, nil
	}

	// Show single context
	ctx := contexts.LookupPath(cue.MakePath(cue.Str(name)))
	if !ctx.Exists() {
		return ShowResult{}, fmt.Errorf("context %q not found", name)
	}

	content := formatShowContent(ctx, "context")

	return ShowResult{
		ItemType: "Context",
		Name:     name,
		Content:  content,
		AllNames: allNames,
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

	agent := agents.LookupPath(cue.MakePath(cue.Str(name)))
	if !agent.Exists() {
		return ShowResult{}, fmt.Errorf("agent %q not found", name)
	}

	content := formatShowContent(agent, "agent")

	return ShowResult{
		ItemType:   "Agent",
		Name:       name,
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

	// If no name specified, return list only
	if name == "" {
		return ShowResult{
			ItemType: "Task",
			AllNames: allNames,
			ListOnly: true,
		}, nil
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
		ItemType: "Task",
		Name:     resolvedName,
		Content:  content,
		AllNames: allNames,
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

	switch itemType {
	case "agent":
		if desc := v.LookupPath(cue.ParsePath("description")); desc.Exists() {
			if s, err := desc.String(); err == nil {
				sb.WriteString(fmt.Sprintf("Description: %s\n\n", s))
			}
		}
		if bin := v.LookupPath(cue.ParsePath("bin")); bin.Exists() {
			if s, err := bin.String(); err == nil {
				sb.WriteString(fmt.Sprintf("Binary: %s\n", s))
			}
		}
		if cmd := v.LookupPath(cue.ParsePath("command")); cmd.Exists() {
			if s, err := cmd.String(); err == nil {
				sb.WriteString(fmt.Sprintf("Command: %s\n", s))
			}
		}
		if models := v.LookupPath(cue.ParsePath("models")); models.Exists() {
			sb.WriteString("\nModels:\n")
			iter, err := models.Fields()
			if err == nil {
				for iter.Next() {
					modelName := iter.Selector().Unquoted()
					if s, err := iter.Value().String(); err == nil {
						sb.WriteString(fmt.Sprintf("  %s: %s\n", modelName, s))
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
					sb.WriteString(fmt.Sprintf("\nTags: %s\n", strings.Join(tagList, ", ")))
				}
			}
		}

	case "role", "context", "task":
		if desc := v.LookupPath(cue.ParsePath("description")); desc.Exists() {
			if s, err := desc.String(); err == nil {
				sb.WriteString(fmt.Sprintf("Description: %s\n\n", s))
			}
		}

		if file := v.LookupPath(cue.ParsePath("file")); file.Exists() {
			if s, err := file.String(); err == nil {
				sb.WriteString(fmt.Sprintf("File: %s\n", s))
			}
		}
		if cmd := v.LookupPath(cue.ParsePath("command")); cmd.Exists() {
			if s, err := cmd.String(); err == nil {
				sb.WriteString(fmt.Sprintf("Command: %s\n", s))
			}
		}
		if prompt := v.LookupPath(cue.ParsePath("prompt")); prompt.Exists() {
			if s, err := prompt.String(); err == nil {
				sb.WriteString(fmt.Sprintf("\n%s\n", s))
			}
		}

		if itemType == "context" {
			if req := v.LookupPath(cue.ParsePath("required")); req.Exists() {
				if b, err := req.Bool(); err == nil {
					sb.WriteString(fmt.Sprintf("Required: %t\n", b))
				}
			}
			if def := v.LookupPath(cue.ParsePath("default")); def.Exists() {
				if b, err := def.Bool(); err == nil {
					sb.WriteString(fmt.Sprintf("Default: %t\n", b))
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
						sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(tagList, ", ")))
					}
				}
			}
		}

		if itemType == "task" {
			if role := v.LookupPath(cue.ParsePath("role")); role.Exists() {
				if s, err := role.String(); err == nil {
					sb.WriteString(fmt.Sprintf("Role: %s\n", s))
				}
			}
		}
	}

	return sb.String()
}

// printPreview writes the ShowResult to the given writer.
func printPreview(w io.Writer, r ShowResult) {
	// Handle list-only output
	if r.ListOnly {
		printListOnly(w, r)
		return
	}

	// Show list of all items if available (for agent/role)
	if len(r.AllNames) > 0 {
		fmt.Fprintf(w, "%ss: %s\n", r.ItemType, strings.Join(r.AllNames, ", "))
		fmt.Fprintln(w)
	}

	// Show which item and why
	if r.ShowReason != "" {
		fmt.Fprintf(w, "Showing: %s (%s)\n", r.Name, r.ShowReason)
	} else {
		fmt.Fprintf(w, "%s: %s\n", r.ItemType, r.Name)
	}
	fmt.Fprintln(w, strings.Repeat("─", 79))

	// Show full content
	fmt.Fprint(w, r.Content)
	if !strings.HasSuffix(r.Content, "\n") {
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, strings.Repeat("─", 79))
}

// printListOnly prints a list-only result without content preview.
func printListOnly(w io.Writer, r ShowResult) {
	// Pluralize item type for header
	plural := r.ItemType + "s"

	fmt.Fprintf(w, "%s: %s\n", plural, strings.Join(r.AllNames, ", "))

	// Context-specific fields
	if r.ItemType == "Context" {
		if len(r.DefaultContexts) > 0 {
			fmt.Fprintf(w, "\nDefault: %s\n", strings.Join(r.DefaultContexts, ", "))
		}
		if len(r.RequiredContexts) > 0 {
			fmt.Fprintf(w, "Required: %s\n", strings.Join(r.RequiredContexts, ", "))
		}
		if len(r.AllTags) > 0 {
			fmt.Fprintf(w, "\nTags: %s\n", strings.Join(r.AllTags, ", "))
		}
	}
}

