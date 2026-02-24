# P-035: CLI Config Open Command

- Status: Complete
- Started: 2026-02-24
- Completed: 2026-02-24

## Progress Tracking

Update this document as work proceeds:

- Set Status to In Progress and fill in Started when work begins
- Mark each phase complete in the Phases section as it finishes
- Tick off Success Criteria checkboxes as each item is verified
- Set Status to Complete and fill in Completed when all criteria pass
- Move this file to `.ai/projects/completed/` and update `AGENTS.md`

## Overview

Adds `start config open [category]` as a new top-level config verb command. It opens the appropriate `.cue` configuration file in `$EDITOR`, prompting for the category when none is supplied.

This command was originally scoped to p-032 (CLI Config Verb-First Refactor) but is extracted here because it has no dependency on noun-group removal — it can land cleanly on top of the current codebase as a new additive command. Delivering it first reduces p-032's scope and lets users access file-based editing earlier.

The command replaces the `config <type> edit` (no name) behaviour currently scattered across the four noun-group files. After p-032 removes those files, `config open` is the canonical way to edit a CUE file directly.

## Required Reading

| Document | Why |
| --- | --- |
| dr-026 | CLI logic and I/O separation |
| dr-042 | Terminal colour standard |

## Current State

The noun-group files implement "no name = open file" as a special case inside each `edit` command:

- `config agent edit` (no name) → `openInEditor(agentPath)` (config_agent.go:343)
- `config role edit` (no name) → `openInEditor(rolePath)`
- `config context edit` (no name) → `openInEditor(contextPath)`
- `config task edit` (no name) → `openInEditor(taskPath)`

After this project, `start config open [category]` replaces that pattern with a dedicated, discoverable command.

Existing infrastructure `config_open.go` will use directly:

- `openInEditor(path string) error` — config_helpers.go:446, opens $EDITOR (fallback $VISUAL, then vi)
- `promptSelectCategory(w io.Writer, r io.Reader, categories []string) (string, error)` — config_helpers.go:633, numbered interactive menu; returns `"", nil` on cancel (empty input)
- `config.ResolvePaths("") (Paths, error)` — resolves global/local config directory paths
- `paths.Dir(local bool) string` — returns global or local config directory path
- `getFlags(cmd *cobra.Command) *Flags` — start.go:44, fields include `.Local bool`
- `isTerminal(r io.Reader) bool` — root.go:159, used to guard interactive paths

File name mapping:

| Category | File |
| --- | --- |
| agent | agents.cue |
| role | roles.cue |
| context | contexts.cue |
| task | tasks.cue |
| setting | settings.cue |

The settings file path can be derived: `filepath.Join(paths.Dir(local), "settings.cue")`.

The `--local` flag determines which directory is targeted (same as all other config commands).

Non-interactive no-arg behavior: when no category is supplied and stdin is not a terminal, return an error — category required. The interactive prompt path runs only when `isTerminal(stdin)` is true.

Related existing function for reference (settings only):

- `editSettings(localOnly bool) error` — config_settings.go:229, also calls `openInEditor` but additionally pre-creates the settings file and directory first. The `config open` implementation does not pre-create files; `openInEditor` is called directly (consistent with agent/role/context/task behaviour).

Categories list: `config_interactive.go` defines `allConfigCategories = []string{"agents", "roles", "contexts", "tasks"}` (plural, excludes settings). `config_open.go` must define its own five-category slice (singular or plural — consistent with the canonical values in this document) for use in the prompt.

Colour note: `tui.CategoryColor(category)` — tui/tui.go, used inside `promptSelectCategory` to colour each label. It only matches plural names ("agents", "roles", "contexts", "tasks"); singular names fall back to `ColorDim`. Using plural names in the `config_open.go` categories slice gives correct colours for four of five categories; "settings" has no colour assigned in DR-042 and always renders dim regardless.

Test infrastructure:

- `slowStdin(data string) io.Reader` — config_testhelpers_test.go:21, one-byte-per-read wrapper for sequential prompts
- `chdir(t, dir)` — start_test.go:20, sets working directory for test
- `t.Setenv("XDG_CONFIG_HOME", tmpDir)` — isolates global config path
- `t.TempDir()` — isolated temporary directory, auto-cleaned

## Goals

1. Implement `start config open [category]` command in `config_open.go`
2. Register the command in `config.go`
3. Support all five categories: agent, role, context, task, setting
4. Accept plural aliases: agents, roles, contexts, tasks, settings
5. Prompt interactively when no category is supplied
6. Write tests for all paths

## Scope

In Scope:

- `internal/cli/config_open.go` (new)
- `internal/cli/config.go` (register new command)
- Tests for `config open`

Out of Scope:

- Removing noun-group edit "no name" behaviour (that happens in p-032)
- Any changes to `config_helpers.go`
- Any other config command changes

## New Command Detail

```
start config open              # prompt: agent/role/context/task/setting?
start config open agent        # open agents.cue in $EDITOR
start config open agents       # plural alias
start config open role         # open roles.cue in $EDITOR
start config open roles        # plural alias
start config open context      # open contexts.cue in $EDITOR
start config open contexts     # plural alias
start config open task         # open tasks.cue in $EDITOR
start config open tasks        # plural alias
start config open setting      # open settings.cue in $EDITOR
start config open settings     # plural alias
```

No-argument behaviour: prompt for category (agent/role/context/task/setting).

Category normalisation: strip a trailing `s` from plural inputs to get the canonical singular before switching on the value. Canonical values: `agent`, `role`, `context`, `task`, `setting`.

If `$EDITOR` is not set, fall back to `$VISUAL`, then `vi` (existing `openInEditor` behaviour).

The `--local` flag determines which config directory is targeted. If the file does not yet exist, `openInEditor` will still open the editor — the user can create it from scratch.

## File Changes

### Create

- `internal/cli/config_open.go` — `addConfigOpenCommand`, `runConfigOpen`, `resolveConfigOpenPath`, category normalisation

### Modify

- `internal/cli/config.go` — call `addConfigOpenCommand(configCmd)`

## Success Criteria

- [x]`start config open` prompts for which file (agent/role/context/task/setting)
- [x]`start config open agent` opens `agents.cue` in `$EDITOR`
- [x]`start config open agents` works (plural alias)
- [x]`start config open role` opens `roles.cue` in `$EDITOR`
- [x]`start config open roles` works (plural alias)
- [x]`start config open context` opens `contexts.cue` in `$EDITOR`
- [x]`start config open contexts` works (plural alias)
- [x]`start config open task` opens `tasks.cue` in `$EDITOR`
- [x]`start config open tasks` works (plural alias)
- [x]`start config open setting` opens `settings.cue` in `$EDITOR`
- [x]`start config open settings` works (plural alias)
- [x]`start config open --local agent` targets `.start/agents.cue`
- [x]`start config open unknown` returns a usage error
- [x]Tests pass via `scripts/invoke-tests`

## Testing Strategy

Follow dr-024 and dr-026:

- Extract path resolution into a helper `resolveConfigOpenPath(local bool, category string) (string, error)` — this keeps `runConfigOpen` a thin wrapper and makes the path directly assertable in tests without involving the editor process (DR-026 pattern)
- Test `resolveConfigOpenPath` for each category (singular and plural input), `--local` flag effect, and unknown category error
- Test plural alias normalisation via `resolveConfigOpenPath`
- Test no-argument path triggers category prompt (use `slowStdin` from `config_testhelpers_test.go`)
- Test no-argument non-interactive (non-TTY stdin) returns an error
- For any end-to-end cobra tests that exercise `runConfigOpen`, set `t.Setenv("EDITOR", "true")` so the editor call exits immediately without opening anything

## Phases

### Phase 1 — Implementation

- Create `config_open.go` with `addConfigOpenCommand` and `runConfigOpen`
- Update `config.go` to register the new command

### Phase 2 — Tests and cleanup

- Write tests
- Run `scripts/invoke-tests` — all tests must pass

## Deliverables

Implementation files (new):

- `internal/cli/config_open.go`

Implementation files (modified):

- `internal/cli/config.go`

## Dependencies

None. This project is self-contained and can begin immediately.
