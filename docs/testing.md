# Testing Coverage Gaps

This document tracks known test coverage gaps and their justifications. Last updated after the comprehensive test coverage review on 2025-12-17.

## Current Coverage

**Overall: 80.6%**

| Package | Coverage | Target | Status |
|---------|----------|--------|--------|
| `internal/detection` | 100.0% | 80%+ | Exceeds |
| `internal/config` | 88.9% | 80%+ | Exceeds |
| `internal/shell` | 88.1% | 80%+ | Exceeds |
| `internal/cue` | 87.6% | 80%+ | Exceeds |
| `internal/temp` | 83.9% | 80%+ | Exceeds |
| `internal/registry` | 81.1% | 80%+ | Meets |
| `internal/orchestration` | 78.7% | 80%+ | Close |
| `internal/cli` | 77.0% | 70%+ | Exceeds |
| `cmd/start` | 0.0% | N/A | Expected |

## Remaining Gaps

### Intentionally Untested (E2E Coverage Only)

These functions cannot be unit tested due to their nature but are covered by E2E tests.

#### `orchestration/executor.go:Execute()` - 0%

**Reason:** Uses `syscall.Exec` which replaces the current process. Cannot return to test code after execution.

**Mitigation:**
- `ExecuteWithoutReplace()` provides equivalent logic and is fully tested
- `BuildCommand()` is tested at 92.9%
- E2E tests in `test/e2e/autosetup_test.go` exercise the full execution path

**Risk:** Low - the syscall itself is a standard library function.

#### `orchestration/autosetup.go:Run()` - 0%

**Reason:** Main entry point that orchestrates network calls (registry fetch), file I/O (config writing), and interactive prompts. Testing requires mocking the entire registry client.

**Mitigation:**
- All component functions are individually tested:
  - `NeedsSetup()` - 100%
  - `promptSelection()` - 96.3%
  - `noAgentsError()` - 93.3%
  - `extractAgentFromValue()` - 100%
  - `writeConfig()` - 71.4%
  - `generateAgentCUE()` - 100%
  - `generateSettingsCUE()` - 100%
- E2E tests cover the complete flow with real binaries

**Risk:** Low - composition of tested components.

#### `orchestration/autosetup.go:loadAgentFromModule()` - 0%

**Reason:** Requires fetched CUE module from registry. The module loading uses `cue/load` which expects real filesystem paths from the registry cache.

**Mitigation:**
- `extractAgentFromValue()` is tested at 100% with all CUE value formats
- Integration between loader and agent extraction is verified
- E2E tests exercise the complete module loading path

**Risk:** Low - CUE loading is well-tested by CUE library.

#### `registry/index.go:FetchIndex()` - 0%

**Reason:** Wrapper that calls `Fetch()` then `LoadIndex()`. Requires network access to CUE Central Registry.

**Mitigation:**
- `Fetch()` is tested at 94.1% with retry logic and context cancellation
- `LoadIndex()` is tested at 83.3%
- E2E tests use real registry

**Risk:** Low - thin wrapper over tested functions.

### Acceptable Gaps (Display/Entry Functions)

These functions are low-risk display or CLI entry points.

| Function | Coverage | Reason |
|----------|----------|--------|
| `cli/start.go:runStart()` | 0% | Cobra wrapper, delegates to `executeStart()` (67.7%) |
| `cli/start.go:printExecutionInfo()` | 0% | Display-only, output format not critical |
| `cli/start.go:runAutoSetup()` | 0% | TTY detection wrapper |
| `cli/task.go:runTask()` | 0% | Cobra wrapper, delegates to `executeTask()` (67.5%) |
| `cli/task.go:printTaskExecutionInfo()` | 0% | Display-only |
| `cli/prompt.go:runPrompt()` | 0% | Command not yet implemented |
| `cmd/start/main.go:main()` | 0% | Entry point, calls `cli.Execute()` |
| `temp/manager.go:NewDryRunManager()` | 0% | Simple constructor |

### Functions Below Target

These functions are partially tested but could benefit from additional coverage.

| Function | Coverage | Gap |
|----------|----------|-----|
| `shell/detection.go:DetectShell()` | 40.0% | Difficult to test `$SHELL` not set on most systems |
| `temp/manager.go:EnsureUTDDir()` | 66.7% | Error path when `os.MkdirAll` fails |
| `registry/index.go:decodeIndex()` | 70.0% | Some error accumulation paths |
| `orchestration/autosetup.go:writeConfig()` | 71.4% | Error path when `os.MkdirAll` fails |

## Test Types

Per DR-024, tests are organized into three tiers:

- **Unit tests** (`*_test.go` alongside source) - Run by default
- **Integration tests** (`test/integration/`) - Run with `-tags=integration`
- **E2E tests** (`test/e2e/`) - Run with `-tags=e2e`, requires built binary

Run all tests: `./scripts/invoke-tests -a`

## Adding New Tests

When adding tests for the remaining gaps:

1. Prefer testing via the public API rather than internal functions
2. Use `t.TempDir()` for file operations
3. Use real CUE validation (don't mock CUE library)
4. For network-dependent code, create integration tests with `//go:build integration`
5. For binary-dependent code, create E2E tests with `//go:build e2e`

## References

- DR-024: Testing Strategy (`docs/design/design-records/dr-024-testing-strategy.md`)
- Test script: `scripts/invoke-tests`
