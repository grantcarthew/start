# P-033: CLI Show Noun Subcommand Removal

- Status: Completed
- Started: 2026-02-24
- Completed: 2026-02-24

## Overview

`start show` has four noun subcommands — `show agent`, `show role`, `show context`, `show task` — that restrict output to a single category. These are inconsistent with the verb-first direction established by dr-044. The top-level `start show [name]` already performs cross-category search-by-name, making the category subcommands redundant.

This project removes the four noun subcommands, cleans up any dead code they leave behind, and updates tests. It is a prerequisite to p-032 (CLI Config Verb-First Refactor), which handles the larger `start config` restructure.

## Required Reading

| Document | Why |
| --- | --- |
| dr-044 | The architectural decision motivating this change |
| dr-024 | Testing strategy — real behaviour over mocks |

## Goals

1. Remove `show agent`, `show role`, `show context`, `show task` subcommands from `show.go`
2. Remove dead code left by subcommand removal (`runShowItem`, `prepareShow` if unused)
3. Verify `start show <name>` cross-category search-by-name still works
4. Update `show_test.go` to cover removed paths and confirm search behaviour

## Scope

In Scope:

- `internal/cli/show.go`
- `internal/cli/show_test.go`

Out of Scope:

- `start show` listing and search behaviour (unchanged)
- `start show --local` and `start show --global` flags (unchanged)
- `start config` (covered by p-032)

## Current State

`show.go` is 755 lines. `addShowCommand` registers four noun subcommands alongside the top-level `show [name]` command:

- `showRoleCmd` — `Use: "role [name]"`, aliases: `["roles"]`, calls `runShowItem(KeyRoles, "Role")`
- `showContextCmd` — `Use: "context [name]"`, aliases: `["contexts"]`, calls `runShowItem(KeyContexts, "Context")`
- `showAgentCmd` — `Use: "agent [name]"`, aliases: `["agents"]`, calls `runShowItem(KeyAgents, "Agent")`
- `showTaskCmd` — `Use: "task [name]"`, aliases: `["tasks"]`, calls `runShowItem(KeyTasks, "Task")`

`runShowItem(cueKey, itemType)` (lines 442–461) is a closure factory returning a `RunE` handler. It is only called by the four subcommands — after their removal it becomes dead code and must be deleted.

`prepareShow(name, scope, cueKey, itemType)` (lines 463–530) is NOT dead code after subcommand removal. It is called by `showVerboseItem` (line 433), which is called throughout `runShowSearch`. It must be kept.

The top-level `show [name]` command dispatches to `runShowListing` (no args) or `runShowSearch` (with arg). No changes to that logic are needed.

`show_test.go` is 1140 lines. The tests fall into three groups:

Group A — direct `prepareShow` calls (keep, function survives):
- `TestPrepareShowAgent`, `TestPrepareShowRole`, `TestPrepareShowContext`, `TestPrepareShowTask`
- `TestPrepareShowLocalNoConfig`, `TestPrepareShowGlobalNoConfig`

Group B — noun subcommand paths via Cobra (remove or convert):
- `TestShowCommandIntegration` has 7 noun subcommand cases:
  - 4 no-name cases: `["show", "agent"]`, `["show", "role"]`, `["show", "context"]`, `["show", "task"]` — previously showed the first item in category; after removal these become cross-category name searches that return not-found errors (no item in test config is named "agent", "role", "context", or "task")
  - 3 two-arg cases: `["show", "agent", "claude"]`, `["show", "context", "environment"]`, `["show", "task", "review"]` — after removal these exceed `MaximumNArgs(1)` and return a cobra args error; remove these tests
- `TestShowGlobalFlag` sub-test "show agent subcommand with --global" uses `["show", "agent", "--global"]`; after removal "agent" is a name search against global config, which holds "global-agent" — the substring "agent" matches "global-agent", so it returns the verbose dump as before; rename sub-test to reflect new meaning

Group C — top-level listing and search (no changes needed):
- `TestShowListingDescriptions`, `TestShowListingNoDescriptions`
- `TestShowCrossCategory`, `TestShowCrossCategoryMultipleExact`
- `TestVerboseDump*`, `TestFormatCUEDefinition`, `TestResolveShowFile`, `TestDeriveCacheDir`

`go test ./internal/cli/...` passes — confirmed green baseline.

## File Changes

Modify:

- `internal/cli/show.go` — remove the four noun subcommand variable declarations and their `showCmd.AddCommand` calls; remove `runShowItem` (dead code); keep `prepareShow` (still used by `showVerboseItem`)

Update:

- `internal/cli/show_test.go` — remove noun subcommand tests; add or update tests to confirm `show agent` is treated as a name search (not a subcommand)

## Success Criteria

- [x] `start show` lists all items (unchanged)
- [x] `start show claude` searches and shows resolved content
- [x] `start show golang/assistant` shows resolved role content
- [x] `start show agent` is treated as a name search, not a subcommand
- [x] `start show role` is treated as a name search, not a subcommand
- [x] `start show context` is treated as a name search, not a subcommand
- [x] `start show task` is treated as a name search, not a subcommand
- [x] `start show --local` works (unchanged)
- [x] `start show --global` works (unchanged)
- [x] All tests pass via `scripts/invoke-tests`
- [x] No tests reference removed noun subcommand paths

## Testing Strategy

Follow dr-024:

- Use real files via `t.TempDir()` — no mocks
- Verify removed subcommand paths return an appropriate error or are treated as name searches
- Verify existing search-by-name and listing behaviour is unchanged

Run tests: `scripts/invoke-tests`

## Deliverables

- `internal/cli/show.go` (modified)
- `internal/cli/show_test.go` (updated)

## Dependencies

- p-029: CLI Show Verbose Inspection (establishes current show.go structure)
