# P-029: CLI Show Verbose Inspection

- Status: In Progress
- Started: 2026-02-17
- Completed:

## Overview

Enhance the `start show` command from a basic metadata viewer into a comprehensive resource inspection tool. Currently the command displays CUE field values (description, file path string, prompt template) but does not show actual file contents, CUE definitions, origin/cache metadata, or support cross-category search.

This project addresses GitHub issue #42. The show command becomes the primary tool for understanding what any installed or registry asset contains, without needing `--dry-run` to see file contents.

## Goals

1. Enhanced no-arg listing with descriptions alongside names
2. Cross-category search via `start show <name>` across all asset types
3. Verbose dump showing full CUE definition, file contents, origin, cache paths, and config source
4. Selection list for ambiguous cross-category matches with registry auto-install
5. Inline error handling for unreadable files (show error, do not abort)

## Scope

In Scope:

- Rewrite `runShow()` for enhanced listing with descriptions
- Add cross-category search argument to top-level `start show`
- Rewrite `formatShowContent()` for verbose dump output
- Add exported wrappers in `composer.go` for `resolveModulePath()`, `extractOrigin()`, `extractUTDFields()`
- File content display for `@module/`, `~/`, and absolute/relative paths
- CUE definition formatting via `v.Syntax()` + `format.Node()`
- Config file path via `v.Pos().Filename()`
- Selection list for multiple matches (TTY interactive, non-TTY error)
- Registry auto-install on selection (reuse existing patterns)
- DR-042 colour standard compliance
- Updated tests

Out of Scope:

- Changes to subcommand behaviour beyond adopting the new verbose dump format
- Changes to `--dry-run` or runtime execution
- Changes to registry index structure
- New design records (existing DR-042 and DR-041 patterns apply)

## Current State

The show command lives in `internal/cli/show.go` (~400 lines) with these functions:

- `addShowCommand()` (line 24): Sets up `show` + 4 subcommands (role, context, agent, task)
- `runShow()` (line 84): Lists all items grouped by category, names only, no descriptions
- `runShowItem()` (line 128): Returns RunE handler for subcommands, delegates to `prepareShow()` + `printPreview()`
- `prepareShow()` (line 149): Loads config, resolves name (exact/substring), calls `formatShowContent()`
- `formatShowContent()` (line 241): Manually extracts and formats individual CUE fields per type
- `printPreview()` (line 374): Outputs header, separator, content, separator

Top-level `start show` currently rejects arguments (line 88-90: `unknownCommandError`).

The three-tier resolution pattern exists in `internal/cli/resolve.go` (`resolveAsset()` line 85) and the selection prompt pattern exists in both `resolve.go` (`promptAssetSelection()` line 526) and `task.go` (`promptTaskSelection()` line 627).

CUE definition formatting exists in `internal/cue/loader.go:267` (`formatValue()` using `v.Syntax() + cueformat.Node()`).

File resolution for `@module/` paths exists in `internal/orchestration/composer.go:754` (`resolveModulePath()`), but the function is unexported.

`v.Pos()` for getting CUE source file positions is not currently used anywhere in the codebase.

Tests in `show_test.go` cover exact match, substring match, first-item default, nonexistent items, and integration tests (~600 lines).

## Technical Approach

### Phase 1: Exported Wrappers in Composer

Add thin exported wrappers in `internal/orchestration/composer.go`:

- `ExtractUTDFields(v cue.Value) UTDFields` wrapping `extractUTDFields()`
- `ExtractOrigin(v cue.Value) string` wrapping `extractOrigin()`
- `ResolveModulePath(path, origin string) (string, error)` wrapping `resolveModulePath()`

These follow the existing pattern (e.g., `GetTaskRole()` is already exported in the same file).

### Phase 2: Enhanced No-Arg Listing

Rewrite `runShow()` to display name + description for each item:

```
agents/
  claude              Claude by Anthropic
roles/
  assistant           General assistant
  code-reviewer       Code reviewer
contexts/
  environment         System environment information
  git-status          Git repository status
tasks/
  review/git-diff     Comprehensive review of code changes
```

Category names coloured per `categoryColor()`. Asset names in default colour. Descriptions in `colorDim`. Column alignment based on longest name per category.

Extract descriptions by looking up `description` field from each item's CUE value. Items without descriptions show no trailing text.

### Phase 3: Verbose Dump

Replace `formatShowContent()` with a verbose dump that shows everything about a resource. The output order for each resource:

1. Header: Category-coloured type label + name
2. Separator: Magenta 79-char line
3. Metadata section:
   - `Config:` source `.cue` file path with item name in parentheses. Determined in show code by using `internalcue.NewLoader().LoadSingle()` on each config dir (global, local) and checking which contains the item. Self-contained in show — no changes to loader or other packages.
   - `Origin:` registry module path (only if origin field exists)
   - `Cache:` resolved CUE package cache directory (only if origin exists), derived from origin string
4. CUE Definition: The resolved CUE value formatted as CUE syntax using `v.Syntax(cue.Concrete(false), cue.Definitions(true), cue.Hidden(true), cue.Optional(true))` + `cueformat.Node()`. No `cue.Final()` — tolerates non-concrete constraint values.
5. File contents: For each `file` field reference:
   - `File:` original file reference (e.g., `@module/task.md`)
   - `Path:` resolved absolute path on disk
   - Full file contents in default colour
   - If file cannot be read, show `[error: <message>]` inline
6. Command: Display as string, not executed (label: `Command:`)
7. Closing separator

All metadata labels (`Config:`, `Origin:`, `Cache:`, `File:`, `Path:`, `Command:`) in `colorDim`. Parenthesised text uses `colorCyan` for `()` and `colorDim` for content inside.

File resolution handles three path types:
- `@module/` paths: use `ExtractOrigin()` + `ResolveModulePath()` to find in CUE cache
- `~/` paths: use `ReadFilePath()` which handles tilde expansion
- Absolute/relative paths: use `ReadFilePath()` directly

Cache directory derivation from origin: strip the relative file path from the resolved module path to get the module root directory. Or construct from origin string by splitting at `@` to get module path and version, then build cache path as `<cue-cache>/mod/extract/<module-path>@<version>/`.

### Phase 4: Cross-Category Search

Accept an optional argument on `start show` for cross-category search. Remove the `unknownCommandError` guard.

Resolution follows the same three-tier pattern as `resolveAsset()`:

1. Search all four categories in installed config (exact match, then substring)
2. Search all four categories in registry index (exact match)
3. Combined substring search across installed + registry

Results include category prefix (e.g., `tasks/review/git-diff`, `roles/code-reviewer`).

For single match: display verbose dump directly.
For multiple matches: show selection list (same pattern as `promptAssetSelection()`):

```
Found 3 matches for "review":

  1. tasks/review/git-diff        installed
  2. tasks/review/pr-comments     installed
  3. roles/code-reviewer          registry

Select (1-3):
```

Registry matches auto-install on selection (reuse `autoInstall()` from resolver).

The cross-category search needs a `resolver` instance. Create one in the `runShow()` handler when args are provided, similar to how `start task` creates a resolver in its run function.

Subcommands (`show agent`, `show role`, `show context`, `show task`) continue to work as category-scoped variants. Update them to use the new verbose dump format.

### Phase 5: Tests

Update `show_test.go`:

- Update `setupTestConfig()` to include items with origin fields and file references for testing verbose dump
- Update existing `TestPrepareShow*` tests for new verbose output format
- Add tests for enhanced listing (descriptions present)
- Add tests for CUE definition output (check for CUE syntax markers like `{` and field names)
- Add tests for config file path display
- Add tests for origin/cache metadata display (when origin field present)
- Add tests for file content display (create temp files, verify content appears in output)
- Add tests for file read error handling (reference nonexistent file, verify inline error)
- Add tests for cross-category search (match across categories, single match auto-select, ambiguity)
- Keep `skipRegistry: true` for all tests touching resolver

All tests remain non-parallel due to `os.Chdir()` usage in setup.

## Success Criteria

- [x] `start show` (no args) lists all items with descriptions, coloured per DR-042
- [x] `start show <name>` searches across all categories and displays verbose dump
- [x] `start show <name>` with multiple matches shows interactive selection list (TTY)
- [x] `start show <name>` with multiple matches shows error with match list (non-TTY)
- [x] Verbose dump shows CUE definition formatted as CUE syntax
- [x] Verbose dump shows config file source path
- [x] Verbose dump shows origin and cache path for registry-installed assets
- [x] Verbose dump shows full file contents for file-based resources
- [x] Verbose dump shows inline error for unreadable files
- [x] Verbose dump shows command as string (not executed)
- [x] Subcommands (`show agent/role/context/task`) use verbose dump format
- [x] Registry matches auto-install on selection
- [x] All output follows DR-042 colour standard
- [x] All existing tests pass (updated for new format)
- [x] New tests cover cross-category search, verbose dump, file content display, error handling
- [x] Tests run via `scripts/invoke-tests`

## Deliverables

- Updated `internal/orchestration/composer.go` with exported functions (renamed from unexported)
- Rewritten `internal/cli/show.go` with enhanced listing, cross-category search, verbose dump
- Updated `internal/cli/show_test.go` with comprehensive test coverage
- GitHub issue #42 closable upon completion

## Dependencies

- p-024 (CLI Flag Asset Search) - completed, provides three-tier resolution pattern
- p-025 (Terminal Colour Standard) - completed, provides DR-042 colour definitions

## Testing Strategy

Follow dr-024 testing approach:

- Test real behaviour with actual CUE config (no mocks)
- Use `t.TempDir()` for file content tests
- Table-driven tests for multiple cases
- Integration tests via Cobra command execution
- Set `skipRegistry: true` on resolver for unit tests
- Create temp files with known content to verify file display
- Test both TTY and non-TTY paths for selection

## Decision Points

1. Config source file path strategy — Decided: Show code uses `LoadSingle()` per config dir to find which file defines the item, then `v.Pos().Filename()` on the single-dir value for the real path. Self-contained in show code, no changes to loader.

## Known Bugs

- `start show review` vs `start show review/`: Without trailing slash, cross-category search finds "project/review" via short-name exact match in `findExactInstalledName` and displays it directly. With trailing slash it shows all review items. These should produce the same result. Fix: copy the pattern from `start task` in `task.go` — after finding an exact/short-name match, also run a substring search; if the substring search finds more matches than the single exact match, fall through to show the full match list instead of silently selecting the exact match. See `executeTask()` around the "For tasks: when an exact/short name match exists" comment block (~line 162-185 in task.go).

## Notes

- Cross-category search uses a resolver (and potentially registry). The existing resolver pattern already handles offline gracefully — installed matches work without registry access. No special handling needed.
