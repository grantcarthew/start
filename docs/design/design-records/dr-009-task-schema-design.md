# DR-009: Task Schema Design

- Date: 2025-12-03
- Status: Accepted
- Category: CUE

## Problem

Tasks are reusable workflows that combine AI agent behavior with dynamic content generation. The task schema must:

- Support dynamic content from files, commands, and templates
- Reference roles and agents without duplicating their definitions
- Enable discovery through descriptions and tags
- Integrate with the asset distribution system
- Validate at both schema load time and runtime

## Decision

Tasks embed the UTD pattern for content generation and add references to roles and agents by name. Tasks are identified by their map key, eliminating the need for a separate name field.

Schema structure:

```cue
#Task: {
    #UTD                      // file, command, prompt, shell, timeout
    description?: string      // Human-readable description
    tags?: [...string]        // For categorization and search
    role?:  string | #Role    // String reference OR imported role
    agent?: string            // Reference to agent by name
}
```

## Why

UTD embedding for content generation:

- Tasks need dynamic content (file contents, command output, templates)
- UTD provides consistent pattern across tasks, roles, and contexts
- Single mechanism for all content generation needs
- Placeholders like `{{.instructions}}`, `{{.command_output}}` enable flexible prompts

References instead of inline definitions:

- Roles and agents are defined once, referenced many times
- Avoids duplication and inconsistency
- Easy to swap roles or agents without editing task definitions
- Runtime validation ensures references resolve correctly

Map key as identifier:

- `tasks["code-review"]` - key IS the name
- No redundant name field
- Cleaner schema, less duplication
- Pattern constraint `[Name=_]: {name: Name}` available if name injection needed

Tags for discovery:

- Enable search across task catalog
- Support categorization (golang, security, git, etc.)
- Consistent with role and context schemas

## Trade-offs

Accept:

- Runtime validation needed for role/agent references (CUE cannot validate cross-references)
- Users must ensure referenced roles and agents exist
- No compile-time checking of reference validity

Gain:

- Clean separation between task definition and role/agent configuration
- Reusable roles and agents across many tasks
- Flexible content generation via UTD
- Simple schema with minimal required fields
- Easy to extend with new optional fields

## Structure

Task configuration:

```cue
tasks: [_]: #Task & {
    timeout: *120 | _
    shell:   *"/bin/bash" | _
}

tasks: {
    "code-review": {
        description: "Review code changes before committing"
        role:        "code-reviewer"
        agent:       "claude"
        command:     "git diff --staged"
        prompt: """
Review these code changes:

{{.command_output}}

Focus on: {{.instructions}}
"""
        tags: ["review", "git", "quality"]
    }
}
```

Field definitions:

From #UTD:
- `file` (string, optional) - Path to file for content
- `command` (string, optional) - Shell command to execute
- `prompt` (string, optional) - Go template with placeholders
- `shell` (string, optional) - Override default shell
- `timeout` (int, optional) - Command timeout in seconds (1-3600)

Task-specific:
- `description` (string, optional) - Human-readable summary
- `tags` ([]string, optional) - Keywords for search and categorization
- `role` (string | #Role, optional) - String name for runtime resolution, or imported #Role for CUE dependency
- `agent` (string, optional) - Name of agent to use

## Validation

At schema load time (CUE):

- At least one of `file`, `command`, or `prompt` present (UTD requirement)
- `shell` not empty if provided
- `timeout` between 1 and 3600 if provided

At runtime (Go):

- If `role` is string, referenced role exists in configuration
- If `role` is #Role struct, extract content directly
- Referenced `agent` exists in configuration
- File paths resolve correctly
- Commands execute successfully

## Usage Examples

Task with all fields:

```cue
tasks: "security-audit": {
    description: "Security-focused code audit"
    role:        "security-auditor"
    agent:       "claude"
    file:        "./SECURITY.md"
    command:     "git diff --staged"
    prompt: """
Security guidelines:
{{.file_contents}}

Changes to review:
{{.command_output}}

Audit focus: {{.instructions}}
"""
    timeout: 60
    tags: ["security", "audit", "review"]
}
```

Minimal task:

```cue
tasks: "quick-check": {
    command: "go vet ./..."
    prompt:  "Check results: {{.command_output}}"
}
```

Task with role override at runtime:

```bash
start task security-audit --role go-expert
```

Published task with imported role (CUE dependency):

```cue
import (
    "github.com/grantcarthew/start-assets/schemas@v0"
    agentRole "github.com/grantcarthew/start-assets/roles/golang/agent@v0:agent"
)

task: schemas.#Task & {
    description: "Go code review"
    role: agentRole.role    // Creates CUE module dependency
    file: "@module/task.md"
    prompt: """
        {{.file_contents}}

        ## Custom Instructions

        {{.instructions}}
        """
}
```

## Updates

- 2025-12-09: Changed `role` field from `string` to `string | #Role` to support CUE dependencies (see DR-022)
