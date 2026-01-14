# dr-004: Module Naming Convention

- Date: 2025-12-02
- Status: Accepted
- Category: Module

## Problem

What naming convention should CUE modules follow in the registry?

Given:

- Asset type: task, role, agent, context
- Category: golang, git, ai, etc.
- Item name: code-review, lint, pre-commit

How should the module path be constructed?

Options:

1. `github.com/grantcarthew/start-task-code-review@v0` (no category)
2. `github.com/grantcarthew/start-task-golang-code-review@v0` (full category)
3. `github.com/grantcarthew/start-golang-task-code-review@v0` (category before type)

## Decision

Module paths include the full category in order: `start-{type}-{category}-{item}@v{major}`

Format: `github.com/grantcarthew/start-task-golang-code-review@v0`

Components:

- Domain: `github.com/grantcarthew`
- Prefix: `start` (product branding)
- Type: `task` (or role, agent, context)
- Category: `golang` (or git, ai, debug, etc.)
- Item: `code-review`
- Version: `@v0` (major version)

Examples:

- `github.com/grantcarthew/start-task-golang-code-review@v0`
- `github.com/grantcarthew/start-task-git-pre-commit@v0`
- `github.com/grantcarthew/start-role-programming-go-expert@v0`
- `github.com/grantcarthew/start-agent-ai-claude@v0`

## Why

Self-documenting module paths:

- Clear what type of asset (task, role, agent)
- Clear what category (golang, git, ai)
- Clear what specific item (code-review, lint)
- No ambiguity when browsing registry

Matches directory structure:

- Directory: `tasks/golang/code-review/`
- Module: `start-task-golang-code-review@v0`
- Index key: `"golang/code-review"`
- Consistent naming across all layers

Search and discovery:

- Registry search for "golang" finds all golang-related modules
- Module names are grep-able and searchable
- Type prefix groups related assets
- Category enables filtering

Avoids name collisions:

- `start-task-golang-lint` vs `start-task-python-lint`
- Category prevents conflicts across languages/domains
- Explicit type prevents task/role name conflicts

## Trade-offs

Accept:

- Longer module paths
- More verbose in configuration
- Category must be chosen upfront

Gain:

- Self-documenting modules
- Clear organization and discoverability
- Prevents name collisions
- Matches directory structure
- Easy to understand at a glance

## Alternatives

No category in module name:

Example: `start-task-code-review@v0`

Pro: Shorter paths, simpler naming
Con: Name collisions inevitable, unclear what language/domain, loses organizational clarity
Rejected: Doesn't scale, ambiguous

Category before type:

Example: `start-golang-task-code-review@v0`

Pro: Groups by category first
Con: Breaks type grouping (all tasks together), less conventional
Rejected: Type is more important primary grouping

Nested module paths:

Example: `github.com/grantcarthew/start/task/golang/code-review@v0`

Pro: Hierarchical, matches directory exactly
Con: OCI registry restrictions on path depth, more complex versioning
Rejected: OCI naming constraints make this problematic
