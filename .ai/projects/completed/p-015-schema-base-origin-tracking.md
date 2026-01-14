# P-015: Schema Base and Origin Tracking

- Status: Complete
- Started: 2026-01-05
- Completed: 2026-01-05

## Overview

Implement the `#Base` schema for common asset fields and add origin tracking to distinguish registry-installed assets from user-defined ones. This addresses schema duplication across asset types and enables the CLI to display asset provenance.

Discovered during P-012 testing (issue #13): `start config task` showed `(global)` but not whether assets came from registry packages.

## Goals

1. Create `#Base` schema with common fields (description, tags, origin)
2. Update all asset schemas (#Agent, #Role, #Task, #Context) to embed `#Base`
3. Publish updated schemas to CUE Central Registry
4. Update CLI to write `origin` field when installing from registry
5. Update CLI to display `(global, registry)` or `(local, registry)` for registry assets

## Scope

In Scope:

- Create `base.cue` schema file
- Modify agent.cue, role.cue, task.cue, context.cue to embed `#Base`
- Publish schema updates to registry
- Update `assets_add.go` to write `origin` field
- Update `config_*.go` files to read `origin` field
- Update display logic for asset listings

Out of Scope:

- Changes to UTD pattern
- Changes to settings schema
- Version tracking (origin stores module path only, not version)

## Success Criteria

- [x] `#Base` schema created in `context/start-assets/schemas/base.cue`
- [x] All four asset schemas embed `#Base`
- [x] Schemas validate correctly with `cue vet`
- [x] Updated schemas published to CUE Central Registry (v0.1.0)
- [x] `start assets add` writes `origin` field
- [x] `start config task/role/context/agent` shows `(registry)` indicator
- [x] Existing configs without `origin` field still work (backward compatible)

## Deliverables

- `context/start-assets/schemas/base.cue` - New schema file
- Updated schema files (agent.cue, role.cue, task.cue, context.cue)
- Updated CLI code (assets_add.go, config_task.go, config_role.go, config_context.go, config_agent.go)
- Published schema modules to CUE Central Registry

## Dependencies

- DR-037: Base Schema for Common Asset Fields (design decisions)

## Testing Strategy

1. Schema validation: `cue vet` on all schema files
2. Unit tests: Verify origin field is written and read correctly
3. Integration tests: End-to-end test of `assets add` followed by `config task`
4. Backward compatibility: Verify existing configs without `origin` still load

## Notes

Design decisions documented in DR-037:

- Field name: `origin` (not `source` to avoid conflict with UTD content sources)
- Value format: Module path without version (e.g., `start.cue.works/tasks/code-review`)
- Display format: `(global, registry)` or `(local, registry)`
- Schema name: `#Base` (neutral, allows functional fields in future)
