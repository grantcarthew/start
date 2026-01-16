# prompt

Launch an AI agent with a custom prompt.

## Usage

```
start prompt [text]
```

## Description

The `prompt` command launches an AI agent with custom prompt text. Unlike the default `start` command, it only includes required contexts (not default contexts), keeping the focus on your prompt.

Use this for one-off questions or tasks that don't need the full context setup.

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `text` | No | Prompt text or file path |

## File Path Support

The prompt argument can be a file path instead of inline text. File paths are detected by their prefix:

- `./` - relative to current directory
- `/` - absolute path
- `~` - relative to home directory

```bash
# Prompt from a file
start prompt ./prompts/analyze.md

# Prompt from home directory
start prompt ~/prompts/review.md
```

## Examples

```bash
# Inline prompt
start prompt "Explain this error message"

# From a file
start prompt ./prompts/question.md

# Include default contexts explicitly
start prompt "What can you help with?" -c default

# Include specific contexts
start prompt "Review this code" -c security

# Use a specific role
start prompt "Write a function to..." --role go-expert

# Preview without launching
start prompt "Test prompt" --dry-run
```

## See Also

- [start](start.md) - Launch with full context setup
- [task](task.md) - Run predefined tasks
