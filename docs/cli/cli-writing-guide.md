# CLI Documentation Writing Guide

Guidelines for writing CLI command documentation.

Location: `docs/cli/`

Read when: Creating or updating command documentation.

---

## File Structure

One file per top-level command:

```
docs/cli/
├── cli-writing-guide.md   # This guide
├── completion.md          # start completion
├── config.md              # start config
├── doctor.md              # start doctor
└── ...
```

Commands with subcommands are documented in a single file with sections for each subcommand.

---

## Document Template

Every command doc should follow this structure:

````markdown
# command-name

One-line description of what the command does.

## Usage

```
start command-name [flags]
start command-name subcommand [flags]
```

## Description

A paragraph or two explaining when and why you'd use this command.
Keep it practical - what problem does it solve?

## Subcommands

(If applicable)

| Subcommand | Description |
|------------|-------------|
| `foo` | Does foo things |
| `bar` | Does bar things |

### foo

Detailed explanation of the foo subcommand.

```bash
start command-name foo
```

### bar

Detailed explanation of the bar subcommand.

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--verbose` | `-v` | Enable verbose output |
| `--output` | `-o` | Output file path |

## Examples

```bash
# Basic usage
start command-name

# With flags
start command-name --verbose

# Common workflow
start command-name foo && start command-name bar
```

## See Also

- [related-command](related-command.md) - Related functionality
````

---

## Section Guidelines

### Usage

Show the command syntax. Use `[flags]` for optional flags, `<required>` for required arguments.

```
start task <name> [instructions]
start config agent add <name> [flags]
```

### Description

Explain the "why" not just the "what". Help users understand when to use this command.

Good:
> Shell completion lets you press Tab to auto-complete commands and flags. Set it up once and save yourself a lot of typing.

Avoid:
> This command generates shell completion scripts.

### Subcommands

Use a table for quick scanning, then detail each subcommand in its own section.

### Flags

Document all flags in a table. Include both long and short forms. List required flags first.

For flags shared across commands (global flags), mention they're documented in the main CLI overview.

### Examples

Show real, working examples. Start simple and progress to more complex use cases.

Include comments to explain what each example does:

```bash
# Generate bash completion and install it
start completion bash > ~/.bash_completion.d/start

# Verify it's working
start <TAB>
```

### See Also

Link to related commands. Keep it to 2-3 relevant links, not an exhaustive list.

---

## Style Notes

### Command Formatting

- Use backticks for commands in prose: "Run `start completion bash`"
- Use code blocks for multi-line examples
- Show the prompt (`$`) only when showing command + output together

### Flag Documentation

- Document the most common flags first
- Group related flags together
- Note default values when non-obvious

### Keep It Scannable

Users often skim docs looking for specific information. Help them find it:

- Use clear headers
- Put the most important info first
- Use tables for structured data
- Keep paragraphs short
