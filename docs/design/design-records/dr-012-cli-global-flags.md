# DR-012: CLI Global Flags

- Date: 2025-12-03
- Status: Accepted
- Category: CLI

## Problem

The CLI needs a consistent set of flags that work across commands. Flags fall into different categories based on which commands they apply to. The behavior when a flag is passed to an irrelevant command needs to be defined.

## Decision

Define global flags with clear applicability. Flags passed to commands where they have no effect are silently ignored.

Flag categories:

Truly global (all commands):

| Flag | Short | Description |
|------|-------|-------------|
| `--help` | `-h` | Show help |
| `--version` | `-v` | Show version |
| `--verbose` | | Detailed output |
| `--debug` | | Full debug output |
| `--quiet` | `-q` | Suppress output |

Agent-launching commands (start, prompt, task):

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | `-a` | Override agent selection |
| `--role` | `-r` | Override role (system prompt) |
| `--model` | `-m` | Override model selection |
| `--context` | `-c` | Select contexts by tag |

Path-dependent commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--directory` | `-d` | Override working directory |

Config-writing commands:

| Flag | Short | Description |
|------|-------|-------------|
| `--local` | `-l` | Target local config (`./.start/`) |

## Why

Consistent flag naming:

- Users learn flags once, apply everywhere
- Short flags for common operations (`-a`, `-r`, `-m`, `-c`, `-d`, `-l`, `-q`)
- Long flags are self-documenting

Silent ignore for irrelevant flags:

- User-friendly: no errors when building muscle memory
- Avoids frustrating "unknown flag" errors
- Users can alias commands with flags without worrying about subcommand compatibility
- Example: `alias s="start --verbose"` works for `s task foo` and `s config show`

## Trade-offs

Accept:

- Silent ignore may hide user mistakes (typos in flag names still error)
- Users may not realize a flag has no effect on certain commands
- More flags to document per command (applicability matrix)

Gain:

- Consistent UX across all commands
- No frustrating errors for irrelevant flags
- Simpler mental model: flags are "hints" not "requirements"
- Aliases and scripts work broadly

## Alternatives

Error on irrelevant flags:

- Pro: Strict, catches mistakes
- Pro: Clear feedback on flag applicability
- Con: Frustrating UX, especially with aliases
- Con: Users must memorize which flags work where
- Rejected: User friction outweighs strictness benefits

Warn on irrelevant flags:

- Pro: Informs user without blocking
- Pro: Helps users learn applicability
- Con: Noisy output, especially with `--quiet`
- Con: Warnings become noise users ignore
- Rejected: Adds noise without significant benefit

## Flag Applicability Matrix

| Flag | start | prompt | task | show | assets | config | init | doctor |
|------|-------|--------|------|------|--------|--------|------|--------|
| `--help` | Y | Y | Y | Y | Y | Y | Y | Y |
| `--version` | Y | Y | Y | Y | Y | Y | Y | Y |
| `--verbose` | Y | Y | Y | Y | Y | Y | Y | Y |
| `--debug` | Y | Y | Y | Y | Y | Y | Y | Y |
| `--quiet` | Y | Y | Y | Y | Y | Y | Y | Y |
| `--agent` | Y | Y | Y | Y | - | - | - | - |
| `--role` | Y | Y | Y | Y | - | - | - | - |
| `--model` | Y | Y | Y | Y | - | - | - | - |
| `--context` | Y | Y | Y | - | - | - | - | - |
| `--directory` | Y | Y | Y | Y | - | - | - | - |
| `--local` | - | - | - | - | Y | Y | Y | - |

Y = applies, - = silently ignored

## Flag Details

Agent flag (`--agent`, `-a`):

- Overrides default agent from config
- Resolution: exact match first, then prefix match
- Prefix ambiguity: error in non-TTY, interactive selection in TTY

Role flag (`--role`, `-r`):

- Overrides default role from config
- Resolution: same as agent (exact, then prefix)

Model flag (`--model`, `-m`):

- Overrides default model for selected agent
- Resolution: exact match in agent's models, then prefix match, then passthrough to agent
- Passthrough allows using model identifiers not in config

Context flag (`--context`, `-c`):

- Selects contexts by tag (see DR-008)
- Supports comma-separated: `-c golang,security`
- Supports multiple flags: `-c golang -c security`
- Reserved pseudo-tag `default` selects default contexts

Directory flag (`--directory`, `-d`):

- Overrides working directory for context resolution
- Relative paths in config resolve from this directory

Local flag (`--local`, `-l`):

- Targets local config (`./.start/`) instead of global (`~/.config/start/`)
- Applies to: init, config editing, asset installation

Quiet flag (`--quiet`, `-q`):

- Suppresses informational output
- Agent output still shown
- Useful for scripting and pipelines

Verbose flag (`--verbose`):

- Shows detailed operation information
- Config resolution, file paths, sizes
- Useful for understanding behavior

Debug flag (`--debug`):

- Shows all internal operations
- Config merging, placeholder resolution, command construction
- Useful for troubleshooting

## Implementation Notes

Cobra persistent flags:

- Define all flags on root command as persistent
- Subcommands inherit automatically
- Check flag values only in commands where they apply
- Unused flags naturally ignored (no explicit handling needed)

Flag value access:

- Use Cobra's flag binding to struct fields
- Centralize flag definitions in single location
- Consistent naming between flag and struct field
