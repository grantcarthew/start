# P-010: Shell Completion

- Status: Proposed
- Started: -
- Completed: -

## Overview

Implement shell completion for bash, zsh, and fish. This provides tab-completion for commands, subcommands, flags, and dynamic values (agent names, role names, task names, etc.).

## Required Reading

| Document | Why |
|----------|-----|
| Prototype DR-028 | Shell completion support |
| Cobra shell completion docs | Implementation approach |
| Context/cobra completion examples | Code patterns |

## Goals

1. Implement basic command/subcommand completion
2. Implement flag name completion
3. Implement dynamic value completion (agents, roles, tasks, contexts)
4. Support bash, zsh, and fish shells

## Scope

In Scope:

- `start completion bash` - Generate bash completion script
- `start completion zsh` - Generate zsh completion script
- `start completion fish` - Generate fish completion script
- Command and subcommand completion
- Flag name completion
- Dynamic completion for --agent, --role, --model, --context values
- Dynamic completion for task names
- Installation instructions in output

Out of Scope:

- PowerShell completion
- Automatic installation (user runs eval or sources script)
- Completion for package names (requires registry query)

## Success Criteria

- [ ] `start completion bash` outputs valid bash completion script
- [ ] `start completion zsh` outputs valid zsh completion script
- [ ] `start completion fish` outputs valid fish completion script
- [ ] Tab-completing `start <TAB>` shows subcommands
- [ ] Tab-completing `start --<TAB>` shows flags
- [ ] Tab-completing `start --agent <TAB>` shows configured agents
- [ ] Tab-completing `start task <TAB>` shows configured tasks
- [ ] Installation instructions included in output

## Workflow

### Phase 1: Research and Design

- [ ] Read all required documentation
- [ ] Review Cobra completion API
- [ ] Review prototype completion implementation
- [ ] Discuss dynamic completion approach
- [ ] Create DR for shell completion (if complex trade-offs)

### Phase 2: Implementation

- [ ] Implement basic completion command
- [ ] Implement dynamic agent/role/task completion
- [ ] Implement dynamic context tag completion
- [ ] Test on bash, zsh, fish

### Phase 3: Validation

- [ ] Write integration tests
- [ ] Manual testing on each shell
- [ ] Document installation steps

### Phase 4: Review

- [ ] External code review (if significant changes)
- [ ] Fix reported issues
- [ ] Update project document

## Deliverables

Files:

- `internal/cli/completion.go` - Completion command
- `internal/cli/completion_dynamic.go` - Dynamic value completers
- `docs/cli/completion.md` - User documentation

Design Records:

- DR-0XX: Shell Completion (if needed)

## Technical Approach

To be determined after Phase 1 research. Key questions:

1. How does Cobra's dynamic completion work?
2. How to load config for dynamic completion without slowing down shell?
3. Should completion cache config, or load fresh each time?
4. How to handle completion when config is invalid?

## Dependencies

Requires:

- P-004 (CUE loading for dynamic values)
- All CLI commands implemented (P-005, P-006, P-007, P-008, P-009)

Note: This project should be done last as it needs all commands to exist for complete coverage.

## Progress

(No progress yet - project not started)
