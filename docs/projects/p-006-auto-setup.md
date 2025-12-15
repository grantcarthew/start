# P-006: Auto-Setup

- Status: In Progress
- Started: 2025-12-15
- Completed: -

## Overview

Implement auto-setup for first-run users. When no configuration exists, `start` automatically detects installed AI CLI tools, prompts the user to select one if multiple are found, fetches the agent configuration from the registry, and writes it to global config.

This completes the orchestration system started in P-005, enabling the zero-config to agent launch workflow.

## Required Reading

| Document | Why |
|----------|-----|
| DR-018 | Auto-setup flow, terminal output, exit codes |
| DR-019 | Index bin field for agent detection |
| dr-writing-guide.md | DR creation guidelines |
| docs/cue/integration-notes.md | CUE Go API patterns |

## Goals

1. Implement registry interaction (fetch index, fetch agent packages)
2. Implement agent detection (PATH checking via exec.LookPath)
3. Implement auto-setup flow (TTY prompts, config writing)
4. E2E validation with real P-002 assets

## Scope

In Scope:

- Fetch index from CUE registry
- Check each agent's `bin` field against PATH
- Single agent: auto-select
- Multiple agents: prompt user (TTY) or error (non-TTY)
- Fetch selected agent package from registry
- Write config to `~/.config/start/agents.cue`
- E2E tests for full workflow

Out of Scope:

- Package management commands
- Offline bootstrap / bundled index
- Windows support

## Success Criteria

- [ ] First run with no config triggers auto-setup
- [ ] Auto-setup detects agents via PATH checking
- [ ] Single agent detected: auto-selects without prompt
- [ ] Multiple agents detected: prompts user to select
- [ ] Non-TTY with multiple agents: exits with error
- [ ] No agents detected: helpful error message
- [ ] Selected agent config written to ~/.config/start/
- [ ] E2E demo: zero config to agent launch works

## Workflow

This project follows the standard development workflow:

### Phase 1: Research and Design

- [x] Read all required documentation
- [x] Research CUE Go API for module fetching
- [x] Discuss findings and options
- [x] Create DR for registry interaction (if trade-offs exist)

### Phase 2: Implementation

- [x] Implement registry interaction
- [x] Implement agent detection
- [x] Implement auto-setup flow
- [x] Integrate with existing CLI commands

### Phase 3: Validation

- [x] Write unit tests
- [x] Write integration tests
- [ ] Write E2E tests (blocked until index published)
- [x] Create index module for publishing

### Phase 4: Review

- [ ] External code review (if significant changes)
- [ ] Fix reported issues
- [ ] Update project document

## Deliverables

Files:

- `internal/registry/index.go` - Index fetching
- `internal/registry/package.go` - Package fetching
- `internal/detection/agent.go` - Agent binary detection
- `internal/orchestration/autosetup.go` - Auto-setup flow
- `test/e2e/autosetup_test.go` - E2E tests

Design Records:

- DR-027: Registry Module Fetching - CUE Go API, index location, config writing, error handling

## Technical Approach

Determined in Phase 1 research (see DR-027):

1. CUE Go API: `modconfig.NewRegistry()` - handles auth, caching, registry resolution
2. Index location: `github.com/grantcarthew/start-assets/index@v0`
3. Config writing: Decode CUE to Go struct (`orchestration.Agent`), encode back to CUE
4. Caching: Use CUE's native cache (`~/.cache/cue/`)
5. Error handling: Retry with exponential backoff (2-3 attempts), then exit code 2

## Dependencies

Requires:

- P-001 (CUE schemas)
- P-002 (example assets for testing)
- P-003 (registry distribution)
- P-005 (CLI foundation, orchestration)

## Progress

### 2025-12-15

- Project created
- Phase 1 (Research and Design) completed
- Read required documentation (DR-018, DR-019, dr-writing-guide.md, integration-notes.md)
- Researched CUE Go API: modconfig, modregistry, modcache packages
- Key decisions made:
  - Index location: `github.com/grantcarthew/start-assets/index@v0`
  - API: `modconfig.NewRegistry()` for auth and caching
  - Config writing: decode to Go struct, encode to CUE
  - Error handling: retry with exponential backoff
- Created DR-027: Registry Module Fetching
- Phase 2 (Implementation) completed:
  - `internal/registry/client.go` - Registry client with retry logic
  - `internal/registry/index.go` - Index fetching and parsing
  - `internal/detection/agent.go` - PATH-based agent detection
  - `internal/orchestration/autosetup.go` - Auto-setup flow
  - `internal/cli/start.go` - CLI integration
- All tests passing
- Phase 3 (Validation) mostly complete:
  - Unit tests: `internal/detection/agent_test.go`, `internal/registry/index_test.go`, `internal/orchestration/autosetup_test.go`
  - Integration tests: `test/integration/autosetup_test.go`
  - Created index module: `context/start-assets/index/`
  - Index module published to registry

### Session 2 (2025-12-15 continued)

Fixed issues with index loading:
- Changed `IndexModulePath` from `@v0` to `@v0.0.1` (canonical version required by `module.ParseVersion`)
- Changed `load.Config.Package` from `"*"` to `"index"` (wildcard creates empty synthetic package)
- Added `modconfig.Registry` parameter to `LoadIndex` for dependency resolution
- Updated tests to include `package index` declaration

Manual testing progress:
- Index fetching: Working
- Agent detection: Working (detects aichat, claude, gemini)
- TTY prompt: Working (displays selection menu)
- Version resolution: Added `ResolveLatestVersion` to convert `@v0` to canonical version

**Current Bug**: `ResolveLatestVersion` fails with "no versions found"
```
Error: resolving agent version: no versions found for github.com/grantcarthew/start-assets/agents/claude@v0
```
The `ModuleVersions` API call returns empty. Investigation needed:
- Check if module path format is correct for the versions API
- May need to strip the version suffix before querying
- Test with `cue mod` CLI to compare behavior
