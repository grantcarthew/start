# P-030: Assets Index Setting

- Status: Complete
- Started: 2026-02-20
- Completed: 2026-02-20

## Overview

The CUE module path for the start-assets index is hardcoded as `IndexModulePath` in `internal/registry/index.go`. All `assets` commands and the `doctor` command use this constant to discover and install assets. Users cannot point `start` at a custom or fork assets repository without modifying source code.

This project adds `assets_index` as an optional field to `#Settings`, threads the configured value through to every consumer, and keeps the existing constant as the default fallback. This unblocks issue #46 (assets validate command), which derives a git clone URL from the index module path.

## Goals

1. Add `assets_index` optional field to `#Settings` schema in `start-assets/schemas/settings.cue`
2. Register `assets_index` as a valid settings key in `internal/cli/config_settings.go` (`validSettingsKeys`)
3. Add a helper in the `registry` package that returns the effective index path (configured value or `IndexModulePath` fallback)
4. Update `registry.FetchIndex` to accept the index path as a parameter rather than using the constant directly
5. Update all three consumers that reference `IndexModulePath` directly to load settings and pass the resolved path
6. Add tests covering the fallback and override behaviour

## Scope

In Scope:
- `start-assets/schemas/settings.cue` — add `assets_index` field
- `internal/registry/index.go` — refactor `FetchIndex` to accept index path parameter; add `EffectiveIndexPath` helper
- `internal/cli/config_settings.go` — add `assets_index` to `validSettingsKeys`
- `internal/cli/assets_index.go` — load settings and use resolved index path
- `internal/cli/assets_add.go`, `assets_list.go`, `assets_search.go`, `assets_update.go`, `assets_info.go` — load settings and use resolved index path where `IndexModulePath` or `FetchIndex` is called
- `internal/cli/search.go` — calls `FetchIndex` for registry-backed search; pass resolved index path
- `internal/cli/resolve.go` — calls `FetchIndex` inside `resolveContext`; pass resolved index path
- `internal/cli/doctor.go` — load settings in `resolveIndexVersion` and pass resolved path
- Unit tests for the new `EffectiveIndexPath` helper and the settings key addition

Out of Scope:
- New CLI commands or subcommands
- Documentation or guide updates
- Publishing a new schema version to the CUE Central Registry (local `start-assets` directory only)
- Issue #46 (assets validate command) — this project is a prerequisite, not an implementation

## Technical Approach

Add a function `EffectiveIndexPath(configured string) string` to `internal/registry/index.go`:

```go
func EffectiveIndexPath(configured string) string {
    if configured != "" {
        return configured
    }
    return IndexModulePath
}
```

Change `FetchIndex` signature to accept the index path:

```go
func (c *Client) FetchIndex(ctx context.Context, indexPath string) (*Index, error)
```

Add a shared helper in the `cli` package (e.g. `resolveAssetsIndexPath() string`) that calls `loadSettingsForScope(false)` and returns the `assets_index` value or empty string. Each `assets` command and `resolveIndexVersion` calls this helper before invoking `FetchIndex` or `ResolveLatestVersion`.

## Success Criteria

- `assets_index` field present in `start-assets/schemas/settings.cue` with constraint `string & !=""`
- `start config settings assets_index <path>` sets the value without error
- `start assets index`, `add`, `list`, `search`, `update`, `info` all use the configured index path when set
- `start doctor` uses the configured index path when checking the registry version
- When `assets_index` is absent, behaviour is identical to today (uses `IndexModulePath`)
- All existing tests pass
- New unit test for `EffectiveIndexPath` with both empty and non-empty inputs

## Deliverables

- Updated `start-assets/schemas/settings.cue` with `assets_index` field
- Updated `internal/registry/index.go` with `EffectiveIndexPath` helper and updated `FetchIndex` signature
- Updated `internal/cli/config_settings.go` with `assets_index` in `validSettingsKeys`
- Updated asset command files to load and use resolved index path
- Updated `internal/cli/doctor.go` to use resolved index path
- Tests for the new helper and settings integration

## Current State

### `start-assets/schemas/settings.cue`

Has three optional fields: `default_agent`, `shell`, `timeout`. No `assets_index` field exists yet.

### `internal/registry/index.go`

`IndexModulePath` constant is defined at the top. `FetchIndex(ctx context.Context)` uses it directly:

```go
func (c *Client) FetchIndex(ctx context.Context) (*Index, error) {
    resolvedPath, err := c.ResolveLatestVersion(ctx, IndexModulePath)
    ...
}
```

`EffectiveIndexPath` does not exist yet. `LoadIndex` and `decodeIndex` are unchanged by this project.

### `internal/cli/config_settings.go`

`validSettingsKeys` currently contains three entries:

```go
var validSettingsKeys = map[string]string{
    "default_agent": "string",
    "shell":         "string",
    "timeout":       "int",
}
```

Three places hardcode the valid keys list in error messages and command help text:
- `showSetting` error: `"Valid settings: default_agent, shell, timeout"`
- `setSetting` error: `"Valid settings: default_agent, shell, timeout"`
- `addConfigSettingsCommand` Long description lists all three settings

All three must be updated to include `assets_index`.

### `internal/cli/assets_index.go`

Does not call `FetchIndex`. Instead calls `ResolveLatestVersion` and `Fetch` directly with `registry.IndexModulePath`:

```go
resolvedPath, err := client.ResolveLatestVersion(ctx, registry.IndexModulePath)
```

Needs to call `resolveAssetsIndexPath()` and pass the result through `registry.EffectiveIndexPath` before `ResolveLatestVersion`.

### `internal/cli/assets_add.go`

`runAssetsAdd` calls `client.FetchIndex(ctx)` directly. Will fail to compile after `FetchIndex` signature change.

### `internal/cli/assets_list.go`

`checkForUpdates(ctx, client, installed)` calls `client.FetchIndex(ctx)` internally. `runAssetsList` is the entry point. The cleanest fix is to load the index path in `runAssetsList` and pass it as a new parameter to `checkForUpdates`.

### `internal/cli/assets_search.go`

`runAssetsSearch` calls `client.FetchIndex(ctx)` directly. Will fail to compile after signature change.

### `internal/cli/assets_update.go`

`runAssetsUpdate` calls `client.FetchIndex(ctx)` directly. Will fail to compile after signature change.

### `internal/cli/assets_info.go`

`runAssetsInfo` calls `client.FetchIndex(ctx)` directly. Will fail to compile after signature change.

### `internal/cli/search.go`

`runSearch` calls `client.FetchIndex(ctx)` as part of a graceful-fallback registry search block. Will fail to compile after the `FetchIndex` signature change. Needs to call `resolveAssetsIndexPath()` and pass the result.

### `internal/cli/resolve.go`

A private `resolveContext` helper method (around line 684) calls `client.FetchIndex(ctx)` with graceful fallback. Will fail to compile after the `FetchIndex` signature change. Needs to call `resolveAssetsIndexPath()` and pass the result.

### `internal/cli/doctor.go`

`resolveIndexVersion()` calls `client.ResolveLatestVersion(ctx, registry.IndexModulePath)` directly. Needs to call `resolveAssetsIndexPath()` and use `registry.EffectiveIndexPath` before calling `ResolveLatestVersion`.

### Shared helper placement

`loadSettingsForScope` lives in `config_settings.go`. The `resolveAssetsIndexPath()` helper naturally belongs there alongside it, as a package-level function accessible to all `cli` command files.

### Existing tests

No existing tests call `FetchIndex` directly — `client_test.go` tests `Fetch`, `ResolveLatestVersion`, and `sourceLocToPath`. `index_test.go` tests `LoadIndex` only. No tests will break due to the `FetchIndex` signature change. The new `TestEffectiveIndexPath` test goes in `internal/registry/index_test.go`.
