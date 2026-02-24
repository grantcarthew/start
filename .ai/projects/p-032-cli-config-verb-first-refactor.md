# P-032: CLI Config Verb-First Refactor

- Status: Pending
- Started:
- Completed:

## Progress Tracking

Update this document as work proceeds:

- Set Status to In Progress and fill in Started when work begins
- Mark each phase complete in the Phases section as it finishes
- Tick off Success Criteria checkboxes as each item is verified
- Set Status to Complete and fill in Completed when all criteria pass
- Move this file to `.ai/projects/completed/` and update `AGENTS.md`

## Overview

`start config` uses noun-first commands — category subcommands group verbs underneath them (`config agent edit`, `config role add`). `start assets` uses verb-first — verbs are top-level and targets are arguments (`assets add`, `assets list`).

This project restructures `start config` to verb-first, removes all noun-group subcommands, and introduces search-by-name with interactive menus for commands that target existing items. The result is a consistent pattern across the CLI.

The equivalent change to `start show` is covered by p-033, which is a prerequisite to this project.

This is a breaking change to the command interface. It is intentional and timed for early in the project lifecycle before a wide user base forms.

## Required Reading

| Document | Why |
| --- | --- |
| dr-044 | The architectural decision this project implements |
| dr-024 | Testing strategy — real behaviour over mocks |
| dr-026 | CLI logic and I/O separation |
| dr-042 | Terminal colour standard |

## Current State

### start config — noun-first structure

```
start config add              # interactive: prompt category → add flow
start config edit             # interactive: prompt category → pick item → edit
start config remove           # interactive: prompt category → pick item → remove
start config order            # interactive: prompt category → reorder
start config search [query]
start config settings [key] [value]

start config agent            # noun group
  config agent add
  config agent default [name]
  config agent edit [name]    # no name = open agents.cue in $EDITOR
  config agent info <name>
  config agent list
  config agent remove <name>

start config role             # noun group
  config role add
  config role edit [name]     # no name = open roles.cue in $EDITOR
  config role info <name>
  config role list
  config role order
  config role remove <name>

start config context          # noun group
  config context add
  config context edit [name]  # no name = open contexts.cue in $EDITOR
  config context info <name>
  config context list
  config context order
  config context remove <name>

start config task             # noun group
  config task add
  config task edit [name]     # no name = open tasks.cue in $EDITOR
  config task info <name>
  config task list
  config task remove <name>
```

### start show — noun subcommands

```
start show [name]             # cross-category search, verbose dump
start show agent [name]       # restrict to agents
start show role [name]        # restrict to roles
start show context [name]     # restrict to contexts
start show task [name]        # restrict to tasks
```

### Relevant files

Implementation:

- `internal/cli/config.go` — 178 lines, root config command and noun-group registration
- `internal/cli/config_types.go` — 583 lines, all config types, loaders, and writers (created by p-036)
- `internal/cli/config_agent.go` — 623 lines, agent noun group (add/edit/remove/list/info/default); types/loaders/writers removed by p-036
- `internal/cli/config_role.go` — 509 lines, role noun group (add/edit/remove/list/info/order); types/loaders/writers removed by p-036
- `internal/cli/config_context.go` — 567 lines, context noun group (add/edit/remove/list/info/order); types/loaders/writers removed by p-036
- `internal/cli/config_task.go` — 515 lines, task noun group (add/edit/remove/list/info); types/loaders/writers removed by p-036
- `internal/cli/config_helpers.go` — 837 lines, shared CUE read/write helpers, field prompt helpers
- `internal/cli/config_interactive.go` — 200 lines, shared interactive picker flows (routes to noun-group subcommands via cmd.Root().Find — replaced entirely by new verb files)
- `internal/cli/config_open.go` — 93 lines, open command (unchanged, delivered by p-035)
- `internal/cli/config_order.go` — 318 lines, top-level order command; contains dead code `addConfigContextOrderCommand`/`addConfigRoleOrderCommand` (called from noun-group files, dead after deletion)
- `internal/cli/config_settings.go` — 468 lines, settings command (unchanged)
- `internal/cli/config_search.go` — 138 lines, search command (unchanged)

Tests:

- `internal/cli/config_test.go` — 1915 lines
- `internal/cli/config_integration_test.go` — 952 lines (reduced from 1103 by p-034 interactive conversion)
- `internal/cli/config_order_test.go` — 818 lines (grew from 754 during p-037)
- `internal/cli/config_helpers_test.go` — 461 lines
- `internal/cli/config_open_test.go` — 210 lines (unchanged, p-035)
- `internal/cli/config_interactive_test.go` — 102 lines
- `internal/cli/config_testhelpers_test.go` — 23 lines, contains `slowReader`/`slowStdin` helper that prevents bufio over-consumption when multiple sequential prompt functions each create their own `bufio.NewReader`; rewritten tests must use this

### Codebase observations

p-036 migrated all types, loaders, and writers to `config_types.go`. The noun-group files no longer define any of the following; they only use them:

- `AgentConfig`, `RoleConfig`, `ContextConfig`, `TaskConfig` struct types
- `loadAgentsForScope/FromDir`, `loadRolesForScope/FromDir`, `loadContextsForScope/FromDir`, `loadTasksForScope/FromDir`
- `writeAgentsFile`, `writeRolesFile`, `writeContextsFile`, `writeTasksFile`
- `loadConfigForScope`, `getDefaultAgentFromConfig`

The noun-group files can be deleted directly without losing any shared code.

`config_interactive.go` currently routes interactive add/edit/remove to noun-group subcommands via `cmd.Root().Find([]string{"config", singular, "add"})`. This routing pattern is entirely replaced by the new verb files (`config_add.go`, `config_edit.go`, `config_remove.go`). After the refactor, `config_interactive.go` retains only `allConfigCategories` and `loadNamesForCategory`.

`config_order.go` contains `addConfigContextOrderCommand` and `addConfigRoleOrderCommand` which are registered by the noun-group files. These registration functions become dead code after noun group deletion and should be removed from `config_order.go`. The reorder logic (`reorderContexts`, `reorderRoles`, `runReorderLoop`) is reusable and stays.

Integration tests in `config_integration_test.go` are converted to stdin-driven input by p-034, which is a prerequisite. By the time p-032 begins, the integration tests already use interactive input. The test rewrite in p-032 is therefore a command-path update rather than a full interactive conversion.

`config_helpers.go` already contains the shared infrastructure the new verb commands will call directly:

- `promptSelectCategory`, `promptSelectOneFromList`, `promptSelectFromList` — interactive menus
- `resolveAllMatchingNames`, `resolveRemoveNames` — cross-map search and resolution
- `confirmMultiRemoval` — confirmation dialog
- `loadForScope[T]` — generic merge strategy (called by the typed loaders in `config_types.go`)

Write function signatures differ between orderable and non-orderable types:

- `writeAgentsFile(path, agents)` — no order (sorts alphabetically)
- `writeTasksFile(path, tasks)` — no order (sorts alphabetically)
- `writeRolesFile(path, roles, order)` — takes explicit order slice
- `writeContextsFile(path, contexts, order)` — takes explicit order slice

The `promptModels` bug fix (clear option calls `promptModelsEdit` instead of returning nil) and its corresponding test in `config_test.go` were committed prior to p-032 start. These do not affect the refactor scope.

`internal/orchestration` has a pre-existing test failure unrelated to p-032 (`TestBuildCommand_WithEnvVarPrefix` fails because the test environment has no `gemini` binary). This failure predates p-032. The success criterion "All tests pass via `scripts/invoke-tests`" should be read as "all CLI tests pass and no new failures are introduced". Verify with `go test ./internal/cli/...` to confirm CLI-scope test health independently.

`go test ./internal/cli/...` passes — confirmed green baseline (re-verified post p-037 commits). p-034 is complete; all add/edit commands are already always interactive with no field flags.

Cross-category search for `edit [query]`, `remove [query]`, and `info [query]` requires aggregating results across all four category maps. The existing helpers (`resolveAllMatchingNames`, `resolveRemoveNames`) operate on a single typed map. The new verb commands need an internal helper that loads all four maps, searches each, and returns results tagged with their category — so the menu can display `claude (agent)` vs `golang (role)`, and removal/edit can write back to the correct category file. This is implementation detail with no design decision required.

For `config edit` with no args, the intent matches the current `runConfigInteractiveEdit` flow: prompt for category → show items in that category → edit. The phrasing "prompt to pick from all items" in the no-arg table means any category is reachable, not a flat cross-category list. This is consistent with how `config info` and `config remove` no-arg are described.

The README was updated prior to p-032 start (agent/role/task name updates and added Inspection section). The config and show command examples in the README still use the old noun-first paths and will be updated in Phase 3 of this project.

p-033 is complete. `start show` noun subcommands (`show agent`, `show role`, `show context`, `show task`) have been removed. The prerequisite for p-032 is met.

p-035 is complete. `config open [category]` is implemented in `config_open.go` and registered in `config.go`. Unchanged by p-032.

p-036 is complete. All config types (`AgentConfig`, `RoleConfig`, `ContextConfig`, `TaskConfig`), loaders (`loadAgentsForScope/FromDir`, etc.), and writers (`writeAgentsFile`, etc.) have been migrated to `config_types.go`. The noun-group files now use these definitions but no longer own them, and can be deleted directly without losing any shared code.

## Goals

1. Remove all noun-group subcommands from `start config`
2. Implement new verb-first commands: `add`, `edit`, `remove`, `list`, `info`
3. Implement search-by-name with interactive menus for `edit`, `remove`, `info`
4. Remove `start config agent default` (duplicate of `config settings default_agent`)
5. Update all tests to reflect new command paths
6. Update README.md and docs
7. All new output in new files uses `internal/tui` colour helpers per dr-042

## Scope

In Scope:

- All `internal/cli/config*.go` files (except config_settings.go and config_search.go)
- All associated test files
- `README.md` configuration and usage sections

Out of Scope:

- `start assets` (already correct)
- `start config settings` internals (unchanged)
- `start config search` internals (unchanged)
- `start config open` — delivered by p-035 (complete before p-032 begins)
- CUE schemas, config file formats, or registry interaction
- Adding new features beyond the restructure

## New Command Structure

### start config

```
start config                          # list effective config (unchanged behaviour)
start config list [category]          # list items; all categories if omitted
start config info [query]             # search by name, show raw config fields; prompt interactively if no query
start config add [category]           # add item; prompt for category if omitted
start config edit [query]             # search by name, edit matched item; prompt interactively if no query
start config remove [query]           # search by name, confirm, delete; prompt interactively if no query
start config open [category]          # open .cue file in $EDITOR; prompt if no category (p-035)
start config order [category]         # reorder items; prompt if no category
start config search [query]           # unchanged
start config settings [key] [value]   # unchanged
```

## Command Behaviour Detail

### No-argument behaviour

| Command | No argument |
| --- | --- |
| `start config` | list all (unchanged) |
| `start config list` | list all items grouped by category |
| `start config info` | prompt category → item → display raw fields |
| `start config add` | prompt for category (agent/role/context/task) |
| `start config edit` | interactive: prompt to pick from all items |
| `start config remove` | prompt category → item picker → confirmation → delete |
| `start config open` | prompt for which file — delivered by p-035 |
| `start config order` | prompt for category (context/role only) |

### start config list

```
start config list              # list all: agents, roles, contexts, tasks grouped by category
start config list agent        # list only agents
start config list role         # list only roles
start config list context      # list only contexts
start config list task         # list only tasks
```

Output matches the current per-category list commands.

### start config info

```
start config info                        # prompt: category → item → display raw fields
start config info claude                 # search by name, menu if multiple matches
start config info claude/interactive     # exact match, show raw config fields
```

Shows raw stored configuration fields — not resolved content. Distinct from `start show` which resolves file/command sources after merging global and local config.

### start config add

```
start config add               # prompt: which category?
start config add agent         # add agent — prompt for required fields
start config add role          # add role
start config add context       # add context
start config add task          # add task
```

Category is required; when omitted, prompt. No search-by-name — adding always creates something new.

Always interactive — no field flags. Users who want scripted or non-interactive editing should use `start config open` to edit the CUE file directly.

Plural aliases: `agents`, `roles`, `contexts`, `tasks` accepted as aliases for their singular forms.

### start config edit

```
start config edit                        # interactive: prompt to pick item from all categories
start config edit claude                 # search, menu if multiple, then edit interactively
start config edit claude/interactive     # exact match, edit interactively
```

If query matches zero items: inform user, exit. If one match: go straight to edit. If multiple: show numbered menu.

Always interactive — no field flags. Users who want scripted or non-interactive editing should use `start config open` to edit the CUE file directly.

### start config remove

```
start config remove                       # prompt: category → item picker → confirmation → delete
start config remove claude/interactive    # exact match, confirmation dialog, delete
start config remove claude                # search, menu if multiple, confirmation, delete
start config remove claude/interactive -y # skip confirmation dialog
```

Query is optional. With no argument, prompts for category then item(s) interactively, then confirms before deleting. Confirmation dialog always shown unless `--yes` / `-y` flag is provided. The `-y` flag is the only non-global flag on this command.

### start config order

The optional category argument (`order context`, `order role`, `order agent`, `order task`) is delivered by p-037 before p-032 begins. By the time p-032 starts, direct category routing already works.

The remaining change in p-032 is removing dead code: `addConfigContextOrderCommand` and `addConfigRoleOrderCommand` are registered by the noun-group files and become unreachable after those files are deleted.

## Removed Commands

Every path listed below must return a "command not found" or usage error after this refactor:

```
start config agent
start config agent add
start config agent default
start config agent edit
start config agent info
start config agent list
start config agent remove
start config role
start config role add
start config role edit
start config role info
start config role list
start config role order
start config role remove
start config context
start config context add
start config context edit
start config context info
start config context list
start config context order
start config context remove
start config task
start config task add
start config task edit
start config task info
start config task list
start config task remove
```

## File Changes

### Delete

- `internal/cli/config_agent.go`
- `internal/cli/config_role.go`
- `internal/cli/config_context.go`
- `internal/cli/config_task.go`

### Create

- `internal/cli/config_add.go` — `config add [category]` command
- `internal/cli/config_edit.go` — `config edit [query]` command
- `internal/cli/config_remove.go` — `config remove <query>` command
- `internal/cli/config_list.go` — `config list [category]` command
- `internal/cli/config_info.go` — `config info [query]` command

### Modify

- `internal/cli/config.go` — register new verb commands, remove noun-group registrations
- `internal/cli/config_helpers.go` — review and retain shared helpers; remove any helpers only used by deleted noun groups
- `internal/cli/config_interactive.go` — update shared interactive flows for new verb structure
- `internal/cli/config_order.go` — remove dead code (`addConfigContextOrderCommand`, `addConfigRoleOrderCommand`)

### Unchanged

- `internal/cli/config_settings.go`
- `internal/cli/config_search.go`
- `internal/cli/config_open.go` (delivered by p-035)

### Tests — Rewrite

- `internal/cli/config_test.go` — rewrite for new verb commands
- `internal/cli/config_integration_test.go` — rewrite full workflow tests
- `internal/cli/config_helpers_test.go` — review, remove tests for deleted helpers
- `internal/cli/config_interactive_test.go` — update for new flows

### Docs

- `README.md` — update Configuration section and Usage > Configuration section

## Success Criteria

### start config — new commands

- [ ] `start config` lists all items (unchanged)
- [ ] `start config list` lists all items grouped by category
- [ ] `start config list agent` lists only agents
- [ ] `start config list role` lists only roles
- [ ] `start config list context` lists only contexts
- [ ] `start config list task` lists only tasks
- [ ] `start config info` prompts for category then item, displays raw fields
- [ ] `start config info claude` searches and shows config fields for claude
- [ ] `start config info claude/interactive` shows config for exact name match
- [ ] `start config info golang` shows menu when multiple items match
- [ ] `start config add` prompts for category
- [ ] `start config add agent` skips category prompt, starts add flow
- [ ] `start config add agents` works (plural alias)
- [ ] `start config add role` starts role add flow
- [ ] `start config add context` starts context add flow
- [ ] `start config add task` starts task add flow
- [ ] `start config edit` prompts interactively to pick an item
- [ ] `start config edit claude` finds and edits claude agent interactively
- [ ] `start config edit golang/assistant` exact match, edits interactively
- [ ] `start config remove` prompts for category, item picker, confirmation, deletes
- [ ] `start config remove claude/interactive` prompts confirmation then deletes
- [ ] `start config remove claude` shows menu if multiple matches, then confirms
- [ ] `start config remove claude/interactive -y` skips confirmation and deletes
- [ ] `start config remove claude/interactive --yes` skips confirmation and deletes
- [ ] `start config open` prompts for which file (delivered by p-035)
- [ ] `start config order` prompts for context or role (unchanged)
- [ ] `start config order context` goes straight to context reorder
- [ ] `start config order role` goes straight to role reorder
- [ ] `start config order agent` falls back to the context/role prompt menu
- [ ] `start config order task` falls back to the context/role prompt menu
- [ ] `start config search golang` works (unchanged)
- [ ] `start config settings default_agent claude` works (unchanged)

### Zero-match behaviour

- [ ] `start config edit name-that-doesnt-exist` informs user, exits non-zero
- [ ] `start config remove name-that-doesnt-exist` informs user, exits non-zero
- [ ] `start config info name-that-doesnt-exist` informs user, exits non-zero

### start config — removed commands

- [ ] `start config agent` returns error or help indicating command not found
- [ ] `start config agent edit claude` returns error
- [ ] `start config role add` returns error
- [ ] `start config context list` returns error
- [ ] `start config task remove review` returns error
- [ ] `start config agent default claude` returns error

### Tests

- [ ] All tests pass via `scripts/invoke-tests`
- [ ] No tests reference deleted noun-group command paths
- [ ] Integration tests cover full add/edit/remove/list workflows for each category
- [ ] Integration tests use `t.TempDir()` real config files (no mocks)

### Docs

- [ ] README.md Configuration section uses new command paths
- [ ] README.md Usage > Configuration section uses new command paths

## Testing Strategy

Follow dr-024:

- Test real behaviour — use actual CUE validation, real files via `t.TempDir()`
- Table-driven tests for command routing and argument handling
- Integration tests cover full workflows: add an item, list it, edit it, remove it
- Test no-argument behaviour explicitly for each verb command
- Test plural aliases for `add`, `list`, `order`
- Test that all removed command paths return errors

### Required test cases

#### config list

- `config list` with no config — each category section prints a zero-count or "none" message
- `config list` with items in all four categories — output contains all item names grouped by category, each category heading present
- `config list agent` — only agents section present in output
- `config list role` — only roles section present
- `config list context` — only contexts section present
- `config list task` — only tasks section present
- `config list agents` — plural alias accepted, same output as `config list agent`
- `config list unknowncategory` — returns error

#### config info — per-category field display

Each type has distinct fields that must appear in output:

- Agent: `config info <name>` output contains `bin`, `command`, and `models` block when present; `default_model` when set; `description` when set
- Role: `config info <name>` output contains `prompt` content; `optional` marker when set
- Context: `config info <name>` output contains `required` and `default` markers when set; `tags` when present
- Task: `config info <name>` output contains `description` when set; `prompt` content; `role` when set

#### config info — search and no-arg

- `config info <exact-name>` — shows fields with no menu
- `config info <substring>` matching one item — resolves directly, shows fields
- `config info <substring>` matching zero items — informs user, exits non-zero
- `config info` with no args — returns terminal-required error in non-interactive mode

#### cross-category search (config edit, config remove, config info)

These cases require seeding items with the same name in different categories (e.g., an agent named `shared` and a role named `shared`):

- Query that matches one item in one category — goes directly to the action (no menu)
- Query that matches items in multiple categories — presents a menu showing each match with its category label (e.g., `shared (agent)`, `shared (role)`)
- After menu selection, action targets the correct category — verify the right CUE file is written
- `config remove shared --yes` with matches in two categories — removes all matches, both CUE files updated

#### config remove — confirmation and --yes flag

- `config remove <name>` in non-interactive mode without `--yes` — returns error requiring `--yes`
- `config remove <name> --yes` — skips confirmation prompt, removes item
- `config remove <name> -y` — short flag accepted, same behaviour as `--yes`
- `config remove <name>` interactively with `y` response — removes item
- `config remove <name>` interactively with `n` response — cancels, item still present
- `config remove <no-match> --yes` — informs user, exits non-zero, no file modified

#### zero-match behaviour

Table-driven, one case per command:

- `config edit name-that-doesnt-exist` — error message contains "not found", exits non-zero
- `config remove name-that-doesnt-exist --yes` — error message contains "not found", exits non-zero
- `config info name-that-doesnt-exist` — error message contains "not found", exits non-zero

#### removed command paths

Table-driven, covers representative paths from all four deleted noun groups:

- `config agent` — error
- `config agent add` — error
- `config agent default claude` — error
- `config agent edit claude` — error
- `config role add` — error
- `config role list` — error
- `config context order` — error
- `config task remove review` — error

Run tests:

```
scripts/invoke-tests
```

## Phases

### Phase 1 — DR (complete)

dr-044 written and accepted. dr-017 and dr-029 marked superseded.

### Phase 2 — start config

Main body of work:

- Delete `config_agent.go`, `config_role.go`, `config_context.go`, `config_task.go`
- Create new verb command files
- Update `config.go`, `config_helpers.go`, `config_interactive.go`, `config_order.go`
- Rewrite test files

The logical order within Phase 2:

1. Update `config.go` — register new verbs, remove noun registrations
2. Implement `config list` and `config info` — read-only, lower risk
3. Implement `config add` — carries over from noun-group add commands
4. Implement `config edit` — carries over from noun-group edit commands
5. Implement `config remove` — carries over from noun-group remove commands
6. Update `config_order.go` — remove dead code (`addConfigContextOrderCommand`, `addConfigRoleOrderCommand`)
7. Update helpers and interactive flows
8. Rewrite tests

Checkpoint: `scripts/invoke-tests` passes before moving to Phase 3.

### Phase 3 — Docs and cleanup

- Update README.md
- Verify shell completion works (Cobra regenerates automatically)
- Final full test run

## Deliverables

Implementation files (new):

- `internal/cli/config_add.go`
- `internal/cli/config_edit.go`
- `internal/cli/config_remove.go`
- `internal/cli/config_list.go`
- `internal/cli/config_info.go`

Implementation files (modified):

- `internal/cli/config.go`
- `internal/cli/config_helpers.go`
- `internal/cli/config_interactive.go`
- `internal/cli/config_order.go` (remove dead code only — category arg delivered by p-037)

Implementation files (deleted):

- `internal/cli/config_agent.go`
- `internal/cli/config_role.go`
- `internal/cli/config_context.go`
- `internal/cli/config_task.go`

Test files (rewritten/updated):

- `internal/cli/config_test.go`
- `internal/cli/config_integration_test.go`
- `internal/cli/config_order_test.go`
- `internal/cli/config_helpers_test.go`
- `internal/cli/config_interactive_test.go`

Documentation:

- `README.md`

Design Records:

- dr-044 (already created)

## Dependencies

Requires all config-touching projects complete:

- p-008 Configuration Editing
- p-013 CLI Configuration Testing
- p-017 CLI Config Edit Flags
- p-018 CLI Interactive Edit Completeness
- p-023 CLI Config Reorder
- p-027 CLI Content Source Menu Extraction
- p-033 CLI Show Noun Subcommand Removal (complete)
- p-034 CLI Config Add/Edit Flags Removal
- p-035 CLI Config Open Command
- p-036 CLI Config Types Migration
- p-037 CLI Config Order Category Argument
