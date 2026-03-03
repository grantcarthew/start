# P-040: CLI JSON Output - Config, Search, and Doctor Commands

- Status: Pending
- Started:
- Completed:
- GitHub: #69

## Overview

Add rich `--json` output to config, search, and doctor commands. This is the second of two projects implementing issue #69, building on the patterns and conventions established in p-039.

## Goals

1. Add JSON tags to all config type structs and doctor report structs
2. Add `--json` to `config list`, `config info`, `config search`
3. Add `--json` to the top-level `search` command
4. Add `--json` to the `doctor` command
5. Add `MarshalJSON` for `doctor.Status` type to emit string representations

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
- `MarshalJSON` for `doctor.Status` to emit `"pass"`, `"warn"`, `"fail"`, `"info"`, `"notfound"`
- `json:"-"` on display-only fields (`CheckResult.Indent`, `CheckResult.NoIcon`, `SectionResult.NoIcons`)
- `--json` flag and marshal logic for `config list`, `config info`, `config search`, `search`, `doctor`
- Tests for each command's JSON output

Out of Scope:

- Assets commands - completed in p-039
- Changes to human-readable output
- New doctor checks or config features

## Current State

- Config types in `internal/cli/config_types.go` have no JSON tags across 4 structs (~500 lines)
- `searchSection` in `internal/cli/search.go:21` has no JSON tags
- Doctor types in `internal/doctor/doctor.go` have no JSON tags
- `doctor.Status` is `type Status int` with `String()` method returning `"pass"`, `"warn"`, etc. but no `MarshalJSON`
- Display-only fields exist: `CheckResult.Indent`, `CheckResult.NoIcon`, `SectionResult.NoIcons`
- `searchSection.Path` is a display concern; decide whether to include in JSON
- The `search` command and `config search` share the `searchSection` type

## Success Criteria

- [ ] All 4 config type structs have JSON tags with camelCase field names and `omitempty` on optional fields
- [ ] `doctor.Status` marshals as string via `MarshalJSON`
- [ ] Doctor report structs have JSON tags; display-only fields tagged `json:"-"`
- [ ] `searchSection` has JSON tags
- [ ] `config list --json` outputs all configured items with full config data per category
- [ ] `config info --json` outputs single config item with all fields
- [ ] `config search --json` and `search --json` output search results grouped by section
- [ ] `doctor --json` outputs diagnostic report with sections, checks, status, label, message, fix
- [ ] Empty results output `[]` for arrays, `{}` for objects
- [ ] Tests cover JSON output for each modified command
- [ ] All tests pass via `scripts/invoke-tests`

## Deliverables

- Updated `internal/cli/config_types.go` with JSON tags on all 4 config structs
- Updated `internal/cli/search.go` with JSON tags on `searchSection`
- Updated `internal/doctor/doctor.go` with JSON tags and `Status.MarshalJSON`
- Updated `internal/cli/config_list.go` with `--json` flag
- Updated `internal/cli/config_info.go` with `--json` flag
- Updated `internal/cli/config_search.go` with `--json` flag
- Updated `internal/cli/search.go` with `--json` flag
- Updated `internal/cli/doctor.go` with `--json` flag
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

## Dependencies

- p-039: Establishes JSON tagging conventions and pattern; `SearchResult` tags needed for config search
