# P-005: Orchestration Core Engine

- Status: Complete
- Started: 2025-12-12
- Completed: 2025-12-15

## Overview

Implement the core orchestration engine that ties everything together: auto-setup for first run, prompt composition from CUE configurations, UTD template processing, and agent execution. This is where `start` becomes a working AI agent orchestrator.

This project brings together all previous work (P-001 schemas, P-002 assets, P-003 distribution, P-004 CLI foundation) into a functioning system.

## Required Reading

Before working on this project, read these design records:

| DR | Title | Why |
|----|-------|-----|
| DR-005 | Go Templates for UTD Pattern | Template syntax, placeholders, execution flow |
| DR-006 | Shell Configuration and Command Execution | Shell selection, timeout, command execution |
| DR-007 | UTD Error Handling by Context | Fail vs warn based on entity type |
| DR-013 | CLI Start Command | Root command execution flow |
| DR-014 | CLI Prompt Command | Prompt command with minimal context |
| DR-015 | CLI Task Command | Task execution, resolution, dependencies |
| DR-016 | CLI Dry Run Flag | Dry run output, temp directory pattern |
| DR-018 | CLI Auto-Setup | First-run agent detection and configuration |
| DR-019 | Index Bin Field for Agent Detection | Agent binary detection |
| DR-020 | Template Processing and File Resolution | Temp file naming, `.start/temp/` directory |
| DR-024 | Testing Strategy | Test patterns, what to mock |

Also review:

- `docs/cue/integration-notes.md` - CUE Go API, value extraction
- P-004 deliverables (loader, validator, config paths)

## Goals

1. Implement auto-setup flow (DR-018)
2. Implement UTD template processing (DR-005, DR-020)
3. Implement shell command execution (DR-006)
4. Implement prompt composition from roles, tasks, contexts
5. Implement agent command execution
6. Implement `start`, `start prompt`, `start task` commands
7. Implement `--dry-run` flag (DR-016)
8. Validate end-to-end workflow with real assets

## Scope

In Scope:

- Auto-setup: agent detection, registry fetch, config write (DR-018)
- UTD template processing with Go templates (DR-005)
- Shell command execution with timeout (DR-006)
- Error handling by entity type (DR-007)
- Temp file management in `.start/temp/` (DR-020)
- Prompt composition from roles, tasks, contexts
- Agent command building and execution
- Process replacement execution model (syscall.Exec on Unix)
- `start` command (DR-013)
- `start prompt` command (DR-014)
- `start task` command (DR-015)
- `--dry-run` flag (DR-016)
- Agent-specific flags: `--agent`, `--role`, `--model`, `--context`
- Tests following DR-024 patterns

Out of Scope:

- Package management commands
- Configuration editing commands
- Shell completion
- Doctor/health check commands
- Windows support (Unix only per DR-006)
- Streaming output handling
- Multi-agent workflows

## Success Criteria

- [x] UTD templates process correctly (file, command, prompt fields)
- [x] Shell commands execute with timeout protection
- [x] Error handling follows DR-007 (tasks fail, contexts warn)
- [x] Temp files written to `.start/temp/` with meaningful names
- [x] `start` launches agent with merged config, contexts, role
- [x] `start prompt "text"` launches with custom prompt
- [x] `start task name` executes task workflow
- [x] `--dry-run` writes output files without executing agent

Deferred to P-006:

- Auto-setup (first run, agent detection, config writing)
- E2E validation with P-002 assets

## Deliverables

CLI Commands:

- `internal/cli/start.go` - Start command (RunE for root)
- `internal/cli/prompt.go` - Prompt command
- `internal/cli/task.go` - Task command

Orchestration:

- `internal/orchestration/autosetup.go` - First-run auto-setup
- `internal/orchestration/composer.go` - Prompt composition
- `internal/orchestration/executor.go` - Agent execution
- `internal/orchestration/template.go` - UTD template processing

Shell Execution:

- `internal/shell/runner.go` - Command execution with timeout
- `internal/shell/detection.go` - Shell auto-detection

Temp Files:

- `internal/temp/manager.go` - Temp directory and file management

Tests:

- Unit tests for template, shell, composer
- Integration tests for commands
- E2E tests for full workflows

## Non-Deliverables

These are explicitly NOT part of this project:

- New design records (all required DRs already exist)
- CLI documentation (after commands stabilise)
- Package management
- `start init` as a command (auto-setup replaces it)

## Technical Approach

### Phase 1: Auto-Setup (DR-018)

1. Implement agent detection
   - Fetch index from registry (or use cached)
   - Check each agent's `bin` field against PATH using `exec.LookPath()`
   - Return list of detected agents

2. Implement auto-setup flow
   - Check for existing config (`~/.config/start/`, `./.start/`)
   - If no config: enter auto-setup mode
   - If one agent detected: use it
   - If multiple detected: prompt for selection (TTY) or error (non-TTY)
   - Fetch agent package from registry
   - Write to `~/.config/start/agents.cue`
   - Continue with execution

3. Implement registry interaction
   - Fetch index
   - Fetch agent package
   - Use CUE module tooling

### Phase 2: UTD Template Processing (DR-005, DR-020)

1. Implement template processor
   - Parse content as Go template
   - Build template data map (file, command, date, instructions)
   - Execute template
   - Return resolved content

2. Implement lazy evaluation
   - Scan template for `{{.file_contents}}` - read file only if present
   - Scan template for `{{.command_output}}` - execute only if present
   - Avoid unnecessary I/O

3. Implement temp file management (DR-020)
   - Create `.start/temp/` directory
   - Generate path-derived names (e.g., `role-golang-assistant.md`)
   - Write resolved content
   - Warn if not in `.gitignore`

### Phase 3: Shell Execution (DR-006)

1. Implement shell runner
   - Parse shell string (e.g., `"bash -c"` → binary + flags)
   - Build command with user command appended
   - Set working directory
   - Execute with timeout (context.WithTimeout)
   - Capture stdout/stderr

2. Implement timeout handling
   - SIGTERM on timeout
   - Wait 1 second
   - SIGKILL if still running
   - Return partial output

3. Implement shell auto-detection
   - Check for bash in PATH
   - Fallback to sh
   - Error if neither found

### Phase 4: Prompt Composition

1. Implement prompt composer
    - Load selected contexts (required + default/tagged)
    - Process each context through UTD
    - Concatenate in definition order
    - Append task prompt or custom text
    - Return final composed prompt

2. Implement role resolution
    - Load selected role
    - Process through UTD
    - Return for system prompt injection

3. Implement agent command building
    - Load agent configuration
    - Build command from template
    - Substitute placeholders (role, prompt, model, bin)
    - Apply shell escaping

### Phase 5: Agent Execution

1. Implement agent executor
    - Build final command string
    - On Unix: use syscall.Exec for process replacement
    - Handle exec errors

2. Implement dry-run mode (DR-016)
    - Create `/tmp/start-YYYYMMDDHHmmss/`
    - Write role.md, prompt.md, command.txt
    - Display 5-line preview
    - Exit without executing agent

### Phase 6: Commands

1. Implement start command (DR-013)
    - Check for config, trigger auto-setup if needed
    - Load config, select agent/role/contexts
    - Compose prompt, resolve role
    - Execute agent (or dry-run)

2. Implement prompt command (DR-014)
    - Required contexts only (no defaults)
    - Custom text appended
    - Same execution flow

3. Implement task command (DR-015)
    - Resolve task by name
    - Process task UTD (with instructions)
    - Use task's role if specified
    - Same execution flow

### Phase 7: Testing

1. Write tests per DR-024
    - Template processing: real templates, table-driven
    - Shell execution: use `echo` and simple commands
    - Composer: real CUE configs via `t.TempDir()`
    - Commands: Cobra testing pattern

2. E2E testing
    - Full workflow with mock agent (echo binary)
    - Verify output files in dry-run mode
    - Test auto-setup with mocked PATH

## Directory Structure

```
start/
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   ├── start.go
│   │   ├── prompt.go
│   │   ├── task.go
│   │   └── flags.go
│   ├── orchestration/
│   │   ├── autosetup.go
│   │   ├── autosetup_test.go
│   │   ├── composer.go
│   │   ├── composer_test.go
│   │   ├── executor.go
│   │   ├── executor_test.go
│   │   ├── template.go
│   │   └── template_test.go
│   ├── shell/
│   │   ├── runner.go
│   │   ├── runner_test.go
│   │   ├── detection.go
│   │   └── detection_test.go
│   ├── temp/
│   │   ├── manager.go
│   │   └── manager_test.go
│   ├── cue/
│   │   └── (from P-004)
│   └── config/
│       └── (from P-004)
├── test/
│   ├── integration/
│   │   ├── start_test.go
│   │   ├── prompt_test.go
│   │   └── task_test.go
│   └── e2e/
│       └── workflow_test.go
└── scripts/
    └── invoke-tests
```

## Dependencies

Requires:

- P-001 (CUE schemas)
- P-002 (example assets to test with)
- P-003 (registry distribution)
- P-004 (CUE loading, validation, show command, global flags)

Blocks:

- Nothing - this completes the core system

## Questions Resolved

These questions are answered by existing design records:

| Question | Answer | Source |
|----------|--------|--------|
| What's the order of composition? | Contexts first (definition order), then task/prompt | DR-013, DR-015 |
| How are contexts selected? | Required always, default for `start`, tagged via `-c` | DR-013, DR-014, DR-015 |
| What placeholders are needed? | file, file_contents, command, command_output, date, instructions, role, role_file, prompt, bin, model | DR-005, DR-020 |
| How do we handle missing values? | Tasks fail, contexts/roles warn and skip | DR-007 |
| What shell to use? | User-specified string, or auto-detect bash/sh | DR-006 |
| Where do temp files go? | `.start/temp/` with path-derived names | DR-020 |
| How does process replacement work? | syscall.Exec on Unix, no Windows support | DR-006 |

## Progress

### Completed (2025-12-12)

Phase 2: UTD Template Processing

- `internal/orchestration/template.go` - Template processor with lazy evaluation
- Supports placeholders: File, FileContents, Command, CommandOutput, Date, Instructions
- Only reads files/executes commands when placeholder is actually used

Phase 3: Shell Command Execution

- `internal/shell/runner.go` - Command execution with timeout (SIGTERM/SIGKILL)
- `internal/shell/detection.go` - Shell auto-detection (bash, fallback to sh)
- Process group management for clean termination

Temp File Management

- `internal/temp/manager.go` - Dry-run directories and UTD temp files
- Path-derived filenames (e.g., `role-code-reviewer.md`)
- Gitignore checking for `.start/temp/`

Phase 4: Prompt Composition

- `internal/orchestration/composer.go` - Context selection and prompt building
- Supports required, default, and tagged context selection
- Role resolution with UTD processing
- Task resolution with instructions placeholder

Phase 5: Agent Execution

- `internal/orchestration/executor.go` - Command building and process replacement
- Model resolution from agent config
- Shell escaping for security
- Dry-run command generation

All unit tests passing (55+ test functions, 248 test cases across 6 packages).

Phase 6: CLI Commands

- `internal/cli/start.go` - Root command with RunE, dry-run support, flag handling
- `internal/cli/prompt.go` - Prompt command (required contexts only)
- `internal/cli/task.go` - Task command with instructions placeholder
- All commands support `--dry-run`, `--agent`, `--role`, `--model`, `--context` flags
- Process replacement via syscall.Exec for agent execution

### Deferred

Phase 1: Auto-Setup moved to P-006.

## Notes

Why auto-setup in P-005:

Auto-setup (DR-018) requires registry fetching which depends on P-003 distribution work. It also requires the CLI foundation from P-004. It's the first thing users experience, so it belongs in the orchestration project.

Security considerations:

- Shell quote escaping is critical (prevent command injection)
- Timeout protection prevents infinite loops
- Commands run with user's permissions (no escalation)
- Preview with `--dry-run` before execution

Testing strategy for agent execution:

- Create a mock agent binary (bash script that echoes arguments)
- Test command construction without actual API calls
- E2E tests use the mock agent
- Manual testing with real agents (Claude, etc.)

Success demonstration:

The project is complete when this workflow works:

1. User has no config, runs `start`
2. Auto-setup detects claude, fetches config
3. `start` launches Claude with contexts
4. `start task code-review` executes a task
5. `start --dry-run` shows what would execute
