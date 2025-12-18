# P-008: Configuration Editing

- Status: In Progress
- Started: 2025-12-18
- Completed: -

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
- Support for --local flag to target local config
- Hybrid input: flags for any field, interactive prompts for missing required fields

Out of Scope:

- Bulk import/export
- Config file format conversion
- Config validation command (covered by doctor)
- Config backup/restore

## Success Criteria

- [ ] `start config agent add` creates new agent config
- [ ] `start config agent edit <name>` edits existing agent
- [ ] `start config agent edit` (no name) opens file in $EDITOR
- [ ] `start config agent remove <name>` removes agent
- [ ] `start config agent default <name>` sets default agent
- [ ] `start config agent list` displays all agents
- [ ] `start config agent show <name>` displays single agent details
- [ ] Same pattern works for role, context, task
- [ ] --local flag targets .start/ instead of ~/.config/start/
- [ ] Flags allow scripted usage, prompts fill missing required fields

## Workflow

### Phase 1: Research and Design

- [x] Read all required documentation
- [x] Review prototype config commands for UX patterns
- [x] Discuss interactive vs non-interactive approaches
- [x] Discuss CUE file editing strategies
- [x] Create DR for configuration editing CLI (DR-029)

### Phase 2: Implementation

- [ ] Implement config agent commands
- [ ] Implement config role commands
- [ ] Implement config context commands
- [ ] Implement config task commands

### Phase 3: Validation

- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Manual testing with real configs

### Phase 4: Review

- [ ] External code review (if significant changes)
- [ ] Fix reported issues
- [ ] Update project document

## Deliverables

Files:

- `internal/cli/config.go` - Config command group
- `internal/cli/config_agent.go` - Agent subcommands
- `internal/cli/config_role.go` - Role subcommands
- `internal/cli/config_context.go` - Context subcommands
- `internal/cli/config_task.go` - Task subcommands
- `internal/config/editor.go` - CUE file editing utilities

Design Records:

- DR-029: CLI Configuration Editing Commands

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
