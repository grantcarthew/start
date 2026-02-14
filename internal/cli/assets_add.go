package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"golang.org/x/term"
)

// NOTE(design): This file shares registry client creation, index fetching, and config
// loading patterns with assets_list.go, assets_search.go, assets_update.go, and
// assets_index.go. This duplication is accepted - each command uses the results
// differently and a shared helper would couple them for modest line savings.

// addAssetsAddCommand adds the add subcommand to the assets command.
func addAssetsAddCommand(parent *cobra.Command) {
	addCmd := &cobra.Command{
		Use:     "add <query>...",
		Aliases: []string{"install"},
		Short:   "Install assets from registry",
		Long: `Install one or more assets from the CUE registry to your configuration.

Searches the registry index for matching assets. If multiple matches are found,
prompts for selection. Use a direct path (e.g., "golang/code-review") for exact match.

Multiple queries can be provided to install several assets at once.

By default, installs to global config (~/.config/start/).
Use --local to install to project config (./.start/).`,
		Args: cobra.MinimumNArgs(1),
		RunE: runAssetsAdd,
	}

	parent.AddCommand(addCmd)
}

// runAssetsAdd searches for and installs one or more assets.
func runAssetsAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	flags := getFlags(cmd)
	w := cmd.OutOrStdout()

	// Create registry client
	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	// Fetch index
	if !flags.Quiet {
		_, _ = fmt.Fprintln(w, "Fetching index...")
	}
	index, err := client.FetchIndex(ctx)
	if err != nil {
		return fmt.Errorf("fetching index: %w", err)
	}

	// Determine config path
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	var scopeName string
	if flags.Local {
		configDir = paths.Local
		scopeName = "local"
	} else {
		configDir = paths.Global
		scopeName = "global"
	}

	// Load CUE config once for existence checks across all queries.
	// On error with no CUE files (fresh install), cfg is a zero-value cue.Value;
	// LookupPath on it returns non-existent, so AssetExists correctly returns false.
	loader := internalcue.NewLoader()
	cfg, err := loader.LoadSingle(configDir)
	if err != nil {
		if matches, _ := filepath.Glob(filepath.Join(configDir, "*.cue")); len(matches) > 0 {
			return fmt.Errorf("invalid config in %s:\n%s\nRun 'start doctor' to diagnose",
				configDir, internalcue.IdentifyBrokenFiles(matches))
		}
	}

	// Install each queried asset
	var errs []error
	for _, query := range args {
		if err := installAsset(ctx, cmd, client, index, query, configDir, scopeName, flags, cfg); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", query, err))
			_, _ = fmt.Fprintf(w, "Error installing %q: %v\n", query, err)
		}
	}

	return errors.Join(errs...)
}

// installAsset searches for, selects, and installs a single asset.
func installAsset(ctx context.Context, cmd *cobra.Command, client *registry.Client, index *registry.Index, query, configDir, scopeName string, flags *Flags, cfg cue.Value) error {
	w := cmd.OutOrStdout()

	// Search for matching assets
	results, err := assets.SearchIndex(index, query)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no assets found matching %q", query)
	}

	// Select asset
	var selected assets.SearchResult
	if len(results) == 1 {
		selected = results[0]
	} else {
		var err error
		selected, err = promptAssetSelection(w, cmd.InOrStdin(), results, cfg)
		if err != nil {
			return err
		}
	}

	// Check if already installed
	if assets.AssetExists(cfg, selected.Category, selected.Name) {
		origin := assets.GetInstalledOrigin(cfg, selected.Category, selected.Name)

		// Manually-added asset (no origin) — warn and proceed with install
		if origin == "" {
			if !flags.Quiet {
				printWarning(w, "replacing manually-added %s/%s with registry version",
					selected.Category, selected.Name)
			}
		} else {
			if !flags.Quiet {
				installedVer := assets.VersionFromOrigin(origin)
				latestVer := selected.Entry.Version
				outdated := latestVer != "" && installedVer != "" && semver.Compare(latestVer, installedVer) > 0

				if outdated {
					_, _ = fmt.Fprint(w, "○ ")
				} else {
					_, _ = colorSuccess.Fprint(w, "✓ ")
				}
				_, _ = colorDim.Fprint(w, "Already installed: ")
				_, _ = categoryColor(selected.Category).Fprint(w, selected.Category)
				_, _ = fmt.Fprintf(w, "/%s ", selected.Name)
				_, _ = colorCyan.Fprint(w, "(")
				if installedVer != "" {
					_, _ = colorDim.Fprint(w, installedVer)
				}
				if outdated {
					_, _ = fmt.Fprint(w, " ")
					_, _ = colorBlue.Fprint(w, "->")
					_, _ = fmt.Fprint(w, " ")
					_, _ = colorWarning.Fprint(w, latestVer)
				} else {
					_, _ = fmt.Fprint(w, " ")
					_, _ = colorBlue.Fprint(w, "->")
					_, _ = fmt.Fprint(w, " ")
					_, _ = colorDim.Fprint(w, "current")
				}
				_, _ = colorCyan.Fprintln(w, ")")
			}
			return nil
		}
	}

	// Install the asset
	if !flags.Quiet {
		_, _ = fmt.Fprintln(w, "Fetching asset...")
	}

	if err := assets.InstallAsset(ctx, client, index, selected, configDir); err != nil {
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
		_, _ = fmt.Fprintf(w, "\nInstalled %s/%s to %s config\n", selected.Category, selected.Name, scopeName)
		_, _ = fmt.Fprintf(w, "Config: %s/%s\n", configDir, configFile)
	}

	return nil
}

// promptAssetSelection prompts the user to select an asset from multiple matches.
func promptAssetSelection(w io.Writer, r io.Reader, results []assets.SearchResult, cfg cue.Value) (assets.SearchResult, error) {
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
		marker := "  "
		if assets.AssetExists(cfg, res.Category, res.Name) {
			marker = colorInstalled.Sprint("★") + " "
		}
		_, _ = fmt.Fprintf(w, "  %s%d. ", marker, i+1)
		_, _ = categoryColor(res.Category).Fprint(w, res.Category)
		_, _ = fmt.Fprintf(w, "/%s ", res.Name)
		_, _ = colorDim.Fprintf(w, "- %s", res.Entry.Description)
		_, _ = fmt.Fprintln(w)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprint(w, "Select asset ")
	_, _ = colorCyan.Fprint(w, "(")
	_, _ = colorDim.Fprint(w, "number or name")
	_, _ = colorCyan.Fprint(w, ")")
	_, _ = fmt.Fprint(w, ": ")

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
