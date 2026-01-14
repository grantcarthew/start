# dr-008: Context Schema Design

- Date: 2025-12-03
- Status: Accepted
- Category: CUE

## Problem

Contexts provide environment, project, and domain-specific information to AI agents. The context schema must:

- Support dynamic content from files, commands, and templates
- Allow flexible grouping and selection based on workflow
- Distinguish between essential, standard, and specialized contexts
- Enable discovery through descriptions and tags
- Work naturally with different command types (interactive, prompt, task)

A simple required/optional boolean is insufficient. Users need flexible grouping and selection of contexts based on their current workflow.

## Decision

Contexts embed the UTD pattern for content generation and add selection behavior through `required`, `default`, and `tags` fields. Contexts are identified by their map key. A global `-c`/`--context` CLI flag selects contexts by tag.

Schema structure:

```cue
#Context: {
    #UTD                   // file, command, prompt, shell, timeout
    description?: string   // Human-readable description
    tags?: [...string]     // For grouping and selection
    required?: bool        // Always included, every command
    default?: bool         // Included in plain `start` only
}
```

Selection behavior matrix:

| Command | Required | Default | Tagged |
|---------|----------|---------|--------|
| `start` | Yes | Yes | No |
| `start prompt` | Yes | No | No |
| `start task` | Yes | No | No |
| `start --context foo` | Yes | No | If tagged `foo` |
| `start --context default,foo` | Yes | Yes | If tagged `foo` |

## Why

UTD embedding for content generation:

- Contexts need dynamic content (environment info, project state, etc.)
- UTD provides consistent pattern across tasks, roles, and contexts
- Placeholders like `{{.file_contents}}`, `{{.command_output}}` enable flexible content

Three-tier selection for flexibility:

- `required: true` = essential context, always present (environment, core docs)
- `default: true` = standard context for interactive sessions
- Tags only = specialized context, loaded when explicitly requested

Tag-based grouping:

- Users define arbitrary context groups (golang, security, frontend, etc.)
- Single mechanism for all selection needs
- Easy to combine: `-c golang,security`
- Scales to complex workflows without schema changes

Pseudo-tag `default` for composability:

- `start` is equivalent to `start --context default`
- Users can combine: `--context default,golang` (defaults plus golang-specific)
- Consistent mental model: everything is tag-based selection

Map key as identifier:

- `contexts["environment"]` - key IS the name
- No redundant name field
- Consistent with task, role, and agent patterns

## Trade-offs

Accept:

- Two boolean fields plus tags adds complexity
- Users must understand three-tier system (required/default/tagged)
- Case-insensitive matching requires normalization
- Orphan contexts (no tags, not required, not default) can never load

Gain:

- Flexible context grouping without schema changes
- Clear separation: required (essential) vs default (standard) vs tagged (specialized)
- Composable selection with multiple tags
- Works naturally with different command types
- Easy to extend (just add tags, no config structure changes)

## Alternatives

Single boolean `required` field:

- Pro: Simpler, only two states
- Pro: Clear behavior per command type
- Con: No grouping capability
- Con: Cannot select specialized contexts without including all defaults
- Rejected: Insufficient for specialized workflows

Tags only (no boolean fields):

- Pro: Single mechanism, very flexible
- Pro: Simpler schema
- Con: No clear distinction between "essential" and "groupable"
- Con: Must use reserved tag names for required/default behavior
- Rejected: Semantic distinction between required/default/tagged is valuable

Per-command context arrays in tasks:

```cue
tasks: "review": {
    contexts: ["environment", "project"]
}
```

- Pro: Fine-grained control per task
- Pro: Explicit about what each task needs
- Con: Must manage context list for every task
- Con: Easy to forget critical contexts
- Con: Inconsistent context inclusion across tasks
- Rejected: Automatic required context inclusion is simpler

## Structure

Context configuration:

```cue
contexts: [_]: #Context & {
    timeout: *30 | _
    shell:   *"/bin/bash" | _
}

contexts: {
    "environment": {
        required:    true
        description: "Local environment and user preferences"
        file:        "~/reference/ENVIRONMENT.md"
        prompt:      "Environment context:\n{{.file_contents}}"
    }

    "project": {
        default:     true
        description: "Current project documentation"
        file:        "./PROJECT.md"
        prompt:      "Project context:\n{{.file_contents}}"
    }

    "golang-conventions": {
        description: "Go language conventions"
        tags: ["golang", "programming"]
        file: "~/.config/start/contexts/golang.md"
        prompt: "Go conventions:\n{{.file_contents}}"
    }
}
```

Field definitions:

From #UTD:

- `file` (string, optional) - Path to context file
- `command` (string, optional) - Command for dynamic content
- `prompt` (string, optional) - Go template for content composition
- `shell` (string, optional) - Override default shell
- `timeout` (int, optional) - Command timeout in seconds (1-3600)

Context-specific:

- `description` (string, optional) - Human-readable summary
- `tags` ([]string, optional) - Keywords for grouping and selection (kebab-case)
- `required` (bool, optional) - Always included in all commands
- `default` (bool, optional) - Included in plain `start` only

## Tag Matching

Tag format:

- Lowercase kebab-case: `[a-z0-9]+(-[a-z0-9]+)*`
- Examples: `golang`, `code-review`, `frontend-react`

Matching behavior:

- Case-insensitive: `--context GOLANG` matches tag `golang`
- Any match: context included if ANY of its tags match ANY requested tags
- Union logic: `--context golang,security` includes contexts tagged with either

Reserved pseudo-tag:

- `default` selects contexts with `default: true`
- Not a literal tag value, interpreted specially

## Context Ordering

Injection order when multiple contexts are selected:

1. Required contexts (definition order in config)
2. Default contexts if selected (definition order in config)
3. Tagged contexts (definition order in config)

Within each category, contexts appear in the order defined in the configuration file.

## Validation

At schema load time (CUE):

- At least one of `file`, `command`, or `prompt` present (UTD requirement)
- `shell` not empty if provided
- `timeout` between 1 and 3600 if provided
- Tags match pattern `[a-z0-9]+(-[a-z0-9]+)*`
- Warn if context has no tags and is not required or default (orphan context)

At runtime (Go):

- Warn if `--context <tag>` matches no contexts
- UTD resolution errors: warn and skip
- File paths resolve correctly
- Commands execute successfully

## CLI Flag

Flag definition:

```bash
start -c <tag>[,<tag>...]
start --context <tag> [--context <tag>...]
```

Implementation using Cobra StringSliceVar:

- Supports comma-separated: `-c golang,security`
- Supports multiple flags: `-c golang -c security`
- Supports mixed: `-c golang,security -c frontend`

Flag scope:

- Defined as persistent flag on root command
- Inherited by: start (root), prompt, task
- Silently ignored by: assets, config, doctor

## Usage Examples

Required context (always included):

```cue
contexts: "environment": {
    required:    true
    description: "Local environment context"
    file:        "~/reference/ENVIRONMENT.md"
    prompt:      "{{.file_contents}}"
}
```

Default context (interactive sessions):

```cue
contexts: "project": {
    default:     true
    description: "Project documentation"
    file:        "./PROJECT.md"
    prompt:      "{{.file_contents}}"
}
```

Tagged context (on-demand):

```cue
contexts: "security-checklist": {
    description: "Security review checklist"
    tags: ["security", "review"]
    prompt: """
Security Checklist:
- Check for injection vulnerabilities
- Verify input validation
- Review authentication
"""
}
```

CLI usage:

```bash
start                              # required + default
start prompt "question"            # required only
start task code-review             # required only
start --context golang             # required + golang-tagged
start -c golang,security           # required + golang + security tagged
start --context default,golang     # required + default + golang-tagged
```

## Edge Cases

Context with both `default: true` and tags:

- Included in plain `start` (matches default)
- Included with `--context <matching-tag>` (matches tag)
- NOT included with `--context <other-tag>` (no match, --context skips defaults)

Context with `required: true` and `default: true`:

- Redundant but valid
- Required takes precedence (always included)

Orphan context (no tags, not required, not default):

- Can never be loaded
- Warn during configuration validation

No tag matches:

```bash
start --context nonexistent
# Warning: no contexts matched tag 'nonexistent'
# Loads: required contexts only
```

Lazy resolution:

- Filter contexts by selection criteria first
- Only resolve UTD for selected contexts
- Avoids unnecessary command execution
