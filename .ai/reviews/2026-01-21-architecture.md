# Architecture Review: start

**Date:** 2026-01-21
**Reviewer:** Claude Opus 4.5
**Version:** Based on commit a7f3b6f (feat(cli): add 'tasks' alias for task command)

## Executive Summary

The `start` project demonstrates a well-structured Go codebase with strong adherence to idiomatic Go practices. The architecture follows a clean separation of concerns with proper layering, minimal global state, and thoughtful interface design. The project benefits from extensive design documentation (38 Design Records) that capture architectural decisions and their rationale.

**Overall Assessment:** The architecture is sound, maintainable, and well-suited for its purpose as a CLI orchestrator for AI agents. Minor refinements are suggested but no significant architectural issues were identified.

---

## 1. Package Structure

### Current Structure

```
cmd/start/           # Entry point (18 lines)
internal/
├── cli/             # Command implementations (22 files)
├── config/          # Path resolution and validation (4 files)
├── cue/             # CUE loading, validation, error formatting (7 files)
├── detection/       # Agent binary detection (2 files)
├── doctor/          # Health check diagnostics (5 files)
├── orchestration/   # Core engine - composition, execution, templates (10 files)
├── registry/        # CUE Central Registry client (4 files)
├── shell/           # Shell command execution (4 files)
└── temp/            # Temporary file management (2 files)
test/
├── e2e/             # End-to-end tests
├── integration/     # Integration tests
└── testdata/        # Test fixtures
```

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Idiomatic layout | **Pass** | Uses `cmd/` for entry point, `internal/` for private packages |
| Package names | **Pass** | Short, lowercase, singular nouns |
| Cohesive concepts | **Pass** | Each package has a clear, single responsibility |
| Minimal main | **Pass** | `main.go` is 18 lines - just wiring and error handling |
| No prohibited names | **Pass** | No `util`, `common`, `misc`, or `helpers` packages |

### Strengths

1. **Clean `cmd/start/main.go`**: Entry point only handles error formatting and exit codes - all logic delegated to `cli.Execute()`.

2. **Consistent package sizing**: Packages contain between 2-22 files, with no god packages or overly granular splitting.

3. **Internal isolation**: All packages properly scoped under `internal/`, preventing external imports.

---

## 2. Dependency Flow

### Package Dependencies

```
cmd/start
    └── internal/cli

internal/cli
    ├── internal/config
    ├── internal/cue
    ├── internal/orchestration
    ├── internal/registry
    ├── internal/shell
    └── internal/temp

internal/orchestration
    ├── internal/config
    ├── internal/cue
    ├── internal/detection
    ├── internal/registry
    └── internal/temp

internal/config
    └── internal/cue

internal/doctor
    ├── internal/config
    └── internal/cue

internal/detection
    └── internal/registry
```

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Unidirectional flow | **Pass** | Dependencies flow downward, no cycles detected |
| Shallow graph | **Pass** | Maximum depth of 3 levels |
| External isolation | **Pass** | External dependencies (cobra, cue, color) isolated to specific packages |

### Observations

1. **No circular dependencies**: The `go mod graph` and package structure confirm clean, acyclic dependencies.

2. **Clear layering**:
   - **Presentation**: `cli/` (Cobra commands, output formatting)
   - **Business Logic**: `orchestration/`, `doctor/`, `detection/`
   - **Infrastructure**: `cue/`, `config/`, `registry/`, `shell/`, `temp/`

3. **External dependency isolation**: CUE library usage is concentrated in `internal/cue/`, `internal/orchestration/`, and `internal/registry/`, with type leakage limited to `cue.Value` in orchestration.

---

## 3. Interface Design

### Defined Interfaces

```go
// internal/orchestration/template.go
type ShellRunner interface {
    Run(command, workingDir, shell string, timeout int) (string, error)
}

type FileReader interface {
    Read(path string) (string, error)
}
```

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Small and focused | **Pass** | Interfaces have 1 method each |
| Consumer-defined | **Pass** | Defined in `orchestration/` where consumed |
| Accept interfaces, return structs | **Pass** | Constructors return concrete types |
| Discovered from usage | **Pass** | Interfaces match actual usage patterns |

### Strengths

1. **Minimal interface surface**: Only two interfaces are defined, both single-method, avoiding interface pollution.

2. **Proper location**: Interfaces are defined where they are consumed (`orchestration/`), not where implemented (`shell/`).

3. **Optional implementations**: `NewTemplateProcessor` accepts `nil` for `FileReader`, defaulting to `DefaultFileReader`, enabling easy testing.

### Potential Enhancement

The `shell.Runner` type could implement the `ShellRunner` interface explicitly with a type assertion at compile time to ensure compatibility:

```go
var _ orchestration.ShellRunner = (*Runner)(nil)
```

This is a minor enhancement and not a deficiency.

---

## 4. API Design

### Exported APIs

The codebase follows a minimal export philosophy. Key exported types per package:

| Package | Key Exports |
|---------|-------------|
| `cli` | `Execute()`, `NewRootCmd()`, `Flags` |
| `config` | `Paths`, `Scope`, `ResolvePaths()`, `ValidateConfig()` |
| `cue` | `Loader`, `Validator`, `ValidationError`, key constants |
| `orchestration` | `Composer`, `Executor`, `Agent`, `ContextSelection`, `*Result` types |
| `registry` | `Client`, `Index`, `IndexEntry` |
| `doctor` | `Report`, `SectionResult`, `CheckResult`, `Status` |

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Minimal exports | **Pass** | Only necessary types/functions exported |
| Clear signatures | **Pass** | Function purposes evident from signatures |
| Structs from constructors | **Pass** | `NewLoader()`, `NewValidator()`, etc. return concrete types |
| Options pattern | **Pass** | Used for `Validator.Validate(...ValidationOption)` |

### Strengths

1. **Functional options**: `ValidationOption` type provides clean optional configuration without parameter bloat.

2. **Result types**: Composition results use dedicated structs (`ComposeResult`, `ProcessResult`, `LoadResult`) with clear semantics.

3. **Flags scoping**: CLI flags are instance-scoped (`Flags` struct per command), enabling parallel test execution.

---

## 5. Coupling and Cohesion

### Package Cohesion Analysis

| Package | Cohesion | Responsibility |
|---------|----------|----------------|
| `config` | **High** | Path resolution and configuration validation only |
| `cue` | **High** | CUE loading, validation, error formatting |
| `shell` | **High** | Shell detection and command execution |
| `temp` | **High** | Temporary file management |
| `detection` | **High** | Agent binary detection in PATH |
| `registry` | **High** | CUE Central Registry interaction |
| `doctor` | **High** | Health check infrastructure |
| `orchestration` | **Medium-High** | Core engine (composition, execution, templates, auto-setup) |
| `cli` | **Medium** | Command implementations - necessarily broader |

### Coupling Assessment

1. **Low coupling between infrastructure packages**: `shell/`, `temp/`, `detection/` have minimal dependencies.

2. **Moderate coupling in orchestration**: The `orchestration/` package imports several internal packages, but this is appropriate for its role as the core engine.

3. **CLI package coupling**: `cli/` imports most internal packages, which is expected for a CLI package that orchestrates features.

### Observations

The `orchestration/` package handles multiple concerns:
- Template processing (`template.go`)
- Prompt composition (`composer.go`)
- Agent execution (`executor.go`)
- Auto-setup flow (`autosetup.go`)
- File path handling (`filepath.go`)

While cohesive around "orchestrating AI agent execution", the auto-setup functionality could potentially be extracted if the package grows further. Currently, this is not a concern.

---

## 6. Layering

### Architecture Layers

```
┌─────────────────────────────────────────────────────────┐
│                    CLI Layer (cli/)                      │
│  Commands, flags, output formatting, user interaction    │
├─────────────────────────────────────────────────────────┤
│                 Business Logic Layer                     │
│  orchestration/  │  doctor/  │  detection/               │
│  Composition, execution, diagnostics, agent detection    │
├─────────────────────────────────────────────────────────┤
│                Infrastructure Layer                      │
│  cue/  │  config/  │  registry/  │  shell/  │  temp/    │
│  CUE handling, paths, registry, shell exec, temp files   │
└─────────────────────────────────────────────────────────┘
```

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Concerns separated | **Pass** | Clear separation between presentation, logic, infrastructure |
| Logic testable without infrastructure | **Pass** | Interfaces enable mocking shell, file I/O |
| Configuration separate from logic | **Pass** | `config/` handles paths, logic in `orchestration/` |
| Side effects isolated | **Pass** | Process execution, file I/O in dedicated packages |

### Strengths

1. **DR-026 compliance**: The codebase follows the documented Logic/IO separation pattern:
   - Logic functions return data (no I/O)
   - Output functions accept `io.Writer` and data
   - Command functions wire Cobra to logic and output

2. **Testable design**: Business logic can be tested without file system or network by injecting mock implementations.

---

## 7. Error Handling Architecture

### Patterns Observed

1. **Wrapped errors with context**:
   ```go
   return fmt.Errorf("loading %s: %w", dir, err)
   ```

2. **Dedicated error types**:
   ```go
   type ValidationError struct {
       Path, Message string
       Line, Column  int
       Filename      string
       Context       string
   }
   ```

3. **Error formatting with source context**: `FormatErrorWithContext()` provides helpful error messages with file snippets.

4. **Warnings as non-fatal**: `ComposeResult.Warnings` collects issues that don't prevent execution.

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Consistent wrapping | **Pass** | Errors wrapped with context throughout |
| Domain errors defined locally | **Pass** | `ValidationError` in `cue/`, domain-specific |
| Appropriate handling levels | **Pass** | Errors bubble up, handled at CLI layer |
| User-friendly messages | **Pass** | `DetailedError()` provides actionable information |

### Strengths

1. **Source context in errors**: When CUE validation fails, users see the relevant lines with column pointers.

2. **Separation of error types**: `ValidationError` has both `Error()` (concise) and `DetailedError()` (verbose) methods.

---

## 8. Configuration Architecture

### Design

- **Global config**: `~/.config/start/` (XDG-compliant)
- **Local config**: `./.start/`
- **Merge semantics**: Local overrides global (per DR-025)
- **Format**: CUE files with schema validation

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Loaded at startup | **Pass** | Configuration loaded once, passed through context |
| Sensible defaults | **Pass** | Auto-setup creates minimal working config |
| Validated at load | **Pass** | CUE validation on load, detailed error reporting |
| Structure documented | **Pass** | DR-025, DR-030 document merge semantics and settings |

### Strengths

1. **Two-level merge semantics**: Collections (agents, tasks, etc.) merge at item level; settings merge at field level.

2. **Auto-setup flow**: First run detects installed agents and creates minimal configuration.

3. **Validation with recovery**: Invalid config produces actionable error messages with source context.

---

## 9. Concurrency Architecture

### Patterns Observed

1. **Parallel agent detection**: `detection.DetectAgents()` uses goroutines with `sync.Mutex` for concurrent PATH lookups.

2. **Context propagation**: `context.Context` passed through registry operations and command execution.

3. **Process replacement**: `syscall.Exec` replaces process for clean agent handoff (Unix-only, per DR-006).

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Model documented | **Pass** | DR-006 documents platform scope and process model |
| Goroutine lifetimes defined | **Pass** | Used only for parallel detection with WaitGroup |
| Context propagation | **Pass** | Registry operations respect context cancellation |
| Cancellation boundaries | **Pass** | Timeouts on shell commands via context |

---

## 10. Extensibility

### Extension Points

1. **Interfaces**: `ShellRunner`, `FileReader` allow alternative implementations.

2. **CUE schemas**: Configuration format is schema-validated, enabling extension via CUE inheritance.

3. **Registry-driven assets**: Agents, roles, tasks, contexts can be distributed via CUE Central Registry.

4. **Command structure**: Cobra's subcommand model allows adding new commands without modifying existing ones.

### Assessment

| Criterion | Status | Notes |
|-----------|--------|-------|
| Add without modify | **Pass** | New commands, agents, tasks added via configuration |
| Extension points defined | **Pass** | Interfaces, registry, CUE schemas |
| Open/closed principle | **Pass** | Core code stable, extension via configuration |

---

## 11. Anti-Pattern Audit

### Checked Anti-Patterns

| Anti-Pattern | Status | Notes |
|--------------|--------|-------|
| Circular dependencies | **Not found** | Clean dependency graph |
| `util`/`common`/`misc` packages | **Not found** | All packages have clear names |
| Large `init()` functions | **Not found** | No `init()` functions in `internal/` |
| Global mutable state | **Minimal** | Only compile-time constants (regex patterns, keys) |
| Deep package hierarchies | **Not found** | Maximum 2 levels (`internal/cli/`) |
| Import-heavy packages | **Acceptable** | `cli/` and `orchestration/` import appropriately for their roles |

### Package-Level Globals

Six package-level `var` declarations found, all appropriate:

1. `executor.go`: Compiled regex patterns (immutable after init)
2. `loader.go`: Constant map of collection keys
3. `root.go`: Version template string
4. `doctor.go`: Sentinel error type
5. `config_settings.go`: Valid settings keys map

All are effectively constants or compile-time values - no mutable global state.

---

## 12. Testing Architecture

### Test Organisation

- **Unit tests**: `*_test.go` alongside source files
- **Integration tests**: `test/integration/` with `//go:build integration`
- **E2E tests**: `test/e2e/` with `//go:build e2e`
- **Test fixtures**: `test/testdata/`

### Testing Patterns

1. **Table-driven tests**: Standard Go pattern used throughout
2. **Real file system**: `t.TempDir()` for file operations
3. **Parallel execution**: `t.Parallel()` in tests
4. **Command testing**: Cobra's `SetOut()`, `SetErr()`, `SetArgs()` pattern
5. **Minimal mocking**: Interfaces only where needed (shell, files)

### Assessment

The testing strategy (per DR-024) prioritises:
- Real behaviour over mocks
- Real CUE validation
- Actual file operations via `t.TempDir()`

This is a sound approach that catches integration issues early.

---

## 13. Documentation

### Design Documentation

The `.ai/design/design-records/` directory contains 38 Design Records documenting:
- Schema designs (DR-008 through DR-011, DR-030, DR-037)
- CLI commands (DR-012 through DR-018, DR-028, DR-029, DR-031-036)
- Testing strategy (DR-024)
- Merge semantics (DR-025)
- I/O separation (DR-026)
- File path handling (DR-038)

### Assessment

The design documentation is thorough and provides valuable context for architectural decisions. Key DRs are referenced in code comments where appropriate.

---

## 14. Recommendations

### Minor Improvements (Optional)

1. **Interface compile-time check**: Add explicit interface satisfaction assertions:
   ```go
   var _ orchestration.ShellRunner = (*shell.Runner)(nil)
   ```

2. **Package documentation**: Add package-level comments to some packages that lack them (e.g., `detection/`).

3. **Consider extracting auto-setup**: If `orchestration/` grows further, the auto-setup flow could become its own package, but this is not currently necessary.

### Observations (Not Issues)

1. **CLI package size**: The `cli/` package has 22 files, which is larger than others. This is acceptable for a CLI package that implements multiple commands, though it could potentially be split by command grouping in the future if it grows significantly.

2. **CUE value leakage**: `cue.Value` appears in public APIs of `orchestration/`. This is pragmatic given the CUE-centric nature of the project, though it does create tight coupling to CUE types.

---

## 15. Conclusion

The `start` codebase demonstrates mature Go engineering practices:

- **Idiomatic structure**: Follows Go project layout conventions
- **Clean architecture**: Clear separation of concerns with proper layering
- **Minimal global state**: No mutable globals, no `init()` functions
- **Thoughtful interface design**: Small, focused interfaces defined at consumption point
- **Comprehensive testing strategy**: Three-tier approach with emphasis on real behaviour
- **Excellent documentation**: 38 Design Records capture rationale for decisions

The architecture is well-suited for its purpose as a CLI orchestrator. The codebase is maintainable, testable, and ready for continued development.

### Architectural Fitness Score

| Category | Score | Notes |
|----------|-------|-------|
| Package Structure | 5/5 | Idiomatic, well-organised |
| Dependency Management | 5/5 | Clean flow, no cycles |
| Interface Design | 5/5 | Minimal, properly placed |
| API Design | 5/5 | Clear, consistent |
| Coupling/Cohesion | 4/5 | Minor: CLI package size |
| Layering | 5/5 | Clear separation |
| Error Handling | 5/5 | Consistent, user-friendly |
| Configuration | 5/5 | Validated, documented |
| Testability | 5/5 | Designed for testing |
| **Overall** | **49/50** | **Excellent** |

---

*Report generated by Claude Opus 4.5*
