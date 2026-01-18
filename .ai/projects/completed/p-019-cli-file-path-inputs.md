# P-019: CLI File Path Inputs

- Status: Complete
- Started: 2026-01-16
- Completed: 2026-01-16

## Overview

Implement file path support for CLI inputs as specified in DR-038. This allows users to pass file paths directly to `--role`, `--context`, `start task`, and `start prompt` instead of requiring CUE configuration.

Detection rule: Values starting with `./`, `/`, or `~` are treated as file paths.

## Goals

1. Add file path detection utility function
2. Implement file path support for `--role` flag
3. Implement file path support for `--context` flag (including mixed inputs)
4. Implement file path support for `start task` command
5. Implement file path support for `start prompt` command
6. Update help text for all affected commands
7. Update CLI documentation with examples

## Scope

In Scope:

- File path detection and reading for role, context, task, prompt
- Mixed context inputs (file paths and config names)
- Help text updates
- Documentation updates
- Unit and integration tests

Out of Scope:

- Changes to CUE schemas or configuration format
- New CLI flags or commands
- Validation of file content structure

## Success Criteria

- [x] `start --role ./path/to/role.md` reads file and uses as role content
- [x] `start --context ./path/to/ctx.md` reads file and uses as context
- [x] `start --context ./a.md,config-name,./b.md` works with mixed inputs in order
- [x] `start task ./path/to/task.md` reads file and uses as task prompt
- [x] `start prompt ./path/to/prompt.md` reads file and uses as prompt text
- [x] Missing files show appropriate error/warning (○ status for contexts)
- [x] File paths display as-is in output (e.g., `Role: ./my-role.md`)
- [x] Help text documents file path support
- [x] CLI docs include file path examples
- [x] All new code has test coverage

## Deliverables

Code:

- `internal/orchestration/filepath.go` - `isFilePath()` utility function
- `internal/orchestration/filepath_test.go` - unit tests
- Updated `internal/orchestration/composer.go` - role and context file path support
- Updated `internal/cli/task.go` - task file path support
- Updated `internal/cli/prompt.go` - prompt file path support
- Updated `internal/cli/root.go` - flag description updates

Documentation:

- Create `docs/cli/start.md` (does not exist)
- Create `docs/cli/task.md` (does not exist)
- Create `docs/cli/prompt.md` (does not exist)

## Technical Approach

1. Create `isFilePath(s string) bool` utility:
   - Returns true if string starts with `./`, `/`, or `~`
   - Used consistently across all inputs

2. Role resolution (`composer.go`):
   - In `ComposeWithRole`, check if roleName is a file path
   - If file path, read file content directly instead of config lookup
   - Set RoleName to the file path for display

3. Context resolution (`composer.go`):
   - In context selection/resolution, check each context value
   - If file path, read file content and create Context with path as name
   - Preserve order when mixing paths and config names

4. Task resolution (`task.go`):
   - Check if task argument is a file path
   - If file path, read file content and use as task prompt
   - Skip config lookup for file-based tasks

5. Prompt resolution (`prompt.go`):
   - Check if prompt argument is a file path
   - If file path, read file content and use as customText
   - Existing inline text behavior unchanged

## Current State

Key code locations and patterns:

**Tilde expansion reference:**
- `internal/cli/root.go:125-155` - `resolveDirectory()` implements tilde expansion for `--directory` flag
- Pattern: Check if path starts with `~`, expand using `os.UserHomeDir()`, convert to absolute path

**Role resolution:**
- `internal/cli/start.go:142` - `flags.Role` passed to `ComposeWithRole()`
- `internal/orchestration/composer.go:146-169` - `ComposeWithRole()` calls `resolveRole()` for config lookup
- `internal/orchestration/composer.go:309-355` - `resolveRole()` looks up role in CUE config, extracts UTD fields

**Context handling:**
- `internal/cli/root.go:77` - `flags.Context` is `[]string` (comma-separated via `StringSliceVarP`)
- `internal/orchestration/composer.go:171-254` - `selectContexts()` iterates CUE config contexts with tag filtering
- `internal/orchestration/composer.go:256-306` - `resolveContext()` resolves individual context through UTD

**Task resolution:**
- `internal/cli/task.go:34-43` - `runTask()` extracts task name from args
- `internal/cli/task.go:46-177` - `executeTask()` calls `findTask()` then `ResolveTask()`
- `internal/cli/task.go:291-324` - `findTask()` does exact/substring matching in CUE config
- `internal/orchestration/composer.go:404-454` - `ResolveTask()` looks up task in config, processes UTD

**Prompt handling:**
- `internal/cli/prompt.go:24-39` - `runPrompt()` passes args[0] as customText to `executeStart()`

**Documentation:**
- `docs/cli/start.md` - Does not exist yet (needs creation)
- `docs/cli/task.md` - Does not exist yet (needs creation)
- `docs/cli/prompt.md` - Does not exist yet (needs creation)
- `docs/cli/cli-writing-guide.md` - Exists as reference for documentation style

**Test patterns:**
- `internal/orchestration/composer_test.go` - Table-driven tests, uses `t.TempDir()` for file tests
- `internal/cli/start_test.go` - Integration tests using `setupStartTestConfig()` helper

## Dependencies

- DR-038: CLI File Path Inputs (design specification)

## Review Notes

**Implementation readiness confirmed.** Analysis of codebase shows:

- Tilde expansion pattern exists in `resolveDirectory()` - reusable for file path handling
- Error handling patterns are clear: role errors → warnings, task errors → fatal, context errors → ○ status
- Test patterns established: table-driven tests, `t.TempDir()`, `setupStartTestConfig()` helper
- File reading is straightforward - no temp file management needed (that pattern is for `@module/` paths only)

**No blocking decisions required.** DR-038 has specified:

- Detection rule: `./`, `/`, `~` prefixes only (relative paths like `foo/bar.md` without prefix treated as config names)
- Display: file paths shown as-is in output
- Missing files: contexts show ○ status; roles/tasks follow existing error patterns

## Testing Strategy

Unit tests:

- `isFilePath()` with various inputs (paths, names, edge cases)
- File reading with valid files
- Error handling for missing files

Integration tests:

- Each command with file path inputs
- Mixed context inputs
- Missing file behavior
- Display output verification
