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
		Long:    `Display resolved configuration content after UTD processing and config merging.`,
		RunE:    runShow,
	}

	showRoleCmd := &cobra.Command{
		Use:     "role [name]",
		Aliases: []string{"roles"},
		Short:   "Display resolved role content",
		Long:    `Display resolved role content after UTD processing.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowItem(internalcue.KeyRoles, "Role"),
	}

	showContextCmd := &cobra.Command{
		Use:     "context [name]",
		Aliases: []string{"contexts"},
		Short:   "Display resolved context content",
		Long:    `Display resolved context content after UTD processing.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowItem(internalcue.KeyContexts, "Context"),
	}

	showAgentCmd := &cobra.Command{
		Use:     "agent [name]",
		Aliases: []string{"agents"},
		Short:   "Display agent configuration",
		Long:    `Display effective agent configuration after config merging.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowItem(internalcue.KeyAgents, "Agent"),
	}

	showTaskCmd := &cobra.Command{
		Use:     "task [name]",
		Aliases: []string{"tasks"},
		Short:   "Display task template",
		Long:    `Display resolved task prompt template.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowItem(internalcue.KeyTasks, "Task"),
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

	if result, err := prepareShow("", scope, internalcue.KeyAgents, "Agent"); err == nil {
		sections = append(sections, section{"agents", result.AllNames})
	}
	if result, err := prepareShow("", scope, internalcue.KeyRoles, "Role"); err == nil {
		sections = append(sections, section{"roles", result.AllNames})
	}
	if result, err := prepareShow("", scope, internalcue.KeyContexts, "Context"); err == nil {
		sections = append(sections, section{"contexts", result.AllNames})
	}
	if result, err := prepareShow("", scope, internalcue.KeyTasks, "Task"); err == nil {
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

// runShowItem returns a cobra RunE handler that displays a specific item type.
func runShowItem(cueKey, itemType string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}

		scope, _ := cmd.Flags().GetString("scope")
		result, err := prepareShow(name, scope, cueKey, itemType)
		if err != nil {
			return err
		}

		printPreview(cmd.OutOrStdout(), result)
		return nil
	}
}

// prepareShow prepares show output for an item type.
// cueKey is the top-level CUE key (e.g., internalcue.KeyRoles).
// itemType is the display name (e.g., "Role").
func prepareShow(name, scope, cueKey, itemType string) (ShowResult, error) {
	cfg, err := loadConfig(scope)
	if err != nil {
		return ShowResult{}, err
	}

	typePlural := strings.ToLower(itemType) + "s"

	items := cfg.Value.LookupPath(cue.ParsePath(cueKey))
	if !items.Exists() {
		return ShowResult{}, fmt.Errorf("no %s defined in configuration", typePlural)
	}

	// Collect all names in config order
	var allNames []string
	iter, err := items.Fields()
	if err != nil {
		return ShowResult{}, fmt.Errorf("reading %s: %w", typePlural, err)
	}
	for iter.Next() {
		allNames = append(allNames, iter.Selector().Unquoted())
	}
	if len(allNames) == 0 {
		return ShowResult{}, fmt.Errorf("no %s defined in configuration", typePlural)
	}

	// Determine which item to show and why
	showReason := ""
	if name == "" {
		name = allNames[0]
		showReason = "first in config"
	}

	resolvedName := name
	item := items.LookupPath(cue.MakePath(cue.Str(name)))
	if !item.Exists() {
		// Try substring match
		var matches []string
		for _, n := range allNames {
			if strings.Contains(n, name) {
				matches = append(matches, n)
			}
		}

		switch len(matches) {
		case 0:
			return ShowResult{}, fmt.Errorf("%s %q not found", strings.ToLower(itemType), name)
		case 1:
			resolvedName = matches[0]
			item = items.LookupPath(cue.MakePath(cue.Str(resolvedName)))
		default:
			return ShowResult{}, fmt.Errorf("ambiguous %s name %q matches: %s", strings.ToLower(itemType), name, strings.Join(matches, ", "))
		}
	}

	content := formatShowContent(item, strings.ToLower(itemType))

	return ShowResult{
		ItemType:   itemType,
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

	// Show list of all items if available
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
	printSeparator(w)

	// Show full content
	_, _ = fmt.Fprint(w, r.Content)
	if !strings.HasSuffix(r.Content, "\n") {
		_, _ = fmt.Fprintln(w)
	}
	printSeparator(w)
}
