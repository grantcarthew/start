# dr-026: CLI Logic and I/O Separation

- Date: 2025-12-12
- Status: Accepted
- Category: CLI

## Problem

CLI commands need to be testable. The standard approach of writing output directly to stdout via `fmt.Printf()` makes testing difficult because:

- Tests cannot capture output without redirecting os.Stdout
- Logic and presentation are coupled
- Mocking becomes necessary for simple output verification

Cobra provides `cmd.OutOrStdout()` which returns a custom writer if set, otherwise stdout. However, threading an `io.Writer` through every function call becomes verbose as the codebase grows.

## Decision

Separate logic from I/O in CLI commands using three layers:

1. Logic functions (e.g., `prepareShowAgent`) - Return data structs, no I/O
2. Output functions (e.g., `printPreview`) - Accept `io.Writer` and data, handle formatting
3. Command functions (e.g., `runShowAgent`) - Thin layer wiring Cobra to logic and output

Pattern:

```
runShowCommand(cmd, args)
    │
    ├── prepareShow*(args) → ShowResult, error
    │       └── Returns data only, no printing
    │
    └── printPreview(cmd.OutOrStdout(), result)
            └── Formats and writes to writer
```

## Why

Testability without mocking:

- Logic functions return values that can be directly asserted
- Tests call `prepareShowAgent("claude", "")` and check `result.Content`
- No need to capture stdout or mock writers for logic tests

Clear separation of concerns:

- Logic functions focus on data transformation
- Output functions focus on formatting
- Command functions are minimal glue code

Follows Go idioms:

- Functions return values, callers decide what to do with them
- Errors returned, not printed
- Writers passed explicitly where needed

Scales with codebase:

- New commands follow the same pattern
- Output format changes only affect output functions
- Logic can be reused across commands

## Structure

Data struct for show commands:

```go
type ShowResult struct {
    ItemType string // "Agent", "Role", "Context", "Task"
    Name     string // Item name or summary
    Content  string // Formatted content
    FilePath string // Path to temp file
    FileSize int64  // Size of temp file
}
```

Logic function signature:

```go
func prepareShowAgent(name, scope string) (ShowResult, error)
```

Output function signature:

```go
func printPreview(w io.Writer, r ShowResult)
```

Command function:

```go
func runShowAgent(cmd *cobra.Command, args []string) error {
    result, err := prepareShowAgent(name, showScope)
    if err != nil {
        return err
    }
    printPreview(cmd.OutOrStdout(), result)
    return nil
}
```

## Trade-offs

Accept:

- More functions to write (logic + output + command)
- Data must be structured into return types
- Slightly more verbose for simple commands

Gain:

- Logic is directly testable without mocking
- Output format can change independently
- Clear boundaries between concerns
- Integration tests via Cobra work naturally with `cmd.SetOut(buf)`

## Alternatives

Pass io.Writer through all functions:

- Pro: Single function handles everything
- Pro: Fewer layers
- Con: Writer parameter pollutes all function signatures
- Con: Deep call chains become verbose
- Rejected: Adds noise without improving testability

Use package-level writer variable:

- Pro: No parameter passing
- Pro: Easy to swap for tests
- Con: Global state, harder to reason about
- Con: Parallel tests could conflict
- Rejected: Global state is problematic

Test only via Cobra command execution:

- Pro: Tests the real code path
- Pro: No internal function exposure needed
- Con: Cannot test logic in isolation
- Con: Output parsing in tests is fragile
- Rejected: Unit testing logic is more robust

## Implementation Notes

Commands where output happens:

- Command functions are the boundary where `cmd.OutOrStdout()` is used
- Internal functions return data, never print

Error handling:

- Logic functions return errors
- Command functions return errors to Cobra
- Cobra handles error display

Future commands:

- `start`, `prompt`, `task` commands should follow this pattern
- Agent execution output may need additional consideration (streaming)
