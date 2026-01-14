# dr-027: Registry Module Fetching

- Date: 2025-12-15
- Status: Accepted
- Category: Registry

## Problem

Auto-setup (dr-018) needs to fetch modules from the CUE Central Registry:

1. The index module containing agent metadata and `bin` fields for detection
2. The selected agent module to write to user config

How should we interact with the registry, where does the index live, and how do we write the fetched config to disk?

## Decision

Use `modconfig.NewRegistry()` to fetch modules from the CUE Central Registry. The index lives at `github.com/grantcarthew/start-assets/index@v0`. Fetched agent configuration is decoded to Go structs and encoded back to CUE for the user config file.

## Why

Index location:

- All start assets live in the `github.com/grantcarthew/start-assets` repository
- CUE registry requires module paths to match GitHub repository structure
- Dedicated index module (`/index@v0`) enables small fetch (~1-2 KB) for detection
- Independent versioning from schemas and individual assets

API choice:

- `modconfig.NewRegistry()` is the high-level CUE API for registry access
- Handles authentication via `cue login` automatically
- Respects `CUE_REGISTRY` environment variable
- Uses standard CUE cache (`~/.cache/cue/`)
- Less code than raw `modregistry.Client`

Config writing strategy:

- Go structs for agents already exist (`internal/orchestration/executor.go`)
- CUE encoding logic already exists (`internal/cue/loader.go`)
- Decode/encode produces clean minimal config without module boilerplate
- Consistent with existing codebase patterns

Error handling:

- Network failures use retry with exponential backoff (2-3 attempts)
- Transient failures are common; immediate failure frustrates users
- After retries exhausted, exit with code 1 and helpful message

## Index Module Structure

```cue
package index

import "github.com/grantcarthew/start-assets/schemas@v0"

agents: [string]: schemas.#IndexEntry
tasks: [string]: schemas.#IndexEntry
roles: [string]: schemas.#IndexEntry
contexts: [string]: schemas.#IndexEntry

agents: {
    "ai/claude": {
        module:      "github.com/grantcarthew/start-assets/agents/claude@v0"
        description: "Anthropic Claude AI agent"
        bin:         "claude"
    }
    "ai/gemini": {
        module:      "github.com/grantcarthew/start-assets/agents/gemini@v0"
        description: "Google Gemini AI agent"
        bin:         "gemini"
    }
}
```

## Fetch Flow

```
1. modconfig.NewRegistry(nil)
   → Returns Registry interface with auth and caching

2. registry.Fetch(ctx, "github.com/grantcarthew/start-assets/index@v0")
   → Downloads and caches module
   → Returns module.SourceLoc with filesystem path

3. Load CUE from SourceLoc, decode agents map
   → Extract bin field for each agent
   → Check exec.LookPath(bin) in parallel

4. Select agent (auto or prompt based on count)

5. registry.Fetch(ctx, selectedAgent.module)
   → Downloads agent module

6. Decode agent CUE to Go struct (orchestration.Agent)

7. Encode Go struct back to CUE
   → Write to ~/.config/start/agents.cue
```

## Retry Strategy

Network operations use exponential backoff:

```
Attempt 1: immediate
Attempt 2: 1 second delay
Attempt 3: 2 second delay
Total max wait: ~3 seconds
```

After exhausting retries:

```
Error: Cannot fetch configuration from registry.
Check your network connection and try again.

If the problem persists, verify registry access:
  cue login
```

Exit code 1 (unified exit code policy).

## Trade-offs

Accept:

- Dependency on CUE's auth and caching mechanisms
- Index module must be maintained alongside assets
- Network required for first run (no offline bootstrap)
- Less control over cache location

Gain:

- Works with existing `cue login` authentication
- Respects user's `CUE_REGISTRY` configuration
- Standard CUE cache shared with other CUE tools
- Minimal code using existing patterns
- Retry logic handles transient failures

## Alternatives

Separate index repository:

- Pro: Fully independent versioning
- Con: Requires maintaining separate GitHub repo
- Con: Breaks convention of assets in single repo
- Rejected: Module paths must match repo structure

Raw `modregistry.Client`:

- Pro: Full control over registry interaction
- Pro: Custom cache location possible
- Con: Manual auth setup required
- Con: More code to maintain
- Rejected: `modconfig` handles complexity for us

Copy raw CUE files:

- Pro: Preserves original formatting and comments
- Con: Includes module boilerplate (package, imports)
- Con: May include fields not needed in user config
- Rejected: Decode/encode produces cleaner output

Fail immediately on network error:

- Pro: Simpler implementation
- Con: Frustrating for transient failures
- Rejected: Retry is low cost and improves UX

## Updates

- 2025-12-22: Aligned exit codes with unified policy (0 success, 1 failure)
