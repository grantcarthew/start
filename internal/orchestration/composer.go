package orchestration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/temp"
)

// ContextSelection specifies which contexts to include.
type ContextSelection struct {
	// IncludeRequired always includes required contexts.
	IncludeRequired bool
	// IncludeDefaults includes default contexts (for `start` command).
	IncludeDefaults bool
	// Tags specifies which tagged contexts to include.
	Tags []string
}

// Context represents a resolved context.
type Context struct {
	Name        string
	Description string
	Content     string
	Required    bool
	Default     bool
	Tags        []string
	File        string // Source file path (if file-based)
	Error       string // Error message if resolution failed
}

// Composer handles prompt composition from CUE configuration.
type Composer struct {
	processor   *TemplateProcessor
	tempManager *temp.Manager
	workingDir  string
}

// NewComposer creates a new prompt composer.
func NewComposer(processor *TemplateProcessor, workingDir string) *Composer {
	return &Composer{
		processor:   processor,
		tempManager: temp.NewUTDManager(workingDir),
		workingDir:  workingDir,
	}
}

// resolveFileToTemp reads a source file and writes it to .start/temp/.
// Returns the temp file path, or empty string if no file to resolve.
// The entityType is "task", "role", or "context".
// The name is the entity name (e.g., "code-review", "start/create-task").
func (c *Composer) resolveFileToTemp(entityType, name, filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading %s file %s: %w", entityType, filePath, err)
	}

	tempPath, err := c.tempManager.WriteUTDFile(entityType, name, string(content))
	if err != nil {
		return "", fmt.Errorf("writing %s temp file: %w", entityType, err)
	}

	return tempPath, nil
}

// ComposeResult contains the result of prompt composition.
type ComposeResult struct {
	// Prompt is the fully composed prompt.
	Prompt string
	// Contexts is the list of contexts that were included.
	Contexts []Context
	// Role is the resolved role content.
	Role string
	// RoleName is the name of the role used.
	RoleName string
	// Warnings contains any non-fatal issues.
	Warnings []string
}

// Compose builds the final prompt from configuration.
func (c *Composer) Compose(cfg cue.Value, selection ContextSelection, customText, instructions string) (ComposeResult, error) {
	var result ComposeResult

	// Get contexts in definition order
	contexts, err := c.selectContexts(cfg, selection)
	if err != nil {
		return result, fmt.Errorf("selecting contexts: %w", err)
	}

	// Check for tags that matched no contexts
	for _, tag := range selection.Tags {
		if tag == "default" {
			continue // pseudo-tag, handled separately
		}
		matched := false
		for _, ctx := range contexts {
			for _, ctxTag := range ctx.Tags {
				if ctxTag == tag {
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			result.Warnings = append(result.Warnings, fmt.Sprintf("tag %q matched no contexts", tag))
		}
	}

	// Resolve each context through UTD
	var promptParts []string
	for _, ctx := range contexts {
		resolved, err := c.resolveContext(cfg, ctx.Name)
		if err != nil {
			// Store error on context for display (per DR-007, errors are non-fatal)
			ctx.Error = err.Error()
		} else {
			ctx.Content = resolved.Content
			if resolved.Content != "" {
				promptParts = append(promptParts, resolved.Content)
			}
		}
		result.Contexts = append(result.Contexts, ctx)
	}

	// Append custom text or task instructions
	if customText != "" {
		promptParts = append(promptParts, customText)
	}

	result.Prompt = strings.Join(promptParts, "\n\n")
	return result, nil
}

// ComposeWithRole composes prompt and resolves role.
func (c *Composer) ComposeWithRole(cfg cue.Value, selection ContextSelection, roleName, customText, instructions string) (ComposeResult, error) {
	result, err := c.Compose(cfg, selection, customText, instructions)
	if err != nil {
		return result, err
	}

	// Resolve role
	if roleName == "" {
		roleName = c.getDefaultRole(cfg)
	}
	result.RoleName = roleName

	if roleName != "" {
		roleContent, err := c.resolveRole(cfg, roleName)
		if err != nil {
			// Role errors are warnings (per DR-007)
			result.Warnings = append(result.Warnings, fmt.Sprintf("role %q: %v", roleName, err))
		} else {
			result.Role = roleContent
		}
	}

	return result, nil
}

// selectContexts returns contexts matching the selection criteria in definition order.
func (c *Composer) selectContexts(cfg cue.Value, selection ContextSelection) ([]Context, error) {
	contextsVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyContexts))
	if !contextsVal.Exists() {
		return nil, nil // No contexts defined is OK
	}

	var contexts []Context
	iter, err := contextsVal.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating contexts: %w", err)
	}

	tagSet := make(map[string]bool)
	for _, tag := range selection.Tags {
		tagSet[tag] = true
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		ctxVal := iter.Value()

		ctx := Context{Name: name}

		// Extract context properties
		if desc := ctxVal.LookupPath(cue.ParsePath("description")); desc.Exists() {
			ctx.Description, _ = desc.String()
		}
		if req := ctxVal.LookupPath(cue.ParsePath("required")); req.Exists() {
			ctx.Required, _ = req.Bool()
		}
		if def := ctxVal.LookupPath(cue.ParsePath("default")); def.Exists() {
			ctx.Default, _ = def.Bool()
		}
		if tags := ctxVal.LookupPath(cue.ParsePath("tags")); tags.Exists() {
			tagIter, err := tags.List()
			if err == nil {
				for tagIter.Next() {
					if s, err := tagIter.Value().String(); err == nil {
						ctx.Tags = append(ctx.Tags, s)
					}
				}
			}
		}
		if file := ctxVal.LookupPath(cue.ParsePath("file")); file.Exists() {
			ctx.File, _ = file.String()
		}

		// Check if context should be included
		include := false

		// Required contexts always included
		if selection.IncludeRequired && ctx.Required {
			include = true
		}

		// Default contexts included if IncludeDefaults is set
		if selection.IncludeDefaults && ctx.Default {
			include = true
		}

		// Tagged contexts included if matching tag in selection
		if len(selection.Tags) > 0 {
			// Special handling for "default" pseudo-tag
			if tagSet["default"] && ctx.Default {
				include = true
			}

			// Check actual tags
			for _, tag := range ctx.Tags {
				if tagSet[tag] {
					include = true
					break
				}
			}
		}

		if include {
			contexts = append(contexts, ctx)
		}
	}

	return contexts, nil
}

// resolveContext resolves a context through UTD processing.
func (c *Composer) resolveContext(cfg cue.Value, name string) (ProcessResult, error) {
	ctxVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyContexts)).LookupPath(cue.MakePath(cue.Str(name)))
	if !ctxVal.Exists() {
		return ProcessResult{}, fmt.Errorf("context not found")
	}

	fields := extractUTDFields(ctxVal)
	if !IsUTDValid(fields) {
		return ProcessResult{}, fmt.Errorf("invalid UTD: no file, command, or prompt")
	}

	// Resolve @module/ paths using origin field (per DR-023)
	if strings.HasPrefix(fields.File, "@module/") {
		origin := extractOrigin(ctxVal)
		if origin != "" {
			resolved, err := resolveModulePath(fields.File, origin)
			if err == nil {
				fields.File = resolved
			}
		}
	}

	// Write file to temp BEFORE processing so {{.file}} gets temp path
	var tempPath string
	if fields.File != "" {
		var err error
		tempPath, err = c.resolveFileToTemp("context", name, fields.File)
		if err != nil {
			return ProcessResult{}, err
		}
		fields.File = tempPath
	}

	result, err := c.processor.Process(fields, "")
	if err != nil {
		return result, err
	}

	result.TempFile = tempPath
	return result, nil
}

// resolveRole resolves a role through UTD processing.
func (c *Composer) resolveRole(cfg cue.Value, name string) (string, error) {
	roleVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyRoles)).LookupPath(cue.MakePath(cue.Str(name)))
	if !roleVal.Exists() {
		return "", fmt.Errorf("role not found")
	}

	fields := extractUTDFields(roleVal)
	if !IsUTDValid(fields) {
		return "", fmt.Errorf("invalid UTD: no file, command, or prompt")
	}

	// Resolve @module/ paths using origin field (per DR-023)
	if strings.HasPrefix(fields.File, "@module/") {
		origin := extractOrigin(roleVal)
		if origin != "" {
			resolved, err := resolveModulePath(fields.File, origin)
			if err == nil {
				fields.File = resolved
			}
		}
	}

	// Write file to temp BEFORE processing so {{.file}} gets temp path
	if fields.File != "" {
		tempPath, err := c.resolveFileToTemp("role", name, fields.File)
		if err != nil {
			return "", err
		}
		fields.File = tempPath
	}

	result, err := c.processor.Process(fields, "")
	if err != nil {
		return "", err
	}

	return result.Content, nil
}

// getDefaultRole returns the default role name from config.
func (c *Composer) getDefaultRole(cfg cue.Value) string {
	// Check settings.default_role
	if def := cfg.LookupPath(cue.ParsePath(internalcue.KeySettings + ".default_role")); def.Exists() {
		if s, err := def.String(); err == nil {
			return s
		}
	}

	// Fall back to first role in definition order
	roles := cfg.LookupPath(cue.ParsePath(internalcue.KeyRoles))
	if roles.Exists() {
		iter, err := roles.Fields()
		if err == nil && iter.Next() {
			return iter.Selector().Unquoted()
		}
	}

	return ""
}

// extractUTDFields extracts UTD fields from a CUE value.
func extractUTDFields(v cue.Value) UTDFields {
	var fields UTDFields

	if file := v.LookupPath(cue.ParsePath("file")); file.Exists() {
		fields.File, _ = file.String()
	}
	if cmd := v.LookupPath(cue.ParsePath("command")); cmd.Exists() {
		fields.Command, _ = cmd.String()
	}
	if prompt := v.LookupPath(cue.ParsePath("prompt")); prompt.Exists() {
		fields.Prompt, _ = prompt.String()
	}
	if shell := v.LookupPath(cue.ParsePath("shell")); shell.Exists() {
		fields.Shell, _ = shell.String()
	}
	if timeout := v.LookupPath(cue.ParsePath("timeout")); timeout.Exists() {
		if i, err := timeout.Int64(); err == nil {
			fields.Timeout = int(i)
		}
	}

	return fields
}

// ResolveTask resolves a task by name and processes its UTD.
func (c *Composer) ResolveTask(cfg cue.Value, name, instructions string) (ProcessResult, error) {
	taskVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyTasks)).LookupPath(cue.MakePath(cue.Str(name)))
	if !taskVal.Exists() {
		return ProcessResult{}, fmt.Errorf("task %q not found", name)
	}

	fields := extractUTDFields(taskVal)
	if !IsUTDValid(fields) {
		return ProcessResult{}, fmt.Errorf("invalid UTD: no file, command, or prompt")
	}

	// Resolve @module/ paths using origin field (per DR-023)
	if strings.HasPrefix(fields.File, "@module/") {
		origin := extractOrigin(taskVal)
		if origin != "" {
			resolved, err := resolveModulePath(fields.File, origin)
			if err == nil {
				fields.File = resolved
			}
			// If resolution fails, keep original path (will produce clearer error later)
		}
	}

	// Write file to temp BEFORE processing so {{.file}} gets temp path
	var tempPath string
	if fields.File != "" {
		var err error
		tempPath, err = c.resolveFileToTemp("task", name, fields.File)
		if err != nil {
			return ProcessResult{}, err
		}
		fields.File = tempPath
	}

	result, err := c.processor.Process(fields, instructions)
	if err != nil {
		return result, err
	}

	result.TempFile = tempPath
	return result, nil
}

// extractOrigin extracts the origin field from a CUE value.
func extractOrigin(v cue.Value) string {
	if origin := v.LookupPath(cue.ParsePath("origin")); origin.Exists() {
		if s, err := origin.String(); err == nil {
			return s
		}
	}
	return ""
}

// resolveModulePath resolves an @module/ path to the CUE cache location.
// Per DR-023, @module/ paths resolve relative to the cached module directory.
func resolveModulePath(path, origin string) (string, error) {
	if !strings.HasPrefix(path, "@module/") {
		return path, nil
	}

	// Strip @module/ prefix
	relativePath := strings.TrimPrefix(path, "@module/")

	// Get CUE cache directory
	cacheDir, err := getCUECacheDir()
	if err != nil {
		return "", fmt.Errorf("getting CUE cache dir: %w", err)
	}

	// Origin format: "github.com/grantcarthew/start-assets/tasks/golang/code-review"
	// Module path in cache: cacheDir/mod/extract/github.com/grantcarthew/start-assets/tasks/golang/code-review@v0.x.x/
	// We need to find the version directory
	moduleBase := filepath.Join(cacheDir, "mod", "extract", origin)

	// Find version directory (there should be one matching @v*)
	entries, err := os.ReadDir(filepath.Dir(moduleBase))
	if err != nil {
		return "", fmt.Errorf("reading cache directory: %w", err)
	}

	baseName := filepath.Base(origin)
	var moduleDir string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), baseName+"@v") {
			moduleDir = filepath.Join(filepath.Dir(moduleBase), entry.Name())
			break
		}
	}

	if moduleDir == "" {
		return "", fmt.Errorf("module %s not found in cache", origin)
	}

	return filepath.Join(moduleDir, relativePath), nil
}

// getCUECacheDir returns the CUE cache directory.
// Respects CUE_CACHE_DIR environment variable.
func getCUECacheDir() (string, error) {
	if dir := os.Getenv("CUE_CACHE_DIR"); dir != "" {
		return dir, nil
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "cue"), nil
}

// GetTaskRole returns the role specified for a task.
func GetTaskRole(cfg cue.Value, taskName string) string {
	taskVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyTasks)).LookupPath(cue.MakePath(cue.Str(taskName)))
	if !taskVal.Exists() {
		return ""
	}

	if role := taskVal.LookupPath(cue.ParsePath("role")); role.Exists() {
		if s, err := role.String(); err == nil {
			return s
		}
	}

	return ""
}
