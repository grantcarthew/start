# dr-020: Template Processing and File Resolution

- Date: 2025-12-08
- Status: Accepted
- Category: UTD

## Problem

Content files (roles, tasks, contexts) may contain Go template syntax that must be resolved before use. When resolved content is passed to an AI agent, the agent needs file access without additional security configuration.

Requirements:

- Process all content files through Go's template engine
- Provide resolved content via file path or inline text
- Ensure agents can access resolved files without sandbox exceptions
- Use meaningful file names that provide context to agents
- Handle template resolution transparently

## Decision

All UTD content files are processed through Go's `text/template` engine. Resolved content is written to `.start/temp/` with path-derived file names. Placeholders return resolved content, not original file references.

Temp directory location: `.start/temp/` (project-local)

File naming convention: `<type>-<path-segments>.md`

Examples:

- `roles/golang/assistant/role.md` → `.start/temp/role-golang-assistant.md`
- `tasks/code-review/task.md` → `.start/temp/task-code-review.md`
- `contexts/environment.md` → `.start/temp/context-environment.md`

## Why

Project-local temp directory:

- Agents already have project access (no security config needed)
- No sandbox exceptions or allowlists required
- Files accessible to any agent with project read access
- Predictable location for debugging

Path-derived naming:

- Semantic names provide context to agents
- "Read role-golang-assistant.md" is meaningful
- "Read tmp-a8f3b2c1.md" provides no context
- Names are unique (derived from full path)

Always resolve to temp files:

- Consistent behavior (no conditional logic)
- Agents always see resolved content
- Original files may contain unresolved template syntax
- Simple mental model for users

## Trade-offs

Accept:

- Requires `.start/temp/` in `.gitignore`
- Creates files in project directory
- User manages cleanup manually
- Some disk I/O for temp file creation

Gain:

- Zero security configuration for agent access
- Meaningful file names for agent context
- Consistent template resolution
- Simple, predictable behavior

## Alternatives

System temp directory (/tmp):

- Pro: Standard temp location
- Pro: No project directory changes
- Con: Agent may not have access (sandboxing)
- Con: Random names provide no context
- Con: Requires security exceptions
- Rejected: Agent access is critical

User config temp (~/.config/start/temp/):

- Pro: Outside project directory
- Pro: No .gitignore needed
- Con: Agent may not have access
- Con: Path not project-relative
- Rejected: Agent access is critical

Conditional resolution (only if templates present):

- Pro: Skip I/O for simple files
- Con: Inconsistent behavior
- Con: False positives for `{{` in content
- Con: Complex logic
- Rejected: Consistency outweighs I/O savings

In-memory only (no temp files):

- Pro: No disk I/O
- Con: Some agents require file paths (Gemini)
- Con: Cannot use `{{.file}}` placeholder
- Rejected: File path access is required

## Placeholder Semantics

UTD placeholders (in content files):

| Placeholder | Value |
|-------------|-------|
| `{{.file}}` | Original file path for local files; temp path for `@module/` files |
| `{{.file_contents}}` | Resolved file contents |
| `{{.command}}` | Command string |
| `{{.command_output}}` | Command execution output |
| `{{.date}}` | Current timestamp (ISO 8601) |
| `{{.instructions}}` | User CLI arguments (tasks only) |

**Note on `{{.file}}` behavior:**

- **Local files** (`file: "AGENTS.md"`): `{{.file}}` returns the original path for semantic clarity in prompts
- **Module files** (`file: "@module/task.md"`): `{{.file}}` returns the temp path because the CUE cache is inaccessible to AI agents

This ensures prompts like "Read {{.file}} for context" produce meaningful output ("Read AGENTS.md") for local files while still providing accessible paths for module files.

Agent command placeholders:

| Placeholder | Value |
|-------------|-------|
| `{{.role}}` | Resolved role content (inline text) |
| `{{.role_file}}` | Path to resolved role temp file |
| `{{.prompt}}` | Assembled prompt text |
| `{{.bin}}` | Agent binary |
| `{{.model}}` | Selected model |

Both `{{.role}}` and `{{.role_file}}` retained: different agents require different input methods (inline vs file path).

## Execution Flow

Template resolution:

1. Read content file (role.md, task.md, context.md)
2. Parse as Go template
3. Execute template with placeholder data
4. Generate temp file path from source path
5. Write resolved content to `.start/temp/<derived-name>.md`
6. Provide path via `{{.file}}` or content via `{{.file_contents}}`

File naming derivation:

1. Take source path relative to asset root
2. Replace path separators with hyphens
3. Prepend type prefix (role-, task-, context-)
4. Append .md extension

Example: `roles/golang/assistant/role.md` → `role-golang-assistant.md`

Conflict handling:

- If temp file exists, overwrite it
- No versioning or deduplication
- Each resolution replaces previous

## Temp Directory Management

Location: `.start/temp/`

Cleanup: Manual (user responsibility)

.gitignore: Required entry `.start/temp/`

Bootstrap check: On startup, if in a git repository and `.start/temp/` not in `.gitignore`, emit warning:

```
Warning: .start/temp/ not in .gitignore - resolved files may be committed
```

No automatic modification of .gitignore.

## Scope

Template processing applies to:

| Type | Template Processing |
|------|---------------------|
| Roles | Yes |
| Tasks | Yes |
| Contexts | Yes |
| Agent commands | Yes (different placeholder set) |

## Usage Examples

Role with conditional content:

```markdown
# Go Expert

{{if .command_output}}
Environment: {{.command_output}}
{{end}}

## Instructions
- Provide idiomatic Go solutions
```

Agent command using file path:

```
GEMINI_SYSTEM_MD={{.role_file}} {{.bin}} --model {{.model}} {{.prompt}}
```

Agent command using inline content:

```
{{.bin}} --model {{.model}} --append-system-prompt {{.role}} {{.prompt}}
```

Escaping template syntax:

To output literal `{{`, use `{{"{{"}}`:

```markdown
In Go templates, use {{"{{"}} .variable {{"}}"}} for substitution.
```
