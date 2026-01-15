# Architecture Review: start

**Review Date:** 2026-01-15
**Reviewer:** Claude Opus 4.5
**Repository:** github.com/grantcarthew/start

## Executive Summary

The `start` project demonstrates **excellent architectural fitness** for a CLI tool of its scope. The codebase follows idiomatic Go patterns, maintains clear separation of concerns, and exhibits thoughtful design decisions documented through Design Records. The architecture is well-suited for the project's purpose as an AI agent CLI orchestrator built on CUE.

**Overall Rating:** Strong

---

## 1. Package Structure

### Assessment: Excellent

The package layout follows idiomatic Go conventions:

```
start/
├── cmd/start/          # Minimal main.go (entry point only)
├── internal/           # Private packages
│   ├── cli/            # Cobra command definitions and CLI logic
│   ├── config/         # Configuration path resolution
│   ├── cue/            # CUE loading and validation
│   ├── detection/      # Agent binary detection
│   ├── doctor/         # Health check diagnostics
│   ├── orchestration/  # Core business logic (composition, execution)
│   ├── registry/       # CUE Central Registry client
│   ├── shell/          # Shell command execution
│   └── temp/           # Temporary file management
└── test/               # Integration and E2E tests
```

**Strengths:**

- `cmd/start/main.go` is minimal (18 lines) - pure wiring, no business logic
- All packages under `internal/` - correctly scoped for a CLI tool
- No `pkg/` directory - appropriate since nothing is intended for external consumption
- Package names are short, lowercase, singular nouns
- Each package represents a cohesive concept

**No issues identified.**

---

## 2. Dependencies

### Assessment: Excellent

**Internal dependency flow (from `go list ./...` and code inspection):**

```
cmd/start/main.go
    └── internal/cli
            ├── internal/config
            ├── internal/cue
            ├── internal/orchestration
            │       ├── internal/cue (keys only)
            │       ├── internal/shell
            │       └── internal/temp
            ├── internal/shell
            └── internal/temp

internal/detection
    └── internal/registry

internal/doctor
    └── (no internal dependencies)
```

**Strengths:**

- Unidirectional dependency flow - no cycles detected
- Shallow dependency graph - most packages are 1-2 levels deep
- High-level packages (`cli`) depend on lower-level packages (`config`, `cue`, `orchestration`)
- Core business logic (`orchestration`) has minimal dependencies
- External dependencies are isolated (CUE SDK, Cobra)

**External dependencies (from go.mod):**

| Dependency | Purpose | Assessment |
|------------|---------|------------|
| `cuelang.org/go` | CUE language SDK | Essential, well-isolated |
| `github.com/spf13/cobra` | CLI framework | Industry standard |
| `github.com/fatih/color` | Terminal colours | Minimal, UI-only |
| `golang.org/x/term` | Terminal detection | Standard library extension |

**No issues identified.** Dependency set is minimal and appropriate.

---

## 3. Interface Design

### Assessment: Good

**Interfaces found:**

```go
// internal/orchestration/template.go
type ShellRunner interface {
    Run(command, workingDir, shell string, timeout int) (string, error)
}

type FileReader interface {
    Read(path string) (string, error)
}
```

**Strengths:**

- Interfaces are small (1 method each) - follows Go idiom
- Interfaces are defined where consumed (`orchestration`), not where implemented (`shell`)
- Follows "accept interfaces, return structs" pattern
- Used for testability - allows dependency injection

**Observations:**

- Limited interface usage across the codebase - most dependencies are concrete types
- This is acceptable for a CLI tool where most components are wired once at startup
- The `io.Writer` pattern is used appropriately for output testability

**Suggestion:** Consider defining a `Loader` interface in the `cue` package if multiple implementations become needed. Currently acceptable as-is.

---

## 4. API Design

### Assessment: Excellent

**Public APIs within internal packages:**

The CLI package exposes a single entry point:

```go
// internal/cli/root.go
func Execute() error           // Main entry point
func NewRootCmd() *cobra.Command  // Factory for testing
```

**Constructors return concrete types:**

```go
func NewLoader() *Loader
func NewComposer(processor *TemplateProcessor, workingDir string) *Composer
func NewExecutor(workingDir string) *Executor
func NewRunner() *Runner
func NewClient() (*Client, error)
```

**Strengths:**

- Minimal exported surface - only what's needed
- Consistent constructor patterns (`NewXxx`)
- Factory pattern (`NewRootCmd`) enables test isolation
- Clear, consistent function signatures

**No issues identified.**

---

## 5. Coupling and Cohesion

### Assessment: Excellent

| Package | Cohesion | Coupling | File Count |
|---------|----------|----------|------------|
| `cli` | High | Medium (uses orchestration, config, cue) | 21 files |
| `config` | High | Low (standalone) | 4 files |
| `cue` | High | Low (CUE SDK only) | 7 files |
| `orchestration` | High | Medium (uses cue keys, shell, temp) | 8 files |
| `detection` | High | Low (uses registry types) | 2 files |
| `doctor` | High | Low (standalone) | 5 files |
| `registry` | High | Low (CUE SDK only) | 4 files |
| `shell` | High | Low (standalone) | 4 files |
| `temp` | High | Low (standalone) | 2 files |

**Strengths:**

- Each package has a single, clear responsibility
- Changes are well-localised - modifying `shell` doesn't affect `doctor`
- No "god packages" - even `cli` (21 files) is appropriately scoped for command definitions
- No packages with only 1 file (would indicate over-granularity)

**No issues identified.**

---

## 6. Layering

### Assessment: Excellent

The architecture follows a clean layered approach:

```
┌─────────────────────────────────────────────────────────┐
│                     CLI Layer                           │
│  (internal/cli - Cobra commands, flag handling, I/O)   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                 Orchestration Layer                     │
│  (internal/orchestration - business logic, composition)│
└─────────────────────────────────────────────────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
┌────────────────┐ ┌────────────┐ ┌────────────┐
│  Config Layer  │ │ CUE Layer  │ │ Shell Layer│
│ (path resolve) │ │ (loading)  │ │ (execution)│
└────────────────┘ └────────────┘ └────────────┘
```

**Strengths:**

- Clear separation: transport (CLI), business logic (orchestration), infrastructure (shell, cue)
- Business logic can be tested without CLI infrastructure (via `prepareExecutionEnv`, `Compose`)
- Configuration is separate from logic (DR-026 explicitly addresses this)
- Side effects isolated in `Executor.Execute()` which calls `syscall.Exec`

**The architecture aligns with Hexagonal Architecture principles:**

- Core logic in `orchestration`
- Adapters for external systems: `shell` (OS), `registry` (network), `cue` (file system)
- Dependencies flow inward toward business logic

---

## 7. Error Handling Architecture

### Assessment: Excellent

**Consistent patterns observed:**

```go
// Errors wrapped with context
return result, fmt.Errorf("loading %s: %w", dir, err)

// Errors handled at appropriate levels
if err := v.Err(); err != nil {
    return cue.Value{}, fmt.Errorf("building instance: %w", err)
}

// Sentinel errors not over-used - raw errors preferred
```

**Design Records document error philosophy:**

- DR-007: UTD error handling - non-fatal errors stored as warnings
- DR-026: CLI logic and I/O separation - errors returned, not printed

**Strengths:**

- Consistent `fmt.Errorf("context: %w", err)` wrapping pattern
- Errors bubble up to CLI layer for user-facing output
- Main function handles top-level error display with colour
- Warnings vs errors distinction maintained (non-fatal issues don't abort)

**No issues identified.**

---

## 8. Concurrency Architecture

### Assessment: Good

**Concurrency usage is minimal but appropriate:**

```go
// internal/detection/agent.go - parallel binary detection
func DetectAgents(index *registry.Index) []DetectedAgent {
    var (
        mu       sync.Mutex
        detected []DetectedAgent
        wg       sync.WaitGroup
    )
    for key, entry := range index.Agents {
        wg.Add(1)
        go func(k string, e registry.IndexEntry) {
            defer wg.Done()
            // ... detect logic
            mu.Lock()
            detected = append(detected, ...)
            mu.Unlock()
        }(key, entry)
    }
    wg.Wait()
    return detected
}
```

**Strengths:**

- Goroutine lifetimes are well-defined (bounded by `WaitGroup`)
- Context propagation used where appropriate (registry client)
- Simple sync.Mutex for result aggregation - appropriate pattern

**Observation:** The codebase is primarily synchronous, which is appropriate for a CLI tool. Concurrency is used sparingly and correctly where it provides benefit (parallel agent detection).

---

## 9. Configuration

### Assessment: Excellent

**Configuration loading:**

```go
// internal/config/paths.go
func ResolvePaths(workingDir string) (Paths, error)
func globalConfigDir() (string, error)  // XDG_CONFIG_HOME aware

// internal/cue/loader.go
func (l *Loader) Load(dirs []string) (LoadResult, error)
```

**Strengths:**

- Configuration loaded once at startup via `loadMergedConfig`
- XDG Base Directory Specification respected
- Sensible defaults (global config in `~/.config/start/`)
- Validation at load time via `config.ValidateConfig()`
- Configuration structure documented through Design Records and CUE schemas

**Configuration merge semantics documented in DR-025:**

- Two-level merge: collection items replaced, fields merged
- Predictable behaviour for global + local config

---

## 10. Extensibility

### Assessment: Good

**Extension points:**

1. **Agents** - New AI agents added via CUE configuration, not code changes
2. **Tasks/Roles/Contexts** - CUE modules from registry
3. **Shell runners** - Interface allows alternative implementations
4. **File readers** - Interface allows alternative implementations

**Strengths:**

- CUE schemas enable extension without code modification
- Registry-based distribution for community packages
- Interface-based design for testability doubles as extension points

**Observation:** The architecture prioritises configuration-based extension (CUE) over code-based extension, which is appropriate for this domain.

---

## Anti-Patterns Audit

| Anti-Pattern | Status |
|--------------|--------|
| Circular dependencies | None detected |
| `util/common/misc/helpers` packages | None present |
| Large `init()` functions | None observed |
| Global mutable state | Minimal - only `color.NoColor` for terminal output |
| Deep package hierarchies (>3 levels) | Maximum 1 level (`internal/xxx`) |
| Packages importing most of codebase | None - `cli` imports most but it's the entry point |

---

## Structural Concerns

### Minor Observations

1. **CLI package size (21 files):** While large, the files are logically grouped by command. This is acceptable for a CLI tool. If growth continues, consider sub-packages per command group.

2. **Flags struct scope:** The `Flags` struct in `start.go` is coupled to many commands. Current design uses context-based propagation which works well.

3. **CUE key constants:** The `internal/cue/keys.go` file centralises string constants, which is good practice but creates a minor coupling point.

### None Critical

No architectural concerns requiring immediate attention.

---

## Recommendations

### Keep Doing

1. **Design Records** - Continue documenting architectural decisions before implementation
2. **Testing strategy** - Real behaviour over mocks approach is working well
3. **Package cohesion** - Each package maintains single responsibility
4. **Interface minimalism** - Small interfaces defined where consumed

### Consider

1. **CLI subpackages:** If the `cli` package grows beyond ~30 files, consider grouping commands (e.g., `cli/config/`, `cli/assets/`). Not urgent.

2. **Metrics/Tracing:** If debugging production issues becomes important, consider structured logging. Current `debugf` approach is simple and effective for CLI use.

3. **Error types:** If error handling becomes more sophisticated, consider defining domain error types in the `orchestration` package. Current `fmt.Errorf` wrapping is sufficient.

---

## Key Questions Answered

| Question | Assessment |
|----------|------------|
| Does a new developer understand the structure quickly? | Yes - clear package boundaries, good naming |
| Can components be tested in isolation? | Yes - interfaces where needed, `t.TempDir()` for files |
| Can the system evolve without major rewrites? | Yes - CUE-based extension, clean layering |
| Are package boundaries at natural domain boundaries? | Yes - config, cue, orchestration, shell all represent distinct concerns |

---

## Conclusion

The `start` codebase exhibits **strong architectural design** that is:

- **Idiomatic** - Follows Go conventions and community patterns
- **Testable** - Design decisions documented in DR-024/DR-026 enable effective testing
- **Maintainable** - Clear package boundaries, minimal coupling
- **Extensible** - CUE-based configuration allows growth without code changes

The architecture is well-suited for the project's current scope and has headroom for future growth. No refactoring is recommended at this time.
