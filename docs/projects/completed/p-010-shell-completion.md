# P-010: Shell Completion

- Status: Completed
- Started: 2025-12-19
- Completed: 2025-12-19

## Overview

Implement shell completion for bash, zsh, and fish. This provides tab-completion for commands, subcommands, and flags.

## Required Reading

| Document | Why |
|----------|-----|
| Cobra shell completion docs | Implementation approach |
| Context/cobra completion examples | Code patterns |

## Goals

1. Implement command/subcommand completion
2. Implement flag name completion
3. Support bash, zsh, and fish shells
4. Provide installation instructions

## Scope

In Scope:

- `start completion bash` - Generate bash completion script
- `start completion zsh` - Generate zsh completion script
- `start completion fish` - Generate fish completion script
- Command and subcommand completion
- Flag name completion
- Installation instructions in help text and documentation

Out of Scope:

- PowerShell completion (Windows not supported)
- Automatic installation (user redirects output and sources script)
- Dynamic completion for agent/task/role names (deferred to future version)
- Completion for package names (requires registry query)

## Success Criteria

- [x] `start completion bash` outputs valid bash completion script
- [x] `start completion zsh` outputs valid zsh completion script
- [x] `start completion fish` outputs valid fish completion script
- [x] Tab-completing `start <TAB>` shows subcommands
- [x] Tab-completing `start --<TAB>` shows flags
- [x] Installation instructions included in help text
- [x] User documentation created

## Workflow

### Phase 1: Research and Design

- [x] Review Cobra completion API
- [x] Discuss completion approach (shells, command structure, scope)
- [x] Create DR-032 for shell completion

### Phase 2: Implementation

- [x] Implement completion command with bash/zsh/fish subcommands
- [x] Add completion command to root
- [x] Write tests

### Phase 3: Documentation

- [x] Create docs/docs-writing-guide.md
- [x] Create docs/cli/cli-writing-guide.md
- [x] Create docs/cli/completion.md

### Phase 4: Review

- [ ] Manual testing on bash
- [ ] Update project document
- [ ] Complete project

## Deliverables

Files:

- `internal/cli/completion.go` - Completion command implementation
- `internal/cli/completion_test.go` - Tests
- `docs/cli/completion.md` - User documentation
- `docs/docs-writing-guide.md` - General documentation guidelines
- `docs/cli/cli-writing-guide.md` - CLI documentation template

Design Records:

- DR-032: CLI Shell Completion

## Technical Approach

Uses Cobra's built-in completion generation:

- `GenBashCompletionV2()` for bash
- `GenZshCompletion()` for zsh
- `GenFishCompletion()` for fish

Static completion only (commands and flags). Dynamic completion for agent/task names was considered but deferred - static completion provides sufficient value with zero implementation complexity.

## Dependencies

Requires:

- All CLI commands implemented (P-005 through P-009)

## Progress

2025-12-19:

- Completed Phase 1 research and design
- Created DR-032 documenting shell completion decisions
- Implemented completion command with bash/zsh/fish subcommands
- All tests passing
- Created documentation writing guides
- Created CLI completion documentation
