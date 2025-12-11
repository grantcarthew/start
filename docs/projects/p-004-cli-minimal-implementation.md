# P-004: Minimal CLI Implementation

- Status: In Progress
- Started: 2025-12-10
- Completed: -

## Overview

Build minimal CLI commands to interact with CUE configurations. This project implements just enough CLI functionality to validate the architecture works end-to-end, adapting the well-designed CLI interface from the prototype but making it CUE-native.

The focus is on core commands that prove the system works, not comprehensive coverage. Complexity comes later after validation.

## Goals

1. Implement start init command to generate initial CUE configuration
2. Implement start show command to display and validate configuration
3. Implement basic configuration loading and validation from Go
4. Validate that CLI can load CUE configs and report errors
5. Test with real CUE assets from P-002
6. Document CLI architecture and command structure
7. Create foundation for future CLI expansion

## Scope

In Scope:

- Initialize project with CUE configuration (start init)
- Display and validate configuration (start show)
- Load CUE from Go using official CUE API
- Validate CUE and report helpful errors
- Basic flag handling (--role, --agent, etc.)
- Use Cobra framework (reference/cobra available)
- Create DR-005: CLI Command Structure
- Basic error handling and user feedback

Out of Scope:

- Task execution (P-005)
- Agent orchestration (P-005)
- Package management commands (future)
- Shell completion (future)
- Interactive browsing (future)
- Doctor/health check commands (future)
- All commands except init and show

## Success Criteria

- [ ] start init creates valid CUE configuration files
- [ ] start show displays configuration correctly
- [ ] start show reports validation errors helpfully
- [ ] Can load and validate CUE from Go using official API
- [ ] Works with assets from P-002
- [ ] Created DR-005: CLI Command Structure
- [ ] CLI follows patterns from reference/cobra
- [ ] Error messages are clear and actionable
- [ ] Basic help text and usage documentation

## Deliverables

CLI Commands:

- cmd/start/init.go - Initialize command
- cmd/start/show.go - Show/validate command
- cmd/start/root.go - Root command setup

Go Implementation:

- internal/cue/loader.go - CUE loading from Go
- internal/cue/validator.go - CUE validation
- internal/config/config.go - Configuration structure

Design Records:

- DR-005: CLI Command Structure and Organization

Documentation:

- docs/cli/start.md - Root command documentation
- docs/cli/start-init.md - Init command documentation
- docs/cli/start-show.md - Show command documentation
- docs/cue/go-integration.md - CUE-Go integration patterns

Tests:

- Test init creates valid files
- Test show validates correctly
- Test error handling

## Dependencies

Requires:

- P-001 (need architecture and schemas)
- P-002 (need assets to test with)
- P-003 (need to understand package loading)

Blocks:

- P-005 (orchestration needs CLI foundation)

## Technical Approach

Setup Phase:

1. Initialize Go project
   - Create go.mod with CUE dependencies
   - Set up Cobra CLI framework
   - Create basic project structure
   - Set up testing framework

2. Study reference implementations
   - Review reference/cobra patterns
   - Review prototype CLI docs (reference/start-prototype/docs/cli/)
   - Study CUE Go API (reference/cue/doc/)

Implementation Phase:

1. Implement start init
   - Generate cue.mod/module.cue
   - Create starter configuration files
   - Set up directory structure
   - Provide helpful output and next steps

2. Implement CUE loading
   - Use CUE Go API to load configurations
   - Handle file not found gracefully
   - Handle invalid CUE with clear errors
   - Extract configuration values to Go structs

3. Implement start show
   - Load CUE configuration
   - Validate against schemas
   - Display formatted output
   - Report validation errors with context

4. Implement error handling
   - CUE validation errors to user-friendly messages
   - File system errors with helpful suggestions
   - Configuration errors with fix guidance

Testing Phase:

1. Test with P-002 assets
   - Use example roles, tasks, contexts, agents
   - Verify validation works
   - Test error cases
   - Verify output is useful

2. Test edge cases
   - Missing files
   - Invalid CUE syntax
   - Schema violations
   - Empty configurations

Documentation Phase:

1. Write DR-005
   - CLI command structure
   - Why these commands first
   - How it relates to prototype CLI design
   - Future expansion strategy

2. Write CLI documentation
    - Command reference docs
    - Usage examples
    - Common workflows
    - Troubleshooting

3. Document Go-CUE integration
    - Loading patterns
    - Validation patterns
    - Error handling patterns
    - Best practices discovered

## Questions & Uncertainties

CUE Go API:

- What's the best way to load CUE files from Go?
- How do we handle CUE validation errors?
- Can we get structured error information?
- What's the performance of loading and validation?
- How do we extract values to Go structs?

CLI Design:

- Should init be interactive or flag-driven?
- What should show display (raw CUE, formatted, JSON)?
- How verbose should validation errors be?
- Should show validate or just display?

Configuration Structure:

- Single file or multiple files for init?
- What directory structure should init create?
- Where do CUE configs live (root, .start/, etc.)?
- How do we handle global vs project configs?

Error Handling:

- How detailed should error messages be?
- Should we show CUE errors raw or translate?
- How do we guide users to fix issues?
- What's the balance between helpful and overwhelming?

Prototype Adaptation:

- Which CLI patterns from prototype still apply?
- What changes because of CUE?
- Which commands are most valuable first?
- How do we evolve from minimal to complete?

## Research Areas

High Priority:

1. CUE Go API
   - Loading CUE files
   - Validation from Go
   - Error handling
   - Value extraction

2. Cobra patterns
   - Command structure
   - Flag handling
   - Subcommands
   - Help text

3. Configuration loading
   - File discovery
   - Multiple file handling
   - Merging configurations
   - Default values

Medium Priority:

1. Error message design
   - User-friendly validation errors
   - Helpful suggestions
   - Context and location
   - Fix guidance

2. Output formatting
   - CUE display formats
   - JSON output option
   - Human-readable formatting
   - Validation results

Low Priority:

1. Future expansion
   - Additional commands
   - Flag standardization
   - Configuration options
   - Plugin architecture

## Notes

Prototype CLI Reference:

- reference/start-prototype/docs/cli/ - Well-designed CLI interface
- Adapt concepts, not implementation
- Many commands deferred to future projects

Key Difference from Prototype:
The prototype built comprehensive CLI upfront. This project builds minimal viable CLI to validate architecture, then expands based on learnings.

Command Priority Rationale:

- init: Users need to start somewhere
- show: Users need to validate configs work
- Everything else: Deferred until core orchestration works (P-005)

This is not the final CLI - it's the minimal CLI needed to prove the architecture. More commands come after P-005 validates end-to-end orchestration.

Go Project Structure:

- cmd/start/ - CLI entry point (main.go only)
- internal/cli/ - Cobra commands (root.go, init.go, show.go)
- internal/cue/ - CUE loading and validation
- internal/config/ - Configuration structures

This project is complete when we can initialize a project and validate its CUE configuration, proving the Go-CUE integration works.

## Progress Log

### 2025-12-10: Step 1 Complete - Go Project Initialized

Completed Step 1 (Initialize Go project) from Technical Approach.

**Research conducted:**

- Reviewed official Go module layout docs (go.dev/doc/modules/layout)
- Reviewed golang-standards/project-layout patterns (cloned to ~/context/golang-project-layout/)
- Reviewed Cobra user guide for CLI patterns
- Confirmed latest dependency versions via Go proxy

**Files created:**

- `go.mod` - Module with Cobra v1.10.2 dependency (CUE v0.15.1 added but not yet used)
- `go.sum` - Generated dependency checksums
- `cmd/start/main.go` - Minimal entry point, calls internal/cli.Execute()
- `internal/cli/root.go` - Root Cobra command with Execute() function

**Directory structure established:**

```
start/
├── go.mod
├── go.sum
├── cmd/
│   └── start/
│       └── main.go
├── internal/
│   ├── cli/
│   │   └── root.go
│   ├── config/
│   └── cue/
```

**Verified:**

- Build succeeds: `go build ./cmd/start/`
- CLI runs: `go run ./cmd/start/ --help` outputs description

**Next steps:**

- Step 2: Study reference implementations (Cobra patterns, prototype CLI, CUE Go API)
- Step 3: Implement `start init` command
