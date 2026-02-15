package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/grantcarthew/start/internal/config"
	"github.com/spf13/cobra"
)

// anyFlagChanged returns true if any of the named flags were explicitly set.
func anyFlagChanged(cmd *cobra.Command, names ...string) bool {
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

// addConfigCommand adds the config command group and its subcommands to the parent.
func addConfigCommand(parent *cobra.Command) {
	configCmd := &cobra.Command{
		Use:     "config",
		GroupID: "commands",
		Short:   "Manage start configuration",
		Long: `Manage configuration for agents, roles, contexts, and tasks.

Configuration can be stored globally (~/.config/start/) or locally (./.start/).
Use --local to target project-specific configuration.`,
		RunE: runConfigList,
	}

	// Add entity subcommand groups
	addConfigAgentCommand(configCmd)
	addConfigRoleCommand(configCmd)
	addConfigContextCommand(configCmd)
	addConfigTaskCommand(configCmd)
	addConfigSettingsCommand(configCmd)
	addConfigOrderCommand(configCmd)

	parent.AddCommand(configCmd)
}

// runConfigList displays an overview of all configuration.
func runConfigList(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}
	if len(args) > 0 {
		return unknownCommandError("start config", args[0])
	}

	w := cmd.OutOrStdout()
	flags := getFlags(cmd)
	local := flags.Local

	// Show config paths
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	_, _ = colorHeader.Fprintln(w, "Configuration Paths:")
	_, _ = fmt.Fprintln(w)
	globalStatus := "not found"
	if paths.GlobalExists {
		globalStatus = "exists"
	}
	localStatus := "not found"
	if paths.LocalExists {
		localStatus = "exists"
	}
	_, _ = fmt.Fprintf(w, "  Global: %s ", paths.Global)
	_, _ = colorCyan.Fprint(w, "(")
	_, _ = colorDim.Fprint(w, globalStatus)
	_, _ = colorCyan.Fprintln(w, ")")
	_, _ = fmt.Fprintf(w, "  Local:  %s ", paths.Local)
	_, _ = colorCyan.Fprint(w, "(")
	_, _ = colorDim.Fprint(w, localStatus)
	_, _ = colorCyan.Fprintln(w, ")")

	// Determine scope for listing
	scopeLabel := "merged"
	if local {
		scopeLabel = "local"
	}

	stderr := cmd.ErrOrStderr()

	// Agents
	agents, err := loadAgentsForScope(local)
	if err != nil {
		printWarning(stderr, "failed to load agents: %s", err)
	}
	_, _ = fmt.Fprintln(w)
	_, _ = colorAgents.Fprint(w, "agents")
	_, _ = fmt.Fprint(w, "/ ")
	_, _ = colorCyan.Fprint(w, "(")
	_, _ = colorDim.Fprintf(w, "%s", scopeLabel)
	_, _ = colorCyan.Fprint(w, ")")
	_, _ = colorDim.Fprintf(w, ": %d\n", len(agents))
	if len(agents) > 0 {
		defaultAgent := ""
		if cfg, err := loadConfigForScope(local); err == nil {
			defaultAgent = getDefaultAgentFromConfig(cfg)
		}
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
			_, _ = fmt.Fprintf(w, "  %s%s ", marker, name)
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, agent.Source)
			_, _ = colorCyan.Fprintln(w, ")")
		}
	}

	// Roles
	roles, roleOrder, err := loadRolesForScope(local)
	if err != nil {
		printWarning(stderr, "failed to load roles: %s", err)
	}
	_, _ = fmt.Fprintln(w)
	_, _ = colorRoles.Fprint(w, "roles")
	_, _ = fmt.Fprint(w, "/ ")
	_, _ = colorCyan.Fprint(w, "(")
	_, _ = colorDim.Fprintf(w, "%s", scopeLabel)
	_, _ = colorCyan.Fprint(w, ")")
	_, _ = colorDim.Fprintf(w, ": %d\n", len(roles))
	if len(roles) > 0 {
		for _, name := range roleOrder {
			role := roles[name]
			_, _ = fmt.Fprintf(w, "    %s ", name)
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, role.Source)
			_, _ = colorCyan.Fprintln(w, ")")
		}
	}

	// Contexts
	contexts, contextOrder, err := loadContextsForScope(local)
	if err != nil {
		printWarning(stderr, "failed to load contexts: %s", err)
	}
	_, _ = fmt.Fprintln(w)
	_, _ = colorContexts.Fprint(w, "contexts")
	_, _ = fmt.Fprint(w, "/ ")
	_, _ = colorCyan.Fprint(w, "(")
	_, _ = colorDim.Fprintf(w, "%s", scopeLabel)
	_, _ = colorCyan.Fprint(w, ")")
	_, _ = colorDim.Fprintf(w, ": %d\n", len(contexts))
	if len(contexts) > 0 {
		for _, name := range contextOrder {
			ctx := contexts[name]
			_, _ = fmt.Fprintf(w, "    %s ", name)
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, ctx.Source)
			_, _ = colorCyan.Fprint(w, ")")
			if ctx.Required {
				_, _ = fmt.Fprint(w, " ")
				_, _ = colorCyan.Fprint(w, "[")
				_, _ = colorDim.Fprint(w, "required")
				_, _ = colorCyan.Fprint(w, "]")
			}
			if ctx.Default {
				_, _ = fmt.Fprint(w, " ")
				_, _ = colorCyan.Fprint(w, "[")
				_, _ = colorDim.Fprint(w, "default")
				_, _ = colorCyan.Fprint(w, "]")
			}
			if len(ctx.Tags) > 0 {
				_, _ = fmt.Fprint(w, " ")
				_, _ = colorDim.Fprint(w, "tags:")
				_, _ = colorCyan.Fprint(w, "[")
				_, _ = colorDim.Fprintf(w, "%s", strings.Join(ctx.Tags, ", "))
				_, _ = colorCyan.Fprint(w, "]")
			}
			_, _ = fmt.Fprintln(w)
		}
	}

	// Tasks
	tasks, err := loadTasksForScope(local)
	if err != nil {
		printWarning(stderr, "failed to load tasks: %s", err)
	}
	_, _ = fmt.Fprintln(w)
	_, _ = colorTasks.Fprint(w, "tasks")
	_, _ = fmt.Fprint(w, "/ ")
	_, _ = colorCyan.Fprint(w, "(")
	_, _ = colorDim.Fprintf(w, "%s", scopeLabel)
	_, _ = colorCyan.Fprint(w, ")")
	_, _ = colorDim.Fprintf(w, ": %d\n", len(tasks))
	if len(tasks) > 0 {
		var names []string
		for name := range tasks {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			task := tasks[name]
			_, _ = fmt.Fprintf(w, "    %s ", name)
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, task.Source)
			_, _ = colorCyan.Fprintln(w, ")")
		}
	}

	return nil
}
