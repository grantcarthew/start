# dr-015: CLI Task Command

- Date: 2025-12-03
- Status: Accepted
- Category: CLI

## Problem

Users need reusable workflows for common AI-assisted tasks like code review, documentation generation, and analysis. These workflows should be shareable via CUE packages while remaining flexible for local customization.

## Decision

The `start task` command runs predefined workflow tasks. Tasks use UTD for prompt generation and can be distributed as CUE packages with role dependencies.

Synopsis:

```bash
start task [name] [instructions] [flags]
```

## Why

Reusable workflows:

- Define once, run many times
- Share via CUE packages
- Customize with instructions argument

UTD for flexibility:

- Tasks use file, command, or prompt fields
- Dynamic content via commands (e.g., `git diff`)
- Placeholders for runtime data

Package dependencies:

- Packaged tasks import packaged roles
- CUE handles dependency resolution
- Role is fetched automatically with task

Agent is user's choice:

- Packaged tasks do not specify agents
- User's default agent or `--agent` flag
- Avoids binary dependency issues

## Task Resolution

Terminology:

- **Installed**: Task exists in user's config (local `./.start/` or global `~/.config/start/`)
- **Registry**: Task exists in CUE Central Registry but not yet installed

Resolution order:

1. Exact match installed → run immediately (no registry fetch)
2. No exact match installed → fetch registry index, then:
   a. Exact match in registry → install and run
   b. Substring match across installed + registry (deduplicated) →
      - Single match: run (install first if from registry)
      - Multiple matches: error (non-TTY) or interactive selection (TTY)
   c. No matches: error

Interactive selection:

- Display shows task name and source (installed/registry)
- Limited to 20 results; if more, prompt user to refine search
- User selects by number or by typing task name

```
Found 16 tasks matching "golang":

   1. golang/debug                       installed
   2. golang/refactor                    installed
   3. golang/github-homebrew-release     registry
   4. golang/review/architecture         installed
   ...

Select (1-16):
```

Large result sets:

```
Found 47 tasks matching "lang":

   1. golang/debug                       installed
   ...
  20. clang/format                       registry

Showing 20 of 47 matches. Refine search for more specific results.

Select (1-20):
```

## Execution Flow

1. Resolve task by name

2. If task is a package:
   - CUE fetches task module
   - CUE fetches role dependency (if declared)
   - Cache locally

3. Select agent
   - `--agent` flag if provided
   - Else task's `agent` field (if configured by user)
   - Else `default_agent` from config
   - Else first agent in config

4. Select role
   - `--role` flag if provided
   - Else task's `role` field (required for packaged tasks)
   - Else first role in config (definition order, skipping optional roles with missing files)

5. Select contexts
   - Required contexts only (same as `start prompt`)
   - Plus tagged contexts if `--context` provided

6. Resolve task UTD
   - Read file if specified
   - Execute command if specified (e.g., `git diff --staged`)
   - Apply Go template with placeholders:
     - `{{.instructions}}` - user's instructions argument
     - `{{.file_contents}}` - content from file field
     - `{{.command_output}}` - output from command field
     - `{{.date}}` - current timestamp

7. Resolve context UTD (definition order)

8. Build prompt
   - Context outputs first
   - Task prompt appended

9. Resolve role UTD

10. Execute agent

## Context Selection

Selection matrix:

| Command | Required | Default | Tagged |
|---------|----------|---------|--------|
| `start task foo` | Yes | No | No |
| `start task foo -c bar` | Yes | No | If tagged `bar` |
| `start task foo -c default` | Yes | Yes | No |

## Flags

Applicable flags (see dr-012):

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | `-a` | Override agent selection |
| `--role` | `-r` | Override role selection |
| `--model` | `-m` | Override model selection |
| `--context` | `-c` | Select contexts by tag |
| `--directory` | `-d` | Override working directory |
| `--dry-run` | | Preview execution without launching agent |
| `--quiet` | `-q` | Suppress output |
| `--verbose` | | Detailed output |
| `--debug` | | Full debug output |

## Arguments

name (required):

- Task name or prefix
- Resolved via resolution order above

instructions (optional):

- Fills `{{.instructions}}` placeholder in task prompt
- Multi-word instructions must be quoted
- If omitted, `{{.instructions}}` resolves to empty string

Examples:

```bash
start task code-review
start task code-review "focus on error handling"
start task review "check security"  # substring match
```

## Packaged Tasks

Structure:

- Task is a CUE module
- Imports role as CUE dependency
- Does not specify agent

Example task module:

```cue
import "github.com/grantcarthew/start-role-code-reviewer@v0"

tasks: "code-review": {
    role:        roles["code-reviewer"]
    description: "Review code for quality and best practices"
    command:     "git diff --staged"
    prompt: """
        Review the following changes:

        ## Instructions
        {{.instructions}}

        ## Changes
        ```diff
        {{.command_output}}
        ```
        """
}
```

Dependency resolution:

- CUE fetches task module
- CUE sees import, fetches role module
- Both cached locally
- Validation happens at CUE level

Agent behavior for packaged tasks:

| Scenario | Agent Used |
|----------|------------|
| `start task xyz` | Default agent or first |
| `start task xyz --agent claude` | claude |
| User configures `agent` in task | Configured agent |

## User Task Customization

Users can override packaged task settings in their config:

```cue
tasks: "code-review": {
    agent: "claude"      // Always use claude for this task
    timeout: 60          // Override timeout
}
```

Local config overrides global config overrides package defaults.

## Output

Normal mode:

```
Starting Task: code-review
===============================================================================
Agent: claude (model: claude-sonnet-4-20250514)

Context documents (required only):
  ✓ environment     ~/reference/ENVIRONMENT.md

Role: code-reviewer
Command: git diff --staged (42 lines)
Instructions: focus on error handling

Executing...
```

## Exit Codes

All commands use Unix minimal exit codes: 0 on success, 1 on any error. Error messages printed to stderr describe the specific failure.

## Trade-offs

Accept:

- Packaged tasks cannot recommend agents
- Users must have an agent configured
- Role dependency adds package complexity

Gain:

- Clean separation: tasks define workflow, users choose tools
- CUE handles role dependencies automatically
- No binary dependency issues in packages
- Predictable behavior across environments

## Alternatives

Tasks specify agents:

- Pro: Task author recommends best agent
- Pro: Consistent behavior for package users
- Con: Binary dependency outside CUE's domain
- Con: Fails if user doesn't have that agent
- Rejected: Agent is a user environment concern

Prompt for agent on first run:

- Pro: User explicitly chooses
- Pro: No silent defaults
- Con: Not scriptable
- Con: Annoying for automation
- Rejected: Defaults are better UX

Tasks bundle role content inline:

- Pro: Self-contained, no dependencies
- Pro: Simpler package structure
- Con: Cannot share roles across tasks
- Con: Role updates require task updates
- Rejected: Dependency model enables reuse

## Updates

- 2025-12-22: Changed task resolution from prefix match to substring match for better UX
- 2025-12-22: Aligned exit codes with unified policy (0 success, 1 failure)
- 2026-01-20: Unified task resolution across installed config and registry; added interactive selection with source labels; added 20-result limit for large result sets; clarified "installed" vs "registry" terminology
