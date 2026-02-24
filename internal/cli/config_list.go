package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/grantcarthew/start/internal/tui"
	"github.com/spf13/cobra"
)

// addConfigListCommand adds the "config list [category]" command.
func addConfigListCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "list [category]",
		Aliases: []string{"ls"},
		Short:   "List configuration items",
		Long: `List configured agents, roles, contexts, and tasks.

Without a category, lists all items grouped by category.
With a category (agent, role, context, task), lists only that category.

Plural aliases (agents, roles, contexts, tasks) are accepted.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runConfigListCmd,
	}
	parent.AddCommand(cmd)
}

// runConfigListCmd is the handler for "config list [category]".
func runConfigListCmd(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}

	local := getFlags(cmd).Local
	w := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	if len(args) == 0 {
		// List all categories
		if err := listAgents(w, stderr, local); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(w)
		if err := listRoles(w, stderr, local); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(w)
		if err := listContexts(w, stderr, local); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(w)
		return listTasks(w, stderr, local)
	}

	category := normalizeCategoryArg(args[0])
	if category == "" {
		return fmt.Errorf("unknown category %q: expected agent, role, context, or task", args[0])
	}

	switch category {
	case "agent":
		return listAgents(w, stderr, local)
	case "role":
		return listRoles(w, stderr, local)
	case "context":
		return listContexts(w, stderr, local)
	case "task":
		return listTasks(w, stderr, local)
	}
	return nil
}

// listAgents prints the agents section to w.
func listAgents(w io.Writer, stderr io.Writer, local bool) error {
	agents, order, err := loadAgentsForScope(local)
	if err != nil {
		printWarning(stderr, "failed to load agents: %s", err)
	}

	_, _ = tui.ColorAgents.Fprint(w, "agents")
	_, _ = fmt.Fprintln(w, "/")

	if len(agents) == 0 {
		_, _ = tui.ColorDim.Fprintln(w, "  none")
		return nil
	}

	_, _ = fmt.Fprintln(w)

	defaultAgent := ""
	if cfg, err := loadConfigForScope(local); err == nil {
		defaultAgent = getDefaultAgentFromConfig(cfg)
	}

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

// listRoles prints the roles section to w.
func listRoles(w io.Writer, stderr io.Writer, local bool) error {
	roles, order, err := loadRolesForScope(local)
	if err != nil {
		printWarning(stderr, "failed to load roles: %s", err)
	}

	_, _ = tui.ColorRoles.Fprint(w, "roles")
	_, _ = fmt.Fprintln(w, "/")

	if len(roles) == 0 {
		_, _ = tui.ColorDim.Fprintln(w, "  none")
		return nil
	}

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

// listContexts prints the contexts section to w.
func listContexts(w io.Writer, stderr io.Writer, local bool) error {
	contexts, order, err := loadContextsForScope(local)
	if err != nil {
		printWarning(stderr, "failed to load contexts: %s", err)
	}

	_, _ = tui.ColorContexts.Fprint(w, "contexts")
	_, _ = fmt.Fprintln(w, "/")

	if len(contexts) == 0 {
		_, _ = tui.ColorDim.Fprintln(w, "  none")
		return nil
	}

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
		if len(ctx.Tags) > 0 {
			_, _ = fmt.Fprint(w, " ")
			_, _ = tui.ColorDim.Fprint(w, "tags:")
			_, _ = fmt.Fprint(w, tui.Bracket("%s", strings.Join(ctx.Tags, ", ")))
		}
		_, _ = fmt.Fprintln(w)
	}
	return nil
}

// runConfigTaskList is a Cobra-compatible wrapper used by task.go.
func runConfigTaskList(cmd *cobra.Command, _ []string) error {
	return listTasks(cmd.OutOrStdout(), cmd.ErrOrStderr(), getFlags(cmd).Local)
}

// listTasks prints the tasks section to w.
func listTasks(w io.Writer, stderr io.Writer, local bool) error {
	tasks, order, err := loadTasksForScope(local)
	if err != nil {
		printWarning(stderr, "failed to load tasks: %s", err)
	}

	_, _ = tui.ColorTasks.Fprint(w, "tasks")
	_, _ = fmt.Fprintln(w, "/")

	if len(tasks) == 0 {
		_, _ = tui.ColorDim.Fprintln(w, "  none")
		return nil
	}

	_, _ = fmt.Fprintln(w)

	for _, name := range order {
		task := tasks[name]
		source := task.Source
		if task.Origin != "" {
			source += ", registry"
		}
		if task.Description != "" {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = tui.ColorDim.Fprint(w, "- "+task.Description+" ")
			_, _ = fmt.Fprintln(w, tui.Annotate("%s", source))
		} else {
			_, _ = fmt.Fprintf(w, "  %s ", name)
			_, _ = fmt.Fprintln(w, tui.Annotate("%s", source))
		}
	}
	return nil
}
