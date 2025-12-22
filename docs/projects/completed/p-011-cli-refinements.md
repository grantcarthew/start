# P-011: CLI Refinements

- Status: Completed
- Started: 2025-12-22
- Completed: 2025-12-22

## Overview

Address CLI usability issues identified during documentation review. Two specific problems:

1. Command naming collision between `start config agent show` and `start show agent`
2. Inconsistent exit codes across commands

This project improves CLI coherence without changing core functionality.

## Required Reading

| Document | Why |
|----------|-----|
| DR-017 | Show command design and rationale |
| DR-029 | Config editing commands structure |
| DR-012 | Global flags and exit code references |
| docs/workflow.md | Development workflow |

## Goals

1. Resolve naming collision between `config <type> show` and `show <type>`
2. Establish unified exit code strategy across all commands
3. Update affected documentation (DRs, command-tests.md)
4. Maintain backward compatibility where feasible

## Scope

In Scope:

- Rename `config <type> show` subcommand to eliminate collision
- Define and document unified exit code scheme
- Update CLI implementation
- Update tests
- Update documentation

Out of Scope:

- New CLI commands
- Changes to CUE schemas
- Registry or asset changes
- Functional behavior changes beyond renaming

## Success Criteria

- [x] No command uses `show` in two different contexts
- [x] All commands follow unified exit code scheme
- [x] All tests pass
- [x] Documentation updated (DR-012, DR-017, DR-029, command-tests.md)

## Deliverables

Files modified:

- `internal/cli/config_agent.go` - Renamed show → info
- `internal/cli/config_role.go` - Renamed show → info
- `internal/cli/config_context.go` - Renamed show → info
- `internal/cli/config_task.go` - Renamed show → info
- `internal/cli/config_test.go` - Updated tests
- `internal/cli/config_integration_test.go` - Updated tests
- `docs/command-tests.md` - Updated examples

Design Records updated:

- DR-012: Added unified exit code policy (0 success, 1 failure)
- DR-013: Aligned exit codes with unified policy
- DR-014: Aligned exit codes with unified policy
- DR-015: Aligned exit codes with unified policy
- DR-017: Removed semantic exit codes, aligned with unified policy
- DR-018: Aligned exit codes with unified policy
- DR-027: Aligned exit codes with unified policy
- DR-028: Aligned exit codes with unified policy
- DR-029: Renamed show → info subcommand

## Workflow

### Phase 1: Research and Design

- [x] Read DR-017 (show command)
- [x] Read DR-029 (config editing)
- [x] Review current `config <type> show` implementation
- [x] Discuss rename options with user
- [x] Discuss exit code strategy with user

### Phase 2: Implementation - Command Rename

- [x] Rename `config agent show` → `config agent info`
- [x] Rename `config role show` → `config role info`
- [x] Rename `config context show` → `config context info`
- [x] Rename `config task show` → `config task info`
- [x] Update tests
- [x] Verify help text is correct

### Phase 3: Implementation - Exit Codes

- [x] Define unified exit code policy (0 success, 1 failure)
- [x] Document policy in DR-012 Exit Codes section
- [x] Update all DRs to reference unified policy

### Phase 4: Documentation

- [x] Update DR-012 with Exit Codes section
- [x] Update DR-017 to use unified exit codes
- [x] Update DR-029 with show → info rename
- [x] Update command-tests.md
- [x] Update all other affected DRs (DR-013, DR-014, DR-015, DR-018, DR-027, DR-028)

### Phase 5: Validation

- [x] Run full test suite
- [x] Manual verification of renamed commands
- [x] Verify exit codes match documentation

## Decisions Made

Command rename:

- Chose `info` over alternatives (`get`, `view`, `inspect`)
- Rationale: npm-style, implies metadata display, clear and concise
- Considered implicit `config agent <name>` pattern but rejected due to reserved word conflicts

Exit codes:

- Chose Unix minimal (0 success, 1 failure) over BSD sysexits.h and semantic ranges
- Rationale: Simple, universal, no edge case complexity
- Updated existing DRs rather than creating new DR-035

Documentation approach:

- Updated existing design records rather than creating new DR-034/DR-035
- Per dr-writing-guide.md: use Updates section with dated entries, no cross-links

## Dependencies

Requires:

- All prior CLI work (P-004 through P-010)

Blocks:

- None (refinement project)

## Notes

This project emerged from the documentation review conducted on 2025-12-22. The issues were identified as architectural concerns that, while not blocking, reduce CLI usability and consistency.

Completed in a single session. All implementation, tests, and documentation updated together.
