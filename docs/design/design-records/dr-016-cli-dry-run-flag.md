# DR-016: CLI Dry Run Flag

- Date: 2025-12-04
- Status: Accepted
- Category: CLI

## Problem

Users need to preview what an agent-launching command would do before executing it. This includes seeing which agent, role, and contexts would be used, the resolved prompt content, and the exact command that would execute. Prompts can be large (hundreds or thousands of lines), making terminal output impractical.

## Decision

Add a `--dry-run` flag to agent-launching commands (`start`, `start prompt`, `start task`). When set, the command performs all resolution and preparation steps, writes output files to a temporary directory, and exits without executing the agent.

Synopsis:

```bash
start --dry-run
start prompt "analyze security" --dry-run
start task code-review "focus on errors" --dry-run
```

## Why

Standard Unix pattern:

- `--dry-run` is widely understood (make, rsync, apt, etc.)
- Users intuitively know what it means
- Discoverable via `--help`

Clear semantics:

- Same command, same flags, just add `--dry-run`
- No confusion about which command to use for preview
- Muscle memory: type the command you want, add `--dry-run` to preview

File-based output:

- Prompts can be very large, terminal output is impractical
- Users can inspect files with their preferred editor
- Markdown files get syntax highlighting
- Files are self-contained and can be saved if needed

Separation of concerns:

- Execution preview belongs on execution commands
- Content inspection (viewing resolved roles, contexts) is a separate concern handled by `start show`

## Output Directory

Directory naming:

```
/tmp/start-YYYYMMDDHHmmss/
```

Example:

```
/tmp/start-20251204110941/
```

On collision (same second), append incrementing suffix: `-1`, `-2`, etc.

## Output Files

Three files are written:

```
/tmp/start-20251204110941/
  role.md        # Resolved system prompt after UTD processing
  prompt.md      # Composed prompt (contexts + task/user text)
  command.txt    # Command that would execute with metadata
```

role.md:

- The fully resolved role content
- After UTD processing (file read, command executed, template applied)
- This is the system prompt that would be sent to the agent

prompt.md:

- The fully composed user prompt
- Contexts concatenated in definition order
- Task prompt or user text appended
- After all UTD processing and placeholder substitution

command.txt:

```
# Agent: claude
# Model: claude-sonnet-4-20250514
# Role: code-reviewer
# Contexts: environment, project
# Working Directory: /Users/grant/Projects/myapp
# Generated: 2025-12-04T11:09:41+10:00

claude --model claude-sonnet-4-20250514 --system-prompt-file /tmp/start-20251204110941/role.md '...'
```

## Terminal Output

Summary with 5-line preview of each file:

```
Dry Run - Agent Not Executed
===============================================================================
Agent: claude (model: claude-sonnet-4-20250514)
Role: code-reviewer
Contexts: environment, project

Role (5 lines):
  You are an expert code reviewer with deep knowledge of software
  engineering best practices, security vulnerabilities, and performance
  optimization.

  Focus on:
  ... (342 more lines)

Prompt (5 lines):
  Read ~/reference/ENVIRONMENT.md for environment context.
  Read ~/reference/INDEX.csv for documentation index.
  Read ./AGENTS.md for repository overview.
  Read ./PROJECT.md. Respond with summary.

  ... (128 more lines)

Files: /tmp/start-20251204110941/
  role.md      (2.3 KB)
  prompt.md    (15.4 KB)
  command.txt
```

With `--verbose`:

- Shows context resolution details
- Shows file paths for each context
- Shows UTD processing steps

With `--debug`:

- Shows config merge steps
- Shows placeholder substitution
- Shows full resolution trace

With `--quiet`:

- Shows only file paths, no preview

## Execution Flow

When `--dry-run` is set:

1. Load configuration (global, local merge)
2. Select agent (flag, config default, first defined)
3. Select role (flag, task role, config default, first defined)
4. Select contexts (required + default or tagged)
5. Resolve all UTD content (files read, commands executed, templates applied)
6. Build prompt (concatenate contexts + task/prompt text)
7. Build agent command string
8. Create output directory `/tmp/start-YYYYMMDDHHmmss/`
9. Write role.md, prompt.md, command.txt
10. Display summary with file paths
11. Exit without executing agent

All resolution steps are identical to normal operation.

## Flag Definition

| Flag | Short | Description |
|------|-------|-------------|
| `--dry-run` | | Preview execution without launching agent |

No short form. Dry run is intentional, not a quick shortcut.

## Applicability

| Command | Applies |
|---------|---------|
| `start` | Yes |
| `start prompt` | Yes |
| `start task` | Yes |
| `start show` | No (already non-executing) |
| `start init` | No |
| `start config` | No |
| `start doctor` | No |

## Trade-offs

Accept:

- Commands execute UTD (files read, commands run) even in dry-run
- Task commands like `git diff --staged` run during preview
- Files left in /tmp for user to clean up (or OS will eventually)
- Slightly more typing than a dedicated preview command

Gain:

- Intuitive, standard pattern
- Same command structure for preview and execution
- Large prompts are inspectable in editor
- Files can be saved if user wants to keep them
- Clear separation from content inspection
- No command ambiguity

## Alternatives

Terminal output with truncation:

- Pro: No file management
- Pro: Immediate visibility
- Con: Large prompts are unusable
- Con: No way to see full content easily
- Rejected: Files are more practical for real-world prompt sizes

Write to project directory (.start/dry-run/):

- Pro: Easy to find, project-aware
- Con: Clutters project directory
- Con: Needs cleanup mechanism
- Rejected: /tmp is simpler, OS handles cleanup

Use `--preview` instead of `--dry-run`:

- Pro: Slightly more descriptive
- Con: Less standard than `--dry-run`
- Rejected: `--dry-run` is more widely recognized

Add `-n` short flag (like make):

- Pro: Quick to type
- Con: Easy to accidentally enable
- Rejected: Dry run should be intentional

## Implementation Notes

Cobra flag:

- Define on root command as persistent
- Check in PreRunE of agent-launching commands
- Create temp directory with timestamp naming

File writing:

- Use os.MkdirTemp or manual creation with timestamp
- Handle collision by appending suffix
- Write files with 0644 permissions
