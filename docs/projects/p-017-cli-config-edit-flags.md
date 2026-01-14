# P-017: CLI Config Edit Flags

- Status: Proposed
- Started: -

## Overview

Add flag-based (non-interactive) editing support to `config <type> edit` commands. Currently edit commands only support interactive prompts or $EDITOR mode. This enhancement enables scripting, automation, and quick single-field changes without TTY interaction.

Discovered during P-013 testing when test 1.7 (config agent edit with flags) failed because the feature does not exist.

## Goals

1. Add flag-based editing to `config agent edit`
2. Add flag-based editing to `config role edit`
3. Add flag-based editing to `config context edit`
4. Add flag-based editing to `config task edit`
5. Maintain consistency with existing `add` command flags

## Scope

In Scope:
- `config agent edit <name> --bin --command --default-model --description --models`
- `config role edit <name> --content --file --description`
- `config context edit <name> --file --command --required --default --tags --description`
- `config task edit <name> --prompt --file --command --role --contexts --description`
- Flag validation (at least one edit flag required when name provided)
- Partial updates (only specified fields are changed)

Out of Scope:
- Interactive editing (already exists)
- $EDITOR mode (already exists)
- Settings edit flags (settings has limited fields, interactive is sufficient)

## Success Criteria

- [ ] `config agent edit <name> --bin <value>` updates agent bin field
- [ ] `config role edit <name> --content <value>` updates role content
- [ ] `config context edit <name> --file <value>` updates context file
- [ ] `config task edit <name> --prompt <value>` updates task prompt
- [ ] Multiple flags can be combined in single edit
- [ ] Flags match those available on `add` commands
- [ ] Error when name provided but no edit flags given
- [ ] P-013 test 1.7 passes after implementation

## Technical Approach

1. For each entity type, the edit command already has flag definitions from add command reuse
2. Modify RunE function to detect if flags are provided
3. If name + flags: perform non-interactive update
4. If name only: fall back to interactive prompts (existing behaviour)
5. If no name: open $EDITOR (existing behaviour)

## Deliverables

- Updated `config agent edit` command with flag support
- Updated `config role edit` command with flag support
- Updated `config context edit` command with flag support
- Updated `config task edit` command with flag support
- P-013 test 1.7 verification

## Dependencies

- P-008: Configuration Editing (provides existing edit commands)

## Notes

This is a small enhancement to improve CLI usability for scripting and automation use cases.
