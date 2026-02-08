# DR-041: CLI Flag Asset Search

- Date: 2026-02-08
- Status: Accepted
- Category: CLI

## Problem

CLI flags `--agent`, `--role`, `--model`, and `--context` require exact names to resolve assets. Users must remember precise config key names, leading to errors like:

```
start --agent gemini-non-interactive prompt "Fix the bug."
Error: loading agent: agent "gemini-non-interactive" not found
```

The `start task` command already supports a three-tier resolution strategy (exact config, exact registry, substring search) with auto-install. The same user-friendly search should extend to all asset-selecting flags.

DR-012 specified "exact match first, then prefix match" for `--agent` and `--role` but this was never implemented. This DR supersedes that partial specification with a complete search design covering all four flags.

## Decision

Extend substring search to `--agent`, `--role`, `--model`, and `--context` flags. Each flag uses a resolution chain appropriate to its asset type. The existing `SearchIndex` scoring algorithm (name +3, module path +2, description +1, tags +1) is reused for registry and installed config searches.

### Resolution Chains

Agent (`--agent`, `-a`):

1. Exact match in installed config
2. Exact match in registry index
3. Substring search across installed config + registry
4. Error if no matches

Role (`--role`, `-r`):

1. File path check (if value looks like a path, use directly per DR-038)
2. Exact match in installed config
3. Exact match in registry index
4. Substring search across installed config + registry
5. Error if no matches

Model (`--model`, `-m`):

1. Exact match in selected agent's models map
2. Substring match in selected agent's models map
3. Passthrough to agent binary (value used as-is)

Context (`--context`, `-c`):

1. File path check (if value looks like a path, use directly per DR-038)
2. Exact name match in installed config
3. Exact name match in registry index
4. Substring search across installed config + registry (all matches included)
5. Warning if no matches, continue with other selected contexts

### Unified Context Search

The `--context` flag moves from tag-only selection to unified search across context names, descriptions, and tags. This replaces the previous tag-matching mechanism from DR-008.

Backward compatibility: existing tag-based usage like `-c golang` continues to work because tags are a searched field. A search for "golang" matches any context with "golang" in its name, description, or tags.

Comma-separated values and multiple flags are treated as independent searches with results unioned:

```bash
start -c golang,security        # Two searches, union results
start -c golang -c "dotai index" # Same: two searches, union results
```

A minimum match score threshold is applied to context results to prevent overly broad matches.

### Ambiguity Handling

For single-select flags (`--agent`, `--role`, `--model`):

| Scenario | TTY | Non-TTY |
|----------|-----|---------|
| Single match | Auto-select | Auto-select |
| Multiple matches | Interactive prompt | Error with match list |
| No matches | Error | Error |

For `--context` (multi-select): all matches above the score threshold are included. No ambiguity prompt needed.

### Auto-Install

When a search matches a registry asset that is not installed locally, auto-install it to config and proceed with execution. Same behaviour as `start task` today.

## Why

Consistency with task resolution:

- `start task` already implements three-tier resolution with search and auto-install
- Users expect the same convenience when specifying agents, roles, and contexts
- Reduces friction: users don't need to remember exact config key names
- Eliminates the separate `start assets add` step for flag-triggered discovery

Unified context search:

- Tag-only selection required users to know which tags exist
- Searching names and descriptions makes contexts discoverable
- Users can find assets by what they remember: `--context "dotai index"` instead of knowing the exact key name
- Tags still work as a search field, so existing tag workflows are preserved

Model search with passthrough:

- Models are agent-specific with a small search space (typically 3-5 entries)
- Passthrough must remain as final fallback for arbitrary model identifiers
- Substring matching helps: `--model son` resolves to `sonnet` without ambiguity

## Trade-offs

Accept:

- Registry fetch adds latency when exact config match fails
- Auto-install modifies config as a side effect of running a command
- Unified context search changes the mental model from "tags are selectors" to "everything is searchable"
- Minimum score threshold for contexts is a tuning parameter that may need adjustment

Gain:

- Consistent search UX across all asset-selecting flags
- Assets are discoverable without knowing exact names
- Auto-install removes friction for first-time use of registry assets
- Backward compatible: exact names still work as before, tag-based context selection still works

## Alternatives

Search on agent/role only (exclude model and context):

- Pro: Simpler, fewer edge cases
- Pro: Context tag semantics unchanged
- Con: Inconsistent UX across flags
- Con: Users still frustrated by model and context exact-match requirements
- Rejected: Partial search creates confusion about which flags support it

Tag-first with search fallback for contexts:

- Pro: Preserves tag semantics exactly
- Pro: No minimum score threshold needed for tag matches
- Con: Two mental models (tags vs search) depending on match
- Con: Users must know whether their input is a tag or a search term
- Rejected: Unified search is simpler and still supports tag-based workflows

Suggest-only for registry matches (no auto-install):

- Pro: No surprise config changes
- Pro: User explicitly decides to install
- Con: Extra step for every first-time use
- Con: Inconsistent with how `start task` already works
- Rejected: Friction outweighs safety concern; config changes are visible and reversible

## Execution Flow

When an agent-launching command (`start`, `prompt`, `task`) is executed with asset flags:

1. Parse all flag values
2. For each flag with a value, check for file path (role, context only)
   - If file path detected, resolve directly, skip search
3. Check `--no-role` flag; if set, skip role resolution entirely
4. Fetch registry index once (lazy, only if needed by any flag)
   - Share the fetched index across all flag resolutions in the same invocation
5. Resolve each flag through its resolution chain
6. For registry matches not installed locally, auto-install to config
7. For `--context`, apply minimum score threshold, include all qualifying matches
8. For `--agent`/`--role` with multiple matches, prompt in TTY or error in non-TTY
9. Proceed with command execution using resolved assets

## Edge Cases

File path precedence:

- `--role ./my-role.md` and `--context ./docs/guide.md` bypass search entirely
- File path detection happens before any resolution step

No-role flag:

- `--no-role` takes precedence over `--role`
- If both specified, role resolution is skipped entirely

Model with no models map:

- Agent has no `models` field defined
- Exact and substring match steps find nothing
- Value passes through directly to agent binary

Context with zero matches:

- Warn: "no contexts matched 'nonexistent'"
- Continue execution with other selected contexts (required contexts still load)

Registry unavailable:

- Skip registry tiers (exact registry match and registry search)
- Fall back to installed config search only
- Do not block execution if registry is unreachable

## Updates

- 2026-02-08: Initial version. Extends search to all asset-selecting flags. Context selection moves from tag-only to unified search.
