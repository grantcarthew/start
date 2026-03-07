# CLI Command Structure

All commands use verb-first structure. `config` and `assets` follow the same pattern: verb at the top level, targets as arguments. Parent commands run a sensible default action rather than showing help.

## Top-Level Commands

```
start                              # Launch agent (required + default contexts)
start prompt [text]                # Launch agent with custom prompt (required contexts only)
start task [name] [instructions]   # Run a named task
start search <query>...            # Search local, global, and registry
start find <query>...              # Alias for search
start config [verb] [args]         # Manage configuration
start assets [verb] [args]         # Manage registry assets
start show [query]                 # Show resolved content
start doctor                       # Diagnose installation and configuration
start completion bash|zsh|fish     # Output shell completion script
```

## start config

Verb-first. No noun subcommands.

```
start config                          # List effective config with paths (default)
start config add [category]           # Add item; prompts for category if omitted
start config edit [query]             # Search by name, menu if multiple, then edit
start config remove [query]           # Search by name, menu if multiple, confirm, delete
start config list [category]          # List items; all categories if omitted
start config info [query]             # Search by name, show raw config fields
start config open [category]          # Open .cue file in $EDITOR
start config order [category]         # Reorder items (contexts and roles only)
start config reorder [category]       # Alias for order
start config search [query]           # Search by keyword across names, descriptions, tags
start config settings [key] [value]   # Manage settings
```

Valid categories: `agent`/`agents`, `role`/`roles`, `context`/`contexts`, `task`/`tasks`. Singular is canonical, plural is alias.

`config order` / `config reorder` applies to contexts and roles only. If supplied with `agent` or `task`, prompts user to choose context or role instead.

`config add` and `config edit` are always interactive (no field flags). Use `config open` for scripted changes.

`config remove` accepts `--yes` / `-y` to skip confirmation.

### No-argument behaviour

| Command | No argument |
|---------|-------------|
| `start config` | List all |
| `start config list` | List all |
| `start config add` | Prompt for category |
| `start config edit` | Prompt interactively |
| `start config remove` | Prompt category → item picker → confirmation |
| `start config info` | Prompt category → item → display |
| `start config open` | Prompt for category |
| `start config order` | Prompt for category |

## start assets

```
start assets                         # List installed assets (default)
start assets browse                  # Open asset repository in browser
start assets search <query>          # Search registry index only
start assets add <query>...          # Install asset(s) from registry
start assets list                    # List installed assets with update status
start assets info <query>            # Show detailed asset information
start assets update [query]          # Update installed assets
```

`assets search` is registry-only. `start search` covers local + global + registry.

## start show

```
start show [query]    # Search by name, show resolved content; list all if no query
```

Noun subcommands (`show agent`, `show role`, etc.) do not exist. Cross-category search by name with interactive menu on multiple matches.

Distinction from `config info`:

- `start config info <query>` — raw stored config fields (name, file path, tags, command template)
- `start show <query>` — resolved content after global+local merge, file read, command execution

## Parent Command Defaults

Parent commands run a sensible default rather than showing help:

| Command | Default action |
|---------|----------------|
| `start config` | List all config with paths |
| `start assets` | List installed assets |
| `start show` | Show all (all categories) |

`help` as a subcommand argument is explicitly handled and shows help. Unknown subcommands return an error with a help suggestion.

## Agent-Launching Commands

`start`, `start prompt`, and `start task` all support these flags (see cli-flags.md):

- `--agent` / `-a` — override agent
- `--role` / `-r` — override role
- `--model` / `-m` — override model
- `--context` / `-c` — select contexts
- `--dry-run` — preview without executing

### Context selection matrix

| Command | Required | Default | Tagged/searched |
|---------|----------|---------|-----------------|
| `start` | Yes | Yes | No |
| `start prompt` | Yes | No | No |
| `start task` | Yes | No | No |
| Any + `-c foo` | Yes | No | Matching `foo` |
| Any + `-c default,foo` | Yes | Yes | Matching `foo` |

### start task resolution

1. Exact match in installed config → run immediately
2. Exact match in registry index → install and run
3. Substring match across installed + registry → single: run; multiple: menu (TTY) or error (non-TTY)
4. No matches → error

## Asset Flag Resolution

All asset-selecting flags use the same search pattern:

Agent (`--agent`): exact config → exact registry → substring search → error

Role (`--role`): file path check → exact config → exact registry → substring search → error

Model (`--model`): exact agent models map → substring models map → passthrough to agent binary

Context (`--context`): file path check → exact config → exact registry → substring search (all matches above threshold included)

File path detection: values starting with `./`, `/`, or `~` are treated as file paths and bypass search.

Registry matches not installed locally are auto-installed before execution.

## Exit Codes

All commands: `0` on success, `1` on any error. Error messages printed to stderr describe the failure.
