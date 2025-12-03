# DR-013: CLI Start Command

- Date: 2025-12-03
- Status: Accepted
- Category: CLI

## Problem

The root `start` command is the primary entry point for launching AI agents with context. It needs to orchestrate config loading, context selection, UTD resolution, and agent execution in a predictable way.

## Decision

The `start` command launches an AI agent with merged configuration, selected contexts, and a role as the system prompt.

Synopsis:

```bash
start [flags]
```

## Why

Single entry point:

- Users run `start` to begin an AI session
- All complexity hidden behind simple command
- Flags provide overrides when needed

Context-aware by default:

- Automatically includes required and default contexts
- No manual file specification needed
- Consistent context across sessions

Flexible when needed:

- `--context` flag for specialized workflows
- `--agent`, `--role`, `--model` for overrides
- `--directory` for different project contexts

## Execution Flow

1. Load configuration
   - Load global config (`~/.config/start/`)
   - Load local config (`./.start/`) if exists
   - Merge configs (local overrides global for matching keys)

2. Select agent
   - `--agent` flag if provided
   - Else `default_agent` from config
   - Else first agent in config (definition order)

3. Select role
   - `--role` flag if provided
   - Else `default_role` from config
   - Else first role in config (definition order)

4. Select contexts
   - If `--context` flag provided: required + tagged contexts
   - Else: required + default contexts
   - Filter to contexts matching criteria

5. Resolve context UTD
   - For each selected context (definition order):
     - Read file if specified
     - Execute command if specified
     - Apply Go template with placeholders
   - Skip contexts with resolution errors (warn)

6. Build prompt
   - Concatenate context outputs (definition order)
   - Required contexts first, then default/tagged

7. Resolve role UTD
   - Read file if specified
   - Execute command if specified
   - Apply Go template with placeholders

8. Execute agent
   - Build agent command from template
   - Inject role as system prompt
   - Inject composed prompt
   - Replace process with agent command

## Flags

Applicable flags (see DR-012):

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | `-a` | Override agent selection |
| `--role` | `-r` | Override role (system prompt) |
| `--model` | `-m` | Override model selection |
| `--context` | `-c` | Select contexts by tag |
| `--directory` | `-d` | Override working directory |
| `--quiet` | `-q` | Suppress output |
| `--verbose` | | Detailed output |
| `--debug` | | Full debug output |

## Context Selection

Selection matrix:

| Command | Required | Default | Tagged |
|---------|----------|---------|--------|
| `start` | Yes | Yes | No |
| `start -c foo` | Yes | No | If tagged `foo` |
| `start -c default,foo` | Yes | Yes | If tagged `foo` |

The pseudo-tag `default` explicitly includes default contexts when using `--context`.

## Output

Normal mode:

```
Starting AI Agent
===============================================================================
Agent: claude (model: claude-sonnet-4-20250514)

Context documents:
  ✓ environment     ~/reference/ENVIRONMENT.md
  ✓ project         ./PROJECT.md

Role: code-reviewer

Executing...
```

Quiet mode (`--quiet`):

- No output, launches agent directly

Verbose mode (`--verbose`):

- Shows config resolution details
- Shows full file paths and sizes
- Shows context resolution steps

Debug mode (`--debug`):

- Shows all internal operations
- Config merging details
- Placeholder resolution
- Command construction

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (agent launched) |
| 1 | Configuration error |
| 2 | Agent/role not found |
| 3 | File/directory error |
| 4 | Runtime error (agent failed to execute) |

## Trade-offs

Accept:

- Multiple resolution steps add complexity
- Silent context skip on errors may hide problems
- Process replacement means no post-agent cleanup

Gain:

- Simple user interface
- Predictable behavior
- Flexible overrides
- Clean process model (no wrapper overhead)

## Alternatives

Require explicit context specification:

- Pro: User knows exactly what's included
- Pro: No implicit behavior
- Con: Verbose for common case
- Con: Easy to forget important contexts
- Rejected: Automatic context inclusion is a core feature

Interactive context selection:

- Pro: User chooses at runtime
- Pro: No config needed
- Con: Slows down workflow
- Con: Not scriptable
- Rejected: Tags provide non-interactive selection
