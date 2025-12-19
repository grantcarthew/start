# P-008: Configuration Editing

- Status: Complete
- Started: 2025-12-18
- Completed: 2025-12-18

## Overview

Implement configuration editing commands for managing agents, roles, contexts, and tasks. This provides a user-friendly interface for configuring start without manually editing CUE files.

## Required Reading

| Document | Why |
|----------|-----|
| DR-008 | Context schema design |
| DR-009 | Task schema design |
| DR-010 | Role schema design |
| DR-011 | Agent schema design |
| DR-012 | Global flags (--local flag) |
| DR-025 | Configuration merge semantics |
| Prototype start-config*.md | Prior art for CLI interface |

## Goals

1. Design CLI interface for configuration editing
2. Implement agent configuration commands
3. Implement role configuration commands
4. Implement context configuration commands
5. Implement task configuration commands

## Scope

In Scope:

- `start config agent` - Agent management (list, add, show, edit, remove, default)
- `start config role` - Role management (list, add, show, edit, remove, default)
- `start config context` - Context management (list, add, show, edit, remove)
- `start config task` - Task management (list, add, show, edit, remove)
- `start config settings` - Settings management (list, get, set, edit)
- Support for --local flag to target local config
- Hybrid input: flags for any field, interactive prompts for missing required fields

Out of Scope:

- Bulk import/export
- Config file format conversion
- Config validation command (covered by doctor)
- Config backup/restore

## Success Criteria

- [x] `start config agent add` creates new agent config
- [x] `start config agent edit <name>` edits existing agent
- [x] `start config agent edit` (no name) opens file in $EDITOR
- [x] `start config agent remove <name>` removes agent
- [x] `start config agent default <name>` sets default agent
- [x] `start config agent list` displays all agents
- [x] `start config agent show <name>` displays single agent details
- [x] Same pattern works for role, context, task
- [x] --local flag targets .start/ instead of ~/.config/start/
- [x] Flags allow scripted usage, prompts fill missing required fields
- [x] Plural aliases work (e.g., `start config agents` = `start config agent`)

## Workflow

### Phase 1: Research and Design

- [x] Read all required documentation
- [x] Review prototype config commands for UX patterns
- [x] Discuss interactive vs non-interactive approaches
- [x] Discuss CUE file editing strategies
- [x] Create DR for configuration editing CLI (DR-029)

### Phase 2: Implementation

- [x] Implement config agent commands
- [x] Implement config role commands
- [x] Implement config context commands
- [x] Implement config task commands

### Phase 3: Validation

- [x] Write unit tests
- [x] Write integration tests
- [x] Manual testing with real configs

### Phase 4: Review

- [x] External code review (if significant changes)
- [x] Fix reported issues
- [x] Update project document

## Deliverables

Files:

- `internal/cli/config.go` - Config command group
- `internal/cli/config_agent.go` - Agent subcommands
- `internal/cli/config_role.go` - Role subcommands
- `internal/cli/config_context.go` - Context subcommands
- `internal/cli/config_task.go` - Task subcommands
- `internal/cli/config_settings.go` - Settings subcommands
- `internal/cli/config_test.go` - Unit tests for config commands
- `internal/cli/config_integration_test.go` - Integration tests for full workflows
- `context/start-assets/schemas/settings.cue` - Settings schema
- `context/start-assets/schemas/settings_example.cue` - Settings examples

Design Records:

- DR-029: CLI Configuration Editing Commands
- DR-030: Settings Schema (config.cue rename, settings CLI)

## Technical Approach

Decisions from Phase 1 (documented in DR-029):

1. CUE file editing: Template-based generation, one top-level key per file (agents.cue contains only `agents: {...}`). Regenerate entire file on modification. No AST manipulation.

2. Input handling: Hybrid approach - flags for any field, interactive prompts for missing required fields. Fully scriptable when all flags provided.

3. Validation: Validate after every write operation. Immediate feedback on configuration issues.

4. Editor integration: File-level only. `start config <type> edit` (no name) opens file in $EDITOR. No editor integration for individual fields.

5. File creation: Generate files using template functions (following `generateAgentCUE` pattern from auto-setup).

6. Backups: None (can be added later if needed). Users rely on version control.

## Dependencies

Requires:

- P-004 (CUE loading foundation)
- P-006 (auto-setup creates initial config)

## Progress

2025-12-18: Phase 1 complete. Created DR-029 documenting all design decisions. Ready for Phase 2 implementation.

2025-12-18: Phase 2 and Phase 3 (unit tests) complete. Implemented all config commands:
- `start config agent` - list, add, show, edit, remove, default
- `start config role` - list, add, show, edit, remove, default
- `start config context` - list, add, show, edit, remove
- `start config task` - list, add, show, edit, remove

All commands support:
- `--local` flag to target project-specific config
- Hybrid input (flags for scripted use, interactive prompts for missing fields)
- Template-based CUE file generation
- Graceful handling of empty/missing config directories
- Plural aliases (`start config agents` works same as `start config agent`)

Unit tests added in config_test.go covering list, show, add, remove operations.

Integration tests added in config_integration_test.go covering:
- Full agent workflow (add, list, show, default, remove)
- Full role workflow (add with file/prompt, list, show)
- Full context workflow (add required/default, list with markers)
- Full task workflow (add with role, list, show)
- Local/global config isolation (--local flag)

Manual testing verified:
- Real global config loads correctly
- Agent list/show displays existing gemini config
- Task list/show displays existing code-review task
- Local context creation works with --local flag
- Generated CUE files are valid syntax

2025-12-18: Added settings management:
- Renamed `config.cue` to `settings.cue` across codebase for consistency
- Created DR-030 documenting settings schema and CLI design
- Implemented `start config settings` command with positional interface:
  - `start config settings` - list all settings
  - `start config settings <key>` - show setting value
  - `start config settings <key> <value>` - set setting value
  - `start config settings edit` - open in $EDITOR
- Created settings schema (`settings.cue`, `settings_example.cue`) in start-assets
- Added 8 unit tests for settings command
- Updated all 20 dependent CUE modules to schemas v0.0.3

2025-12-19: External code review completed. Fixed reported issues:
- Centralized CUE key constants (`internal/cue/keys.go`) to prevent typos
- Removed `os.ExpandEnv` from `escapeForShell` - environment variables no longer expanded in prompts
- Refactored CLI logic - extracted common setup from `start.go` and `task.go` into `prepareExecutionEnv()` helper
- Added Windows startup guard in `root.go` with clear error message
- Updated tests to reflect new behaviour
