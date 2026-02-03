# p-021: Auto-Setup Default Assets

- Status: Pending
- Started: (not yet started)
- Completed: (not yet completed)
- Issue: #6

## Overview

Enhance auto-setup to install commonly-needed assets during first-run configuration. Currently auto-setup only configures the agent, requiring users to manually install contexts like `cwd/agents-md` that provide repository-specific guidelines.

This project extracts asset installation logic to a shared internal package and extends auto-setup to install default contexts that nearly all users will want.

## Goals

1. Extract asset installation logic from CLI commands to shared package
2. Install `contexts/cwd/agents-md` during auto-setup by default
3. Handle installation errors gracefully (warn and continue, don't fail setup)
4. Update DR-018 to document the change

## Scope

In Scope:

- Create `internal/assets` package with installation logic
- Move `SearchResult` type and `SearchIndex` function to shared location
- Move asset installation helpers (extract, format, write) to shared location
- Update CLI commands to use new shared package
- Add default asset installation to auto-setup flow
- Install only `contexts/cwd/agents-md` initially
- Graceful error handling (warn and continue)

Out of Scope:

- Installing roles or tasks by default (contexts only)
- User-configurable list of default assets
- Command to install defaults for existing configs
- Installing multiple default contexts
- Interactive prompts about default assets

## Success Criteria

- [ ] Created `internal/assets/install.go` with installation logic
- [ ] Created `internal/assets/search.go` with search functionality
- [ ] `internal/cli/assets_add.go` uses `assets.InstallAsset()`
- [ ] Auto-setup calls `assets.InstallAsset()` for `cwd/agents-md`
- [ ] Installation errors during auto-setup are logged but don't fail setup
- [ ] Updated DR-018 with default assets section
- [ ] All existing tests pass
- [ ] Manual test: fresh install includes `cwd/agents-md` in global config

## Current State

Verified: 2026-02-03

Asset installation logic:

- Located in `internal/cli/assets_add.go` (runAssetsAdd function, lines 105-147)
- Includes: search, fetch, extract, format, write to config
- Used only by CLI commands, not accessible to orchestration package
- Key functions to extract:
  - `extractAssetContent()` (lines 222-266) - Loads module and extracts CUE content
  - `formatAssetStruct()` (lines 268-308) - Formats CUE value as struct with origin field
  - `formatFieldValue()` (lines 310-379) - Formats individual field values as CUE syntax
  - `assetTypeToConfigFile()` (lines 206-220) - Maps category to config filename
  - `writeAssetToConfig()` (lines 388-460) - Writes asset to config file
  - Helper functions: `findAssetKey()`, `findMatchingBrace()`, `findOpeningBrace()`, `updateAssetInConfig()`

Search functionality:

- `SearchResult` type in `internal/cli/assets_search.go` (lines 14-20)
- `searchIndex()` function (lines 78-100) - Searches all categories
- Helper functions: `searchCategory()`, `matchScore()`, `categoryOrder()`

Auto-setup flow:

- Located in `internal/orchestration/autosetup.go`
- Currently only writes agent config and settings (lines 315-341)
- Writes to `~/.config/start/agents.cue` and `settings.cue`
- DR-018 (dr-018-cli-auto-setup.md) documents current behaviour
- No default asset installation currently implemented

Default asset to install:

- `cwd/agents-md` context from registry
- Purpose: Reads repository AGENTS.md file for project-specific guidelines
- Location: `github.com/grantcarthew/start-assets/contexts/cwd/agents-md`
- Fields: required=true, default=true, file="AGENTS.md", includes git remote command

Package structure:

- `internal/cli` - Cobra command handlers
- `internal/orchestration` - Core orchestration (auto-setup, composition)
- `internal/registry` - Registry interaction (NewClient, FetchIndex, Fetch, ResolveLatestVersion)
- No `internal/assets` package exists yet (confirmed via directory listing)

Testing patterns:

- Table-driven tests with `t.Run()`
- Real file operations via `t.TempDir()` (not mocked)
- Parallel tests where appropriate (`t.Parallel()`)
- Test real behaviour over mocks (CUE validation, file I/O)
- See `dr-024-testing-strategy.md` for full testing approach

Dependencies:

- Go 1.24.0
- CUE v0.15.1
- Cobra v1.10.2
- All prerequisite projects complete (p-006, p-007)

## Deliverables

Files:

- `internal/assets/install.go` - Asset installation logic
- `internal/assets/search.go` - Search types and functions
- Updated `internal/cli/assets_add.go` - Uses assets package
- Updated `internal/cli/assets_search.go` - Uses assets package
- Updated `internal/cli/task.go` - Uses assets package for installation
- Updated `internal/cli/assets_update.go` - Uses assets package for installation
- Updated `internal/orchestration/autosetup.go` - Installs default assets

Design Records:

- Updated DR-018 - Document default asset installation

Tests:

- `internal/assets/search_test.go` - Unit tests for search functionality (moved from cli/assets_test.go)
- `internal/assets/install_test.go` - Unit tests for installation logic
- Updated `internal/cli/assets_test.go` - CLI-specific tests remain
- Updated `internal/orchestration/autosetup_test.go` - Test default asset installation

## Technical Approach

Package Structure:

```
internal/assets/
  search.go    - SearchResult type, SearchIndex() function
  install.go   - InstallAsset() and helpers (extract, format, write)
```

Migration Strategy:

1. Create `internal/assets/search.go`
   - Move `SearchResult` from `cli/assets_search.go`
   - Move `searchIndex` as `SearchIndex` (export)
   - Move helper functions (searchCategory, matchScore, categoryOrder)

2. Create `internal/assets/install.go`
   - Move installation logic from `runAssetsAdd` (lines 105-147)
   - Export as `InstallAsset(ctx, client, selected, configDir, quiet, stdout)`
   - Move helpers: extractAssetContent, formatAssetStruct, formatFieldValue
   - Move helpers: writeAssetToConfig, assetTypeToConfigFile

3. Update CLI commands
   - `assets_search.go`: import assets, call `assets.SearchIndex()`
   - `assets_add.go`: import assets, call `assets.InstallAsset()`
   - `task.go`: import assets, call `assets.InstallAsset()` (currently uses extractAssetContent + writeAssetToConfig)
   - `assets_update.go`: import assets, call `assets.InstallAsset()` or extracted helpers

4. Update auto-setup
   - Add `installDefaultAssets()` method
   - Look up `cwd/agents-md` in index
   - Call `assets.InstallAsset()` with error handling
   - Log warnings but continue on error

Default Assets:

- Currently only `contexts/cwd/agents-md`
- Hardcoded in `installDefaultAssets()` method
- Future: could be configurable or marked in registry index

Error Handling:

- Installation errors logged to stderr
- Auto-setup continues (returns success)
- User sees: "Warning: Failed to install cwd/agents-md: ..."
- Agent setup completes normally

## Dependencies

Requires:

- p-006 (Auto-Setup) - Complete
- p-007 (Package Management) - Complete

## Decision Points

Approved: 2026-02-03

1. **Duplicate Installation Handling**

   Decision: A - Skip silently (check before attempting installation)

   Avoids noise during auto-setup when asset already exists.

2. **Success Logging**

   Decision: B - Silent installation (only log errors)

   Keeps auto-setup output clean, consistent with silent duplicate handling.

3. **Registry Unavailability**

   Decision: B - Warn and continue (agent configured, contexts missing)

   If agent fetch succeeds but default context fetch fails, complete auto-setup with warning. Context is helpful but not essential. User can install later via `start assets add cwd/agents-md`.

   Note: Registry client already has 3 retries with exponential backoff built-in.

## Testing Strategy

Unit Tests:

- `internal/assets/search_test.go` - Test SearchIndex functionality
- `internal/assets/install_test.go` - Test InstallAsset with mock registry
- `internal/orchestration/autosetup_test.go` - Test default asset installation

Integration Tests:

- Fresh auto-setup installs `cwd/agents-md` to global config
- Installation failure doesn't break auto-setup
- Existing assets tests still pass with new package structure

Manual Tests:

1. Remove config: `rm -rf ~/.config/start`
2. Run auto-setup: `start`
3. Verify: `cat ~/.config/start/contexts.cue` contains `cwd/agents-md`

## Notes

Design Decision - Why `internal/assets`:

- Shared logic needed by both CLI and orchestration
- Wrong to have orchestration depend on cli package (dependency inversion)
- Clean separation of concerns (assets is domain logic, cli is interface)
- Makes asset operations testable in isolation

Alternative Considered - Duplicate logic in autosetup.go:

- Rejected: code duplication, maintenance burden
- 200+ lines of complex CUE extraction/formatting logic
- Single source of truth is better

Future Enhancements:

- Configurable default asset list
- Registry index could mark assets as "recommended"
- Command to install defaults: `start assets install-defaults`
- Interactive prompt: "Install recommended contexts?"
