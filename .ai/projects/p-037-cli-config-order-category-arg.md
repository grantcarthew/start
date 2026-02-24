# P-037: CLI Config Order Category Argument

- Status: Pending
- Started:
- Completed:

## Overview

`start config order` currently requires an interactive menu to choose between contexts and roles. This project adds an optional category argument so users can navigate directly to the reorder flow without the menu step.

This is extracted from p-032 as an independent, self-contained change to `config_order.go`. It has no dependency on the noun-group deletion and can be shipped before p-032 begins.

## Current State

`addConfigOrderCommand` in `config_order.go:17` registers the order command with `Args: noArgsOrHelp`. `runConfigOrder` always prompts with `promptSelectCategory` — there is no way to bypass the menu from the command line.

The noun-group files currently register `addConfigContextOrderCommand` and `addConfigRoleOrderCommand` as subcommands of `config context` and `config role`. These are separate from the top-level order command and are not affected by this project.

`config_order_test.go` (754 lines) tests the existing no-argument interactive flow. Tests for the new category argument will be added alongside.

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
start config order agent       # non-orderable: ignore and show the context/role menu
start config order task        # non-orderable: ignore and show the context/role menu
```

Non-orderable categories silently fall back to the menu rather than returning an error, keeping the interaction forgiving.

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

## Dependencies

None — can start immediately.

Unblocks: p-032
