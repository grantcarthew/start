# P-034: CLI Config Add/Edit Flags Removal

- Status: Completed
- Started: 2026-02-24
- Completed: 2026-02-24

## Overview

`start config <type> add` and `start config <type> edit` currently support two modes: non-interactive (via field flags) and interactive (via prompts). The non-interactive flag path is being removed. Both commands will be always interactive.

This change implements part of the decision in dr-044, which states "add and edit are always interactive — no field-specific flags." It is a prerequisite to p-032 (CLI Config Verb-First Refactor) and reduces that project's scope by eliminating the need to handle flag-to-stdin test migration during the larger structural refactor.

## Goals

1. Remove all field flags from `config agent add` and `config agent edit`
2. Remove all field flags from `config role add` and `config role edit`
3. Remove all field flags from `config context add` and `config context edit`
4. Remove all field flags from `config task add` and `config task edit`
5. Update integration tests to use stdin-driven interactive input instead of flags
6. Remove unit tests that exercised flag-based paths

## Scope

In Scope:

- `internal/cli/config_agent.go` — add and edit commands
- `internal/cli/config_role.go` — add and edit commands
- `internal/cli/config_context.go` — add and edit commands
- `internal/cli/config_task.go` — add and edit commands
- `internal/cli/config.go` — delete `anyFlagChanged` (becomes dead code)
- `internal/cli/config_test.go` — remove flag-path tests
- `internal/cli/config_integration_test.go` — convert flag adds to stdin-driven

Out of Scope:

- `config <type> remove --yes`/`-y` flag (not a field flag — confirmation bypass, retained by p-032)
- Interactive prompt logic itself (no behaviour changes, prompts stay as-is)
- `config_settings.go`, `config_search.go`, `config_helpers.go` (not affected)
- `config_order.go`, `config_interactive.go` (not affected)
- p-032 structural refactor (separate project)

## Current State

Each noun-group file (`config_agent.go`, `config_role.go`, `config_context.go`, `config_task.go`) implements add and edit with a flag-branch pattern:

- Flag definitions are registered on the command via `cmd.Flags().String(...)` etc.
- A `hasFlags` / `hasEditFlags` check (`anyFlagChanged(cmd, ...)`) branches between non-interactive (flag) and interactive (prompt) paths
- 4 such branches exist per file (2 for add, 2 for edit — the check and the flag-read block)

Field flags per type:

- agent add: `name`, `bin`, `command`, `default-model`, `description`, `model`, `tag`
- agent edit: `bin`, `command`, `default-model`, `description`, `model`, `tag`
- role add: `name`, `description`, `file`, `command`, `prompt`, `tag`, `optional`
- role edit: `description`, `file`, `command`, `prompt`, `tag`, `optional`
- context add: `name`, `description`, `file`, `command`, `prompt`, `required`, `default`, `tag`
- context edit: `description`, `file`, `command`, `prompt`, `required`, `default`, `tag`
- task add: `name`, `description`, `file`, `command`, `prompt`, `role`, `tag`
- task edit: `description`, `file`, `command`, `prompt`, `role`, `tag`

`config_integration_test.go` (1095 lines) uses flag-style adds throughout — exactly 58 flag usages that must be converted to stdin-driven input.

`config_test.go` (1988 lines) contains two unit tests exercising flag-based paths that must be removed:

- `TestConfigAgentAdd_NonInteractive_MissingFlags` — asserts `--name is required` error; invalid after this change
- `TestConfigAgentAdd_WithFlags` — exercises the flag-driven add path; removed with the path itself

`go test ./internal/cli/...` passes — confirmed green baseline. Working tree is clean.

`anyFlagChanged` in `config.go` is called exclusively from the four add/edit run functions. After flags are removed it becomes dead code and must be deleted.

Error messages in add functions that reference flags (e.g., `"--name is required (run interactively or provide flag)"`) must be updated to remove the flag references (e.g., `"name is required"`). Same for `--command`.

## Success Criteria

### Flags removed

- [ ] `start config agent add --name foo` returns unknown flag error
- [ ] `start config agent edit claude --bin foo` returns unknown flag error
- [ ] `start config role add --name foo` returns unknown flag error
- [ ] `start config role edit golang/assistant --description foo` returns unknown flag error
- [ ] `start config context add --name foo` returns unknown flag error
- [ ] `start config context edit environment --file foo` returns unknown flag error
- [ ] `start config task add --name foo` returns unknown flag error
- [ ] `start config task edit review --description foo` returns unknown flag error

### Interactive paths unchanged

- [ ] `start config agent add` prompts interactively for all fields
- [ ] `start config agent edit <name>` prompts interactively for all fields
- [ ] `start config role add` prompts interactively for all fields
- [ ] `start config role edit <name>` prompts interactively for all fields
- [ ] `start config context add` prompts interactively for all fields
- [ ] `start config context edit <name>` prompts interactively for all fields
- [ ] `start config task add` prompts interactively for all fields
- [ ] `start config task edit <name>` prompts interactively for all fields

### Tests

- [ ] All tests pass via `scripts/invoke-tests`
- [ ] No add/edit tests reference removed flag names (`--name`, `--bin`, `--command`, `--default-model`, `--description`, `--model`, `--file`, `--prompt`, `--default`, `--required`, `--optional`, `--role`, `--tag`)
- [ ] Integration tests use stdin-driven input for all add and edit operations

## Testing Strategy

Follow dr-024:

- Integration tests drive adds and edits by writing prompt responses to stdin
- Use `t.TempDir()` real config files — no mocks
- Table-driven tests where multiple cases share the same command structure

Run tests:

```
scripts/invoke-tests
```

## Deliverables

- `internal/cli/config_agent.go` (modified — flags and flag-branch removed from add/edit)
- `internal/cli/config_role.go` (modified)
- `internal/cli/config_context.go` (modified)
- `internal/cli/config_task.go` (modified)
- `internal/cli/config.go` (modified — `anyFlagChanged` deleted)
- `internal/cli/config_test.go` (updated — flag-path tests removed)
- `internal/cli/config_integration_test.go` (updated — flag adds converted to stdin-driven)

## Dependencies

- p-008: Configuration Editing
- p-013: CLI Configuration Testing
- p-017: CLI Config Edit Flags
- p-018: CLI Interactive Edit Completeness

Blocks: p-032
