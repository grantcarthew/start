# dr-014: CLI Prompt Command

- Date: 2025-12-03
- Status: Accepted
- Category: CLI

## Problem

Users need a way to run quick, focused queries with custom prompts while still benefiting from essential context. The root `start` command includes default contexts which may be unnecessary for one-off questions.

## Decision

The `start prompt` command launches an AI agent with a custom prompt and only required contexts. Default contexts are excluded to keep the prompt focused.

Synopsis:

```bash
start prompt [text] [flags]
```

## Why

Focused context:

- Only required contexts included (essential information)
- Default contexts excluded (reduces noise for quick queries)
- User prompt is the focus, not background context

Flexible usage:

- With text: custom prompt appended to required contexts
- Without text: just required contexts (interactive session)
- With `-c` flag: add specific tagged contexts

Complements root command:

- `start` = full context (required + default)
- `start prompt` = minimal context (required only)
- Clear semantic difference

## Execution Flow

1. Load configuration (global, local overrides global)

2. Select agent
   - `--agent` flag if provided
   - Else `default_agent` from config
   - Else first agent in config

3. Select role
   - `--role` flag if provided
   - Else `default_role` from config
   - Else first role in config

4. Select contexts
   - If `--context` flag provided: required + tagged contexts
   - Else: required contexts only
   - Default contexts never included (unless `-c default`)

5. Resolve context UTD (definition order)

6. Build prompt
   - Concatenate context outputs
   - Append custom prompt text (if provided)

7. Resolve role UTD

8. Execute agent

## Context Selection

Selection matrix:

| Command | Required | Default | Tagged |
|---------|----------|---------|--------|
| `start prompt` | Yes | No | No |
| `start prompt "text"` | Yes | No | No |
| `start prompt -c foo` | Yes | No | If tagged `foo` |
| `start prompt -c default` | Yes | Yes | No |
| `start prompt -c default,foo` | Yes | Yes | If tagged `foo` |

## Flags

Applicable flags (see dr-012):

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | `-a` | Override agent selection |
| `--role` | `-r` | Override role (system prompt) |
| `--model` | `-m` | Override model selection |
| `--context` | `-c` | Select contexts by tag |
| `--directory` | `-d` | Override working directory |
| `--dry-run` | | Preview execution without launching agent |
| `--quiet` | `-q` | Suppress output |
| `--verbose` | | Detailed output |
| `--debug` | | Full debug output |

## Arguments

text (optional):

- Custom prompt text to send to the agent
- Appended after context prompts
- Multi-word prompts must be quoted

Examples:

```bash
start prompt "analyze this codebase for security issues"
start prompt "what does the main function do?"
start prompt  # No custom text, just required contexts
```

## Output

Normal mode:

```
Starting AI Agent
===============================================================================
Agent: claude (model: claude-sonnet-4-20250514)

Custom prompt: analyze this codebase for security issues

Context documents (required only):
  ✓ environment     ~/reference/ENVIRONMENT.md

Role: code-reviewer

Executing...
```

## Exit Codes

All commands use Unix minimal exit codes: 0 on success, 1 on any error. Error messages printed to stderr describe the specific failure.

## Trade-offs

Accept:

- Users must understand required vs default distinction
- May need `-c default` to get full context

Gain:

- Focused prompts without noise
- Quick queries are fast
- Clear semantic difference from `start`

## Alternatives

Same behavior as `start` (include defaults):

- Pro: Simpler, one less thing to learn
- Pro: Consistent context across commands
- Con: No way to get focused context
- Con: One-off queries include unnecessary context
- Rejected: Semantic distinction is valuable

Require explicit context flags always:

- Pro: User always knows what's included
- Con: Verbose for common case
- Con: Loses benefit of required contexts
- Rejected: Required contexts should be automatic

## Updates

- 2025-12-22: Aligned exit codes with unified policy (0 success, 1 failure)
