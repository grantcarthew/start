# Go Concurrency Code Review

**Repository:** start
**Date:** 2026-01-15
**Reviewer:** Claude Opus 4.5

## Executive Summary

The codebase has **minimal concurrency** with only 2 goroutine creation points. All concurrent code follows Go best practices with proper synchronisation. The race detector reports no issues.

**Overall Assessment:** No concurrency issues found.

---

## 1. Race Detector Results

```
go test -race ./...
```

**Result:** All packages pass with race detector enabled.

| Package | Status |
|---------|--------|
| internal/cli | PASS |
| internal/config | PASS |
| internal/cue | PASS |
| internal/detection | PASS |
| internal/doctor | PASS |
| internal/orchestration | PASS |
| internal/registry | PASS |
| internal/shell | PASS |
| internal/temp | PASS |

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
        detected = append(detected, DetectedAgent{
            Key:        k,
            Entry:      e,
            BinaryPath: path,
        })
        mu.Unlock()
    }(key, entry)
}

wg.Wait()
```

**Analysis:**
- WaitGroup `Add()` called before goroutine spawn
- `defer wg.Done()` ensures completion signalling
- Loop variables passed as parameters (avoids closure capture issues)
- Mutex protects shared `detected` slice
- Lock held for minimal duration (append only)
- All goroutines guaranteed to exit via `wg.Wait()`

**Verdict:** Correctly implemented.

### 2.2 Test Code

**Location:** `internal/registry/client_test.go:189`

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(50 * time.Millisecond)
    cancel()
}()

_, err := client.Fetch(ctx, "github.com/test/module@v0.0.1")
```

**Analysis:**
- Test goroutine for simulating context cancellation
- No shared state modified
- Goroutine completes after `cancel()` call
- Parent function blocks on `Fetch()` which respects cancellation

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

**Usage Pattern:**
- Protects append operations to shared slice
- Lock/unlock in correct order (lock before critical section)
- Uses `defer mu.Unlock()` for safe unlock on all code paths

**Status:** ✅ Uses idiomatic `defer mu.Unlock()` pattern.

### 3.2 sync.WaitGroup

**Location:** `internal/detection/agent.go:28`

**Usage Pattern:**
- `Add(1)` called before `go func()`
- `Done()` called with `defer` inside goroutine
- `Wait()` called after all `Add()` calls complete

**Verdict:** Correct usage following Go best practices.

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
- Returns context error for proper error propagation
- No goroutine leak risk (both cases are non-blocking eventually)

**Verdict:** Correctly implemented.

---

## 5. Context Usage

### 5.1 Context as First Parameter

All functions accepting context use the idiomatic signature:

| Function | Signature |
|----------|-----------|
| `Client.Fetch` | `func (c *Client) Fetch(ctx context.Context, modulePath string)` |
| `Client.ModuleVersions` | `func (c *Client) ModuleVersions(ctx context.Context, modulePath string)` |
| `Client.ResolveLatestVersion` | `func (c *Client) ResolveLatestVersion(ctx context.Context, modulePath string)` |
| `Client.FetchIndex` | `func (c *Client) FetchIndex(ctx context.Context)` |
| `AutoSetup.Run` | `func (a *AutoSetup) Run(ctx context.Context)` |
| `checkForUpdates` | `func checkForUpdates(ctx context.Context, client *registry.Client, ...)` |
| `checkAndUpdate` | `func checkAndUpdate(ctx context.Context, client *registry.Client, ...)` |

**Verdict:** Follows Go conventions.

### 5.2 Context Storage

No contexts are stored in structs. All contexts are passed explicitly through call chains.

**Verdict:** Correct practice.

### 5.3 context.Background() Usage

The codebase uses `context.Background()` at CLI command entry points:

- `internal/cli/assets_list.go:47`
- `internal/cli/assets_add.go:46`
- `internal/cli/assets_search.go:48`
- `internal/cli/assets_update.go:53`
- `internal/cli/assets_info.go:36`
- `internal/cli/task.go:339`
- `internal/cli/start.go:431`
- `internal/shell/runner.go:84`

**Analysis:** This is appropriate for CLI applications where there is no parent context. Shell runner uses `context.WithTimeout()` to add timeout semantics.

**Verdict:** Correct usage.

---

## 6. Timeout Handling

**Location:** `internal/shell/runner.go:84-119`

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSecs)*time.Second)
defer cancel()
// ...
cmd := exec.CommandContext(ctx, shellBin, args...)
// ...
if ctx.Err() == context.DeadlineExceeded {
    result.TimedOut = true
    if cmd.Process != nil {
        _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
    }
    return result, fmt.Errorf("command timed out after %d seconds", timeoutSecs)
}
```

**Analysis:**
- Context with timeout created correctly
- `defer cancel()` prevents context leak
- Timeout detection via `ctx.Err()`
- Process group kill for clean child process termination

**Verdict:** Well-implemented timeout handling.

---

## 7. Testing Concurrency

### 7.1 Race Condition Testing

**Location:** `internal/detection/agent_test.go:115-134`

```go
func TestDetectAgents_ParallelExecution(t *testing.T) {
    // ...
    for i := 0; i < 10; i++ {
        detected := DetectAgents(index)
        if len(detected) != 0 {
            t.Errorf("iteration %d: expected 0 detected agents, got %d", i, len(detected))
        }
    }
}
```

**Analysis:** Multiple iterations help expose race conditions when combined with `-race` flag.

### 7.2 t.Parallel() Usage

The codebase now uses `t.Parallel()` in 18 test files (100+ test functions).

**Status:** ✅ Implemented across all safe test files.

---

## 8. Anti-Pattern Checks

| Anti-Pattern | Status |
|--------------|--------|
| Goroutine leak (no exit path) | Not found |
| Capturing loop variable (pre-Go 1.22) | Not found (parameters used) |
| Unlock before lock | Not found |
| Closing channel from receiver | Not found |
| Storing context in struct | Not found |
| Missing WaitGroup.Add before goroutine | Not found |
| WaitGroup.Done without defer | Not found |
| Nested locks (deadlock risk) | Not found |
| Send on closed channel | Not found (no channels closed) |

---

## 9. Summary

### Issues Found

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High | 0 |
| Medium | 0 |
| Low | 0 |
| Informational | 0 |

### Informational Notes (Resolved)

1. **Mutex unlock pattern** (`internal/detection/agent.go:45-51`): ✅ FIXED - Now uses `defer mu.Unlock()` for consistency.

2. **Test parallelism**: ✅ FIXED - Added `t.Parallel()` to 18 safe test files (100+ test functions).

### Positive Observations

- Minimal concurrency footprint reduces complexity
- Proper use of sync primitives (Mutex, WaitGroup)
- Context passed correctly through call chains
- Context cancellation respected in retry loops
- Loop variables passed as goroutine parameters
- Race detector passes on all packages
- Dedicated test for parallel execution (`TestDetectAgents_ParallelExecution`)

---

## 10. Recommendations

1. **No action required** - All informational issues have been resolved.

2. **Maintain current practices** - Continue passing context as first parameter and avoiding context storage in structs.

---

## 11. Changes Made

### 11.1 Mutex Unlock Pattern

**File:** `internal/detection/agent.go`

Changed from explicit unlock to defer pattern:

```go
// Before
mu.Lock()
detected = append(detected, DetectedAgent{...})
mu.Unlock()

// After
mu.Lock()
defer mu.Unlock()
detected = append(detected, DetectedAgent{...})
```

### 11.2 Test Parallelism

Added `t.Parallel()` to 18 test files:

| Package | Files Modified | Tests Added |
|---------|----------------|-------------|
| internal/cue | 3 | 12 tests |
| internal/config | 1 | 3 tests |
| internal/detection | 1 | 5 tests |
| internal/doctor | 2 | 19 tests |
| internal/orchestration | 3 | 18 tests |
| internal/registry | 2 | 16 tests |
| internal/shell | 2 | 6 tests |
| internal/temp | 1 | 7 tests |
| internal/cli | 3 | 15 tests |

**Total:** 18 files, 100+ test functions now run in parallel.

All tests pass with `-race` flag enabled.
