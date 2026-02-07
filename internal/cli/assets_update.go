package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// UpdateResult tracks the result of an update operation.
type UpdateResult struct {
	Asset      InstalledAsset
	OldVersion string
	NewVersion string
	Updated    bool
	Error      error
}

// addAssetsUpdateCommand adds the update subcommand to the assets command.
func addAssetsUpdateCommand(parent *cobra.Command) {
	updateCmd := &cobra.Command{
		Use:   "update [query]",
		Short: "Update installed assets",
		Long: `Update installed assets to their latest versions.

Without arguments, updates all installed assets.
With a query, updates only matching assets.

CUE's major version (@v0) automatically resolves to the latest compatible version
when modules are re-fetched.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runAssetsUpdate,
	}

	updateCmd.Flags().Bool("dry-run", false, "Preview without applying")
	updateCmd.Flags().Bool("force", false, "Re-fetch even if current")

	parent.AddCommand(updateCmd)
}

// runAssetsUpdate updates installed assets.
func runAssetsUpdate(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	ctx := context.Background()

	// Load configuration
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	if !paths.AnyExists() {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No configuration found. Run 'start' to set up.")
		return nil
	}

	// Load merged config
	dirs := paths.ForScope(config.ScopeMerged)
	loader := internalcue.NewLoader()
	cfg, err := loader.Load(dirs)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// Collect installed assets
	installed := collectInstalledAssets(cfg.Value, paths)

	if len(installed) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No assets installed from registry.")
		return nil
	}

	// Filter by query if provided
	if query != "" {
		var filtered []InstalledAsset
		queryLower := strings.ToLower(query)
		for _, a := range installed {
			if strings.Contains(strings.ToLower(a.Name), queryLower) ||
				strings.Contains(strings.ToLower(a.Category), queryLower) {
				filtered = append(filtered, a)
			}
		}
		installed = filtered

		if len(installed) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No installed assets matching %q\n", query)
			return nil
		}
	}

	// Create registry client
	client, err := registry.NewClient()
	if err != nil {
		return fmt.Errorf("creating registry client: %w", err)
	}

	// Fetch index for version comparison
	flags := getFlags(cmd)
	if !flags.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Checking for updates...")
	}

	index, err := client.FetchIndex(ctx)
	if err != nil {
		return fmt.Errorf("fetching index: %w", err)
	}

	// Check each asset for updates
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	var results []UpdateResult
	for _, asset := range installed {
		result := checkAndUpdate(ctx, client, paths, index, asset, dryRun, force)
		results = append(results, result)
	}

	// Print results
	printUpdateResults(cmd.OutOrStdout(), results, dryRun)

	return nil
}

// checkAndUpdate checks for updates and optionally applies them.
func checkAndUpdate(ctx context.Context, client *registry.Client, paths config.Paths, index *registry.Index, asset InstalledAsset, dryRun, force bool) UpdateResult {
	result := UpdateResult{Asset: asset}

	// Find asset in index
	entry := findInIndex(index, asset.Category, asset.Name)
	if entry == nil {
		// Not in index, can't update
		return result
	}

	// Get current and latest versions
	result.OldVersion = asset.InstalledVer
	result.NewVersion = entry.Version

	// Check if update is needed
	needsUpdate := force || (entry.Version != "" && entry.Version != asset.InstalledVer)

	if !needsUpdate && !force {
		return result
	}

	if dryRun {
		result.Updated = true
		return result
	}

	// Re-fetch the module to get latest version
	// First resolve @v0 to canonical version (e.g., @v0.0.1)
	modulePath := entry.Module
	if !strings.Contains(modulePath, "@") {
		modulePath += "@v0"
	}

	// Resolve to canonical version before fetching
	resolvedPath, err := client.ResolveLatestVersion(ctx, modulePath)
	if err != nil {
		result.Error = err
		return result
	}

	fetchResult, err := client.Fetch(ctx, resolvedPath)
	if err != nil {
		result.Error = err
		return result
	}

<<<<<<< Updated upstream
	// Extract the new content from fetched module
	searchResult := assets.SearchResult{
=======
	// Build SearchResult for extractAssetContent
	searchResult := SearchResult{
>>>>>>> Stashed changes
		Category: asset.Category,
		Name:     asset.Name,
		Entry:    *entry,
	}

<<<<<<< Updated upstream
	// Use resolved path with version for origin field (e.g., "github.com/.../task@v0.1.1")
	// This preserves full provenance information for future updates.
	assetContent, err := extractAssetContent(fetchResult.SourceDir, searchResult, client.Registry(), resolvedPath)
=======
	// Strip version from modulePath for origin field
	originPath := modulePath
	if idx := strings.Index(originPath, "@"); idx != -1 {
		originPath = originPath[:idx]
	}

	// Extract the asset content from the fetched module
	assetContent, err := extractAssetContent(fetchResult.SourceDir, searchResult, client.Registry(), originPath)
>>>>>>> Stashed changes
	if err != nil {
		result.Error = fmt.Errorf("extracting asset content: %w", err)
		return result
	}

<<<<<<< Updated upstream
	// Update the config file with new content
	// ConfigFile should always be set by collectInstalledAssets, but check defensively.
	if asset.ConfigFile == "" {
		result.Error = fmt.Errorf("no config file path for asset")
		return result
	}

	// Replace the asset definition in-place, preserving file structure and other assets.
	// Note: This validates that the asset exists before attempting update.
	if err := assets.UpdateAssetInConfig(asset.ConfigFile, asset.Category, asset.Name, assetContent); err != nil {
		result.Error = fmt.Errorf("updating config: %w", err)
=======
	// Determine config directory based on asset scope
	var configDir string
	if asset.Scope == "local" {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

	// Determine the config file and replace the asset entry
	configFile := assetTypeToConfigFile(asset.Category)
	configPath := filepath.Join(configDir, configFile)

	if err := replaceAssetInConfig(configPath, asset.Name, assetContent); err != nil {
		result.Error = fmt.Errorf("writing updated config: %w", err)
>>>>>>> Stashed changes
		return result
	}

	result.Updated = true
	return result
}

// replaceAssetInConfig replaces an existing asset entry in a config file.
// It finds the asset key and its brace-delimited block, then replaces it with new content.
func replaceAssetInConfig(configPath, assetName, newContent string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	content := string(data)
	assetKey := getAssetKey(assetName)

	// Find the asset key - try quoted form first, then unquoted
	keyPatterns := []string{
		fmt.Sprintf("%q:", assetKey),
		assetKey + ":",
	}

	keyStart := -1
	keyLen := 0
	for _, pattern := range keyPatterns {
		idx := strings.Index(content, pattern)
		if idx != -1 {
			keyStart = idx
			keyLen = len(pattern)
			break
		}
	}

	if keyStart == -1 {
		return fmt.Errorf("asset %q not found in %s", assetKey, configPath)
	}

	// Find the opening brace after the key
	afterKey := content[keyStart+keyLen:]
	braceOffset := strings.Index(afterKey, "{")
	if braceOffset == -1 {
		return fmt.Errorf("no opening brace found for asset %q", assetKey)
	}

	// Count brace depth to find the matching close brace
	blockStart := keyStart + keyLen + braceOffset
	depth := 0
	blockEnd := -1
	for i := blockStart; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				blockEnd = i + 1
				break
			}
		}
		if blockEnd != -1 {
			break
		}
	}

	if blockEnd == -1 {
		return fmt.Errorf("no matching close brace found for asset %q", assetKey)
	}

	// Replace the key + block with new key + content
	var sb strings.Builder
	sb.WriteString(content[:keyStart])
	sb.WriteString(fmt.Sprintf("%q: ", assetKey))

	// Indent the content to match the existing indentation
	lines := strings.Split(newContent, "\n")
	for i, line := range lines {
		if i == 0 {
			sb.WriteString(line)
		} else {
			if line != "" {
				sb.WriteString("\n\t" + line)
			} else {
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString(content[blockEnd:])

	return os.WriteFile(configPath, []byte(sb.String()), 0644)
}

// printUpdateResults prints the results of the update operation.
func printUpdateResults(w io.Writer, results []UpdateResult, dryRun bool) {
	if dryRun {
		_, _ = fmt.Fprintln(w, "\nDry run - no changes applied:")
	} else {
		_, _ = fmt.Fprintln(w)
	}

	var updated, current, failed int

	for _, r := range results {
		prefix := "  "
		if r.Error != nil {
			_, _ = fmt.Fprintf(w, "%sFailed %s/%s: %v\n", prefix, r.Asset.Category, r.Asset.Name, r.Error)
			failed++
		} else if r.Updated {
			if r.OldVersion != "" && r.NewVersion != "" {
				_, _ = fmt.Fprintf(w, "%sUpdated %s/%-20s %s -> %s\n", prefix, r.Asset.Category, r.Asset.Name, r.OldVersion, r.NewVersion)
			} else if r.NewVersion != "" {
				_, _ = fmt.Fprintf(w, "%sUpdated %s/%-20s -> %s\n", prefix, r.Asset.Category, r.Asset.Name, r.NewVersion)
			} else {
				_, _ = fmt.Fprintf(w, "%sUpdated %s/%s\n", prefix, r.Asset.Category, r.Asset.Name)
			}
			updated++
		} else {
			_, _ = fmt.Fprintf(w, "%sCurrent %s/%s\n", prefix, r.Asset.Category, r.Asset.Name)
			current++
		}
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Updated: %d, Current: %d", updated, current)
	if failed > 0 {
		_, _ = fmt.Fprintf(w, ", Failed: %d", failed)
	}
	_, _ = fmt.Fprintln(w)
}

// extractAssetContent loads the asset module and extracts its content as CUE.
// This is a copy of the function from the assets package, kept here for update-specific needs.
func extractAssetContent(moduleDir string, asset assets.SearchResult, reg interface{}, originPath string) (string, error) {
	cctx := cuecontext.New()

	cfg := &load.Config{
		Dir: moduleDir,
	}

	// Add registry if provided
	if regVal, ok := reg.(modconfig.Registry); ok {
		cfg.Registry = regVal
	}

	insts := load.Instances([]string{"."}, cfg)
	if len(insts) == 0 {
		return "", fmt.Errorf("no CUE instances found in %s", moduleDir)
	}

	inst := insts[0]
	if inst.Err != nil {
		return "", fmt.Errorf("loading module: %w", inst.Err)
	}

	v := cctx.BuildInstance(inst)
	if err := v.Err(); err != nil {
		return "", fmt.Errorf("building module: %w", err)
	}

	// Extract the asset definition - try singular field name first
	singular := strings.TrimSuffix(asset.Category, "s")
	assetVal := v.LookupPath(cue.ParsePath(singular))
	if !assetVal.Exists() {
		// Try the key name
		assetKey := asset.Name
		assetVal = v.LookupPath(cue.MakePath(cue.Str(assetKey)))
	}
	if !assetVal.Exists() {
		return "", fmt.Errorf("asset definition not found in module (tried %q)", singular)
	}

	// Format as concrete struct
	return formatAssetStruct(v, assetVal, asset.Category, originPath)
}

// formatAssetStruct formats a CUE value as a concrete struct.
func formatAssetStruct(ctx cue.Value, v cue.Value, category, originPath string) (string, error) {
	var sb strings.Builder
	sb.WriteString("{\n")

	// Write origin field first
	sb.WriteString(fmt.Sprintf("\torigin: %q\n", originPath))

	// Define which fields to extract based on category
	var fields []string
	switch category {
	case "tasks":
		fields = []string{"description", "tags", "role", "file", "command", "prompt"}
	case "roles":
		fields = []string{"description", "tags", "file", "command", "prompt", "optional"}
	case "agents":
		fields = []string{"description", "tags", "bin", "command", "default_model", "models"}
	case "contexts":
		fields = []string{"description", "tags", "file", "command", "prompt", "required", "default"}
	default:
		fields = []string{"description", "tags", "prompt"}
	}

	for _, field := range fields {
		fieldVal := v.LookupPath(cue.ParsePath(field))
		if !fieldVal.Exists() {
			continue
		}

		// Format the field value
		formatted, err := formatFieldValue(field, fieldVal)
		if err != nil {
			continue // Skip fields that can't be formatted
		}
		sb.WriteString(formatted)
	}

	sb.WriteString("}")
	return sb.String(), nil
}

// formatFieldValue formats a single field value as CUE syntax.
func formatFieldValue(name string, v cue.Value) (string, error) {
	var sb strings.Builder

	switch v.Kind() {
	case cue.StringKind:
		s, err := v.String()
		if err != nil {
			return "", err
		}
		// Use multi-line string for prompts and long strings
		if strings.Contains(s, "\n") || len(s) > 80 {
			sb.WriteString(fmt.Sprintf("\t%s: \"\"\"\n", name))
			for _, line := range strings.Split(s, "\n") {
				sb.WriteString(fmt.Sprintf("\t\t%s\n", line))
			}
			sb.WriteString("\t\t\"\"\"\n")
		} else {
			sb.WriteString(fmt.Sprintf("\t%s: %q\n", name, s))
		}

	case cue.BoolKind:
		b, err := v.Bool()
		if err != nil {
			return "", err
		}
		sb.WriteString(fmt.Sprintf("\t%s: %t\n", name, b))

	case cue.ListKind:
		iter, err := v.List()
		if err != nil {
			return "", err
		}
		var items []string
		for iter.Next() {
			if s, err := iter.Value().String(); err == nil {
				items = append(items, fmt.Sprintf("%q", s))
			}
		}
		if len(items) > 0 {
			sb.WriteString(fmt.Sprintf("\t%s: [%s]\n", name, strings.Join(items, ", ")))
		}

	case cue.StructKind:
		// For maps like "models"
		sb.WriteString(fmt.Sprintf("\t%s: {\n", name))
		iter, err := v.Fields()
		if err != nil {
			return "", err
		}
		for iter.Next() {
			key := iter.Selector().Unquoted()
			if s, err := iter.Value().String(); err == nil {
				sb.WriteString(fmt.Sprintf("\t\t%q: %q\n", key, s))
			}
		}
		sb.WriteString("\t}\n")

	default:
		// Try to get string representation
		syn := v.Syntax()
		formatted, err := format.Node(syn, format.Simplify())
		if err != nil {
			return "", err
		}
		sb.WriteString(fmt.Sprintf("\t%s: %s\n", name, string(formatted)))
	}

	return sb.String(), nil
}
