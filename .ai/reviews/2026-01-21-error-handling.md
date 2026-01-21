# Go Error Handling Code Review

**Date:** 2026-01-21
**Reviewer:** Claude Opus 4.5
**Scope:** All Go source files in the `start` repository

---

## Executive Summary

The `start` codebase demonstrates generally excellent error handling practices. Errors are consistently checked, wrapped with context using `%w`, and propagated appropriately up the call stack. The codebase shows strong adherence to Go idioms and thoughtful consideration of edge cases.

**Overall Assessment:** High quality with minor areas for improvement.

| Category | Assessment |
|----------|------------|
| Error Checking | Excellent |
| Error Wrapping | Excellent |
| Sentinel Errors | Minimal (appropriate) |
| Panic Usage | None (correct) |
| Edge Cases | Good |
| Consistency | Excellent |

---

## 1. Error Checking

### Findings

All error returns are checked appropriately. The codebase does not contain any instances of errors being silently discarded with `_ = err`.

**Positive Examples:**

- `internal/cli/start.go:73-76` - Working directory errors properly checked:
  ```go
  workingDir, err = os.Getwd()
  if err != nil {
      return nil, fmt.Errorf("getting working directory: %w", err)
  }
  ```

- `internal/registry/client.go:47-51` - Module path parsing checked:
  ```go
  mv, err := module.ParseVersion(modulePath)
  if err != nil {
      return FetchResult{}, fmt.Errorf("parsing module path %q: %w", modulePath, err)
  }
  ```

### Defer Error Handling

**File:** `internal/cue/errors.go:127`
```go
defer func() { _ = file.Close() }()
```

**Assessment:** This is the only instance in the codebase where a `Close()` error is intentionally discarded with `_`. This is acceptable because:
1. The file is opened for reading only (`os.Open`)
2. Read-only file close errors are extremely rare
3. The function's purpose is to generate error context - logging a close error here would add noise

**Recommendation:** No change needed. This is an acceptable pattern for read-only file handles.

### Intentional Error Discarding Pattern

The codebase uses `_, _` assignments for `fmt.Fprint*` calls throughout the CLI output code:
```go
_, _ = fmt.Fprintln(w, "Starting AI Agent")
_, _ = colorSuccess.Fprintf(w, "%s", status)
```

**Assessment:** This is correct. Output functions returning write errors to stdout/stderr cannot meaningfully recover from those errors. The `_, _` pattern makes the intentional discarding explicit.

---

## 2. Error Wrapping and Context

### Findings

Error wrapping is consistently excellent throughout the codebase. The `%w` verb is used correctly to preserve the error chain.

**Exemplary Patterns:**

1. **Hierarchical Context** - `internal/orchestration/composer.go:143-145`:
   ```go
   if err != nil {
       return result, fmt.Errorf("selecting contexts: %w", err)
   }
   ```

2. **Operation + Resource Context** - `internal/temp/manager.go:69-71`:
   ```go
   if err := os.WriteFile(path, []byte(content), 0600); err != nil {
       return fmt.Errorf("writing %s: %w", name, err)
   }
   ```

3. **Multi-level Context** - `internal/cli/start.go:383-384`:
   ```go
   return internalcue.LoadResult{}, fmt.Errorf("resolving config paths: %w", err)
   ```

### Error Message Quality

Error messages are generally actionable and descriptive:

- Include the operation being performed
- Include relevant identifiers (file paths, names)
- Use consistent formatting ("verb + noun: %w")

**Improvement Opportunities:**

1. **`internal/orchestration/executor.go:217-218`** - Could include shell path in error:
   ```go
   // Current:
   return fmt.Errorf("no shell available")

   // Better:
   return fmt.Errorf("no shell available: checked bash, sh")
   ```

2. **`internal/orchestration/autosetup.go:161`** - Uses `%s` instead of `%w`:
   ```go
   return fmt.Errorf("%s", sb.String())
   ```
   This creates a new error instead of wrapping. However, this is intentional as the string is user-facing help text, not a wrapped error. Acceptable.

---

## 3. Sentinel Errors and Custom Error Types

### Sentinel Errors

The codebase uses minimal sentinel errors, which is appropriate for an application (vs library):

**`internal/cli/doctor.go:63-67`:**
```go
var errDoctorIssuesFound = &doctorError{}

type doctorError struct{}

func (e *doctorError) Error() string { return "issues found" }
```

**Assessment:** This is a private error used to set exit code 1 when doctor finds issues. The pattern is correct - it's a private type that cannot be compared externally, used purely for control flow.

### Custom Error Types

**`internal/cue/validator.go:72-94`:**
```go
type ValidationError struct {
    Path     string
    Message  string
    Line     int
    Column   int
    Filename string
    Context  string
}

func (e *ValidationError) Error() string { ... }
func (e *ValidationError) DetailedError() string { ... }
```

**Assessment:** Excellent implementation:
- Provides rich context for CUE validation errors
- Has both `Error()` (for logging) and `DetailedError()` (for user display)
- Used correctly with `errors.As` not needed (always returned directly)

**Missing:** No `Unwrap()` method, but this is intentional - ValidationError is always a leaf error, not wrapping another error.

### Error Comparison

The codebase correctly uses `errors.Is` for error comparison:

**`internal/registry/client_test.go:199`:**
```go
if !errors.Is(err, context.Canceled) {
    t.Errorf("expected context.Canceled, got %v", err)
}
```

**Assessment:** Tests use `errors.Is` correctly. Production code does not need `errors.Is` because it doesn't compare against specific error values.

---

## 4. Panic Usage

### Findings

**No panics in production code.** This is correct behaviour for an application.

The codebase was searched for `panic(` with no matches in production code. This demonstrates disciplined error handling - all failures are returned as errors rather than panicking.

### Recovery

No `recover()` calls are present, which is appropriate because:
1. No panics are raised
2. The application is not a server with goroutine-per-request semantics
3. CLI applications can safely let panics crash with stack traces

---

## 5. Edge Cases and Nil Handling

### Nil Pointer Handling

**Positive Examples:**

1. **`internal/orchestration/template.go:79-80`:**
   ```go
   if fr == nil {
       fr = &DefaultFileReader{}
   }
   ```

2. **`internal/detection/agent.go:21-23`:**
   ```go
   if index == nil || len(index.Agents) == 0 {
       return nil
   }
   ```

3. **`internal/shell/runner.go:116-118`:**
   ```go
   if cmd.Process != nil {
       _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
   }
   ```

### Empty Value Handling

**`internal/cli/start.go:200-208`:**
```go
func resolveModel(flagModel, configModel string) (model, source string) {
    if flagModel != "" {
        return flagModel, "--model"
    }
    if configModel != "" {
        return configModel, "config"
    }
    return "", ""
}
```

**Assessment:** Empty strings are handled distinctly from populated strings throughout the codebase.

### Slice Handling

**`internal/cue/loader.go:99-101`:**
```go
if len(values) == 0 {
    return result, fmt.Errorf("no valid CUE configuration found")
}
```

**Assessment:** Empty slices are properly checked before use.

### Bounds Checking

CUE iteration uses the safe pattern:
```go
iter, err := v.Fields()
if err != nil {
    return ...
}
for iter.Next() {
    // process
}
```

No direct slice indexing without bounds checks was found.

---

## 6. Context Errors

### Context Cancellation

**`internal/registry/client.go:57-60`:**
```go
select {
case <-ctx.Done():
    return FetchResult{}, ctx.Err()
case <-time.After(wait):
}
```

**Assessment:** Context cancellation is properly checked during retry loops.

### Timeout Handling

**`internal/shell/runner.go:113-119`:**
```go
if ctx.Err() == context.DeadlineExceeded {
    result.TimedOut = true
    if cmd.Process != nil {
        _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
    }
    return result, fmt.Errorf("command timed out after %d seconds", timeoutSecs)
}
```

**Assessment:** Timeout errors are properly distinguished and cleaned up. The process group is killed to prevent zombie processes.

---

## 7. Error Handling Consistency

### Pattern Adherence

The codebase follows a consistent pattern:

1. **Immediate return on error:**
   ```go
   if err != nil {
       return ..., fmt.Errorf("operation: %w", err)
   }
   ```

2. **No error accumulation** (except where semantically appropriate)

3. **Errors handled at appropriate levels:**
   - Low-level packages return wrapped errors
   - CLI layer formats for user display
   - `main.go` handles final output with colour

### User vs Internal Errors

**`internal/config/validation.go`** demonstrates the distinction:
- Returns `*ValidationError` with source context for display
- Internal errors wrapped with `%w` for debugging

**`internal/cue/validator.go`:**
- `Error()` - concise for logs
- `DetailedError()` - verbose with source snippet for users

---

## 8. Areas for Improvement

### Minor Issues

1. **Missing context in some CUE extraction errors**

   **Files:** `internal/orchestration/executor.go:299`, `internal/orchestration/autosetup.go:270`

   Pattern:
   ```go
   agent.Bin, _ = bin.String()
   ```

   **Issue:** When extracting CUE values, errors from `.String()` are discarded with `_`. While the field existence is checked first (`bin.Exists()`), if the field exists but isn't a string, the error is lost.

   **Recommendation:** These cases are edge cases that would only occur with malformed CUE that passed schema validation. The current pattern is acceptable given CUE schema enforcement, but could be improved by logging warnings in verbose mode.

2. **Shell error formatting includes stderr**

   **File:** `internal/orchestration/executor.go:264`
   ```go
   return stdout.String(), fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
   ```

   **Assessment:** This embeds stderr in the error message which could be very long. Consider truncating or separating.

### Suggestions (Non-Critical)

1. **Consider adding error wrapping to shell kill operation**

   `internal/shell/runner.go:117`:
   ```go
   _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
   ```

   While failure to kill is acceptable (process may have already exited), logging at debug level could aid troubleshooting.

2. **Consider errors.Join for multiple validation errors**

   The `internal/cue/errors.go` package could use `errors.Join` when CUE returns multiple errors:
   ```go
   cueErrs := errors.Errors(err)
   if len(cueErrs) > 1 {
       return errors.Join(cueErrs...)
   }
   ```

   Currently only the first error is returned with a count. This is acceptable UX but `errors.Join` would preserve all errors for programmatic handling.

---

## 9. Positive Highlights

### Exceptional Patterns

1. **Validation error with source context** - `internal/cue/errors.go:120-181`

   Generates a source code snippet around errors with line numbers and column pointers. This is excellent UX.

2. **Retry with exponential backoff** - `internal/registry/client.go:54-76`

   Proper retry logic with context cancellation, exponential backoff, and error chain preservation.

3. **Graceful degradation** - `internal/cli/doctor.go:94-143`

   Doctor command continues with available checks even when some fail.

4. **Clean process replacement error handling** - `internal/orchestration/executor.go:228-234`

   Uses `syscall.Exec` for zero-overhead agent execution with proper error propagation.

---

## 10. Summary

### What's Done Well

- All errors checked (no `err` ignored without explicit `_`)
- Consistent `%w` wrapping with descriptive context
- No panics in production code
- Nil checks before pointer dereference
- Context cancellation respected
- Clear separation of user-facing vs internal errors

### Recommendations

| Priority | Item | Location |
|----------|------|----------|
| Low | Log shell kill failures at debug level | `shell/runner.go:117` |
| Low | Consider `errors.Join` for multiple CUE errors | `cue/errors.go` |
| Low | Truncate stderr in command failure errors | `orchestration/executor.go:264` |

### Risk Assessment

**No high or medium priority issues identified.** The error handling in this codebase is production-ready and demonstrates Go best practices.

---

*Review completed: 2026-01-21*
