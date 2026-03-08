# P-041: Unified Asset Resolution

- Status: Complete
- Started: 2026-03-08
- Completed: 2026-03-08

## Overview

Asset resolution (for tasks, roles, and agents) uses a tiered fast-path approach that has repeatedly introduced bugs. The pattern is: find an installed match, return early if it looks unambiguous, apply a registry guard to catch edge cases. Each time a new fast path is added, the guard is missed or misplaced.

The root bug: when exactly 1 installed substring match exists, resolution returns immediately without checking the registry for additional matches. If the registry has a second matching asset, the selection menu is never shown. The registry guard added in p-038 only covers the exact-match path, not the single-substring path. The same structural gap exists in both `executeTask` (task.go) and `resolveAsset` (resolve.go).

The fix is architectural: remove the fast paths and guards entirely, replacing them with a single two-phase approach that is correct by construction.

## Goals

1. Eliminate the guard-per-fast-path pattern from task resolution
2. Eliminate the guard-per-fast-path pattern from role/agent resolution (`resolveAsset`)
3. Restore correct selection menu behaviour when installed + registry matches exceed 1
4. Ensure the refactor does not regress single-match or exact-match cases
5. Leave no structural gap for this class of bug to recur

## Technical Approach

Two-phase resolution for both sites:

Phase 1: Full exact name match (e.g. `cwd/agents-md/create` typed verbatim) - use directly, no registry check needed. This is unambiguous by definition.

Phase 2: Everything else - collect all candidates from installed config and registry (cached index, already memoised within a run), merge, then decide:
- 0 matches: error
- 1 match: use it (auto-install if registry-only)
- N matches: show selection menu

The `ensureIndex()` call is already memoised via `r.didFetch` and cache-backed (24h TTL from p-038), so calling it unconditionally has no performance cost after the first call.

## Scope

In Scope:

- `executeTask` resolution logic in `internal/cli/task.go`
- `resolveAsset` resolution logic in `internal/cli/resolve.go`
- Removing the `registryGuard` block and single-substring fast paths
- Updating tests to cover the fixed behaviour

Out of Scope:

- Context resolution (`resolveContexts`) - uses different semantics (pass-through for unmatched terms)
- Model resolution - no registry involvement
- Changes to `findInstalledTasks`, `findRegistryTasks`, `mergeTaskMatches`, or search functions
- Changes to registry, cache, or orchestration packages

## Current State

- `executeTask` (task.go ~line 170-370): three-tier logic with exact match, registry guard (`if resolved != ""`), single-substring fast path (`len == 1 && !registryGuardTriggered`), then combined search. Guard never fires from the substring path.
- `resolveAsset` (resolve.go ~line 89-176): same pattern - exact/short name match, single-substring fast path at line 117-121 with no registry check, then combined search.
- `r.ensureIndex()` is memoised: second call returns cached `r.index` immediately (resolve.go ~line 665-668).
- Tests are in `package cli` (same package), so `r.didFetch = true` and `r.index = &testIndex` can be set directly on the resolver struct to inject a fake registry index for testing the combined-match path without network access. No additional test infrastructure is needed.

## Success Criteria

- [x] `start task agents-md` shows selection menu when registry has additional matches beyond the single installed match
- [x] `start --role <name>` and `start --agent <name>` show selection menu under the same conditions
- [x] Exact full name input (e.g. `start task cwd/agents-md/create`) still resolves directly without a menu
- [x] Single unambiguous match (installed only, no registry match) still resolves directly without a menu
- [x] All existing task and resolve tests pass
- [x] No new registry network calls introduced (cache covers it)

## Deliverables

- `internal/cli/task.go` - simplified two-phase task resolution
- `internal/cli/resolve.go` - simplified two-phase asset resolution in `resolveAsset`
- Updated tests covering the previously broken case
