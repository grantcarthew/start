# Configuration Merge Semantics

When loading configuration from global (`~/.config/start/`) and local (`./.start/`), two-level merge applies with distinct behaviour for collections versus settings. Local always takes precedence.

## Rules

Collections (`agents`, `contexts`, `roles`, `tasks`):

- Items merge additively by name — both sources contribute their items
- Same-named item: local completely replaces global (no field-level merge)

Settings (`settings`):

- Fields merge additively
- Same field: local value replaces global value

Scalar keys:

- Local completely replaces global

## Examples

### Collections: different names — both survive

```cue
// Global (~/.config/start/)
agents: {
    claude: { command: "claude", bin: "claude" }
}

// Local (./.start/)
agents: {
    gemini: { command: "gemini", bin: "gemini" }
}

// Result
agents: {
    claude: { command: "claude", bin: "claude" }
    gemini: { command: "gemini", bin: "gemini" }
}
```

### Collections: same name — local replaces entirely

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

// Result: local replaces entirely (timeout gone)
roles: {
    reviewer: {
        description: "Local reviewer"
        prompt: "Local prompt"
    }
}
```

### Settings: field-level merge

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

// Result
settings: {
    timeout: 120            // from global
    shell: "/bin/bash"      // from global
    default_agent: "gemini" // local overrides
}
```

## Config File Naming

Each file uses a key matching its filename:

| File | Top-level key |
|------|---------------|
| `agents.cue` | `agents:` |
| `roles.cue` | `roles:` |
| `contexts.cue` | `contexts:` |
| `tasks.cue` | `tasks:` |
| `settings.cue` | `settings:` |
