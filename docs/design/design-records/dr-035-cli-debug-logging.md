# DR-035: CLI Debug Logging

- Date: 2025-12-23
- Status: Accepted
- Category: CLI

## Problem

The `--debug` flag exists but DEBUG logging coverage is inconsistent and limited:

- Only `start.go` has DEBUG statements
- Other packages (`orchestration`, `cue`, `config`, `task`, `prompt`) lack visibility
- Users troubleshooting issues have limited insight into execution flow
- Developers adding DEBUG statements have no guidelines for consistency

## Decision

Implement comprehensive DEBUG logging with:

1. Standardized format: `[DEBUG] <category>: <message>`
2. Coverage across all execution phases
3. Consistent helper function usage
4. Categories to aid filtering and understanding

## Categories

DEBUG messages grouped by execution phase:

| Category | Description | Package(s) |
|----------|-------------|------------|
| config | Configuration loading and merging | cli, cue, config |
| agent | Agent selection and validation | cli, orchestration |
| role | Role selection and UTD processing | orchestration |
| context | Context selection, filtering, UTD | orchestration |
| task | Task resolution and execution | cli, orchestration |
| compose | Prompt composition | orchestration |
| shell | Shell command execution | shell |
| exec | Final command building and execution | orchestration |

## Format

Standard format for all DEBUG messages:

```
[DEBUG] <category>: <message>
```

Examples:

```
[DEBUG] config: Loading global config from /Users/foo/.config/start
[DEBUG] config: Loading local config from /Users/foo/project/.start
[DEBUG] config: Merged 2 config directories
[DEBUG] agent: Selected agent "claude" (from config default)
[DEBUG] agent: Binary: claude, Command template: --print {prompt}
[DEBUG] role: Selected role "assistant" (from config default)
[DEBUG] role: Processing UTD - source: prompt field
[DEBUG] context: Selection criteria: required=true, defaults=true, tags=[]
[DEBUG] context: Including "project" (required)
[DEBUG] context: Including "codebase" (default)
[DEBUG] context: Skipping "testing" (no matching tags)
[DEBUG] compose: Role content: 42 bytes
[DEBUG] compose: Prompt content: 1523 bytes
[DEBUG] exec: Final command: claude --print /tmp/start-xxx/prompt.md
```

## Coverage Points

Minimum DEBUG points per phase:

### Config Phase
- Config directory resolution (global/local paths)
- Which directories exist
- Load start/completion for each directory
- Merge result summary

### Agent Phase
- Agent selection source (flag vs config default)
- Agent details (binary, command template, default model)
- Binary existence check result

### Role Phase
- Role selection source (flag vs task vs config default)
- UTD source type (file, command, prompt)
- UTD processing result (success/warning)

### Context Phase
- Selection criteria (required, defaults, tags)
- Each context: included or skipped with reason
- UTD processing per context
- Total contexts included

### Task Phase (task command only)
- Task resolution (exact match, substring match, ambiguous)
- Task details (prompt template, associated role)
- Instructions placeholder substitution

### Compose Phase
- Role content size
- Context content sizes
- Final prompt size

### Exec Phase
- Command template expansion
- Final command string
- Execution method (exec vs dry-run)

## Helper Function

Centralised helper in `cli/start.go`:

```go
func debugf(flags *Flags, category, format string, args ...interface{}) {
    if flags.Debug {
        fmt.Fprintf(os.Stderr, "[DEBUG] %s: "+format+"\n", append([]interface{}{category}, args...)...)
    }
}
```

For packages without access to Flags, pass a debug boolean or use a debug writer pattern.

## Trade-offs

Accept:

- Additional code in each package
- Slight performance overhead when debug enabled
- Maintenance burden to keep DEBUG statements current
- DEBUG output can be verbose

Gain:

- Users can self-diagnose common issues
- Faster issue resolution with debug output in bug reports
- Developers understand execution flow
- Consistent troubleshooting experience

## Alternatives

Structured logging library (logrus, zap, slog):

- Pro: Levels, formatting, output options
- Pro: Industry standard approach
- Con: Additional dependency
- Con: Overkill for CLI tool debug output
- Rejected: Simple printf-style sufficient for CLI tool

Log to file instead of stderr:

- Pro: Doesn't clutter terminal output
- Con: Users must find/open file
- Con: Additional complexity
- Rejected: Immediate stderr output more useful for CLI debugging

## Implementation Notes

1. Update existing `debugf` in `start.go` to include category parameter
2. Add DEBUG statements to each coverage point
3. Consider passing debug flag through to orchestration package
4. Test with `--debug --dry-run` to verify coverage

## Usage Example

```bash
./start --debug --dry-run

[DEBUG] config: Resolving paths (working dir: /Users/foo/project)
[DEBUG] config: Global: /Users/foo/.config/start (exists)
[DEBUG] config: Local: /Users/foo/project/.start (not found)
[DEBUG] config: Loading from 1 directory
[DEBUG] agent: Selected "claude" (config default)
[DEBUG] agent: Binary: claude
[DEBUG] role: Selected "assistant" (config default)
[DEBUG] role: UTD source: prompt field
[DEBUG] context: Selection: required=true, defaults=true, tags=[]
[DEBUG] context: Including "project" (required)
[DEBUG] context: Including "codebase" (default)
[DEBUG] compose: Role: 28 bytes
[DEBUG] compose: Prompt: 1842 bytes (2 contexts)
[DEBUG] exec: Dry-run mode, skipping execution
Dry Run - Agent Not Executed
...
```
