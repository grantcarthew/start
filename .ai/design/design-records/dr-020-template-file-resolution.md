# dr-020: Template Processing and File Resolution

- Date: 2025-12-08
- Updated: 2026-01-21
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

All UTD content files are processed through Go's `text/template` engine. Temp files are created **only for non-local files** (files outside the working directory, such as those from the CUE module cache). Local files are validated for existence but not copied.

Temp directory location: `.start/temp/` (project-local)

File naming convention: `<type>-<path-segments>.md`

Examples (non-local files only):

- `@module/role.md` (from CUE cache) → `.start/temp/role-golang-assistant.md`
- `@module/task.md` (from CUE cache) → `.start/temp/task-code-review.md`

Local files (relative paths or absolute paths under working directory):

- `AGENTS.md` → No temp file created, original path preserved
- `./docs/context.md` → No temp file created, original path preserved
- `/project/path/file.md` (if under working dir) → No temp file created

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

Conditional temp file creation (local vs non-local):

- Local files are already accessible to AI agents - no copy needed
- Avoids unnecessary disk I/O for common case (local project files)
- Non-local files (CUE cache) require copying to accessible location
- `{{.file}}` returns meaningful paths for local files ("AGENTS.md" vs temp path)
- File existence is still validated for local files (fails fast if missing)

## Trade-offs

Accept:

- Requires `.start/temp/` in `.gitignore` (only created when non-local files are used)
- Slightly more complex logic (local vs non-local check)
- User manages cleanup manually

Gain:

- Zero security configuration for agent access
- No disk I/O for local files (common case)
- Meaningful `{{.file}}` paths for local files
- Non-local files still accessible via temp copies

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

Conditional resolution based on template presence:

- Pro: Skip I/O for simple files
- Con: Inconsistent behavior
- Con: False positives for `{{` in content
- Con: Complex logic
- Rejected: Too fragile, doesn't address the real issue

Conditional resolution based on file locality (adopted):

- Pro: Skip I/O for local files (already accessible)
- Pro: Meaningful `{{.file}}` paths for local files
- Pro: Simple logic (relative path or under working dir)
- Pro: Still validates file existence
- Adopted: Best balance of efficiency and correctness

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
| `{{.role_file}}` | Path to role file (original for local, temp for non-local/inline) |
| `{{.prompt}}` | Assembled prompt text |
| `{{.bin}}` | Agent binary |
| `{{.model}}` | Selected model |

Both `{{.role}}` and `{{.role_file}}` retained: different agents require different input methods (inline vs file path).

**`{{.role_file}}` semantics:**

The `{{.role_file}}` placeholder follows the same local vs non-local logic as UTD `{{.file}}`:

| Role Source | `{{.role_file}}` Value |
|-------------|------------------------|
| `file:` with local path | Original file path (already accessible) |
| `file:` with `@module/` path | Temp file path (`.start/temp/role-<name>.md`) |
| `prompt:` or `command:` (inline) | Temp file path (resolved content written to temp) |

This ensures:

- No unnecessary temp files for local role configurations
- Agents can always read role content via the file path
- Consistent behaviour with UTD `{{.file}}` placeholder

When a role uses inline content (`prompt:` or `command:`), there is no source file, so the resolved content must be written to a temp file for agents that require file-based input (e.g., Claude's `--append-system-prompt-file`).

## Execution Flow

Template resolution:

1. Extract UTD fields (file, command, prompt)
2. Resolve `@module/` paths using origin field
3. Determine if file is local:
   - Relative path (e.g., `AGENTS.md`, `./file.md`) → local
   - Absolute path under working directory → local
   - Otherwise → non-local
4. For local files: validate existence with `os.Stat()`
5. For non-local files: copy to `.start/temp/` and update path
6. Process through Go template engine
7. Provide path via `{{.file}}` or content via `{{.file_contents}}`

File naming derivation (for non-local files):

1. Take entity type and name
2. Replace path separators with hyphens
3. Format as `<type>-<name>.md`

Example: Task `start/create-task` → `task-start-create-task.md`

Conflict handling:

- If temp file exists, overwrite it
- No versioning or deduplication
- Each resolution replaces previous

## Temp Directory Management

Location: `.start/temp/`

Creation: Only when non-local files are used (e.g., registry-installed assets with `@module/` paths). Projects using only local files will never see this directory.

Cleanup: Manual (user responsibility)

.gitignore: Entry `.start/temp/` recommended if using registry assets

Bootstrap check: On startup, if in a git repository and `.start/temp/` exists but not in `.gitignore`, emit warning:

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
