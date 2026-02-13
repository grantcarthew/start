# dr-030: Settings Schema

- Date: 2025-12-18
- Status: Accepted
- Category: Config

## Problem

Configuration files follow a naming convention where the filename matches the top-level key:

| File | Top-level key |
|------|---------------|
| `agents.cue` | `agents:` |
| `roles.cue` | `roles:` |
| `contexts.cue` | `contexts:` |
| `tasks.cue` | `tasks:` |
| `config.cue` | `settings:` |

The `config.cue` filename is inconsistent. Additionally, there is no formal schema for settings and no CLI command to manage them.

## Decision

1. Rename `config.cue` to `settings.cue` for consistency
2. Create `#Settings` schema with four fields
3. Add `start config settings` command with positional get/set interface

## Why

Filename consistency:

- All other config files follow `<key>.cue` naming
- `settings.cue` containing `settings:` is intuitive
- Reduces cognitive load when navigating config directory

Formal schema:

- Enables validation at load time
- Documents available settings
- Consistent with other config type schemas

CLI management:

- Enables scripted configuration without manual file editing
- Provides discoverability via listing

## Structure

```cue
#Settings: {
    default_agent?: string & !=""
    shell?:         string & !=""
    timeout?:       int & >0
}
```

All fields are optional. Schema is closed (no unknown fields).

| Field | Type | Description |
|-------|------|-------------|
| `default_agent` | string | Agent when `--agent` not specified |
| `shell` | string | Shell for command execution (default: auto-detect) |
| `timeout` | int | Command timeout in seconds |

## CLI Interface

Positional argument design:

```bash
start config settings                    # List all settings
start config settings shell              # Get shell value
start config settings shell /bin/zsh     # Set shell value
start config settings edit               # Open in $EDITOR
```

The `edit` keyword is reserved and cannot be used as a setting name.

To clear a setting, set to empty string or use `edit`.

The existing `start config agent default` subcommand remains unchanged. Both approaches work:

```bash
start config agent default claude          # Via agent command
start config settings default_agent claude # Via settings command
```

## Trade-offs

Accept:

- Breaking change for existing users (must rename config.cue)
- `edit` is a reserved keyword
- Two ways to set default_agent

Gain:

- Consistent naming convention
- Formal validation of settings
- CLI management without manual editing
- Flexibility in how users prefer to work

## Alternatives

Explicit subcommands (`list`, `show`, `set`, `unset`):

- More verbose but explicit
- Rejected: positional interface is cleaner for key-value data

Remove `default` subcommand from agent/role:

- Single source of truth
- Rejected: both approaches are valid, user preference

Keep `config.cue` filename:

- No breaking change
- Rejected: consistency is worth one-time migration

## Migration

Existing `config.cue` files continue to work (CUE loads all `.cue` files). For clean migration:

```bash
mv ~/.config/start/config.cue ~/.config/start/settings.cue
mv .start/config.cue .start/settings.cue
```

## Updates

2026-02-13: Removed `default_role` field (GitHub issue #36). Role resolution uses definition order with `start config role reorder` to control the default. The `default_role` setting was redundant and strictly less capable than the definition-order fallback chain (it bypassed optional role skipping).
