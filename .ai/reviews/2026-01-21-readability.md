# Go Readability Code Review

**Date:** 2026-01-21
**Reviewer:** Claude Opus 4.5
**Codebase:** start - AI Agent CLI Orchestrator

---

## Executive Summary

The `start` codebase demonstrates **excellent readability** overall. The code follows Go idioms consistently, uses clear naming conventions, and is well-organised. The codebase reads as coherent, maintainable software with a clear separation of concerns.

**Rating:** Very Good

**Key Strengths:**
- Consistent coding style throughout
- Excellent use of doc comments on exported symbols
- Clear package boundaries with single responsibilities
- Well-structured tests using table-driven patterns
- Good use of early returns to reduce nesting

**Areas for Improvement:**
- A few functions exceed recommended length
- Some code duplication in config handling commands
- Minor inconsistencies in error message formatting

---

## 1. Formatting

### Findings

**Import Grouping:** Consistent throughout the codebase. Standard library imports are grouped first, followed by external dependencies, then internal packages. Example from `internal/cli/start.go:1-17`:

```go
import (
    "context"
    "fmt"
    "io"
    "os"
    "strings"

    "github.com/grantcarthew/start/internal/config"
    internalcue "github.com/grantcarthew/start/internal/cue"
    "github.com/grantcarthew/start/internal/orchestration"
    ...
)
```

**Line Length:** Generally well-managed. Long lines are broken sensibly, particularly in struct initialisations and function calls.

**Vertical Spacing:** Good use of blank lines to group related code blocks. Functions are clearly separated.

### Concerns

None significant. The code appears to be formatted with `gofmt`.

---

## 2. Naming

### Variables

**Strengths:**
- Short names used appropriately for short scopes (`w` for writer, `r` for reader, `i` in loops)
- Longer descriptive names for longer scopes (`workingDir`, `configDir`, `defaultAgent`)
- Consistent acronym casing (`URL`, `TTY`, `CUE`)

**Examples of good naming:**
- `internal/cli/start.go:56-62`: `ExecutionEnv` struct with clear field names
- `internal/config/paths.go:46-55`: `Paths` struct with self-documenting fields

### Functions

**Strengths:**
- Function names describe what they do (`loadMergedConfig`, `resolveDirectory`, `prepareExecutionEnv`)
- Boolean-returning functions read as questions where appropriate (`IsUTDValid`, `IsBinaryAvailable`, `AnyExists`)
- Getters follow Go convention without "Get" prefix (`getDefaultRole`, though this is private - acceptable)

**Example:** `internal/orchestration/template.go:195-199`:
```go
// IsUTDValid checks if UTD fields satisfy the minimum requirement.
func IsUTDValid(fields UTDFields) bool {
    return fields.File != "" || fields.Command != "" || fields.Prompt != ""
}
```

### Packages

**Strengths:**
- All package names are short, lowercase, and singular
- No stuttering (e.g., `orchestration.Executor` not `orchestration.OrchestrationExecutor`)
- Clear purpose from name alone: `config`, `shell`, `temp`, `doctor`, `detection`, `registry`

### Types

**Strengths:**
- Type names are nouns describing what they represent (`Composer`, `Executor`, `Manager`, `Runner`)
- Interface names describe capabilities (`ShellRunner`, `FileReader`)
- Enums use clear naming (`Scope`, `Status`, `TaskSource`)

**Example:** `internal/orchestration/template.go:33-40`:
```go
// ShellRunner executes shell commands and returns output.
type ShellRunner interface {
    Run(command, workingDir, shell string, timeout int) (string, error)
}
```

### Concerns

| Location | Issue | Suggestion |
|----------|-------|------------|
| `internal/cli/start.go:49-52` | `debugf` function takes `category` as separate parameter | Consider a structured logger or consistent category constants |
| `internal/cue/loader.go:136` | `mergeWithReplacement` is a long name | Acceptable given complexity, but could be `merge` if context is clear |

---

## 3. Function Design

### Strengths

**Early Returns:** Consistent use of early returns to handle errors and special cases, reducing nesting depth.

**Example:** `internal/config/paths.go:60-82`:
```go
func ResolvePaths(workingDir string) (Paths, error) {
    var p Paths

    globalPath, err := globalConfigDir()
    if err != nil {
        return p, err
    }
    p.Global = globalPath
    // ... continues with happy path
}
```

**Focused Functions:** Most functions do one thing well. Good examples:
- `internal/detection/agent.go:20-57`: `DetectAgents` has single responsibility
- `internal/temp/manager.go:89-102`: `WriteUTDFile` is concise and clear

### Concerns

| Location | Issue | Suggestion |
|----------|-------|------------|
| `internal/cli/task.go:71-298` | `executeTask` is 227 lines | Consider extracting registry lookup into separate function |
| `internal/cli/config_agent.go:153-298` | `runConfigAgentAdd` is 145 lines | Consider extracting interactive prompts into helper |
| `internal/orchestration/composer.go:115-208` | `Compose` at 93 lines is borderline | The nested helper function adds complexity; consider extraction |

### Nesting Depth

Generally excellent. Most functions stay at 2-3 levels of nesting maximum. The use of early returns keeps complexity manageable.

---

## 4. Code Flow

### Strengths

**Top-to-Bottom Reading Order:** Files are generally organised with:
1. Package comment
2. Imports
3. Constants/variables
4. Types
5. Constructors (`New...` functions)
6. Methods
7. Helper functions

**Example of good organisation:** `internal/doctor/doctor.go` - types and methods are logically grouped.

**Exported Code Positioning:** Exported symbols are generally at the top of files, with unexported helpers following.

### Concerns

| Location | Issue | Suggestion |
|----------|-------|------------|
| `internal/cli/start.go` | `Flags` struct and `flagsKey` are defined mid-file | Consider moving to top with other types |
| `internal/cli/config_agent.go` | File is 1186 lines | Consider splitting into separate files per subcommand |

---

## 5. Comments

### Strengths

**Doc Comments:** Excellent coverage on exported symbols. All exported functions, types, and constants have clear doc comments.

**Examples:**
- `internal/orchestration/executor.go:70-72`: Clear purpose explanation
- `internal/cue/loader.go:37-43`: Explains Load behaviour including edge cases
- `internal/shell/runner.go:44-46`: Documents the Run method clearly

**Design Record References:** Comments reference DRs (Design Records) where relevant, providing traceability.

**Example:** `internal/cli/start.go:45`:
```go
// Per DR-014: required contexts only, no defaults
```

### Concerns

| Location | Issue | Suggestion |
|----------|-------|------------|
| `internal/cli/config_agent.go:964-967` | Comment `// Would parse existing settings here` with unused data | Remove or implement |
| General | Some functions have brief comments; others have detailed ones | Consistency could be improved |

### Dead Commented Code

None found. The codebase is clean of commented-out code.

---

## 6. Complexity

### Strengths

**Named Intermediate Values:** Complex expressions are generally broken into named values.

**Example:** `internal/orchestration/executor.go:139-146`:
```go
data := CommandData{
    "bin":       escapeForShell(expandTilde(cfg.Agent.Bin)),
    "model":     escapeForShell(model),
    "role":      escapeForShell(cfg.Role),
    ...
}
```

**Extracted Conditions:** Boolean conditions are often extracted to named variables or functions.

### Concerns

| Location | Issue | Suggestion |
|----------|-------|------------|
| `internal/cli/task.go:99-200` | Long if-else chain for task resolution | Consider strategy pattern or state machine |
| `internal/cue/loader.go:136-260` | `mergeWithReplacement` is 124 lines with nested loops | Consider breaking into smaller functions |
| `internal/cli/show.go:491-603` | `formatShowContent` switch with nested conditions | Could be table-driven or use type-specific formatters |

---

## 7. Consistency

### Strengths

**Similar Things Done Similarly:** Across the codebase:
- All CLI commands follow the same pattern (`addXxxCommand`, `runXxx`)
- Error handling follows consistent patterns (`fmt.Errorf("verb: %w", err)`)
- Config loading follows same pattern in all commands

**Example of consistent command structure:**
- `internal/cli/prompt.go:11-24` - addPromptCommand
- `internal/cli/show.go:29-84` - addShowCommand
- `internal/cli/doctor.go:18-39` - addDoctorCommand

All follow the same pattern: create command, set properties, add to parent.

### Concerns

| Location | Issue | Suggestion |
|----------|-------|------------|
| `internal/cli/config_*.go` | Each file repeats similar loading/writing patterns | Consider shared helpers in a config_helpers.go |
| Error messages | Some use `"Error: "` prefix, some don't | Standardise error message formatting |
| `internal/orchestration/executor.go:201-207` vs `:239-268` | `Execute` and `ExecuteWithoutReplace` are similar | Consider shared implementation with flag |

---

## 8. Magic Values

### Strengths

**Named Constants:** Magic values are generally replaced with named constants.

**Examples:**
- `internal/shell/runner.go:15`: `DefaultTimeout = 30`
- `internal/cli/task.go:39`: `maxTaskResults = 20`
- `internal/cli/output.go:50`: Separator line width (though inline)

### Concerns

| Location | Issue | Suggestion |
|----------|-------|------------|
| `internal/cli/output.go:50` | `strings.Repeat("─", 79)` hardcoded | Define `const separatorWidth = 79` |
| `internal/cli/config_agent.go:267` | `0755` file permission | Consider named constant for clarity |
| `internal/temp/manager.go:52,69,79,97` | `0755` and `0600` permissions repeated | Define package-level constants |
| `internal/orchestration/executor.go:21-22` | Regex patterns inline | Consider package-level named patterns with comments |

---

## 9. Dead Code

### Findings

The codebase appears free of dead code. No unreachable code paths or unused functions were identified.

| Location | Observation |
|----------|-------------|
| `internal/cli/start.go:334-338` | `loadMergedConfig()` appears to be unused (only `loadMergedConfigFromDir` used) | Verify and remove if unused |
| `internal/cli/config_agent.go:964-967` | Unused variable `data` with TODO comment | Complete or remove |

---

## 10. Test Readability

### Strengths

**Table-Driven Tests:** Consistent use of table-driven tests throughout.

**Example:** `internal/cli/start_test.go:132-171`:
```go
tests := []struct {
    name        string
    selection   orchestration.ContextSelection
    wantContext string
}{
    {
        name: "required and default",
        selection: orchestration.ContextSelection{
            IncludeRequired: true,
            IncludeDefaults: true,
        },
        wantContext: "env",
    },
    ...
}
```

**Helper Functions:** Test setup is extracted to helpers with `t.Helper()` calls.

**Example:** `internal/cli/start_test.go:16-73` - `setupStartTestConfig`

**Clear Test Names:** Tests follow `Test<Function>_<Scenario>` naming convention.

### Concerns

| Location | Issue | Suggestion |
|----------|-------|------------|
| `internal/cli/start_test.go` | Multiple tests change working directory | Consider table-driven approach to reduce duplication |
| Some test files | Tests over 100 lines | Consider extracting common setup to `TestMain` or shared fixtures |

---

## Patterns That Work Well

1. **Factory Functions:** `New...` functions create properly initialised instances
2. **Interface Injection:** `ShellRunner` and `FileReader` interfaces enable testing
3. **Command Pattern:** All CLI commands follow consistent structure
4. **Result Types:** `LoadResult`, `ProcessResult`, `ComposeResult` bundle related return values
5. **Scope Enum:** `Scope` type with `String()` and `ParseScope()` methods

---

## Recommendations

### High Priority

1. **Extract long functions:** Break down `executeTask` (227 lines) and `runConfigAgentAdd` (145 lines) into smaller, focused functions.

2. **Consolidate config helpers:** Create `internal/cli/config_helpers.go` for shared patterns across config_agent.go, config_role.go, etc.

### Medium Priority

3. **Define permission constants:** Create `internal/config/permissions.go`:
   ```go
   const (
       DirPerm  = 0755
       FilePerm = 0644
       PrivatePerm = 0600
   )
   ```

4. **Standardise error messages:** Choose consistent format (with or without capitalisation, with or without trailing punctuation).

### Low Priority

5. **Consider splitting large files:** `internal/cli/config_agent.go` at 1186 lines could be split by subcommand.

6. **Document complex algorithms:** `mergeWithReplacement` in `internal/cue/loader.go` would benefit from a block comment explaining the merge strategy.

---

## Conclusion

This codebase demonstrates strong readability practices. A new team member could understand the code structure and patterns within a reasonable timeframe. The consistent use of Go idioms, clear naming, and well-organised packages make navigation intuitive.

The main areas for improvement are around function length in a few key files and some minor inconsistencies in error handling patterns. These are refinements rather than fundamental issues.

**Overall Assessment:** The code reads like well-written prose in most places, with occasional sections that require closer reading due to complexity or length.

---

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Google Go Style Guide](https://google.github.io/styleguide/go/)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
