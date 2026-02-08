# p-024: CLI Flag Asset Search

- Status: Pending
- Started: (not yet started)
- Completed: (not yet completed)
- Issues: #5

## Overview

Extend substring search to CLI flags `--agent`, `--role`, `--model`, and `--context`. Currently these flags require exact config key names, producing hard errors on mismatch. The `start task` command already implements three-tier resolution (exact config, exact registry, substring search) with auto-install. This project brings the same user-friendly search to all asset-selecting flags.

This also changes `--context` from tag-only selection to unified search across names, descriptions, and tags.

Design: DR-041

## Goals

1. Add three-tier resolution (exact config, exact registry, substring search) to `--agent` flag
2. Add three-tier resolution to `--role` flag (with file path bypass per DR-038)
3. Add substring search within agent's models map to `--model` flag (preserving passthrough)
4. Replace `--context` tag-only selection with unified search across names, descriptions, and tags
5. Auto-install registry matches that aren't locally installed
6. Handle ambiguity: interactive prompt in TTY, error with match list in non-TTY
7. Fetch registry index once per invocation, shared across all flag resolutions

## Scope

In Scope:

- Resolution logic for `--agent`, `--role`, `--model`, `--context` flags
- Searching installed config entries (not just registry index)
- Auto-install on registry match
- TTY vs non-TTY ambiguity handling for single-select flags
- Minimum match score threshold for context multi-select
- Shared registry index fetch across flags
- File path bypass for `--role` and `--context`
- `--no-role` precedence over `--role`
- Tests for all new resolution paths
- Graceful fallback when registry is unavailable

Out of Scope:

- Multi-word search (issue #16, separate feature)
- Changes to `start assets search` command behaviour
- Changes to `start task` resolution (already works)
- New CLI flags or commands
- Changes to the search scoring algorithm itself
- Shell completion for search results

## Success Criteria

- [ ] `start --agent <substring>` resolves via search when exact match fails
- [ ] `start --role <substring>` resolves via search when exact match fails
- [ ] `start --model <substring>` resolves via substring match in agent's models map
- [ ] `start --model <unknown>` still passes through to agent binary
- [ ] `start --context <term>` searches across name, description, and tags
- [ ] `start -c golang` still works (backward compatible with tag-based usage)
- [ ] `start -c golang,security` performs two searches, unions results
- [ ] Context results filtered by minimum match score threshold
- [ ] Single match auto-selects for agent/role/model
- [ ] Multiple matches prompt in TTY for agent/role/model
- [ ] Multiple matches error with list in non-TTY for agent/role/model
- [ ] Registry match auto-installs to config
- [ ] Registry index fetched once when multiple flags need it
- [ ] `--role ./path/to/file.md` bypasses search (file path detection)
- [ ] `--context ./path/to/file.md` bypasses search (file path detection)
- [ ] `--no-role` skips role resolution entirely
- [ ] Registry unavailable: falls back to installed config search only
- [ ] All existing tests pass
- [ ] New tests cover each resolution chain

## Current State

Verified: 2026-02-08

Flag resolution (exact match only):

- `prepareExecutionEnv()` in `internal/cli/start.go` (line 67) resolves agent, role, model, context
- `ExtractAgent()` in `internal/orchestration/executor.go` (line 344) does exact CUE lookup
- `resolveRole()` in `internal/orchestration/composer.go` (line 452) does exact CUE lookup
- `selectContexts()` in `internal/orchestration/composer.go` (line 308) does tag-based selection

Affected entry points (all flow through `prepareExecutionEnv` and `Compose`/`ComposeWithRole`):

- `runStart` in `internal/cli/start.go` (root command)
- `runPrompt` in `internal/cli/prompt.go` (calls `executeStart`)
- `executeTask` in `internal/cli/task.go` (calls `prepareExecutionEnv` directly)

Reference implementation (three-tier search):

- `executeTask()` in `internal/cli/task.go` (line 72) implements exact config, exact registry, substring search with auto-install and interactive prompt

Existing search infrastructure:

- `SearchIndex()` in `internal/assets/search.go` searches registry index with scoring (name +3, path +2, description +1, tags +1)
- `InstallAsset()` in `internal/assets/install.go` handles registry fetch and config writing
- Both already exported and used by CLI commands and auto-setup

Installed config search gap:

- `SearchIndex()` operates on `registry.Index` (registry entries with `IndexEntry` struct)
- Installed config entries are CUE values, not `IndexEntry` structs
- Need a way to search installed config entries with the same scoring approach

Flag definitions:

- All flags defined as persistent on root command in `internal/cli/root.go`
- `--agent` (`-a`): `StringVarP`
- `--role` (`-r`): `StringVarP`
- `--model` (`-m`): `StringVarP`
- `--context` (`-c`): `StringSliceVarP`
- `--no-role`: `BoolVar` (added in commit 4c7c2c0)

## Deliverables

Files:

- Updated `internal/cli/start.go` - Flag resolution with search fallback
- Updated `internal/orchestration/executor.go` - Search-aware agent extraction or new helper
- Updated `internal/orchestration/composer.go` - Search-aware role and context resolution
- New or updated search helpers in `internal/assets/` - Installed config search support

Tests:

- Tests for agent flag resolution chain (exact, registry, search, ambiguity)
- Tests for role flag resolution chain (file path, exact, registry, search)
- Tests for model flag resolution chain (exact, substring, passthrough)
- Tests for context unified search (name, description, tags, multi-select, threshold)
- Tests for TTY vs non-TTY ambiguity handling
- Tests for auto-install on registry match
- Tests for registry unavailable fallback

Design Records:

- DR-041 (already written)

## Technical Approach

Phase 1 - Installed config search:

- Create a function to search installed config entries using the same scoring approach as `SearchIndex`
- Extract name, description, and tags from CUE config values for each asset type
- Return results in the same `SearchResult` format for uniform handling

Phase 2 - Shared resolution helper:

- Create a resolution function that implements the three-tier chain
- Accept parameters for: config value, registry index, query string, asset category
- Return: resolved asset name, whether it needs auto-install, or match list for ambiguity
- Handle file path bypass, single/multi match, TTY detection

Phase 3 - Flag integration:

- Wire resolution helper into `prepareExecutionEnv()` for agent, role, model
- Wire unified search into context selection in composer
- Share registry index fetch across all resolutions in a single invocation
- Lazy fetch: only contact registry if exact config match fails

Phase 4 - Model search:

- Simpler: search only the selected agent's models map
- No registry involvement
- Preserve passthrough as final fallback

Phase 5 - Context unified search:

- Replace tag-matching logic in `selectContexts()` with unified search
- Each comma-separated value or repeated flag is an independent search
- Union all results above minimum score threshold
- Backward compatible: tag matches still score via the tags field

## Dependencies

Requires:

- p-021 (Auto-Setup Default Assets) - Complete (provides `internal/assets` package)

## Testing Strategy

Unit Tests:

- Resolution chain for each flag type with mock config and index
- Scoring threshold filtering for context multi-select
- File path detection bypass
- `--no-role` precedence
- TTY vs non-TTY match handling
- Registry unavailable fallback

Integration Tests:

- End-to-end: `start --agent <partial>` resolves and launches correct agent
- End-to-end: `start -c <term>` includes matching contexts
- Auto-install flow: search finds registry match, installs, proceeds
- Multiple flags in one invocation share single index fetch

## Notes

Interaction with issue #16 (multi-word search):

- Issue #16 is about positional argument parsing (`start assets search one two three`)
- Flag values are always a single string (shell/Cobra handles quoting)
- The two features are orthogonal and can be implemented independently
- The underlying search function takes a single query string either way

Context backward compatibility:

- Existing `-c golang` usage continues to work because tags are a searched field
- "golang" as a substring matches contexts with "golang" in tags, name, or description
- No breaking change for current tag-based workflows
