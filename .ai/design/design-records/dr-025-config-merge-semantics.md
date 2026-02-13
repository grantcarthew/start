# dr-025: Configuration Merge Semantics

- Date: 2025-12-11
- Status: Accepted
- Category: Config

## Problem

When loading configuration from multiple sources (global `~/.config/start/` and local `./.start/`), the merge behaviour must be well-defined. CUE's native unification requires compatible values, but configuration override scenarios need replacement semantics.

Key questions:

1. If global defines `agents.claude` and local defines `agents.gemini`, should both exist or does local replace global entirely?
2. If both define `agents.claude`, should fields merge or should local replace the entire agent?
3. How should `settings` behave differently from collections like `agents`?

Without clear semantics, users cannot predict how their local configuration will interact with global settings.

## Decision

Implement two-level merge with distinct behaviour for collections versus settings.

Collection keys (`agents`, `contexts`, `roles`, `tasks`):

- Items merge additively by name
- Same-named items: local completely replaces global (no field-level merge)

Settings and other struct keys:

- Fields merge additively
- Same field: local value replaces global value

Scalar keys:

- Local completely replaces global

## Structure

### Collections: Additive by Item Name

```cue
// Global (~/.config/start/)
agents: {
    claude: { command: "claude", bin: "claude" }
}

// Local (./.start/)
agents: {
    gemini: { command: "gemini", bin: "gemini" }
}

// Result: Both agents exist
agents: {
    claude: { command: "claude", bin: "claude" }
    gemini: { command: "gemini", bin: "gemini" }
}
```

### Collections: Same Name Replaces Entirely

```cue
// Global
roles: {
    reviewer: {
        description: "Global reviewer"
        prompt: "Global prompt"
        timeout: 60
    }
}

// Local
roles: {
    reviewer: {
        description: "Local reviewer"
        prompt: "Local prompt"
    }
}

// Result: Local replaces entirely (timeout is gone)
roles: {
    reviewer: {
        description: "Local reviewer"
        prompt: "Local prompt"
    }
}
```

### Settings: Field-Level Merge

```cue
// Global
settings: {
    timeout: 120
    shell: "/bin/bash"
    default_agent: "claude"
}

// Local
settings: {
    default_agent: "gemini"
}

// Result: Fields merge, same fields replaced
settings: {
    timeout: 120            // from global
    shell: "/bin/bash"      // from global
    default_agent: "gemini" // local overrides
}
```

## Why

Additive Collections:

- Users expect local config to extend global, not replace entirely
- Adding a project-specific agent should not remove globally-configured agents
- Matches mental model of "local overrides and extends global"

Item-Level Replacement:

- If you redefine an agent locally, you want full control
- Partial field merging creates confusion about which fields apply
- Explicit replacement is predictable

Field-Level Settings:

- Settings are typically independent options
- Changing one setting should not require redeclaring all others
- Matches common configuration patterns (environment variables, dotfiles)

## Trade-offs

Accept:

- Slightly more complex merge logic
- Must maintain list of collection keys
- Behaviour differs by key type

Gain:

- Intuitive extension of global config
- Predictable replacement when same item defined
- Settings behave as users expect
- No need to duplicate global config in local

## Alternatives

Top-Level Replacement:

- Pro: Simpler implementation
- Con: Users must duplicate global agents if they want to add one locally
- Rejected: Too much duplication required

Full Deep Merge:

- Pro: Maximum flexibility
- Con: Confusing - which fields from which source?
- Con: Hard to "remove" a field defined globally
- Rejected: Too unpredictable

CUE Native Unification:

- Pro: Uses CUE's built-in semantics
- Con: Requires compatible values (same types, constraints)
- Con: Cannot replace a string with a different string
- Rejected: Configuration needs replacement, not unification
