# P-007: Package Management

- Status: In Progress
- Started: 2025-12-17
- Completed: -

## Overview

Implement package management commands for discovering, adding, and updating assets from the CUE Central Registry. This enables users to find and install roles, tasks, contexts, and agents published by the community.

## Required Reading

| Document | Why |
|----------|-----|
| DR-003 | Index category structure |
| DR-004 | Module naming convention |
| DR-021 | Module and package naming conventions |
| DR-023 | Module path prefix for file resolution |
| Prototype start-assets*.md | Prior art for CLI interface |
| docs/cue/integration-notes.md | CUE Go API patterns |

## Goals

1. Design CLI interface for package management
2. Implement registry search and discovery
3. Implement package installation
4. Implement package updates
5. Implement package listing and info

## Scope

In Scope:

- `start assets browse` - Open GitHub asset registry in browser
- `start assets search` - Search registry index for packages
- `start assets add` - Install package from registry
- `start assets list` - List installed packages with update status
- `start assets info` - Show package details
- `start assets update` - Update installed packages
- `start assets index` - Regenerate index.cue (asset repo maintainers)
- Integration with CUE Central Registry

Out of Scope:

- Package publishing (use cue publish directly)
- Package removal (manual config editing)
- Offline package installation
- Private registries

## Success Criteria

- [ ] `start assets browse` opens GitHub asset registry in browser
- [ ] `start assets search <query>` finds packages in registry
- [ ] `start assets add <package>` installs to config
- [ ] `start assets list` shows installed packages with update status
- [ ] `start assets info <package>` shows package details
- [ ] `start assets update` updates installed packages
- [ ] `start assets index` regenerates index in asset repos
- [ ] Works with packages published in P-002/P-003

## Workflow

### Phase 1: Research and Design

- [x] Read all required documentation
- [x] Review prototype asset commands for UX patterns
- [x] Research CUE registry API for search/fetch
- [x] Discuss CLI interface options
- [x] Create DR for package management CLI (DR-028)

### Phase 2: Implementation

- [ ] Implement assets command group
- [ ] Implement browse subcommand
- [ ] Implement search subcommand
- [ ] Implement add subcommand
- [ ] Implement list subcommand
- [ ] Implement info subcommand
- [ ] Implement update subcommand
- [ ] Implement index subcommand

### Phase 3: Validation

- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Manual testing with real registry

### Phase 4: Review

- [ ] External code review (if significant changes)
- [ ] Fix reported issues
- [ ] Update project document

## Deliverables

Files:

- `internal/cli/assets.go` - Assets command group
- `internal/cli/assets_browse.go` - Browse subcommand
- `internal/cli/assets_search.go` - Search subcommand
- `internal/cli/assets_add.go` - Add subcommand
- `internal/cli/assets_list.go` - List subcommand
- `internal/cli/assets_info.go` - Info subcommand
- `internal/cli/assets_update.go` - Update subcommand
- `internal/cli/assets_index.go` - Index subcommand
- `internal/registry/` - Registry interaction (extends P-006 work)

Design Records:

- DR-028: CLI Assets Command

## Technical Approach

Decisions from Phase 1 research (see DR-028):

1. **Search**: Fetch index from CUE registry (`github.com/grantcarthew/start-assets/index@v0`), search locally with substring matching
2. **Results**: Grouped by type (agents, roles, tasks, contexts), alphabetical within type
3. **Versions**: Use `@v0` major version, CUE resolves to latest compatible automatically
4. **Scope**: Global (`~/.config/start/`) by default, `--local` flag for project (`./.start/`)
5. **Updates**: Re-fetch modules to get latest within major version
6. **Asset repo detection**: Check for `agents/`, `roles/`, `tasks/`, `contexts/` directories

Reuse existing infrastructure:

- `internal/registry.Client` for module fetching (from P-006)
- `internal/registry.FetchIndex()` for index retrieval
- `IndexEntry` struct with module, description, tags, bin

## Dependencies

Requires:

- P-006 (registry interaction foundation)

## Progress

### 2025-12-17

- Completed Phase 1: Research and Design
- Reviewed prototype asset commands for UX patterns
- Discussed CLI interface options and made key decisions
- Created DR-028: CLI Assets Command
- Updated project document with decisions and file structure
