# P-007: Package Management

- Status: Proposed
- Started: -
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

- `start pkg search` - Search registry for packages
- `start pkg add` - Install package from registry
- `start pkg list` - List installed packages
- `start pkg info` - Show package details
- `start pkg update` - Update installed packages
- Integration with CUE Central Registry

Out of Scope:

- Package publishing (use cue publish directly)
- Package removal (manual config editing)
- Offline package installation
- Private registries

## Success Criteria

- [ ] `start pkg search <query>` finds packages in registry
- [ ] `start pkg add <package>` installs to config
- [ ] `start pkg list` shows installed packages
- [ ] `start pkg info <package>` shows package details
- [ ] `start pkg update` updates installed packages
- [ ] Works with packages published in P-002/P-003

## Workflow

### Phase 1: Research and Design

- [ ] Read all required documentation
- [ ] Review prototype asset commands for UX patterns
- [ ] Research CUE registry API for search/fetch
- [ ] Discuss CLI interface options
- [ ] Create DR for package management CLI

### Phase 2: Implementation

- [ ] Implement registry search
- [ ] Implement package installation
- [ ] Implement package listing
- [ ] Implement package info
- [ ] Implement package updates

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

- `internal/cli/pkg.go` - Package command group
- `internal/cli/pkg_search.go` - Search subcommand
- `internal/cli/pkg_add.go` - Add subcommand
- `internal/cli/pkg_list.go` - List subcommand
- `internal/cli/pkg_info.go` - Info subcommand
- `internal/cli/pkg_update.go` - Update subcommand
- `internal/registry/` - Registry interaction (may extend P-006 work)

Design Records:

- DR-0XX: Package Management CLI

## Technical Approach

To be determined after Phase 1 research. Key questions:

1. What CUE registry API is available for search?
2. How to present search results (table, list, interactive)?
3. How to handle version selection during install?
4. Where to write installed packages (global vs local)?
5. How to track installed packages for updates?

## Dependencies

Requires:

- P-006 (registry interaction foundation)

## Progress

(No progress yet - project not started)
