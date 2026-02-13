package cli

import (
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
	"github.com/spf13/cobra"
)

// Valid settings keys
var validSettingsKeys = map[string]string{
	"default_agent": "string",
	"shell":         "string",
	"timeout":       "int",
}

// addConfigSettingsCommand adds the settings subcommand to config.
func addConfigSettingsCommand(parent *cobra.Command) {
	settingsCmd := &cobra.Command{
		Use:     "settings [key] [value]",
		Aliases: []string{"setting"},
		Short:   "Manage settings configuration",
		Long: `Manage settings for start.

Available settings:
  default_agent  Agent to use when --agent not specified
  shell          Shell for command execution (default: auto-detect)
  timeout        Command timeout in seconds`,
		Example: `  start config settings              List all settings
  start config settings <key>        Show a setting value
  start config settings <key> <val>  Set a setting value
  start config settings edit         Open settings.cue in $EDITOR`,
		Args: cobra.MaximumNArgs(2),
		RunE: executeConfigSettings,
	}

	parent.AddCommand(settingsCmd)
}

// executeConfigSettings handles the settings command.
func executeConfigSettings(cmd *cobra.Command, args []string) error {
	if shown, err := checkHelpArg(cmd, args); shown || err != nil {
		return err
	}

	stdout := cmd.OutOrStdout()
	flags := getFlags(cmd)
	local := getFlags(cmd).Local

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

// listSettings displays all settings.
func listSettings(w io.Writer, localOnly bool) error {
	settings, err := loadSettingsForScope(localOnly)
	if err != nil {
		// No settings is fine, just show empty
		if strings.Contains(err.Error(), "no config found") {
			_, _ = fmt.Fprintln(w, "No settings configured")
			return nil
		}
		return err
	}

	if len(settings) == 0 {
		_, _ = fmt.Fprintln(w, "No settings configured")
		return nil
	}

	// Sort keys for consistent output
	var keys []string
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		_, _ = colorDim.Fprintf(w, "%s: ", k)
		_, _ = fmt.Fprintln(w, settings[k])
	}

	return nil
}

// showSetting displays a single setting value.
func showSetting(w io.Writer, key string, localOnly bool) error {
	// Validate key
	if _, valid := validSettingsKeys[key]; !valid {
		return fmt.Errorf("unknown setting %q\n\nValid settings: default_agent, shell, timeout", key)
	}

	settings, err := loadSettingsForScope(localOnly)
	if err != nil {
		if strings.Contains(err.Error(), "no config found") {
			_, _ = colorDim.Fprintf(w, "%s: ", key)
			_, _ = colorCyan.Fprint(w, "(")
			_, _ = colorDim.Fprint(w, "not set")
			_, _ = colorCyan.Fprintln(w, ")")
			return nil
		}
		return err
	}

	if val, exists := settings[key]; exists {
		_, _ = colorDim.Fprintf(w, "%s: ", key)
		_, _ = fmt.Fprintln(w, val)
	} else {
		_, _ = colorDim.Fprintf(w, "%s: ", key)
		_, _ = colorCyan.Fprint(w, "(")
		_, _ = colorDim.Fprint(w, "not set")
		_, _ = colorCyan.Fprintln(w, ")")
	}

	return nil
}

// setSetting sets a setting value.
func setSetting(w io.Writer, flags *Flags, key, value string, localOnly bool) error {
	// Validate key
	keyType, valid := validSettingsKeys[key]
	if !valid {
		return fmt.Errorf("unknown setting %q\n\nValid settings: default_agent, shell, timeout", key)
	}

	// Validate value type
	if keyType == "int" {
		if _, err := strconv.Atoi(value); err != nil {
			return fmt.Errorf("setting %q requires an integer value", key)
		}
	}

	// Get config directory
	paths, err := config.ResolvePaths("")
	if err != nil {
		return fmt.Errorf("resolving config paths: %w", err)
	}

	var configDir string
	if localOnly {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Load existing settings
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

	var configDir string
	if localOnly {
		configDir = paths.Local
	} else {
		configDir = paths.Global
	}

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
func loadSettingsForScope(localOnly bool) (map[string]string, error) {
	paths, err := config.ResolvePaths("")
	if err != nil {
		return nil, fmt.Errorf("resolving config paths: %w", err)
	}

	settings := make(map[string]string)

	if localOnly {
		// Local only
		if paths.LocalExists {
			localSettings, err := loadSettingsFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for k, v := range localSettings {
				settings[k] = v
			}
		}
	} else {
		// Merged: global first, then local overrides
		if paths.GlobalExists {
			globalSettings, err := loadSettingsFromDir(paths.Global)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for k, v := range globalSettings {
				settings[k] = v
			}
		}
		if paths.LocalExists {
			localSettings, err := loadSettingsFromDir(paths.Local)
			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
			for k, v := range localSettings {
				settings[k] = v
			}
		}
	}

	if len(settings) == 0 && !paths.GlobalExists && !paths.LocalExists {
		return nil, fmt.Errorf("no config found")
	}

	return settings, nil
}

// loadSettingsFromDir loads settings from a specific directory.
func loadSettingsFromDir(dir string) (map[string]string, error) {
	settings := make(map[string]string)

	loader := internalcue.NewLoader()
	result, err := loader.Load([]string{dir})
	if err != nil {
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
		key := iter.Selector().String()

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
			if keyType, exists := validSettingsKeys[k]; exists && keyType == "int" {
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
