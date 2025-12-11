# P-004: Minimal CLI Implementation

- Status: In Progress
- Started: 2025-12-10
- Completed: -

## Overview

Build minimal CLI infrastructure to validate the CUE-based architecture works end-to-end. This project implements just enough functionality to prove Go can load, validate, and work with CUE configurations.

The focus is on core infrastructure that enables future commands, not comprehensive CLI coverage.

## Required Reading

Before working on this project, read these design records:

| DR | Title | Why |
|----|-------|-----|
| DR-012 | CLI Global Flags | Defines all flags and applicability matrix |
| DR-013 | CLI Start Command | Root command execution flow |
| DR-016 | CLI Dry Run Flag | Output directory patterns, file structure |
| DR-017 | CLI Show Command | Show subcommands and output format |
| DR-018 | CLI Auto-Setup | First-run behaviour, no mandatory init |
| DR-024 | Testing Strategy | Test patterns, what to mock, file organisation |

Also review:

- `docs/cue/integration-notes.md` - CUE Go API patterns
- `Context/cobra/command.go` - Cobra command structure
- `Context/cobra/command_test.go` - Cobra testing patterns

## Goals

1. Implement CUE loading and validation infrastructure
2. Implement `start show` command to display and validate configuration
3. Implement global flags (DR-012)
4. Validate that CLI can load CUE configs and report errors helpfully
5. Test with real CUE assets from P-002
6. Create foundation for auto-setup (DR-018) and future commands

## Scope

In Scope:

- CUE loading from Go using official CUE API
- CUE validation with user-friendly error messages
- `start show` command with subcommands (role, context, agent, task)
- Global flags: `--verbose`, `--debug`, `--quiet`, `--help`, `--version`
- Show-specific flags: `--scope`
- Temp directory output pattern (per DR-016/DR-017)
- Basic error handling and user feedback
- Tests following DR-024 patterns

Out of Scope:

- Auto-setup flow (P-005 - needs registry integration)
- Agent execution (P-005)
- Task execution (P-005)
- `--dry-run` flag (needs agent execution context)
- Agent-specific flags: `--agent`, `--role`, `--model`, `--context`
- Package management commands
- Shell completion
- Interactive prompts

## Success Criteria

- [ ] Can load CUE configuration from `~/.config/start/` and `./.start/`
- [ ] Can merge global and local configurations
- [ ] Can validate CUE against schemas
- [ ] `start show role` displays resolved role content
- [ ] `start show context` displays resolved context content
- [ ] `start show agent` displays agent configuration
- [ ] `start show task` displays task configuration
- [ ] Validation errors are clear and actionable
- [ ] `--scope global` and `--scope local` work correctly
- [ ] Output follows temp directory pattern from DR-016
- [ ] Tests cover loader, validator, and show command
- [ ] Works with assets from P-002

## Deliverables

CLI Commands:

- `internal/cli/root.go` - Root command with global flags
- `internal/cli/show.go` - Show command with subcommands
- `internal/cli/flags.go` - Centralised flag definitions

CUE Infrastructure:

- `internal/cue/loader.go` - Load CUE from directories
- `internal/cue/validator.go` - Validate and report errors
- `internal/cue/errors.go` - User-friendly error formatting

Configuration:

- `internal/config/config.go` - Configuration structures
- `internal/config/paths.go` - Config directory resolution

Tests:

- `internal/cue/loader_test.go`
- `internal/cue/validator_test.go`
- `internal/cli/show_test.go`
- `test/testdata/` - CUE fixtures for testing

## Non-Deliverables

These are explicitly NOT part of this project:

- `start init` command - Auto-setup in P-005 replaces this
- Agent execution - P-005
- Registry fetching - P-005
- CLI documentation in `docs/cli/` - After commands stabilise

## Technical Approach

### Phase 1: CUE Infrastructure

1. Implement CUE loader
   - Load from directory using `load.Instances()`
   - Handle missing directories gracefully
   - Support both global and local config paths
   - Merge configurations using CUE unification

2. Implement CUE validator
   - Validate against schemas
   - Convert CUE errors to user-friendly messages
   - Include file path and line numbers
   - Suggest fixes where possible

3. Implement config path resolution
   - Global: `~/.config/start/`
   - Local: `./.start/`
   - Handle missing directories
   - Detect which configs exist

### Phase 2: Show Command

4. Implement show command structure
   - Parent `show` command
   - Subcommands: `role`, `context`, `agent`, `task`
   - `--scope` flag for global/local filtering

5. Implement show role
   - Load and validate configuration
   - Extract role by name (or default)
   - Process UTD (read files, but skip commands for now)
   - Write to temp directory
   - Display 5-line preview

6. Implement show context/agent/task
   - Same pattern as show role
   - Handle "show all" for contexts

7. Implement temp directory output
   - Create `/tmp/start-YYYYMMDDHHmmss/`
   - Handle timestamp collisions
   - Write markdown files

### Phase 3: Global Flags

8. Implement global flags
   - `--verbose` - Detailed output
   - `--debug` - Full debug output
   - `--quiet` - Suppress output
   - `--help` - Show help (Cobra default)
   - `--version` - Show version

9. Implement output modes
   - Normal: 5-line preview + file path
   - Verbose: Additional metadata
   - Debug: Full resolution trace
   - Quiet: File path only

### Phase 4: Testing

10. Write tests per DR-024
    - Unit tests for loader, validator
    - Integration tests for show command
    - Use real CUE files via `t.TempDir()`
    - Table-driven tests for error cases

11. Test with P-002 assets
    - Copy example configs to test directories
    - Verify validation passes
    - Verify output is correct

## Directory Structure

```
start/
├── cmd/start/
│   └── main.go
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   ├── root_test.go
│   │   ├── show.go
│   │   ├── show_test.go
│   │   └── flags.go
│   ├── cue/
│   │   ├── loader.go
│   │   ├── loader_test.go
│   │   ├── validator.go
│   │   ├── validator_test.go
│   │   └── errors.go
│   └── config/
│       ├── config.go
│       └── paths.go
├── test/
│   ├── testdata/
│   │   ├── valid/
│   │   └── invalid/
│   └── integration/
│       └── show_test.go
└── scripts/
    └── invoke-tests
```

## Dependencies

Requires:

- P-001 (CUE schemas)
- P-002 (example assets to test with)
- P-003 (understanding of module structure)

Blocks:

- P-005 (orchestration needs this foundation)

## Questions Resolved

These questions from the original P-004 are now answered by design records:

| Question | Answer | Source |
|----------|--------|--------|
| Should init be interactive? | No mandatory init; auto-setup on first run | DR-018 |
| What should show display? | 5-line preview + temp file | DR-017 |
| Where do configs live? | `~/.config/start/` and `./.start/` | DR-013 |
| How verbose should errors be? | User-friendly with file/line info | DR-017 |

## Notes

Why no `start init`:

DR-018 defines auto-setup behaviour where `start` (the root command) automatically detects agents and configures on first run. A separate `init` command is only needed for advanced use cases (local config, custom roles) which is P-005 scope.

Why show command first:

- Validates CUE loading works
- Validates error handling works
- Doesn't require agent execution
- Useful for debugging during development
- Foundation for `--dry-run` later

Relationship to DR-005:

The original P-004 mentioned creating "DR-005: CLI Command Structure" but DR-005 already exists (Go Templates for UTD Pattern). CLI structure is defined across DR-012 through DR-018.

## Progress Log

### 2025-12-10: Step 1 Complete - Go Project Initialised

Completed initial Go project setup.

Files created:

- `go.mod` - Module with Cobra v1.10.2, CUE v0.15.1
- `go.sum` - Generated dependency checksums
- `cmd/start/main.go` - Minimal entry point
- `internal/cli/root.go` - Root Cobra command

Verified:

- Build succeeds: `go build ./cmd/start/`
- CLI runs: `go run ./cmd/start/ --help`

### 2025-12-11: Step 2 Complete - Reference Study

Reviewed reference implementations:

- Cobra patterns from `Context/cobra/`
- CUE Go API from `docs/cue/integration-notes.md`
- Design records DR-012 through DR-018

Key findings:

- DR-018 changes init from prototype design to auto-setup
- DR-017 defines show command output format
- DR-012 defines all global flags
- Prototype CLI docs are reference only, not specification

Updated P-004 to align with design records.

Next: Phase 1 - CUE Infrastructure (loader, validator, paths)
