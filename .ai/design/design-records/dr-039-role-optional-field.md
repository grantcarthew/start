# dr-039: Role Optional Field

- Date: 2026-01-31
- Status: Accepted
- Category: Role

## Problem

File-based roles like dotai roles reference external files (e.g., `.ai/roles/default.md`, `~/.ai/roles/default.md`). When these files don't exist:

- Current behaviour: warn and continue without a role
- Problem: user configured roles expecting one to work, but gets no role applied
- Worse: if multiple roles are configured, the first is selected regardless of whether its file exists

Use case: dotai roles provide project-specific (cwd) and global (home) default roles. Users want automatic fallback - use project role if it exists, otherwise global role, otherwise something else. Currently this doesn't work because the first role is selected even if its file is missing.

## Decision

Add an `optional` field to the role schema:

```cue
#Role: {
    #Base
    #UTD
    optional: bool | *false
}
```

Behaviour:

| Role State | optional | Behaviour |
|------------|----------|-----------|
| File exists | any | Use this role |
| File missing | true | Skip, try next role in order |
| File missing | false | Error, stop execution |
| Explicit --role + missing | any | Error (user explicitly requested it) |
| settings.default_role + missing | any | Error (user explicitly configured it) |
| All roles missing | n/a | Error: no valid roles found |

## Why

Discovery-based roles need different semantics:

- Regular roles: "I am the golang expert role" - file must exist
- Dotai roles: "If this file exists, use it" - file may not exist

The `optional` field makes this distinction explicit:

- `optional: false` (default): role MUST be usable, error if not
- `optional: true`: role is opportunistic, skip if unavailable

Explicit selection always errors:

When user specifies `--role <name>`, they explicitly requested that role. If it's broken, that's an error regardless of the optional flag. Optional only affects automatic fallback behaviour.

All missing is an error:

If a user has configured roles, they expect at least one to work. Having all roles fail (whether optional or required) and running with no role would be surprising. This prevents silent failures.

## Trade-offs

Accept:

- Schema change requires publishing new version
- Users must update dotai roles to set optional: true
- Slightly more complex resolution logic

Gain:

- Clear semantics for discovery-based roles
- Graceful fallback through role chain
- Predictable behaviour: required roles must work
- No silent failures: all-missing is caught

## Structure

Schema addition to #Role:

```cue
#Role: {
    #Base
    #UTD
    optional: bool | *false
}
```

Field definition:

- `optional` (bool, default: false) - Whether this role can be skipped if its file is missing

## Resolution Logic

When explicit --role flag provided:

1. Resolve specified role
2. If resolution fails → error (regardless of optional)
3. User asked for it, must work

When settings.default_role is configured:

1. Use that specific role
2. If resolution fails → error (regardless of optional)
3. User configured it explicitly, must work

When no explicit role selection (automatic fallback):

1. Iterate through roles in definition order
2. Resolve role content (file, command, prompt)
3. If resolution fails:
   - optional: true → skip, try next role
   - optional: false → error, stop
4. If all roles exhausted → error: no valid roles found
5. First successful role wins

## UI Display

Role resolution status:

| Symbol | Meaning |
|--------|---------|
| ✓ | File found, role applied |
| ○ | File not found |

Context determines interpretation:

- Command succeeds with ○ roles shown: those were optional, skipped
- Command fails with ○ role: that was required, caused error

Example output (success):

```
Role:
  dotai-cwd-default   ○  skipped
  dotai-home-default  ○  skipped
  golang-agent        ✓  golang.md
```

Example output (error):

```
Role:
  dotai-cwd-default   ○  skipped
  dotai-home-default  ○  skipped
  golang-agent        ○  not found

Error: role "golang-agent" file not found: ~/.ai/roles/golang.md
```

Example output (all missing):

```
Role:
  dotai-cwd-default   ○  skipped
  dotai-home-default  ○  skipped

Error: no valid roles found (all optional roles skipped)
```

## Usage Examples

Dotai role (optional):

```cue
roles: "dotai-cwd-default": {
    description: "Project-specific default role"
    file: ".ai/roles/default.md"
    optional: true
    tags: ["dotai", "cwd", "project"]
}
```

Regular role (required, default):

```cue
roles: "golang-agent": {
    description: "Go language expert"
    file: "~/.ai/roles/golang.md"
    tags: ["golang", "programming"]
}
```

Typical user configuration order:

```cue
roles: {
    "dotai-cwd-default": { file: ".ai/roles/default.md", optional: true }
    "dotai-home-default": { file: "~/.ai/roles/default.md", optional: true }
    "golang-agent": { file: "~/.ai/roles/golang.md" }
}
```

This configuration means:

1. Try project-specific role (skip if missing)
2. Try global user role (skip if missing)
3. Use golang-agent (error if missing)

## Alternatives

No optional field, validate during fallback:

Always check file existence, skip missing roles automatically:

- Pro: No schema change
- Con: Can't distinguish "this should exist" from "this might exist"
- Con: Silent fallback to potentially inappropriate role (golang on JS project)
- Rejected: Users need control over which roles are strict

Fallback list in settings:

```cue
settings: {
    default_role: ["dotai-cwd", "dotai-home", "golang"]
}
```

- Pro: Explicit fallback chain
- Con: Schema change to settings
- Con: Separates fallback logic from role definition
- Con: Duplicates role ordering already in config
- Rejected: optional field is simpler, keeps logic with role

Role-level fallback field:

```cue
role: {
    file: ".ai/roles/default.md"
    fallback: "golang-agent"
}
```

- Pro: Explicit fallback relationship
- Con: Distributed logic across roles
- Con: Can create fallback cycles
- Rejected: Definition order already provides fallback chain

## Implementation Notes

Changes required:

1. Update #Role schema in start-assets/schemas/role.cue
2. Update role resolution in internal/orchestration/composer.go
3. Update getDefaultRole() to iterate and check file existence for optional roles
4. Update UI output in internal/cli/output.go to show resolution status
5. Update dotai roles in start-assets to set optional: true
6. Publish new schema version to registry

CLI support:

- `config role add --optional` flag to create optional roles
- `config role edit --optional` flag to set optional on existing roles
- RoleConfig struct preserves optional field on load/write
