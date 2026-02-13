package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/grantcarthew/start/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// addConfigOrderCommand adds the bare order command to the config command.
func addConfigOrderCommand(parent *cobra.Command) {
	orderCmd := &cobra.Command{
		Use:     "order",
		Aliases: []string{"reorder"},
		Short:   "Reorder configuration items",
		Long: `Reorder contexts or roles interactively.

Prompts to choose between contexts or roles, then provides
an interactive reorder flow.`,
		Args: cobra.NoArgs,
		RunE: runConfigOrder,
	}

	parent.AddCommand(orderCmd)
}

// runConfigOrder prompts the user to select contexts or roles, then runs reorder.
func runConfigOrder(cmd *cobra.Command, _ []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()

	if !isInteractiveInput(stdin) {
		return fmt.Errorf("interactive reordering requires a terminal")
	}

	_, _ = fmt.Fprintln(stdout, "Reorder:")
	_, _ = fmt.Fprintln(stdout, "  1. Contexts")
	_, _ = fmt.Fprintln(stdout, "  2. Roles")
	_, _ = fmt.Fprintf(stdout, "Choice %s%s%s: ", colorCyan.Sprint("["), colorDim.Sprint("1"), colorCyan.Sprint("]"))

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
		return runConfigContextOrder(cmd, nil)
	case "2":
		return runConfigRoleOrder(cmd, nil)
	default:
		return fmt.Errorf("invalid choice: %s", choice)
	}
}

// addConfigContextOrderCommand adds the order subcommand to the context command.
func addConfigContextOrderCommand(parent *cobra.Command) {
	orderCmd := &cobra.Command{
		Use:     "order",
		Aliases: []string{"reorder"},
		Short:   "Reorder contexts",
		Long: `Reorder context configuration items interactively.

Displays a numbered list of contexts. Enter a number to move that
item up one position. Repeat to achieve the desired order.

Press Enter to save the new order, or q to cancel.`,
		Args: cobra.NoArgs,
		RunE: runConfigContextOrder,
	}

	parent.AddCommand(orderCmd)
}

// runConfigContextOrder runs the interactive reorder flow for contexts.
func runConfigContextOrder(cmd *cobra.Command, _ []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	if !isInteractiveInput(stdin) {
		return fmt.Errorf("interactive reordering requires a terminal")
	}

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

	contexts, order, err := loadContextsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading contexts: %w", err)
	}

	if len(order) == 0 {
		_, _ = fmt.Fprintln(stdout, "No contexts configured.")
		return nil
	}

	contextPath := filepath.Join(configDir, "contexts.cue")
	heading := fmt.Sprintf("Reorder Contexts %s%s%s:", colorCyan.Sprint("("), colorDim.Sprintf("%s - %s", scopeString(local), contextPath), colorCyan.Sprint(")"))

	formatItem := func(i int, name string) string {
		ctx := contexts[name]
		markers := ""
		if ctx.Required {
			markers += " " + colorCyan.Sprint("[") + colorDim.Sprint("required") + colorCyan.Sprint("]")
		}
		if ctx.Default {
			markers += " " + colorCyan.Sprint("[") + colorDim.Sprint("default") + colorCyan.Sprint("]")
		}
		return fmt.Sprintf("  %d. %s%s", i+1, name, markers)
	}

	newOrder, saved, err := runReorderLoop(stdout, stdin, heading, order, formatItem)
	if err != nil {
		return err
	}

	if !saved {
		_, _ = fmt.Fprintln(stdout, "Cancelled.")
		return nil
	}

	if err := writeContextsFile(contextPath, contexts, newOrder); err != nil {
		return fmt.Errorf("writing contexts file: %w", err)
	}

	_, _ = fmt.Fprintln(stdout, "Order saved.")
	return nil
}

// addConfigRoleOrderCommand adds the order subcommand to the role command.
func addConfigRoleOrderCommand(parent *cobra.Command) {
	orderCmd := &cobra.Command{
		Use:     "order",
		Aliases: []string{"reorder"},
		Short:   "Reorder roles",
		Long: `Reorder role configuration items interactively.

Displays a numbered list of roles. Enter a number to move that
item up one position. Repeat to achieve the desired order.

Press Enter to save the new order, or q to cancel.`,
		Args: cobra.NoArgs,
		RunE: runConfigRoleOrder,
	}

	parent.AddCommand(orderCmd)
}

// runConfigRoleOrder runs the interactive reorder flow for roles.
func runConfigRoleOrder(cmd *cobra.Command, _ []string) error {
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	local := getFlags(cmd).Local

	if !isInteractiveInput(stdin) {
		return fmt.Errorf("interactive reordering requires a terminal")
	}

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

	roles, order, err := loadRolesFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading roles: %w", err)
	}

	if len(order) == 0 {
		_, _ = fmt.Fprintln(stdout, "No roles configured.")
		return nil
	}

	rolePath := filepath.Join(configDir, "roles.cue")
	heading := fmt.Sprintf("Reorder Roles %s%s%s:", colorCyan.Sprint("("), colorDim.Sprintf("%s - %s", scopeString(local), rolePath), colorCyan.Sprint(")"))

	formatItem := func(i int, name string) string {
		role := roles[name]
		markers := ""
		if role.Optional {
			markers += " " + colorCyan.Sprint("[") + colorDim.Sprint("optional") + colorCyan.Sprint("]")
		}
		return fmt.Sprintf("  %d. %s%s", i+1, name, markers)
	}

	newOrder, saved, err := runReorderLoop(stdout, stdin, heading, order, formatItem)
	if err != nil {
		return err
	}

	if !saved {
		_, _ = fmt.Fprintln(stdout, "Cancelled.")
		return nil
	}

	if err := writeRolesFile(rolePath, roles, newOrder); err != nil {
		return fmt.Errorf("writing roles file: %w", err)
	}

	_, _ = fmt.Fprintln(stdout, "Order saved.")
	return nil
}

// isInteractiveInput returns true if the reader supports interactive input.
// Returns true for TTY file descriptors or for non-file readers (e.g., explicitly
// set via cmd.SetIn in tests). Returns false for piped file descriptors.
func isInteractiveInput(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		// Not a file - stdin was explicitly set (e.g., in tests)
		return true
	}
	return term.IsTerminal(int(f.Fd()))
}

// runReorderLoop runs the interactive move-up reorder loop.
// Returns the final order, whether the user saved (true) or cancelled (false), and any error.
func runReorderLoop(w io.Writer, r io.Reader, heading string, order []string, formatItem func(int, string) string) ([]string, bool, error) {
	// Make a copy to avoid mutating the caller's slice
	current := make([]string, len(order))
	copy(current, order)

	reader := bufio.NewReader(r)

	// Display initial list
	_, _ = fmt.Fprintln(w, heading)
	_, _ = fmt.Fprintln(w)
	for i, name := range current {
		_, _ = fmt.Fprintln(w, formatItem(i, name))
	}

	for {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "Move up %s%s%s, Enter to save, q to cancel: ", colorCyan.Sprint("("), colorDim.Sprint("number"), colorCyan.Sprint(")"))

		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, false, fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(input)

		// Empty input: save
		if input == "" {
			return current, true, nil
		}

		// Cancel
		lower := strings.ToLower(input)
		if lower == "q" || lower == "quit" || lower == "exit" {
			return nil, false, nil
		}

		// Try to parse as number
		num, err := strconv.Atoi(input)
		if err != nil {
			_, _ = fmt.Fprintf(w, "Invalid input: %s\n", input)
			continue
		}

		if num < 1 || num > len(current) {
			_, _ = fmt.Fprintf(w, "Invalid number: %d %s%s%s\n", num, colorCyan.Sprint("("), colorDim.Sprintf("must be 1-%d", len(current)), colorCyan.Sprint(")"))
			continue
		}

		if num == 1 {
			_, _ = fmt.Fprintln(w, "Already at top.")
			continue
		}

		// Swap item at position num with the one above it (num-1)
		idx := num - 1
		current[idx], current[idx-1] = current[idx-1], current[idx]

		// Re-display
		_, _ = fmt.Fprintln(w)
		for i, name := range current {
			_, _ = fmt.Fprintln(w, formatItem(i, name))
		}
	}
}
