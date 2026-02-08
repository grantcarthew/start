# p-023: CLI Config Reorder

- Status: Completed
- Started: 2026-02-08
- Completed: 2026-02-08
- Issue: #3

## Overview

Add an `order` command to context and role configuration, allowing users to reorder items via a simple interactive CLI flow. Context order controls AI injection order (general instructions before specific). Role order controls display and selection order.

Currently all write functions sort alphabetically, destroying any manual ordering. This project refactors context and role write/load functions to preserve definition order, then adds the reorder command.

Design: DR-040

## Goals

1. Add `start config context order` command with interactive move-up reordering
2. Add `start config role order` command with same interaction
3. Add `start config order` bare command that prompts for type selection
4. Refactor context write function to accept and preserve explicit order
5. Refactor role write function to accept and preserve explicit order
6. Refactor role load function to return definition order (contexts already does)
7. Ensure add/edit/remove operations preserve existing order for contexts and roles

## Scope

In Scope:

- `order` action for context and role commands
- `reorder` alias for `order`
- `start config order` bare command with type prompt
- Write function changes for contexts and roles (explicit order parameter)
- Load function change for roles (return order slice)
- Updating all callers of changed write/load functions
- `--local` flag support for scope selection
- Tests for new commands and refactored functions

Out of Scope:

- Reordering agents or tasks (remain alphabetical)
- TUI library or interactive cursor-based navigation
- Merged-view reordering (global and local reordered independently)
- Changes to `internal/assets/install.go` (separate from config write functions)

## Success Criteria

- [ ] `start config context order` displays numbered list and allows move-up reordering
- [ ] `start config role order` displays numbered list and allows move-up reordering
- [ ] `start config order` prompts user to choose contexts or roles, then runs reorder
- [ ] `start config context reorder` works as alias
- [ ] `start config role reorder` works as alias
- [ ] `--local` flag targets local config directory
- [ ] Heading shows scope and file path
- [ ] Enter saves new order, `q`/`quit`/`exit` cancels (case insensitive)
- [ ] Item at position 1 shows "already at top" message
- [ ] Invalid input re-prompts with error message
- [ ] writeContextsFile accepts order parameter and writes fields in that order
- [ ] writeRolesFile accepts order parameter and writes fields in that order
- [ ] loadRolesForScope returns definition order slice
- [ ] Context add/edit/remove operations preserve existing order
- [ ] Role add/edit/remove operations preserve existing order
- [ ] Generated CUE passes syntax validation after reorder
- [ ] All existing tests pass
- [ ] New tests cover order command, write function ordering, load function ordering

## Current State

Verified: 2026-02-08

Write functions (alphabetically sorted):

- `writeContextsFile` in `internal/cli/config_context.go` (lines 872-938) - sorts via `sort.Strings(names)`
- `writeRolesFile` in `internal/cli/config_role.go` (lines 889-952) - sorts via `sort.Strings(names)`

Load functions:

- `loadContextsForScope` in `internal/cli/config_context.go` (lines 736-795) - returns `(map, []string, error)` with order
- `loadContextsFromDir` in `internal/cli/config_context.go` (lines 797-869) - iterates CUE fields preserving definition order
- `loadRolesForScope` in `internal/cli/config_role.go` (lines 772-816) - returns `(map, error)` without order
- `loadRolesFromDir` in `internal/cli/config_role.go` (lines 818-887) - returns `(map, error)` without order, needs same pattern as `loadContextsFromDir`

Write function callers (need order parameter added):

- `writeContextsFile` callers (all load via `loadContextsFromDir` which already returns order):
  - `runConfigContextAdd` (line 332) - loads at line 317, discards order with `_`
  - `runConfigContextEdit` flag path (line 498) - loads at line 459, discards order with `_`
  - `runConfigContextEdit` interactive path (line 627) - same load
  - `runConfigContextRemove` (line 710) - loads at line 673, discards order with `_`
- `writeRolesFile` callers (all load via `loadRolesFromDir` which needs order return added):
  - `runConfigRoleAdd` (line 316) - loads at line 301
  - `runConfigRoleEdit` flag path (line 476) - loads at line 440
  - `runConfigRoleEdit` interactive path (line 584) - same load
  - `runConfigRoleRemove` (line 667) - loads at line 630

`loadRolesForScope` callers (need to handle new order return value):

- `runConfigRoleList` (line 71) - display, currently sorts alphabetically (lines 91-96)
- `runConfigList` roles section (config.go line 106) - display, currently sorts (lines 114-118)
- `runConfigRoleInfo` (line 349) - uses only map, will use `_` for order
- `runConfigRoleDefault` (line 732) - uses only map, will use `_` for order

`loadRolesFromDir` direct callers (beyond loadRolesForScope):

- `runConfigRoleAdd` (line 301)
- `runConfigRoleEdit` (line 440)
- `runConfigRoleRemove` (line 630)

Interactive patterns:

- `promptString` helper in config_agent.go (lines 992-1010) - bufio.Reader with default value
- Choice menus using `bufio.NewReader(stdin)` and `reader.ReadString('\n')` (e.g., config_agent.go lines 216-224)
- Confirmation prompts with case-insensitive matching (e.g., config_agent.go lines 601-613)

Tests:

- All config tests are in `internal/cli/config_test.go` (34 KB) and `config_integration_test.go` (15 KB)
- No separate `config_context_test.go` or `config_role_test.go` files exist
- `TestConfigContextList_PreservesDefinitionOrder` (config_test.go lines 463-529) already validates load-to-display order preservation
- Write function tests (`TestWriteContextsFile`, `TestWriteRolesFile`) use `t.TempDir()` and `strings.Contains` assertions

Dependencies:

- golang.org/x/term v0.36.0 already present (no new dependencies needed)
- cobra for command registration

## Deliverables

Files:

- New `internal/cli/config_order.go` - Order command implementation and shared reorder logic
- Updated `internal/cli/config_context.go` - writeContextsFile order parameter, order subcommand registration
- Updated `internal/cli/config_role.go` - writeRolesFile order parameter, loadRolesForScope order return, order subcommand registration
- Updated `internal/cli/config.go` - `start config order` bare command registration

Tests:

- New `internal/cli/config_order_test.go` - Tests for reorder interaction logic
- Updated `internal/cli/config_test.go` - Tests for ordered writes, ordered loads, and order preservation through add/edit/remove

Design Records:

- DR-040 (already written)

## Technical Approach

Phase 1 - Write function refactor:

1. Change `writeContextsFile` signature to accept `order []string`
2. Replace `sort.Strings(names)` with provided order
3. Update all callers to pass order (read existing order via load, append/remove as needed)
4. Same changes for `writeRolesFile`

Phase 2 - Load function refactor:

1. Update `loadRolesFromDir` to return `(map[string]RoleConfig, []string, error)` - follow `loadContextsFromDir` pattern using `iter.Next()` and `append(order, name)`
2. Update `loadRolesForScope` to return `(map[string]RoleConfig, []string, error)` - propagate order from `loadRolesFromDir` (global order then local order appended, matching context pattern)
3. Update all callers to handle the new return value (including display functions that currently sort)

Phase 3 - Order command:

1. Create reorder interaction function: display list, prompt for move-up, re-display, repeat
2. Register `order` subcommand (with `reorder` alias) on context and role commands
3. Register `start config order` bare command with type selection prompt
4. Wire up scope handling via `--local` flag

Reorder interaction flow:

1. Print heading with scope and file path
2. Print numbered list with metadata (required/default/optional flags for contexts, optional flag for roles)
3. Prompt: `Move up (number), Enter to save, q to cancel: `
4. Read line, trim whitespace, lowercase for comparison
5. Empty: write file with current order, confirm, exit
6. `q`/`quit`/`exit`: discard, confirm cancellation, exit
7. Valid number >= 2 and <= len: swap items at positions n and n-1, re-display, re-prompt
8. Number 1: print "already at top", re-prompt
9. Other: print error, re-prompt

## Testing Strategy

Unit Tests:

- Reorder logic: test move-up swaps correctly, boundary cases (position 1, last position)
- Write function ordering: write with explicit order, verify CUE field order matches
- Load function ordering: load from CUE file, verify returned order matches definition order
- Input parsing: valid numbers, q/quit/exit, empty, invalid input, case insensitivity
- Caller updates: add preserves order (appends), remove preserves order (drops), edit preserves order

Integration Tests:

- Full reorder flow with simulated stdin input
- Reorder then load: verify order persists through write/read cycle
- Add after reorder: verify new item appends to end of custom order

## Notes

Interaction with p-022 (AST refactor):

- p-022 refactors `internal/assets/install.go` (registry asset writing)
- This project changes `internal/cli/config_*.go` (config entity writing)
- These are separate code paths with no overlap
- No dependency between the two projects
