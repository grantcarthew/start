# DR-017: CLI Show Command

- Date: 2025-12-04
- Status: Accepted
- Category: CLI

## Problem

Users need to inspect resolved configuration content for debugging and understanding. This includes viewing roles, contexts, agents, and tasks after UTD processing (files read, commands executed, templates applied). Content can be large, making full terminal output impractical.

## Decision

The `start show` command displays resolved content after UTD processing and config merging. Output is written to a temporary directory with a 5-line preview shown in the terminal.

Synopsis:

```bash
start show role [name]
start show context [name]
start show agent [name]
start show task [name]
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

File-based output:

- Resolved content can be very large
- Users can inspect files with their preferred editor
- Markdown files get syntax highlighting
- Consistent with `--dry-run` output pattern

Inspect without executing:

- View role content without starting a session
- Check context resolution without running a task
- Verify agent configuration is correct

## Output Directory

Same pattern as `--dry-run`:

```
/tmp/start-YYYYMMDDHHmmss/
```

On collision (same second), append incrementing suffix: `-1`, `-2`, etc.

## Subcommands

### start show role

Display resolved role content after UTD processing.

```bash
start show role              # Show default role
start show role <name>       # Show named role
start show role --scope global
start show role --scope local
```

Output file: `role.md`

Terminal output:

```
Role: code-reviewer
===============================================================================
Source: global config

You are an expert code reviewer with deep knowledge of software
engineering best practices, security vulnerabilities, and performance
optimization.

Focus on:
... (342 more lines)

Full content: /tmp/start-20251204111532/role.md (2.3 KB)
```

### start show context

Display resolved context content after UTD processing.

```bash
start show context           # Show all contexts
start show context <name>    # Show named context
start show context --scope global
start show context --scope local
```

Output files:

- Single context: `context-<name>.md`
- All contexts: `contexts.md` (concatenated with headers)

Terminal output (single context):

```
Context: git-status
===============================================================================
Source: local config
Required: false
Tags: git, status
Command: git status --short

Working tree status:
 M main.go
 M README.md
?? newfile.go
... (12 more lines)

Full content: /tmp/start-20251204111532/context-git-status.md (456 bytes)
```

Terminal output (all contexts):

```
Contexts (4 total)
===============================================================================

environment (global, required):
  Read ~/reference/ENVIRONMENT.md for environment context.
  ... (2 more lines)

project (local, default):
  Read ./PROJECT.md. Respond with summary.

git-status (local, tagged: git):
  Working tree status:
   M main.go
  ... (8 more lines)

agents (local, required):
  Read ./AGENTS.md for repository overview.

Full content: /tmp/start-20251204111532/contexts.md (4.2 KB)
```

### start show agent

Display effective agent configuration after config merging.

```bash
start show agent             # Show default agent
start show agent <name>      # Show named agent
start show agent --scope global
start show agent --scope local
```

Output file: `agent-<name>.md`

Terminal output:

```
Agent: claude
===============================================================================
Source: global config
Description: Anthropic Claude via Claude Code CLI

Command template:
  {bin} --model {model} --system-prompt '{role}' '{prompt}'

Binary: claude
Default model: sonnet (claude-sonnet-4-20250514)

Models:
  haiku  -> claude-3-5-haiku-20241022
  sonnet -> claude-sonnet-4-20250514
  opus   -> claude-opus-4-20250514

Full content: /tmp/start-20251204111532/agent-claude.md (312 bytes)
```

Note: Agent configuration is typically small enough to display fully in terminal. File is still written for consistency.

### start show task

Display resolved task prompt template.

```bash
start show task <name>       # Show named task (required)
start show task --scope global
start show task --scope local
```

Output file: `task-<name>.md`

Terminal output:

```
Task: code-review
===============================================================================
Source: global config
Description: Review code for quality and best practices
Role: code-reviewer
Agent: (uses default)
Command: git diff --staged

Review the following changes:

## Instructions
{{.instructions}}
... (15 more lines)

Full content: /tmp/start-20251204111532/task-code-review.md (1.2 KB)
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
| `--quiet` | `-q` | Show only file path, no preview |
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

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Configuration error |
| 2 | Named item not found |
| 3 | File/directory error (UTD resolution failed) |

## Trade-offs

Accept:

- Executes UTD processing (commands run, files read) to show resolved content
- Creates temp files even for small content
- Another command to learn beyond config commands

Gain:

- Clear separation: config shows structure, show displays content
- Inspect individual components without full execution
- Debug UTD processing issues
- Understand config merging behavior
- Large content is inspectable in editor
- Consistent pattern with `--dry-run`

## Alternatives

Terminal-only output with truncation:

- Pro: No file management
- Pro: Simpler implementation
- Con: Large roles/contexts are unusable
- Con: Inconsistent with `--dry-run` approach
- Rejected: File-based is more practical and consistent

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
- Shared flags defined on parent show command

UTD processing:

- Reuse UTD resolution logic from execution path
- Handle errors gracefully (show error, exit with code 3)
- For contexts with commands, execute and display output

Output formatting:

- Write full content to temp file
- Display first 5 lines in terminal
- Show line count for remaining content
- Show file path and size

Temp directory:

- Same timestamp-based naming as `--dry-run`
- Reuse directory creation logic
- Handle collision with suffix
