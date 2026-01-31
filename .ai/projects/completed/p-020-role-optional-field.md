# p-020: Role Optional Field

- Status: Complete
- Started: 2026-01-31
- Completed: 2026-01-31

## Overview

Add an `optional` field to the role schema enabling discovery-based roles that gracefully skip when their files don't exist. This supports dotai roles (project-specific and global default roles) that should fall through to the next role in the chain rather than causing warnings or running without a role.

Currently, if a file-based role's file is missing, the system warns and continues without a role. This is problematic when users configure multiple roles expecting automatic fallback behaviour.

## Goals

1. Add `optional` field to #Role schema (default: false)
2. Update role resolution to skip optional roles with missing files
3. Update role resolution to error on required roles with missing files
4. Error when all configured roles fail (no silent "no role" state)
5. Always error on explicit `--role` with missing file regardless of optional flag
6. Update UI output to show role resolution status
7. Update dotai roles in start-assets to use optional: true

## Scope

In Scope:

- Schema change to #Role
- Role resolution logic in composer
- Default role selection in getDefaultRole()
- UI display of role resolution status
- Dotai role updates in start-assets
- Schema version publish to registry

Out of Scope:

- Similar changes to contexts (may be future work)
- Similar changes to agents
- Changes to task role handling (uses same resolution)

## Success Criteria

- [x] #Role schema includes `optional: bool | *false`
- [x] Optional role with missing file skips to next role
- [x] Required role with missing file errors with clear message
- [x] All roles missing errors with "no valid roles found"
- [x] Explicit --role with missing file always errors
- [x] UI shows ✓ for loaded roles, ○ for missing
- [x] Dotai roles updated with optional: true
- [x] New schema version published to registry (schemas/v0.2.0)
- [x] Existing tests pass
- [x] New tests cover optional role behaviour

## Deliverables

- Updated `start-assets/schemas/role.cue` with optional field
- Updated `internal/orchestration/composer.go` role resolution
- Updated `internal/cli/output.go` role status display
- Updated `start-assets/roles/dotai/*/role.cue` with optional: true
- New schema version published to CUE Central Registry
- Test coverage for optional role scenarios

## Current State

Role schema (start-assets/schemas/role.cue:11-17):

```cue
#Role: {
    #Base
    #UTD
}
```

Role resolution (internal/orchestration/composer.go):

- `getDefaultRole()` (line 464-482): Returns first role in definition order or settings.default_role. No file existence check.
- `ComposeWithRole()` (line 211-249): Resolves role and adds warning if file missing (per current DR-007).
- `resolveRole()` (line 396-461): Returns error if file missing, but caller treats as warning.

ComposeResult struct (internal/orchestration/composer.go:98-112):

```go
type ComposeResult struct {
    Prompt   string
    Contexts []Context
    Role     string
    RoleFile string
    RoleName string
    Warnings []string
}
```

Needs new field for role resolution chain to support UI display.

UI output (internal/cli/output.go):

- `PrintRoleTable()` (line 144-186): Shows ✓ for loaded, ○ for failed/missing.
- Currently shows single role, not resolution chain.

Call sites requiring update:

- `internal/cli/start.go:271,291` - calls PrintRoleTable
- `internal/cli/task.go:353,385` - calls PrintRoleTable

Dotai roles (start-assets/roles/dotai/):

- `cwd/default/role.cue` - file: `.ai/roles/default.md`
- `home/default/role.cue` - file: `~/.ai/roles/default.md`
- Neither has `optional` field yet.

Existing tests (internal/orchestration/composer_test.go):

- `TestComposer_ComposeWithRole` (line 271-344): Tests role resolution including "nonexistent role adds warning" case that will change to error.
- `TestGetDefaultRole` (line 531-601): Tests default role selection.

## Technical Approach

1. Schema: Add `optional: bool | *false` to #Role in start-assets/schemas/role.cue

2. Resolution: Update role selection logic:
   - If `settings.default_role` is set: use that role, error on failure (no fallback)
   - If no explicit default: iterate through roles in definition order
   - For each role, extract `optional` field from CUE value
   - For file-based roles, validate file exists (use ExpandFilePath for tilde)
   - If resolution fails and optional: true, continue to next role
   - If resolution fails and optional: false, return error immediately
   - Return first valid role name
   - If all roles exhausted, return error (not empty string)

3. ComposeResult: Add role resolution tracking:
   - Add `RoleResolutions []RoleResolution` field
   - `RoleResolution` struct: Name, Status ("loaded"/"skipped"/"error"), File, Optional

4. Composition: Update `ComposeWithRole()` to:
   - If explicit roleName provided (--role flag), always error on failure
   - If no explicit roleName, call updated getDefaultRole() which handles chain
   - Populate RoleResolutions from resolution process
   - Error (not warn) on required role failure
   - Error if all roles exhausted with "no valid roles found"

5. UI: Update `PrintRoleTable()` to:
   - Accept `[]RoleResolution` instead of single role params
   - Display each checked role with ✓ (loaded) or ○ (skipped/error)
   - Update call sites in start.go and task.go

6. Dotai roles: Update both dotai role definitions to include `optional: true`

7. Publish: Bump schema version and publish to registry

## Design Decisions

- dr-039: Role Optional Field (documents the optional field design)
- dr-010: Role Schema Design (updated with reference to dr-039)
- dr-007: UTD Error Handling (updated - required roles now error, not warn)

## Testing Strategy

Unit tests:

- Optional role skipped when file missing
- Required role errors when file missing
- First valid role selected from chain
- All optional missing returns error
- Explicit --role errors regardless of optional
- Mixed optional/required chain resolution
- settings.default_role errors on failure (no fallback even if optional)

Integration tests:

- End-to-end with dotai roles configured
- Fallback chain with real files

## Decision Points

1. When `settings.default_role` specifies an optional role that fails to resolve:

   Decision: A - Error immediately. The `default_role` setting is an explicit choice by the user and must work regardless of the `optional` flag. The `optional` flag only affects automatic fallback behaviour when no explicit default is configured.

## Dependencies

- None (builds on existing role infrastructure)
