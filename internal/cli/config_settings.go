package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/grantcarthew/start/internal/shell"
	"github.com/grantcarthew/start/internal/tui"
	"github.com/spf13/cobra"
)

// settingInfo describes a valid settings key with its type.
type settingInfo struct {
	Type string // "string" or "int"
}

// settingEntry holds a resolved setting value and its source.
type settingEntry struct {
	Value  string
	Source string // "default", "global", "local", or "not set"
}

// settingsRegistry defines all valid settings keys and their types.
var settingsRegistry = map[string]settingInfo{
	"assets_index":  {Type: "string"},
	"default_agent": {Type: "string"},
	"shell":         {Type: "string"},
	"timeout":       {Type: "int"},
}

// validSettingsKeysString returns a sorted, comma-separated list of valid setting keys.
func validSettingsKeysString() string {
	keys := make([]string, 0, len(settingsRegistry))
	for k := range settingsRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// settingDefault returns the default value for a setting key.
// Returns empty string if the key has no default.
func settingDefault(key string) string {
	switch key {
	case "assets_index":
		return registry.IndexModulePath
	case "shell":
		if detected, err := shell.DetectShell(); err == nil {
			return strings.TrimSuffix(detected, " -c")
		}
		return ""
	case "timeout":
		return strconv.Itoa(shell.DefaultTimeout)
	default:
		return ""
	}
}

// resolveAllSettings resolves all valid settings with their values and sources.
func resolveAllSettings(paths config.Paths, localOnly bool) (map[string]settingEntry, error) {
	entries := make(map[string]settingEntry, len(settingsRegistry))

	// Start with defaults
	for key := range settingsRegistry {
		if def := settingDefault(key); def != "" {
			entries[key] = settingEntry{Value: def, Source: "default"}
		} else {
			entries[key] = settingEntry{Source: "not set"}
		}
	}

	if localOnly {
		if paths.LocalExists {
			localSettings, err := loadSettingsFromDir(paths.Local)
			if err != nil {
				return nil, err
			}
			for k, v := range localSettings {
				entries[k] = settingEntry{Value: v, Source: "local"}
			}
		}
	} else {
		if paths.GlobalExists {
			globalSettings, err := loadSettingsFromDir(paths.Global)
			if err != nil {
				return nil, err
			}
			for k, v := range globalSettings {
				entries[k] = settingEntry{Value: v, Source: "global"}
			}
		}
		if paths.LocalExists {
			localSettings, err := loadSettingsFromDir(paths.Local)
			if err != nil {
				return nil, err
			}
			for k, v := range localSettings {
				entries[k] = settingEntry{Value: v, Source: "local"}
			}
		}
	}

	return entries, nil
}

// addConfigSettingsCommand adds the settings subcommand to config.
func addConfigSettingsCommand(parent *cobra.Command) {
	settingsCmd := &cobra.Command{
		Use:     "settings [key] [value]",
		Aliases: []string{"setting"},
		Short:   "Manage settings configuration",
		Long: `Manage settings for start.

Available settings:
  assets_index   CUE module path for the assets index (default: built-in)
  default_agent  Agent to use when --agent not specified
  shell          Shell for command execution (default: auto-detect)
  timeout        Command timeout in seconds`,
		Example: `  start config settings                                               List all settings
  start config settings <key>                                         Show a setting value
  start config settings <key> <val>                                   Set a setting value
  start config settings <key> --unset                                 Remove a setting value
  start config settings edit                                          Open settings.cue in $EDITOR

  start config settings assets_index                                  Show current index path
  start config settings assets_index "github.com/grantcarthew/start-assets/index@v0"
  start config settings assets_index --unset                          Restore default index
  start config settings default_agent claude
  start config settings shell /bin/bash
  start config settings timeout 120`,
		Args: cobra.MaximumNArgs(2),
		RunE: executeConfigSettings,
	}

	settingsCmd.Flags().Bool("unset", false, "Remove a setting value")

	parent.AddCommand(settingsCmd)
}

// executeConfigSettings handles the settings command.
func executeConfigSettings(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}

	stdout := cmd.OutOrStdout()
	flags := getFlags(cmd)
	local := flags.Local
	unset, _ := cmd.Flags().GetBool("unset")

	if unset {
		if len(args) == 0 {
			return fmt.Errorf("--unset requires a setting key")
		}
		if len(args) > 1 {
			return fmt.Errorf("--unset takes only one argument")
		}
		return unsetSetting(stdout, flags, args[0], local)
	}

	switch len(args) {
	case 0:
		// List all settings
		return listSettings(stdout, local)
	case 1:
		if args[0] == "list" || args[0] == "ls" {
			return listSettings(stdout, local)
		}
		if args[0] == "edit" {
			return editSettings(local)
		}
		// Show single setting
		return showSetting(stdout, args[0], local)
	case 2:
		// Set setting
		return setSetting(stdout, flags, args[0], args[1], local)
	default:
		return fmt.Errorf("too many arguments")
	}
}

// listSettings displays all settings with their values and sources.
func listSettings(w io.Writer, localOnly bool) error {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	// Show config paths
	printConfigPaths(w, paths)
	_, _ = fmt.Fprintln(w)

	entries, err := resolveAllSettings(paths, localOnly)
	if err != nil {
		return err
	}

	printSettingsEntries(w, entries)
	return nil
}

// printConfigPaths displays the configuration directory paths.
func printConfigPaths(w io.Writer, paths config.Paths) {
	_, _ = tui.ColorHeader.Fprintln(w, "Configuration Paths:")
	_, _ = fmt.Fprintln(w)
	globalStatus := "not found"
	if paths.GlobalExists {
		globalStatus = "exists"
	}
	localStatus := "not found"
	if paths.LocalExists {
		localStatus = "exists"
	}
	_, _ = fmt.Fprintf(w, "  Global: %s ", paths.Global)
	_, _ = fmt.Fprintln(w, tui.Annotate("%s", globalStatus))
	_, _ = fmt.Fprintf(w, "  Local:  %s ", paths.Local)
	_, _ = fmt.Fprintln(w, tui.Annotate("%s", localStatus))
}

// printSettingsEntries displays resolved setting entries in a formatted table.
func printSettingsEntries(w io.Writer, entries map[string]settingEntry) {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	maxLen := 0
	for _, k := range keys {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	for _, k := range keys {
		entry := entries[k]
		_, _ = tui.ColorDim.Fprintf(w, "%*s: ", maxLen, k)
		if entry.Source == "not set" {
			_, _ = fmt.Fprintln(w, tui.Annotate("not set"))
		} else {
			source := entry.Source
			if _, known := settingsRegistry[k]; !known {
				source += ", unknown key"
			}
			_, _ = fmt.Fprintf(w, "%s %s\n", entry.Value, tui.Annotate("%s", source))
		}
	}
}

// showSetting displays a single setting value with its source.
func showSetting(w io.Writer, key string, localOnly bool) error {
	if _, valid := settingsRegistry[key]; !valid {
		return fmt.Errorf("unknown setting %q\n\nValid settings: %s", key, validSettingsKeysString())
	}

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	entries, err := resolveAllSettings(paths, localOnly)
	if err != nil {
		return err
	}

	entry := entries[key]
	_, _ = tui.ColorDim.Fprintf(w, "%s: ", key)
	if entry.Source == "not set" {
		_, _ = fmt.Fprintln(w, tui.Annotate("not set"))
	} else {
		_, _ = fmt.Fprintf(w, "%s %s\n", entry.Value, tui.Annotate("%s", entry.Source))
	}

	return nil
}

// setSetting sets a setting value.
func setSetting(w io.Writer, flags *Flags, key, value string, localOnly bool) error {
	info, valid := settingsRegistry[key]
	if !valid {
		return fmt.Errorf("unknown setting %q\n\nValid settings: %s", key, validSettingsKeysString())
	}

	if info.Type == "int" {
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("setting %q requires an integer value", key)
		}
	}

	// Get config directory
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(localOnly)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Load existing settings (error ignored: missing file means start fresh)
	settings, _ := loadSettingsFromDir(configDir)
	if settings == nil {
		settings = make(map[string]string)
	}

	// Set the value
	settings[key] = value

	// Write settings file
	settingsPath := filepath.Join(configDir, "settings.cue")
	if err := writeSettingsFile(settingsPath, settings); err != nil {
		return fmt.Errorf("writing settings file: %w", err)
	}

	if !flags.Quiet {
		_, _ = fmt.Fprintf(w, "Set %s to %q\n", key, value)
	}

	return nil
}

// editSettings opens the settings file in the user's editor.
func editSettings(localOnly bool) error {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(localOnly)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	settingsPath := filepath.Join(configDir, "settings.cue")

	// Create file if it doesn't exist
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		if err := writeSettingsFile(settingsPath, nil); err != nil {
			return fmt.Errorf("creating settings file: %w", err)
		}
	}

	return openInEditor(settingsPath)
}

// loadSettingsForScope loads settings from the appropriate scope.
// Returns an empty map if no settings are configured.
func loadSettingsForScope(localOnly bool) (map[string]string, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return nil, fmt.Errorf("resolving config paths: %w", err)
	}

	settings := make(map[string]string)

	if localOnly {
		if paths.LocalExists {
			localSettings, err := loadSettingsFromDir(paths.Local)
			if err != nil {
				return nil, err
			}
			for k, v := range localSettings {
				settings[k] = v
			}
		}
	} else {
		if paths.GlobalExists {
			globalSettings, err := loadSettingsFromDir(paths.Global)
			if err != nil {
				return nil, err
			}
			for k, v := range globalSettings {
				settings[k] = v
			}
		}
		if paths.LocalExists {
			localSettings, err := loadSettingsFromDir(paths.Local)
			if err != nil {
				return nil, err
			}
			for k, v := range localSettings {
				settings[k] = v
			}
		}
	}

	return settings, nil
}

// loadSettingsFromDir loads settings from a specific directory.
func loadSettingsFromDir(dir string) (map[string]string, error) {
	settings := make(map[string]string)

	loader := internalcue.NewLoader()
	result, err := loader.Load([]string{dir})
	if err != nil {
		if errors.Is(err, internalcue.ErrNoCUEFiles) {
			return settings, nil
		}
		return nil, err
	}

	// Extract settings
	settingsVal := result.Value.LookupPath(cue.ParsePath(internalcue.KeySettings))
	if !settingsVal.Exists() {
		return settings, nil
	}

	// Iterate over settings fields
	iter, err := settingsVal.Fields(cue.Concrete(true))
	if err != nil {
		return nil, fmt.Errorf("iterating settings: %w", err)
	}

	for iter.Next() {
		key := iter.Selector().Unquoted()

		switch iter.Value().Kind() {
		case cue.StringKind:
			if str, err := iter.Value().String(); err == nil {
				settings[key] = str
			}
		case cue.IntKind:
			if i, err := iter.Value().Int64(); err == nil {
				settings[key] = strconv.FormatInt(i, 10)
			}
		}
	}

	return settings, nil
}

// hasNonSettingsContent checks if CUE content has non-settings top-level keys.
// This prevents accidental data loss when overwriting settings.cue.
func hasNonSettingsContent(content string) bool {
	ctx := cuecontext.New()
	v := ctx.CompileString(content)
	if v.Err() != nil {
		return false
	}

	nonSettingsKeys := []string{
		internalcue.KeyAgents, internalcue.KeyContexts,
		internalcue.KeyRoles, internalcue.KeyTasks,
	}
	for _, key := range nonSettingsKeys {
		if v.LookupPath(cue.ParsePath(key)).Exists() {
			return true
		}
	}
	return false
}

// writeSettingsFile writes the settings to a CUE file.
// It checks for existing non-settings content to prevent data loss.
func writeSettingsFile(path string, settings map[string]string) error {
	// Check if file exists with non-settings content
	if existingContent, err := os.ReadFile(path); err == nil {
		if hasNonSettingsContent(string(existingContent)) {
			return fmt.Errorf("settings.cue contains non-settings content (agents, roles, etc.)\n\nPlease edit the file manually: %s", path)
		}
	}

	var sb strings.Builder

	sb.WriteString("// Auto-generated by start config\n")
	sb.WriteString("// Edit this file to customize your settings\n\n")
	sb.WriteString("settings: {\n")

	if len(settings) > 0 {
		// Sort keys for consistent output
		var keys []string
		for k := range settings {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := settings[k]
			// Check if this is an int setting
			if info, exists := settingsRegistry[k]; exists && info.Type == "int" {
				// Write as integer (no quotes)
				sb.WriteString(fmt.Sprintf("\t%s: %s\n", k, v))
			} else {
				// Write as string (with quotes)
				sb.WriteString(fmt.Sprintf("\t%s: %q\n", k, v))
			}
		}
	}

	sb.WriteString("}\n")

	return os.WriteFile(path, []byte(sb.String()), 0644)
}

// unsetSetting removes a setting key from the settings file.
func unsetSetting(w io.Writer, flags *Flags, key string, localOnly bool) error {
	if _, valid := settingsRegistry[key]; !valid {
		return fmt.Errorf("unknown setting %q\n\nValid settings: %s", key, validSettingsKeysString())
	}

	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	configDir := paths.Dir(localOnly)

	// Only load (and propagate errors from) the settings file if it exists.
	// Absence of the file is not an error — it means nothing is configured.
	settingsPath := filepath.Join(configDir, "settings.cue")
	if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
		if !flags.Quiet {
			_, _ = fmt.Fprintf(w, "%s is not set\n", key)
		}
		return nil
	}

	settings, err := loadSettingsFromDir(configDir)
	if err != nil {
		return fmt.Errorf("loading settings: %w", err)
	}
	if len(settings) == 0 {
		if !flags.Quiet {
			_, _ = fmt.Fprintf(w, "%s is not set\n", key)
		}
		return nil
	}

	if _, exists := settings[key]; !exists {
		if !flags.Quiet {
			_, _ = fmt.Fprintf(w, "%s is not set\n", key)
		}
		return nil
	}

	delete(settings, key)

	if err := writeSettingsFile(settingsPath, settings); err != nil {
		return fmt.Errorf("writing settings file: %w", err)
	}

	if !flags.Quiet {
		_, _ = fmt.Fprintf(w, "Unset %s\n", key)
	}
	return nil
}

// resolveAssetsIndexPath returns the configured assets_index setting value,
// or empty string if not set or on any error. Callers should pass the result
// to registry.EffectiveIndexPath to get the final module path.
func resolveAssetsIndexPath() string {
	settings, err := loadSettingsForScope(false)
	if err != nil {
		return ""
	}
	return settings["assets_index"]
}
