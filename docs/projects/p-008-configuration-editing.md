# P-008: Configuration Editing

- Status: Proposed
- Started: -
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

- `start config agent` - Agent management (new, edit, remove, default)
- `start config role` - Role management (new, edit, remove, default)
- `start config context` - Context management (new, edit, remove)
- `start config task` - Task management (new, edit, remove)
- Support for --local flag to target local config
- Interactive prompts for required fields

Out of Scope:

- Bulk import/export
- Config file format conversion
- Config validation command (covered by doctor)
- Config backup/restore

## Success Criteria

- [ ] `start config agent new` creates new agent config
- [ ] `start config agent edit <name>` edits existing agent
- [ ] `start config agent remove <name>` removes agent
- [ ] `start config agent default <name>` sets default agent
- [ ] Same pattern works for role, context, task
- [ ] --local flag targets .start/ instead of ~/.config/start/
- [ ] Interactive prompts guide users through required fields

## Workflow

### Phase 1: Research and Design

- [ ] Read all required documentation
- [ ] Review prototype config commands for UX patterns
- [ ] Discuss interactive vs non-interactive approaches
- [ ] Discuss CUE file editing strategies
- [ ] Create DR for configuration editing CLI

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

- DR-0XX: Configuration Editing CLI

## Technical Approach

To be determined after Phase 1 research. Key questions:

1. How to edit CUE files programmatically (preserve comments, formatting)?
2. Interactive prompts vs flag-based input?
3. How to handle schema validation during editing?
4. Editor integration ($EDITOR) for complex fields like prompts?
5. How to handle config file creation if none exists?

## Dependencies

Requires:

- P-004 (CUE loading foundation)
- P-006 (auto-setup creates initial config)

## Progress

(No progress yet - project not started)
