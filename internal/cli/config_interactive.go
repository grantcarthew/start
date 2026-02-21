package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// allConfigCategories is the ordered set of interactive config categories.
// Each name must be the plural of its corresponding "config <singular>" subcommand
// (i.e. strip a trailing "s" to get the subcommand name: "agents" → "agent").
var allConfigCategories = []string{"agents", "roles", "contexts", "tasks"}

// addConfigInteractiveAddCommand registers the top-level "config add" command.
func addConfigInteractiveAddCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a config item interactively",
		Long: `Interactively add a new agent, role, context, or task.

Prompts for the category then runs the interactive add flow for that type.`,
		Args: noArgsOrHelp,
		RunE: runConfigInteractiveAdd,
	}
	parent.AddCommand(cmd)
}

// runConfigInteractiveAdd picks a category then delegates to the category add command.
func runConfigInteractiveAdd(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	if !isTerminal(stdin) {
		return fmt.Errorf("interactive add requires a terminal")
	}

	_, _ = fmt.Fprintln(stdout, "Add:")
	category, err := promptSelectCategory(stdout, stdin, allConfigCategories)
	if err != nil || category == "" {
		return err
	}

	singular := strings.TrimSuffix(category, "s")
	subCmd, _, err := cmd.Root().Find([]string{"config", singular, "add"})
	if err != nil || !strings.HasPrefix(subCmd.Use, "add") {
		return fmt.Errorf("config %s add command not found", singular)
	}
	if subCmd.RunE == nil {
		return fmt.Errorf("config %s add command not executable", singular)
	}
	subCmd.SetContext(cmd.Context())
	return subCmd.RunE(subCmd, nil)
}

// addConfigInteractiveEditCommand registers the top-level "config edit" command.
func addConfigInteractiveEditCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a config item interactively",
		Long: `Interactively select and edit an agent, role, context, or task.

Prompts for the category, shows a numbered list, then runs the interactive
edit flow for the selected item.`,
		Args: noArgsOrHelp,
		RunE: runConfigInteractiveEdit,
	}
	parent.AddCommand(cmd)
}

// runConfigInteractiveEdit picks a category and item then delegates to the category edit command.
func runConfigInteractiveEdit(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	if !isTerminal(stdin) {
		return fmt.Errorf("interactive edit requires a terminal")
	}

	_, _ = fmt.Fprintln(stdout, "Edit:")
	category, err := promptSelectCategory(stdout, stdin, allConfigCategories)
	if err != nil || category == "" {
		return err
	}

	names, err := loadNamesForCategory(category, local)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		_, _ = fmt.Fprintf(stdout, "No %s configured.\n", category)
		return nil
	}

	singular := strings.TrimSuffix(category, "s")
	_, _ = fmt.Fprintln(stdout)
	selected, err := promptSelectOneFromList(stdout, stdin, singular, names)
	if err != nil || selected == "" {
		return err
	}

	subCmd, _, err := cmd.Root().Find([]string{"config", singular, "edit"})
	if err != nil || !strings.HasPrefix(subCmd.Use, "edit") {
		return fmt.Errorf("config %s edit command not found", singular)
	}
	if subCmd.RunE == nil {
		return fmt.Errorf("config %s edit command not executable", singular)
	}
	subCmd.SetContext(cmd.Context())
	return subCmd.RunE(subCmd, []string{selected})
}

// addConfigInteractiveRemoveCommand registers the top-level "config remove" command.
func addConfigInteractiveRemoveCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove one or more config items interactively",
		Long: `Interactively select and remove agents, roles, contexts, or tasks.

Prompts for the category, shows a numbered list for item selection, then
confirms before removing.`,
		Args: noArgsOrHelp,
		RunE: runConfigInteractiveRemove,
	}
	parent.AddCommand(cmd)
}

// runConfigInteractiveRemove picks a category and items then delegates to the category remove command.
func runConfigInteractiveRemove(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	if !isTerminal(stdin) {
		return fmt.Errorf("interactive remove requires a terminal")
	}

	_, _ = fmt.Fprintln(stdout, "Remove:")
	category, err := promptSelectCategory(stdout, stdin, allConfigCategories)
	if err != nil || category == "" {
		return err
	}

	names, err := loadNamesForCategory(category, local)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		_, _ = fmt.Fprintf(stdout, "No %s configured.\n", category)
		return nil
	}

	singular := strings.TrimSuffix(category, "s")
	_, _ = fmt.Fprintln(stdout)
	selected, err := promptSelectFromList(stdout, stdin, singular, "", names)
	if err != nil || selected == nil {
		return err
	}

	subCmd, _, err := cmd.Root().Find([]string{"config", singular, "remove"})
	if err != nil || !strings.HasPrefix(subCmd.Use, "remove") {
		return fmt.Errorf("config %s remove command not found", singular)
	}
	if subCmd.RunE == nil {
		return fmt.Errorf("config %s remove command not executable", singular)
	}
	subCmd.SetContext(cmd.Context())
	return subCmd.RunE(subCmd, selected)
}

// loadNamesForCategory loads the ordered list of names for a config category.
func loadNamesForCategory(category string, local bool) ([]string, error) {
	switch category {
	case "agents":
		_, order, err := loadAgentsForScope(local)
		return order, err
	case "roles":
		_, order, err := loadRolesForScope(local)
		return order, err
	case "contexts":
		_, order, err := loadContextsForScope(local)
		return order, err
	case "tasks":
		_, order, err := loadTasksForScope(local)
		return order, err
	default:
		return nil, fmt.Errorf("unknown category %q", category)
	}
}
