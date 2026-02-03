package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// addAssetsAddCommand adds the add subcommand to the assets command.
func addAssetsAddCommand(parent *cobra.Command) {
	addCmd := &cobra.Command{
		Use:     "add <query>",
		Aliases: []string{"install"},
		Short:   "Install asset from registry",
		Long: `Install an asset from the CUE registry to your configuration.

Searches the registry index for matching assets. If multiple matches are found,
prompts for selection. Use a direct path (e.g., "golang/code-review") for exact match.

By default, installs to global config (~/.config/start/).
Use --local to install to project config (./.start/).`,
		Args: cobra.ExactArgs(1),
		RunE: runAssetsAdd,
	}

	parent.AddCommand(addCmd)
}

// runAssetsAdd searches for and installs an asset.
func runAssetsAdd(cmd *cobra.Command, args []string) error {
	query := args[0]
	ctx := context.Background()
	flags := getFlags(cmd)

	// Create registry client
	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	// Fetch index
	if !flags.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Fetching index...")
	}
	index, err := client.FetchIndex(ctx)
	if err != nil {
		return fmt.Errorf("fetching index: %w", err)
	}

	// Search for matching assets
	results := assets.SearchIndex(index, query)

	if len(results) == 0 {
		return fmt.Errorf("no assets found matching %q", query)
	}

	// Select asset
	var selected assets.SearchResult
	if len(results) == 1 {
		selected = results[0]
	} else {
		selected, err = promptAssetSelection(cmd.OutOrStdout(), cmd.InOrStdin(), results)
		if err != nil {
			return err
		}
	}

	// Determine config path
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	var scopeName string
	local := getFlags(cmd).Local
	if local {
		configDir = paths.Local
		scopeName = "local"
	} else {
		configDir = paths.Global
		scopeName = "global"
	}

	// Install the asset
	if !flags.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Fetching asset...")
	}

	if err := assets.InstallAsset(ctx, client, selected, configDir); err != nil {
		return err
	}

	if !flags.Quiet {
		configFile := map[string]string{
			"agents":   "agents.cue",
			"roles":    "roles.cue",
			"tasks":    "tasks.cue",
			"contexts": "contexts.cue",
		}[selected.Category]
		if configFile == "" {
			configFile = "settings.cue"
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nInstalled %s/%s to %s config\n", selected.Category, selected.Name, scopeName)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Config: %s/%s\n", configDir, configFile)
	}

	return nil
}

// promptAssetSelection prompts the user to select an asset from multiple matches.
func promptAssetSelection(w io.Writer, r io.Reader, results []assets.SearchResult) (assets.SearchResult, error) {
	// Check if stdin is a TTY
	isTTY := false
	if f, ok := r.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	if !isTTY {
		var names []string
		for _, res := range results {
			names = append(names, fmt.Sprintf("%s/%s", res.Category, res.Name))
		}
		return assets.SearchResult{}, fmt.Errorf(
			"multiple assets found: %s\nSpecify exact path or run interactively",
			strings.Join(names, ", "),
		)
	}

	_, _ = fmt.Fprintf(w, "Found %d matches:\n\n", len(results))

	for i, res := range results {
		_, _ = fmt.Fprintf(w, "  %d. %s/%s - %s\n", i+1, res.Category, res.Name, res.Entry.Description)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprint(w, "Select asset (number or name): ")

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return assets.SearchResult{}, fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Try parsing as number
	if choice, err := strconv.Atoi(input); err == nil {
		if choice >= 1 && choice <= len(results) {
			return results[choice-1], nil
		}
		return assets.SearchResult{}, fmt.Errorf("invalid selection: %s (choose 1-%d)", input, len(results))
	}

	// Try matching by name
	inputLower := strings.ToLower(input)
	for _, res := range results {
		fullPath := fmt.Sprintf("%s/%s", res.Category, res.Name)
		if strings.ToLower(res.Name) == inputLower || strings.ToLower(fullPath) == inputLower {
			return res, nil
		}
	}

	return assets.SearchResult{}, fmt.Errorf("invalid selection: %s", input)
}
