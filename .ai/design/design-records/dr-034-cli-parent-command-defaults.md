# dr-034: CLI Parent Command Default Behaviour

- Date: 2025-12-23
- Status: Accepted
- Category: CLI

## Problem

Parent commands (commands with subcommands) in Cobra show help text when run without arguments. This is unhelpful for users who want to quickly see content. For example:

- `start config` shows help instead of listing configuration
- `start assets` shows help instead of listing installed assets
- `start show` shows help instead of displaying configuration

Users must discover and type the `list` or equivalent subcommand to see content, adding friction.

Additionally, Cobra's default `help` subcommand only works for commands without a `RunE`. Once a command has `RunE`, typing `start config help` passes "help" as an argument rather than showing help.

## Decision

Parent commands with subcommands run a sensible default action instead of showing help:

| Command | Default Action |
|---------|---------------|
| `start config` | List all config with paths |
| `start config agent` | List agents |
| `start config role` | List roles |
| `start config context` | List contexts |
| `start config task` | List tasks |
| `start config settings` | List settings |
| `start assets` | List installed assets |
| `start show` | Show all (agents, roles, contexts, tasks) |

Additionally:

1. `help` as a subcommand is explicitly handled and shows help
2. Unknown subcommands return an error with help suggestion

## Why

- Reduces friction: Users see useful content immediately
- Progressive disclosure: Simple command shows overview, subcommands provide detail
- Consistency: All parent commands behave the same way
- Discoverability: `help` subcommand works everywhere, not just for commands without `RunE`

## Trade-offs

Accept:

- Each parent command needs explicit `RunE` with help/unknown handling
- Slightly more code per parent command
- Help is no longer the default (users must use `--help` or `help` subcommand)

Gain:

- Immediate useful output from parent commands
- Consistent `help` subcommand across all commands
- Clear error messages for typos/unknown subcommands
- Better user experience for common operations

## Alternatives

Keep Cobra defaults (show help for parent commands):

- Pro: Zero implementation effort
- Pro: Standard Cobra behaviour
- Con: Users must type longer commands to see content
- Con: `help` subcommand doesn't work when `RunE` is defined
- Rejected: User friction outweighs implementation simplicity

## Usage Examples

```bash
# Parent command runs default action
start config              # Lists all config
start config agent        # Lists agents
start assets              # Lists installed assets
start show                # Shows all configuration

# Help subcommand works everywhere
start config help         # Shows config command help
start config agent help   # Shows agent command help
start assets help         # Shows assets command help

# Unknown subcommands show helpful error
start config xyz
# Error: unknown command "xyz" for "start config"
# Run 'start config --help' for usage
```

## Implementation Notes

Helper functions in `root.go`:

- `checkHelpArg(cmd, args)` - Returns true if first arg is "help", shows help
- `unknownCommandError(cmdPath, arg)` - Returns formatted error with help suggestion

Pattern for parent command `RunE`:

1. Check for `help` argument
2. Check for unknown arguments (error if any)
3. Run default action (typically the `list` subcommand)
