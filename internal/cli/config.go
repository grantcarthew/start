package cli

import (
	"fmt"
	"sort"

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
		Use:   "config",
		Short: "Manage start configuration",
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

	_, _ = fmt.Fprintln(w, "Configuration Paths:")
	_, _ = fmt.Fprintln(w)
	globalStatus := "not found"
	if paths.GlobalExists {
		globalStatus = "exists"
	}
	localStatus := "not found"
	if paths.LocalExists {
		localStatus = "exists"
	}
	_, _ = fmt.Fprintf(w, "  Global: %s (%s)\n", paths.Global, globalStatus)
	_, _ = fmt.Fprintf(w, "  Local:  %s (%s)\n", paths.Local, localStatus)

	// Determine scope for listing
	scopeLabel := "merged"
	if local {
		scopeLabel = "local"
	}

	// Agents
	agents, _ := loadAgentsForScope(local)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Agents (%s): %d\n", scopeLabel, len(agents))
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
				marker = "* "
			}
			_, _ = fmt.Fprintf(w, "  %s%s (%s)\n", marker, name, agent.Source)
		}
	}

	// Roles
	roles, _ := loadRolesForScope(local)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Roles (%s): %d\n", scopeLabel, len(roles))
	if len(roles) > 0 {
		defaultRole := ""
		if cfg, err := loadConfigForScope(local); err == nil {
			defaultRole = getDefaultRoleFromConfig(cfg)
		}
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
			_, _ = fmt.Fprintf(w, "  %s%s (%s)\n", marker, name, role.Source)
		}
	}

	// Contexts
	contexts, contextOrder, _ := loadContextsForScope(local)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Contexts (%s): %d\n", scopeLabel, len(contexts))
	if len(contexts) > 0 {
		for _, name := range contextOrder {
			ctx := contexts[name]
			flags := ""
			if ctx.Required {
				flags += " [required]"
			}
			if ctx.Default {
				flags += " [default]"
			}
			if len(ctx.Tags) > 0 {
				flags += fmt.Sprintf(" tags:%v", ctx.Tags)
			}
			_, _ = fmt.Fprintf(w, "    %s (%s)%s\n", name, ctx.Source, flags)
		}
	}

	// Tasks
	tasks, _ := loadTasksForScope(local)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Tasks (%s): %d\n", scopeLabel, len(tasks))
	if len(tasks) > 0 {
		var names []string
		for name := range tasks {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			task := tasks[name]
			_, _ = fmt.Fprintf(w, "    %s (%s)\n", name, task.Source)
		}
	}

	return nil
}
