# P-030: Assets Index Setting

- Status: Pending
- Started:
- Completed:

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
