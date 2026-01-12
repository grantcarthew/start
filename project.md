# Project: Fix {{.file}} Placeholder to Use Local Temp Directory

## Problem

The `{{.file}}` template placeholder returns the CUE cache path instead of a local `.start/temp/` path. This causes permission issues when AI agents (like Claude Code) attempt to read task/role content files.

### Observed Behaviour

When running a task:

```
❯ ./start task start/create-task

Read /Users/gcarthew/Library/Caches/cue/mod/extract/github.com/grantcarthew/start-assets/tasks/start/create-task@v0.0.1/task.md to understand your task.
```

Claude Code then prompts for permission to read from the CUE cache:

```
Read(~/Library/Caches/cue/mod/extract/.../task.md)

Do you want to proceed?
❯ 1. Yes
  2. Yes, allow reading from create-task@v0.0.1/ during this session
```

### Expected Behaviour

Per DR-020, `{{.file}}` should return a path to `.start/temp/`:

```
Read .start/temp/task-start-create-task.md to understand your task.
```

The file would be in the project directory, which is already in an allowed path for AI agents.

## Documentation Reference

### DR-020: Template Processing and File Resolution

Location: `docs/design/design-records/dr-020-template-file-resolution.md`

Key excerpts:

> Resolved content is written to `.start/temp/` with path-derived file names.

> Temp directory location: `.start/temp/` (project-local)

Placeholder semantics table:

| Placeholder | Value |
|-------------|-------|
| `{{.file}}` | Path to resolved temp file |
| `{{.file_contents}}` | Resolved file contents |

Template resolution process (from DR-020):

1. Read content file (role.md, task.md, context.md)
2. Parse as Go template
3. Execute template with placeholder data
4. Generate temp file path from source path
5. Write resolved content to `.start/temp/<derived-name>.md`
6. Provide path via `{{.file}}` or content via `{{.file_contents}}`

## Code References

### Current Implementation (Bug)

File: `internal/orchestration/template.go:141-146`

```go
// Build template data with lowercase keys to match documented placeholders
data := TemplateData{
    "file":         fields.File,  // ← BUG: Uses raw source path (CUE cache)
    "command":      fields.Command,
    "date":         time.Now().Format(time.RFC3339),
    "instructions": instructions,
}
```

The `fields.File` value is the resolved `@module/task.md` path, which points to the CUE cache directory (e.g., `~/Library/Caches/cue/mod/extract/...`).

### Temp Manager (Exists, Unused for This Purpose)

File: `internal/temp/manager.go`

The temp manager already has the required functionality:

```go
// NewUTDManager creates a manager for UTD temp files.
// Files are written to .start/temp/
func NewUTDManager(workingDir string) *Manager {
    return &Manager{BaseDir: filepath.Join(workingDir, ".start", "temp")}
}

// WriteUTDFile writes a temp file with a path-derived name.
// entityType is "role", "context", or "task".
// name is the entity name (e.g., "code-reviewer").
// Returns the path to the written file.
func (m *Manager) WriteUTDFile(entityType, name, content string) (string, error) {
    // ... writes to .start/temp/<entityType>-<name>.md
}
```

### File Name Derivation

File: `internal/temp/manager.go:104-126`

```go
// deriveFileName creates a filename from entity type and name.
// Examples:
//   - ("role", "code-reviewer") -> "role-code-reviewer.md"
//   - ("context", "project/readme") -> "context-project-readme.md"
func deriveFileName(entityType, name string) string {
    // ... sanitizes and formats the name
}
```

## Proposed Solution

### Option 1: Modify TemplateProcessor.Process()

Update `internal/orchestration/template.go` to:

1. Accept a `*temp.Manager` dependency
2. When `fields.File` is set:
   - Read the content from the source file
   - Write to `.start/temp/` using the temp manager
   - Set `data["file"]` to the temp file path

```go
// In TemplateProcessor struct
type TemplateProcessor struct {
    fileReader  FileReader
    shellRunner ShellRunner
    tempManager *temp.Manager  // Add this
    workingDir  string
}

// In Process method, after reading file content
if fields.File != "" {
    content, err := p.fileReader.Read(fields.File)
    if err != nil {
        return result, fmt.Errorf("reading file %s: %w", fields.File, err)
    }

    // Write to temp and use that path
    tempPath, err := p.tempManager.WriteUTDFile(entityType, entityName, content)
    if err != nil {
        return result, fmt.Errorf("writing temp file: %w", err)
    }

    data["file"] = tempPath
    data["file_contents"] = content
    result.FileRead = true
}
```

### Option 2: Pre-process in CLI Layer

Handle the temp file creation in `internal/cli/task.go` or `internal/cli/start.go` before calling the template processor.

### Considerations

1. **Entity type and name**: The template processor needs to know the entity type (task/role/context) and name (e.g., "start/create-task") to derive the temp file name correctly.

2. **Cleanup**: Temp files should be cleaned up periodically. The temp manager has a `Clean()` method.

3. **Gitignore**: `.start/temp/` should be in `.gitignore`. The temp manager has `CheckGitignore()` to verify this.

4. **File content processing**: If the source file contains templates, they should be processed before writing to temp. This is already handled - the content goes through template processing.

## Files to Modify

1. `internal/orchestration/template.go` - Add temp manager integration
2. `internal/cli/task.go` - Pass temp manager to template processor
3. `internal/cli/start.go` - Pass temp manager to template processor
4. Possibly `internal/orchestration/engine.go` if orchestration layer is involved

## Testing

1. Run `./start task start/create-task` and verify prompt shows `.start/temp/task-start-create-task.md`
2. Verify the temp file exists and contains correct content
3. Verify AI agent can read the file without permission prompts
4. Run existing template processor tests
5. Add new tests for temp file creation path

## Related Issues

- Task prompt template changed from `{{.file_contents}}` to `{{.file}}` to avoid large terminal output
- This exposed the bug where `{{.file}}` returns the CUE cache path
- Fix in `internal/registry/index.go` to resolve latest index version was unrelated but done in same session

## Priority

High - This blocks usability of registry-based tasks with AI agents that have file permission controls.
