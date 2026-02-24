# P-037: CLI Config Order Category Argument

- Status: Completed
- Started: 2026-02-24
- Completed: 2026-02-24

## Overview

`start config order` currently requires an interactive menu to choose between contexts and roles. This project adds an optional category argument so users can navigate directly to the reorder flow without the menu step.

This is extracted from p-032 as an independent, self-contained change to `config_order.go`. It has no dependency on the noun-group deletion and can be shipped before p-032 begins.

## Current State

`addConfigOrderCommand` in `config_order.go:17` registers the order command with `Args: noArgsOrHelp`. `runConfigOrder` at `config_order.go:34` always calls `promptSelectCategory` — there is no way to bypass the menu from the command line.

The noun-group files currently register `addConfigContextOrderCommand` and `addConfigRoleOrderCommand` as subcommands of `config context` and `config role`. These are separate from the top-level order command and are not affected by this project.

`config_order_test.go` (755 lines) tests the existing no-argument interactive flow. Tests for the new category argument will be added alongside.

### Reference Implementation

`config_open.go` provides the exact pattern to follow:

- `cobra.MaximumNArgs(1)` replaces `noArgsOrHelp` on the command
- `Use` string updated to `"order [category]"`
- `RunE` checks `len(args) > 0` to extract the optional category before any prompting
- `checkHelpArg` remains at the top of `RunE` — it still works with `cobra.MaximumNArgs(1)` since "help" is a valid single arg
- When no category is given and stdin is not a terminal, `config_open.go` returns an error; `runConfigOrder` already returns "interactive reordering requires a terminal" in that case, so the existing non-terminal handling is unchanged

### Terminal Check and Test Strategy

`runConfigOrder` calls `isTerminal(stdin)` before any routing. This means integration tests through `NewRootCmd().Execute()` will always get "interactive reordering requires a terminal" regardless of what category arg is passed — the routing logic is never reached in tests that pipe stdin.

The cleanest test strategy is:

- Extract the category routing to a pure helper (e.g. `resolveOrderCategory(arg string) string`) that returns "contexts", "roles", or "" for fallback — this can be table-driven tested without any I/O
- The functional paths (`reorderContexts`, `reorderRoles`) are already covered by existing tests
- Integration via cobra is only viable for error-path tests (non-terminal stdin, which rejects every path equally)

## Goals

1. Accept an optional `[category]` argument on `start config order`
2. Route `order context` directly to `reorderContexts`, `order role` directly to `reorderRoles`
3. Treat `order agent` and `order task` as non-orderable — silently fall back to the interactive menu
4. Update `config_order_test.go` to cover the new argument paths

## Scope

In Scope:

- `internal/cli/config_order.go` — argument handling in `addConfigOrderCommand` and `runConfigOrder`
- `internal/cli/config_order_test.go` — new tests for category argument behaviour

Out of Scope:

- `addConfigContextOrderCommand` and `addConfigRoleOrderCommand` — these are noun-group subcommands removed by p-032, not this project
- Any other config command
- Plural aliases for category arguments (p-032 handles that pattern consistently)

## Command Behaviour

```
start config order             # prompt: context or role? (unchanged)
start config order context     # go straight to context reorder
start config order role        # go straight to role reorder
start config order agent       # non-orderable: fall back to the context/role menu silently
start config order task        # non-orderable: fall back to the context/role menu silently
start config order xyz         # unknown: print error, then fall back to the context/role menu
```

Non-orderable categories (agent, task) silently fall back to the menu. Truly unknown categories print an error then fall back to the menu — the user is informed but not left stranded.

## Success Criteria

- [ ] `start config order context` runs context reorder without a menu prompt
- [ ] `start config order role` runs role reorder without a menu prompt
- [ ] `start config order agent` falls back to the context/role menu without an error
- [ ] `start config order task` falls back to the context/role menu without an error
- [ ] `start config order` with no argument is unchanged — still prompts the menu
- [ ] `go test ./internal/cli/...` passes

## Deliverables

- `internal/cli/config_order.go` (modified)
- `internal/cli/config_order_test.go` (modified — new tests added)

## Decision Points

1. Unknown category argument handling — the project specifies behaviour for "context", "role", "agent", and "task" but is silent on truly unknown values (e.g. `start config order xyz`).

   Decision: print an error ("unknown category %q"), then fall through to the interactive Reorder menu. The user is informed of the mistake but not left stranded.

2. Test strategy for routing — the `isTerminal` check in `runConfigOrder` blocks all cobra-level tests from reaching the routing logic.

   Decision: extract a pure `resolveOrderCategory(arg string) string` helper that maps the argument to "contexts", "roles", or "" (fallback). Test it directly with table-driven cases. Mirrors the `resolveConfigOpenPath` pattern in `config_open.go`.

## Dependencies

None — can start immediately.

Unblocks: p-032
