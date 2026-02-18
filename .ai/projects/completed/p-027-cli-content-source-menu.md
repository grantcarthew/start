# P-027: CLI Content Source Menu Extraction

- Status: Complete
- Started: 2026-02-18
- Completed: 2026-02-18

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
- 3 in add commands (config_role.go:192, config_context.go:206, config_task.go:192)
- 3 in edit commands (config_role.go:515, config_context.go:560, config_task.go:527)

Development-time differences between add and edit versions:
- Label: "Content source (choose one):" vs "New content source:" -- no functional reason to differ
- Reader: add creates new `bufio.NewReader(stdin)`, edit reuses existing -- other helpers like `promptString` create their own reader
- Default prompt: add passes `""` to promptText, edit passes current value -- both are just a `currentPrompt` parameter

Additional inconsistencies across entity types:
- Context add/edit uses `promptString` for option 3, while role/task uses `promptText` -- context inline input is single-line only, missing multi-line and editor support
- Context option 3 label says "Inline content" (sub-prompt "Content text"), role/task says "Inline prompt" (sub-prompt "Prompt text") -- the CUE field is `prompt` for all three
- Context edit option 3 passes `""` to `promptString`, losing current value; role/task edit pass current prompt to `promptText`

Default choice varies by entity type:
- Roles and contexts default to "1" (file path)
- Tasks default to "3" (inline prompt)

Test coverage: existing integration tests exercise flag-based (non-interactive) add only. The interactive content source menu has zero test coverage. Consider adding a unit test for the new helper.

## Success Criteria

- [x] Single `promptContentSource` helper in config_helpers.go
- [x] All six menu occurrences replaced with helper calls
- [x] Behaviour unchanged (same defaults, same prompts, same validation)
- [x] All existing tests pass

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

The helper normalises all six occurrences to use:
- Label: "Content source (choose one):" with coloured delimiters (add style)
- Option 3 label: "Inline prompt" (consistent with CUE field name)
- Input method: `promptText` for option 3 (multi-line with editor fallback)
- Sub-prompt: "Prompt text" for option 3
- `currentPrompt` passed through to `promptText` default value

Callers:
- Add (role/context): `file, command, prompt, err = promptContentSource(stdout, stdin, "1", "")`
- Add (task): `file, command, prompt, err = promptContentSource(stdout, stdin, "3", "")`
- Edit (role/context): `newFile, newCommand, newPrompt, err = promptContentSource(stdout, stdin, "1", role.Prompt)`
- Edit (task): `newFile, newCommand, newPrompt, err = promptContentSource(stdout, stdin, "3", task.Prompt)`
