# CLI Flags

All flags are defined on the root command as persistent flags and inherited by subcommands. Flags passed to commands where they have no effect are silently ignored.

## Flag Reference

| Flag | Short | Description |
|------|-------|-------------|
| `--help` | `-h` | Show help |
| `--version` | `-v` | Show version and exit |
| `--verbose` | | Supplementary metadata output |
| `--debug` | | Full debug output (`[DEBUG] category: message`) |
| `--quiet` | `-q` | Suppress informational output |
| `--agent` | `-a` | Override agent selection |
| `--role` | `-r` | Override role (system prompt) |
| `--model` | `-m` | Override model selection |
| `--context` | `-c` | Select contexts by search term |
| `--dry-run` | | Preview execution without launching agent |
| `--directory` | `-d` | Override working directory |
| `--local` | `-l` | Target local config (`./.start/`) |
| `--no-role` | | Skip role resolution entirely |

## Applicability Matrix

`Y` = applies, `-` = silently ignored

| Flag | start | prompt | task | show | config | assets | search | doctor | completion |
|------|-------|--------|------|------|--------|--------|--------|--------|------------|
| `--help` | Y | Y | Y | Y | Y | Y | Y | Y | Y |
| `--version` | Y | Y | Y | Y | Y | Y | Y | Y | Y |
| `--verbose` | - | - | - | - | - | Y | Y | Y | - |
| `--debug` | Y | Y | Y | Y | Y | Y | Y | Y | - |
| `--quiet` | Y | Y | Y | Y | Y | Y | Y | Y | - |
| `--agent` | Y | Y | Y | - | - | - | - | - | - |
| `--role` | Y | Y | Y | - | - | - | - | - | - |
| `--model` | Y | Y | Y | - | - | - | - | - | - |
| `--context` | Y | Y | Y | - | - | - | - | - | - |
| `--dry-run` | Y | Y | Y | - | - | - | - | - | - |
| `--directory` | Y | Y | Y | Y | - | - | - | - | - |
| `--local` | - | - | - | - | Y | Y | - | - | - |
| `--no-role` | Y | Y | Y | - | - | - | - | - | - |

## Flag Details

`--version` / `-v`:

- Prints three lines and exits: version, repo URL, issues URL
- Does not require configuration to exist
- Exit code 0

`--verbose`:

- Shows supplementary metadata (scope, source, tags, file paths)
- Applies to `assets`, `search`, `doctor` — commands with displayable metadata
- No effect on execution commands (`start`, `prompt`, `task`) — use `--debug` for those

`--debug`:

- Format: `[DEBUG] <category>: <message>`
- Categories: `config`, `agent`, `role`, `context`, `task`, `compose`, `shell`, `exec`
- Implies `--verbose`
- Output goes to stderr

`--quiet` / `-q`:

- Suppresses informational output
- Agent output still shown
- Useful for scripting and pipelines

`--agent` / `-a`:

- Resolution: exact config match → exact registry → substring search → interactive (TTY) / error (non-TTY)
- Registry matches auto-installed before execution

`--role` / `-r`:

- Values starting with `./`, `/`, or `~` treated as file paths (bypass search)
- Otherwise: exact config → exact registry → substring search
- `--no-role` takes precedence over `--role` if both are supplied

`--model` / `-m`:

- Resolution: exact match in agent's models map → substring match → passthrough to agent binary
- Passthrough allows using model identifiers not in config

`--context` / `-c`:

- Values starting with `./`, `/`, or `~` treated as file paths
- Otherwise: unified search across context names, descriptions, and tags
- Supports comma-separated: `-c golang,security`
- Supports multiple flags: `-c golang -c security`
- All matches above score threshold are included (no ambiguity prompt)
- Warning emitted if no contexts match, execution continues

`--dry-run`:

- Performs all resolution steps without executing agent
- Writes three files to `/tmp/start-YYYYMMDDHHmmss/`: `role.md`, `prompt.md`, `command.txt`
- Shows 5-line preview of each file in terminal
- No short form (intentional — dry run should not be accidental)

`--directory` / `-d`:

- Overrides working directory for all operations: local config lookup, relative path resolution, agent execution
- Path is expanded (tilde and relative paths resolved)
- Error if directory does not exist

`--local` / `-l`:

- Targets local config (`./.start/`) instead of global (`~/.config/start/`)
- Applies to config editing and asset installation commands

## Exit Codes

All commands: `0` on success, `1` on any error. Error messages printed to stderr describe the failure.
