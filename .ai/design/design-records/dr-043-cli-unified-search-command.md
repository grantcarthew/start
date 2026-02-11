# DR-043: CLI Unified Search Command

- Date: 2026-02-11
- Status: Accepted
- Category: CLI

## Problem

Users naturally type `start search` to find assets but the search command lives under `start assets search`. Adding a simple alias would break the CLI design — `assets` groups all registry operations, and pulling one subcommand to the top level without justification weakens that grouping.

The command needs to earn its top-level position by doing more than the existing `assets search`.

## Decision

Add a top-level `start search <query>...` command that searches across three data sources:

1. Local config (`./.start/`)
2. Global config (`~/.config/start/`)
3. Assets registry (remote index)

`start assets search` remains unchanged as a registry-only search.

## Why

- Users instinctively reach for `start search` — it is the most natural top-level command for discovery
- Searching all data sources justifies a top-level command rather than being a misplaced alias
- All building blocks already exist: `SearchInstalledConfig()` for configs, `SearchIndex()` for registry, same `SearchResult` type and scoring algorithm across both
- The three-section output gives users a complete picture of what is available and where it comes from

## Output Format

Sections appear in order: local, global, registry. Empty sections are omitted.

Within each section, results are sub-grouped by category (agents, roles, tasks, contexts).

```
local (./.start)
  roles/
    golang                    Go programming expert
  tasks/
    pre-commit-review         Review staged changes

global (~/.config/start)
  agents/
    claude                    Anthropic Claude AI assistant
  contexts/
    environment               Local environment info

registry
  roles/
    golang                    Go programming expert *
  tasks/
    pre-commit-review         Review staged changes *
    debug-help                Debug assistance
```

Rules:

- Section headers use the actual resolved path for local and global
- Category names coloured per DR-042 via `categoryColor()`
- Descriptions dimmed via `colorDim`
- Registry items that are already installed are marked with green `*`
- No deduplication across sections — an item appears in every section where it matches
- `--verbose` / `-v` shows tags and module paths (same format as `assets search --verbose`)
- Minimum 3 character query (same as `assets search`)

## Command Structure

```
start search <query>...        # Search local, global, and registry
start search golang            # Example
start search golang --verbose  # With tags and module paths
start find golang              # Alias
```

Flags:

- `--verbose` / `-v`: Show tags and module paths

## Execution Flow

When `start search <query>` is executed:

1. Parse and validate query terms (reuse `assets.ParseSearchTerms`, enforce 3-char minimum)
2. Resolve config paths via `config.ResolvePaths`
3. If local config exists:
   - Load local config only via `cue.NewLoader().Load([]string{paths.Local})`
   - Search all four categories via `assets.SearchInstalledConfig`
4. If global config exists:
   - Load global config only via `cue.NewLoader().Load([]string{paths.Global})`
   - Search all four categories via `assets.SearchInstalledConfig`
5. Fetch registry index via `registry.NewClient` and `FetchIndex`
   - Search via `assets.SearchIndex`
6. Collect installed asset names via `collectInstalledNames` for `*` markers
7. Print non-empty sections with category sub-grouping

## Trade-offs

Accept:

- Two search commands exist (`start search` and `start assets search`) with overlapping but different scope
- Three separate data loads (local config, global config, registry) — slightly slower than a single merged load
- Loading configs separately by scope rather than merged means the same CUE files are parsed twice if an item appears in both

Gain:

- Natural top-level command that users expect
- Complete view across all data sources in one command
- Clear source attribution (users see exactly where each item lives)
- `start assets search` remains focused on registry discovery
- No deduplication keeps the output honest and predictable

## Alternatives

Simple alias (`start search` → `start assets search`):

- Pro: Trivial to implement
- Con: Breaks CLI grouping design without justification
- Con: No additional value over the existing command
- Rejected: Does not earn a top-level position

Unified search replacing `assets search`:

- Pro: Single search command, no overlap
- Con: `assets search` is well-scoped for registry browsing
- Con: Would change existing command behaviour
- Rejected: Both commands serve distinct use cases

Flag-based scope filtering (`start search --local --global --registry`):

- Pro: Flexible filtering
- Con: Adds complexity for little gain — `assets search` already covers registry-only
- Con: More flags to learn
- Rejected: Simplicity over flexibility; always search everything
