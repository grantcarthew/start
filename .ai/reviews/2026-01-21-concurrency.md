# Go Concurrency Code Review

**Repository:** start
**Date:** 2026-01-21
**Reviewer:** Claude Opus 4.5

## Executive Summary

The codebase maintains a **minimal concurrency footprint** with only 2 goroutine creation points (1 in production code, 1 in tests). All concurrent code follows Go best practices with proper synchronisation. The race detector reports no issues.

**Overall Assessment:** No concurrency issues found. No changes to concurrency patterns since last review (2026-01-15).

---

## 1. Race Detector Results

```bash
go test -race ./...
```

**Result:** All packages pass with race detector enabled (1 unrelated test failure in cli package).

| Package | Status | Notes |
|---------|--------|-------|
| internal/cli | PASS (race) | 1 unrelated test logic failure |
| internal/config | PASS | |
| internal/cue | PASS | |
| internal/detection | PASS | |
| internal/doctor | PASS | |
| internal/orchestration | PASS | |
| internal/registry | PASS | |
| internal/shell | PASS | |
| internal/temp | PASS | |

The test failure in `TestFindTask/prefix_match` is a test logic issue unrelated to concurrency.

---

## 2. Goroutine Creation Points

### 2.1 Production Code

**Location:** `internal/detection/agent.go:37`

```go
for key, entry := range index.Agents {
    if entry.Bin == "" {
        continue
    }

    wg.Add(1)
    go func(k string, e registry.IndexEntry) {
        defer wg.Done()

        path, err := exec.LookPath(e.Bin)
        if err != nil {
            return
        }

        mu.Lock()
        defer mu.Unlock()
        detected = append(detected, DetectedAgent{
            Key:        k,
            Entry:      e,
            BinaryPath: path,
        })
    }(key, entry)
}

wg.Wait()
```

**Analysis:**

| Criterion | Status | Details |
|-----------|--------|---------|
| WaitGroup.Add before spawn | OK | `wg.Add(1)` precedes `go func()` |
| Defer WaitGroup.Done | OK | `defer wg.Done()` at goroutine start |
| Loop variable capture | OK | Variables passed as parameters |
| Shared state protection | OK | Mutex guards `detected` slice |
| Lock duration | OK | Minimal (append only) |
| Goroutine termination | OK | All exit via `wg.Wait()` |
| Mutex unlock pattern | OK | Uses `defer mu.Unlock()` |

**Verdict:** Correctly implemented.

### 2.2 Test Code

**Location:** `internal/registry/client_test.go:192`

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(50 * time.Millisecond)
    cancel()
}()

_, err := client.Fetch(ctx, "github.com/test/module@v0.0.1")
```

**Analysis:**

- Test-only goroutine for simulating context cancellation
- No shared state modified
- Goroutine completes after `cancel()` call
- Parent blocks on `Fetch()` which respects cancellation

**Verdict:** Correctly implemented for test purposes.

---

## 3. Synchronisation Mechanisms

### 3.1 sync.Mutex

**Location:** `internal/detection/agent.go:26`

```go
var (
    mu       sync.Mutex
    detected []DetectedAgent
    wg       sync.WaitGroup
)
```

| Check | Status |
|-------|--------|
| Lock before critical section | OK |
| Unlock after critical section | OK |
| Uses defer pattern | OK |
| Minimal lock duration | OK |
| No nested locks | OK |

### 3.2 sync.WaitGroup

**Location:** `internal/detection/agent.go:28`

| Check | Status |
|-------|--------|
| Add before goroutine | OK |
| Done with defer | OK |
| Wait after all Add calls | OK |

---

## 4. Channel Usage

### 4.1 Context Cancellation with Select

**Location:** `internal/registry/client.go:57-60`

```go
select {
case <-ctx.Done():
    return FetchResult{}, ctx.Err()
case <-time.After(wait):
}
```

**Analysis:**

- Respects context cancellation during retry backoff
- Returns context error for proper propagation
- No goroutine leak (both cases complete)
- `time.After` creates short-lived timer (acceptable for retry delays)

**Verdict:** Correctly implemented.

---

## 5. Context Usage

### 5.1 Context as First Parameter

All context-accepting functions use idiomatic signatures:

| Function | File |
|----------|------|
| `Client.Fetch` | registry/client.go |
| `Client.ModuleVersions` | registry/client.go |
| `Client.ResolveLatestVersion` | registry/client.go |
| `AutoSetup.Run` | orchestration/autosetup.go |

### 5.2 Context Storage

No contexts are stored in structs. All contexts are passed explicitly.

### 5.3 context.Background() Usage

Used appropriately at CLI entry points:

- `internal/cli/assets_*.go` - CLI commands
- `internal/cli/task.go` - Task execution
- `internal/cli/start.go` - Main orchestration
- `internal/shell/runner.go` - With `WithTimeout`

**Verdict:** Correct for CLI applications without parent context.

---

## 6. Command Execution

### 6.1 exec.CommandContext

**Location:** `internal/shell/runner.go:89`

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
defer cancel()

cmd := exec.CommandContext(ctx, shellBin, args...)
```

**Analysis:**

| Aspect | Status | Details |
|--------|--------|---------|
| Timeout context | OK | Created with deadline |
| Context cancelled | OK | `defer cancel()` prevents leak |
| Process group | OK | `Setpgid: true` for clean kill |
| Timeout detection | OK | Checks `ctx.Err()` |
| Child process cleanup | OK | Kills process group on timeout |

### 6.2 syscall.Exec

**Location:** `internal/orchestration/executor.go:234`

```go
return syscall.Exec(shell, args, env)
```

**Analysis:** Process replacement - no concurrency implications. Intentional design for clean agent handoff.

---

## 7. Testing Concurrency

### 7.1 Parallel Test Execution

```go
func TestDetectAgents_ParallelExecution(t *testing.T) {
    t.Parallel()
    // Multiple iterations to expose race conditions
    for i := 0; i < 10; i++ {
        detected := DetectAgents(index)
        // ...
    }
}
```

### 7.2 t.Parallel() Coverage

18 test files use `t.Parallel()` across 100+ test functions.

### 7.3 Race Flag in CI

**Observation:** The `scripts/invoke-tests` script does not include `-race` flag by default.

**Recommendation:** Consider adding a `-r, --race` flag option to the test script for CI integration.

---

## 8. Anti-Pattern Checks

| Anti-Pattern | Status |
|--------------|--------|
| Goroutine leak (no exit path) | Not found |
| Loop variable capture (pre-Go 1.22) | Not found |
| Unlock before lock | Not found |
| Closing channel from receiver | N/A (no channels closed) |
| Storing context in struct | Not found |
| WaitGroup.Add after goroutine | Not found |
| WaitGroup.Done without defer | Not found |
| Nested locks (deadlock risk) | Not found |
| Send on closed channel | N/A (no channel send) |

---

## 9. Advanced Concurrency Patterns

The codebase does not use:

- `sync.Pool` - Not needed
- `sync.Cond` - Not needed
- `errgroup` - Not needed
- `semaphore` - Not needed
- `atomic` operations - Not needed
- Runtime functions (`GOMAXPROCS`, etc.) - Not needed

This is appropriate for the project's scope. The minimal concurrency model reduces complexity and potential for bugs.

---

## 10. Changes Since Last Review

**Last Review:** 2026-01-15

**Files Changed:** 35 Go files modified since last review

**Concurrency Changes:** None

The commits since the last review (`060a2d9`) focused on:

- Task command implementation and aliases
- File path support for roles/contexts/tasks
- Composer tilde path expansion
- Interactive model/tag editing
- Output formatting

No new goroutine creation points, channels, or synchronisation primitives were added.

---

## 11. Summary

### Issues Found

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| Informational | 1 |

### Informational Notes

1. **Race flag not in default test script** - Consider adding `-race` option to `scripts/invoke-tests` for CI integration.

### Positive Observations

- Minimal concurrency footprint (1 production goroutine pattern)
- Proper sync primitive usage (Mutex with defer, WaitGroup ordering)
- Context passed correctly through call chains
- Context cancellation respected in retry loops
- Loop variables passed as goroutine parameters (Go 1.22+ compatible)
- Race detector passes on all packages
- Dedicated parallel execution test exists
- No unnecessary concurrency complexity

---

## 12. Recommendations

1. **Add race flag to test script** (informational)

   Consider adding a `-r, --race` option to `scripts/invoke-tests`:

   ```bash
   if [[ "${race}" == true ]]; then
     test_args+=("-race")
   fi
   ```

2. **Maintain current practices** - The minimal concurrency approach is appropriate for this CLI tool.

3. **No code changes required** - All concurrency patterns are correctly implemented.
