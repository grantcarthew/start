package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
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
		_, _ = fmt.Fprint(w, " ")
		_, _ = colorCyan.Fprint(w, "(")
		_, _ = colorDim.Fprint(w, "optional")
		_, _ = colorCyan.Fprint(w, ")")
	} else {
		_, _ = fmt.Fprint(w, label)
	}
	if defaultVal != "" {
		_, _ = fmt.Fprint(w, " ")
		_, _ = colorCyan.Fprint(w, "[")
		_, _ = colorDim.Fprint(w, defaultVal)
		_, _ = colorCyan.Fprint(w, "]")
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
		_, _ = fmt.Fprint(w, " ")
		_, _ = colorCyan.Fprint(w, "[")
		_, _ = colorDim.Fprint(w, defaultVal)
		_, _ = colorCyan.Fprint(w, "]")
	}
	_, _ = fmt.Fprintln(w)
	_, _ = colorDim.Fprintln(w, "  Type text, then press Enter on a blank line to finish")
	_, _ = colorDim.Fprintln(w, "  Or press Enter now to open $EDITOR")
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
			_, _ = fmt.Fprintf(w, "  %d. %s - %s %s%s%s\n", i+1, alias, colorDim.Sprint(models[alias]), colorCyan.Sprint("("), colorInstalled.Sprint("current"), colorCyan.Sprint(")"))
		} else {
			_, _ = fmt.Fprintf(w, "  %d. %s - %s\n", i+1, alias, colorDim.Sprint(models[alias]))
		}
	}

	_, _ = fmt.Fprintln(w)
	if current != "" {
		_, _ = fmt.Fprintf(w, "Select model %s%s%s: ", colorCyan.Sprint("("), colorDim.Sprintf("number, alias, or Enter to keep %q", current), colorCyan.Sprint(")"))
	} else {
		_, _ = fmt.Fprintf(w, "Select model %s%s%s: ", colorCyan.Sprint("("), colorDim.Sprint("number or alias"), colorCyan.Sprint(")"))
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
		_, _ = fmt.Fprintf(w, "Current tags: %s%s%s\n", colorCyan.Sprint("["), colorDim.Sprint(strings.Join(current, ", ")), colorCyan.Sprint("]"))
	} else {
		_, _ = fmt.Fprintf(w, "Current tags: %s%s%s\n", colorCyan.Sprint("("), colorDim.Sprint("none"), colorCyan.Sprint(")"))
	}
	_, _ = fmt.Fprintf(w, "Tags %s%s%s: ", colorCyan.Sprint("("), colorDim.Sprint("comma-separated, - to clear, Enter to keep"), colorCyan.Sprint(")"))

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
			_, _ = fmt.Fprintf(w, "  %s: %s\n", alias, colorDim.Sprint(current[alias]))
		}
	} else {
		_, _ = fmt.Fprintf(w, "Current models: %s%s%s\n", colorCyan.Sprint("("), colorDim.Sprint("none"), colorCyan.Sprint(")"))
	}

	_, _ = fmt.Fprintf(w, "Models: %sk%seep, %sc%slear, %se%sdit %s%s%s: ",
		colorCyan.Sprint("("), colorCyan.Sprint(")"),
		colorCyan.Sprint("("), colorCyan.Sprint(")"),
		colorCyan.Sprint("("), colorCyan.Sprint(")"),
		colorCyan.Sprint("["), colorDim.Sprint("k"), colorCyan.Sprint("]"))
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
			_, _ = fmt.Fprintf(w, "  %s %s%s%s: ", alias, colorCyan.Sprint("["), colorDim.Sprint(currentVal), colorCyan.Sprint("]"))

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

// confirmRemoval prompts the user to confirm removal of a config entity.
// Returns true if confirmed, false if cancelled. Requires --yes flag in non-interactive mode.
func confirmRemoval(w io.Writer, r io.Reader, entityType, name string, local bool) (bool, error) {
	isTTY := isTerminal(r)

	if !isTTY {
		return false, fmt.Errorf("--yes flag required in non-interactive mode")
	}

	_, _ = fmt.Fprintf(w, "Remove %s %q from %s config? %s%s%s ", entityType, name, scopeString(local), colorCyan.Sprint("["), colorDim.Sprint("y/N"), colorCyan.Sprint("]"))
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

	// Sort by score descending, then name ascending
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].name < matches[j].name
	})

	switch len(matches) {
	case 0:
		return "", zero, fmt.Errorf("%s %q not found", typeName, query)
	case 1:
		name := matches[0].name
		return name, items[name], nil
	default:
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = m.name
		}
		return "", zero, fmt.Errorf("ambiguous %s %q matches multiple entries: %s",
			typeName, query, strings.Join(names, ", "))
	}
}
