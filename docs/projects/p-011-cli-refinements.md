# P-011: CLI Refinements

- Status: Proposed
- Started:
- Completed:

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

- [ ] No command uses `show` in two different contexts
- [ ] All commands follow unified exit code scheme
- [ ] DR-034 documents command rename decision
- [ ] DR-035 documents exit code unification
- [ ] All tests pass
- [ ] Documentation updated (DR-012, DR-017, DR-029, command-tests.md)

## Deliverables

Files to modify:

- `internal/cli/config_agent.go` - Rename show subcommand
- `internal/cli/config_role.go` - Rename show subcommand
- `internal/cli/config_context.go` - Rename show subcommand
- `internal/cli/config_task.go` - Rename show subcommand
- `internal/cli/config_settings.go` - Verify consistency
- All affected `*_test.go` files
- `docs/command-tests.md` - Update examples

Design Records:

- DR-034: Config Show Rename
- DR-035: Unified Exit Codes

## Workflow

### Phase 1: Research and Design

- [ ] Read DR-017 (show command)
- [ ] Read DR-029 (config editing)
- [ ] Review current `config <type> show` implementation
- [ ] Discuss rename options with user
- [ ] Discuss exit code strategy with user
- [ ] Create DR-034 (rename decision)
- [ ] Create DR-035 (exit codes)

### Phase 2: Implementation - Command Rename

- [ ] Rename `config agent show` → chosen name
- [ ] Rename `config role show` → chosen name
- [ ] Rename `config context show` → chosen name
- [ ] Rename `config task show` → chosen name
- [ ] Update tests
- [ ] Verify help text is correct

### Phase 3: Implementation - Exit Codes

- [ ] Define exit code constants
- [ ] Update all commands to use unified codes
- [ ] Update tests for exit codes
- [ ] Document in code comments

### Phase 4: Documentation

- [ ] Update DR-012 flag applicability matrix
- [ ] Update DR-017 if needed
- [ ] Update DR-029 if needed
- [ ] Update command-tests.md
- [ ] Update any other affected docs

### Phase 5: Validation

- [ ] Run full test suite
- [ ] Manual testing of renamed commands
- [ ] Verify exit codes match documentation

## Questions & Uncertainties

Command rename options to discuss:

- `config <type> get` - kubectl-style, implies retrieval
- `config <type> view` - implies read-only display
- `config <type> inspect` - docker-style, implies detailed view
- `config <type> info` - npm-style, implies metadata

Exit code options to discuss:

- Option A: Unix minimal (0 success, 1 failure)
- Option B: BSD sysexits.h conventions (64-78 range)
- Option C: Semantic ranges (1-9 user, 10-19 config, 20-29 network)

## Dependencies

Requires:

- All prior CLI work (P-004 through P-010)

Blocks:

- None (refinement project)

## Notes

This project emerged from the documentation review conducted on 2025-12-22. The issues were identified as architectural concerns that, while not blocking, reduce CLI usability and consistency.
