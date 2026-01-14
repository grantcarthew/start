# dr-033: Additional CLI Global Flags

- Date: 2025-12-19
- Status: Accepted
- Category: CLI

## Problem

The CLI is missing three global flags that were present in the prototype and are expected by users:

- Version information display
- Debug output for troubleshooting
- Working directory override for running from different locations

## Decision

Add three new global flags to the root command:

1. `--version` / `-v`: Display version information and exit
2. `--debug`: Enable debug output with `[DEBUG]` prefix
3. `--directory` / `-d`: Override working directory for all operations

## Specification

### --version (-v)

Output format (three lines, no labels):

```
start version x.x.x
https://github.com/grantcarthew/start
https://github.com/grantcarthew/start/issues/new
```

Behaviour:
- Prints version info and exits immediately
- Does not require configuration to exist
- Exit code 0

### --debug

Output format:
- Prefix debug lines with `[DEBUG]`
- Mixed with normal output

Debug output includes:
- Config file paths loaded
- Merge decisions (local overrides global)
- Placeholder resolution (e.g., `{model}` → `claude-sonnet-4`)
- Final command being executed

Behaviour:
- Implies `--verbose` (debug is a superset of verbose)
- When `--debug` is set, `--verbose` behaviour is also enabled

### --directory (-d)

Full override of working directory affecting:
- Local config lookup (`./.start/` relative to specified directory)
- Relative path resolution for context files (e.g., `./AGENTS.md`)
- Agent execution directory

Usage example:

```bash
cd ~
start --directory ~/projects/my-app
# Uses ~/projects/my-app/.start/ for local config
# Resolves ./AGENTS.md as ~/projects/my-app/AGENTS.md
# Executes agent in ~/projects/my-app/
```

Behaviour:
- Path is expanded (tilde, relative paths resolved)
- Error if directory does not exist
- Affects all path resolution throughout the command

## Why

These flags complete the CLI interface:

- `--version`: Standard expectation for CLI tools, needed for bug reports and compatibility checks
- `--debug`: Essential for troubleshooting config issues, placeholder problems, and understanding execution flow
- `--directory`: Enables running start from any location while targeting a specific project

## Trade-offs

Accept:
- Additional flag parsing complexity
- Debug output adds noise when enabled
- Directory validation adds an error path

Gain:
- Complete CLI feature set matching user expectations
- Better troubleshooting capability
- Flexibility in how users invoke the tool

## Alternatives

Version output with labels:
- `Repository: https://...` format
- Rejected: Extra visual noise, URLs are self-explanatory

Debug as separate verbosity level (--verbose --verbose):
- Rejected: Less discoverable, `--debug` is clearer intent

Directory affecting only execution (not config):
- Rejected: Inconsistent behaviour, users expect full context switch
