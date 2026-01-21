# Correctness Review Report

**Date**: 2026-01-21
**Reviewer**: Claude Opus 4.5
**Repository**: github.com/grantcarthew/start
**Scope**: Deep correctness review of Go source code

## Executive Summary

This correctness review examined the `start` CLI tool's Go codebase for logic errors, race conditions, incorrect error handling, edge cases, data corruption risks, and algorithm correctness. The codebase demonstrates high quality with consistent error handling patterns, good test coverage, and thoughtful design. Several issues of varying severity were identified for consideration.

## Severity Classification

- **Critical**: Could cause data loss, security vulnerabilities, or system crashes
- **High**: Logic errors that cause incorrect behavior in common scenarios
- **Medium**: Edge cases that may cause issues under specific conditions
- **Low**: Minor issues, code smells, or potential improvements

---

## Findings

### 1. Race Condition in Module Path Resolution Cache Lookup

**Severity**: Medium
**Location**: `internal/orchestration/composer.go:608-620`

**Issue**: The `resolveModulePath` function reads the cache directory to find version directories without any synchronization. If multiple `start` processes run concurrently and the CUE module cache is being updated (e.g., by `cue mod get`), the directory listing could be in an inconsistent state.

**Code**:
```go
entries, err := os.ReadDir(filepath.Dir(moduleBase))
if err != nil {
    return "", fmt.Errorf("reading cache directory: %w", err)
}

baseName := filepath.Base(origin)
var moduleDir string
for _, entry := range entries {
    if entry.IsDir() && strings.HasPrefix(entry.Name(), baseName+"@v") {
        moduleDir = filepath.Join(filepath.Dir(moduleBase), entry.Name())
        break
    }
}
```

**Impact**: In edge cases with concurrent CUE cache updates, the function might find a partially-written directory or miss a newly-added version.

**Recommendation**: This is mitigated by the fact that CUE's cache uses atomic operations, but the code could add retry logic or validate the selected directory exists before use.

---

### 2. Potential Nil Map Write in Detection Goroutine

**Severity**: Low (Protected by mutex, but could be cleaner)
**Location**: `internal/detection/agent.go:31-52`

**Issue**: While the mutex properly protects the append operation, the pattern of closing over map entries in goroutines can be error-prone. The current implementation is correct because Go 1.22+ changed loop variable semantics, but this could break if compiled with older Go versions.

**Code**:
```go
for key, entry := range index.Agents {
    if entry.Bin == "" {
        continue
    }
    wg.Add(1)
    go func(k string, e registry.IndexEntry) {  // Correctly passes by value
        defer wg.Done()
        // ...
    }(key, entry)
}
```

**Impact**: None with current Go versions (1.22+), but potential issue with older compilers.

**Recommendation**: The code correctly passes values as function parameters. Add a `go.mod` minimum version check or comment explaining the Go version dependency.

---

### 3. Missing Validation for Empty File Content in Temp File Creation

**Severity**: Low
**Location**: `internal/temp/manager.go:89-102`

**Issue**: The `WriteUTDFile` function doesn't validate that content is non-empty before writing. This could create empty temp files which might confuse downstream processing.

**Code**:
```go
func (m *Manager) WriteUTDFile(entityType, name, content string) (string, error) {
    if err := m.EnsureUTDDir(); err != nil {
        return "", err
    }
    fileName := deriveFileName(entityType, name)
    filePath := filepath.Join(m.BaseDir, fileName)
    if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
        return "", fmt.Errorf("writing UTD file: %w", err)
    }
    return filePath, nil
}
```

**Impact**: Empty temp files could be created, which may cause confusion but won't cause data corruption.

**Recommendation**: Consider adding a check or documenting that empty content is acceptable. The current behavior may be intentional for placeholder files.

---

### 4. Unsafe Tilde Expansion Edge Case

**Severity**: Low
**Location**: `internal/orchestration/executor.go:27-45` and `internal/orchestration/filepath.go:22-38`

**Issue**: The `expandTilde` function in `executor.go` doesn't handle the `~username` syntax (e.g., `~root/file`), returning it unexpanded. This could cause confusion if users expect shell-like tilde expansion.

**Code**:
```go
func expandTilde(path string) string {
    // ...
    if len(path) > 1 && path[0] == '~' && path[1] == '/' {
        if home, err := os.UserHomeDir(); err == nil {
            return home + path[1:]
        }
    }
    return path
}
```

**Impact**: `~user/path` style paths will not be expanded, potentially causing "file not found" errors. This is documented behavior per DR-038.

**Recommendation**: The current behavior is consistent and documented. Consider adding a warning when `~username` syntax is detected but not expanded.

---

### 5. Template Parsing Could Panic on Malformed Input

**Severity**: Low
**Location**: `internal/orchestration/template.go:177-189`

**Issue**: While Go's `text/template` package handles most malformed input gracefully, certain edge cases with deeply nested or malformed templates could potentially cause panics (though this is rare).

**Code**:
```go
tmpl, err := template.New("utd").Option("missingkey=zero").Parse(templateStr)
if err != nil {
    return result, fmt.Errorf("parsing template: %w", err)
}

var buf bytes.Buffer
if err := tmpl.Execute(&buf, data); err != nil {
    return result, fmt.Errorf("executing template: %w", err)
}
```

**Impact**: Extremely rare potential for panics with maliciously crafted templates.

**Recommendation**: Consider wrapping template parsing/execution in a recover() block for production use, though the current error handling is generally sufficient.

---

### 6. CUE Loader Map Iteration Order Assumption

**Severity**: Medium
**Location**: `internal/cue/loader.go:117-206`

**Issue**: The `mergeWithReplacement` function tracks field order using slices (`topLevelOrder`, `itemOrder`) which is correct. However, the assumption that CUE's `Fields()` iterator returns fields in a stable order is relied upon but not explicitly documented.

**Code**:
```go
for iter.Next() {
    key := iter.Selector().String()
    fieldValue := iter.Value()

    if _, exists := topLevel[key]; !exists {
        topLevel[key] = make(map[string]cue.Value)
        topLevelOrder = append(topLevelOrder, key)
        itemOrder[key] = nil
    }
    // ...
}
```

**Impact**: If CUE's iteration order is not stable, merged configurations could have unpredictable field ordering (though values would still be correct).

**Recommendation**: CUE's `Fields()` iterator does maintain definition order, so this is correct. Add a comment noting this dependency on CUE's behavior.

---

### 7. Shell Runner Process Group Kill Race Condition

**Severity**: Medium
**Location**: `internal/shell/runner.go:113-119`

**Issue**: When a command times out, the code attempts to kill the process group. However, there's a race between checking `cmd.Process != nil` and the process potentially exiting naturally.

**Code**:
```go
if ctx.Err() == context.DeadlineExceeded {
    result.TimedOut = true
    if cmd.Process != nil {
        _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
    }
    return result, fmt.Errorf("command timed out after %d seconds", timeoutSecs)
}
```

**Impact**: The `syscall.Kill` call might fail with ESRCH if the process has already exited, but this error is intentionally ignored (underscore). This is the correct behavior.

**Recommendation**: The current implementation correctly ignores the error. Consider logging at debug level for diagnostics.

---

### 8. Gitignore Check Only Looks at Top-Level .gitignore

**Severity**: Low
**Location**: `internal/temp/manager.go:153-172`

**Issue**: The `CheckGitignore` function only checks the top-level `.gitignore` file and doesn't consider `.gitignore` files in subdirectories or global gitignore configuration.

**Code**:
```go
func CheckGitignore(workingDir string) bool {
    gitignorePath := filepath.Join(workingDir, ".gitignore")
    content, err := os.ReadFile(gitignorePath)
    if err != nil {
        return false
    }
    // ...
}
```

**Impact**: Users with `.start/temp` ignored via global gitignore or subdirectory gitignore files won't get proper detection.

**Recommendation**: Consider using `git check-ignore` command for accurate detection, or document the limitation.

---

### 9. Potential Integer Overflow in Line Number Calculation

**Severity**: Very Low
**Location**: `internal/cue/errors.go:131-135`

**Issue**: The `generateSourceContext` function calculates `startLine = line - contextLines` which could theoretically underflow for very large line numbers, though this is practically impossible.

**Code**:
```go
startLine := line - contextLines
if startLine < 1 {
    startLine = 1
}
```

**Impact**: None in practice; the check `startLine < 1` handles the relevant case.

**Recommendation**: No action needed; the current code is correct.

---

### 10. Inconsistent Error Wrapping Patterns

**Severity**: Low (Style/Consistency)
**Location**: Various files

**Issue**: Some functions use `fmt.Errorf("message: %w", err)` while others use `fmt.Errorf("message %s: %w", path, err)`. The inconsistency is minor but could affect error message parsing.

**Examples**:
- `internal/config/paths.go:66`: `fmt.Errorf("checking directory %s: %w", dir, err)`
- `internal/orchestration/composer.go:63`: `fmt.Errorf("reading %s file %s: %w", entityType, filePath, err)`

**Impact**: Inconsistent error message formatting, which can affect log parsing.

**Recommendation**: Establish a consistent error message format across the codebase.

---

### 11. Version Resolution Logic May Select Wrong Version

**Severity**: Medium
**Location**: `internal/registry/client.go:86-133`

**Issue**: The `ResolveLatestVersion` function assumes versions are returned in semver order with the latest last. If the registry returns versions in a different order, the wrong version could be selected.

**Code**:
```go
versions, err := c.ModuleVersions(ctx, modulePath)
// ...
latestVersion := versions[len(versions)-1]
```

**Impact**: Could select an older version if the registry doesn't return versions in sorted order.

**Recommendation**: Add explicit semver sorting of the returned versions before selecting the latest, or document the dependency on registry behavior.

---

### 12. File Permission Not Checked Before Reading

**Severity**: Low
**Location**: `internal/orchestration/template.go:63-67`

**Issue**: The `DefaultFileReader.Read` function reads files without checking permissions first. If a file exists but is not readable, the error message may be confusing.

**Code**:
```go
content, err := os.ReadFile(path)
if err != nil {
    return "", err
}
```

**Impact**: Confusing error messages when files are not readable due to permissions.

**Recommendation**: The standard library's error message is generally sufficient, but consider wrapping with more context for user-facing errors.

---

## Positive Observations

### Well-Designed Error Handling

The codebase consistently uses error wrapping with `%w` for proper error chain propagation. Functions return errors rather than panicking, making the code robust.

### Good Test Coverage

The test files demonstrate comprehensive coverage of edge cases, including:
- Empty inputs
- Nonexistent files
- Template parsing errors
- Timeout scenarios
- Tilde expansion
- Local vs external file handling

### Proper Concurrency Patterns

The `internal/detection/agent.go` correctly uses sync.Mutex and WaitGroup for concurrent agent detection.

### Consistent CUE Integration

The CUE loading and validation logic is well-structured with clear separation between loading, merging, and validation.

### Shell Escaping Security

The `escapeForShell` function in `executor.go` properly prevents shell injection by wrapping all values in single quotes and escaping internal quotes.

---

## Recommendations Summary

### Priority 1 (Should Address)

1. **Version Resolution**: Add semver sorting in `ResolveLatestVersion` to ensure correct version selection regardless of registry return order
2. **Document CUE Iteration Order**: Add comments noting dependency on CUE's stable field iteration order

### Priority 2 (Consider Addressing)

3. **Tilde Expansion Warning**: Consider warning when `~username` syntax is used but not expanded
4. **Error Message Consistency**: Standardize error message formatting across the codebase

### Priority 3 (Low Priority)

5. **Gitignore Detection**: Consider using `git check-ignore` for more accurate detection
6. **Empty Content Validation**: Document or validate empty content in temp file creation

---

## Conclusion

The `start` CLI codebase demonstrates solid engineering practices with comprehensive error handling, good test coverage, and thoughtful design patterns. The identified issues are mostly edge cases or minor consistency improvements rather than critical bugs. The most significant finding is the version resolution logic that should use explicit semver sorting to ensure correctness regardless of registry behavior.

The codebase is production-ready with the recommended improvements being optimizations rather than critical fixes.
