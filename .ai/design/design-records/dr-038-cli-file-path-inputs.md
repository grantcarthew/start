# DR-038: CLI File Path Inputs

- Date: 2026-01-16
- Status: Accepted
- Category: CLI

## Problem

Currently, `--role`, `--context`, `start task`, and `start prompt` only accept configured names or inline text. Users must define roles, contexts, and tasks in CUE configuration before using them.

This creates friction for:

- Ad-hoc usage with one-off prompts or roles
- Project-specific content stored as markdown files
- Experimentation before committing to configuration
- Scripting with file-based templates

Users want to point directly at files without configuration overhead.

## Decision

Allow file paths as direct inputs for role, context, task, and prompt.

Detection rule: If the value starts with `./`, `/`, or `~`, treat it as a file path. Otherwise, treat it as a config name (or inline text for prompt).

Supported inputs:

| Input | Config Name | File Path |
|-------|-------------|-----------|
| `--role` | `--role go-expert` | `--role ./roles/reviewer.md` |
| `--context` | `--context project` | `--context ./ctx/repo.md` |
| `start task` | `start task review` | `start task ./tasks/review.md` |
| `start prompt` | `start prompt "text"` | `start prompt ./prompts/analyze.md` |

Mixing is supported for contexts:

```bash
start --context ./local-ctx.md,project-info,./other.md
```

Order is preserved.

## Why

- Simple detection: Path prefixes (`./`, `/`, `~`) are unambiguous - config names never start with these characters
- No performance cost: Only stat filesystem when value looks like a path
- Familiar convention: Matches shell behavior (`./script` vs `script`)
- Enables ad-hoc workflows: Use files without touching configuration
- Consistent interface: Same rule applies to all content inputs

## Trade-offs

Accept:

- Slightly more complex input parsing
- File paths shown in output may be less readable than short names
- No validation of file content (just raw text)

Gain:

- Zero-config usage for ad-hoc files
- Project-local prompt libraries without CUE configuration
- Scripting flexibility with file-based templates
- Consistent behavior across all content inputs

## Alternatives

Explicit prefix (e.g., `file:./path`):

- Pro: No ambiguity even for edge cases
- Con: More typing, less natural
- Rejected: Path syntax is already unambiguous

Extension-based detection (e.g., `.md` files only):

- Pro: Simple check
- Con: Restricts to specific extensions
- Con: `config.md` file would conflict with `config` role name
- Rejected: Path prefix is more reliable

Config-first lookup (check config, then file):

- Pro: Config names take precedence
- Con: Can't override config with local file of same name
- Rejected: Path syntax makes intent explicit

## Display

File paths are shown as-is in CLI output:

```
Role: ./roles/reviewer.md
```

For contexts, missing files follow existing convention:

```
Context documents:
  Name              Status  Flags  File
  ./missing-ctx.md  ○              (not found)
```

## Implementation Notes

Code changes:

- `internal/orchestration/composer.go` - Add file path detection to role and context resolution
- `internal/cli/task.go` - Add file path support for task argument
- `internal/cli/prompt.go` - Add file path support for prompt argument
- Shared utility function for path detection: `isFilePath(s string) bool`

Help text updates:

- `internal/cli/root.go` - Update `--role` and `--context` flag descriptions
- `internal/cli/task.go` - Update command usage and long description
- `internal/cli/prompt.go` - Update command usage and long description

Documentation:

- `docs/cli/start.md` - Add file path examples
- `docs/cli/task.md` - Add file path examples
- `docs/cli/prompt.md` - Add file path examples

Testing:

- Unit tests for `isFilePath()` detection function
- Integration tests for each command with file paths
- Error handling tests for missing files
- Tests for mixed context inputs (paths and names)
