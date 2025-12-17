# DR-028: CLI Assets Command

- Date: 2025-12-17
- Status: Accepted
- Category: CLI

## Problem

Users need to discover, install, and manage assets (roles, tasks, contexts, agents) from the CUE Central Registry. The prototype validated this need with GitHub-based asset management, but this version uses CUE modules distributed via the registry.

## Decision

The `start assets` command provides package management for registry assets. It fetches the published index from the CUE registry for discovery and uses standard CUE module fetching for installation.

Synopsis:

```bash
start assets browse              # Open GitHub asset registry in browser
start assets search <query>      # Search registry index
start assets add <query>         # Install asset to config
start assets list                # List installed assets with update status
start assets info <query>        # Show detailed asset information
start assets update [query]      # Update installed assets
start assets index               # Regenerate index.cue (asset repo only)
```

## Why

Unified command for asset operations:

- Single entry point for all package management
- Semantic separation from `start show` (config inspection) and `start task` (execution)
- Familiar pattern from prototype (`start assets`)

Index-based discovery:

- CUE Central Registry lacks a public search API
- Index module (`github.com/grantcarthew/start-assets/index@v0`) provides searchable metadata
- Small fetch (~5 KB) for fast discovery
- Existing infrastructure from P-006 auto-setup

CUE-native installation:

- Assets are CUE modules with proper versioning
- Installation adds imports to user config files
- `@v0` major version resolves to latest compatible version automatically
- Leverages existing `internal/registry` package

## Subcommands

### start assets browse

Open the configured GitHub asset repository in the default browser.

```bash
start assets browse
```

Opens: `https://github.com/grantcarthew/start-assets` (or configured repo)

Why keep browse:

- Visual exploration of available assets
- View README files and examples
- Complements terminal-based search

### start assets search

Search the registry index by keyword.

```bash
start assets search <query>      # Minimum 3 characters
start assets search golang       # Find golang-related assets
start assets search --verbose    # Show tags and module paths
```

Behaviour:

1. Fetch latest index from CUE registry
2. Substring match against names, descriptions, and tags
3. Display grouped by type (agents, roles, tasks, contexts)

Output:

```
Found 5 matches:

roles/
  golang/assistant      Go programming expert - collaborative mode
  golang/teacher        Go programming expert - instructional mode

tasks/
  golang/code-review    Comprehensive Go code review
  golang/tests          Generate tests for Go code
  golang/debug          Debug and resolve issues in Go code
```

### start assets add

Install an asset from the registry to user configuration.

```bash
start assets add <query>              # Search and install (global)
start assets add <query> --local      # Install to project config
start assets add golang/code-review   # Direct path install
```

Behaviour:

1. Search index for matching asset
2. If multiple matches, prompt for selection
3. If single match, confirm and install
4. Add CUE import to config file (`~/.config/start/` or `./.start/`)

Installation writes import to appropriate config file:

```cue
import "github.com/grantcarthew/start-assets/tasks/golang/code-review@v0"
```

### start assets list

List installed registry assets with update status.

```bash
start assets list                # All installed assets
start assets list --type tasks   # Filter by type
```

Output:

```
Installed assets:

agents/
  ai/claude             v0.0.3  (latest)

roles/
  golang/assistant      v0.0.2  (update available: v0.0.4)

tasks/
  golang/code-review    v0.0.1  (latest)
```

Distinct from `start show`:

- `start assets list` shows registry packages with versions
- `start show` displays resolved configuration content

### start assets info

Show detailed information about a specific asset.

```bash
start assets info golang/code-review
start assets info "code review"       # Search then show
```

Behaviour:

1. Find asset in index
2. Fetch the actual package module
3. Display full details (description, prompt content, file field, etc.)

Output:

```
Asset: golang/code-review
Type: tasks
Module: github.com/grantcarthew/start-assets/tasks/golang/code-review@v0

Description:
  Comprehensive Go code review for correctness, design, and idiomatic patterns

Tags: golang, review, code-quality, best-practices

Status:
  Installed: Yes (global)
  Version: v0.0.2
  Latest: v0.0.2

Use 'start assets add golang/code-review' to install.
```

### start assets update

Update installed assets to latest versions within their major version.

```bash
start assets update              # Update all
start assets update golang       # Update matching assets
start assets update --dry-run    # Preview without applying
start assets update --force      # Re-fetch even if current
```

Behaviour:

- Re-fetch modules to get latest within major version
- CUE's `@v0` automatically resolves to latest `v0.x.x`
- Report what was updated

Output:

```
Checking for updates...

  Updated roles/golang/assistant      v0.0.2 -> v0.0.4
  Current tasks/golang/code-review    v0.0.2 (latest)

Updated: 1 asset
Current: 1 asset
```

### start assets index

Regenerate the index.cue file in an asset repository.

```bash
start assets index
```

Behaviour:

1. Verify current directory is an asset repo (check for `agents/`, `roles/`, `tasks/`, `contexts/` directories)
2. Scan asset directories for published modules
3. Generate/update `index/index.cue`

Error if not in asset repo:

```
Error: Not an asset repository.

Required directories not found: agents/, roles/, tasks/, contexts/

This command is for asset repository maintainers only.
```

## Flags

| Flag | Commands | Description |
|------|----------|-------------|
| `--local` | add | Install to project config (`./.start/`) |
| `--verbose` | search, info, list | Show detailed output |
| `--dry-run` | update | Preview without applying |
| `--force` | update | Re-fetch even if current |
| `--type` | list | Filter by asset type |

Global flags that apply:

| Flag | Description |
|------|-------------|
| `--quiet` | Suppress output |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Network error or invalid input |
| 2 | Asset not found |
| 3 | File system error |

## Implementation Notes

Reuse existing infrastructure:

- `internal/registry.Client` for module fetching
- `internal/registry.FetchIndex()` for index retrieval
- Index already defines `IndexEntry` struct with module, description, tags, bin

Search algorithm:

- Case-insensitive substring matching
- Minimum 3 characters
- Search priority: name > path > description > tags
- Results grouped by type, sorted alphabetically within type

Config file modification:

- Parse existing CUE config
- Add import statement for new asset
- Write back preserving formatting where possible
- Create config file if it doesn't exist

## Trade-offs

Accept:

- Network required for search/add/update operations
- Index must be maintained alongside assets
- No offline package installation

Gain:

- CUE-native package management
- Automatic version resolution via major version
- Leverages existing registry infrastructure
- Familiar UX patterns from prototype

## Alternatives

Use `pkg` or `mod` as command name:

- Pro: Aligns with CUE terminology
- Con: Less descriptive of what's being managed
- Rejected: "assets" clearly describes the content (roles, tasks, etc.)

Omit `browse` command:

- Pro: Simpler command surface
- Con: Loses visual exploration option
- Rejected: Browse provides valuable discovery path

Omit `index` command:

- Pro: Simpler, less to maintain
- Con: Asset maintainers need a way to update the index
- Rejected: Essential for asset repo maintenance

Single `list` command for both registry and config:

- Pro: Fewer commands
- Con: Conflates package status with config content
- Rejected: Separate concerns (assets list = packages, show = content)
