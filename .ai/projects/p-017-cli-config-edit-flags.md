# p-017: CLI Config Edit Flags

- Status: In Progress
- Started: 2026-01-14

## Overview

Add flag-based (non-interactive) editing support to `config <type> edit` commands. Currently edit commands only support interactive prompts or $EDITOR mode. This enhancement enables scripting, automation, and quick single-field changes without TTY interaction.

Discovered during p-013 testing when test 1.7 (config agent edit with flags) failed because the feature does not exist.

## Goals

1. Add flag-based editing to `config agent edit`
2. Add flag-based editing to `config role edit`
3. Add flag-based editing to `config context edit`
4. Add flag-based editing to `config task edit`
5. Maintain consistency with existing `add` command flags

## Current State

Analysis of existing implementation in `internal/cli/`:

config_agent.go:
- `addConfigAgentEditCommand` (lines 363-377): Defines edit with `[name]` arg, no flags
- `runConfigAgentEdit` (lines 379-469): Handles no name → $EDITOR, name → interactive TTY prompts
- `addConfigAgentAddCommand` (lines 125-149): Add flags: `--name`, `--bin`, `--command`, `--default-model`, `--description`, `--model`, `--tag`

config_role.go:
- `addConfigRoleEditCommand` (lines 377-391): No flags defined
- `runConfigRoleEdit` (lines 393-526): Same pattern as agent
- `addConfigRoleAddCommand` (lines 123-149): Add flags: `--name`, `--description`, `--file`, `--command`, `--prompt`, `--tag`

config_context.go:
- `addConfigContextEditCommand` (lines 404-418): No flags defined
- `runConfigContextEdit` (lines 420-574): Same pattern, includes required/default booleans
- `addConfigContextAddCommand` (lines 120-148): Add flags: `--name`, `--description`, `--file`, `--command`, `--prompt`, `--required`, `--default`, `--tag`

config_task.go:
- `addConfigTaskEditCommand` (lines 375-389): No flags defined
- `runConfigTaskEdit` (lines 391-530): Same pattern
- `addConfigTaskAddCommand` (lines 106-133): Add flags: `--name`, `--description`, `--file`, `--command`, `--prompt`, `--role`, `--tag`

Current edit command behavior (all types):
1. No name → opens `<type>.cue` in $EDITOR
2. Name + TTY → interactive prompts with current values as defaults
3. Name + no TTY → error: "interactive editing requires a terminal"

Key Observations:
- Edit commands have NO flag definitions; flags must be added
- Existing pattern uses `cmd.Flags().Changed()` in other commands (can reuse)
- Each `run*Edit` function already loads the entity, prompts for changes, then saves
- Helper functions exist: `loadAgentsFromDir`, `writeAgentsFile`, etc.
- Shared utilities (`promptString`, `openInEditor`, `scopeString`) are in `config_agent.go` but accessible package-wide
- StringSlice flags (`--model`, `--tag`) should replace entirely when specified (matches add command behaviour)

Verified: 2026-01-14 - Code analysis confirms line numbers and structures are accurate.

## Scope

In Scope (flags matching add commands, excluding --name):
- `config agent edit <name> --bin --command --default-model --description --model --tag`
- `config role edit <name> --description --file --command --prompt --tag`
- `config context edit <name> --description --file --command --prompt --required --default --tag`
- `config task edit <name> --description --file --command --prompt --role --tag`
- Flag validation (at least one edit flag required when name provided)
- Partial updates (only specified fields are changed)

Out of Scope:
- Interactive editing (already exists)
- $EDITOR mode (already exists)
- Settings edit flags (settings has limited fields, interactive is sufficient)

## Success Criteria

- [ ] `config agent edit <name> --bin <value>` updates agent bin field
- [ ] `config role edit <name> --prompt <value>` updates role prompt
- [ ] `config context edit <name> --file <value>` updates context file
- [ ] `config task edit <name> --prompt <value>` updates task prompt
- [ ] Multiple flags can be combined in single edit
- [ ] Flags match those available on `add` commands (excluding `--name`)
- [ ] Non-TTY without flags errors (existing behavior preserved)
- [ ] TTY without flags falls back to interactive prompts (existing behavior preserved)
- [ ] p-013 test 1.7 passes after implementation

## Technical Approach

1. Add flag definitions to each edit command (mirroring add command flags, excluding --name)
2. Modify `run*Edit` function to detect if any edit flags are provided
3. If name + flags: perform non-interactive partial update (only modify fields with flags set)
4. If name + no flags + TTY: fall back to interactive prompts (existing behaviour)
5. If name + no flags + no TTY: error (existing behaviour)
6. If no name: open $EDITOR (existing behaviour)

Implementation pattern for each entity type:
```go
// In addConfig<Type>EditCommand:
editCmd.Flags().String("bin", "", "Binary executable name")
editCmd.Flags().String("command", "", "Command template")
// ... repeat for all editable fields

// In runConfig<Type>Edit, after loading entity:
hasEditFlags := cmd.Flags().Changed("bin") || cmd.Flags().Changed("command") // etc.

if hasEditFlags {
    // Non-interactive flag-based update
    if cmd.Flags().Changed("bin") {
        entity.Bin, _ = cmd.Flags().GetString("bin")
    }
    if cmd.Flags().Changed("command") {
        entity.Command, _ = cmd.Flags().GetString("command")
    }
    // ... apply other changed flags
    // Save and return (skip interactive prompts)
    return writeFile(path, entities)
}

// Existing interactive flow continues below...
```

Helper function (optional, to reduce repetition) - place in `config.go`:
```go
// anyFlagChanged returns true if any of the named flags were explicitly set
func anyFlagChanged(cmd *cobra.Command, names ...string) bool {
    for _, name := range names {
        if cmd.Flags().Changed(name) {
            return true
        }
    }
    return false
}
```

## Deliverables

- Updated `config agent edit` command with flag support
- Updated `config role edit` command with flag support
- Updated `config context edit` command with flag support
- Updated `config task edit` command with flag support
- p-013 test 1.7 verification

## Dependencies

- p-008: Configuration Editing (provides existing edit commands)

## Decisions

1. Behaviour when name provided without flags (TTY mode): B - Fall back to interactive prompts (preserves existing behaviour). Non-TTY mode continues to error as before.

2. Content source mutual exclusivity (role/context/task have file, command, prompt): B - Only update the specified field; user responsible for clearing others via $EDITOR if needed. This avoids complexity of tracking which content source is "active".

## Notes

- This is a small enhancement to improve CLI usability for scripting and automation use cases
- Boolean flags (`--required`, `--default` for context) need special handling since `--required=false` is different from not specifying the flag
- Tests should cover: single flag update, multiple flag update, flag + TTY fallback, non-TTY without flags error
