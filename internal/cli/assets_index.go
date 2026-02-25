package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/mod/modconfig"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/grantcarthew/start/internal/tui"
	"github.com/spf13/cobra"
)

// NOTE(design): This file shares registry client creation, index fetching, and config
// loading patterns with assets_add.go, assets_list.go, assets_search.go, and
// assets_update.go. This duplication is accepted - each command uses the results
// differently and a shared helper would couple them for modest line savings.

// addAssetsIndexCommand adds the index subcommand to the assets command.
func addAssetsIndexCommand(parent *cobra.Command) {
	indexCmd := &cobra.Command{
		Use:     "index [category]",
		Aliases: []string{"idx"},
		Short:   "Show registry asset catalog",
		Long: `Display the full asset catalog from the CUE Central Registry.

Shows all available assets grouped by type (agents, roles, tasks, contexts).
Installed assets are marked with ★.

Optionally filter by category: agents, roles, tasks, or contexts.
Category filtering is supported with --json but not with --raw.

Use --json to output machine-readable JSON, or --raw to display the
raw CUE source files from the index module.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAssetsIndex,
	}

	indexCmd.Flags().Bool("json", false, "Output index as JSON")
	indexCmd.Flags().Bool("raw", false, "Output raw CUE source files")

	parent.AddCommand(indexCmd)
}

// runAssetsIndex fetches and displays the full registry asset catalog.
func runAssetsIndex(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}

	// Validate category arg before any network I/O
	var category string
	if len(args) > 0 {
		singular := normalizeCategoryArg(args[0])
		if singular == "" {
			return fmt.Errorf("unknown category %q: expected agents, roles, tasks, or contexts", args[0])
		}
		category = singular + "s"
	}

	ctx := context.Background()
	flags := getFlags(cmd)
	jsonFlag, _ := cmd.Flags().GetBool("json")
	rawFlag, _ := cmd.Flags().GetBool("raw")

	// Create registry client
	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	// Resolve latest version
	indexPath := registry.EffectiveIndexPath(resolveAssetsIndexPath())
	resolvedPath, err := client.ResolveLatestVersion(ctx, indexPath)
	if err != nil {
		return fmt.Errorf("resolving index version: %w", err)
	}

	// Extract version string (after @)
	version := assets.VersionFromOrigin(resolvedPath)
	if version == "" {
		version = resolvedPath
	}

	// Fetch module
	result, err := client.Fetch(ctx, resolvedPath)
	if err != nil {
		return fmt.Errorf("fetching index module: %w", err)
	}

	w := cmd.OutOrStdout()

	switch {
	case rawFlag:
		if category != "" {
			return fmt.Errorf("category filter cannot be used with --raw")
		}
		return printRawIndex(w, result.SourceDir)
	case jsonFlag:
		return printJSONIndex(w, result.SourceDir, client.Registry(), category)
	default:
		index, err := registry.LoadIndex(result.SourceDir, client.Registry())
		if err != nil {
			return fmt.Errorf("loading index: %w", err)
		}
		installed := collectInstalledNames()
		printIndex(w, index, version, flags.Verbose, installed, category)
		return nil
	}
}

// printRawIndex reads and prints all .cue files from the index source directory.
func printRawIndex(w io.Writer, sourceDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("reading index directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".cue" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		_, _ = fmt.Fprintf(w, "// %s\n", entry.Name())
		_, _ = fmt.Fprint(w, string(data))
		_, _ = fmt.Fprintln(w)
	}
	return nil
}

// printJSONIndex loads the index and outputs it as formatted JSON.
// If category is non-empty, only that category is included in the output.
func printJSONIndex(w io.Writer, sourceDir string, reg modconfig.Registry, category string) error {
	index, err := registry.LoadIndex(sourceDir, reg)
	if err != nil {
		return fmt.Errorf("loading index: %w", err)
	}

	if category != "" {
		index = filterIndexByCategory(index, category)
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling index: %w", err)
	}

	_, _ = fmt.Fprintln(w, string(data))
	return nil
}

// filterIndexByCategory returns a new Index containing only the named category.
func filterIndexByCategory(index *registry.Index, category string) *registry.Index {
	switch category {
	case "agents":
		return &registry.Index{Agents: index.Agents}
	case "roles":
		return &registry.Index{Roles: index.Roles}
	case "tasks":
		return &registry.Index{Tasks: index.Tasks}
	case "contexts":
		return &registry.Index{Contexts: index.Contexts}
	default:
		return index
	}
}

// printIndex prints the index in a formatted table grouped by category.
// If category is non-empty, only that category is shown; the total count in the
// header always reflects the full index.
func printIndex(w io.Writer, index *registry.Index, version string, verbose bool, installed map[string]bool, category string) {
	total := len(index.Agents) + len(index.Roles) + len(index.Tasks) + len(index.Contexts)
	_, _ = fmt.Fprintf(w, "Index: %s (%d assets)\n\n", version, total)

	categories := []struct {
		name    string
		entries map[string]registry.IndexEntry
	}{
		{"agents", index.Agents},
		{"roles", index.Roles},
		{"tasks", index.Tasks},
		{"contexts", index.Contexts},
	}

	for _, cat := range categories {
		if len(cat.entries) == 0 {
			continue
		}
		if category != "" && cat.name != category {
			continue
		}

		// Sort names alphabetically
		names := make([]string, 0, len(cat.entries))
		for name := range cat.entries {
			names = append(names, name)
		}
		sort.Strings(names)

		_, _ = tui.CategoryColor(cat.name).Fprint(w, cat.name)
		_, _ = fmt.Fprintf(w, "/ %s\n", tui.Annotate("%d", len(cat.entries)))

		for _, name := range names {
			entry := cat.entries[name]
			marker := "  "
			if installed[cat.name+"/"+name] {
				marker = tui.ColorInstalled.Sprint("★") + " "
			}

			_, _ = fmt.Fprintf(w, "  %s%-25s %s\n", marker, name, tui.ColorDim.Sprint(entry.Description))

			if verbose {
				_, _ = fmt.Fprintf(w, "      Module: %s\n", tui.ColorDim.Sprint(entry.Module))
				if len(entry.Tags) > 0 {
					_, _ = fmt.Fprintf(w, "      Tags: %s\n", tui.ColorDim.Sprint(strings.Join(entry.Tags, ", ")))
				}
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}
