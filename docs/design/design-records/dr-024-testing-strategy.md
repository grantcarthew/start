# DR-024: Testing Strategy

- Date: 2025-12-11
- Status: Accepted
- Category: Testing

## Problem

The project needs a clear testing strategy that:

- Ensures code is testable from the start
- Avoids excessive mocking that obscures real behaviour
- Provides confidence in CUE integration and CLI functionality
- Supports fast iteration during development
- Enables comprehensive validation before releases

Without explicit guidance, testing approaches become inconsistent and code may be written in ways that are difficult to test.

## Decision

Adopt a three-tier testing strategy using Go's standard testing framework with build tags to separate test types. Prioritise testing real behaviour over mocking. Use a Bash script (`scripts/invoke-tests`) as the single entry point for running tests.

## Test Types

Unit Tests:

- Location: `*_test.go` alongside source files
- Purpose: Test pure functions, logic, and isolated components
- Speed: Fast (milliseconds)
- Build tag: None (default)
- Run: `go test ./...`

Integration Tests:

- Location: `test/integration/*_test.go`
- Purpose: Test component interactions (CLI + CUE loader + validator)
- Speed: Medium (seconds)
- Build tag: `//go:build integration`
- Run: `go test -tags=integration ./...`

E2E Tests:

- Location: `test/e2e/*_test.go`
- Purpose: Test compiled binary end-to-end
- Speed: Slow (seconds to minutes)
- Build tag: `//go:build e2e`
- Prerequisite: Binary must be built first
- Run: `go test -tags=e2e ./...`

## Directory Structure

```
start/
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   ├── root_test.go
│   │   ├── init.go
│   │   ├── init_test.go
│   │   └── ...
│   ├── cue/
│   │   ├── loader.go
│   │   ├── loader_test.go
│   │   └── ...
│   └── config/
│       └── ...
├── test/
│   ├── integration/
│   │   ├── init_test.go
│   │   └── show_test.go
│   ├── e2e/
│   │   └── cli_test.go
│   └── testdata/
│       ├── valid/
│       │   └── basic.cue
│       └── invalid/
│           └── bad_syntax.cue
└── scripts/
    └── invoke-tests
```

## Testing Patterns

Cobra Command Testing:

Use Cobra's built-in approach to test commands programmatically:

```go
func executeCommand(root *cobra.Command, args ...string) (string, error) {
    buf := new(bytes.Buffer)
    root.SetOut(buf)
    root.SetErr(buf)
    root.SetArgs(args)
    err := root.Execute()
    return buf.String(), err
}
```

Table-Driven Tests:

Standard Go pattern for testing multiple cases:

```go
tests := []struct {
    name    string
    input   string
    wantErr bool
}{
    {"valid input", "good", false},
    {"invalid input", "bad", true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic
    })
}
```

Real File System:

Use `t.TempDir()` for file operations:

```go
func TestLoader_ValidCUE(t *testing.T) {
    tmpDir := t.TempDir()
    // Write real CUE files, test real loading
}
```

Interactive Input Testing:

Inject `io.Reader`/`io.Writer` for stdin/stdout:

```go
type Command struct {
    In  io.Reader
    Out io.Writer
}
```

E2E Binary Testing:

Test the compiled binary directly:

```go
func TestE2E_StartInit(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test in short mode")
    }

    cmd := exec.Command("./bin/start", "init")
    cmd.Dir = t.TempDir()
    output, err := cmd.CombinedOutput()
    // assertions
}
```

## Mocking Guidelines

Do Not Mock:

- CUE library - Use real CUE validation (fast and deterministic)
- File system - Use `t.TempDir()` for real files
- Time - Unless testing time-based logic specifically

Acceptable to Mock:

- External network calls (HTTP clients for registry access)
- System command detection (`exec.LookPath` via interface)
- Environment variables (override via `cmd.Env`)

## Test Script

The `scripts/invoke-tests` script is the single entry point for running tests. It handles building the binary for e2e tests and provides consistent options.

Usage:

```bash
./scripts/invoke-tests      # Unit tests only
./scripts/invoke-tests -i   # Unit + integration
./scripts/invoke-tests -e   # Unit + integration + e2e
./scripts/invoke-tests -a   # All tests (same as -e)
./scripts/invoke-tests -c   # With coverage report
./scripts/invoke-tests -v   # Verbose output
./scripts/invoke-tests -s   # Short mode (skip long tests)
```

## Why

Real Behaviour Over Mocks:

- Mocks can hide integration bugs
- CUE validation is fast enough to use directly
- Real file operations via `t.TempDir()` are reliable and fast
- Tests that use real components catch more bugs

Build Tags for Separation:

- Unit tests run fast by default
- Integration and e2e tests opt-in via tags
- CI can run different test levels at different stages

Single Script Entry Point:

- Consistent interface for developers and CI
- Handles build prerequisites for e2e tests
- No Makefile dependency

Testable Design:

- Functions accept interfaces/parameters rather than globals
- IO can be injected for interactive components
- Commands can be tested without running the binary

## Trade-offs

Accept:

- E2E tests require building the binary first
- Some test setup code for file fixtures
- Build tags add slight complexity

Gain:

- High confidence in real behaviour
- Fast unit test iteration
- Comprehensive e2e validation
- Clear separation of test concerns
- No mock maintenance burden

## Alternatives

Makefile:

- Pro: Common convention
- Con: Additional dependency and syntax
- Rejected: Bash script is simpler and sufficient

Heavy Mocking:

- Pro: Tests run faster, more isolated
- Con: Mocks drift from real behaviour, hide bugs
- Rejected: Real components are fast enough

Single Test Binary:

- Pro: Simpler build
- Con: All tests run together, slower feedback
- Rejected: Build tags provide better control

## Coverage Goals

- Internal packages (`internal/cue/`, `internal/config/`): 80%+ coverage
- CLI commands (`internal/cli/`): 70%+ coverage
- Overall: Focus on meaningful coverage, not percentage targets
