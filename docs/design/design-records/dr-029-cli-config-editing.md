# DR-029: CLI Configuration Editing Commands

- Date: 2025-12-18
- Status: Accepted
- Category: CLI

## Problem

Users need to manage configuration for agents, roles, contexts, and tasks without manually editing CUE files. The CLI should provide commands for creating, viewing, modifying, and removing configuration items while:

- Supporting both global (`~/.config/start/`) and local (`./.start/`) configurations
- Validating input against schema requirements
- Working with CUE's file-based structure
- Enabling both interactive and scripted usage

## Decision

Implement configuration editing commands following the pattern `start config <entity> <action>` where entity is one of `agent`, `role`, `context`, or `task`.

Command structure:

```
start config agent list|add|show|edit|remove|default
start config role list|add|show|edit|remove|default
start config context list|add|show|edit|remove
start config task list|add|show|edit|remove
```

File editing strategy:

- One top-level key per file (`agents.cue` contains only `agents: {...}`)
- Template-based generation to write files (regenerate entire file on modification)
- Read and write individual files directly, not the unified merged config
- Global and local directories are edited independently

Input handling:

- Hybrid approach: flags for any field, interactive prompts for missing required fields
- Fully scriptable when all required flags provided
- Interactive when run without flags

## Why

Noun-verb command structure (`config agent list` not `config list agent`):

- Matches existing CLI patterns in start (`start assets add`, `start assets list`)
- Better discoverability: users explore "what can I do with agents?"
- Each entity becomes a Cobra command group with subcommands
- Consistent with prototype design

Template-based file generation:

- CUE files in a directory are unified by CUE before we access them
- Editing requires working with individual files, not the merged result
- `generateAgentCUE` pattern already established in auto-setup
- Simpler than AST manipulation, guaranteed valid CUE output

Hybrid input approach:

- Scriptable for automation (`start config agent add --bin claude --command "..."`)
- Interactive for exploration (prompts guide users through required fields)
- Matches common CLI patterns (e.g., `gh pr create`)

No backups:

- Config directories are often version-controlled via dotfiles repos
- Keeps implementation simple
- Can be added later if users request it

No editor integration for individual fields:

- `start config <type> edit` (no name) opens entire file in $EDITOR
- `start config <type> edit <name>` uses interactive prompts
- Roles and tasks typically use `file:` field for long content

Validate on every write:

- Immediate feedback catches errors early
- Template generation should produce valid CUE, validation is safety net
- CUE validation is fast

## Trade-offs

Accept:

- Regenerating entire files loses any user comments or custom formatting
- Users who want advanced CUE features must edit files directly
- No backup protection (relies on user's version control)
- More commands to implement (6 actions x 4 entity types)

Gain:

- Consistent, predictable file structure
- Safe editing with validation
- Works for both interactive and scripted use
- Clear separation: CLI-managed files vs user-managed files
- No complex AST manipulation required

## Alternatives

AST manipulation for CUE files:

- Pro: Preserves comments and formatting
- Pro: Surgical edits to specific items
- Con: CUE's AST API is less mature than Go's
- Con: Complex edge cases with CUE expressions and imports
- Rejected: Complexity outweighs benefits for config editing

Verb-noun command structure (`config list agent`):

- Pro: Groups actions (what can I list?)
- Pro: Pattern used by kubectl, aws
- Con: Doesn't match existing start patterns
- Con: Less intuitive for small number of entity types
- Rejected: Noun-verb matches existing CLI and prototype

Interactive prompts only (no flags):

- Pro: Simpler implementation
- Con: Cannot script or automate
- Rejected: Scripting is essential for power users

Flags only (no prompts):

- Pro: Fully scriptable
- Con: Verbose for interactive use
- Con: Users must know all required fields
- Rejected: Interactive guidance valuable for discoverability

## Structure

### Actions

| Action | Description |
|--------|-------------|
| `list` | Display all items with summary (name, description, source) |
| `add` | Create new item via prompts/flags, write to file |
| `show <name>` | Display full details of single item |
| `edit` | No name: open file in $EDITOR. With name: interactive prompts |
| `remove <name>` | Delete item with confirmation |
| `default <name>` | Set/show default (agent and role only) |

### Scope Handling

The `--local` flag targets local configuration (`./.start/`) instead of global (`~/.config/start/`).

| Action | Default Behaviour | With --local |
|--------|-------------------|--------------|
| `list` | Shows merged view, indicates source | Shows local only |
| `add` | Prompts for scope | Writes to local |
| `show` | Shows from merged config | Shows local only |
| `edit` | Prompts if item in both | Edits local |
| `remove` | Prompts if item in both | Removes from local |
| `default` | Writes to global settings.cue | Writes to local settings.cue |

### File Organisation

Each file contains one top-level key:

| File | Content |
|------|---------|
| `agents.cue` | `agents: { ... }` |
| `roles.cue` | `roles: { ... }` |
| `contexts.cue` | `contexts: { ... }` |
| `tasks.cue` | `tasks: { ... }` |
| `settings.cue` | `settings: { ... }` |

### Generation Pattern

Files are generated using template functions (following `generateAgentCUE` pattern):

```
// Auto-generated by start config
// Edit this file to customize your configuration

agents: {
    "claude": {
        bin:     "claude"
        command: "claude --model {{.model}} {{.prompt}}"
        // ... other fields
    }
}
```

## Validation

After every write operation:

- Parse generated CUE to verify syntax
- Check required fields present
- Validate field values match schema constraints
- Report errors immediately to user

## Usage Examples

Add agent interactively:

```bash
start config agent add
# Prompts for: name, bin, command, models, default_model, description
```

Add agent via flags:

```bash
start config agent add \
  --name gemini \
  --bin gemini \
  --command '{{.bin}} --model {{.model}} {{.prompt}}' \
  --default-model flash
```

List all agents:

```bash
start config agent list
# Shows: name, description, source (global/local)
```

Edit agent interactively:

```bash
start config agent edit claude
# Prompts show current values, enter to keep, type to change
```

Edit agents file directly:

```bash
start config agent edit
# Opens ~/.config/start/agents.cue in $EDITOR
```

Remove agent:

```bash
start config agent remove gemini
# Confirms, then removes from agents.cue
```

Set default agent:

```bash
start config agent default claude
# Writes default_agent: "claude" to settings.cue
```

Add to local config:

```bash
start config context add --local --name project --file PROJECT.md
# Writes to ./.start/contexts.cue
```
