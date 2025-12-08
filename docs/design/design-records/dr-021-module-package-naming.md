# DR-021: Module and Package Naming Conventions

- Date: 2025-12-08
- Status: Accepted
- Category: CUE

## Problem

Assets (roles, tasks, contexts, agents) need consistent naming conventions for:

- CUE module paths (for publishing to Central Registry)
- CUE package names (for imports and referencing)
- Directory structure (for organization)

The CUE Central Registry requires module paths to match existing GitHub repository structure. Package names must be valid CUE identifiers (no hyphens).

## Decision

Module paths follow the repository directory structure under `start-assets`. Package names match the directory basename when possible, converting hyphens to no separator.

Module path format:

```
github.com/grantcarthew/start-assets/<type>/<category>/<name>@v0
```

Examples:

- `github.com/grantcarthew/start-assets/schemas@v0`
- `github.com/grantcarthew/start-assets/roles/golang/assistant@v0`
- `github.com/grantcarthew/start-assets/tasks/golang/code-review@v0`

Package naming rules:

1. Match the directory basename when it contains no hyphens
2. Remove hyphens (concatenate) when basename contains hyphens
3. Use lowercase only

Examples:

| Directory | Basename | Package |
|-----------|----------|---------|
| `roles/golang/assistant/` | `assistant` | `assistant` |
| `tasks/golang/code-review/` | `code-review` | `codereview` |
| `contexts/environment/` | `environment` | `environment` |

## Why

Module paths under start-assets:

- Central Registry requires GitHub repository to exist at module path location
- All assets live in `start-assets` repo, so paths must be under it
- Matches physical directory structure (predictable, discoverable)
- Single repository for all assets simplifies management

Package names matching basename:

- Enables short import form when package matches basename
- No alias needed for single imports
- Self-documenting (package name indicates what it is)

Hyphen handling (remove, don't underscore):

- Hyphens improve directory readability (`code-review` vs `codereview`)
- Package names can't contain hyphens (CUE identifier rules)
- Concatenation (`codereview`) preferred over underscores (`code_review`) for brevity
- Requires explicit `:package` in import, but this is acceptable trade-off

## Trade-offs

Accept:

- Hyphenated directories require `:package` suffix on import
- Package name differs from basename for hyphenated directories
- Must remember concatenation rule for hyphenated names

Gain:

- Readable directory names (hyphens allowed)
- Short imports for non-hyphenated names
- Consistent, predictable conventions
- Works with Central Registry requirements

## Import Patterns

When package matches basename (short import):

```cue
import "github.com/grantcarthew/start-assets/roles/golang/assistant@v0"

r: assistant.role
```

When package differs from basename (explicit package):

```cue
import "github.com/grantcarthew/start-assets/tasks/golang/code-review@v0:codereview"

t: codereview.task
```

Multiple imports:

```cue
import (
    "github.com/grantcarthew/start-assets/roles/golang/assistant@v0"
    "github.com/grantcarthew/start-assets/roles/golang/teacher@v0"
)

r1: assistant.role
r2: teacher.role
```

Multiple imports with aliases (when needed):

```cue
import (
    goassist "github.com/grantcarthew/start-assets/roles/golang/assistant@v0"
    pyassist "github.com/grantcarthew/start-assets/roles/python/assistant@v0"
)

r1: goassist.role
r2: pyassist.role
```

## Directory Structure

Assets follow this structure:

```
start-assets/
├── schemas/                 → github.com/.../schemas@v0
│   ├── cue.mod/
│   └── *.cue
├── roles/
│   └── golang/
│       ├── assistant/       → github.com/.../roles/golang/assistant@v0
│       │   ├── cue.mod/
│       │   ├── role.cue     (package assistant)
│       │   └── role.md
│       └── teacher/         → github.com/.../roles/golang/teacher@v0
├── tasks/
│   └── golang/
│       └── code-review/     → github.com/.../tasks/golang/code-review@v0
│           ├── cue.mod/
│           ├── task.cue     (package codereview)
│           └── task.md
└── contexts/
    └── environment/         → github.com/.../contexts/environment@v0
```

## Export Field Naming

Each package exports a single primary field matching the asset type:

| Asset Type | Export Field |
|------------|--------------|
| Role | `role: schemas.#Role` |
| Task | `task: schemas.#Task` |
| Context | `context: schemas.#Context` |
| Agent | `agent: schemas.#Agent` |

Usage:

```cue
import "github.com/grantcarthew/start-assets/roles/golang/assistant@v0"

myrole: assistant.role  // package.field
```

## Alternatives

Separate repositories per asset:

- Pro: Each asset is independent
- Pro: Package name could match repo name
- Con: Hundreds of tiny repositories
- Con: Central Registry verified one repo exists per module
- Con: Management overhead
- Rejected: Single repository is simpler

Underscores in package names:

- Pro: Matches hyphen positions (`code_review` mirrors `code-review`)
- Con: Longer than concatenated version
- Con: Inconsistent with non-hyphenated names
- Rejected: Concatenation is shorter and consistent

Generic package names (role, task):

- Pro: Short, simple
- Con: All roles have same package name
- Con: Always need aliases for multiple imports
- Con: Not self-documenting
- Rejected: Specific names enable short imports

Underscores in directory names:

- Pro: Package name matches basename exactly
- Con: Less readable (`code_review` vs `code-review`)
- Con: Non-standard for URLs and paths
- Rejected: Readability more important than exact match
