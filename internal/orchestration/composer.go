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

// RoleResolution tracks the resolution status of a role during fallback.
type RoleResolution struct {
	Name     string // Role name (map key or file path)
	Status   string // "loaded", "skipped", "error"
	File     string // Source file path (if file-based)
	Optional bool   // Whether this role is optional
	Error    string // Error message if resolution failed
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

// isCwdPath returns true if the file path is within the working directory (cwd).
// Files within cwd don't need to be copied to temp - they're already accessible.
// A file is within cwd if:
//   - It's a relative path (e.g., "AGENTS.md", "./docs/file.md")
//   - It's an absolute path that's a child of the working directory
func (c *Composer) isCwdPath(filePath string) bool {
	if filePath == "" {
		return false
	}

	// Relative paths are local
	if !filepath.IsAbs(filePath) {
		return true
	}

	// Absolute paths are local if they're under the working directory
	// Clean both paths to normalize them
	cleanPath := filepath.Clean(filePath)
	cleanWorkDir := filepath.Clean(c.workingDir)

	// Check if the file path starts with the working directory
	return strings.HasPrefix(cleanPath, cleanWorkDir+string(filepath.Separator))
}

// ComposeResult contains the result of prompt composition.
type ComposeResult struct {
	// Prompt is the fully composed prompt.
	Prompt string
	// Contexts is the list of contexts that were included.
	Contexts []Context
	// Role is the resolved role content.
	Role string
	// RoleFile is the path to the role file (original for cwd files, temp for external/inline).
	RoleFile string
	// RoleName is the name of the role used.
	RoleName string
	// RoleResolutions tracks all roles checked during resolution for UI display.
	RoleResolutions []RoleResolution
	// Warnings contains any non-fatal issues.
	Warnings []string
}

// Compose builds the final prompt from configuration.
func (c *Composer) Compose(cfg cue.Value, selection ContextSelection, customText, instructions string) (ComposeResult, error) {
	var result ComposeResult
	var promptParts []string
	addedContexts := make(map[string]bool)

	// Helper to resolve and add a config context
	addConfigContext := func(ctx Context) {
		if addedContexts[ctx.Name] {
			return
		}
		addedContexts[ctx.Name] = true

		resolved, err := c.resolveContext(cfg, ctx.Name)
		if err != nil {
			ctx.Error = err.Error()
		} else {
			ctx.Content = resolved.Content
			if resolved.Content != "" {
				promptParts = append(promptParts, strings.TrimRight(resolved.Content, "\n"))
			}
		}
		result.Contexts = append(result.Contexts, ctx)
	}

	// First: add required contexts (config definition order)
	if selection.IncludeRequired {
		requiredSelection := ContextSelection{IncludeRequired: true}
		contexts, err := c.selectContexts(cfg, requiredSelection)
		if err != nil {
			return result, fmt.Errorf("selecting contexts: %w", err)
		}
		for _, ctx := range contexts {
			addConfigContext(ctx)
		}
	}

	// Second: add default contexts if IncludeDefaults and no explicit tags
	if selection.IncludeDefaults && len(selection.Tags) == 0 {
		defaultSelection := ContextSelection{IncludeDefaults: true}
		contexts, err := c.selectContexts(cfg, defaultSelection)
		if err != nil {
			return result, fmt.Errorf("selecting contexts: %w", err)
		}
		for _, ctx := range contexts {
			addConfigContext(ctx)
		}
	}

	// Third: process user tags in order (per DR-038, order is preserved)
	for _, tag := range selection.Tags {
		if IsFilePath(tag) {
			// File path - create context directly
			ctx := Context{
				Name: tag,
				File: tag,
			}
			content, err := ReadFilePath(tag)
			if err != nil {
				ctx.Error = err.Error()
			} else {
				ctx.Content = content
				if content != "" {
					promptParts = append(promptParts, strings.TrimRight(content, "\n"))
				}
			}
			result.Contexts = append(result.Contexts, ctx)
		} else if tag == "default" {
			// "default" pseudo-tag - add default contexts (config order)
			defaultSelection := ContextSelection{IncludeDefaults: true}
			contexts, _ := c.selectContexts(cfg, defaultSelection)
			for _, ctx := range contexts {
				addConfigContext(ctx)
			}
		} else {
			// Try exact context name match first (from search resolution)
			ctxVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyContexts))
			if ctxVal.Exists() && ctxVal.LookupPath(cue.MakePath(cue.Str(tag))).Exists() {
				ctx := Context{Name: tag}
				addConfigContext(ctx)
			} else {
				// Fall back to tag matching
				tagSelection := ContextSelection{Tags: []string{tag}}
				contexts, _ := c.selectContexts(cfg, tagSelection)
				if len(contexts) == 0 {
					result.Warnings = append(result.Warnings, fmt.Sprintf("context %q not found", tag))
				}
				for _, ctx := range contexts {
					addConfigContext(ctx)
				}
			}
		}
	}

	// Append custom text or task instructions
	if customText != "" {
		promptParts = append(promptParts, strings.TrimRight(customText, "\n"))
	}

	result.Prompt = strings.Join(promptParts, "\n\n")
	return result, nil
}

// ComposeWithRole composes prompt and resolves role.
// When roleName is provided (explicit --role), errors are fatal.
// When using default selection, optional roles are skipped gracefully.
func (c *Composer) ComposeWithRole(cfg cue.Value, selection ContextSelection, roleName, customText, instructions string) (ComposeResult, error) {
	result, err := c.Compose(cfg, selection, customText, instructions)
	if err != nil {
		return result, err
	}

	// Track whether this is an explicit role selection
	explicitRole := roleName != ""

	// Resolve role
	if roleName == "" {
		var resolutions []RoleResolution
		var selectErr error
		roleName, resolutions, selectErr = c.selectDefaultRole(cfg)
		result.RoleResolutions = resolutions

		if selectErr != nil {
			// Selection failed (required role missing or all optional roles skipped)
			return result, selectErr
		}
	}
	result.RoleName = roleName

	if roleName != "" {
		var roleContent string
		var roleFilePath string
		var roleErr error

		// Check if roleName is a file path (per DR-038)
		if IsFilePath(roleName) {
			roleContent, roleErr = ReadFilePath(roleName)
			if roleErr == nil {
				// For file path roles, use the expanded path
				roleFilePath, _ = ExpandFilePath(roleName)
			}
			// Add resolution tracking for file path roles
			res := RoleResolution{
				Name: roleName,
				File: roleName,
			}
			if roleErr != nil {
				res.Status = "error"
				res.Error = roleErr.Error()
			} else {
				res.Status = "loaded"
			}
			result.RoleResolutions = append(result.RoleResolutions, res)
		} else {
			roleContent, roleFilePath, roleErr = c.resolveRole(cfg, roleName)

			// Add resolution tracking for config roles (if not already tracked)
			if len(result.RoleResolutions) == 0 || result.RoleResolutions[len(result.RoleResolutions)-1].Name != roleName {
				res := RoleResolution{
					Name: roleName,
					File: roleFilePath,
				}
				if roleErr != nil {
					res.Status = "error"
					res.Error = roleErr.Error()
				} else {
					res.Status = "loaded"
				}
				result.RoleResolutions = append(result.RoleResolutions, res)
			}
		}

		if roleErr != nil {
			if explicitRole {
				// Explicit --role or settings.default_role: always error (per DR-039)
				return result, fmt.Errorf("role %q: %w", roleName, roleErr)
			}
			// This shouldn't happen for default selection (selectDefaultRole already checked)
			// but handle it defensively
			result.Warnings = append(result.Warnings, fmt.Sprintf("role %q: %v", roleName, roleErr))
		} else {
			result.Role = roleContent
			result.RoleFile = roleFilePath
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

	// Write file to temp for agent access (only for external files).
	// Files within cwd are already accessible - no temp copy needed.
	var tempPath string
	if fields.File != "" {
		if c.isCwdPath(fields.File) {
			// Expand tilde and validate cwd file exists (don't copy, just check)
			expandedPath, err := ExpandFilePath(fields.File)
			if err != nil {
				return ProcessResult{}, fmt.Errorf("expanding context file path %s: %w", fields.File, err)
			}
			if _, err := os.Stat(expandedPath); err != nil {
				return ProcessResult{}, fmt.Errorf("reading context file %s: %w", fields.File, err)
			}
			fields.File = expandedPath
		} else {
			var err error
			tempPath, err = c.resolveFileToTemp("context", name, fields.File)
			if err != nil {
				return ProcessResult{}, err
			}
			fields.File = tempPath
		}
	}

	result, err := c.processor.Process(fields, "")
	if err != nil {
		return result, err
	}

	result.TempFile = tempPath
	return result, nil
}

// resolveRole resolves a role through UTD processing.
// Returns the resolved content and the file path where the content can be read.
// For file-based roles: returns original path (cwd) or temp path (external).
// For inline roles (prompt/command): writes content to temp and returns temp path.
func (c *Composer) resolveRole(cfg cue.Value, name string) (content, filePath string, err error) {
	roleVal := cfg.LookupPath(cue.ParsePath(internalcue.KeyRoles)).LookupPath(cue.MakePath(cue.Str(name)))
	if !roleVal.Exists() {
		return "", "", fmt.Errorf("role not found")
	}

	fields := extractUTDFields(roleVal)
	if !IsUTDValid(fields) {
		return "", "", fmt.Errorf("invalid UTD: no file, command, or prompt")
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

	// Track the file path for {{.role_file}} placeholder (per DR-020).
	// For file-based roles: use original path (cwd) or temp path (external).
	// For inline roles: will write to temp after processing.
	var roleFilePath string

	if fields.File != "" {
		if c.isCwdPath(fields.File) {
			// Expand tilde and validate cwd file exists (don't copy, just check)
			expandedPath, err := ExpandFilePath(fields.File)
			if err != nil {
				return "", "", fmt.Errorf("expanding role file path %s: %w", fields.File, err)
			}
			if _, err := os.Stat(expandedPath); err != nil {
				return "", "", fmt.Errorf("reading role file %s: %w", fields.File, err)
			}
			fields.File = expandedPath
			roleFilePath = expandedPath
		} else {
			tempPath, err := c.resolveFileToTemp("role", name, fields.File)
			if err != nil {
				return "", "", err
			}
			fields.File = tempPath
			roleFilePath = tempPath
		}
	}

	result, err := c.processor.Process(fields, "")
	if err != nil {
		return "", "", err
	}

	// For inline roles (no source file), write resolved content to temp.
	// This ensures {{.role_file}} always has a valid path for agents that need it.
	if roleFilePath == "" && result.Content != "" {
		tempPath, err := c.tempManager.WriteUTDFile("role", name, result.Content)
		if err != nil {
			return "", "", fmt.Errorf("writing role temp file: %w", err)
		}
		roleFilePath = tempPath
	}

	return result.Content, roleFilePath, nil
}

// selectDefaultRole returns the default role name and resolution tracking.
// When settings.default_role is set, that role is used (no fallback).
// Otherwise, roles are checked in definition order:
// - Optional roles with missing files are skipped
// - Required roles with missing files cause an error
// - First available role is selected
// Returns empty roleName with nil error if no roles are defined.
// Returns error if all roles fail or a required role fails.
func (c *Composer) selectDefaultRole(cfg cue.Value) (roleName string, resolutions []RoleResolution, err error) {
	// Check settings.default_role
	if def := cfg.LookupPath(cue.ParsePath(internalcue.KeySettings + ".default_role")); def.Exists() {
		if s, err := def.String(); err == nil && s != "" {
			// Explicit default_role - return it directly (no fallback)
			// The caller will handle errors (per DR-039)
			return s, nil, nil
		}
	}

	// Iterate through roles in definition order
	roles := cfg.LookupPath(cue.ParsePath(internalcue.KeyRoles))
	if !roles.Exists() {
		return "", nil, nil
	}

	iter, err := roles.Fields()
	if err != nil {
		return "", nil, fmt.Errorf("iterating roles: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		roleVal := iter.Value()

		// Extract optional field (default: false)
		optional := false
		if opt := roleVal.LookupPath(cue.ParsePath("optional")); opt.Exists() {
			optional, _ = opt.Bool()
		}

		// Extract file field
		var filePath string
		if file := roleVal.LookupPath(cue.ParsePath("file")); file.Exists() {
			filePath, _ = file.String()
		}

		// Check if role is available
		available := true
		var checkErr string

		if filePath != "" {
			// File-based role - check if file exists
			expandedPath, err := ExpandFilePath(filePath)
			if err != nil {
				available = false
				checkErr = fmt.Sprintf("expanding path: %v", err)
			} else if _, err := os.Stat(expandedPath); err != nil {
				available = false
				checkErr = "file not found"
			}
		}
		// Non-file roles (command/prompt only) are always available at selection time

		res := RoleResolution{
			Name:     name,
			File:     filePath,
			Optional: optional,
		}

		if available {
			res.Status = "loaded"
			resolutions = append(resolutions, res)
			return name, resolutions, nil
		}

		// Role not available
		if optional {
			res.Status = "skipped"
			res.Error = checkErr
			resolutions = append(resolutions, res)
			continue
		}

		// Required role failed
		res.Status = "error"
		res.Error = checkErr
		resolutions = append(resolutions, res)
		return "", resolutions, fmt.Errorf("role %q: %s", name, checkErr)
	}

	// All roles exhausted
	if len(resolutions) > 0 {
		return "", resolutions, fmt.Errorf("no valid roles found (all optional roles skipped)")
	}

	return "", nil, nil
}

// getDefaultRole returns the default role name from config.
// Deprecated: Use selectDefaultRole for optional role support.
func (c *Composer) getDefaultRole(cfg cue.Value) string {
	roleName, _, _ := c.selectDefaultRole(cfg)
	return roleName
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

	// Write file to temp for agent access (only for external files).
	// Files within cwd are already accessible - no temp copy needed.
	var tempPath string
	if fields.File != "" {
		if c.isCwdPath(fields.File) {
			// Expand tilde and validate cwd file exists (don't copy, just check)
			expandedPath, err := ExpandFilePath(fields.File)
			if err != nil {
				return ProcessResult{}, fmt.Errorf("expanding task file path %s: %w", fields.File, err)
			}
			if _, err := os.Stat(expandedPath); err != nil {
				return ProcessResult{}, fmt.Errorf("reading task file %s: %w", fields.File, err)
			}
			fields.File = expandedPath
		} else {
			var err error
			tempPath, err = c.resolveFileToTemp("task", name, fields.File)
			if err != nil {
				return ProcessResult{}, err
			}
			fields.File = tempPath
		}
	}

	result, err := c.processor.Process(fields, instructions)
	if err != nil {
		return result, err
	}

	result.TempFile = tempPath
	return result, nil
}

// ProcessContent processes raw content through template substitution.
// This is used for file-based tasks where the content is read directly
// but still needs template processing for placeholders like {{.instructions}}.
func (c *Composer) ProcessContent(content, instructions string) (ProcessResult, error) {
	fields := UTDFields{
		Prompt: content, // Use prompt field so content is treated as template
	}
	return c.processor.Process(fields, instructions)
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

	// Strip version from origin if present
	// Origin format: "github.com/grantcarthew/start-assets/tasks/golang/code-review@v0.0.2"
	// We need just: "github.com/grantcarthew/start-assets/tasks/golang/code-review"
	originWithoutVersion := origin
	if idx := strings.Index(origin, "@"); idx != -1 {
		originWithoutVersion = origin[:idx]
	}

	// Module path in cache: cacheDir/mod/extract/github.com/grantcarthew/start-assets/tasks/golang/code-review@v0.x.x/
	// We need to find the version directory
	moduleBase := filepath.Join(cacheDir, "mod", "extract", originWithoutVersion)

	// Find version directory (there should be one matching @v*)
	entries, err := os.ReadDir(filepath.Dir(moduleBase))
	if err != nil {
		return "", fmt.Errorf("reading cache directory: %w", err)
	}

	baseName := filepath.Base(originWithoutVersion)
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
