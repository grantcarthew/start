package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	"github.com/grantcarthew/start/internal/tui"
)

// loadForScope loads entities from the appropriate scope using a generic merge strategy.
// Returns the entity map, names in definition order, and any error.
// Order: global entries first (in definition order), then local entries (in definition order).
// Local entries override global entries with the same name but retain their global position.
func loadForScope[T any](
	localOnly bool,
	loadFromDir func(string) (map[string]T, []string, error),
	setSource func(*T, string),
) (map[string]T, []string, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return nil, nil, fmt.Errorf("resolving config paths: %w", err)
	}

	items := make(map[string]T)
	var order []string
	seen := make(map[string]bool)

	if localOnly {
		if paths.LocalExists {
			localItems, localOrder, err := loadFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, nil, err
			}
			for _, name := range localOrder {
				item := localItems[name]
				setSource(&item, "local")
				items[name] = item
				order = append(order, name)
			}
		}
	} else {
		if paths.GlobalExists {
			globalItems, globalOrder, err := loadFromDir(paths.Global)
			if err != nil && !os.IsNotExist(err) {
				return nil, nil, err
			}
			for _, name := range globalOrder {
				item := globalItems[name]
				setSource(&item, "global")
				items[name] = item
				order = append(order, name)
				seen[name] = true
			}
		}
		if paths.LocalExists {
			localItems, localOrder, err := loadFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, nil, err
			}
			for _, name := range localOrder {
				item := localItems[name]
				setSource(&item, "local")
				items[name] = item
				// Only add to order if not already present from global
				if !seen[name] {
					order = append(order, name)
				}
			}
		}
	}

	return items, order, nil
}

// promptString prompts for a string value with a default.
func promptString(w io.Writer, r io.Reader, label, defaultVal string) (string, error) {
	// Print label with cyan () delimiters for "(optional)"
	if base, found := strings.CutSuffix(label, " (optional)"); found {
		_, _ = fmt.Fprint(w, base)
		_, _ = fmt.Fprintf(w, " %s", tui.Annotate("optional"))
	} else {
		_, _ = fmt.Fprint(w, label)
	}
	if defaultVal != "" {
		_, _ = fmt.Fprintf(w, " %s", tui.Bracket("%s", defaultVal))
	}
	_, _ = fmt.Fprint(w, ": ")

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

// promptContentSource prompts the user to choose a content source (file, command, or inline prompt).
// defaultChoice is the default menu option ("1" for file, "3" for inline prompt).
// currentPrompt is passed to promptText as the default value for option 3.
// Returns the selected file, command, and prompt values (only one will be non-empty).
func promptContentSource(w io.Writer, r io.Reader, defaultChoice, currentPrompt string) (file, command, prompt string, err error) {
	_, _ = fmt.Fprintf(w, "\nContent source %s:\n", tui.Annotate("choose one"))
	_, _ = fmt.Fprintln(w, "  1. File path")
	_, _ = fmt.Fprintln(w, "  2. Command")
	_, _ = fmt.Fprintln(w, "  3. Inline prompt")
	_, _ = fmt.Fprintf(w, "Choice %s: ", tui.Bracket("%s", defaultChoice))

	reader := bufio.NewReader(r)
	choice, err := reader.ReadString('\n')
	if err != nil {
		return "", "", "", fmt.Errorf("reading input: %w", err)
	}
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = defaultChoice
	}

	switch choice {
	case "1":
		file, err = promptString(w, r, "File path", "")
		if err != nil {
			return "", "", "", err
		}
	case "2":
		command, err = promptString(w, r, "Command", "")
		if err != nil {
			return "", "", "", err
		}
	case "3":
		prompt, err = promptText(w, r, "Prompt text", currentPrompt)
		if err != nil {
			return "", "", "", err
		}
	default:
		return "", "", "", fmt.Errorf("invalid choice: %s", choice)
	}

	return file, command, prompt, nil
}

// promptText prompts for multi-line text input.
// Users can type text directly (finish with a blank line) or press Enter
// to open $EDITOR for longer input.
func promptText(w io.Writer, r io.Reader, label, defaultVal string) (string, error) {
	// Show current value if editing a multi-line default
	if defaultVal != "" && strings.Contains(defaultVal, "\n") {
		_, _ = fmt.Fprintf(w, "Current value:\n%s\n\n", defaultVal)
	}

	_, _ = fmt.Fprint(w, label)
	if defaultVal != "" && !strings.Contains(defaultVal, "\n") {
		_, _ = fmt.Fprintf(w, " %s", tui.Bracket("%s", defaultVal))
	}
	_, _ = fmt.Fprintln(w)
	_, _ = tui.ColorDim.Fprintln(w, "  Type text, then press Enter on a blank line to finish")
	_, _ = tui.ColorDim.Fprintln(w, "  Or press Enter now to open $EDITOR")
	_, _ = fmt.Fprint(w, "> ")

	reader := bufio.NewReader(r)
	firstLine, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	firstLine = strings.TrimRight(firstLine, "\r\n")

	// Empty first line: open editor
	if firstLine == "" {
		tmpFile, err := os.CreateTemp("", "start-prompt-*.md")
		if err != nil {
			return defaultVal, nil
		}
		tmpPath := tmpFile.Name()
		defer func() { _ = os.Remove(tmpPath) }()

		if defaultVal != "" {
			_, _ = tmpFile.WriteString(defaultVal)
		}
		_ = tmpFile.Close()

		if err := openInEditor(tmpPath); err != nil {
			return defaultVal, nil
		}

		content, err := os.ReadFile(tmpPath)
		if err != nil {
			return defaultVal, nil
		}

		result := strings.TrimRight(string(content), " \t\r\n")
		if result == "" {
			return defaultVal, nil
		}
		return result, nil
	}

	// User typed text: read lines until blank line
	var lines []string
	lines = append(lines, firstLine)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// EOF without newline - include what we have
			line = strings.TrimRight(line, "\r\n")
			if line != "" {
				lines = append(lines, line)
			}
			break
		}
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) == "" {
			break
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n"), nil
}

// promptDefaultModel prompts for a default model selection.
// When models are defined, displays a numbered list for selection.
// Falls back to free-text input when no models are defined.
func promptDefaultModel(w io.Writer, r io.Reader, current string, models map[string]string) (string, error) {
	if len(models) == 0 {
		return promptString(w, r, "Default model", current)
	}

	// Sort aliases for stable ordering
	aliases := make([]string, 0, len(models))
	for alias := range models {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)

	_, _ = fmt.Fprintln(w, "Default model:")
	for i, alias := range aliases {
		if alias == current {
			_, _ = fmt.Fprintf(w, "  %d. %s - %s %s\n", i+1, alias, tui.ColorDim.Sprint(models[alias]), tui.Annotate("%s", tui.ColorInstalled.Sprint("current")))
		} else {
			_, _ = fmt.Fprintf(w, "  %d. %s - %s\n", i+1, alias, tui.ColorDim.Sprint(models[alias]))
		}
	}

	_, _ = fmt.Fprintln(w)
	if current != "" {
		_, _ = fmt.Fprintf(w, "Select model %s: ", tui.Annotate("number, alias, or Enter to keep %q", current))
	} else {
		_, _ = fmt.Fprintf(w, "Select model %s: ", tui.Annotate("number or alias"))
	}

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return current, nil
	}

	// Try parsing as number
	if choice, err := strconv.Atoi(input); err == nil {
		if choice >= 1 && choice <= len(aliases) {
			return aliases[choice-1], nil
		}
		return "", fmt.Errorf("invalid selection: %s (choose 1-%d)", input, len(aliases))
	}

	// Try matching by alias
	for _, alias := range aliases {
		if strings.EqualFold(alias, input) {
			return alias, nil
		}
	}

	return "", fmt.Errorf("invalid selection: %q is not a known model alias", input)
}

// promptTags prompts for editing a slice of tags.
// Shows current tags and allows: comma-separated input to replace, empty to clear, Enter to keep.
func promptTags(w io.Writer, r io.Reader, current []string) ([]string, error) {
	if len(current) > 0 {
		_, _ = fmt.Fprintf(w, "Current tags: %s\n", tui.Bracket("%s", strings.Join(current, ", ")))
	} else {
		_, _ = fmt.Fprintf(w, "Current tags: %s\n", tui.Annotate("none"))
	}
	_, _ = fmt.Fprintf(w, "Tags %s: ", tui.Annotate("comma-separated, - to clear, Enter to keep"))

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Enter keeps current
	if input == "" {
		return current, nil
	}

	// "-" clears tags
	if input == "-" {
		return nil, nil
	}

	// Parse comma-separated tags
	var tags []string
	for _, t := range strings.Split(input, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	return tags, nil
}

// promptModels prompts for editing a map of model aliases.
// Offers options: (k)eep, (c)lear, (e)dit.
func promptModels(w io.Writer, r io.Reader, current map[string]string) (map[string]string, error) {
	reader := bufio.NewReader(r)

	if len(current) > 0 {
		_, _ = fmt.Fprintln(w, "Current models:")
		var aliases []string
		for alias := range current {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)
		for _, alias := range aliases {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", alias, tui.ColorDim.Sprint(current[alias]))
		}
	} else {
		_, _ = fmt.Fprintf(w, "Current models: %s\n", tui.Annotate("none"))
	}

	_, _ = fmt.Fprintf(w, "Models: %skeep, %sclear, %sedit %s: ",
		tui.Annotate("k"), tui.Annotate("c"), tui.Annotate("e"),
		tui.Bracket("k"))
	choice, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	choice = strings.TrimSpace(strings.ToLower(choice))

	switch choice {
	case "", "k", "keep":
		return current, nil
	case "c", "clear":
		return nil, nil
	case "e", "edit":
		return promptModelsEdit(w, reader, current)
	default:
		return nil, fmt.Errorf("invalid choice: %s", choice)
	}
}

// promptModelsEdit handles the edit mode for models.
func promptModelsEdit(w io.Writer, reader *bufio.Reader, current map[string]string) (map[string]string, error) {
	result := make(map[string]string)

	// Edit existing models
	if len(current) > 0 {
		_, _ = fmt.Fprintln(w, "Edit existing models (Enter to keep, - to delete):")
		var aliases []string
		for alias := range current {
			aliases = append(aliases, alias)
		}
		sort.Strings(aliases)

		for _, alias := range aliases {
			currentVal := current[alias]
			_, _ = fmt.Fprintf(w, "  %s %s: ", alias, tui.Bracket("%s", currentVal))

			input, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("reading input: %w", err)
			}
			input = strings.TrimSpace(input)

			if input == "-" {
				// Delete this model
				continue
			}
			if input == "" {
				// Keep current value
				result[alias] = currentVal
			} else {
				// Update value
				result[alias] = input
			}
		}
	}

	// Add new models
	_, _ = fmt.Fprintln(w, "Add new models (alias=model-id, empty to finish):")
	for {
		_, _ = fmt.Fprint(w, "  > ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("reading input: %w", err)
		}
		input = strings.TrimSpace(input)

		if input == "" {
			break
		}

		parts := strings.SplitN(input, "=", 2)
		if len(parts) != 2 {
			_, _ = fmt.Fprintln(w, "  Invalid format. Use: alias=model-id")
			continue
		}

		alias := strings.TrimSpace(parts[0])
		modelID := strings.TrimSpace(parts[1])
		if alias == "" || modelID == "" {
			_, _ = fmt.Fprintln(w, "  Invalid format. Use: alias=model-id")
			continue
		}

		result[alias] = modelID
	}

	return result, nil
}

// openInEditor opens a file in the user's editor.
func openInEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// truncatePrompt truncates a prompt for display.
func truncatePrompt(s string, max int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// writeCUETags writes a CUE tags array field into a strings.Builder.
func writeCUETags(sb *strings.Builder, tags []string) {
	if len(tags) == 0 {
		return
	}
	sb.WriteString("\t\ttags: [")
	for i, tag := range tags {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(sb, "%q", tag)
	}
	sb.WriteString("]\n")
}

// writeCUEPrompt writes a CUE prompt field into a strings.Builder,
// using triple-quote syntax for long or multi-line prompts.
func writeCUEPrompt(sb *strings.Builder, prompt string) {
	if prompt == "" {
		return
	}
	if strings.Contains(prompt, "\n") || len(prompt) > 80 {
		sb.WriteString("\t\tprompt: \"\"\"\n")
		for _, line := range strings.Split(prompt, "\n") {
			fmt.Fprintf(sb, "\t\t\t%s\n", line)
		}
		sb.WriteString("\t\t\t\"\"\"\n")
	} else {
		fmt.Fprintf(sb, "\t\tprompt: %q\n", prompt)
	}
}

// extractTags extracts a string slice from the "tags" field of a CUE value.
func extractTags(val cue.Value) []string {
	tagsVal := val.LookupPath(cue.ParsePath("tags"))
	if !tagsVal.Exists() {
		return nil
	}
	tagIter, err := tagsVal.List()
	if err != nil {
		return nil
	}
	var tags []string
	for tagIter.Next() {
		if s, err := tagIter.Value().String(); err == nil {
			tags = append(tags, s)
		}
	}
	return tags
}

// confirmMultiRemoval prompts the user to confirm removal of one or more config entities.
// For a single name it mirrors the old single-item prompt. For multiple names it lists
// them all and asks once. Requires --yes flag in non-interactive mode.
func confirmMultiRemoval(w io.Writer, r io.Reader, entityType string, names []string, local bool) (bool, error) {
	isTTY := isTerminal(r)

	if !isTTY {
		return false, fmt.Errorf("--yes flag required in non-interactive mode")
	}

	scope := scopeString(local)
	if len(names) == 1 {
		_, _ = fmt.Fprintf(w, "Remove %s %q from %s config? %s ", entityType, names[0], scope, tui.Bracket("y/N"))
	} else {
		_, _ = fmt.Fprintf(w, "Remove the following %ss from %s config?\n", entityType, scope)
		for _, name := range names {
			_, _ = fmt.Fprintf(w, "  - %s\n", name)
		}
		_, _ = fmt.Fprintf(w, "%s ", tui.Bracket("y/N"))
	}

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		_, _ = fmt.Fprintln(w, "Cancelled.")
		return false, nil
	}
	return true, nil
}

// scopeString returns "local" or "global" based on the flag.
func scopeString(local bool) string {
	if local {
		return "local"
	}
	return "global"
}

// scoreAndSortNames scores each map key against the compiled patterns and
// returns matching keys sorted by score descending then name ascending.
// Each pattern that matches a key contributes 3 to its score, so keys
// matching more query terms rank higher. Keys with score zero are excluded.
func scoreAndSortNames[T any](items map[string]T, patterns []*regexp.Regexp) []string {
	type match struct {
		name  string
		score int
	}
	var matches []match

	for name := range items {
		score := 0
		for _, pattern := range patterns {
			if pattern.MatchString(name) {
				score += 3 // Each matching pattern adds weight; higher score = more query terms matched
			}
		}
		if score > 0 {
			matches = append(matches, match{name: name, score: score})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].name < matches[j].name
	})

	names := make([]string, len(matches))
	for i, m := range matches {
		names[i] = m.name
	}
	return names
}

// resolveAllMatchingNames resolves a query to all matching names in a map.
// Unlike resolveInstalledName, it returns every match when a query is ambiguous
// rather than erroring. On zero matches it returns a "not found" error.
// Results are sorted by score descending then name ascending.
func resolveAllMatchingNames[T any](items map[string]T, typeName, query string) ([]string, error) {
	// Fast path: exact match
	if _, ok := items[query]; ok {
		return []string{query}, nil
	}

	terms := assets.ParseSearchPatterns(query)
	if len(terms) == 0 {
		return nil, fmt.Errorf("%s %q not found", typeName, query)
	}

	patterns, err := assets.CompileSearchTerms(terms)
	if err != nil {
		return nil, fmt.Errorf("%s %q not found (invalid pattern: %w)", typeName, query, err)
	}

	names := scoreAndSortNames(items, patterns)
	if len(names) == 0 {
		return nil, fmt.Errorf("%s %q not found", typeName, query)
	}
	return names, nil
}

// promptSelectCategory displays a colour-coded numbered list of config categories
// and returns the chosen category name. Returns "" and nil if the user cancels
// (empty input).
func promptSelectCategory(w io.Writer, r io.Reader, categories []string) (string, error) {
	for i, cat := range categories {
		_, _ = fmt.Fprintf(w, "  %d. %s\n", i+1, tui.CategoryColor(cat).Sprint(cat))
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Select %s: ", tui.Annotate("1-%d", len(categories)))

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(input)

	if input == "" {
		_, _ = fmt.Fprintln(w, "Cancelled.")
		return "", nil
	}

	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(categories) {
		return "", fmt.Errorf("invalid selection %q: enter a number between 1 and %d", input, len(categories))
	}
	return categories[n-1], nil
}

// promptSelectOneFromList displays a numbered list and lets the user pick a
// single entry by number. Returns "" and nil if the user cancels (empty input).
func promptSelectOneFromList(w io.Writer, r io.Reader, entityType string, names []string) (string, error) {
	if len(names) == 0 {
		return "", nil
	}
	_, _ = fmt.Fprintf(w, "%d %ss:\n\n", len(names), entityType)
	for i, name := range names {
		_, _ = fmt.Fprintf(w, "  %2d. %s\n", i+1, name)
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Select %s: ", tui.Annotate("1-%d", len(names)))

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(input)

	if input == "" {
		_, _ = fmt.Fprintln(w, "Cancelled.")
		return "", nil
	}

	n, err := strconv.Atoi(input)
	if err != nil || n < 1 || n > len(names) {
		return "", fmt.Errorf("invalid selection %q: enter a number between 1 and %d", input, len(names))
	}
	return names[n-1], nil
}

// promptSelectFromList displays a numbered list of candidates and lets the user
// choose which to include. The user may enter comma-separated numbers, ranges
// (e.g. "1-3"), or "all". Returns the chosen names in list order, or nil if
// the user cancels (empty input).
func promptSelectFromList(w io.Writer, r io.Reader, entityType, query string, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	if query != "" {
		_, _ = fmt.Fprintf(w, "Found %d %ss matching %q:\n\n", len(names), entityType, query)
	} else {
		_, _ = fmt.Fprintf(w, "%d %ss:\n\n", len(names), entityType)
	}

	for i, name := range names {
		_, _ = fmt.Fprintf(w, "  %2d. %s\n", i+1, name)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Select %s or all: ", tui.Annotate("1-%d", len(names)))

	reader := bufio.NewReader(r)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		_, _ = fmt.Fprintln(w, "Cancelled.")
		return nil, nil
	}
	if input == "all" {
		return names, nil
	}

	seen := make(map[int]bool)
	var selected []string

	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if dashIdx := strings.Index(part, "-"); dashIdx > 0 {
			start, err1 := strconv.Atoi(strings.TrimSpace(part[:dashIdx]))
			end, err2 := strconv.Atoi(strings.TrimSpace(part[dashIdx+1:]))
			if err1 != nil || err2 != nil || start < 1 || end > len(names) || start > end {
				return nil, fmt.Errorf("invalid range %q: enter numbers between 1 and %d", part, len(names))
			}
			for i := start; i <= end; i++ {
				if !seen[i] {
					seen[i] = true
					selected = append(selected, names[i-1])
				}
			}
		} else {
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 || n > len(names) {
				return nil, fmt.Errorf("invalid selection %q: enter a number between 1 and %d", part, len(names))
			}
			if !seen[n] {
				seen[n] = true
				selected = append(selected, names[n-1])
			}
		}
	}

	if len(selected) == 0 {
		_, _ = fmt.Fprintln(w, "Cancelled.")
		return nil, nil
	}
	return selected, nil
}

// resolveRemoveNames resolves CLI args to a deduplicated list of names to remove.
// With a single ambiguous arg and an interactive terminal, it shows a picker.
// With multiple args, any ambiguous arg without --yes returns an error.
// Returns (nil, nil) when the user cancels the interactive picker (caller should
// return nil). Returns (nil, err) on any other failure.
func resolveRemoveNames[T any](items map[string]T, typeName string, args []string, skipConfirm bool, stdout io.Writer, stdin io.Reader) ([]string, error) {
	seen := make(map[string]bool)
	var resolvedNames []string
	for _, arg := range args {
		candidates, err := resolveAllMatchingNames(items, typeName, arg)
		if err != nil {
			return nil, err
		}
		if len(candidates) > 1 && !skipConfirm {
			if len(args) > 1 {
				return nil, fmt.Errorf("%q matches multiple %ss — use an exact name or pass --yes to remove all matches", arg, typeName)
			}
			if !isTerminal(stdin) {
				return nil, fmt.Errorf("--yes flag required in non-interactive mode for ambiguous %s %q", typeName, arg)
			}
			candidates, err = promptSelectFromList(stdout, stdin, typeName, arg, candidates)
			if err != nil {
				return nil, err
			}
			if len(candidates) == 0 {
				return nil, nil // user cancelled
			}
		}
		for _, name := range candidates {
			if !seen[name] {
				seen[name] = true
				resolvedNames = append(resolvedNames, name)
			}
		}
	}
	return resolvedNames, nil
}

// resolveInstalledName resolves a name from a map using exact match first,
// then regex-based search. Returns the resolved key and value.
// On zero matches, returns a "not found" error.
// On multiple matches, returns an "ambiguous" error listing the matches.
func resolveInstalledName[T any](items map[string]T, typeName, query string) (string, T, error) {
	var zero T

	// Fast path: exact match
	if val, ok := items[query]; ok {
		return query, val, nil
	}

	// Regex-based search across map keys
	terms := assets.ParseSearchPatterns(query)
	if len(terms) == 0 {
		return "", zero, fmt.Errorf("%s %q not found", typeName, query)
	}

	patterns, err := assets.CompileSearchTerms(terms)
	if err != nil {
		return "", zero, fmt.Errorf("%s %q not found (invalid pattern: %w)", typeName, query, err)
	}

	names := scoreAndSortNames(items, patterns)
	switch len(names) {
	case 0:
		return "", zero, fmt.Errorf("%s %q not found", typeName, query)
	case 1:
		return names[0], items[names[0]], nil
	default:
		return "", zero, fmt.Errorf("ambiguous %s %q matches multiple entries: %s",
			typeName, query, strings.Join(names, ", "))
	}
}
