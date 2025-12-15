# P-009: Doctor & Diagnostics

- Status: Proposed
- Started: -
- Completed: -

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

- [ ] `start doctor` runs all checks and reports status
- [ ] Invalid CUE config detected and reported
- [ ] Missing agent binaries detected and reported
- [ ] Missing referenced files (roles, contexts) detected
- [ ] Clear pass/fail output for each check
- [ ] Exit code 0 for healthy, 1 for issues
- [ ] Actionable fix suggestions provided

## Workflow

### Phase 1: Research and Design

- [ ] Read all required documentation
- [ ] Review prototype doctor command for UX patterns
- [ ] Identify all checks to implement
- [ ] Discuss output format (table, list, verbose)
- [ ] Create DR for doctor command

### Phase 2: Implementation

- [ ] Implement config validation checks
- [ ] Implement agent binary checks
- [ ] Implement file existence checks
- [ ] Implement version display
- [ ] Implement fix suggestions

### Phase 3: Validation

- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Manual testing with broken configs

### Phase 4: Review

- [ ] External code review (if significant changes)
- [ ] Fix reported issues
- [ ] Update project document

## Deliverables

Files:

- `internal/cli/doctor.go` - Doctor command
- `internal/doctor/checks.go` - Individual check implementations
- `internal/doctor/reporter.go` - Output formatting

Design Records:

- DR-0XX: Doctor Command

## Technical Approach

To be determined after Phase 1 research. Key questions:

1. What checks are most valuable to users?
2. How verbose should default output be?
3. Should checks run in parallel or sequential?
4. How to format fix suggestions (commands, documentation links)?
5. Should there be a --fix flag for auto-fixable issues?

## Dependencies

Requires:

- P-004 (CUE loading/validation)
- P-006 (agent detection logic can be reused)

## Progress

(No progress yet - project not started)
