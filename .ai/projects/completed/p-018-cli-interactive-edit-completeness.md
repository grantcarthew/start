# p-018: CLI Interactive Edit Completeness

- Status: Complete
- Started: 2025-01-19
- Completed: 2025-01-19

## Overview

Enhance interactive edit mode for `config <type> edit <name>` commands to support editing all fields. Currently, interactive mode omits some fields (models, tags) that are available via add commands, creating an inconsistent user experience.

Discovered during p-017 review when analysing the gap between add command flags and interactive edit prompts.

## Goals

1. Enable editing all agent fields interactively (including models and tags)
2. Enable editing tags for roles, contexts, and tasks interactively
3. Maintain consistent UX across all entity types

## Current State

**Verified 2025-01-19**: All line numbers and code references confirmed accurate.

All config edit commands are in `internal/cli/`:

### Agent (`config_agent.go`)
- `runConfigAgentEdit` function: lines 393-528
- Interactive prompts section: lines 486-527
- Current prompts: bin, command, default_model, description
- **Missing**: models (map), tags (slice)
- `AgentConfig` struct (line 698): includes `Models map[string]string` and `Tags []string`
- `writeAgentsFile` (line 839): already handles models and tags serialisation

### Role (`config_role.go`)
- `runConfigRoleEdit` function: lines 409-575
- Interactive prompts section: lines 490-574
- Current prompts: description, content source (file/command/prompt)
- **Missing**: tags
- `RoleConfig` struct (line 745): includes `Tags []string`
- `writeRolesFile` (line 869): already handles tags serialisation

### Context (`config_context.go`)
- `runConfigContextEdit` function: lines 439-632
- Interactive prompts section: lines 526-630
- Current prompts: description, content source, required, default
- **Missing**: tags
- `ContextConfig` struct (line 722): includes `Tags []string`
- `writeContextsFile` (line 854): already handles tags serialisation

### Task (`config_task.go`)
- `runConfigTaskEdit` function: lines 409-584
- Interactive prompts section: lines 493-583
- Current prompts: description, content source, role
- **Missing**: tags
- `TaskConfig` struct (line 674): includes `Tags []string`
- `writeTasksFile` (line 802): already handles tags serialisation

### Existing Prompt Utility
- `promptString` function (`config_agent.go:976-994`): prompts for string with default value
- Pattern: `Label [default]: ` format, Enter keeps current value
- No existing utility for slice (tags) or map (models) fields - will need new prompt helpers

### Existing Tests
- `config_test.go`: 1097 lines of tests covering list, add, remove, info, default, and write functions
- `config_integration_test.go`: Full workflow tests for add/list/info/remove operations
- Tests use `t.TempDir()`, `t.Setenv()`, and Cobra command execution pattern
- No existing tests for interactive edit flows - new tests needed for this project

### Testing Interactive Input
Per dr-024, interactive input is tested by injecting stdin via Cobra's `SetIn()`:
```go
cmd.SetIn(strings.NewReader("new-value\n\nkeep-default\n"))
```

The interactive edit flow requires TTY detection which uses `term.IsTerminal()`. For tests:
- Flag-based edit tests work without TTY (already covered by p-017)
- Interactive edit tests would need to either:
  - Mock the TTY check by extracting it to a testable function
  - Test the prompt helper functions directly in isolation
  - Use integration tests with simulated input where TTY check is bypassed

## Scope

In Scope:
- Add models editing to agent interactive edit
- Add tags editing to all entity types (agent, role, context, task)
- Provide clear prompts for slice/map fields

Out of Scope:
- Flag-based editing (covered by p-017)
- $EDITOR mode (already supports all fields via raw CUE editing)
- Adding new fields to entities

## Success Criteria

- [x] `config agent edit <name>` prompts for models (add/remove/keep)
- [x] `config agent edit <name>` prompts for tags (add/remove/keep)
- [x] `config role edit <name>` prompts for tags
- [x] `config context edit <name>` prompts for tags
- [x] `config task edit <name>` prompts for tags
- [x] All prompts show current values and allow keeping them

## Technical Approach

For slice fields (tags):
- Show current tags: `Current tags: [test, example]`
- Prompt: `Tags (comma-separated, empty to clear, Enter to keep): `
- Parse comma-separated input, trim whitespace

For map fields (models):
- Show current models: `Current models: fast=gpt-4-turbo, smart=gpt-4`
- Options: `Models: (k)eep, (c)lear, (e)dit: `
- If edit (modify mode):
  - Show each existing model with current value as default: `fast [gpt-4-turbo]: `
  - Enter keeps current, `-` deletes the model, new value replaces
  - After existing models, prompt: `Add more? (alias=model-id, empty to finish): `

## Deliverables

- Updated `runConfigAgentEdit` with models and tags prompts
- Updated `runConfigRoleEdit` with tags prompt
- Updated `runConfigContextEdit` with tags prompt
- Updated `runConfigTaskEdit` with tags prompt
- New prompt helper functions for tags (slice) and models (map) editing
- Tests for interactive edit flows (following dr-024 testing strategy)

## Dependencies

- p-008: Configuration Editing (provides existing edit commands)

## Decisions

1. **Testing approach for interactive edit flows**: Extract prompt logic into testable helper functions.

   The new `promptTags` and `promptModels` helpers will follow the existing `promptString` pattern, accepting `io.Writer` and `io.Reader` parameters. This allows thorough unit testing without TTY concerns. The edit function integration (TTY check → call helpers → write) remains thin glue code tested manually.

   ```go
   func promptTags(w io.Writer, r io.Reader, current []string) ([]string, error)
   func promptModels(w io.Writer, r io.Reader, current map[string]string) (map[string]string, error)
   ```

2. **Models edit behaviour**: Modify mode.

   When user selects (e)dit for models, show each existing model for editing (Enter to keep current value), then prompt to add new models. To delete a model, user enters `-` or similar sentinel value.

   ```
   fast [gpt-4-turbo]:
   smart [gpt-4]: gpt-4o
   Add more? (alias=model-id, empty to finish):
   > reasoning=o1
   >
   ```

## Notes

This is a small UX enhancement to make interactive mode feature-complete relative to add commands.

## Completion Summary

Implemented two new prompt helper functions:

- `promptTags(w io.Writer, r io.Reader, current []string) ([]string, error)`: Shows current tags, accepts comma-separated input to replace, "-" to clear, or Enter to keep current.
- `promptModels(w io.Writer, r io.Reader, current map[string]string) (map[string]string, error)`: Offers (k)eep, (c)lear, (e)dit options. Edit mode allows modifying existing models (Enter to keep, "-" to delete) and adding new ones in alias=model-id format.

Updated all four edit functions to include prompts for missing fields:
- `runConfigAgentEdit`: Added models and tags prompts
- `runConfigRoleEdit`: Added tags prompt
- `runConfigContextEdit`: Added tags prompt
- `runConfigTaskEdit`: Added tags prompt

Added 14 unit tests for the prompt helper functions covering all scenarios:
- Keep current, clear, replace for both tags and models
- Edit mode: keep, update, delete existing models; add new models
- Error handling for invalid input

All tests pass.
