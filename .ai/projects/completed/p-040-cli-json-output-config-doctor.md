# P-040: CLI JSON Output - Config, Search, and Doctor Commands

- Status: Complete
- Started: 2026-03-06
- Completed: 2026-03-06
- GitHub: #69

## Overview

Add rich `--json` output to config, search, and doctor commands. This is the second of two projects implementing issue #69, building on the patterns and conventions established in p-039.

## Goals

1. Add JSON tags to all config type structs and doctor report structs
2. Add `--json` to `config list`, `config info`, `config search`
3. Add `--json` to the top-level `search` command
4. Add `--json` to the `doctor` command
5. Add `MarshalJSON` for `doctor.Status` type to emit string representations
6. Add `--json` to `config settings` (list and single-key display)

## Scope

In Scope:

- JSON tags on `AgentConfig` (`internal/cli/config_types.go:16`)
- JSON tags on `RoleConfig` (`internal/cli/config_types.go:207`)
- JSON tags on `ContextConfig` (`internal/cli/config_types.go:330`)
- JSON tags on `TaskConfig` (`internal/cli/config_types.go:458`)
- JSON tags on `searchSection` (`internal/cli/search.go:21`)
- JSON tags on `doctor.Report` (`internal/doctor/doctor.go:76`)
- JSON tags on `doctor.SectionResult` (`internal/doctor/doctor.go:68`)
- JSON tags on `doctor.CheckResult` (`internal/doctor/doctor.go:57`)
- JSON tags on `config.SettingEntry` (`internal/config/settings.go:22`)
- `MarshalJSON` for `doctor.Status` to emit `"pass"`, `"warn"`, `"fail"`, `"info"`, `"notfound"`
- `json:"-"` on display-only fields (`CheckResult.Indent`, `CheckResult.NoIcon`, `SectionResult.NoIcons`)
- `--json` flag and marshal logic for `config list`, `config info`, `config search`, `search`, `doctor`, `config settings`
- Tests for each command's JSON output

Out of Scope:

- Assets commands - completed in p-039
- Changes to human-readable output
- New doctor checks or config features

## Current State

Config types (`internal/cli/config_types.go`):
- `AgentConfig` (line 16): no JSON tags; fields: Name, Bin, Command, DefaultModel, Description, Models (map[string]string), Tags, Source, Origin
- `RoleConfig` (line 207): no JSON tags; fields: Name, Description, File, Command, Prompt, Tags, Optional, Source, Origin
- `ContextConfig` (line 330): no JSON tags; fields: Name, Description, File, Command, Prompt, Required, Default, Tags, Source, Origin
- `TaskConfig` (line 458): no JSON tags; fields: Name, Description, File, Command, Prompt, Role, Tags, Source, Origin
- All 4 types have a `Source` field ("global" or "local") annotated as "for display only"; it carries scope data equivalent to `InstalledAsset.Scope` and should be included in JSON as `"source"`

Search section (`internal/cli/search.go:21`):
- `searchSection`: no JSON tags; fields: Label, Path, Results ([]assets.SearchResult), ShowInstalled
- `ShowInstalled` is a pure display flag (drives the ★ installed marker in registry results); exclude with `json:"-"`
- `Path` resolved: include in JSON as it provides useful source context
- `assets.SearchResult` already has JSON tags from p-039

Doctor types (`internal/doctor/doctor.go`):
- `CheckResult` (line 57): no JSON tags; display-only fields `Indent` and `NoIcon` get `json:"-"`; `Details []string` (verbose extra info) gets `json:"details,omitempty"`; `Fix string` gets `json:"fix,omitempty"` (many checks have no fix suggestion)
- `SectionResult` (line 68): no JSON tags; display-only field `NoIcons` gets `json:"-"`; `Summary string` gets `json:"summary,omitempty"` (optional, often empty)
- `Report` (line 76): no JSON tags; single field `Sections []SectionResult`
- `doctor.Status` is `type Status int` with `String()` returning `"pass"/"warn"/"fail"/"info"/"notfound"` but no `MarshalJSON`

Settings (`internal/config/settings.go` and `internal/cli/config_settings.go`):
- Schema defines 4 keys: `default_agent` (string), `shell` (string), `timeout` (int), `assets_index` (string)
- `config.SettingEntry` (line 22): no JSON tags; fields: `Value string`, `Source string` ("default", "global", "local", "not set")
- `config settings` (no args): calls `listSettings` which shows config paths then all 4 settings via `config.ResolveAllSettings`
- `config settings <key>`: calls `showSetting` which shows a single key's value and source
- Write operations (`set`, `unset`, `edit`): not candidates for `--json` — they are imperatives, not queries
- JSON list output: marshal as `map[string]SettingEntry` (object keyed by setting name) — natural for settings lookup; all 4 keys always present (source `"not set"` when unset)
- JSON single-key output: marshal as `SettingEntry` object `{"value": "...", "source": "..."}`; `value` should be `omitempty` so it is absent when source is `"not set"`
- Config paths (`printConfigPaths`) shown in human-readable list output: omit from JSON — path info is display-only context

Command implementations:
- None of the 5 target commands have a `--json` flag
- `--json` must be added as a per-command local flag (pattern: `cmd.Flags().Bool("json", false, "Output as JSON")`)
- The global `Flags` struct (`internal/cli/start.go:29`) does not include JSON; per-command flag access via `cmd.Flags().GetBool("json")` is the established pattern
- `config list` uses per-category helper functions `listAgents/Roles/Contexts/Tasks(w, stderr, local)`; the JSON path needs its own collection and marshal step separate from these helpers; category arg filtering still applies with `--json`
- `config list --json` emits a flat `[]ConfigListItem`; needs a top-level `Category string` field (e.g. `"agent"`, `"role"`, `"context"`, `"task"`) to distinguish types in the flat array; all other fields use `omitempty`
- `ConfigListItem.Models` must be `map[string]string` (alias→model-id) to match `AgentConfig.Models`, not `[]string` as used in `InstalledAsset.Models`
- `config info` no-args case (`config_info.go:41-46`): with `--json`, interactive mode must be suppressed; return error `"query required with --json"` (matching `assets info --json` pattern)
- `config info` ambiguity handling (`config_info.go:62`): errors with "ambiguous query" when stdin is not a terminal; with `--json`, return all matches as JSON array (Decision 2B)
- `config info` search uses `searchAllConfigCategories` (`config_helpers.go`) which returns `[]configMatch{Name, Category}`; loading full config data for each match requires a per-category load using `printConfigInfo`-style logic adapted to build `ConfigListItem` values
- `config info` / `config list` JSON output: `Prompt` field should be full content (not truncated to 100 chars as in the human-readable path); JSON consumers expect complete data
- `config info --json` always emits `[]ConfigListItem` regardless of match count (Decision 3A)
- `config info` JSON helper: a dedicated function (e.g. `buildConfigListItem(m configMatch, local bool) (ConfigListItem, error)`) loads the typed struct via `loadXForScope` functions and maps fields to `ConfigListItem`; shared by both `config info` and `config list` JSON paths
- `ConfigListItem` file placement: define in `config_list.go` (consistent with `InstalledAsset` in `assets_list.go`); reused by `config_info.go`
- `searchSection` JSON for `search --json` and `config search --json`: marshal as `[]searchSection`; `assets.SearchResult` already has JSON tags from p-039; empty scopes (no results) are already excluded before `sections` is built so no special empty handling is needed
- `search --json` and `config search --json` interactive prompt bypass: when `--json` is set and the query fails `ValidateSearchQuery`, return the error immediately instead of entering `promptSearchQuery`; check `jsonFlag` before the `isTerminal` branch
- `doctor` sets `cmd.SilenceErrors = true` and returns a sentinel error on issues found; `--json` should still emit the report and return the same exit code
- test placement: `doctor --json` tests go in `doctor_test.go` without `t.Parallel()` (existing tests use `os.Chdir`); config and search JSON tests can use `t.Parallel()` via env var isolation (`t.Setenv("HOME", ...)` + `t.Setenv("XDG_CONFIG_HOME", ...)` + `chdir(t, ...)` pattern)

## Success Criteria

- [x] All 4 config type structs have JSON tags with camelCase field names and `omitempty` on optional fields
- [x] `doctor.Status` marshals as string via `MarshalJSON`
- [x] Doctor report structs have JSON tags; display-only fields tagged `json:"-"`
- [x] `searchSection` has JSON tags
- [x] `config list --json` outputs all configured items with full config data per category
- [x] `config info --json` outputs single config item with all fields
- [x] `config search --json` and `search --json` output search results grouped by section
- [x] `doctor --json` outputs diagnostic report with sections, checks, status, label, message, fix
- [x] Empty results output `[]` for arrays, `{}` for objects
- [x] `config settings --json` outputs all settings as an object keyed by setting name
- [x] `config settings <key> --json` outputs the single setting as `{"value": "...", "source": "..."}`
- [x] Tests cover JSON output for each modified command
- [x] All tests pass via `scripts/invoke-tests`

## Post-review Fixes

- Removed `omitempty` from `SettingEntry.Value` (`internal/config/settings.go:23`) so `value` field is always present in JSON output, including "not set" entries
- Changed `assets_info.go` empty-result JSON from `fmt.Fprintln(w, "[]")` to `writeJSON(w, []AssetInfoResult{})` for consistency with all other commands

## Deliverables

- Updated `internal/cli/config_types.go` with JSON tags on all 4 config structs
- Updated `internal/cli/search.go` with JSON tags on `searchSection`
- Updated `internal/doctor/doctor.go` with JSON tags and `Status.MarshalJSON`
- Updated `internal/cli/config_list.go` with `--json` flag
- Updated `internal/cli/config_info.go` with `--json` flag
- Updated `internal/cli/config_search.go` with `--json` flag
- Updated `internal/cli/search.go` with `--json` flag
- Updated `internal/cli/doctor.go` with `--json` flag
- Updated `internal/config/settings.go` with JSON tags on `SettingEntry`
- Updated `internal/cli/config_settings.go` with `--json` flag
- New or updated test files covering JSON output

## Technical Approach

Same pattern as p-039 and the existing `assets list` implementation.

For `doctor.Status`, add a `MarshalJSON` method that delegates to `String()`:

```go
func (s Status) MarshalJSON() ([]byte, error) {
    return json.Marshal(s.String())
}
```

For `searchSection.Path`, include it in JSON as it provides useful source context (local/global/registry path).

For `config list --json`, output a flat array of config items (matching the `assets list` pattern) rather than nested category maps.

## Decision Points

1. For `config list --json`, the project specifies a flat array matching the `assets list` pattern. The 4 config types have different fields, so a unified type is needed. Decision: A — Create a new `ConfigListItem` struct with all possible fields and `omitempty`; marshal as `[]ConfigListItem`.

2. For `config info --json` when a query matches multiple items (ambiguous). Decision: B — Return all matches as a JSON array, consistent with `assets info --json`.

3. For `config info --json` output type (single and multiple match). Decision: A — Always emit `[]ConfigListItem` regardless of match count, consistent with `assets info --json` always returning `[]AssetInfoResult`.

## Dependencies

- p-039: Establishes JSON tagging conventions and pattern; `SearchResult` tags needed for config search
