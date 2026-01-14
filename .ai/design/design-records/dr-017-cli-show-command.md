# dr-017: CLI Show Command

- Date: 2025-12-04
- Status: Accepted
- Category: CLI

## Problem

Users need to inspect resolved configuration content for debugging and understanding. This includes viewing roles, contexts, agents, and tasks after UTD processing (files read, commands executed, templates applied).

## Decision

The `start show` command displays resolved content after UTD processing and config merging. Output goes directly to stdout. Plural aliases are supported for convenience.

Synopsis:

```bash
start show role [name]      # or: start show roles [name]
start show context [name]   # or: start show contexts [name]
start show agent [name]     # or: start show agents [name]
start show task [name]      # or: start show tasks [name]
```

## Why

Debugging support:

- See exactly what content would be sent to an agent
- Verify UTD processing worked correctly
- Check placeholder substitution results
- Understand config merging behavior

Distinct from raw config:

- Config commands show structure (fields, paths, settings)
- Show command displays resolved content (actual text)
- Different use cases, different commands

Direct stdout output:

- Simpler user experience (no temp files to manage)
- Content is immediately visible and pipeable
- For very large content, users can redirect to file if needed

Inspect without executing:

- View role content without starting a session
- Check context resolution without running a task
- Verify agent configuration is correct

Plural aliases:

- Natural language flexibility (`show agents` vs `show agent`)
- Consistent with user expectations from other CLIs

## Subcommands

### start show role

Display resolved role content after UTD processing.

```bash
start show role              # List roles, show first/default
start show role <name>       # Show named role
start show role --scope global
start show role --scope local
```

Output (no name - shows list and first role):

```
Roles: assistant, code-reviewer

Showing: assistant (first in config)
───────────────────────────────────────────────────────────────────────────────
Description: General purpose assistant

You are a helpful assistant skilled in software development.
```

Output (with name):

```
Role: code-reviewer
───────────────────────────────────────────────────────────────────────────────
Description: Expert code reviewer

You are an expert code reviewer with deep knowledge of software
engineering best practices, security vulnerabilities, and performance
optimization.
```

### start show context

Display resolved context content after UTD processing.

```bash
start show context           # List contexts with metadata
start show context <name>    # Show named context
start show context --scope global
start show context --scope local
```

Output (no name - list only with metadata):

```
Contexts: environment, project, git-status

Default: project
Required: environment

Tags: git, status
```

Output (with name):

```
Contexts: environment, project, git-status

Context: git-status
───────────────────────────────────────────────────────────────────────────────
Command: git status --short

Working tree status: {{.command_output}}
Tags: git, status
```

### start show agent

Display effective agent configuration after config merging.

```bash
start show agent             # List agents, show first/default
start show agent <name>      # Show named agent
start show agent --scope global
start show agent --scope local
```

Output (no name - shows list and first agent):

```
Agents: claude, gemini

Showing: claude (first in config)
───────────────────────────────────────────────────────────────────────────────
Description: Claude by Anthropic

Binary: claude
Command: {{.bin}} --model {{.model}} {{.prompt}}

Models:
  sonnet: claude-sonnet-4-20250514
  opus: claude-opus-4-20250514
```

### start show task

Display resolved task prompt template.

```bash
start show task              # List tasks
start show task <name>       # Show named task
start show task --scope global
start show task --scope local
```

Output (no name - list only):

```
Tasks: review, explain
```

Output (with name):

```
Tasks: review, explain

Task: review
───────────────────────────────────────────────────────────────────────────────
Description: Review staged changes

Command: git diff --staged

Review the following changes:

## Instructions
{{.instructions}}

## Changes
\`\`\`diff
{{.command_output}}
\`\`\`
Role: code-reviewer
```

Note: Task show displays the template with placeholders visible. Use `start task <name> --dry-run` to see fully resolved output with placeholders filled.

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--scope` | | Show from specific scope: `global` or `local` |

If `--scope` is omitted, shows effective/merged configuration.

Global flags that apply:

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | | Show additional metadata and resolution details |
| `--debug` | | Show full resolution trace |
| `--quiet` | `-q` | Minimal output |
| `--directory` | `-d` | Override working directory |

## Resolution Behavior

Without `--scope`:

- Shows effective configuration after merging
- Local overrides global for matching keys
- Indicates source in output header

With `--scope global`:

- Shows only global configuration
- Ignores local config entirely
- Useful for understanding base configuration

With `--scope local`:

- Shows only local configuration
- Ignores global config
- Useful for understanding project overrides

## Exit Codes

All commands use Unix minimal exit codes: 0 on success, 1 on any error. Error messages printed to stderr describe the specific failure.

## Trade-offs

Accept:

- Executes UTD processing (commands run, files read) to show resolved content
- Another command to learn beyond config commands
- Large content may overflow terminal

Gain:

- Clear separation: config shows structure, show displays content
- Inspect individual components without full execution
- Debug UTD processing issues
- Understand config merging behavior
- Simple output directly to stdout (pipeable, redirectable)
- No temp file management

## Alternatives

File-based output with preview:

- Pro: Handles very large content gracefully
- Pro: Consistent with `--dry-run` approach
- Con: Extra step to view full content
- Con: Temp files accumulate
- Rejected: Direct stdout simpler for typical use; users can redirect if needed

Extend config commands with --resolved flag:

- Pro: Fewer commands
- Con: Conflates structure and content viewing
- Con: Config commands are about management, not inspection
- Rejected: Separate concerns warrant separate commands

No show command, just use --dry-run:

- Pro: Simpler CLI surface
- Con: Cannot inspect individual components
- Con: Overkill for viewing a single role
- Rejected: Show serves distinct debugging needs

## Implementation Notes

Subcommand structure:

- `show` is a Cobra command with subcommands
- Each subcommand (role, context, agent, task) handles its type
- Plural aliases supported (agents, roles, contexts, tasks)
- Shared flags defined on parent show command

Default behavior by type:

- **agent/role**: List all + show first/default with content
- **context/task**: List only (no content unless name specified)

Rationale: Agents and roles have a "current" concept (first in config or default), so showing one makes sense. Contexts and tasks are collections selected by name or tag, so listing is more useful.

UTD processing:

- Reuse UTD resolution logic from execution path
- Handle errors gracefully (show error, exit with code 1)
- For contexts with commands, execute and display output

Output formatting:

- Full content written to stdout
- Separator lines for visual structure
- List of available items shown when applicable

## Updates

- 2025-12-12: Added --scope flag and resolution behavior
- 2025-12-22: Aligned exit codes with unified policy (0 success, 1 failure)
