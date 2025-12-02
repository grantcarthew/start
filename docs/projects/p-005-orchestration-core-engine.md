# P-005: Orchestration Core

- Status: Proposed
- Started: -
- Completed: -

## Overview

Implement the core orchestration logic that ties everything together: load CUE configurations, compose prompts from roles/tasks/contexts, execute agent commands, and handle the complete end-to-end workflow. This is where start actually becomes a working AI agent orchestrator.

This project brings together all previous work (P-001 architecture, P-002 assets, P-003 distribution, P-004 CLI foundation) into a functioning system.

## Goals

1. Implement prompt composition from CUE configurations
2. Implement placeholder substitution in commands and prompts
3. Implement agent command execution
4. Implement task execution workflow
5. Implement process replacement execution model
6. Document orchestration architecture in design records
7. Create CLI documentation for execution commands
8. Validate end-to-end workflow with real assets

## Scope

In Scope:

- Prompt composition from roles, tasks, contexts
- Placeholder substitution system
- Agent command execution
- Task execution (start task <name>)
- Interactive session (start with role)
- Process replacement execution
- Shell quote escaping for substitution
- Context injection and ordering
- Create DR-006: Prompt Composition Architecture
- Create DR-007: Placeholder Substitution System
- Create DR-008: Execution Model and Process Replacement
- CLI documentation for task and interactive commands

Out of Scope:

- Advanced CLI features (completion, doctor, etc.)
- Package management commands
- Configuration editing commands
- Asset browsing and discovery
- Performance optimization
- Streaming output handling
- Error recovery and retry logic

## Success Criteria

- [ ] Can execute a task end-to-end with real CUE assets
- [ ] Can start interactive session with specified role
- [ ] Prompt composition works correctly (role + task + contexts)
- [ ] Placeholder substitution handles all required patterns
- [ ] Shell quote escaping prevents injection vulnerabilities
- [ ] Process replacement execution works (syscall.Exec on Unix)
- [ ] Context order preservation works for prompt injection
- [ ] Created DR-006: Prompt Composition Architecture
- [ ] Created DR-007: Placeholder Substitution System
- [ ] Created DR-008: Execution Model and Process Replacement
- [ ] CLI documentation complete for task and interactive commands
- [ ] Can demonstrate working system with example workflow

## Deliverables

Design Records:
- DR-006: Prompt Composition Architecture
- DR-007: Placeholder Substitution System
- DR-008: Execution Model and Process Replacement
- Possibly DR-009: Context Injection and Ordering (if complex)

CLI Commands:
- cmd/start/task.go - Task execution command
- cmd/start/interactive.go - Interactive session (or in root.go)

Go Implementation:
- internal/orchestration/composer.go - Prompt composition
- internal/orchestration/substitution.go - Placeholder substitution
- internal/orchestration/executor.go - Agent execution
- internal/orchestration/context.go - Context handling
- internal/orchestration/escaping.go - Shell quote escaping

CLI Documentation:
- docs/cli/start-task.md - Task execution documentation
- docs/cli/start-interactive.md - Interactive session documentation (or update start.md)

Architecture Documentation:
- docs/architecture/orchestration-flow.md - End-to-end flow documentation
- docs/architecture/prompt-composition.md - How prompts are built
- docs/architecture/execution-model.md - How agents are executed

Tests:
- Prompt composition tests
- Placeholder substitution tests
- Shell escaping tests
- End-to-end integration tests

## Dependencies

Requires:
- P-001 (need architecture foundation)
- P-002 (need assets to test with)
- P-003 (need to understand package loading)
- P-004 (need CLI foundation and CUE loading)

Blocks:
- Nothing - this completes the core system

## Technical Approach

Composition Phase:

1. Design prompt composition architecture
   - How roles, tasks, and contexts combine
   - Order of composition (role → task → contexts)
   - Override and precedence rules
   - Write DR-006 documenting decisions

2. Implement prompt composer
   - Load role definition from CUE
   - Load task definition from CUE
   - Load context definitions from CUE
   - Combine into final prompt
   - Preserve context order for injection

3. Test composition
   - Test with P-002 assets
   - Verify order preservation
   - Test override behavior
   - Validate output prompts

Substitution Phase:

4. Design placeholder substitution system
   - Identify all placeholder patterns needed
   - Design substitution algorithm
   - Handle edge cases and escaping
   - Write DR-007 documenting system

5. Implement placeholder substitution
   - Support {model}, {prompt}, {instructions}, etc.
   - Handle nested placeholders if needed
   - Implement shell quote escaping (critical for security)
   - Write DR-008 for escaping strategy (adapt from prototype DR-044)

6. Test substitution
   - Test all placeholder patterns
   - Test escaping with malicious input
   - Test edge cases (empty values, special chars)
   - Validate security (no injection vulnerabilities)

Execution Phase:

7. Design execution model
   - Process replacement vs child process
   - Signal handling
   - Exit code propagation
   - Write DR-008 documenting execution model (adapt from prototype DR-043)

8. Implement agent executor
   - Compose final command from agent template
   - Apply placeholder substitution
   - Execute with process replacement (syscall.Exec on Unix)
   - Handle Windows differences if needed

9. Test execution
   - Test with real agents (Claude, GPT if available)
   - Test with mock agents for CI
   - Test error handling
   - Verify process replacement works

Integration Phase:

10. Implement start task command
    - Parse task name
    - Load task from CUE
    - Compose prompt
    - Execute agent
    - Write CLI documentation

11. Implement interactive session
    - Use default or specified role
    - Compose prompt from role + contexts
    - Execute agent interactively
    - Write CLI documentation

12. End-to-end testing
    - Full workflow with real assets
    - Test multiple tasks
    - Test different roles
    - Test different agents

Documentation Phase:

13. Write all DRs
    - DR-006: Prompt Composition
    - DR-007: Placeholder Substitution
    - DR-008: Execution Model
    - Any additional DRs discovered

14. Write CLI documentation
    - start task command reference
    - Interactive session documentation
    - Usage examples and workflows

15. Write architecture documentation
    - Orchestration flow diagrams
    - Prompt composition details
    - Execution model explanation

## Questions & Uncertainties

Prompt Composition:
- What's the exact order of composition?
- How do task instructions override role prompts?
- How are contexts injected (prepend, append, interleave)?
- What's the format of the final composed prompt?
- How do we preserve context order from CUE?

Placeholder Substitution:
- What placeholders are needed beyond {model}, {prompt}, {instructions}?
- How do we handle missing placeholder values?
- Should substitution be recursive (placeholders in placeholders)?
- How do we validate placeholder syntax?
- What escaping is required for different contexts (shell, JSON, etc.)?

Execution Model:
- Does process replacement work on all platforms?
- How do we handle Windows (no exec equivalent)?
- What about signal handling and cleanup?
- How do we test process replacement?
- What's the fallback if exec fails?

Security:
- How do we prevent command injection via placeholders?
- Is shell quote escaping sufficient?
- Should we validate/sanitize inputs?
- What about path traversal in file references?

Context Handling:
- How do we determine which contexts to load?
- How do we handle missing required contexts?
- How do we validate context content?
- What if contexts reference files that don't exist?

Agent Compatibility:
- How do we handle different agent CLIs (different arg patterns)?
- What about agents that need API keys or config?
- How do we test without actual API access?
- What's the mock/test strategy?

## Research Areas

High Priority:

1. Prompt composition patterns
   - How to combine role, task, and contexts
   - Order and precedence
   - Override mechanisms
   - Format and structure

2. Placeholder substitution
   - All needed placeholders
   - Substitution algorithm
   - Edge case handling
   - Security considerations

3. Shell quote escaping
   - Escaping strategies for different shells
   - Security testing
   - Edge cases with special characters
   - Reference implementation patterns

Medium Priority:

4. Process replacement
   - syscall.Exec usage in Go
   - Platform differences
   - Error handling
   - Testing strategies

5. Context injection
   - How contexts are ordered
   - How order is preserved from CUE
   - How to inject into prompts
   - Format considerations

6. Agent command patterns
   - Common CLI patterns across agents
   - Template syntax needed
   - Compatibility considerations

Low Priority:

7. Performance
   - CUE loading performance
   - Composition overhead
   - Optimization opportunities

8. Advanced features
   - Streaming output
   - Interactive prompt building
   - Multi-agent workflows

## Notes

Prototype Patterns to Adapt:
- DR-043: Process Replacement Execution Model - Core concept still applies
- DR-044: Shell Quote Escaping - Security critical, adapt for CUE
- DR-007: Placeholders - Concept carries over, implementation differs

Key Security Consideration:
Shell quote escaping is critical. Improper escaping allows command injection. Must handle all edge cases and special characters correctly.

Context Order Preservation:
This was a core requirement that TOML couldn't satisfy. Must validate that CUE preserves field order and that we maintain it through composition.

Testing Strategy:
- Unit tests for composition, substitution, escaping
- Integration tests with mock agents
- Manual testing with real agents
- Security testing with malicious input

Success Demonstration:
Create a video or documented workflow showing:
1. start init creates project
2. start show validates config
3. start task pre-commit-review executes successfully
4. start with role go-expert launches interactive session

This proves the entire system works end-to-end.

This project is complete when we have a working orchestrator that can execute tasks and interactive sessions using CUE configurations and real agents. All core functionality implemented, all DRs written, all CLI docs complete.
