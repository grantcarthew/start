# DR-019: Index Bin Field for Agent Detection

- Date: 2025-12-04
- Status: Accepted
- Category: Index

## Problem

Auto-setup needs to detect which AI CLI tools are installed on the user's system. The index contains agent metadata but lacks the binary name needed to check PATH. Without this, auto-setup would need to fetch every agent package just to read its `bin` field, which is wasteful.

## Decision

Add an optional `bin` field to `#IndexEntry`. For agent entries, this specifies the executable name to check in PATH for auto-detection.

```cue
#IndexEntry: {
    module:       string & =~"^[a-z0-9.-]+/[a-z0-9/_-]+@v[0-9]+$"
    description?: string
    tags?:        [...string]
    version?:     string & =~"^v[0-9]+\\.[0-9]+\\.[0-9]+$"
    bin?:         string & !=""
}
```

## Why

Lightweight detection:

- Fetch index once (small, cached)
- Check each agent's `bin` against PATH locally
- No need to fetch agent packages for detection
- Only fetch the package user actually selects

Index is the right place:

- Index already contains agent metadata
- Single source of truth for discovery
- Consistent with existing pattern (module, description, tags)

Optional and backward compatible:

- Field is optional, only meaningful for agents
- Existing index entries work without modification
- Tasks, roles, contexts simply don't use it

## Schema

Field definition:

```cue
// bin is the executable name for PATH detection (agents only)
// Used by auto-setup to detect installed AI CLI tools
bin?: string & !=""
```

Constraints:

- Optional (only agents use it)
- Non-empty string if provided
- No regex constraint (binary names vary by platform)

## Usage

Index example:

```cue
agents: {
    "ai/claude": {
        module:      "github.com/grantcarthew/start-agent-ai-claude@v0"
        description: "Anthropic Claude AI agent"
        bin:         "claude"
    }
    "ai/gemini": {
        module:      "github.com/grantcarthew/start-agent-ai-gemini@v0"
        description: "Google Gemini AI agent"
        bin:         "gemini"
    }
    "ai/ollama": {
        module:      "github.com/grantcarthew/start-agent-ai-ollama@v0"
        description: "Ollama local LLM runner"
        bin:         "ollama"
    }
}
```

Detection flow:

1. Fetch index from registry
2. Iterate `index.agents`
3. For each entry with `bin` field, check `exec.LookPath(bin)`
4. Collect matches
5. Single match → auto-select
6. Multiple matches → prompt user
7. Fetch selected agent's package

## Trade-offs

Accept:

- Index schema grows (one optional field)
- Agents must declare `bin` to be auto-detected
- Index must be updated when new agents added

Gain:

- No wasted network requests during detection
- Fast local PATH checking
- Single index fetch enables full detection
- Minimal schema change

## Alternatives

Separate detection file:

- Pro: Keeps index focused on discovery
- Pro: Smaller file for just detection
- Con: Another file to fetch and maintain
- Con: Duplicates agent list
- Rejected: Index already lists agents, adding field is simpler

Built-in detection list:

- Pro: No network fetch for detection
- Pro: Works offline
- Con: Hardcoded in CLI binary
- Con: New agents require CLI update
- Rejected: Registry-based is more flexible

Fetch all agent packages:

- Pro: No index changes needed
- Pro: Always uses authoritative bin from package
- Con: Wasteful network requests
- Con: Slow first run
- Rejected: Too expensive for detection

Convention-based (infer bin from key):

- Pro: No extra field needed
- Pro: `ai/claude` → binary `claude`
- Con: Breaks for non-matching names
- Con: Limits naming flexibility
- Rejected: Explicit is better than implicit
