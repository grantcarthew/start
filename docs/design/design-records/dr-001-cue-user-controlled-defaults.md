# DR-001: User-Controlled Defaults in CUE Schemas

- Date: 2025-12-02
- Status: Accepted
- Category: CUE

## Problem

Where should default values be defined in a CUE-based configuration system? Should schemas contain defaults, or should users control them?

The choice affects how users customize behavior for their environment (slow local model vs fast cloud API), schema complexity, and potential for conflicting defaults.

## Decision

Schemas contain ONLY constraints (validation rules), NO default values.

Users control all defaults via pattern constraints in their configuration files.

```cue
// Schema: pure constraints only
#Task: {
    timeout?: int & >=1 & <=3600  // NO default
}

// User config: user sets defaults
tasks: [_]: #Task & {
    timeout: *120 | _  // User's global default
}
```

## Why

CUE's unification model prevents multiple defaults from working together. When two defaults conflict, CUE treats the field as if NO defaults were provided.

Example problem with schema defaults:
```cue
// Schema
#Task: {timeout: *120 | int}

// User wants different default
tasks: [_]: #Task & {timeout: *300 | int}

// Result: Both defaults conflict, neither applies
```

By keeping schemas pure, users have ONE place to set defaults. User with slow local model changes `*120` to `*600` in one line - affects all tasks.

Clear separation:
- Schemas define "what is valid"
- Users define "what is typical" for their environment

## Trade-offs

Accept:
- Users must write pattern constraints in their config
- More initial setup for users
- Requires understanding CUE's default syntax

Gain:
- Users control ALL defaults in one place
- No conflicting defaults
- Flexibility for different environments
- Schemas simpler and easier to validate
- Defaults visible and documented in user's config

## Alternatives

Schema defaults:

Pro: Users see defaults immediately, batteries included
Con: Conflicts with user defaults, assumes one-size-fits-all
Rejected: CUE's unification makes this unworkable

Template layer:

Pro: Separation of constraints and defaults, follows CUE convention
Con: Adds complexity, still conflicts with user defaults, violates KISS
Rejected: Unnecessary complexity for minimal benefit

Go code defaults:

Pro: Simple fallback
Con: Hidden from users, not discoverable, can't be easily changed
Rejected: Violates principle of visible configuration
