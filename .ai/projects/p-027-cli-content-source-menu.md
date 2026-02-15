# P-027: CLI Content Source Menu Extraction

- Status: Pending
- Started:
- Completed:

## Overview

Normalise and extract the content source selection menu that appears six times across the config add and edit commands for roles, contexts, and tasks. The menu prompts users to choose between file path, command, or inline prompt as the content source.

The six occurrences have cosmetic differences that are development-time inconsistencies rather than deliberate design choices. Once normalised, a single helper can replace all six.

## Goals

1. Normalise differences between add and edit versions of the content source menu
2. Extract a single `promptContentSource` helper function
3. Replace all six occurrences with calls to the helper

## Scope

In Scope:

- Content source menu in add commands: config_role.go, config_context.go, config_task.go
- Content source menu in edit commands: same three files
- Normalising label text, reader creation, and default prompt handling

Out of Scope:

- Changes to the menu options themselves (file/command/inline)
- Changes to the overall add/edit command flow

## Current State

Six occurrences of the same interactive menu:
- 3 in add commands (config_role.go:196, config_context.go:210, config_task.go:203)
- 3 in edit commands (config_role.go:532, config_context.go:561, config_task.go:535)

Development-time differences between add and edit versions:
- Label: "Content source (choose one):" vs "New content source:" -- no functional reason to differ
- Reader: add creates new `bufio.NewReader(stdin)`, edit reuses existing -- other helpers like `promptString` create their own reader
- Default prompt: add passes `""` to promptText, edit passes current value -- both are just a `defaultPrompt` parameter

Default choice varies by entity type:
- Roles and contexts default to "1" (file path)
- Tasks default to "3" (inline prompt)

## Success Criteria

- [ ] Single `promptContentSource` helper in config_helpers.go
- [ ] All six menu occurrences replaced with helper calls
- [ ] Behaviour unchanged (same defaults, same prompts, same validation)
- [ ] All existing tests pass

## Deliverables

- Updated `internal/cli/config_helpers.go` - new promptContentSource helper
- Updated `internal/cli/config_role.go` - add and edit commands use helper
- Updated `internal/cli/config_context.go` - add and edit commands use helper
- Updated `internal/cli/config_task.go` - add and edit commands use helper

## Technical Approach

Helper signature:

```
func promptContentSource(w io.Writer, r io.Reader, defaultChoice, currentPrompt string) (file, command, prompt string, err error)
```

Callers:
- Add (role/context): `file, command, prompt, err = promptContentSource(stdout, stdin, "1", "")`
- Add (task): `file, command, prompt, err = promptContentSource(stdout, stdin, "3", "")`
- Edit (role/context): `newFile, newCommand, newPrompt, err = promptContentSource(stdout, stdin, "1", role.Prompt)`
- Edit (task): `newFile, newCommand, newPrompt, err = promptContentSource(stdout, stdin, "3", task.Prompt)`
