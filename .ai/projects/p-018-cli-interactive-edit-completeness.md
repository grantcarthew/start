# p-018: CLI Interactive Edit Completeness

- Status: Proposed
- Started: -

## Overview

Enhance interactive edit mode for `config <type> edit <name>` commands to support editing all fields. Currently, interactive mode omits some fields (models, tags) that are available via add commands, creating an inconsistent user experience.

Discovered during p-017 review when analysing the gap between add command flags and interactive edit prompts.

## Goals

1. Enable editing all agent fields interactively (including models and tags)
2. Enable editing tags for roles, contexts, and tasks interactively
3. Maintain consistent UX across all entity types

## Current State

Analysis of interactive edit prompts vs add command fields:

Agent (`config_agent.go:426-468`):
- Interactive edit prompts: bin, command, default_model, description
- Missing from interactive: models (map), tags (slice)

Role (`config_role.go:440-526`):
- Interactive edit prompts: description, file/command/prompt (content source)
- Missing from interactive: tags

Context (`config_context.go:467-573`):
- Interactive edit prompts: description, file/command/prompt, required, default
- Missing from interactive: tags

Task (`config_task.go:438-529`):
- Interactive edit prompts: description, file/command/prompt, role
- Missing from interactive: tags

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

- [ ] `config agent edit <name>` prompts for models (add/remove/keep)
- [ ] `config agent edit <name>` prompts for tags (add/remove/keep)
- [ ] `config role edit <name>` prompts for tags
- [ ] `config context edit <name>` prompts for tags
- [ ] `config task edit <name>` prompts for tags
- [ ] All prompts show current values and allow keeping them

## Technical Approach

For slice fields (tags):
- Show current tags: `Current tags: [test, example]`
- Prompt: `Tags (comma-separated, empty to clear, Enter to keep): `
- Parse comma-separated input, trim whitespace

For map fields (models):
- Show current models: `Current models: fast=gpt-4-turbo, smart=gpt-4`
- Options: `Models: (k)eep, (c)lear, (e)dit: `
- If edit: prompt for each model alias=value, empty line to finish

## Deliverables

- Updated `runConfigAgentEdit` with models and tags prompts
- Updated `runConfigRoleEdit` with tags prompt
- Updated `runConfigContextEdit` with tags prompt
- Updated `runConfigTaskEdit` with tags prompt

## Dependencies

- p-008: Configuration Editing (provides existing edit commands)

## Notes

This is a small UX enhancement to make interactive mode feature-complete relative to add commands.
