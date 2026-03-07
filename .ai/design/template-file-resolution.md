# Template and File Resolution

Defines how UTD content files are processed through Go's `text/template` engine, how file paths are resolved, and what placeholders are available in each context.

## File Locality

Files are classified as either local (within the working directory) or external (outside it).

Local files (relative paths or absolute paths under the working directory):

- Existence validated with `os.Stat()`
- No temp file created — original path preserved
- `{{.file}}` returns the original path

External files (CUE module cache, absolute paths outside cwd):

- Copied to `.start/temp/` with a path-derived name
- `{{.file}}` returns the temp path (CUE cache is inaccessible to agents)

Temp file naming: `<type>-<path-segments>.md`
Examples: `@module/role.md` → `.start/temp/role-golang-assistant.md`

## UTD Placeholders

Available in `file`, `command`, and `prompt` fields of tasks, roles, and contexts:

| Placeholder | Value |
|-------------|-------|
| `{{.file}}` | Original path for local files; temp path for `@module/` files |
| `{{.file_contents}}` | Resolved file contents |
| `{{.command}}` | Command string |
| `{{.command_output}}` | Command execution output |
| `{{.datetime}}` | Current timestamp (ISO 8601) |
| `{{.instructions}}` | User CLI argument (tasks only; empty string otherwise) |

Lazy evaluation: `{{.file_contents}}` only reads the file if that placeholder appears in the template. `{{.command_output}}` only executes the command if that placeholder appears.

## Agent Command Placeholders

Available in agent `command` templates only:

| Placeholder | Value |
|-------------|-------|
| `{{.bin}}` | Agent binary (from `bin` field) |
| `{{.model}}` | Resolved model identifier |
| `{{.prompt}}` | Assembled prompt text |
| `{{.role}}` | Resolved role content (inline text) |
| `{{.role_file}}` | Path to role file |

`{{.role_file}}` follows the same locality rules as `{{.file}}`:

| Role source | `{{.role_file}}` value |
|-------------|------------------------|
| `file:` with local path | Original file path |
| `file:` with `@module/` path | Temp file path |
| `prompt:` or `command:` (inline) | Temp file path (content written to temp) |

All placeholder values are automatically shell-escaped (single-quote wrapped) by the executor. Do NOT add quotes around placeholders in command templates:

```cue
// Correct
command: "{{.bin}} --model {{.model}} --append-system-prompt {{.role}} {{.prompt}}"

// Wrong — causes double-quoting
command: "{{.bin}} --prompt '{{.prompt}}'"
```

The executor rejects quoted placeholders with a clear error.

## @module/ Prefix

The `@module/` prefix in a `file` field indicates the path should resolve relative to the CUE module cache, not the working directory.

```cue
file: "@module/task.md"   // resolves to CUE cache location
file: "./task.md"          // resolves relative to working directory
```

Resolution algorithm:

1. If path starts with `@module/`, strip prefix
2. Resolve against CUE cache: `$CUE_CACHE_DIR/mod/extract/<module>@<version>/`
3. Copy file to `.start/temp/<type>-<name>.md`

Cache directory:

| Platform | Default |
|----------|---------|
| macOS | `~/Library/Caches/cue` |
| Linux | `~/.cache/cue` |
| Windows | `%LocalAppData%/cue` |

Override with `CUE_CACHE_DIR` environment variable.

## Temp Directory

Location: `.start/temp/`

Created only when external files are used (e.g., registry assets with `@module/` paths). Projects using only local files will never see this directory.

Cleanup is manual. Add `.start/temp/` to `.gitignore` when using registry assets.

On startup, if `.start/temp/` exists in a git repository but is not in `.gitignore`, a warning is emitted:

```
Warning: .start/temp/ not in .gitignore - resolved files may be committed
```

## Resolution Flow

1. Extract UTD fields (`file`, `command`, `prompt`)
2. Resolve `@module/` paths using origin field from module metadata
3. Classify file as local or external
4. For local files: validate existence
5. For external files: copy to `.start/temp/`
6. Scan template for `{{.file_contents}}` — read file only if present
7. Scan template for `{{.command_output}}` — execute command only if present
8. Process through `text/template` engine
9. Return rendered text

## Escaping Template Syntax

To output a literal `{{` in content:

```markdown
In Go templates, use {{"{{"}} .variable {{"}}"}} for substitution.
```

## Scope

Template processing applies to all asset types:

| Type | UTD placeholders | Agent placeholders |
|------|------------------|--------------------|
| Roles | Yes | No |
| Tasks | Yes | No |
| Contexts | Yes | No |
| Agent commands | No | Yes |
