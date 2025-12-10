# DR-002: No Name Field in Task Schema

- Date: 2025-12-02
- Status: Accepted
- Category: CUE

## Problem

Should tasks have a `name` field that duplicates the map key?

```cue
tasks: {
    "code-review": {
        name: "code-review"  // Duplicates the key
        command: "..."
    }
}
```

The map key already identifies the task. Adding a `name` field creates redundancy and potential for inconsistency.

## Decision

Tasks do NOT have a `name` field. The map key IS the task name.

```cue
#Task: {
    description?: string
    command?: string
    // NO name field
}

tasks: {
    "code-review": {  // Key is the name
        command: "git diff"
    }
}
```

Go code uses the map key when loading tasks:

```go
for taskName, taskConfig := range tasks {
    // taskName = "code-review"
    // Use taskName throughout
}
```

## Why

The map key already serves as the identifier:

- Key is required (can't have anonymous map entries)
- Key is unique within the map
- Key is how users reference tasks (`start task code-review`)
- Key is searchable and discoverable

Adding a `name` field would require:

- Pattern constraint to auto-inject: `tasks: [Name=_]: #Task & {name: Name}`
- Duplication in every task definition
- Risk of mismatch between key and name field
- Extra validation to ensure consistency

Removing the field simplifies:

- Schema is simpler (one less field)
- User config is cleaner (no redundant data)
- No auto-injection pattern needed
- Go code already has the name from iteration

## Trade-offs

Accept:

- Task objects don't contain their own name
- Must pass name separately when task is isolated from map context

Gain:

- Simpler schema
- No duplication or potential inconsistency
- Cleaner user configuration
- No pattern constraint overhead
- Follows standard map usage patterns

## Alternatives

Auto-inject name via pattern constraint:

Pro: Task objects self-documenting, name available in object
Con: Duplicates map key, requires pattern in schema, extra validation
Rejected: Unnecessary complexity for no real benefit

Store name in separate metadata field:

Pro: Could have both key and display name
Con: Even more duplication, confusing semantics
Rejected: Over-engineering

Include name for export scenarios:

Pro: Exported JSON would include name
Con: CUE export can include keys, Go serialization can add names
Rejected: Can be handled at export/serialization time if needed
