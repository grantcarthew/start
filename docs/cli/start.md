# start

Launch an AI agent session with configured contexts and role.

## Usage

```
start [flags]
```

## Description

The `start` command launches an interactive AI agent session. It composes a prompt from your configured contexts, applies a role (system prompt), and hands off to your configured AI agent (Claude, Gemini, etc.).

By default, `start` includes required contexts and default contexts. Use flags to customise what's included.

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | `-a` | Override agent selection |
| `--role` | `-r` | Override role (config name or file path) |
| `--model` | `-m` | Override model selection |
| `--context` | `-c` | Select contexts (tags or file paths) |
| `--dry-run` | | Preview execution without launching agent |
| `--quiet` | `-q` | Suppress output |
| `--verbose` | | Detailed output |
| `--debug` | | Debug output (implies --verbose) |
| `--no-color` | | Disable coloured output |
| `--local` | `-l` | Target local config (./.start/) instead of global |

## File Path Support

The `--role` and `--context` flags accept file paths in addition to config names. File paths are detected by their prefix:

- `./` - relative to current directory
- `/` - absolute path
- `~` - relative to home directory

```bash
# Use a role from a file
start --role ./roles/reviewer.md

# Use a context from a file
start --context ./context/repo-info.md

# Mix file paths and config tags
start --context ./local.md,project,./other.md
```

## Examples

```bash
# Start with defaults
start

# Use a specific role
start --role go-expert

# Use a role from a file
start --role ~/prompts/my-role.md

# Preview what would be sent without launching the agent
start --dry-run

# Use a different agent
start --agent gemini

# Include additional contexts by tag
start --context security,performance
```

## See Also

- [prompt](prompt.md) - Launch with custom prompt text
- [task](task.md) - Run predefined tasks
