# Observability Code Review

**Date:** 2026-01-21
**Scope:** Full codebase observability assessment
**Reviewer:** Claude Opus 4.5

---

## Executive Summary

The `start` CLI is a command-line orchestrator for AI agents with a focused scope: configuration loading, prompt composition, and process execution. This review assesses the codebase's observability characteristics against production debugging and monitoring requirements.

**Overall Assessment:** The codebase has adequate observability for a CLI tool, with good error context and a functional debug mode. However, several gaps exist that would hinder production debugging and operational visibility in more complex deployment scenarios.

| Category | Status | Notes |
|----------|--------|-------|
| Logging | Limited | No structured logging; `--debug` flag provides unstructured stderr output |
| Error Handling | Good | Consistent use of `%w` for error wrapping with context |
| Metrics | None | No instrumentation for performance or business metrics |
| Tracing | None | No distributed tracing support |
| Health Checks | None | CLI tool - not applicable (no long-running process) |
| Sensitive Data | Good | No obvious credential logging; prompts may contain user content |

---

## 1. Logging Analysis

### Current State

The codebase uses a custom `debugf` function for debug-level output:

```go
// internal/cli/start.go:47-52
func debugf(flags *Flags, category, format string, args ...interface{}) {
    if flags.Debug {
        _, _ = fmt.Fprintf(os.Stderr, "[DEBUG] %s: "+format+"\n", append([]interface{}{category}, args...)...)
    }
}
```

**Logging Characteristics:**
- Output destination: `os.Stderr`
- Format: `[DEBUG] <category>: <message>`
- Categories used: `config`, `agent`, `task`, `role`, `context`, `compose`, `exec`
- Activation: `--debug` flag (implies `--verbose`)

**No production logging library is used.** The codebase does not import:
- `log/slog` (Go 1.21+ structured logging)
- `github.com/sirupsen/logrus`
- `go.uber.org/zap`
- `github.com/rs/zerolog`

### Logging Coverage

| Package | Debug Logging | Error Logging | Notes |
|---------|--------------|---------------|-------|
| `cli` | Yes | Via returned errors | Good debug coverage for start/task commands |
| `orchestration` | No | Via returned errors | No internal operation logging |
| `cue` | No | Via returned errors | Silent config loading |
| `registry` | No | Via returned errors | Network operations not logged |
| `shell` | No | Via returned errors | Command execution not logged |
| `detection` | No | Via returned errors | Agent detection not logged |
| `doctor` | No | Via reporter output | Diagnostic output only |

### Gaps Identified

1. **No structured logging**: Debug output uses `fmt.Fprintf` with string interpolation, making parsing difficult
2. **CLI-only debug output**: The `debugf` function is only available in the `cli` package; lower-level packages have no logging
3. **No log levels**: Only debug (on/off); no warn/info/error log levels
4. **No timestamps**: Debug output has no timing information
5. **No correlation IDs**: No request/execution ID to trace a single invocation

### Recommendations

**Priority: Medium** (CLI tool with short execution lifecycle)

1. **Add slog for structured logging** (if future complexity warrants):
   ```go
   import "log/slog"

   slog.Debug("task resolved",
       "task", taskName,
       "source", "installed",
       "duration", time.Since(start),
   )
   ```

2. **Add execution ID for debug correlation**:
   ```go
   execID := uuid.New().String()[:8]
   debugf(flags, "exec", "[%s] Starting execution", execID)
   ```

3. **Add duration logging for operations** (see Section 8)

---

## 2. Error Handling Analysis

### Current State

Error handling is well-implemented with consistent patterns:

**Error Wrapping:**
- 219 occurrences of `%w` error wrapping across 26 files
- 357 occurrences of `fmt.Errorf` total
- Good use of contextual wrapping: `fmt.Errorf("loading configuration: %w", err)`

**Custom Error Types:**
- `cue.ValidationError`: Rich error context with file, line, column, and source snippet
- `DetailedError()` method provides user-friendly output with source context

```go
// internal/cue/validator.go:72-94
type ValidationError struct {
    Path     string
    Message  string
    Line     int
    Column   int
    Filename string
    Context  string // Source context snippet around the error
}
```

### Error Context Quality

| Package | Error Context | Notes |
|---------|--------------|-------|
| `cli` | Good | Operations wrap errors with context |
| `orchestration` | Good | File paths and operation names included |
| `cue` | Excellent | Line numbers, column positions, source snippets |
| `registry` | Good | Module paths and retry counts included |
| `shell` | Good | Exit codes, stderr content, timeout information |
| `config` | Good | Directory paths and validation details |

### Example Good Error Chain

```go
// registry/client.go:76
return FetchResult{}, fmt.Errorf("fetching module %s after %d attempts: %w", modulePath, c.retries, lastErr)

// orchestration/autosetup.go:97
return nil, fmt.Errorf("resolving agent version: %w", err)

// cli/start.go:143-144
if err != nil {
    return fmt.Errorf("composing prompt: %w", err)
}
```

### Gaps Identified

1. **No stack traces**: Unexpected panics would lack recovery context
2. **No error classification**: Errors aren't categorised (user error vs system error vs transient)
3. **Some errors lose context**: Direct `err` returns without wrapping in some paths
4. **No error codes**: Would aid programmatic error handling and documentation

### Recommendations

**Priority: Low** (current implementation is adequate for CLI)

1. **Add panic recovery in main**:
   ```go
   defer func() {
       if r := recover(); r != nil {
           fmt.Fprintf(os.Stderr, "panic: %v\n%s", r, debug.Stack())
           os.Exit(2)
       }
   }()
   ```

2. **Consider error classification** for future programmatic use:
   ```go
   type ErrorKind int
   const (
       ErrConfig ErrorKind = iota
       ErrNetwork
       ErrExecution
   )
   ```

---

## 3. Sensitive Data Analysis

### Current State

The codebase handles several potentially sensitive data types:

| Data Type | Where Used | Logging Risk |
|-----------|-----------|--------------|
| Prompts | `ExecuteConfig.Prompt` | Logged in dry-run; contains user content |
| Roles | `ExecuteConfig.Role` | Logged in dry-run; may contain instructions |
| File paths | Throughout | Logged; may reveal directory structure |
| Model names | Agent config | Logged; not sensitive |
| Commands | Shell execution | Logged in debug; contains prompt data |

### Analysis

**Good Practices:**
- No credential fields in configuration (API keys handled by wrapped CLIs)
- No explicit logging of environment variables
- File contents not logged by default (only in dry-run mode)

**Potential Concerns:**

1. **Debug mode logs full commands**:
   ```go
   // cli/start.go:181
   debugf(flags, "exec", "Final command: %s", cmdStr)
   ```
   This includes shell-escaped prompts which may contain sensitive user content.

2. **Dry-run mode writes files**:
   ```go
   // Writes prompt.md, role.md, command.txt to .start/temp/dry-run/
   ```
   These files persist until manually deleted and contain full prompt content.

3. **Error messages may contain user content**:
   Template errors include the failing content in error messages.

### Recommendations

**Priority: Medium**

1. **Add warning about debug mode**:
   ```go
   if flags.Debug {
       fmt.Fprintln(os.Stderr, "Warning: Debug mode may log sensitive content")
   }
   ```

2. **Truncate large content in debug logs**:
   ```go
   func debugContent(s string, maxLen int) string {
       if len(s) > maxLen {
           return s[:maxLen] + "... (truncated)"
       }
       return s
   }
   ```

3. **Auto-cleanup dry-run files** (optional):
   Add a cleanup mechanism for old dry-run artifacts.

---

## 4. Metrics Analysis

### Current State

**No metrics instrumentation exists.** The codebase does not include:
- Prometheus client
- OpenTelemetry metrics
- Custom timing/counter logic
- Performance measurement

### Missing Metrics (if applicable to future use cases)

| Metric Type | Potential Metrics |
|-------------|------------------|
| Counters | Executions, errors by type, registry fetches |
| Histograms | Config load time, registry fetch latency, command execution time |
| Gauges | Active goroutines (N/A for short-lived CLI) |

### Recommendations

**Priority: Low** (CLI tool; metrics add overhead without clear benefit)

For a CLI tool, metrics are typically not needed. If usage analytics become desired:

1. **Opt-in telemetry** (not recommended for this tool):
   ```go
   if os.Getenv("START_TELEMETRY") == "1" {
       // Send anonymised usage data
   }
   ```

2. **Local timing for performance debugging**:
   ```go
   start := time.Now()
   // operation
   debugf(flags, "perf", "config loaded in %v", time.Since(start))
   ```

---

## 5. Distributed Tracing Analysis

### Current State

**No tracing instrumentation exists.** The codebase does not include:
- OpenTelemetry tracer
- Jaeger/Zipkin clients
- Trace context propagation

### Applicability

Distributed tracing is **not applicable** for this CLI tool because:
- Single-process execution model
- No service-to-service calls (only CUE registry HTTP calls)
- Short execution lifecycle (sub-second to seconds)

### Recommendations

**Priority: None** (not applicable for CLI architecture)

If the tool evolves to include long-running operations or service mode, consider adding OpenTelemetry spans for:
- Registry fetch operations
- Shell command execution
- CUE compilation

---

## 6. Health Checks Analysis

### Current State

The `doctor` command provides diagnostic health checks:

```go
// internal/doctor/doctor.go
type CheckResult struct {
    Status  Status   // pass, warn, fail, info
    Label   string   // e.g., "claude", "agents.cue"
    Message string   // Detail message
    Fix     string   // Suggested fix action
    Details []string // Additional details for verbose mode
}
```

**Doctor checks include:**
- Configuration validation (CUE file syntax)
- Agent binary availability (`exec.LookPath`)
- Path existence verification

### Health Check Coverage

| Check | Present | Notes |
|-------|---------|-------|
| Config syntax | Yes | Via CUE loader validation |
| Agent binaries | Yes | PATH lookup for configured agents |
| Registry connectivity | Implicit | Fails on registry operations |
| Filesystem permissions | No | Would fail on operation |

### Recommendations

**Priority: Low** (existing doctor command is adequate)

1. **Add registry connectivity check to doctor**:
   ```go
   func checkRegistryConnectivity() CheckResult {
       ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
       defer cancel()
       // Attempt index fetch
   }
   ```

2. **Add write permission check for temp directory**

---

## 7. Performance Observability

### Current State

**No performance instrumentation exists.** Operations that could benefit from timing:

| Operation | Location | Typical Duration |
|-----------|----------|-----------------|
| CUE config loading | `cue/loader.go` | 10-100ms |
| Registry fetch | `registry/client.go` | 100ms-5s |
| Template processing | `orchestration/template.go` | <10ms |
| Shell command execution | `shell/runner.go` | Variable |
| Agent binary lookup | `detection/agent.go` | <100ms |

### Recommendations

**Priority: Medium** (aids debugging of slow operations)

1. **Add timing to registry operations**:
   ```go
   func (c *Client) Fetch(ctx context.Context, modulePath string) (FetchResult, error) {
       start := time.Now()
       defer func() {
           // Log timing if debug enabled or slow
           if time.Since(start) > time.Second {
               fmt.Fprintf(os.Stderr, "Registry fetch took %v\n", time.Since(start))
           }
       }()
       // existing logic
   }
   ```

2. **Add slow operation warnings**:
   ```go
   const slowThreshold = 2 * time.Second
   if elapsed := time.Since(start); elapsed > slowThreshold {
       fmt.Fprintf(os.Stderr, "Warning: %s took %v\n", operation, elapsed)
   }
   ```

---

## 8. Debugging Support Analysis

### Current State

**Debug capabilities:**

| Feature | Status | Implementation |
|---------|--------|----------------|
| Debug flag | Yes | `--debug` enables verbose output |
| Verbose flag | Yes | `--verbose` for additional output |
| Dry-run mode | Yes | `--dry-run` previews without execution |
| Debug output format | Basic | `[DEBUG] category: message` |
| Runtime log level change | No | Requires restart with flag |
| Debug endpoints | N/A | CLI, not a service |
| Profiling | No | No pprof integration |

**Debug output categories:**
- `config`: Configuration loading and paths
- `agent`: Agent selection and configuration
- `task`: Task resolution and execution
- `role`: Role selection
- `context`: Context selection and loading
- `compose`: Prompt composition statistics
- `exec`: Command building and execution

### Debug Output Quality

**Good aspects:**
- Categories provide filtering capability (manual grep)
- Key decisions logged (agent selection, task resolution)
- File paths included for debugging config issues

**Gaps:**
- No structured format (JSON option would aid parsing)
- No timing information
- No way to enable per-category debugging

### Recommendations

**Priority: Medium**

1. **Add JSON debug output option**:
   ```go
   if flags.DebugJSON {
       json.NewEncoder(os.Stderr).Encode(map[string]any{
           "category": category,
           "message":  fmt.Sprintf(format, args...),
           "time":     time.Now().Format(time.RFC3339),
       })
   }
   ```

2. **Add category filtering**:
   ```bash
   --debug-category=task,exec  # Only show specific categories
   ```

3. **Add timing to debug output**:
   ```go
   _, _ = fmt.Fprintf(os.Stderr, "[DEBUG] %s: [%s] "+format+"\n",
       category, time.Now().Format("15:04:05.000"), args...)
   ```

---

## 9. Correlation and Request Tracking

### Current State

**No correlation IDs or execution tracking exists.**

Each CLI invocation is independent with no way to:
- Correlate debug output from a single execution
- Track execution through log aggregation
- Link parent-child operations

### Recommendations

**Priority: Low** (single-user CLI; limited benefit)

1. **Add execution ID for complex debugging scenarios**:
   ```go
   type ExecutionContext struct {
       ID        string
       StartTime time.Time
   }

   func NewExecutionContext() ExecutionContext {
       return ExecutionContext{
           ID:        generateShortID(),
           StartTime: time.Now(),
       }
   }
   ```

---

## 10. Alerting Considerations

### Applicability

Alerting is **not applicable** for this CLI tool:
- No long-running process to monitor
- No server-side components
- Errors are reported directly to the user

### Future Considerations

If the tool evolves to include:
- Background operations
- Server mode for IDE integration
- Daemon mode for watch functionality

Then alerting on:
- Repeated configuration errors
- Registry connectivity failures
- Slow operations exceeding thresholds

---

## Summary of Recommendations

### High Priority

None identified - the current observability is adequate for a CLI tool.

### Medium Priority

1. **Add timing to debug output** for performance visibility
2. **Add sensitive content warning** when debug mode is enabled
3. **Truncate large content** in debug logs to prevent sensitive data exposure
4. **Add slow operation warnings** for registry fetches

### Low Priority

1. Consider structured logging (slog) if complexity grows
2. Add execution ID for complex debugging scenarios
3. Add registry connectivity check to doctor command
4. Add JSON debug output option
5. Add panic recovery with stack trace in main

---

## Conclusion

The `start` CLI has observability appropriate for its scope as a single-user command-line tool:

**Strengths:**
- Consistent error wrapping with context (`%w` pattern)
- Rich CUE validation errors with source context
- Functional debug mode with categorised output
- Dry-run mode for execution preview
- Doctor command for self-diagnostics

**Areas for Improvement:**
- No structured logging (acceptable for CLI)
- No performance timing instrumentation
- Debug output lacks timestamps
- Potential sensitive content in debug logs

The codebase prioritises simplicity over observability infrastructure, which is the correct trade-off for a CLI tool. The recommendations above are optional enhancements that would improve debugging in edge cases but are not critical for current functionality.
