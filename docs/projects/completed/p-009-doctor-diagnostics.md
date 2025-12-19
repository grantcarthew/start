# P-009: Doctor & Diagnostics

- Status: Complete
- Started: 2025-12-19
- Completed: 2025-12-19

## Overview

Implement a doctor command for health checks, configuration validation, and diagnostics. This helps users identify and fix configuration issues, missing dependencies, and environment problems.

## Required Reading

| Document | Why |
|----------|-----|
| Prototype DR-024 | Doctor exit codes |
| Prototype start-doctor.md | Prior art for CLI interface |
| DR-018 | Auto-setup (agent detection) |
| DR-019 | Index bin field (PATH checking) |

## Goals

1. Design CLI interface for doctor command
2. Implement configuration validation checks
3. Implement agent binary detection
4. Implement environment checks
5. Implement fix suggestions

## Scope

In Scope:

- `start doctor` - Run all health checks
- Configuration file validation (syntax, schema)
- Agent binary detection (are configured agents installed?)
- Referenced file existence checks
- Version information display
- Clear pass/fail output with fix suggestions

Out of Scope:

- Automatic fixing (suggestions only)
- Network connectivity checks
- Agent-specific health checks (e.g., API key validation)
- Performance diagnostics

## Success Criteria

- [x] `start doctor` runs all checks and reports status
- [x] Invalid CUE config detected and reported
- [x] Missing agent binaries detected and reported
- [x] Missing referenced files (roles, contexts) detected
- [x] Clear pass/fail output for each check
- [x] Exit code 0 for healthy, 1 for issues
- [x] Actionable fix suggestions provided

## Workflow

### Phase 1: Research and Design

- [x] Read all required documentation
- [x] Review prototype doctor command for UX patterns
- [x] Identify all checks to implement
- [x] Discuss output format (table, list, verbose)
- [x] Create DR for doctor command

### Phase 2: Implementation

- [x] Implement config validation checks
- [x] Implement agent binary checks
- [x] Implement file existence checks
- [x] Implement version display
- [x] Implement fix suggestions

### Phase 3: Validation

- [x] Write unit tests
- [x] Write integration tests
- [x] Manual testing with broken configs

### Phase 4: Review

- [x] External code review (skipped)
- [x] Fix reported issues (N/A)
- [x] Update project document

## Deliverables

Files:

- `internal/cli/doctor.go` - Doctor command and CLI integration
- `internal/doctor/doctor.go` - Core types and result structures
- `internal/doctor/checks.go` - Individual check implementations
- `internal/doctor/reporter.go` - Output formatting
- `internal/doctor/doctor_test.go` - Unit tests for core types
- `internal/doctor/checks_test.go` - Unit tests for checks

Design Records:

- DR-031: CLI Doctor Command

## Technical Approach

Decisions from Phase 1 research (see DR-031):

1. Checks: Intro (repo info), Version, Configuration, Agents, Contexts, Roles, Environment
2. Output format: Prototype style with sections and checkmark/cross/warning indicators
3. Sequential execution (simple, predictable output order)
4. Fix suggestions as actionable text (install commands, config edits)
5. No --fix flag (out of scope, suggestions only)
6. Binary exit codes: 0 healthy, 1 issues (no severity distinction)
7. Add intro section with repo URL and issues link for support

## Dependencies

Requires:

- P-004 (CUE loading/validation)
- P-006 (agent detection logic can be reused)

## Progress

2025-12-19: Phase 1 complete. Created DR-031 documenting CLI interface, checks, output format, and exit codes. Ready for Phase 2 implementation.

2025-12-19: Phase 2 and 3 complete. Implemented all checks (intro, version, configuration, agents, contexts, roles, environment). Added reporter with support for normal, quiet, and verbose modes. Unit tests passing. Remaining: integration tests and external code review.

2025-12-19: Project complete. Integration tests added (14 tests in test/integration/doctor_test.go). Code review skipped.
