# P-019: CLI File Path Inputs

- Status: Pending
- Started: -
- Completed: -

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

- [ ] `start --role ./path/to/role.md` reads file and uses as role content
- [ ] `start --context ./path/to/ctx.md` reads file and uses as context
- [ ] `start --context ./a.md,config-name,./b.md` works with mixed inputs in order
- [ ] `start task ./path/to/task.md` reads file and uses as task prompt
- [ ] `start prompt ./path/to/prompt.md` reads file and uses as prompt text
- [ ] Missing files show appropriate error/warning (○ status for contexts)
- [ ] File paths display as-is in output (e.g., `Role: ./my-role.md`)
- [ ] Help text documents file path support
- [ ] CLI docs include file path examples
- [ ] All new code has test coverage

## Deliverables

Code:

- `internal/orchestration/filepath.go` - `isFilePath()` utility function
- `internal/orchestration/filepath_test.go` - unit tests
- Updated `internal/orchestration/composer.go` - role and context file path support
- Updated `internal/cli/task.go` - task file path support
- Updated `internal/cli/prompt.go` - prompt file path support
- Updated `internal/cli/root.go` - flag description updates

Documentation:

- Updated `docs/cli/start.md`
- Updated `docs/cli/task.md`
- Updated `docs/cli/prompt.md`

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

## Dependencies

- DR-038: CLI File Path Inputs (design specification)

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
