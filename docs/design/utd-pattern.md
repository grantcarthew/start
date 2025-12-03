# Unified Template Design (UTD)

A consistent pattern for building prompt text across `start` configuration.

## Overview

The Unified Template Design (UTD) provides a flexible way to build prompt text from static files, dynamic command output, and template text. It's the foundation for composing rich, context-aware prompts for AI agents.

**Used by:**

- Tasks - Build task prompt text
- Roles - Build role/system prompt text
- Contexts - Build context document text

**Not used by:**

- Agents - Use different structure (command execution templates)

## Core Concept

UTD uses three optional fields that work together:

**Fields:**

- `file` - Path to a file
- `command` - Shell command to execute
- `prompt` - Template text

**At least one field must be present.** The Go runtime validates this constraint.

**Resolution priority:** `prompt` > `file` > `command`

- If `prompt` is present, it wins (other fields are inputs)
- If only `file` and `command`, file wins (command can be injected into file)
- If only one field, that field determines the output

## Placeholders

All placeholders use Go template syntax: `{{.name}}`

**UTD Placeholders (available everywhere):**

- `{{.file}}` - File path (e.g., `/Users/grant/PROJECT.md`)
- `{{.file_contents}}` - File contents (actual text)
- `{{.command}}` - Command string (e.g., `git status --short`)
- `{{.command_output}}` - Command execution output (stdout + stderr)
- `{{.date}}` - Current timestamp (ISO 8601 format)

**Task-Specific Placeholder:**

- `{{.instructions}}` - Additional instructions from command line argument

**Go Template Features:**

- Full Go template support: conditionals (`{{if}}`), loops (`{{range}}`), functions
- See [Go template documentation](https://pkg.go.dev/text/template) for complete syntax

## Resolution Flow

The UTD resolver builds final prompt text based on which fields are present and which placeholders are used.

### Lazy Evaluation

**Optimization:** Only perform I/O when necessary:

- File is read only if `{{.file_contents}}` appears in template
- Command is executed only if `{{.command_output}}` appears in template
- `{{.file}}` and `{{.command}}` are just strings (no I/O)

### Resolution Rules

**1. Only `file`**

```cue
file: "./ROLE.md"
```

Result: File contents become the prompt text.

---

**2. Only `command`**

```cue
command: "git status --short"
```

Result: Command output becomes the prompt text.

---

**3. Only `prompt`**

```cue
prompt: "Review this code for security issues."
```

Result: Prompt text (with runtime placeholders like `{{.date}}` resolved).

---

**4. `file` + `command` (no `prompt`)**

```cue
file:    "./PROJECT.md"
command: "git log -5 --oneline"
```

If file contains `{{.command}}` or `{{.command_output}}`:

- Execute command
- Template file contents with command data → prompt text

Otherwise:

- Warn: "command defined but not used in file"
- Use file contents → prompt text

---

**5. `file` + `prompt`**

```cue
file:   "~/reference/ENVIRONMENT.md"
prompt: "Read {{.file}} for environment context."
```

If prompt contains `{{.file}}` or `{{.file_contents}}`:

- Read file
- Template prompt → prompt text

Otherwise:

- Warn: "file defined but not used in prompt"
- Template prompt → prompt text

---

**6. `command` + `prompt`**

```cue
command: "git status --short"
prompt:  "Current status:\n{{.command_output}}"
```

If prompt contains `{{.command}}` or `{{.command_output}}`:

- Execute command
- Template prompt → prompt text

Otherwise:

- Warn: "command defined but not used in prompt"
- Template prompt → prompt text

---

**7. `file` + `command` + `prompt`**

```cue
file:    "./PROJECT.md"
command: "git status --short"
prompt: """
Project: {{.file_contents}}

Status: {{.command_output}}
"""
```

- Read file if `{{.file}}` or `{{.file_contents}}` used (else warn)
- Execute command if `{{.command}}` or `{{.command_output}}` used (else warn)
- Template prompt → prompt text

## Shell Configuration

### Global Settings

Define defaults in `config.cue`:

```cue
settings: {
	shell:   "bash"  // Default: auto-detect (bash > sh)
	timeout: 30      // Default: 30 seconds
}
```

### Per-Section Override

Override shell for specific UTD instances:

```cue
contexts: {
	"node-version": {
		command: "console.log(process.version)"
		shell:   "node"
		timeout: 5
		prompt:  "Node version: {{.command_output}}"
	}
}
```

**Priority:**

1. Section-specific `shell` field (if present)
2. Global `settings.shell` (if configured)
3. Auto-detected shell (`bash` if available, otherwise `sh`)

### Supported Shells

Common shells are automatically supported with appropriate flags:

- **Shells:** bash, sh, zsh, fish
- **JavaScript:** node, nodejs, bun, deno
- **Python:** python, python2, python3
- **Other:** ruby, perl

Unknown shells default to `-c` flag.

### Command Timeout

Commands are subject to timeout limits (default 30 seconds):

```cue
contexts: {
	"quick-check": {
		command: "git status"
		timeout: 5   // 5 seconds
	}

	"slow-analysis": {
		command: "npm run analyze"
		timeout: 120  // 2 minutes
	}
}
```

**Behavior:**

- Command exceeds timeout → Killed, warning emitted
- `{{.command_output}}` = output captured up to timeout point (may be empty)

### Working Directory

Commands execute in:

- Current working directory (default)
- Directory specified by `--directory` flag (if supported)

## Error Handling

Error handling depends on **where UTD is used**:

**Tasks:**

- File missing or command fails → **Fail** (task cannot proceed)
- Template syntax error → **Fail** (invalid configuration)

**Contexts:**

- File missing or command fails → **Warn + skip** entire context
- Template syntax error → **Warn + skip** entire context
- Session continues without this context

**Roles:**

- File missing or command fails → **Warn + skip** role
- Template syntax error → **Warn + skip** role
- Agent runs without role/system prompt

**General principle:** Warn, skip, continue (fail only when critical).

## Examples

### Simple File

```cue
roles: {
	"code-reviewer": {
		file: "./ROLE.md"
	}
}
```

Uses file contents directly as role text.

### Simple Command

```cue
contexts: {
	"git-status": {
		command: "git status --short"
	}
}
```

Uses command output directly as context text.

### Inline Prompt

```cue
contexts: {
	note: {
		prompt: "Important: This project uses Go 1.21"
	}
}
```

Uses prompt text directly.

### File with Template

```cue
contexts: {
	environment: {
		file:   "~/reference/ENVIRONMENT.md"
		prompt: "Read {{.file}} for environment context."
	}
}
```

Injects file path into prompt template.

### Command with Template

```cue
contexts: {
	"recent-changes": {
		command: "git log -5 --oneline"
		prompt: """
Recent commits:
{{.command_output}}

Focus on these changes during the session.
"""
	}
}
```

Injects command output into prompt template.

### File with Command Injection

**File: PROJECT.md**

```markdown
# My Project

## Recent Activity

{{.command_output}}

## Status

Work in progress.
```

**Config:**

```cue
contexts: {
	project: {
		file:    "./PROJECT.md"
		command: "git log -3 --oneline"
	}
}
```

Command output replaces `{{.command_output}}` in the file.

### Combined: File + Command + Prompt

```cue
contexts: {
	"complete-status": {
		file:    "./PROJECT.md"
		command: "git status --short"
		prompt: """
# Full Project Context

## Documentation
{{.file_contents}}

## Working Tree
{{.command_output}}
"""
	}
}
```

Both file contents and command output injected into prompt.

### Task with Instructions

```cue
tasks: {
	"code-review": {
		command: "git diff --staged"
		prompt: """
Review these changes:

{{.command_output}}

Instructions: {{.instructions}}
"""
	}
}
```

Used via: `start task code-review "focus on security"`

The `{{.instructions}}` placeholder receives `"focus on security"`.

### Multi-line Script with Node.js

```cue
contexts: {
	"package-info": {
		shell:   "node"
		command: """
const pkg = require('./package.json');
console.log(`${pkg.name}@${pkg.version}`);
console.log(`Dependencies: ${Object.keys(pkg.dependencies).length}`);
"""
		prompt: "Package details:\n{{.command_output}}"
	}
}
```

### Go Template Features

```cue
contexts: {
	"file-list": {
		command: "ls -1"
		prompt: """
{{if .command_output}}
Files found:
{{.command_output}}
{{else}}
No files in directory.
{{end}}
"""
	}
}
```

Uses Go template conditional.

## Security Considerations

**Command execution runs shell scripts with full system access:**

1. **Validate command sources** - Only execute commands from trusted configurations
2. **Review local configs** - Local `./.start/` configs can execute arbitrary commands
3. **Be cautious with shared configs** - Review before using configs from others
4. **Timeout protection** - Commands are killed after timeout
5. **No automatic sudo** - Commands run with current user permissions
6. **CUE validation** - Schemas validate structure, not security of commands

**Best practices:**

- Keep sensitive commands in local config (not committed to git)
- Review any config before running `start`
- Use minimal permissions for command execution
- Prefer static files over dynamic commands when possible

## Schema Usage

In CUE schemas, UTD is defined as a reusable definition:

```cue
// schemas/utd.cue
package schemas

#UTD: {
	file?:    string
	command?: string
	prompt?:  string
	shell?:   string
	timeout?: int & >=1 & <=3600

	// Note: Go validates at least one of file/command/prompt required
}
```

Tasks, roles, and contexts embed `#UTD`:

```cue
// schemas/role.cue
#Role: {
	#UTD
	description?: string
}
```

This ensures consistent UTD behavior across all use cases.

## See Also

- [Design Records](design-records/) - Architectural decisions
- [Task Schema](../cue/task-schema.md) - Task-specific UTD usage
- [Role Schema](../cue/role-schema.md) - Role-specific UTD usage
- [Context Schema](../cue/context-schema.md) - Context-specific UTD usage
