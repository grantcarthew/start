package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	cueformat "cuelang.org/go/cue/format"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/tui"
	"github.com/spf13/cobra"
)

// ShowResult holds the result of preparing show output.
type ShowResult struct {
	ItemType   string    // "Agent", "Role", "Context", "Task"
	Category   string    // "agents", "roles", "contexts", "tasks"
	CueKey     string    // Top-level CUE key (e.g., "agents")
	Name       string    // Item name (when showing specific item)
	Value      cue.Value // The CUE value for this item
	AllNames   []string  // All available items of this type
	ShowReason string    // Why this item is shown (e.g., "first in config", "default")
}

// showCategory maps category metadata used for cross-category operations.
type showCategory struct {
	key      string // CUE key (e.g., "agents")
	category string // Category name (e.g., "agents")
	itemType string // Display type (e.g., "Agent")
}

var showCategories = []showCategory{
	{internalcue.KeyAgents, "agents", "Agent"},
	{internalcue.KeyRoles, "roles", "Role"},
	{internalcue.KeyContexts, "contexts", "Context"},
	{internalcue.KeyTasks, "tasks", "Task"},
}

// showCategoryFor looks up a showCategory by its category string.
// Returns nil only if category is not in showCategories. All callers pass
// Category values that originate from iterating showCategories, so nil is
// unreachable in practice. If a new AssetMatch source is added, ensure its
// Category is drawn from showCategories.
func showCategoryFor(category string) *showCategory {
	for i := range showCategories {
		if showCategories[i].category == category {
			return &showCategories[i]
		}
	}
	return nil
}

// addShowCommand adds the show command and its subcommands to the parent command.
func addShowCommand(parent *cobra.Command) {
	showCmd := &cobra.Command{
		Use:     "show [name]",
		Aliases: []string{"view"},
		GroupID: "commands",
		Short:   "Display resolved configuration content",
		Long: `Display resolved configuration content after UTD processing and config merging.

Without arguments, lists all configured items with descriptions.
With an argument, searches across all categories and displays a verbose dump.

Use --global to restrict output to the global config (~/.config/start/) or
--local to restrict to the local config (./.start/). These flags are mutually
exclusive; omitting both shows the effective merged configuration.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runShow,
	}

	showRoleCmd := &cobra.Command{
		Use:     "role [name]",
		Aliases: []string{"roles"},
		Short:   "Display resolved role content",
		Long:    `Display resolved role content after UTD processing.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowItem(internalcue.KeyRoles, "Role"),
	}

	showContextCmd := &cobra.Command{
		Use:     "context [name]",
		Aliases: []string{"contexts"},
		Short:   "Display resolved context content",
		Long:    `Display resolved context content after UTD processing.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowItem(internalcue.KeyContexts, "Context"),
	}

	showAgentCmd := &cobra.Command{
		Use:     "agent [name]",
		Aliases: []string{"agents"},
		Short:   "Display agent configuration",
		Long:    `Display effective agent configuration after config merging.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowItem(internalcue.KeyAgents, "Agent"),
	}

	showTaskCmd := &cobra.Command{
		Use:     "task [name]",
		Aliases: []string{"tasks"},
		Short:   "Display task template",
		Long:    `Display resolved task prompt template.`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runShowItem(internalcue.KeyTasks, "Task"),
	}

	// Add --global flag to show command (show-specific scope restriction)
	showCmd.PersistentFlags().Bool("global", false, "Show from global scope only")

	// Add subcommands
	showCmd.AddCommand(showRoleCmd)
	showCmd.AddCommand(showContextCmd)
	showCmd.AddCommand(showAgentCmd)
	showCmd.AddCommand(showTaskCmd)

	// Add show to parent
	parent.AddCommand(showCmd)
}

// runShow displays all configuration or searches for a specific item.
func runShow(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}

	if len(args) == 0 {
		return runShowListing(cmd)
	}

	return runShowSearch(cmd, args[0])
}

// runShowListing displays all items grouped by category with descriptions.
func runShowListing(cmd *cobra.Command) error {
	w := cmd.OutOrStdout()

	scope, err := showScopeFromCmd(cmd)
	if err != nil {
		return err
	}
	cfg, err := loadConfig(scope)
	if err != nil {
		return err
	}

	for _, cat := range showCategories {
		items := cfg.Value.LookupPath(cue.ParsePath(cat.key))
		if !items.Exists() {
			continue
		}

		type entry struct {
			name string
			desc string
		}

		var entries []entry
		maxNameLen := 0

		iter, err := items.Fields()
		if err != nil {
			continue
		}
		for iter.Next() {
			name := iter.Selector().Unquoted()
			desc := ""
			if d := iter.Value().LookupPath(cue.ParsePath("description")); d.Exists() {
				desc, _ = d.String()
			}
			entries = append(entries, entry{name, desc})
			if len(name) > maxNameLen {
				maxNameLen = len(name)
			}
		}

		if len(entries) == 0 {
			continue
		}

		_, _ = tui.CategoryColor(cat.category).Fprint(w, cat.category)
		_, _ = fmt.Fprintln(w, "/")

		for _, e := range entries {
			if e.desc != "" {
				padding := strings.Repeat(" ", maxNameLen-len(e.name)+2)
				_, _ = fmt.Fprintf(w, "  %s%s", e.name, padding)
				_, _ = tui.ColorDim.Fprintln(w, e.desc)
			} else {
				_, _ = fmt.Fprintf(w, "  %s\n", e.name)
			}
		}

		_, _ = fmt.Fprintln(w)
	}

	return nil
}

// runShowSearch handles cross-category search for `start show <name>`.
func runShowSearch(cmd *cobra.Command, name string) error {
	w := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	flags := getFlags(cmd)
	scope, err := showScopeFromCmd(cmd)
	if err != nil {
		return err
	}
	stdin := cmd.InOrStdin()

	cfg, err := loadConfig(scope)
	if err != nil {
		return err
	}

	// Step 1: Exact match in installed config across all categories
	var exactMatches []AssetMatch
	for _, cat := range showCategories {
		resolved, err := findExactInstalledName(cfg.Value, cat.key, name)
		if err != nil {
			return err
		}
		if resolved != "" {
			exactMatches = append(exactMatches, AssetMatch{
				Name:     resolved,
				Category: cat.category,
				Source:   AssetSourceInstalled,
				Score:    100,
			})
		}
	}

	if len(exactMatches) == 1 {
		// Before returning the single exact match, check if a substring search
		// finds additional matches. If so, fall through to show a selection list
		// rather than silently picking the exact match. See task.go for the same
		// pattern applied to task resolution.
		var moreMatches bool
		for _, cat := range showCategories {
			matches, err := searchInstalled(cfg.Value, cat.key, cat.category, name)
			if err != nil {
				continue
			}
			if len(matches) > 1 {
				moreMatches = true
				break
			}
		}
		if !moreMatches {
			cat := showCategoryFor(exactMatches[0].Category)
			return showVerboseItem(w, exactMatches[0].Name, scope, cat.key, cat.itemType)
		}
	}
	if len(exactMatches) > 1 {
		return promptShowSelection(w, stdin, scope, exactMatches, name, nil)
	}

	// Step 2: Substring search in installed config
	var installedMatches []AssetMatch
	for _, cat := range showCategories {
		matches, err := searchInstalled(cfg.Value, cat.key, cat.category, name)
		if err != nil {
			continue
		}
		installedMatches = append(installedMatches, matches...)
	}

	if len(installedMatches) == 1 {
		cat := showCategoryFor(installedMatches[0].Category)
		return showVerboseItem(w, installedMatches[0].Name, scope, cat.key, cat.itemType)
	}

	// Step 3: Registry search
	r := newResolver(cfg, flags, w, stderr, stdin)
	index, _, _ := r.ensureIndex()

	// Exact registry match (only when no installed matches)
	if len(installedMatches) == 0 && index != nil {
		for _, cat := range showCategories {
			entries := registryEntries(index, cat.category)
			if entries == nil {
				continue
			}
			result, err := findExactInRegistry(entries, cat.category, name)
			if err != nil {
				return err
			}
			if result != nil {
				if err := r.autoInstall(r.client, *result); err != nil {
					return err
				}
				// After install, use merged scope to see the new item
				return showVerboseItem(w, result.Name, config.ScopeMerged, cat.key, cat.itemType)
			}
		}
	}

	// Combined search across installed + registry
	var registryMatches []AssetMatch
	if index != nil {
		for _, cat := range showCategories {
			entries := registryEntries(index, cat.category)
			if entries == nil {
				continue
			}
			regMatches, err := searchRegistryCategory(entries, cat.category, name)
			if err != nil {
				continue
			}
			registryMatches = append(registryMatches, regMatches...)
		}
	}

	allMatches := mergeAssetMatches(installedMatches, registryMatches)

	switch len(allMatches) {
	case 0:
		return fmt.Errorf("%q not found", name)
	case 1:
		return displayShowMatch(w, scope, allMatches[0], r)
	default:
		return promptShowSelection(w, stdin, scope, allMatches, name, r)
	}
}

// displayShowMatch handles auto-install (if needed) and displays a verbose dump.
func displayShowMatch(w io.Writer, scope config.Scope, m AssetMatch, r *resolver) error {
	cat := showCategoryFor(m.Category)
	if m.Source == AssetSourceRegistry && r != nil {
		if err := r.autoInstall(r.client, assets.SearchResult{
			Category: m.Category,
			Name:     m.Name,
			Entry:    m.Entry,
		}); err != nil {
			return err
		}
		// After install, use merged scope to see the new item
		return showVerboseItem(w, m.Name, config.ScopeMerged, cat.key, cat.itemType)
	}
	return showVerboseItem(w, m.Name, scope, cat.key, cat.itemType)
}

// promptShowSelection handles interactive selection from multiple cross-category matches.
func promptShowSelection(w io.Writer, stdin io.Reader, scope config.Scope, matches []AssetMatch, query string, r *resolver) error {
	isTTY := isTerminal(stdin)

	if !isTTY {
		var names []string
		for _, m := range matches {
			names = append(names, m.Category+"/"+m.Name)
		}
		return fmt.Errorf("ambiguous name %q matches: %s\nSpecify exact name or run interactively",
			query, strings.Join(names, ", "))
	}

	displayCount := len(matches)
	if displayCount > maxAssetResults {
		displayCount = maxAssetResults
	}

	_, _ = fmt.Fprintf(w, "Found %d matches for %q:\n\n", len(matches), query)

	// Find longest display name for alignment
	maxDisplayLen := 0
	for i := 0; i < displayCount; i++ {
		display := matches[i].Category + "/" + matches[i].Name
		if len(display) > maxDisplayLen {
			maxDisplayLen = len(display)
		}
	}

	for i := 0; i < displayCount; i++ {
		m := matches[i]
		display := m.Category + "/" + m.Name
		padding := strings.Repeat(" ", maxDisplayLen-len(display)+2)
		var sourceLabel string
		if m.Source == AssetSourceInstalled {
			sourceLabel = tui.ColorInstalled.Sprint(m.Source)
		} else {
			sourceLabel = tui.ColorRegistry.Sprint(m.Source)
		}
		_, _ = fmt.Fprintf(w, "  %2d. %s%s%s\n", i+1, display, padding, sourceLabel)
	}

	if displayCount < len(matches) {
		_, _ = fmt.Fprintf(w, "\nShowing %d of %d matches. Refine search for more specific results.\n",
			displayCount, len(matches))
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Select %s: ", tui.Annotate("1-%d", displayCount))

	reader := bufio.NewReader(stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(input)

	// Try number
	if choice, err := strconv.Atoi(input); err == nil {
		if choice >= 1 && choice <= displayCount {
			return displayShowMatch(w, scope, matches[choice-1], r)
		}
		return fmt.Errorf("invalid selection: %s (choose 1-%d)", input, displayCount)
	}

	return fmt.Errorf("invalid selection: %s", input)
}

// showVerboseItem prepares and displays a verbose dump for a single item.
func showVerboseItem(w io.Writer, name string, scope config.Scope, cueKey, itemType string) error {
	result, err := prepareShow(name, scope, cueKey, itemType)
	if err != nil {
		return err
	}
	printVerboseDump(w, result)
	return nil
}

// runShowItem returns a cobra RunE handler that displays a specific item type.
func runShowItem(cueKey, itemType string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}

		scope, err := showScopeFromCmd(cmd)
		if err != nil {
			return err
		}
		result, err := prepareShow(name, scope, cueKey, itemType)
		if err != nil {
			return err
		}

		printVerboseDump(cmd.OutOrStdout(), result)
		return nil
	}
}

// prepareShow prepares show output for an item type.
// cueKey is the top-level CUE key (e.g., internalcue.KeyRoles).
// itemType is the display name (e.g., "Role").
func prepareShow(name string, scope config.Scope, cueKey, itemType string) (ShowResult, error) {
	cfg, err := loadConfig(scope)
	if err != nil {
		return ShowResult{}, err
	}

	typePlural := strings.ToLower(itemType) + "s"

	items := cfg.Value.LookupPath(cue.ParsePath(cueKey))
	if !items.Exists() {
		return ShowResult{}, fmt.Errorf("no %s defined in configuration", typePlural)
	}

	// Collect all names in config order
	var allNames []string
	iter, err := items.Fields()
	if err != nil {
		return ShowResult{}, fmt.Errorf("reading %s: %w", typePlural, err)
	}
	for iter.Next() {
		allNames = append(allNames, iter.Selector().Unquoted())
	}
	if len(allNames) == 0 {
		return ShowResult{}, fmt.Errorf("no %s defined in configuration", typePlural)
	}

	// Determine which item to show and why
	showReason := ""
	if name == "" {
		name = allNames[0]
		showReason = "first in config"
	}

	resolvedName := name
	item := items.LookupPath(cue.MakePath(cue.Str(name)))
	if !item.Exists() {
		// Try substring match
		var matches []string
		for _, n := range allNames {
			if strings.Contains(n, name) {
				matches = append(matches, n)
			}
		}

		switch len(matches) {
		case 0:
			return ShowResult{}, fmt.Errorf("%s %q not found", strings.ToLower(itemType), name)
		case 1:
			resolvedName = matches[0]
			item = items.LookupPath(cue.MakePath(cue.Str(resolvedName)))
		default:
			return ShowResult{}, fmt.Errorf("ambiguous %s name %q matches: %s", strings.ToLower(itemType), name, strings.Join(matches, ", "))
		}
	}

	return ShowResult{
		ItemType:   itemType,
		Category:   typePlural,
		CueKey:     cueKey,
		Name:       resolvedName,
		Value:      item,
		AllNames:   allNames,
		ShowReason: showReason,
	}, nil
}

// showScopeFromCmd derives the config scope from show command flags.
// Returns an error if --local and --global are both set.
func showScopeFromCmd(cmd *cobra.Command) (config.Scope, error) {
	var global bool
	if f := cmd.Flags().Lookup("global"); f != nil {
		global, _ = cmd.Flags().GetBool("global")
	}
	local := getFlags(cmd).Local
	if local && global {
		return config.ScopeMerged, fmt.Errorf("--local and --global are mutually exclusive")
	}
	if global {
		return config.ScopeGlobal, nil
	}
	if local {
		return config.ScopeLocal, nil
	}
	return config.ScopeMerged, nil
}

// loadConfig loads CUE configuration for the given scope.
func loadConfig(scope config.Scope) (internalcue.LoadResult, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return internalcue.LoadResult{}, fmt.Errorf("resolving config paths: %w", err)
	}

	dirs := paths.ForScope(scope)

	if len(dirs) == 0 {
		switch scope {
		case config.ScopeGlobal:
			return internalcue.LoadResult{}, fmt.Errorf("no global configuration found at %s", paths.Global)
		case config.ScopeLocal:
			return internalcue.LoadResult{}, fmt.Errorf("no local configuration found at %s", paths.Local)
		default:
			return internalcue.LoadResult{}, fmt.Errorf("no configuration found (checked %s and %s)", paths.Global, paths.Local)
		}
	}

	loader := internalcue.NewLoader()
	return loader.Load(dirs)
}

// printVerboseDump writes the full verbose dump for a ShowResult.
func printVerboseDump(w io.Writer, r ShowResult) {
	cat := r.Category
	label := tui.ColorDim.Sprint

	// Header
	_, _ = fmt.Fprintln(w)
	_, _ = tui.CategoryColor(cat).Fprint(w, r.ItemType)
	_, _ = fmt.Fprintf(w, ": %s", r.Name)
	if r.ShowReason != "" {
		_, _ = fmt.Fprint(w, " ")
		_, _ = fmt.Fprint(w, tui.Annotate("%s", r.ShowReason))
	}
	_, _ = fmt.Fprintln(w)
	printSeparator(w)

	// Config source
	configSource := findConfigSource(r.CueKey, r.Name)
	if configSource != "" {
		_, _ = fmt.Fprintf(w, "%s %s %s\n",
			label("Config:"), configSource,
			tui.Annotate("%s", r.Name))
	}

	// Origin and cache
	origin := orchestration.ExtractOrigin(r.Value)
	if origin != "" {
		_, _ = fmt.Fprintf(w, "%s %s\n", label("Origin:"), origin)
		cacheDir := deriveCacheDir(origin)
		if cacheDir != "" {
			_, _ = fmt.Fprintf(w, "%s %s\n", label("Cache:"), cacheDir)
		}
	}

	// CUE Definition
	cueDef := formatCUEDefinition(r.Value)
	if cueDef != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, cueDef)
	}

	// File contents
	fields := orchestration.ExtractUTDFields(r.Value)
	if fields.File != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "%s %s\n", label("File:"), fields.File)

		resolvedPath, content, readErr := resolveShowFile(fields.File, origin)
		if resolvedPath != "" && resolvedPath != fields.File {
			_, _ = fmt.Fprintf(w, "%s %s\n", label("Path:"), resolvedPath)
		}

		if readErr != nil {
			_, _ = fmt.Fprintf(w, "[error: %s]\n", readErr)
		} else if content != "" {
			_, _ = fmt.Fprintln(w)
			_, _ = fmt.Fprint(w, content)
			if !strings.HasSuffix(content, "\n") {
				_, _ = fmt.Fprintln(w)
			}
		}
	}

	// Command
	if fields.Command != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "%s %s\n", label("Command:"), fields.Command)
	}

	printSeparator(w)
}

// findConfigSource determines which config file defines an item.
// This loads each config dir separately via LoadSingle rather than reusing the
// merged config because merged CUE values lose per-file position information.
func findConfigSource(cueKey, name string) string {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return ""
	}

	loader := internalcue.NewLoader()

	// Check local first (higher priority - overrides global)
	if paths.LocalExists {
		if v, err := loader.LoadSingle(paths.Local); err == nil {
			item := v.LookupPath(cue.ParsePath(cueKey)).LookupPath(cue.MakePath(cue.Str(name)))
			if item.Exists() {
				if pos := item.Pos(); pos.IsValid() {
					return pos.Filename()
				}
			}
		}
	}

	// Check global
	if paths.GlobalExists {
		if v, err := loader.LoadSingle(paths.Global); err == nil {
			item := v.LookupPath(cue.ParsePath(cueKey)).LookupPath(cue.MakePath(cue.Str(name)))
			if item.Exists() {
				if pos := item.Pos(); pos.IsValid() {
					return pos.Filename()
				}
			}
		}
	}

	return ""
}

// formatCUEDefinition formats a CUE value as CUE syntax.
func formatCUEDefinition(v cue.Value) string {
	syn := v.Syntax(
		cue.Concrete(false),
		cue.Definitions(true),
		cue.Hidden(true),
		cue.Optional(true),
	)

	b, err := cueformat.Node(syn)
	if err != nil {
		return ""
	}
	return string(b)
}

// resolveShowFile resolves a file reference and reads its contents.
// Returns the resolved path, content, and any error.
func resolveShowFile(filePath, origin string) (resolvedPath, content string, err error) {
	if filePath == "" {
		return "", "", nil
	}

	// @module/ paths
	if strings.HasPrefix(filePath, "@module/") {
		if origin == "" {
			return "", "", fmt.Errorf("@module/ path requires origin field: %s", filePath)
		}
		resolved, err := orchestration.ResolveModulePath(filePath, origin)
		if err != nil {
			return "", "", err
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return resolved, "", err
		}
		return resolved, string(data), nil
	}

	// ~/ and other paths
	expanded, err := orchestration.ExpandFilePath(filePath)
	if err != nil {
		return "", "", err
	}

	data, readErr := os.ReadFile(expanded)
	if readErr != nil {
		return expanded, "", readErr
	}
	return expanded, string(data), nil
}

// deriveCacheDir constructs the CUE cache directory for an origin.
func deriveCacheDir(origin string) string {
	cacheDir, err := orchestration.GetCUECacheDir()
	if err != nil {
		return ""
	}

	idx := strings.LastIndex(origin, "@")
	if idx == -1 {
		return ""
	}

	modulePath := origin[:idx]
	version := origin[idx:]
	return filepath.Join(cacheDir, "mod", "extract",
		filepath.Dir(modulePath),
		filepath.Base(modulePath)+version)
}
