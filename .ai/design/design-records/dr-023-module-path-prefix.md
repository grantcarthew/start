# dr-023: Module Path Prefix for File Resolution

- Date: 2025-12-09
- Status: Accepted
- Category: CUE

## Problem

CUE assets (tasks, roles, contexts) may reference content files using the `file` field. When assets are downloaded from the CUE Central Registry to the local cache, relative paths like `./task.md` no longer work because:

- The working directory is the user's project, not the cached module
- Relative paths resolve against the current working directory
- The actual file is in the CUE cache, not the project

The CLI needs a way to identify paths that should resolve relative to the cached module location.

## Decision

Use `@module/` as a path prefix to indicate module-relative file resolution. The CLI detects this prefix and resolves the path relative to the CUE cache directory where the module is stored.

File field format:

```cue
file: "@module/task.md"  // Resolves to CUE cache location
```

Resolution:

1. Check if path starts with `@module/`
2. If yes, strip prefix and resolve relative to cached module directory
3. If no, resolve as normal (working directory, absolute, etc.)

CUE cache location (Go):

```go
// From os.UserCacheDir() or CUE_CACHE_DIR env var
cacheDir, _ := os.UserCacheDir()
cueCache := filepath.Join(cacheDir, "cue")
// Module files at: cueCache/mod/extract/<module>@<version>/
```

## Why

@module/ prefix is simple:

- Single string prefix check
- No template parsing required
- Clear, readable intention
- Easy to implement

Not Go template syntax:

- `{{.module}}` requires template parsing
- Templates already used for content, not paths
- Mixing template and literal content is confusing
- `@module/` is a literal prefix, not dynamic

Shell-like convention:

- Similar to `~/` for home directory
- Familiar pattern for path expansion
- Easy to document and understand

CUE cache is the right resolution target:

- Downloaded modules are stored in CUE cache
- Cache location is well-defined (os.UserCacheDir/cue or CUE_CACHE_DIR)
- Consistent across platforms

## Trade-offs

Accept:

- Custom prefix syntax (not standard)
- CLI must implement resolution logic
- Only works for file field, not arbitrary paths

Gain:

- Published assets can reference companion files
- Works regardless of user's working directory
- Simple implementation (prefix check)
- No template processing overhead

## Resolution Algorithm

When processing file field:

```
1. path = task.file
2. if path.hasPrefix("@module/"):
    a. path = path.trimPrefix("@module/")
    b. cacheDir = getCUECacheDir()
    c. moduleDir = cacheDir + "/mod/extract/" + modulePath + "@" + version
    d. return filepath.Join(moduleDir, path)
3. else:
    a. return normal resolution (absolute, relative to cwd, etc.)
```

CUE cache directory resolution:

```
1. if CUE_CACHE_DIR env var set:
    return CUE_CACHE_DIR
2. else:
    return os.UserCacheDir() + "/cue"
```

Platform cache directories:

| Platform | Default Cache Dir |
|----------|-------------------|
| macOS | ~/Library/Caches/cue |
| Linux | ~/.cache/cue |
| Windows | %LocalAppData%/cue |

## Usage Examples

Task file field:

```cue
task: schemas.#Task & {
    file: "@module/task.md"  // Resolves to cached module location
    prompt: """
        {{.file_contents}}

        ## Custom Instructions

        {{.instructions}}
        """
}
```

Role file field:

```cue
role: schemas.#Role & {
    file: "@module/role.md"
}
```

Local development (no prefix):

```cue
task: schemas.#Task & {
    file: "./task.md"  // Resolves relative to cwd
}
```

Absolute path (no prefix):

```cue
task: schemas.#Task & {
    file: "/etc/my-config/task.md"  // Resolves as-is
}
```

## Scope

Applies to:

- `file` field in #Task, #Role, #Context, #Agent schemas
- Any UTD file field

Does not apply to:

- `prompt` field (content, not path)
- `command` field (shell command, not path)
- Template placeholders within content

## Alternatives

Template syntax ({{.module}}/task.md):

- Pro: Consistent with other placeholders
- Con: Requires template parsing for paths
- Con: Mixing template and literal in paths
- Con: Template errors harder to debug
- Rejected: Prefix is simpler

Variable syntax (${module}/task.md):

- Pro: Shell-like variable expansion
- Con: Another syntax to learn
- Con: Conflicts with shell expansion if quoted improperly
- Rejected: @module/ is clearer

No prefix, detect by context:

- Pro: Simpler for users
- Con: Ambiguous (is "./task.md" local or module?)
- Con: Magic behavior
- Rejected: Explicit is better

Absolute registry path:

- Pro: No resolution needed
- Con: Paths change with versions
- Con: User doesn't know cache location
- Rejected: @module/ abstracts the location
