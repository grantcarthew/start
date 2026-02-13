# task

Run a predefined task with optional instructions.

## Usage

```
start task <name> [instructions]
```

## Description

The `task` command runs predefined workflows. Tasks are reusable prompts defined in your configuration that can include file content, command output, or template text.

Tasks only include required contexts by default, keeping the prompt focused on the task at hand. Use `-c` to include additional contexts.

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `name` | Yes | Task name (config name or file path) |
| `instructions` | No | Additional instructions passed to the task |

## File Path Support

The task name can be a file path instead of a config name. File paths are detected by their prefix:

- `./` - relative to current directory
- `/` - absolute path
- `~` - relative to home directory

```bash
# Run a task from a file
start task ./tasks/review.md

# Run with instructions
start task ~/prompts/analyze.md "Focus on security"
```

## Instructions

Instructions are passed to task templates via the `{{.instructions}}` placeholder. This allows tasks to be parameterised:

```bash
# Code review with specific focus
start task code-review "Focus on error handling"

# Documentation with specific scope
start task write-docs "API endpoints only"
```

## Task Resolution

Tasks are resolved in this order:

1. Exact match in configuration
2. Substring match (e.g., `review` matches `code-review`)
3. Registry lookup and auto-install

## Examples

```bash
# Run a configured task
start task code-review

# Substring matching
start task review

# From a file
start task ./my-task.md

# With instructions
start task code-review "Focus on performance"

# Include additional contexts
start task code-review -c security

# Preview without running
start task code-review --dry-run
```

## See Also

- [start](start.md) - Launch interactive session
- [prompt](prompt.md) - Launch with custom prompt
