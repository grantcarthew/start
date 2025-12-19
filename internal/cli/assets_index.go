package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/spf13/cobra"
)

// AssetRepoCategories are the directories that identify an asset repository.
var AssetRepoCategories = []string{"agents", "roles", "tasks", "contexts"}

// addAssetsIndexCommand adds the index subcommand to the assets command.
func addAssetsIndexCommand(parent *cobra.Command) {
	indexCmd := &cobra.Command{
		Use:   "index",
		Short: "Regenerate index.cue",
		Long: `Regenerate the index.cue file in an asset repository.

This command is for asset repository maintainers. It scans the asset directories
(agents/, roles/, tasks/, contexts/) and generates/updates the index/index.cue file.

Must be run from the root of an asset repository.`,
		Args: cobra.NoArgs,
		RunE: runAssetsIndex,
	}

	parent.AddCommand(indexCmd)
}

// runAssetsIndex regenerates the index.cue file.
func runAssetsIndex(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Verify this is an asset repository
	if !isAssetRepo(cwd) {
		return fmt.Errorf("not an asset repository\n\nRequired directories not found: %s/\n\nThis command is for asset repository maintainers only",
			strings.Join(AssetRepoCategories, "/, "))
	}

	// Scan assets
	index, err := scanAssetRepo(cwd)
	if err != nil {
		return fmt.Errorf("scanning assets: %w", err)
	}

	// Ensure index directory exists
	indexDir := filepath.Join(cwd, "index")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return fmt.Errorf("creating index directory: %w", err)
	}

	// Write index.cue
	indexPath := filepath.Join(indexDir, "index.cue")
	content := generateIndexCUE(index)
	if err := os.WriteFile(indexPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing index: %w", err)
	}

	flags := getFlags(cmd)
	if !flags.Quiet {
		printIndexSummary(cmd.OutOrStdout(), index, indexPath)
	}

	return nil
}

// isAssetRepo checks if the directory contains asset repository structure.
func isAssetRepo(dir string) bool {
	// At least one of the category directories must exist
	for _, cat := range AssetRepoCategories {
		catPath := filepath.Join(dir, cat)
		if info, err := os.Stat(catPath); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// ScannedAsset represents an asset found during scanning.
type ScannedAsset struct {
	Category    string
	Name        string
	Module      string
	Description string
	Tags        []string
	Bin         string // For agents only
}

// ScannedIndex holds all scanned assets.
type ScannedIndex struct {
	Agents   []ScannedAsset
	Roles    []ScannedAsset
	Tasks    []ScannedAsset
	Contexts []ScannedAsset
}

// scanAssetRepo scans the asset repository and returns the index.
func scanAssetRepo(repoDir string) (*ScannedIndex, error) {
	index := &ScannedIndex{}

	for _, cat := range AssetRepoCategories {
		catDir := filepath.Join(repoDir, cat)
		if _, err := os.Stat(catDir); os.IsNotExist(err) {
			continue
		}

		assets, err := scanCategory(catDir, cat)
		if err != nil {
			return nil, fmt.Errorf("scanning %s: %w", cat, err)
		}

		switch cat {
		case "agents":
			index.Agents = assets
		case "roles":
			index.Roles = assets
		case "tasks":
			index.Tasks = assets
		case "contexts":
			index.Contexts = assets
		}
	}

	return index, nil
}

// scanCategory scans a single category directory.
func scanCategory(catDir, category string) ([]ScannedAsset, error) {
	var assets []ScannedAsset

	// Walk the category directory looking for cue.mod directories
	err := filepath.Walk(catDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Check for cue.mod directory indicating a CUE module
		cueMod := filepath.Join(path, "cue.mod")
		if _, err := os.Stat(cueMod); os.IsNotExist(err) {
			return nil
		}

		// This is a CUE module - extract info
		asset, err := extractAssetInfo(path, category, catDir)
		if err != nil {
			// Log but don't fail on individual module errors
			return nil
		}

		assets = append(assets, asset)

		// Don't recurse into this module
		return filepath.SkipDir
	})

	if err != nil {
		return nil, err
	}

	// Sort by name
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Name < assets[j].Name
	})

	return assets, nil
}

// extractAssetInfo extracts metadata from a CUE module.
func extractAssetInfo(moduleDir, category, catDir string) (ScannedAsset, error) {
	// Calculate relative path for name
	relPath, err := filepath.Rel(catDir, moduleDir)
	if err != nil {
		return ScannedAsset{}, err
	}

	asset := ScannedAsset{
		Category: category,
		Name:     relPath,
	}

	// Read module path from cue.mod/module.cue
	moduleFile := filepath.Join(moduleDir, "cue.mod", "module.cue")
	asset.Module = readModulePath(moduleFile)

	// Load the CUE module to extract description and other metadata
	cctx := cuecontext.New()
	cfg := &load.Config{
		Dir: moduleDir,
	}

	insts := load.Instances([]string{"."}, cfg)
	if len(insts) == 0 || insts[0].Err != nil {
		return asset, nil // Return partial info
	}

	v := cctx.BuildInstance(insts[0])
	if v.Err() != nil {
		return asset, nil
	}

	// Extract common fields
	if desc := v.LookupPath(cue.ParsePath("description")); desc.Exists() {
		asset.Description, _ = desc.String()
	}

	if tags := v.LookupPath(cue.ParsePath("tags")); tags.Exists() {
		iter, err := tags.List()
		if err == nil {
			for iter.Next() {
				if s, err := iter.Value().String(); err == nil {
					asset.Tags = append(asset.Tags, s)
				}
			}
		}
	}

	// For agents, extract bin
	if category == "agents" {
		if bin := v.LookupPath(cue.ParsePath("bin")); bin.Exists() {
			asset.Bin, _ = bin.String()
		}
		// Also check under "agent" field
		if bin := v.LookupPath(cue.ParsePath("agent.bin")); bin.Exists() {
			asset.Bin, _ = bin.String()
		}
	}

	return asset, nil
}

// readModulePath reads the module path from a module.cue file.
func readModulePath(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	// Simple extraction - look for module: "path"
	content := string(data)
	if idx := strings.Index(content, "module:"); idx != -1 {
		rest := content[idx+7:]
		// Find quoted string
		start := strings.Index(rest, "\"")
		if start != -1 {
			end := strings.Index(rest[start+1:], "\"")
			if end != -1 {
				return rest[start+1 : start+1+end]
			}
		}
	}

	return ""
}

// generateIndexCUE generates the index.cue content.
func generateIndexCUE(index *ScannedIndex) string {
	var sb strings.Builder

	sb.WriteString("// Auto-generated by 'start assets index'\n")
	sb.WriteString("// Do not edit manually\n\n")
	sb.WriteString("package index\n\n")

	// Write each category
	writeIndexCategory(&sb, "agents", index.Agents, true)
	writeIndexCategory(&sb, "roles", index.Roles, false)
	writeIndexCategory(&sb, "tasks", index.Tasks, false)
	writeIndexCategory(&sb, "contexts", index.Contexts, false)

	return sb.String()
}

// writeIndexCategory writes a category to the index.
func writeIndexCategory(sb *strings.Builder, name string, assets []ScannedAsset, includeBin bool) {
	sb.WriteString(fmt.Sprintf("%s: {\n", name))

	for _, asset := range assets {
		// Use quoted key for paths with slashes
		key := asset.Name
		if strings.Contains(key, "/") {
			key = fmt.Sprintf("%q", key)
		}

		sb.WriteString(fmt.Sprintf("\t%s: {\n", key))
		sb.WriteString(fmt.Sprintf("\t\tmodule: %q\n", asset.Module))

		if asset.Description != "" {
			sb.WriteString(fmt.Sprintf("\t\tdescription: %q\n", asset.Description))
		}

		if len(asset.Tags) > 0 {
			sb.WriteString("\t\ttags: [")
			for i, tag := range asset.Tags {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%q", tag))
			}
			sb.WriteString("]\n")
		}

		if includeBin && asset.Bin != "" {
			sb.WriteString(fmt.Sprintf("\t\tbin: %q\n", asset.Bin))
		}

		sb.WriteString("\t}\n")
	}

	sb.WriteString("}\n\n")
}

// printIndexSummary prints a summary of the generated index.
func printIndexSummary(w io.Writer, index *ScannedIndex, path string) {
	total := len(index.Agents) + len(index.Roles) + len(index.Tasks) + len(index.Contexts)

	fmt.Fprintf(w, "Generated index: %s\n\n", path)
	fmt.Fprintf(w, "Assets indexed: %d\n", total)
	fmt.Fprintf(w, "  Agents:   %d\n", len(index.Agents))
	fmt.Fprintf(w, "  Roles:    %d\n", len(index.Roles))
	fmt.Fprintf(w, "  Tasks:    %d\n", len(index.Tasks))
	fmt.Fprintf(w, "  Contexts: %d\n", len(index.Contexts))
}
