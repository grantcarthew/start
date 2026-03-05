# P-039: CLI JSON Output - Shared Prep and Assets Commands

- Status: Complete
- Started: 2026-03-05
- Completed: 2026-03-05
- GitHub: #69

## Overview

Add rich `--json` output to all assets commands and prepare shared types with JSON tags. This is the first of two projects implementing issue #69. It establishes the JSON tagging conventions and delivers JSON output for the assets command family.

The guiding principle: `--json` output should include every field available to the human-readable output. The JSON is the machine interface to the same data, not a lossy summary.

## Goals

1. Add JSON tags to all shared structs that assets commands marshal
2. Enrich existing `assets list --json` to include config fields (tags, description, models)
3. Add `--json` to `assets search`, `assets info`, `assets validate`, `assets update`
4. Follow the established pattern from `assets list` and `assets index`

## Scope

In Scope:

- JSON tags on `SearchResult` (`internal/assets/search.go:14`)
- JSON tags on `InstalledAsset` enrichment (already tagged, needs richer data)
- JSON tags on `UpdateResult` (`internal/cli/assets_update.go:27`) with `error` field serialised as string
- Exported JSON structs for `validateModuleResult` and `validateCatResult` (`internal/cli/assets_validate.go:33,41`) since these have unexported fields
- `--json` flag and marshal logic for `assets search`, `assets info`, `assets validate`, `assets update`
- Tests for each command's JSON output

Out of Scope:

- Config commands (`config list`, `config info`, `config search`) - covered by p-040
- Doctor command - covered by p-040
- Top-level `search` command - covered by p-040
- Changes to human-readable output

## Current State

Existing JSON output:
- `assets list --json` works: marshals `[]InstalledAsset` via `json.MarshalIndent` at `assets_list.go:157`
- `assets index --json` works: marshals `*registry.Index` via `printJSONIndex` at `assets_index.go:111`
- Both follow the pattern: `jsonFlag, _ := cmd.Flags().GetBool("json")` then early return with JSON output

Struct analysis:

`SearchResult` (`internal/assets/search.go:14`):
- No JSON tags on any field
- Fields: `Category string`, `Name string`, `Entry registry.IndexEntry` (value, not pointer), `MatchScore int`
- `registry.IndexEntry` (`internal/registry/index.go:22`) already has full JSON tags: `module`, `description`, `tags`, `version`, `bin`
- Entry is a value field, not embedded; JSON will nest it as `"entry": {...}` which is clean

`InstalledAsset` (`internal/cli/assets_list.go:28`):
- Already has JSON tags: `category`, `name`, `version`, `latestVersion`, `updateAvailable`, `scope`, `origin`, `configFile`
- Missing: `description`, `tags`, `models` - all available in the CUE value during `collectInstalledAssets` at `assets_list.go:171`
- The CUE iteration at `assets_list.go:186` has access to `assetVal` (full CUE value) but only extracts `origin`
- Enrichment requires adding CUE field lookups for `description` (string), `tags` (list), `models` (list, agents only)

`UpdateResult` (`internal/cli/assets_update.go:27`):
- No JSON tags
- Fields: `Asset InstalledAsset`, `OldVersion string`, `NewVersion string`, `Updated bool`, `Error error`
- `Error` is `error` interface which marshals as `{}` not a string; needs a `MarshalJSON` method or a parallel string field

`validateModuleResult` (`internal/cli/assets_validate.go:33`):
- All unexported: `name`, `version`, `status` (custom `validateModuleStatus` int), `issues []string`
- `validateModuleStatus` is `validateModulePass = 0` or `validateModuleFail = 1`
- Used by `validateModules` at `assets_validate.go:520` and printed by `printValidateModules` at `assets_validate.go:720`

`validateCatResult` (`internal/cli/assets_validate.go:41`):
- All unexported: `name string`, `modules []validateModuleResult`
- The validate command has three output sections: index status (`doctor.SectionResult`), per-category modules, and summary stats

Command analysis:

`assets search` (`internal/cli/assets_search.go`):
- No `--json` flag
- Has interactive prompt fallback at `assets_search.go:63` when query is insufficient and stdin is a terminal
- With `--json`, interactive prompts should be skipped (error on insufficient query)
- Output: `printSearchResults` at `assets_search.go:119` - grouped by category

`assets info` (`internal/cli/assets_info.go`):
- No `--json` flag
- Has interactive selection menu at `assets_info.go:116` when multiple matches found
- With `--json`, should output the best match (first result) without interactive prompt
- Output: `printAssetInfo` at `assets_info.go:136` - single asset detail with installation status

`assets validate` (`internal/cli/assets_validate.go`):
- No `--json` flag
- Gated by `--yes` flag at `assets_validate.go:88` (network protection; applies regardless of output format)
- Three output sections: `printValidateIndexSection`, `printValidateModules`, `printValidateStats`
- Uses `doctor.SectionResult` and `doctor.CheckResult` types for index section
- Note: `doctor.SectionResult` and `doctor.CheckResult` (`internal/doctor/doctor.go:56,67`) have no JSON tags; validate JSON output will need its own structs or doctor types will need tags added (doctor itself is out of scope per p-040, but validate consumes these types)

`assets update` (`internal/cli/assets_update.go`):
- No `--json` flag
- Output: `printUpdateResults` at `assets_update.go:154` - per-asset results with version changes

JSON output struct needs:
- `assets info --json` requires a struct combining `SearchResult` fields with installation status (installed bool, scope string). Options: add fields to `SearchResult`, create an `AssetInfoResult` wrapper, or inline a one-off struct in the marshal block.
- `assets validate --json` needs a top-level struct wrapping the three sections (index, modules, stats). The index section uses `doctor.SectionResult`/`doctor.CheckResult` which have no JSON tags and are out of scope (p-040). Validate-specific exported structs are the cleanest approach.
- `assets search --json` can marshal `[]SearchResult` directly once JSON tags are added.

Test coverage:
- `internal/cli/assets_test.go`: 12 test functions covering list, search, index, info
- `internal/cli/assets_validate_test.go`: 16 test functions covering validate helpers and output
- Established pattern: table-driven tests, buffer output, `strings.Contains` checks, `json.MarshalIndent` round-trip
- Key test: `TestPrintInstalledAssetsJSON` at `assets_test.go:770` validates JSON field names and omitempty

## Success Criteria

- [x] `SearchResult` has JSON tags (with decision on `MatchScore` visibility)
- [x] `assets list --json` includes tags, description, and category-specific fields
- [x] `assets search --json` outputs search results with category, name, description, tags, match score
- [x] `assets info --json` outputs single asset detail with all available fields
- [x] `assets validate --json` outputs validation results per module
- [x] `assets update --json` outputs update results with old/new version and success/error
- [x] Empty results output `[]` for arrays, `{}` for objects
- [x] All new JSON output uses `json.MarshalIndent("", "  ")` with camelCase field names
- [x] Tests cover JSON output for each modified command
- [x] All tests pass via `scripts/invoke-tests`

## Deliverables

- Updated `internal/assets/search.go` with JSON tags on `SearchResult`
- Updated `internal/cli/assets_list.go` with enriched JSON output
- Updated `internal/cli/assets_search.go` with `--json` flag
- Updated `internal/cli/assets_info.go` with `--json` flag
- Updated `internal/cli/assets_validate.go` with exported JSON structs and `--json` flag
- Updated `internal/cli/assets_update.go` with JSON tags and `--json` flag
- New or updated test files covering JSON output

## Technical Approach

Follow the established pattern from `assets list`:

```
jsonFlag, _ := cmd.Flags().GetBool("json")
// ... build data ...
if jsonFlag {
    data, err := json.MarshalIndent(result, "", "  ")
    // ...
    fmt.Fprintln(cmd.OutOrStdout(), string(data))
    return nil
}
```

For `UpdateResult.Error`, serialise as a string field:

- Add a custom `MarshalJSON` method or use a `jsonError` string field populated before marshalling

For `assets validate`, create exported structs (`ValidateModuleResult`, `ValidateCategoryResult`) mirroring the unexported types with JSON tags.

## Implementation Notes

The following edge cases need handling during implementation:

`assets update` empty/no-config paths: Lines 75-78 and 99-100 print text messages for empty state. With `--json`, these should output `[]` and return early (matching the `assets list` pattern).

`assets info --json` with all matches (Decision 2B): The interactive selection prompt at `assets_info.go:115` must be bypassed entirely. Output `results` as a JSON array. The `checkIfInstalled` call at line 133 is per-asset; for JSON array output, either skip install status or iterate all results.

`assets validate --json` struct mapping: The validate command's three output sections (index via `doctor.SectionResult`, modules via `validateCatResult`, stats) need a single top-level JSON struct. Since `doctor.CheckResult` and `doctor.SectionResult` have no JSON tags and are out of scope for this project, create validate-specific exported structs that mirror the data.

`assets search --json` prompt bypass: The interactive prompt fallback at `assets_search.go:63` is already gated by `isTerminal(stdin)`. With `--json`, the validation error should be returned directly regardless of terminal status.

## Decision Points (Resolved)

1. Include `MatchScore` in `SearchResult` JSON output? Decision: A — Include as `"matchScore"`
2. `assets info --json` with multiple matches? Decision: B — Output all matches as a JSON array
3. `UpdateResult.Error` serialisation approach? Decision: B — Parallel `ErrorMessage string` field with `json:"error,omitempty"`, `Error` tagged `json:"-"`

4. `assets info --json` installation status in array output? Decision: C — Load config once, build lookup set (reuse `collectInstalledNames` pattern from `assets_search.go:125`), check each result against it
