# P-036: CLI Config Types Migration

- Status: Pending
- Started:
- Completed:

## Overview

The four noun-group config files (`config_agent.go`, `config_role.go`, `config_context.go`, `config_task.go`) each contain struct type definitions, loaders, and writers buried among their command implementation code. P-032 will delete these files entirely, but the types and functions they contain are shared across `config.go`, `config_order.go`, and `config_interactive.go`.

This project extracts all shared types, loaders, and writers into a new `config_types.go` before p-032 begins. It is a pure refactor — no behaviour change, no interface change, no test impact beyond confirming the move compiled correctly.

## Current State

Types, loaders, and writers currently defined in the noun-group files:

- `AgentConfig` struct — `config_agent.go:628`
- `loadAgentsForScope`, `loadAgentsFromDir` — `config_agent.go:642-717`
- `writeAgentsFile` — `config_agent.go:721-774`
- `loadConfigForScope` — `config_agent.go:777-804`
- `getDefaultAgentFromConfig` — `config_agent.go:807-814`
- `RoleConfig` struct — `config_role.go:514`
- `loadRolesForScope`, `loadRolesFromDir` — `config_role.go:528-594`
- `writeRolesFile` — `config_role.go:598`
- `ContextConfig` struct — `config_context.go:572`
- `loadContextsForScope`, `loadContextsFromDir` — `config_context.go:587-654`
- `writeContextsFile` — `config_context.go:658`
- `TaskConfig` struct — `config_task.go:521`
- `loadTasksForScope`, `loadTasksFromDir` — `config_task.go:535-599`
- `writeTasksFile` — `config_task.go:603`

These are all called by `config.go` (`runConfigList`), `config_order.go` (`reorderContexts`, `reorderRoles`), and `config_interactive.go` (`loadNamesForCategory`). They must exist in a non-deleted file before p-032 can proceed.

`loadForScope[T]` in `config_helpers.go` is the generic base called by all typed loaders — it stays in `config_helpers.go`.

`config_types.go` will require these imports:

```go
import (
    "errors"
    "fmt"
    "os"
    "sort"
    "strings"

    "cuelang.org/go/cue"
    "github.com/grantcarthew/start/internal/config"
    internalcue "github.com/grantcarthew/start/internal/cue"
)
```

After removing the moved code, these imports become unused in the noun-group files and must be removed:

- `config_agent.go`: remove `errors`, `cuelang.org/go/cue`, `internalcue` (sort, config, and remaining stdlib are still used)
- `config_role.go`: remove `errors`, `cuelang.org/go/cue`, `internalcue`
- `config_context.go`: remove `errors`, `cuelang.org/go/cue`, `internalcue`
- `config_task.go`: remove `errors`, `cuelang.org/go/cue`, `internalcue`, `sort`

Note: the remaining code in each file calls `loadConfigForScope`/`loadAgentsForScope`/etc. by name only — the `cue.Value` type is inferred via `:=` and never referenced explicitly in the remaining code, so `cuelang.org/go/cue` is safe to remove.

## Goals

1. Create `internal/cli/config_types.go` containing all four struct types, their loaders, their writers, `loadConfigForScope`, and `getDefaultAgentFromConfig`
2. Remove those definitions from the four noun-group files
3. Confirm `go test ./internal/cli/...` passes with no changes to behaviour

## Scope

In Scope:

- Moving type definitions, loaders, and writers to `config_types.go`
- Removing the moved code from the noun-group files

Out of Scope:

- Any behaviour change
- Adding new types or functions
- Changing function signatures
- Modifying tests (they should pass unchanged)
- Deleting the noun-group files (done by p-032)

## Success Criteria

- [ ] `internal/cli/config_types.go` exists and contains all four struct types, loaders, writers, `loadConfigForScope`, and `getDefaultAgentFromConfig`
- [ ] The moved definitions are removed from `config_agent.go`, `config_role.go`, `config_context.go`, `config_task.go`
- [ ] `go test ./internal/cli/...` passes
- [ ] No function signatures changed

## Deliverables

- `internal/cli/config_types.go` (new)
- `internal/cli/config_agent.go` (modified — definitions removed)
- `internal/cli/config_role.go` (modified — definitions removed)
- `internal/cli/config_context.go` (modified — definitions removed)
- `internal/cli/config_task.go` (modified — definitions removed)

## Dependencies

None — can start immediately.

Unblocks: p-032
