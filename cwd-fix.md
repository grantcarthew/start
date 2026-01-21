# Terminology Fix: "local" → "cwd"

## Problem

The term "local" is overloaded:
- **Local config** = `.start/` (project-level config)
- **Global config** = `~/.config/start/` (user-level config)
- **Local file** = file within working directory (current usage in DR-020 and code)

This causes confusion when discussing file path handling.

## Proposed Change

Replace "local file" / "non-local file" terminology with "cwd" / "external":

| Current | Proposed |
|---------|----------|
| local file | file within working directory (cwd) |
| non-local file | external file |
| `isLocalFile()` | `isCwdPath()` |

## Files to Update

### Code
- `internal/orchestration/composer.go` - rename `isLocalFile()` → `isCwdPath()`
- `internal/orchestration/composer_test.go` - rename test `TestComposer_isLocalFile`

### Documentation
- `.ai/design/design-records/dr-020-template-file-resolution.md` - update terminology throughout

## Notes

- "cwd" (current working directory) is unambiguous
- "external" clearly means outside the working directory (e.g., CUE module cache)
- Aligns with existing `workingDir` field names in code
