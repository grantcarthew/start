# PROJECT.md - Session Continuity Document

Last Updated: 2025-12-04

## Session Summary

This session focused on reviewing prototype CLI commands and designing the new CUE-based CLI structure. Major decisions were made about command organization, auto-setup, and the wizard experience.

## Design Records Created This Session

| DR | Title | Summary |
|----|-------|---------|
| DR-016 | CLI Dry Run Flag | `--dry-run` on start/prompt/task writes to /tmp/start-YYYYMMDDHHmmss/, shows 5-line preview |
| DR-017 | CLI Show Command | `start show role/context/agent/task` for inspecting resolved content, same temp file pattern |
| DR-018 | CLI Auto-Setup | First run with no config: detect agents via index `bin` field, prompt if multiple, fetch package |
| DR-019 | Index Bin Field | Added `bin` field to #IndexEntry for agent PATH detection |

Updated existing DRs: DR-012, DR-013, DR-014, DR-015 (added --dry-run flag)

## CLI Command Structure Decided

### Agent-Launching Commands (have DRs)

| Command | Status | Notes |
|---------|--------|-------|
| `start` | DR-013 | Root command, launches agent with contexts |
| `start prompt [text]` | DR-014 | Quick prompt with required contexts only |
| `start task [name] [instructions]` | DR-015 | Run predefined workflow |

### Flags (DR-012, DR-016)

- Global: `--help`, `--version`, `--verbose`, `--debug`, `--quiet`
- Agent-launching: `--agent`, `--role`, `--model`, `--context`, `--dry-run`
- Path: `--directory`
- Config: `--local`

### Inspection Commands (DR-017)

| Command | Purpose |
|---------|---------|
| `start show role [name]` | Show resolved role content |
| `start show context [name]` | Show resolved context content |
| `start show agent [name]` | Show agent configuration |
| `start show task [name]` | Show task template |

All show commands write to /tmp/start-YYYYMMDDHHmmss/ with 5-line terminal preview.

### Auto-Setup (DR-018, DR-019)

First run with no config:
1. Fetch index from registry
2. Check each agent's `bin` field against PATH
3. One found → use it automatically
4. Multiple found → prompt user to select
5. Fetch selected agent's package
6. Write to ~/.config/start/
7. Continue with normal execution

Conditional command templates (Go templates) handle missing values gracefully:
```cue
command: "{{.bin}}{{if .model}} --model {{.model}}{{end}}{{if .role}} --append-system-prompt '{{.role}}'{{end}}{{if .prompt}} '{{.prompt}}'{{end}}"
```

## Commands Still To Design

### start assets (replaces start init)

Decision: `start init` is DITCHED. The wizard functionality moves to `start assets`.

Proposed structure:
| Command | Purpose |
|---------|---------|
| `start assets` | Full interactive wizard (browse registry, multi-select agents/roles/contexts/tasks) |
| `start assets add <query>` | Quick install specific item |
| `start assets search <query>` | Search registry |
| `start assets info <query>` | Show package details |
| `start assets update` | Update installed packages |

The `start assets` wizard flow (section by section):
1. Agents - browse, multi-select, choose default
2. Roles - browse, multi-select (optional)
3. Contexts - browse, multi-select (optional), AGENTS.md offered as default
4. Tasks - browse, multi-select (optional)
5. Summary and write config

Each section has a brief description explaining the concept.

Open questions for assets:
- `start assets browse` (open in browser) - still relevant with CUE registry?
- `start assets update` - how does this work with CUE modules?
- `--local` flag on `start assets` wizard?

### start config

Not yet discussed. Purpose: manage custom configuration (not from registry).

Prototype had:
- `start config show`
- `start config agent/role/task/context` subcommands

Needs design for CUE-based configuration.

### start doctor

Not yet discussed. Prototype: diagnose configuration issues.

## Schema Updates Made

Updated `reference/start-assets/schemas/`:
- `index.cue` - added `bin` field to #IndexEntry
- `index_example.cue` - added bin examples for agents
- `README.md` - documented agent detection flow

## Key Design Decisions

1. **Temp file output**: Both `--dry-run` and `start show` write to /tmp/start-YYYYMMDDHHmmss/ with 5-line preview in terminal. Handles large prompts gracefully.

2. **Conditional templates**: Agent command templates use Go template conditionals to handle missing values, enabling minimal config to work.

3. **Auto-setup vs wizard**: Auto-setup is automatic (first run, minimal). Wizard (`start assets`) is interactive (comprehensive setup).

4. **No --force on wizard**: `start assets` is always interactive. Auto-setup handles the automatic case.

5. **Registry browsing**: Wizard browses index.cue, lets users discover and select packages.

6. **AGENTS.md as default context**: The only context offered by default in wizard setup.

## Next Session TODO

1. Create DR for `start assets` command (wizard + subcommands)
2. Design `start config` command structure
3. Review `start doctor` from prototype
4. Consider lazy loading: `start task foo` auto-fetches if not found
5. Reconciliation: review DRs for consistency after this batch

## Files Modified This Session

Design records:
- docs/design/design-records/dr-012-cli-global-flags.md (updated)
- docs/design/design-records/dr-013-cli-start-command.md (updated)
- docs/design/design-records/dr-014-cli-prompt-command.md (updated)
- docs/design/design-records/dr-015-cli-task-command.md (updated)
- docs/design/design-records/dr-016-cli-dry-run-flag.md (created)
- docs/design/design-records/dr-017-cli-show-command.md (created)
- docs/design/design-records/dr-018-cli-auto-setup.md (created)
- docs/design/design-records/dr-019-index-bin-field.md (created)
- docs/design/design-records/README.md (updated)

Schemas:
- reference/start-assets/schemas/index.cue (updated)
- reference/start-assets/schemas/index_example.cue (updated)
- reference/start-assets/schemas/README.md (updated)
