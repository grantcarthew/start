# P-016: Fix {{.file}} Placeholder to Use Local Temp Directory

- Status: Completed
- Started: 2026-01-13
- Completed: 2026-01-13

## Overview

Fixed a bug where the `{{.file}}` template placeholder returned the CUE cache path instead of a local `.start/temp/` path. This caused permission issues when AI agents (like Claude Code) attempted to read task/role content files from external cache directories.

## Problem

When running a task with a file from the CUE registry:

```
$ ./start task start/create-task

Read /Users/gcarthew/Library/Caches/cue/mod/extract/github.com/grantcarthew/start-assets/tasks/start/create-task@v0.0.1/task.md to understand your task.
```

Claude Code would prompt for permission to read from the CUE cache, which is outside the project's allowed paths.

## Solution

Pre-write temp files in `Composer` before calling `TemplateProcessor.Process()`. The sequence is:

1. Extract UTD fields from CUE config
2. Resolve `@module/` paths to CUE cache paths
3. Read source file content
4. Write content to `.start/temp/` with derived filename
5. Replace `fields.File` with temp file path
6. Call `processor.Process()` - template renders with temp path
7. Return result with `TempFile` field populated

## Key Changes

| File | Changes |
|------|---------|
| `internal/orchestration/template.go` | Added `TempFile` field to `ProcessResult` |
| `internal/orchestration/composer.go` | Added `temp` import, `tempManager` field, `resolveFileToTemp()` helper, updated `ResolveTask()`, `resolveRole()`, `resolveContext()` |
| `internal/orchestration/composer_test.go` | Added 5 new tests for temp file creation |

## Additional Fixes

- Added `@module/` path resolution to `resolveRole()` and `resolveContext()` (was only implemented for tasks)

## Verification Checklist

- [x] `internal/orchestration/composer.go` imports `github.com/grantcarthew/start/internal/temp`
- [x] `Composer` struct has `tempManager *temp.Manager` field
- [x] `NewComposer()` initialises temp manager with `temp.NewUTDManager(workingDir)`
- [x] `resolveFileToTemp()` helper method added
- [x] `ResolveTask()` calls `resolveFileToTemp()` before `processor.Process()`
- [x] `ResolveTask()` sets `result.TempFile` after processing
- [x] `resolveRole()` includes `@module/` path resolution (was missing)
- [x] `resolveRole()` calls `resolveFileToTemp()` before `processor.Process()`
- [x] `resolveContext()` includes `@module/` path resolution (was missing)
- [x] `resolveContext()` calls `resolveFileToTemp()` before `processor.Process()`
- [x] `resolveContext()` sets `result.TempFile` after processing
- [x] `ProcessResult` has `TempFile string` field in `template.go`
- [x] All existing tests pass: `go test ./...`
- [x] New temp file tests pass
- [x] `.gitignore` contains `.start/`

## Reference Documentation

- **DR-020:** Template processing and temp file design
- **DR-023:** Module path resolution using origin field
- **Temp Manager:** `internal/temp/manager.go`

## Dependencies

- P-005: Orchestration Core Engine (template processing)
- P-015: Schema Base and Origin Tracking (origin field)
