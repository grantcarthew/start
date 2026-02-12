# Project: Apply Terminal Colour Standard (DR-042)

GitHub Issue: #25 - feat: apply terminal colour standard (DR-042) across all output
Repository: grantcarthew/start

## Goal

Audit every CLI command that produces terminal output and apply consistent colours following DR-042. Each command is reviewed visually, adjusted if needed, then marked complete.

## Workflow

1. Agent copies the next command to clipboard: `echo "./start <command>" | clip`
2. User runs the command in a separate terminal and reports what they see
3. Agent adjusts code if colours need changing, rebuilds with `go build -o start .`
4. User re-runs until happy with the output
5. Agent marks the command complete in the checklist and moves to the next one

The binary must be run as `./start` (local build, not system-installed).

## Colour Standard (DR-042)

Source: `.ai/design/design-records/dr-042-terminal-colour-standard.md`
Definitions: `internal/cli/output.go`

### Message Types

| Element | Colour | Variable |
|---------|--------|----------|
| Errors | Red | `colorError` |
| Warnings | Yellow | `colorWarning` |
| Success markers | Green | `colorSuccess` |
| Headers/titles | Green | `colorHeader` |
| Separators | Magenta | `colorSeparator` |
| Dim/secondary | Faint | `colorDim` |

### Asset Categories

| Category | Colour | Variable |
|----------|--------|----------|
| agents | Blue | `colorAgents` |
| roles | Green | `colorRoles` |
| contexts | Cyan | `colorContexts` |
| tasks | HiYellow | `colorTasks` |

### Markers

| Marker | Colour | Usage |
|--------|--------|-------|
| Installed `*` | HiGreen | `colorInstalled` |
| Version arrows `->` | Blue | `colorBlue` |
| Parenthetical delimiters | Cyan | `colorCyan` |

### Formatting Rules

- Category names are coloured, trailing `/` is default
- Asset names are default colour
- Descriptions and metadata are dim (faint)
- When colours conflict, the more specific role wins
- Helper: `categoryColor(category)` maps category string to colour

## Known Issues From Investigation

- `output.go:13` comment says "DR-036", should say "DR-042"
- `assets_info.go` - no colours at all in `printAssetInfo()`
- `show.go` - no category colours on headers, separators uncoloured
- `config.go` - no colours on section headers, status markers, source info
- `resolve.go` - no colours on asset selection or install messages
- Doctor package (`internal/doctor/reporter.go`) has its own colour definitions and already follows DR-042

## Command Review Checklist

Review each command by running it, checking output against DR-042, adjusting code if needed.

### Display Commands

- [x] `./start show`
- [x] `./start show agent`
- [x] `./start show agent <name>`
- [x] `./start show role`
- [x] `./start show role <name>`
- [x] `./start show context`
- [x] `./start show context <name>`
- [x] `./start show task`
- [x] `./start show task <name>`

### Asset Commands

- [x] `./start assets list`
- [x] `./start assets list --verbose`
- [x] `./start assets search <query>`
- [x] `./start assets search <query> --verbose`
- [x] `./start assets info <query>`
- [x] `./start assets info <query> --verbose`
- [ ] `./start assets add <query>`
- [ ] `./start assets update`
- [ ] `./start assets update <query>`
- [ ] `./start assets browse`

### Search

- [ ] `./start search <query>`
- [ ] `./start search <query> --verbose`

### Config Overview

- [ ] `./start config`
- [ ] `./start config --local`

### Config Agent

- [ ] `./start config agent list`
- [ ] `./start config agent info <name>`
- [ ] `./start config agent add` (interactive)
- [ ] `./start config agent edit <name>` (interactive)
- [ ] `./start config agent remove <name>`
- [ ] `./start config agent default`
- [ ] `./start config agent default <name>`

### Config Role

- [ ] `./start config role list`
- [ ] `./start config role info <name>`
- [ ] `./start config role add` (interactive)
- [ ] `./start config role edit <name>` (interactive)
- [ ] `./start config role remove <name>`
- [ ] `./start config role default`
- [ ] `./start config role default <name>`
- [ ] `./start config role order` (interactive)

### Config Context

- [ ] `./start config context list`
- [ ] `./start config context info <name>`
- [ ] `./start config context add` (interactive)
- [ ] `./start config context edit <name>` (interactive)
- [ ] `./start config context remove <name>`
- [ ] `./start config context order` (interactive)

### Config Task

- [ ] `./start config task list`
- [ ] `./start config task info <name>`
- [ ] `./start config task add` (interactive)
- [ ] `./start config task edit <name>` (interactive)
- [ ] `./start config task remove <name>`

### Config Settings

- [ ] `./start config settings`
- [ ] `./start config settings <key> <value>`

### Config Order

- [ ] `./start config order`

### Doctor

- [ ] `./start doctor`

### Execution Commands

- [ ] `./start --dry-run`
- [ ] `./start task <name> --dry-run`
- [ ] `./start` (live launch - visual check of header/separator/tables)
- [ ] `./start task <name>` (live launch - visual check)

## Files Reference

| File | Role |
|------|------|
| `internal/cli/output.go` | Colour definitions and helper functions |
| `internal/cli/show.go` | show command and subcommands |
| `internal/cli/assets_list.go` | assets list |
| `internal/cli/assets_search.go` | assets search |
| `internal/cli/assets_info.go` | assets info |
| `internal/cli/assets_add.go` | assets add |
| `internal/cli/assets_update.go` | assets update |
| `internal/cli/assets_browse.go` | assets browse |
| `internal/cli/search.go` | search command |
| `internal/cli/config.go` | config overview |
| `internal/cli/config_agent.go` | config agent subcommands |
| `internal/cli/config_role.go` | config role subcommands |
| `internal/cli/config_context.go` | config context subcommands |
| `internal/cli/config_task.go` | config task subcommands |
| `internal/cli/config_settings.go` | config settings |
| `internal/cli/config_order.go` | config order / reorder |
| `internal/cli/resolve.go` | asset resolution prompts |
| `internal/cli/start.go` | start execution output |
| `internal/cli/task.go` | task execution output |
| `internal/cli/doctor.go` | doctor command (delegates to internal/doctor) |
| `internal/doctor/reporter.go` | doctor output formatting (own colour defs) |
