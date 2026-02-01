package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
	"github.com/grantcarthew/start/internal/config"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// addAssetsAddCommand adds the add subcommand to the assets command.
func addAssetsAddCommand(parent *cobra.Command) {
	addCmd := &cobra.Command{
		Use:   "add <query>",
		Short: "Install asset from registry",
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
	results := searchIndex(index, query)

	if len(results) == 0 {
		return fmt.Errorf("no assets found matching %q", query)
	}

	// Select asset
	var selected SearchResult
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

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Fetch the actual asset module from registry
	if !flags.Quiet {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Fetching asset...")
	}

	modulePath := selected.Entry.Module
	if !strings.Contains(modulePath, "@") {
		modulePath += "@v0"
	}

	// Resolve to canonical version (e.g., @v0 -> @v0.0.1)
	resolvedPath, err := client.ResolveLatestVersion(ctx, modulePath)
	if err != nil {
		return fmt.Errorf("resolving asset version: %w", err)
	}

	fetchResult, err := client.Fetch(ctx, resolvedPath)
	if err != nil {
		return fmt.Errorf("fetching asset module: %w", err)
	}

	// Extract asset content from fetched module
	// Use resolved path with version for origin field (e.g., "github.com/.../task@v0.1.1")
	assetContent, err := extractAssetContent(fetchResult.SourceDir, selected, client.Registry(), resolvedPath)
	if err != nil {
		return fmt.Errorf("extracting asset content: %w", err)
	}

	// Determine the config file to write to based on asset type
	configFile := assetTypeToConfigFile(selected.Category)
	configPath := filepath.Join(configDir, configFile)

	// Write the asset to config
	if err := writeAssetToConfig(configPath, selected, assetContent, modulePath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	if !flags.Quiet {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nInstalled %s/%s to %s config\n", selected.Category, selected.Name, scopeName)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", configPath)
	}

	return nil
}

// promptAssetSelection prompts the user to select an asset from multiple matches.
func promptAssetSelection(w io.Writer, r io.Reader, results []SearchResult) (SearchResult, error) {
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
		return SearchResult{}, fmt.Errorf(
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
		return SearchResult{}, fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Try parsing as number
	if choice, err := strconv.Atoi(input); err == nil {
		if choice >= 1 && choice <= len(results) {
			return results[choice-1], nil
		}
		return SearchResult{}, fmt.Errorf("invalid selection: %s (choose 1-%d)", input, len(results))
	}

	// Try matching by name
	inputLower := strings.ToLower(input)
	for _, res := range results {
		fullPath := fmt.Sprintf("%s/%s", res.Category, res.Name)
		if strings.ToLower(res.Name) == inputLower || strings.ToLower(fullPath) == inputLower {
			return res, nil
		}
	}

	return SearchResult{}, fmt.Errorf("invalid selection: %s", input)
}

// assetTypeToConfigFile returns the config file name for an asset type.
func assetTypeToConfigFile(category string) string {
	switch category {
	case "agents":
		return "agents.cue"
	case "roles":
		return "roles.cue"
	case "tasks":
		return "tasks.cue"
	case "contexts":
		return "contexts.cue"
	default:
		return "settings.cue"
	}
}

// extractAssetContent loads the asset module and extracts its content as CUE.
// originPath is the module path (without version) to store in the origin field.
func extractAssetContent(moduleDir string, asset SearchResult, reg interface{}, originPath string) (string, error) {
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
	// e.g., "task", "role", "agent", "context"
	singular := strings.TrimSuffix(asset.Category, "s")
	assetVal := v.LookupPath(cue.ParsePath(singular))
	if !assetVal.Exists() {
		// Try the key name
		assetKey := getAssetKey(asset.Name)
		assetVal = v.LookupPath(cue.MakePath(cue.Str(assetKey)))
	}
	if !assetVal.Exists() {
		return "", fmt.Errorf("asset definition not found in module (tried %q)", singular)
	}

	// Build a concrete struct with just the fields we need
	return formatAssetStruct(assetVal, asset.Category, originPath)
}

// formatAssetStruct formats a CUE value as a concrete struct.
// originPath is written as the origin field to track registry provenance.
func formatAssetStruct(v cue.Value, category, originPath string) (string, error) {
	var sb strings.Builder
	sb.WriteString("{\n")

	// Write origin field first (tracks registry provenance)
	sb.WriteString(fmt.Sprintf("\torigin: %q\n", originPath))

	// Define which fields to extract based on category
	var fields []string
	switch category {
	case "tasks":
		fields = []string{"description", "tags", "role", "file", "command", "prompt"}
	case "roles":
		fields = []string{"description", "tags", "file", "command", "prompt"}
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

// getAssetKey returns the asset key name for use in config.
// Per DR-003, the full category/item path is preserved to avoid collisions.
// e.g., "golang/code-review" -> "golang/code-review"
func getAssetKey(name string) string {
	return name
}

// writeAssetToConfig writes the asset content to the config file.
func writeAssetToConfig(configPath string, asset SearchResult, content, modulePath string) error {
	// Read existing content if file exists
	var existingContent string
	if data, err := os.ReadFile(configPath); err == nil {
		existingContent = string(data)
	}

	assetKey := getAssetKey(asset.Name)

	// Check if already installed
	if strings.Contains(existingContent, fmt.Sprintf("%q:", assetKey)) ||
		strings.Contains(existingContent, assetKey+":") {
		return fmt.Errorf("asset %q already installed", assetKey)
	}

	// Build new content
	var sb strings.Builder

	if existingContent == "" {
		// New file
		sb.WriteString("// start configuration\n")
		sb.WriteString("// Managed by 'start assets add'\n\n")
		sb.WriteString(fmt.Sprintf("%s: {\n", asset.Category))
	} else {
		// Append to existing - find the closing brace of the category
		categoryStart := strings.Index(existingContent, asset.Category+":")
		if categoryStart == -1 {
			// Category doesn't exist, append it
			sb.WriteString(existingContent)
			if !strings.HasSuffix(existingContent, "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("\n%s: {\n", asset.Category))
		} else {
			// Find the closing brace and insert before it
			// This is a simple approach - for complex files might need proper parsing
			closingBrace := strings.LastIndex(existingContent, "}")
			if closingBrace == -1 {
				sb.WriteString(existingContent)
				sb.WriteString(fmt.Sprintf("\n%s: {\n", asset.Category))
			} else {
				sb.WriteString(existingContent[:closingBrace])
				sb.WriteString("\n")
			}
		}
	}

	// Add the asset definition
	sb.WriteString(fmt.Sprintf("\t%q: ", assetKey))

	// Indent the content
	lines := strings.Split(content, "\n")
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

	if existingContent == "" || !strings.Contains(existingContent, asset.Category+":") {
		sb.WriteString("\n}\n")
	} else {
		sb.WriteString("\n}\n")
	}

	return os.WriteFile(configPath, []byte(sb.String()), 0644)
}

// findAssetKey finds the position of an asset key in CUE content, ignoring occurrences
// in comments and strings. Returns the start position and length of the matched key pattern.
//
// Supported CUE syntax:
//   - Single-line strings: "..."
//   - Multi-line strings: """..."""
//   - Line comments: // (CUE does not support block comments /* */)
//   - Escape sequences: \" inside strings
//
// Searches for both quoted ("key":) and unquoted (key:) patterns.
// Returns the first match found in normal state (not inside strings/comments).
func findAssetKey(content, assetKey string) (keyStart, keyLen int, err error) {
	type state int
	const (
		stateNormal state = iota
		stateInString
		stateInMultiString
		stateInComment
	)

	quotedKey := fmt.Sprintf("%q:", assetKey)
	unquotedKey := assetKey + ":"
	currentState := stateNormal
	pos := 0

	for pos < len(content) {
		ch := content[pos]

		switch currentState {
		case stateNormal:
			// Check for matches only in normal state
			if pos+len(quotedKey) <= len(content) && content[pos:pos+len(quotedKey)] == quotedKey {
				return pos, len(quotedKey), nil
			}
			if pos+len(unquotedKey) <= len(content) && content[pos:pos+len(unquotedKey)] == unquotedKey {
				return pos, len(unquotedKey), nil
			}

			// Check for string start
			if ch == '"' {
				if pos+2 < len(content) && content[pos:pos+3] == `"""` {
					currentState = stateInMultiString
					pos += 2
				} else {
					currentState = stateInString
				}
			} else if ch == '/' && pos+1 < len(content) && content[pos+1] == '/' {
				currentState = stateInComment
				pos++
			}

		case stateInString:
			if ch == '\\' && pos+1 < len(content) {
				pos++
			} else if ch == '"' {
				currentState = stateNormal
			}

		case stateInMultiString:
			if ch == '"' && pos+2 < len(content) && content[pos:pos+3] == `"""` {
				currentState = stateNormal
				pos += 2
			}

		case stateInComment:
			if ch == '\n' {
				currentState = stateNormal
			}
		}

		pos++
	}

	return 0, 0, fmt.Errorf("asset %q not found in config", assetKey)
}

// findMatchingBrace finds the position after the matching closing brace for an opening brace.
// It respects CUE syntax: strings (both " and """), comments (//), and only counts braces
// that are part of the actual structure, not those inside strings or comments.
//
// Supported CUE syntax (same as findAssetKey):
//   - Single-line strings: "..."
//   - Multi-line strings: """..."""
//   - Line comments: // (CUE does not support block comments /* */)
//   - Escape sequences: \" inside strings
//
// Returns the position immediately after the matching closing brace.
func findMatchingBrace(content string, openBracePos int) (int, error) {
	type state int
	const (
		stateNormal        state = iota
		stateInString            // Inside "..." string
		stateInMultiString       // Inside """...""" string
		stateInComment           // After // until newline
	)

	currentState := stateNormal
	braceCount := 1
	pos := openBracePos + 1

	for pos < len(content) && braceCount > 0 {
		ch := content[pos]

		switch currentState {
		case stateNormal:
			// Check for string start
			if ch == '"' {
				// Check if it's a triple-quote
				if pos+2 < len(content) && content[pos:pos+3] == `"""` {
					currentState = stateInMultiString
					pos += 2 // Skip next 2 quotes (we'll increment at end of loop)
				} else {
					currentState = stateInString
				}
			} else if ch == '/' && pos+1 < len(content) && content[pos+1] == '/' {
				// Start of comment
				currentState = stateInComment
				pos++ // Skip second /
			} else if ch == '{' {
				braceCount++
			} else if ch == '}' {
				braceCount--
			}

		case stateInString:
			// Check for escape sequences
			if ch == '\\' && pos+1 < len(content) {
				pos++ // Skip next character
			} else if ch == '"' {
				currentState = stateNormal
			}

		case stateInMultiString:
			// Check for end of multi-line string
			if ch == '"' && pos+2 < len(content) && content[pos:pos+3] == `"""` {
				currentState = stateNormal
				pos += 2 // Skip next 2 quotes
			}

		case stateInComment:
			// Comment ends at newline
			if ch == '\n' {
				currentState = stateNormal
			}
		}

		pos++
	}

	if braceCount != 0 {
		return 0, fmt.Errorf("unmatched braces (count: %d)", braceCount)
	}

	return pos, nil
}

// findOpeningBrace finds the position of the first opening brace '{' after startPos,
// while respecting CUE syntax (ignoring braces inside comments and strings).
//
// Supported CUE syntax (same as findAssetKey and findMatchingBrace):
//   - Single-line strings: "..."
//   - Multi-line strings: """..."""
//   - Line comments: // (CUE does not support block comments /* */)
//   - Escape sequences: \" inside strings
//
// Returns the position of the opening brace, or error if not found.
func findOpeningBrace(content string, startPos int) (int, error) {
	type state int
	const (
		stateNormal state = iota
		stateInString
		stateInMultiString
		stateInComment
	)

	currentState := stateNormal
	pos := startPos

	for pos < len(content) {
		ch := content[pos]

		switch currentState {
		case stateNormal:
			// Found opening brace in normal state - this is what we're looking for
			if ch == '{' {
				return pos, nil
			}

			// Check for string start
			if ch == '"' {
				if pos+2 < len(content) && content[pos:pos+3] == `"""` {
					currentState = stateInMultiString
					pos += 2 // Skip next 2 quotes (we'll increment at end of loop)
				} else {
					currentState = stateInString
				}
			} else if ch == '/' && pos+1 < len(content) && content[pos+1] == '/' {
				currentState = stateInComment
				pos++ // Skip second /
			}

		case stateInString:
			if ch == '\\' && pos+1 < len(content) {
				pos++ // Skip next character (escape sequence)
			} else if ch == '"' {
				currentState = stateNormal
			}

		case stateInMultiString:
			if ch == '"' && pos+2 < len(content) && content[pos:pos+3] == `"""` {
				currentState = stateNormal
				pos += 2 // Skip next 2 quotes
			}

		case stateInComment:
			if ch == '\n' {
				currentState = stateNormal
			}
		}

		pos++
	}

	return 0, fmt.Errorf("opening brace not found")
}

// updateAssetInConfig replaces an existing asset entry in the config file.
func updateAssetInConfig(configPath, category, name, newContent string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading config file: %w", err)
	}

	content := string(data)
	assetKey := getAssetKey(name)

	// Find the asset entry using context-aware search (ignores keys in comments/strings)
	keyStart, keyLen, err := findAssetKey(content, assetKey)
	if err != nil {
		return err
	}

	// Find the opening brace after the key using context-aware search
	// This properly handles comments and strings between the key and brace
	// (e.g., "key": // comment { with brace } \n {)
	braceStart, err := findOpeningBrace(content, keyStart+keyLen)
	if err != nil {
		return fmt.Errorf("invalid config format: no opening brace for %q", assetKey)
	}

	// Find the matching closing brace using context-aware parsing
	braceEnd, err := findMatchingBrace(content, braceStart)
	if err != nil {
		return fmt.Errorf("finding closing brace for %q: %w", assetKey, err)
	}

	// Build the new content - preserve key and replace value
	var sb strings.Builder
	sb.WriteString(content[:keyStart])
	// NOTE: Always uses quoted key format ("key":) even if original was unquoted (key:).
	// CUE accepts both formats, but this normalizes to quoted for consistency.
	sb.WriteString(fmt.Sprintf("%q: ", assetKey))

	// Indent the new content
	// LIMITATION: This assumes a flat config structure where assets are direct children
	// of the category (e.g., tasks: { "name": { ... } }). It adds exactly one tab to
	// each line (except the first). If configs ever use nested structures, this would
	// need to detect the current indentation level from content[:keyStart].
	//
	// Current structure: category: { asset: { fields } }
	// - Line 0: '{' (no indent added, inherits position after key)
	// - Line 1+: Add one tab to match category level
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

	sb.WriteString(content[braceEnd:])

	return os.WriteFile(configPath, []byte(sb.String()), 0644)
}

// generateAssetCUE generates CUE content for an asset (stub version for tests).
func generateAssetCUE(asset SearchResult, modulePath, existingContent string) string {
	// If file already has content, append to it
	if existingContent != "" {
		// Check if the module is already imported
		if strings.Contains(existingContent, modulePath) {
			// Already imported, return as-is
			return existingContent
		}

		// Add a comment and the new asset reference
		return existingContent + fmt.Sprintf("\n// Added: %s/%s\n// Module: %s\n", asset.Category, asset.Name, modulePath)
	}

	// Generate new file content
	var sb strings.Builder

	sb.WriteString("// start configuration\n")
	sb.WriteString(fmt.Sprintf("// Added: %s/%s\n", asset.Category, asset.Name))
	sb.WriteString(fmt.Sprintf("// Module: %s\n\n", modulePath))

	// Generate the appropriate struct based on category
	switch asset.Category {
	case "agents":
		sb.WriteString("agents: {\n")
		sb.WriteString(fmt.Sprintf("\t// Import from: %s\n", modulePath))
		sb.WriteString(fmt.Sprintf("\t// %s: ...\n", asset.Name))
		sb.WriteString("}\n")
	case "roles":
		sb.WriteString("roles: {\n")
		sb.WriteString(fmt.Sprintf("\t// Import from: %s\n", modulePath))
		sb.WriteString(fmt.Sprintf("\t// %s: ...\n", asset.Name))
		sb.WriteString("}\n")
	case "tasks":
		sb.WriteString("tasks: {\n")
		sb.WriteString(fmt.Sprintf("\t// Import from: %s\n", modulePath))
		sb.WriteString(fmt.Sprintf("\t// %s: ...\n", asset.Name))
		sb.WriteString("}\n")
	case "contexts":
		sb.WriteString("contexts: {\n")
		sb.WriteString(fmt.Sprintf("\t// Import from: %s\n", modulePath))
		sb.WriteString(fmt.Sprintf("\t// %s: ...\n", asset.Name))
		sb.WriteString("}\n")
	}

	return sb.String()
}
