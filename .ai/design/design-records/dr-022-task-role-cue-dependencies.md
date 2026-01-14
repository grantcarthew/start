# dr-022: Task Role CUE Dependencies

- Date: 2025-12-09
- Status: Accepted
- Category: CUE

## Problem

Tasks reference roles by name. When tasks are published to the CUE Central Registry, they should declare roles as CUE module dependencies. This enables:

- Version pinning for role compatibility
- Automatic dependency resolution via `cue mod tidy`
- Registry verification of complete dependency graphs

However, the original schema (`role?: string`) only supports runtime name resolution. A string value like `role: "golang/agent"` does not create a CUE import dependency.

## Decision

The role field accepts either a string (runtime resolution) or a #Role struct (CUE dependency). Published tasks import their role package and reference the exported role value.

Schema change:

```cue
#Task: {
    #UTD
    description?: string
    tags?: [...string]
    role?:  string | #Role  // String OR imported role
    agent?: string
}
```

Published task pattern:

```cue
import (
    "github.com/grantcarthew/start-assets/schemas@v0"
    agentRole "github.com/grantcarthew/start-assets/roles/golang/agent@v0:agent"
)

task: schemas.#Task & {
    role: agentRole.role  // CUE dependency
    // ...
}
```

## Why

String values don't create CUE dependencies:

- `role: "golang/agent"` is a plain string
- `cue mod tidy` sees no import, removes unused deps
- Registry cannot verify role exists
- No version pinning

Importing role package creates dependency:

- Import statement declares the dependency
- `cue mod tidy` preserves it in module.cue
- Registry verifies role module exists
- Version pinned in `deps` section

Union type preserves local task flexibility:

- Published tasks: Import role, get CUE dependency
- Local tasks: Use string, resolve at runtime
- Both patterns valid against same schema

## Trade-offs

Accept:

- Published tasks require import statement for role
- More verbose than simple string reference
- Schema is slightly more complex (union type)

Gain:

- CUE Central Registry tracks role dependencies
- Version pinning prevents breaking changes
- `cue mod tidy` manages dependencies automatically
- Clear distinction between compile-time and runtime references

## Structure

Local task (string reference):

```cue
task: schemas.#Task & {
    role: "my-local-role"  // Resolved at runtime
    prompt: "..."
}
```

Published task (CUE dependency):

```cue
import (
    "github.com/grantcarthew/start-assets/schemas@v0"
    agentRole "github.com/grantcarthew/start-assets/roles/golang/agent@v0:agent"
)

task: schemas.#Task & {
    role: agentRole.role  // CUE import creates dependency
    prompt: "..."
}
```

Resulting module.cue deps:

```cue
deps: {
    "github.com/grantcarthew/start-assets/roles/golang/agent@v0": {
        v: "v0.0.1"
    }
    "github.com/grantcarthew/start-assets/schemas@v0": {
        v: "v0.0.2"
    }
}
```

## Validation

At schema load time (CUE):

- `role` field validates as either string or #Role struct
- No additional type checking required

At runtime (Go):

- If role is struct, extract content for use
- If role is string, resolve by name from configuration
- Both paths produce role content for agent

## Alternatives

Role field as struct only:

- Pro: Always a CUE dependency
- Con: Local tasks cannot use simple string references
- Con: Breaking change from original schema
- Rejected: Union type preserves flexibility

Separate fields (roleRef vs roleImport):

- Pro: Explicit distinction
- Con: Two fields for same purpose
- Con: Confusing which to use
- Rejected: Union type is simpler

Keep string only, document convention:

- Pro: No schema change
- Con: No actual CUE dependency created
- Con: Registry cannot verify dependencies
- Rejected: CUE dependency tracking is the goal
