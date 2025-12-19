package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
)

// CheckIntro returns the intro section with repository info.
func CheckIntro() SectionResult {
	return SectionResult{
		Name:    "Repository",
		NoIcons: true,
		Results: []CheckResult{
			{Status: StatusInfo, Label: RepoURL},
			{Status: StatusInfo, Label: IssuesURL},
		},
	}
}

// CheckVersion returns the version section with build info.
func CheckVersion(info BuildInfo) SectionResult {
	return SectionResult{
		Name:    "Version",
		NoIcons: true,
		Results: []CheckResult{
			{Status: StatusInfo, Label: "start " + info.Version},
			{Status: StatusInfo, Label: "Commit", Message: info.Commit},
			{Status: StatusInfo, Label: "Built", Message: info.BuildDate},
			{Status: StatusInfo, Label: "Go", Message: info.GoVersion},
			{Status: StatusInfo, Label: "Platform", Message: info.Platform},
		},
	}
}

// CheckConfiguration validates CUE configuration files.
func CheckConfiguration(paths config.Paths) SectionResult {
	section := SectionResult{Name: "Configuration"}

	// Check global config
	globalResults := checkConfigDir(paths.Global, "Global", paths.GlobalExists)
	section.Results = append(section.Results, globalResults...)

	// Check local config
	localResults := checkConfigDir(paths.Local, "Local", paths.LocalExists)
	section.Results = append(section.Results, localResults...)

	// If both exist, try to load and merge
	if paths.GlobalExists || paths.LocalExists {
		loader := internalcue.NewLoader()
		dirs := paths.ForScope(config.ScopeMerged)
		_, err := loader.Load(dirs)
		if err != nil {
			section.Results = append(section.Results, CheckResult{
				Status:  StatusFail,
				Label:   "Merge",
				Message: "Failed",
				Fix:     fmt.Sprintf("Fix CUE syntax: %v", err),
			})
		} else {
			section.Results = append(section.Results, CheckResult{
				Status:  StatusPass,
				Label:   "Validation",
				Message: "Valid",
			})
		}
	}

	return section
}

// checkConfigDir checks a single configuration directory.
func checkConfigDir(dir, scope string, exists bool) []CheckResult {
	var results []CheckResult

	// Create scope label
	shortDir := shortenPath(dir)
	scopeLabel := fmt.Sprintf("%s (%s)", scope, shortDir)

	if !exists {
		results = append(results, CheckResult{
			Status:  StatusInfo,
			Label:   scopeLabel,
			Message: "Not found",
		})
		return results
	}

	// List CUE files in directory
	entries, err := os.ReadDir(dir)
	if err != nil {
		results = append(results, CheckResult{
			Status:  StatusFail,
			Label:   scopeLabel,
			Message: fmt.Sprintf("Cannot read: %v", err),
		})
		return results
	}

	var cueFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".cue") {
			cueFiles = append(cueFiles, entry.Name())
		}
	}

	if len(cueFiles) == 0 {
		results = append(results, CheckResult{
			Status:  StatusInfo,
			Label:   scopeLabel,
			Message: "No CUE files",
		})
		return results
	}

	// Try to load the directory to validate CUE syntax
	loader := internalcue.NewLoader()
	_, err = loader.LoadSingle(dir)
	if err != nil {
		// Report the directory as having errors
		results = append(results, CheckResult{
			Status:  StatusFail,
			Label:   scopeLabel,
			Message: fmt.Sprintf("Invalid: %v", err),
			Fix:     "Fix CUE syntax errors in this directory",
		})
	} else {
		// Add header first (with colon), then list each file as valid
		results = append(results, CheckResult{
			Status: StatusInfo,
			Label:  scopeLabel + ":",
		})
		for _, f := range cueFiles {
			results = append(results, CheckResult{
				Status: StatusPass,
				Label:  "  " + f,
			})
		}
	}

	return results
}

// CheckAgents validates configured agent binaries are available.
func CheckAgents(cfgValue cue.Value) SectionResult {
	section := SectionResult{Name: "Agents"}

	agents := cfgValue.LookupPath(cue.ParsePath(internalcue.KeyAgents))
	if !agents.Exists() {
		section.Results = append(section.Results, CheckResult{
			Status:  StatusInfo,
			Label:   "None configured",
			Message: "",
		})
		return section
	}

	iter, err := agents.Fields()
	if err != nil {
		section.Results = append(section.Results, CheckResult{
			Status:  StatusFail,
			Label:   "Error",
			Message: fmt.Sprintf("Cannot read agents: %v", err),
		})
		return section
	}

	count := 0
	for iter.Next() {
		count++
		name := iter.Selector().Unquoted()
		agent := iter.Value()

		binVal := agent.LookupPath(cue.ParsePath("bin"))
		if !binVal.Exists() {
			section.Results = append(section.Results, CheckResult{
				Status:  StatusWarn,
				Label:   name,
				Message: "No bin field",
			})
			continue
		}

		bin, err := binVal.String()
		if err != nil {
			section.Results = append(section.Results, CheckResult{
				Status:  StatusWarn,
				Label:   name,
				Message: "Invalid bin field",
			})
			continue
		}

		path, err := exec.LookPath(bin)
		if err != nil {
			section.Results = append(section.Results, CheckResult{
				Status:  StatusFail,
				Label:   name,
				Message: "NOT FOUND",
				Fix:     fmt.Sprintf("Install %s or remove from config", bin),
			})
		} else {
			section.Results = append(section.Results, CheckResult{
				Status:  StatusPass,
				Label:   name,
				Message: path,
			})
		}
	}

	if count > 0 {
		section.Summary = fmt.Sprintf("%d configured", count)
	}

	return section
}

// CheckContexts validates configured context files exist.
func CheckContexts(cfgValue cue.Value) SectionResult {
	section := SectionResult{Name: "Contexts"}

	contexts := cfgValue.LookupPath(cue.ParsePath(internalcue.KeyContexts))
	if !contexts.Exists() {
		section.Results = append(section.Results, CheckResult{
			Status:  StatusInfo,
			Label:   "None configured",
			Message: "",
		})
		return section
	}

	iter, err := contexts.Fields()
	if err != nil {
		section.Results = append(section.Results, CheckResult{
			Status:  StatusFail,
			Label:   "Error",
			Message: fmt.Sprintf("Cannot read contexts: %v", err),
		})
		return section
	}

	count := 0
	for iter.Next() {
		count++
		name := iter.Selector().Unquoted()
		ctx := iter.Value()

		result := checkFileField(ctx, name)
		if result != nil {
			// Check if this is a required context
			required := false
			if req := ctx.LookupPath(cue.ParsePath("required")); req.Exists() {
				required, _ = req.Bool()
			}

			// Downgrade to warning for optional contexts
			if result.Status == StatusFail && !required {
				result.Status = StatusWarn
			}

			section.Results = append(section.Results, *result)
		}
	}

	if count > 0 {
		section.Summary = fmt.Sprintf("%d configured", count)
	}

	return section
}

// CheckRoles validates configured role files exist.
func CheckRoles(cfgValue cue.Value) SectionResult {
	section := SectionResult{Name: "Roles"}

	roles := cfgValue.LookupPath(cue.ParsePath(internalcue.KeyRoles))
	if !roles.Exists() {
		section.Results = append(section.Results, CheckResult{
			Status:  StatusInfo,
			Label:   "None configured",
			Message: "",
		})
		return section
	}

	iter, err := roles.Fields()
	if err != nil {
		section.Results = append(section.Results, CheckResult{
			Status:  StatusFail,
			Label:   "Error",
			Message: fmt.Sprintf("Cannot read roles: %v", err),
		})
		return section
	}

	count := 0
	for iter.Next() {
		count++
		name := iter.Selector().Unquoted()
		role := iter.Value()

		result := checkFileField(role, name)
		if result != nil {
			section.Results = append(section.Results, *result)
		}
	}

	if count > 0 {
		section.Summary = fmt.Sprintf("%d configured", count)
	}

	return section
}

// checkFileField checks if a config item has a valid file field.
func checkFileField(v cue.Value, name string) *CheckResult {
	fileVal := v.LookupPath(cue.ParsePath("file"))
	if !fileVal.Exists() {
		// No file field - check for prompt or command instead
		if prompt := v.LookupPath(cue.ParsePath("prompt")); prompt.Exists() {
			return &CheckResult{
				Status:  StatusPass,
				Label:   name,
				Message: "(inline prompt)",
			}
		}
		if cmd := v.LookupPath(cue.ParsePath("command")); cmd.Exists() {
			return &CheckResult{
				Status:  StatusPass,
				Label:   name,
				Message: "(command)",
			}
		}
		return &CheckResult{
			Status:  StatusWarn,
			Label:   name,
			Message: "No file, prompt, or command",
		}
	}

	filePath, err := fileVal.String()
	if err != nil {
		return &CheckResult{
			Status:  StatusWarn,
			Label:   name,
			Message: "Invalid file field",
		}
	}

	// Expand ~ to home directory
	expandedPath := expandPath(filePath)

	if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
		return &CheckResult{
			Status:  StatusFail,
			Label:   name,
			Message: fmt.Sprintf("%s (not found)", shortenPath(filePath)),
			Fix:     fmt.Sprintf("Create file or update path"),
		}
	} else if err != nil {
		return &CheckResult{
			Status:  StatusFail,
			Label:   name,
			Message: fmt.Sprintf("%s (%v)", shortenPath(filePath), err),
		}
	}

	return &CheckResult{
		Status:  StatusPass,
		Label:   name,
		Message: shortenPath(filePath),
	}
}

// CheckEnvironment validates runtime environment.
func CheckEnvironment(paths config.Paths) SectionResult {
	section := SectionResult{Name: "Environment"}

	// Check config directory is writable (if it exists)
	if paths.GlobalExists {
		if isWritable(paths.Global) {
			section.Results = append(section.Results, CheckResult{
				Status:  StatusPass,
				Label:   "Config directory",
				Message: "writable",
			})
		} else {
			section.Results = append(section.Results, CheckResult{
				Status:  StatusFail,
				Label:   "Config directory",
				Message: "not writable",
				Fix:     fmt.Sprintf("Check permissions on %s", paths.Global),
			})
		}
	}

	// Check working directory is accessible
	wd, err := os.Getwd()
	if err != nil {
		section.Results = append(section.Results, CheckResult{
			Status:  StatusFail,
			Label:   "Working directory",
			Message: fmt.Sprintf("Cannot access: %v", err),
		})
	} else {
		section.Results = append(section.Results, CheckResult{
			Status:  StatusPass,
			Label:   "Working directory",
			Message: shortenPath(wd),
		})
	}

	return section
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// shortenPath replaces home directory with ~.
func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// isWritable checks if a directory is writable.
func isWritable(dir string) bool {
	testFile := filepath.Join(dir, ".write-test")
	f, err := os.Create(testFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(testFile)
	return true
}
