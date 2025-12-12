package orchestration

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
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
}

// Composer handles prompt composition from CUE configuration.
type Composer struct {
	processor  *TemplateProcessor
	workingDir string
}

// NewComposer creates a new prompt composer.
func NewComposer(processor *TemplateProcessor, workingDir string) *Composer {
	return &Composer{
		processor:  processor,
		workingDir: workingDir,
	}
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

	// Resolve each context through UTD
	var promptParts []string
	for _, ctx := range contexts {
		resolved, err := c.resolveContext(cfg, ctx.Name)
		if err != nil {
			// Context errors are warnings, not failures (per DR-007)
			result.Warnings = append(result.Warnings, fmt.Sprintf("context %q: %v", ctx.Name, err))
			continue
		}
		ctx.Content = resolved.Content
		result.Contexts = append(result.Contexts, ctx)
		if resolved.Content != "" {
			promptParts = append(promptParts, resolved.Content)
		}
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
	contextsVal := cfg.LookupPath(cue.ParsePath("contexts"))
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
	ctxVal := cfg.LookupPath(cue.ParsePath("contexts")).LookupPath(cue.MakePath(cue.Str(name)))
	if !ctxVal.Exists() {
		return ProcessResult{}, fmt.Errorf("context not found")
	}

	fields := extractUTDFields(ctxVal)
	if !IsUTDValid(fields) {
		return ProcessResult{}, fmt.Errorf("invalid UTD: no file, command, or prompt")
	}

	return c.processor.Process(fields, "")
}

// resolveRole resolves a role through UTD processing.
func (c *Composer) resolveRole(cfg cue.Value, name string) (string, error) {
	roleVal := cfg.LookupPath(cue.ParsePath("roles")).LookupPath(cue.MakePath(cue.Str(name)))
	if !roleVal.Exists() {
		return "", fmt.Errorf("role not found")
	}

	fields := extractUTDFields(roleVal)
	if !IsUTDValid(fields) {
		return "", fmt.Errorf("invalid UTD: no file, command, or prompt")
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
	if def := cfg.LookupPath(cue.ParsePath("settings.default_role")); def.Exists() {
		if s, err := def.String(); err == nil {
			return s
		}
	}

	// Fall back to first role in definition order
	roles := cfg.LookupPath(cue.ParsePath("roles"))
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
	taskVal := cfg.LookupPath(cue.ParsePath("tasks")).LookupPath(cue.MakePath(cue.Str(name)))
	if !taskVal.Exists() {
		return ProcessResult{}, fmt.Errorf("task %q not found", name)
	}

	fields := extractUTDFields(taskVal)
	if !IsUTDValid(fields) {
		return ProcessResult{}, fmt.Errorf("invalid UTD: no file, command, or prompt")
	}

	return c.processor.Process(fields, instructions)
}

// GetTaskRole returns the role specified for a task.
func GetTaskRole(cfg cue.Value, taskName string) string {
	taskVal := cfg.LookupPath(cue.ParsePath("tasks")).LookupPath(cue.MakePath(cue.Str(taskName)))
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
