# Testing Strategy

Three-tier testing using Go's standard testing framework with build tags. Prioritise real behaviour over mocking. Use `scripts/invoke-tests` as the single entry point.

## Test Types

Unit tests:

- Location: `*_test.go` alongside source files
- Purpose: Pure functions, logic, isolated components
- Build tag: None (default)
- Run: `go test ./...`

Integration tests:

- Location: `test/integration/*_test.go`
- Purpose: Component interactions (CLI + CUE loader + validator)
- Build tag: `//go:build integration`
- Run: `go test -tags=integration ./...`

E2E tests:

- Location: `test/e2e/*_test.go`
- Purpose: Compiled binary end-to-end
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
│   │   └── ...
│   ├── cue/
│   │   ├── loader.go
│   │   ├── loader_test.go
│   │   └── ...
│   └── config/
├── test/
│   ├── integration/
│   ├── e2e/
│   └── testdata/
│       ├── valid/
│       └── invalid/
└── scripts/
    └── invoke-tests
```

## Test Script

`scripts/invoke-tests` is the single entry point. It handles building the binary for e2e tests.

```bash
./scripts/invoke-tests      # Unit tests only
./scripts/invoke-tests -i   # Unit + integration
./scripts/invoke-tests -e   # Unit + integration + e2e
./scripts/invoke-tests -a   # All tests (same as -e)
./scripts/invoke-tests -c   # With coverage report
./scripts/invoke-tests -v   # Verbose output
./scripts/invoke-tests -s   # Short mode (skip long tests)
```

## Patterns

Cobra command testing:

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

Table-driven tests:

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

Real file system — use `t.TempDir()` for file operations:

```go
func TestLoader_ValidCUE(t *testing.T) {
    tmpDir := t.TempDir()
    // Write real CUE files, test real loading
}
```

Interactive input — inject `io.Reader`/`io.Writer`:

```go
type Command struct {
    In  io.Reader
    Out io.Writer
}
```

E2E binary testing:

```go
func TestE2E_StartInit(t *testing.T) {
    cmd := exec.Command("./bin/start", "init")
    cmd.Dir = t.TempDir()
    output, err := cmd.CombinedOutput()
    // assertions
}
```

## Mocking Guidelines

Do not mock:

- CUE library — use real CUE validation (fast and deterministic)
- File system — use `t.TempDir()` for real files
- Time — unless testing time-based logic specifically

Acceptable to mock:

- External network calls (HTTP clients for registry access)
- System command detection (`exec.LookPath` via interface)
- Environment variables (override via `cmd.Env`)

## Coverage Goals

- `internal/cue/`, `internal/config/`: 80%+ coverage
- `internal/cli/`: 70%+ coverage
- Overall: focus on meaningful coverage, not percentage targets
