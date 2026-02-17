# P-026: CLI Config Loader Return Type Alignment

- Status: Completed
- Started: 2026-02-15
- Completed: 2026-02-17

## Overview

Align return types across the four entity loader functions so agents and tasks return definition order like roles and contexts. Currently `loadAgentsFromDir` and `loadTasksFromDir` discard CUE definition order and return `(map, error)`, while roles and contexts return `(map, []string, error)`. This is a development-time inconsistency, not a deliberate design choice -- CUE's `Fields()` iterator already preserves order in all four cases.

Aligning the return types enables definition-order display (replacing alphabetical sort) and is the prerequisite for any future generic scope-loading abstraction.

Settings remains a separate case (map[string]string, no struct type) and is out of scope.

## Goals

1. Align `loadAgentsFromDir` and `loadTasksFromDir` to return `(map, []string, error)` matching roles and contexts
2. Propagate order through `loadAgentsForScope` and `loadTasksForScope` with the same `(map, []string, error)` return shape
3. Update agent and task list commands to use definition order instead of alphabetical sort
4. Update all callers for new return signatures

## Scope

In Scope:

- Return type alignment for agent and task loaders (FromDir and ForScope)
- Definition order tracking with seen-map deduplication for agents and tasks (matching existing roles/contexts pattern)
- Callers of loadAgentsForScope and loadTasksForScope updated to accept the new return shape
- Remove `sort.Strings(names)` workaround in agent and task list commands

Out of Scope:

- Settings loader changes (structurally different, map[string]string)
- Agent/task reorder commands (separate feature if desired later)
- Genericising the full loadXxxForScope pattern (future decision once return types are aligned)
- Helper extractions already completed (confirmRemoval, extractTags, writeCUETags, writeCUEPrompt)
- Content source menu normalisation (see p-027)

## Current State

- `loadRolesFromDir` and `loadContextsFromDir` return `(map, []string, error)` preserving CUE definition order
- `loadAgentsFromDir` and `loadTasksFromDir` return `(map, error)` discarding order
- Agent and task list commands sort alphabetically via `sort.Strings(names)` as a workaround
- CUE `Fields()` iterator already preserves definition order in all four loaders; agents and tasks just don't capture it
- All config entity files have NOTE(design) comments acknowledging structural duplication

## Success Criteria

- [ ] loadAgentsFromDir returns (map[string]AgentConfig, []string, error)
- [ ] loadTasksFromDir returns (map[string]TaskConfig, []string, error)
- [ ] loadAgentsForScope returns (map[string]AgentConfig, []string, error)
- [ ] loadTasksForScope returns (map[string]TaskConfig, []string, error)
- [ ] Agent and task list commands display in definition order (no sort.Strings)
- [ ] All callers updated for new return signatures
- [ ] All existing tests pass
- [ ] NOTE(design) comments updated to reflect current state

## Deliverables

- Updated `internal/cli/config_agent.go` - aligned return types
- Updated `internal/cli/config_task.go` - aligned return types
- Updated NOTE(design) comments across affected files

## Technical Approach

- Add `var order []string` and `seen` map to `loadAgentsFromDir` and `loadTasksFromDir`
- Capture `name` into order slice during `iter.Next()` loop (same pattern as roles/contexts)
- Update return statements and all callers
- Replace `sort.Strings(names)` in list commands with order slice iteration
- Update NOTE(design) comments to reflect current state
