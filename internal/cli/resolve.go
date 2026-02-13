package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/registry"
	"golang.org/x/term"
)

// AssetSource indicates where an asset was found.
type AssetSource string

const (
	AssetSourceInstalled AssetSource = "installed"
	AssetSourceRegistry  AssetSource = "registry"
)

// AssetMatch represents a single matched asset during resolution.
type AssetMatch struct {
	Name     string
	Category string
	Source   AssetSource
	Entry    registry.IndexEntry
	Score    int
}

// contextScoreThreshold is the minimum match score for context inclusion.
const contextScoreThreshold = 2

// maxAssetResults is the maximum number of results to display in interactive selection.
const maxAssetResults = 20

// resolver performs three-tier resolution for asset-selecting flags.
// It lazily fetches the registry index and tracks whether any installs occurred.
type resolver struct {
	cfg          internalcue.LoadResult
	flags        *Flags
	stdout       io.Writer
	stdin        io.Reader
	index        *registry.Index
	client       *registry.Client
	indexErr     error
	didFetch     bool
	didInstall   bool
	skipRegistry bool // When true, skip registry fetch (for testing)
}

// newResolver creates a resolver for the given config.
func newResolver(cfg internalcue.LoadResult, flags *Flags, stdout io.Writer, stdin io.Reader) *resolver {
	return &resolver{
		cfg:    cfg,
		flags:  flags,
		stdout: stdout,
		stdin:  stdin,
	}
}

// resolveAgent resolves an agent name through three-tier search:
// 1. Exact match in installed config
// 2. Exact match in registry index
// 3. Substring search across installed config + registry
func (r *resolver) resolveAgent(name string) (string, error) {
	if name == "" {
		return "", nil
	}

	// Tier 1: Exact match in installed config
	if hasExactInstalled(r.cfg.Value, internalcue.KeyAgents, name) {
		debugf(r.flags, "resolve", "Agent %q: exact installed match", name)
		return name, nil
	}

	// Tier 2: Exact match in registry
	if !r.flags.Quiet {
		_, _ = fmt.Fprintf(r.stdout, "Agent %q not found in configuration\n", name)
	}
	index, client, err := r.ensureIndex()
	if err == nil && index != nil {
		if result := findExactInRegistry(index.Agents, "agents", name); result != nil {
			debugf(r.flags, "resolve", "Agent %q: exact registry match %q", name, result.Name)
			if err := r.autoInstall(client, *result); err != nil {
				return "", err
			}
			return result.Name, nil
		}
	}

	// Tier 3: Regex search across installed + registry
	installedMatches, err := searchInstalled(r.cfg.Value, internalcue.KeyAgents, "agents", name)
	if err != nil {
		return "", err
	}
	var registryMatches []AssetMatch
	if index != nil {
		registryMatches, err = searchRegistryCategory(index.Agents, "agents", name)
		if err != nil {
			return "", err
		}
	}
	allMatches := mergeAssetMatches(installedMatches, registryMatches)

	debugf(r.flags, "resolve", "Agent %q: %d installed, %d registry, %d total matches",
		name, len(installedMatches), len(registryMatches), len(allMatches))

	selected, err := r.selectSingleMatch(allMatches, "agent", name)
	if err != nil {
		return "", err
	}

	if selected.Source == AssetSourceRegistry {
		if err := r.autoInstall(client, assets.SearchResult{
			Category: selected.Category,
			Name:     selected.Name,
			Entry:    selected.Entry,
		}); err != nil {
			return "", err
		}
	}

	return selected.Name, nil
}

// resolveRole resolves a role name through file path bypass then three-tier search.
func (r *resolver) resolveRole(name string) (string, error) {
	if name == "" {
		return "", nil
	}

	// File path bypass (per DR-038)
	if orchestration.IsFilePath(name) {
		debugf(r.flags, "resolve", "Role %q: file path bypass", name)
		return name, nil
	}

	// Tier 1: Exact match in installed config
	if hasExactInstalled(r.cfg.Value, internalcue.KeyRoles, name) {
		debugf(r.flags, "resolve", "Role %q: exact installed match", name)
		return name, nil
	}

	// Tier 2: Exact match in registry
	if !r.flags.Quiet {
		_, _ = fmt.Fprintf(r.stdout, "Role %q not found in configuration\n", name)
	}
	index, client, err := r.ensureIndex()
	if err == nil && index != nil {
		if result := findExactInRegistry(index.Roles, "roles", name); result != nil {
			debugf(r.flags, "resolve", "Role %q: exact registry match %q", name, result.Name)
			if err := r.autoInstall(client, *result); err != nil {
				return "", err
			}
			return result.Name, nil
		}
	}

	// Tier 3: Regex search across installed + registry
	installedMatches, err := searchInstalled(r.cfg.Value, internalcue.KeyRoles, "roles", name)
	if err != nil {
		return "", err
	}
	var registryMatches []AssetMatch
	if index != nil {
		registryMatches, err = searchRegistryCategory(index.Roles, "roles", name)
		if err != nil {
			return "", err
		}
	}
	allMatches := mergeAssetMatches(installedMatches, registryMatches)

	debugf(r.flags, "resolve", "Role %q: %d installed, %d registry, %d total matches",
		name, len(installedMatches), len(registryMatches), len(allMatches))

	selected, err := r.selectSingleMatch(allMatches, "role", name)
	if err != nil {
		return "", err
	}

	if selected.Source == AssetSourceRegistry {
		if err := r.autoInstall(client, assets.SearchResult{
			Category: selected.Category,
			Name:     selected.Name,
			Entry:    selected.Entry,
		}); err != nil {
			return "", err
		}
	}

	return selected.Name, nil
}

// resolveModelName resolves a model name against an agent's models map.
// 1. Exact match in agent.Models
// 2. Substring match in agent.Models (multi-term AND if comma/space separated)
// 3. Passthrough (value used as-is)
func (r *resolver) resolveModelName(name string, agent orchestration.Agent) string {
	if name == "" {
		return ""
	}

	// Exact match
	if _, ok := agent.Models[name]; ok {
		debugf(r.flags, "resolve", "Model %q: exact match in models map", name)
		return name
	}

	// Multi-term AND substring match
	terms := assets.ParseSearchTerms(name)
	if len(terms) == 0 {
		return name
	}

	var matches []string
	for key := range agent.Models {
		keyLower := strings.ToLower(key)
		allMatch := true
		for _, term := range terms {
			if !strings.Contains(keyLower, term) {
				allMatch = false
				break
			}
		}
		if allMatch {
			matches = append(matches, key)
		}
	}

	sort.Strings(matches)

	if len(matches) == 1 {
		debugf(r.flags, "resolve", "Model %q: match %q", name, matches[0])
		return matches[0]
	}

	if len(matches) > 1 {
		debugf(r.flags, "resolve", "Model %q: multiple matches %v, using passthrough", name, matches)
	}

	// Passthrough
	debugf(r.flags, "resolve", "Model %q: passthrough", name)
	return name
}

// resolveContexts resolves context flag values.
// Per-term: file path bypass -> "default" passthrough -> exact name -> search (all above threshold).
// Returns the resolved list of context terms for ContextSelection.Tags.
func (r *resolver) resolveContexts(terms []string) []string {
	if len(terms) == 0 {
		return nil
	}

	var resolved []string
	for _, term := range terms {
		// File path bypass
		if orchestration.IsFilePath(term) {
			debugf(r.flags, "resolve", "Context %q: file path bypass", term)
			resolved = append(resolved, term)
			continue
		}

		// "default" pseudo-tag passthrough
		if term == "default" {
			debugf(r.flags, "resolve", "Context %q: default passthrough", term)
			resolved = append(resolved, term)
			continue
		}

		// Exact name match in installed config
		if hasExactInstalled(r.cfg.Value, internalcue.KeyContexts, term) {
			debugf(r.flags, "resolve", "Context %q: exact installed match", term)
			resolved = append(resolved, term)
			continue
		}

		// Exact name match in registry
		if !r.flags.Quiet {
			_, _ = fmt.Fprintf(r.stdout, "Context %q not found in configuration\n", term)
		}
		index, client, _ := r.ensureIndex()
		if index != nil {
			if result := findExactInRegistry(index.Contexts, "contexts", term); result != nil {
				debugf(r.flags, "resolve", "Context %q: exact registry match %q", term, result.Name)
				if err := r.autoInstall(client, *result); err == nil {
					resolved = append(resolved, result.Name)
					continue
				}
			}
		}

		// Search across installed + registry (all matches above threshold)
		installedMatches, err := searchInstalled(r.cfg.Value, internalcue.KeyContexts, "contexts", term)
		if err != nil {
			// Invalid regex in context term - pass through as-is
			debugf(r.flags, "resolve", "Context %q: invalid pattern, passing through", term)
			resolved = append(resolved, term)
			continue
		}
		var registryMatches []AssetMatch
		if index != nil {
			registryMatches, err = searchRegistryCategory(index.Contexts, "contexts", term)
			if err != nil {
				debugf(r.flags, "resolve", "Context %q: invalid pattern, passing through", term)
				resolved = append(resolved, term)
				continue
			}
		}
		allMatches := mergeAssetMatches(installedMatches, registryMatches)

		// Filter by threshold
		var qualified []AssetMatch
		for _, m := range allMatches {
			if m.Score >= contextScoreThreshold {
				qualified = append(qualified, m)
			}
		}

		debugf(r.flags, "resolve", "Context %q: %d matches above threshold", term, len(qualified))

		if len(qualified) == 0 {
			// No matches - pass through as-is (composer will warn)
			debugf(r.flags, "resolve", "Context %q: no matches, passing through", term)
			resolved = append(resolved, term)
			continue
		}

		// Install any registry matches and add all to resolved
		for _, m := range qualified {
			if m.Source == AssetSourceRegistry && client != nil {
				if err := r.autoInstall(client, assets.SearchResult{
					Category: m.Category,
					Name:     m.Name,
					Entry:    m.Entry,
				}); err != nil {
					continue
				}
			}
			resolved = append(resolved, m.Name)
		}
	}

	return resolved
}

// hasExactInstalled checks if an exact name exists under a category in the config.
func hasExactInstalled(cfg cue.Value, cueKey, name string) bool {
	catVal := cfg.LookupPath(cue.ParsePath(cueKey))
	if !catVal.Exists() {
		return false
	}
	return catVal.LookupPath(cue.MakePath(cue.Str(name))).Exists()
}

// findExactInRegistry searches for an exact name match in registry entries.
// Supports both full name (e.g., "golang/assistant") and short name match.
func findExactInRegistry(entries map[string]registry.IndexEntry, category, name string) *assets.SearchResult {
	for entryName, entry := range entries {
		if entryName == name {
			return &assets.SearchResult{
				Category: category,
				Name:     entryName,
				Entry:    entry,
			}
		}
		// Short name match (e.g., "assistant" matches "golang/assistant")
		if idx := strings.LastIndex(entryName, "/"); idx != -1 {
			if entryName[idx+1:] == name {
				return &assets.SearchResult{
					Category: category,
					Name:     entryName,
					Entry:    entry,
				}
			}
		}
	}
	return nil
}

// searchInstalled searches installed config entries and returns AssetMatch results.
func searchInstalled(cfg cue.Value, cueKey, category, query string) ([]AssetMatch, error) {
	results, err := assets.SearchInstalledConfig(cfg, cueKey, category, query)
	if err != nil {
		return nil, err
	}
	var matches []AssetMatch
	for _, r := range results {
		matches = append(matches, AssetMatch{
			Name:     r.Name,
			Category: r.Category,
			Source:   AssetSourceInstalled,
			Entry:    r.Entry,
			Score:    r.MatchScore,
		})
	}
	return matches, nil
}

// searchRegistryCategory searches registry entries and returns AssetMatch results.
func searchRegistryCategory(entries map[string]registry.IndexEntry, category, query string) ([]AssetMatch, error) {
	results, err := assets.SearchCategoryEntries(category, entries, query)
	if err != nil {
		return nil, err
	}
	var matches []AssetMatch
	for _, r := range results {
		matches = append(matches, AssetMatch{
			Name:     r.Name,
			Category: r.Category,
			Source:   AssetSourceRegistry,
			Entry:    r.Entry,
			Score:    r.MatchScore,
		})
	}
	return matches, nil
}

// mergeAssetMatches combines installed and registry matches, deduplicating by name.
// Installed matches take precedence. Results are sorted by score descending, then name.
func mergeAssetMatches(installed, reg []AssetMatch) []AssetMatch {
	seen := make(map[string]bool)
	var merged []AssetMatch

	for _, m := range installed {
		seen[m.Name] = true
		merged = append(merged, m)
	}

	for _, m := range reg {
		if !seen[m.Name] {
			merged = append(merged, m)
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Score != merged[j].Score {
			return merged[i].Score > merged[j].Score
		}
		return merged[i].Name < merged[j].Name
	})

	return merged
}

// selectSingleMatch handles single-select resolution: auto-select on one match,
// prompt on multiple matches (TTY), error on multiple (non-TTY), error on zero.
func (r *resolver) selectSingleMatch(matches []AssetMatch, assetType, query string) (AssetMatch, error) {
	switch len(matches) {
	case 0:
		return AssetMatch{}, fmt.Errorf("%s %q not found", assetType, query)
	case 1:
		return matches[0], nil
	default:
		return r.promptAssetSelection(matches, assetType, query)
	}
}

// promptAssetSelection prompts the user to select from multiple matches.
// In non-TTY mode, returns an error with the match list.
func (r *resolver) promptAssetSelection(matches []AssetMatch, assetType, query string) (AssetMatch, error) {
	isTTY := false
	if f, ok := r.stdin.(*os.File); ok {
		isTTY = term.IsTerminal(int(f.Fd()))
	}

	if !isTTY {
		var names []string
		for _, m := range matches {
			names = append(names, m.Name)
		}
		return AssetMatch{}, fmt.Errorf("ambiguous %s %q matches: %s\nSpecify exact name or run interactively",
			assetType, query, strings.Join(names, ", "))
	}

	displayCount := len(matches)
	truncated := false
	if displayCount > maxAssetResults {
		displayCount = maxAssetResults
		truncated = true
	}

	_, _ = fmt.Fprintf(r.stdout, "Found %d %ss matching %q:\n\n", len(matches), assetType, query)

	// Find longest name for alignment
	maxNameLen := 0
	for i := 0; i < displayCount; i++ {
		if len(matches[i].Name) > maxNameLen {
			maxNameLen = len(matches[i].Name)
		}
	}

	for i := 0; i < displayCount; i++ {
		m := matches[i]
		padding := strings.Repeat(" ", maxNameLen-len(m.Name)+2)
		_, _ = fmt.Fprintf(r.stdout, "  %2d. %s%s%s\n", i+1, m.Name, padding, m.Source)
	}

	if truncated {
		_, _ = fmt.Fprintf(r.stdout, "\nShowing %d of %d matches. Refine search for more specific results.\n",
			displayCount, len(matches))
	}

	_, _ = fmt.Fprintln(r.stdout)
	_, _ = fmt.Fprintf(r.stdout, "Select %s%s%s: ", colorCyan.Sprint("("), colorDim.Sprintf("1-%d", displayCount), colorCyan.Sprint(")"))

	reader := bufio.NewReader(r.stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return AssetMatch{}, fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(input)

	// Try number
	if choice, err := strconv.Atoi(input); err == nil {
		if choice >= 1 && choice <= displayCount {
			return matches[choice-1], nil
		}
		return AssetMatch{}, fmt.Errorf("invalid selection: %s (choose 1-%d)", input, displayCount)
	}

	// Try exact name match
	inputLower := strings.ToLower(input)
	for i := 0; i < displayCount; i++ {
		if strings.ToLower(matches[i].Name) == inputLower {
			return matches[i], nil
		}
	}

	// Try substring
	var subMatches []AssetMatch
	for i := 0; i < displayCount; i++ {
		if strings.Contains(strings.ToLower(matches[i].Name), inputLower) {
			subMatches = append(subMatches, matches[i])
		}
	}
	if len(subMatches) == 1 {
		return subMatches[0], nil
	}

	return AssetMatch{}, fmt.Errorf("invalid selection: %s", input)
}

// autoInstall installs a registry asset to global config.
func (r *resolver) autoInstall(client *registry.Client, result assets.SearchResult) error {
	if client == nil {
		return fmt.Errorf("registry client unavailable")
	}

	ctx := context.Background()

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	if !r.flags.Quiet {
		_, _ = fmt.Fprintf(r.stdout, "Installing %s from registry...\n", result.Name)
	}

	if err := assets.InstallAsset(ctx, client, r.index, result, paths.Global); err != nil {
		return err
	}

	if !r.flags.Quiet {
		_, _ = fmt.Fprintf(r.stdout, "Installed %s to global config\n\n", result.Name)
	}

	r.didInstall = true
	return nil
}

// ensureIndex lazily fetches the registry index. Returns nil index with nil error
// if the registry is unavailable (graceful fallback).
func (r *resolver) ensureIndex() (*registry.Index, *registry.Client, error) {
	if r.skipRegistry {
		return nil, nil, nil
	}

	if r.didFetch {
		return r.index, r.client, r.indexErr
	}
	r.didFetch = true

	if !r.flags.Quiet {
		_, _ = fmt.Fprintf(r.stdout, "Fetching registry index...\n")
	}

	client, err := registry.NewClient()
	if err != nil {
		debugf(r.flags, "resolve", "Registry unavailable: %v", err)
		r.indexErr = err
		return nil, nil, nil // Graceful fallback
	}
	r.client = client

	ctx := context.Background()
	index, err := client.FetchIndex(ctx)
	if err != nil {
		debugf(r.flags, "resolve", "Index fetch failed: %v", err)
		r.indexErr = err
		return nil, client, nil // Graceful fallback
	}

	r.index = index
	return index, client, nil
}

// reloadConfig reloads the merged config after installs.
func (r *resolver) reloadConfig(workingDir string) error {
	cfg, err := loadMergedConfigFromDirWithDebug(workingDir, r.flags)
	if err != nil {
		return fmt.Errorf("reloading configuration: %w", err)
	}
	r.cfg = cfg
	return nil
}
