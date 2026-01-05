# DR-037: Base Schema for Common Asset Fields

- Date: 2026-01-05
- Status: Accepted
- Category: CUE

## Problem

All four asset types (agents, roles, tasks, contexts) share common fields that are duplicated across their schemas:

- `description?: string`
- `tags?: [...string]`

Additionally, when assets are installed from the CUE registry, there is no way to distinguish them from user-defined assets in listings. The `start config task` output shows `(global)` or `(local)` but not whether the asset originated from a registry package.

This creates two issues:

1. Schema duplication - common fields defined in multiple places
2. No origin tracking - users cannot tell which assets came from registry

## Decision

Create a `#Base` schema that defines common fields for all asset types, including a new `origin` field for tracking asset provenance.

All asset schemas embed `#Base`.

## Why

- DRY principle - common fields defined once, reducing maintenance burden
- Extensibility - adding new common fields requires only one schema change
- Origin tracking - `origin` field enables distinguishing registry vs user-defined assets
- Neutral naming - `#Base` does not imply only metadata; functional fields can be added later

The `origin` field is optional for backward compatibility. When present, it contains the module path (e.g., `start.cue.works/tasks/code-review`). When empty or undefined, the asset is user-defined.

## Trade-offs

Accept:

- Schema dependency - all asset schemas now depend on `#Base`
- Breaking change for schema consumers - existing code importing schemas needs update
- Registry republish required - new schema versions must be published

Gain:

- Single source of truth for common fields
- Consistent tag validation across all asset types
- Clear provenance tracking for installed assets
- Easier future extensions (add once, applies everywhere)

## Alternatives

Hidden field approach:

- Use `_package: string` as an internal field
- Pro: No schema change needed
- Con: Hidden fields are not validated, could be inconsistent
- Rejected: Schema change is cleaner and enables validation

Comment-based tracking:

- Current approach: `// Source: module-path` comment when installing
- Pro: Already implemented
- Con: Comments are fragile, can be lost on edit
- Rejected: Not machine-readable

Separate metadata struct per type:

- Define `#TaskMetadata`, `#RoleMetadata`, etc.
- Pro: Type-specific flexibility
- Con: Still duplicates common fields
- Rejected: Defeats purpose of reducing duplication

## Structure

File: `context/start-assets/schemas/base.cue`

```cue
package schemas

// #Base defines common fields for all asset types.
// Embedded by #Agent, #Role, #Task, and #Context.
#Base: {
    // Human-readable description of the asset
    description?: string

    // Tags for categorization and filtering
    // Must be lowercase kebab-case
    tags?: [...string & =~"^[a-z0-9]+(-[a-z0-9]+)*$"]

    // Module path when installed from registry
    // Example: "start.cue.works/tasks/code-review"
    // Empty/undefined = user-defined asset
    origin?: string
}
```

Updated asset schemas:

```cue
#Task: {
    #Base
    #UTD
    role?:  string | #Role
    agent?: string
}

#Role: {
    #Base
    #UTD
}

#Agent: {
    #Base
    command: string & !=""
    bin?: string & !=""
    default_model?: string
    models?: [string]: string & !=""
}

#Context: {
    #Base
    #UTD
    required?: bool
    default?:  bool
}
```

## Implementation Notes

CLI changes required:

1. `assets_add.go` - Write `origin` field when installing from registry (module path without version)
2. `config_*.go` - Read `origin` field when loading assets
3. Display logic - Show `(global, registry)` or `(local, registry)` when origin is set

Display format:

- User-defined global: `task-name (global)`
- Registry global: `task-name (global, registry)`
- User-defined local: `task-name (local)`
- Registry local: `task-name (local, registry)`
