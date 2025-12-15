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

- [ ] Read all required documentation
- [ ] Research CUE Go API for module fetching
- [ ] Discuss findings and options
- [ ] Create DR for registry interaction (if trade-offs exist)

### Phase 2: Implementation

- [ ] Implement registry interaction
- [ ] Implement agent detection
- [ ] Implement auto-setup flow
- [ ] Integrate with existing CLI commands

### Phase 3: Validation

- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Write E2E tests
- [ ] Manual testing with real agents

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

Design Records (if needed):

- DR-0XX: Registry Interaction - CUE Go API for module fetching

## Technical Approach

To be determined after Phase 1 research. Key questions:

1. What CUE Go API to use for module fetching?
2. Where does the index live (module path, version)?
3. How to extract/write fetched config to disk?
4. Caching strategy for index and packages?
5. Error handling for network failures?

## Dependencies

Requires:

- P-001 (CUE schemas)
- P-002 (example assets for testing)
- P-003 (registry distribution)
- P-005 (CLI foundation, orchestration)

## Progress

### 2025-12-15

- Project created
- Phase 1 (Research and Design) starting
