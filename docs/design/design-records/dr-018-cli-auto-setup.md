# DR-018: CLI Auto-Setup

- Date: 2025-12-04
- Status: Accepted
- Category: CLI

## Problem

New users need to configure at least one agent before `start` can work. Requiring an explicit `start init` command adds friction to the first-run experience. The tool should "just work" when possible while remaining configurable for advanced use cases.

## Decision

When no configuration exists, `start` automatically detects installed AI CLI tools, prompts the user to select one if multiple are found, fetches the agent configuration from the registry, and writes it to global config. No explicit `start init` is required for basic usage.

## Why

Minimal friction:

- First run should work with minimal user intervention
- Users with one AI tool installed get zero prompts
- Users with multiple tools make one quick choice

Progressive complexity:

- Basic: auto-detect, auto-configure, start using
- Intermediate: run `start init` for roles, contexts, local config
- Advanced: manually edit CUE files, create packages

Fail gracefully:

- No agents detected → helpful error with install suggestions
- Network unavailable → clear error about needing to fetch config
- Detection is best-effort, not blocking

## Index Extension

Add `bin` field to `#IndexEntry` for agent detection:

```cue
#IndexEntry: {
    module:       string & =~"^[a-z0-9.-]+/[a-z0-9/_-]+@v[0-9]+$"
    description?: string
    tags?:        [...string]
    version?:     string & =~"^v[0-9]+\\.[0-9]+\\.[0-9]+$"
    bin?:         string  // Agent binary name for PATH detection
}
```

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
}
```

The `bin` field is optional and only meaningful for agents. It specifies the binary name to check in PATH for auto-detection.

## Conditional Command Templates

Agent command templates use Go template conditionals to handle missing values:

```cue
agents: "claude": {
    bin:     "claude"
    command: "{{.bin}}{{if .model}} --model {{.model}}{{end}}{{if .role}} --append-system-prompt {{.role}}{{end}}{{if .prompt}} {{.prompt}}{{end}}"
    default_model: "sonnet"
    models: {
        sonnet: "claude-sonnet-4-20250514"
    }
}
```

This enables:

- Minimal config (just agent) → command runs without role/prompt flags
- Missing role file → warn, skip, command still executes
- Missing contexts → warn, skip, prompt may be empty but command valid
- Graceful degradation when components are unavailable

## Auto-Setup Flow

First run with no config:

1. Check for config in `~/.config/start/` and `./.start/`
2. No config found → enter auto-setup mode
3. Fetch index from registry
4. Scan agents in index, check each `bin` value against PATH
5. Results:
   - None found → error with helpful message listing common AI CLIs to install
   - One found → use it automatically
   - Multiple found → prompt user to select
6. Fetch selected agent's package from registry
7. Write agent config to `~/.config/start/agents.cue`
8. Continue with normal execution

Subsequent runs:

- Config exists → skip auto-setup, use existing config
- User can re-run setup via `start init` if desired

## Terminal Output

No agents detected:

```
No AI CLI tools detected in PATH.

Install one of:
  claude  - Anthropic Claude (https://claude.ai/claude-code)
  gemini  - Google Gemini CLI
  aichat  - Multi-provider AI CLI
  ollama  - Local LLM runner

Then run 'start' again.
```

Single agent detected:

```
Detected: claude
Fetching configuration...
Configuration saved to ~/.config/start/

Starting AI agent...
```

Multiple agents detected:

```
Multiple AI CLI tools detected:

  1. claude  - Anthropic Claude
  2. gemini  - Google Gemini CLI
  3. ollama  - Local LLM runner

Select agent [1-3]: 2

Fetching configuration...
Configuration saved to ~/.config/start/

Starting AI agent...
```

## Exit Codes

All commands use Unix minimal exit codes: 0 on success, 1 on any error. Error messages printed to stderr describe the specific failure.

## Trade-offs

Accept:

- First run requires network access to fetch index and package
- Index must be kept updated with agent bin mappings
- Detection is PATH-based, may miss non-standard installations
- Auto-setup only configures agent, not roles or contexts

Gain:

- Zero-config first run for users with one AI tool
- One prompt for users with multiple tools
- No mandatory init command
- Progressive complexity - basics work immediately
- Graceful handling of missing values via conditional templates

## Alternatives

Require explicit init:

- Pro: User explicitly chooses configuration
- Pro: No network on first run
- Con: Extra step before tool works
- Con: Friction for simple use case
- Rejected: Auto-setup provides better first-run experience

Built-in agent configs:

- Pro: No network fetch needed
- Pro: Works offline immediately
- Con: Configs bundled in binary, harder to update
- Con: Binary grows with each agent added
- Rejected: Registry-based keeps CLI small and configs updatable

Detect and configure all found agents:

- Pro: User has all options configured
- Pro: Can switch agents without re-fetching
- Con: Fetches packages user may never use
- Con: Clutters config with unused agents
- Rejected: Fetch only what's needed

Hardcoded agent priority (skip prompt):

- Pro: Never prompts, always "just works"
- Con: Our priority may not match user preference
- Con: Users discover other agents accidentally
- Rejected: Brief prompt respects user choice

## Implementation Notes

PATH detection:

- Use `exec.LookPath()` for each `bin` value in index
- Check in parallel for faster detection
- Cache results during auto-setup flow

Index fetching:

- Fetch from CUE registry
- Cache locally after first fetch
- Consider bundling minimal index for offline bootstrap

Package installation:

- Use `cue mod` commands to fetch agent package
- Write to `~/.config/start/` as CUE files
- Set as default agent in settings

Interactive selection:

- Simple numbered list for TTY
- Error with list of options for non-TTY

## Updates

- 2025-12-22: Aligned exit codes with unified policy (0 success, 1 failure)
