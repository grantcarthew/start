# dr-010: Role Schema Design

- Date: 2025-12-03
- Status: Accepted
- Category: CUE

## Problem

Roles define AI agent behavior through system prompts. The role schema must:

- Support static file-based prompts for simple cases
- Support dynamic content generation for context-aware prompts
- Enable reuse across multiple tasks
- Integrate with asset distribution for sharing
- Provide consistent behavior across different AI agents

## Decision

Roles embed the UTD pattern for system prompt generation and add metadata for discovery. Roles are identified by their map key.

Schema structure:

```cue
#Role: {
    #UTD                   // file, command, prompt, shell, timeout
    description?: string   // Human-readable description
    tags?: [...string]     // For categorization and search
}
```

## Why

UTD embedding for prompt generation:

- Simple roles: just a file reference
- Complex roles: dynamic content via commands and templates
- Environment-aware roles: inject runtime context (Go version, git status, etc.)
- Single pattern supports all use cases

Roles as named entities:

- Define once, reference from multiple tasks
- Tasks use `role: "code-reviewer"` instead of embedding prompt inline
- Easy to swap roles for experimentation
- Override via `--role` CLI flag

Map key as identifier:

- `roles["code-reviewer"]` - key IS the name
- No redundant name field
- Consistent with task and context patterns

Minimal schema:

- Only UTD fields and metadata
- No role-specific behavioral fields
- Roles define "what the AI is", not "how it behaves at runtime"

## Trade-offs

Accept:

- Roles are simple containers for system prompts
- No role-specific validation beyond UTD
- Complex behaviors belong in task definitions

Gain:

- Maximum flexibility in prompt definition
- Consistent pattern with tasks and contexts
- Easy to create and share roles
- Simple mental model: role = system prompt

## Structure

Role configuration:

```cue
roles: [_]: #Role & {
    timeout: *60 | _
    shell:   *"/bin/bash" | _
}

roles: {
    "code-reviewer": {
        description: "Expert code reviewer"
        file:        "~/.config/start/roles/reviewer.md"
        prompt: """
{{.file_contents}}

Additional focus:
- Security implications
- Performance considerations
- Error handling
"""
        tags: ["review", "code", "quality"]
    }
}
```

Field definitions:

From #UTD:

- `file` (string, optional) - Path to system prompt file
- `command` (string, optional) - Command for dynamic content
- `prompt` (string, optional) - Go template for prompt composition
- `shell` (string, optional) - Override default shell
- `timeout` (int, optional) - Command timeout in seconds (1-3600)

Role-specific:

- `description` (string, optional) - Human-readable summary
- `tags` ([]string, optional) - Keywords for search and categorization

## Usage Examples

Simple file-based role:

```cue
roles: "general-assistant": {
    description: "General purpose AI assistant"
    file:        "~/.config/start/roles/general.md"
}
```

Inline prompt role:

```cue
roles: "documentation-writer": {
    description: "Technical documentation specialist"
    prompt: """
You are a technical documentation specialist.

Guidelines:
- Use clear, concise language
- Include code examples where helpful
- Focus on user needs
"""
    tags: ["documentation", "writing"]
}
```

Dynamic role with environment context:

```cue
roles: "go-expert": {
    description: "Go language expert with environment awareness"
    file:        "~/.config/start/roles/go-base.md"
    command:     "go version 2>/dev/null || echo 'Go not installed'"
    prompt: """
{{.file_contents}}

Environment: {{.command_output}}

Apply Go-specific best practices and idioms.
"""
    tags: ["golang", "programming"]
}
```

Role selection precedence:

1. CLI `--role` flag (highest priority)
2. Task `role` field
3. Default role from settings
4. First role in configuration

## Agent Integration

Roles are provided to agents through placeholders:

- `{{.role}}` - Inline content for agents accepting system prompts directly
- `{{.role_file}}` - File path for agents requiring file-based prompts

Agent command examples:

```cue
agents: "claude": {
    command: "{{.bin}} --model {{.model}} --append-system-prompt {{.role}} {{.prompt}}"
}

agents: "gemini": {
    command: "GEMINI_SYSTEM_MD={{.role_file}} {{.bin}} --model {{.model}} {{.prompt}}"
}
```
