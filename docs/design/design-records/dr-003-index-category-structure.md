# DR-003: Index Category Structure

- Date: 2025-12-02
- Status: Accepted
- Category: Index

## Problem

How should the asset discovery index organize and identify modules?

Options for index keys:

1. Flat keys: `"code-review"` → module path
2. Category in key: `"golang/code-review"` → module path
3. Nested structure: `golang: {"code-review": {...}}`

This affects user search commands, module paths, and directory organization.

## Decision

Index keys use `category/item` format matching both user input and directory structure.

```cue
tasks: {
    "golang/code-review": {
        module: "github.com/grantcarthew/start-task-golang-code-review@v0"
        description: "Review Go code changes"
        tags: ["golang", "review"]
    }
    "git/pre-commit": {
        module: "github.com/grantcarthew/start-task-git-pre-commit@v0"
    }
}
```

This maps to:

- User command: `start task golang/code-review`
- Directory: `tasks/golang/code-review/`
- Module: `start-task-golang-code-review@v0`

## Why

Consistency across all layers:

- User types: `golang/code-review`
- Index lookup: `tasks["golang/code-review"]`
- Directory path: `tasks/golang/code-review/`
- Module name includes category for clarity

Categories provide organization:

- Hundreds of tasks need structure
- `golang/lint`, `golang/test`, `golang/benchmark` group naturally
- Search can filter by category
- Browse by category for discovery

Direct lookup efficiency:

```go
// User: start task golang/code-review
indexEntry := index.tasks["golang/code-review"]
modulePath := indexEntry.module
// cue mod get <modulePath>
```

## Trade-offs

Accept:

- Longer keys to type
- Category required for all assets
- Must choose appropriate categories

Gain:

- Clear organization at scale
- Easy to browse by category
- Consistent user experience (input matches structure)
- Simple Go implementation (direct map lookup)
- Module names self-documenting

## Alternatives

Flat keys with category in module path only:

Pro: Shorter user input, simpler keys
Con: Loses organizational structure, name collisions likely
Rejected: Doesn't scale to hundreds of assets

Nested structure by category:

```cue
tasks: {
    golang: {
        "code-review": {...}
        "lint": {...}
    }
}
```

Pro: Natural grouping, clear hierarchy
Con: Complicates search (must iterate nested maps), breaks direct lookup pattern
Rejected: Makes search and resolution more complex

Category as separate field:

```cue
tasks: {
    "code-review": {
        category: "golang"
        module: "..."
    }
}
```

Pro: Flat keys, category available for filtering
Con: User input doesn't match key, confusing resolution
Rejected: Breaks consistency between user input and storage
