# P-038: CLI Index Cache

- Status: Complete
- Started: 2026-03-01
- Completed: 2026-03-02

## Overview

When `start task <term>` resolves a task, the multi-match guard only checks installed tasks. Registry tasks are invisible unless the installed search comes up empty, at which point a fresh index is fetched. This causes unexpected behaviour: `start task start` silently runs `cwd/dotai/project/start` (short name match on last path segment) instead of showing a selection list that includes registry tasks like `start/assets/agent/create`.

This project adds index caching so `start task` can check both installed and registry tasks without a network call on every invocation.

## Goals

1. Cache the registry index version and fetch timestamp in `~/.cache/start/cache.cue`
2. Use the cached index in `start task` multi-match resolution without a network call
3. Refresh the cache when stale (> 24 hours) or missing
4. Update the cache as a side effect whenever any command fetches the index
5. Add cache staleness check to `start doctor`
6. Respect XDG_CACHE_HOME for the cache directory

## Scope

In Scope:

- Cache file creation and management in XDG cache directory
- Reading/writing `~/.cache/start/cache.cue` with `index_updated` and `index_version` fields
- Modifying `start task` resolution to use cached index for multi-match checking
- Updating all commands that fetch the index to write cache metadata
- Doctor check for cache existence and staleness
- 24-hour staleness threshold as a hardcoded constant

Out of Scope:

- Configurable staleness threshold (add later if requested)
- Schema changes in start-assets (cache is a CLI artifact)
- Caching the index data itself (CUE's module cache handles this)

## Current State

Registry fetching:
- `ensureIndex()` in `resolve.go:646-695` creates a `registry.NewClient()` and calls `client.FetchIndex()` with lazy caching on the resolver struct (`didFetch` flag)
- `FetchIndex()` in `registry/index.go:48-61` calls `ResolveLatestVersion()` then `Fetch()` then `LoadIndex()`. Returns `(*Index, error)` only — does not return the resolved canonical version string
- `ResolveLatestVersion()` in `registry/client.go:89-121` calls `ModuleVersions()` (network call) to resolve `@v0` to canonical `@v0.x.y`. Short-circuits if version is already canonical (line 95)
- `Fetch()` in `registry/client.go:50-80` calls `c.registry.Fetch()` which uses CUE's module cache. For already-cached canonical versions, this should serve from `~/.cache/cue/mod/` without a network call
- `NewClient()` creates the registry resolver (local operation, no network call)

Call sites for `FetchIndex()` (8 total, plus 1 manual):
- `assets_add.go:107`, `assets_info.go:85`, `assets_list.go:244`, `assets_search.go:89`, `assets_update.go:130` — asset commands
- `resolve.go:686` — resolver's `ensureIndex()`
- `search.go:161` — unified search command
- `autosetup.go:66` — auto-setup (uses built-in default, runs before user settings)
- `assets_index.go:83-96` — calls `ResolveLatestVersion()` + `Fetch()` directly (already has resolved version)

Multi-match guard:
- `task.go:176-200` — after `findExactInstalledName()` finds a match, calls `findInstalledTasks()` for substring search. If `len(installedMatches) > 1`, resets `resolved = ""` to fall through to full search
- Only checks installed config — registry tasks are invisible to the guard
- Full registry fetch only happens at step 2 (`task.go:236`) when no installed match found

XDG cache support:
- `internal/config/paths.go` has `globalConfigDir()` with XDG_CONFIG_HOME support
- `internal/orchestration/composer.go:810-821` has `GetCUECacheDir()` using `os.UserCacheDir()` which respects XDG_CACHE_HOME
- No `~/.cache/start/` directory exists yet

Doctor pattern:
- `internal/doctor/checks.go` — each check is a function returning `doctor.SectionResult` with `[]CheckResult` items
- `internal/cli/doctor.go:73-180` — `prepareDoctor()` builds report by appending section results sequentially
- Status types: `StatusPass`, `StatusWarn`, `StatusFail`, `StatusInfo`, `StatusNotFound`

## Technical Approach

### Cache package (`internal/cache/`)

Cache file at `~/.cache/start/cache.cue` (or `$XDG_CACHE_HOME/start/cache.cue`):

```cue
index_updated: "2026-03-01T10:30:00+10:00"
index_version: "github.com/grantcarthew/start-assets/index@v0.3.46"
```

API:

```go
type IndexCache struct {
    Updated time.Time
    Version string
}

func Dir() (string, error)
func ReadIndex() (IndexCache, error)
func WriteIndex(version string) error
func (c IndexCache) IsFresh(maxAge time.Duration) bool
```

`ReadIndex` returns error if file is missing or malformed — callers treat any error as "no cache". `WriteIndex` creates the directory if needed. Cache write failures are best-effort — log debug message and continue, never fail the command.

### `FetchIndex()` signature change

Change from `(*Index, error)` to `(*Index, string, error)`. The string is the resolved canonical version (e.g., `github.com/grantcarthew/start-assets/index@v0.3.46`). All 8 callers updated.

### Cache-aware `ensureIndex()` in `resolve.go`

Modified flow:

1. Read `cache.cue` via `cache.ReadIndex()`
2. If fresh (< 24h): pass canonical `index_version` to `FetchIndex()` — `ResolveLatestVersion()` short-circuits on canonical version (no `ModuleVersions()` network call), `Fetch()` serves from CUE's module cache
3. If stale or missing: call `FetchIndex()` with the default `@v0` path as normal (network call for version resolution). Write `cache.cue` with returned version
4. Result cached on resolver via `didFetch` flag as before

### Multi-match guard in `task.go`

Extended guard at lines 183-200:

1. `findInstalledTasks()` — get installed substring matches
2. Call `ensureIndex()` (now cache-aware) — get index without network call if cache is fresh
3. If index available: `findRegistryTasks()` — get registry matches
4. If installed + registry matches > 1 → reset `resolved = ""`, fall through to selection list
5. Step 2/3 reuse the same index via `didFetch` — no redundant fetch

### Asset command call sites (always-fresh, write cache)

`assets_add.go`, `assets_info.go`, `assets_list.go`, `assets_search.go`, `assets_update.go`, `search.go` — accept version from `FetchIndex()`, call `cache.WriteIndex(version)` after successful fetch.

`assets_index.go` — already has resolved version from `ResolveLatestVersion()`, call `cache.WriteIndex(resolvedPath)`.

`autosetup.go` — write cache to seed it on first run.

### Doctor integration

`resolveIndexVersion()` at `doctor.go:242` — read `cache.cue` first for the version. Fall back to current network call if cache is missing.

New `CheckCache()` in `internal/doctor/checks.go`:
- Missing cache → `StatusNotFound` with fix suggestion
- Fresh (< 24h) → `StatusPass` with age (e.g., "2 hours ago")
- Stale (>= 24h) → `StatusWarn` with age (e.g., "3 days ago")

Appended in `prepareDoctor()` after the Version section.

### Testing

`internal/cache/` — unit tests for `ReadIndex`, `WriteIndex`, `IsFresh`, `Dir`. Use `t.TempDir()` with `XDG_CACHE_HOME` override. Cases: valid cache, missing file, malformed file, fresh vs stale, directory creation.

`ensureIndex()` — test with seeded cache file. Verify fresh cache skips "Fetching registry index..." output.

Multi-match guard — extend task resolution tests to verify guard triggers when cached index contains matching registry tasks.

`FetchIndex()` — no existing tests; add tests for the new `(*Index, string, error)` return signature using mock registry.

## Success Criteria

- [ ] `~/.cache/start/cache.cue` is created/updated when any command fetches the registry index
- [ ] `start task start` shows a selection list (not silent execution) when registry tasks also match
- [ ] `start task` with a fresh cache does not make a network call for the multi-match check
- [ ] `start task` with a stale cache (> 24h) fetches fresh and updates the cache
- [ ] `start task` with no cache falls back to current behaviour (fetch fresh)
- [ ] `start doctor` reports index cache status and age
- [ ] XDG_CACHE_HOME is respected for cache directory location
- [ ] Existing tests pass; new tests cover cache read/write and staleness logic

## Deliverables

- `internal/cache/` package with `ReadIndex`, `WriteIndex`, `IsFresh`, `Dir` functions and tests
- `FetchIndex()` signature changed to return `(*Index, string, error)`
- Cache-aware `ensureIndex()` in `resolve.go`
- Extended multi-match guard in `task.go` with registry awareness
- 9 `FetchIndex`/index fetch call sites updated to write cache
- `resolveIndexVersion()` in `doctor.go` updated to read cache first
- `CheckCache()` in `internal/doctor/checks.go` with tests
- New `FetchIndex` tests covering the `(*Index, string, error)` return signature

## Decisions

1. Cache package location: `internal/cache/` — clean separation, cache is a CLI artifact not a registry concept
2. `FetchIndex()` signature change: return `(*Index, string, error)` — all 8 callers get the canonical version, enforces cache writes everywhere
3. Cache-aware `ensureIndex()` for all resolver paths — one code path, 24-hour staleness window applies to both task resolution and flag resolution. Asset commands remain always-fresh (they don't use the resolver) and write the cache.

## Dependencies

- p-024 (CLI Flag Asset Search) — established the resolver and multi-match patterns
