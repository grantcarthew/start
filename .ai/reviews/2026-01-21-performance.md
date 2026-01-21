# Go Performance Code Review

**Date:** 2026-01-21
**Reviewer:** Claude Opus 4.5
**Scope:** Deep analysis of code efficiency and resource usage

## Executive Summary

The `start` codebase demonstrates solid performance foundations with appropriate use of Go idioms. The application is primarily I/O bound (CUE loading, file operations, registry fetches) rather than CPU bound. Most execution paths are single-shot CLI operations where startup latency matters more than sustained throughput.

**Overall Assessment:** Good performance hygiene with minor optimisation opportunities.

**Key Findings:**

- Hot paths correctly identified: CUE loading, template processing, file I/O
- No memory leaks or resource lifecycle issues detected
- Minor allocation improvements possible in string building and slice operations
- Regex compilation could be cached at package level
- No concurrency issues (single-threaded CLI tool)

---

## 1. Hot Path Analysis

### Identified Hot Paths

1. **CUE Configuration Loading** (`internal/cue/loader.go`)
   - Called on every command execution
   - Performs file I/O, CUE parsing, and merging
   - Most expensive operation in typical usage

2. **Prompt Composition** (`internal/orchestration/composer.go`)
   - Template processing for contexts, roles, and tasks
   - File reading and content assembly
   - Called once per execution

3. **Template Processing** (`internal/orchestration/template.go`)
   - Go template parsing and execution
   - UTD field resolution
   - Called for each context/role/task

4. **Registry Operations** (`internal/registry/client.go`, `internal/registry/index.go`)
   - Network I/O to CUE registry
   - Only called when needed (auto-setup, task installation)
   - Retry logic with exponential backoff

### Performance Characteristics

The tool follows a single-request model:
1. Load configuration (most expensive)
2. Process templates and compose prompt
3. Execute agent via `syscall.Exec` (process replacement)

The use of `syscall.Exec` for agent execution is an excellent design choice - zero wrapper overhead.

---

## 2. Allocations Review

### Observations

#### Slice Allocations

**Finding:** Most slice operations lack capacity hints.

| Location | Issue | Impact |
|----------|-------|--------|
| `composer.go:117` | `var promptParts []string` | Minor - typically <10 items |
| `task.go:237-239` | `contextNames` slice appending | Minor - single allocation per context |
| `loader.go:54` | `var values []cue.Value` | Minor - typically 2 items max |

**Example:**
```go
// Current (composer.go:117)
var promptParts []string

// Improved (if perf-critical)
promptParts := make([]string, 0, 8) // estimate based on typical usage
```

**Verdict:** Not worth optimising. The overhead is negligible for typical context counts (<10).

#### Map Allocations

**Finding:** Maps are correctly pre-sized where beneficial.

```go
// Good: loader.go:147-149
topLevel := make(map[string]map[string]cue.Value)
itemOrder := make(map[string][]string)
```

```go
// Good: executor.go:139-146
data := CommandData{
    "bin":       escapeForShell(expandTilde(cfg.Agent.Bin)),
    // ... literal initialization
}
```

```go
// Good: index.go:80-85
idx := &Index{
    Agents:   make(map[string]IndexEntry),
    Tasks:    make(map[string]IndexEntry),
    Roles:    make(map[string]IndexEntry),
    Contexts: make(map[string]IndexEntry),
}
```

**Verdict:** Good practice followed consistently.

#### Struct Copies

**Finding:** No large struct copies on hot paths. Composer and Executor use pointer receivers.

---

## 3. String Operations Review

### Findings

#### String Building

**Good:** `strings.Builder` used correctly for multi-line output:

```go
// executor.go:363-376
func GenerateDryRunCommand(...) string {
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf(...))
    // ...
}
```

```go
// autosetup.go:131-162
func (a *AutoSetup) noAgentsError(...) error {
    var sb strings.Builder
    // ...
}
```

```go
// loader.go:209-251 - Correct Builder usage for CUE source generation
var sb strings.Builder
sb.WriteString("{\n")
```

#### String Concatenation

**Finding:** Some concatenation in non-hot paths is acceptable:

```go
// composer.go:206
result.Prompt = strings.Join(promptParts, "\n\n")  // Single allocation, correct
```

#### fmt.Sprintf in Hot Paths

**Observation:** `fmt.Sprintf` is used moderately. Most occurrences are for error messages or logging, not hot path data processing.

```go
// template.go:147-148 - Called once per template
data := TemplateData{
    "date": time.Now().Format(time.RFC3339),  // OK - single allocation
}
```

**Verdict:** String operations are efficient.

### Regex Compilation

**Issue:** Regex patterns are compiled at package level - this is correct.

```go
// executor.go:21-25
var quotedPlaceholderPattern = regexp.MustCompile(`...`)
var singleBracePlaceholderPattern = regexp.MustCompile(`...`)
```

```go
// temp/manager.go:114
reg := regexp.MustCompile(`[^a-zA-Z0-9-_.]`)  // Called per file write
```

**Recommendation:** Move `temp/manager.go:114` regex to package level:

```go
// Package level (currently compiled each call)
var unsafeCharPattern = regexp.MustCompile(`[^a-zA-Z0-9-_.]`)

func deriveFileName(entityType, name string) string {
    // Use unsafeCharPattern instead of compiling inline
}
```

**Impact:** Minor - `deriveFileName` is called once per UTD file write.

---

## 4. Memory Management Review

### Memory Lifecycle

**Finding:** No memory leaks detected. The application follows Go's garbage collection model appropriately.

#### Potential Concerns Checked:

1. **Growing maps/slices:** None found. All collections are scoped to request lifecycle.

2. **Reference holding:** No long-lived references to large objects.

3. **Buffer reuse:** Not applicable - single-shot CLI, not a server.

### Large Allocations

**Finding:** File contents are read into memory but immediately processed:

```go
// template.go:114
content, err := p.fileReader.Read(fields.File)  // Full file in memory
```

```go
// filepath.go:48
content, err := os.ReadFile(expanded)  // Full file in memory
```

**Assessment:** Acceptable for prompt files (typically <100KB). Streaming would add complexity without benefit.

---

## 5. Resource Lifecycle Review

### File Handles

**Finding:** File operations use `os.ReadFile` and `os.WriteFile` which handle closing internally.

```go
// template.go:63
content, err := os.ReadFile(path)  // Correct - no leak possible
```

```go
// temp/manager.go:69
if err := os.WriteFile(path, []byte(content), 0600); err != nil {
    return fmt.Errorf("writing %s: %w", name, err)
}
```

**One concern checked:**

```go
// doctor/checks.go:449-454
f, err := os.Create(testFile)
if err != nil {
    return false
}
_ = f.Close()
_ = os.Remove(testFile)
```

**Verdict:** Correctly closed. The `_ =` pattern is fine for cleanup code.

### Process Handles

```go
// shell/runner.go:99
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
```

**Good:** Process group is set for proper child cleanup on timeout.

```go
// shell/runner.go:116-118
if cmd.Process != nil {
    _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
```

**Correct:** Entire process group killed on timeout.

### Registry Client

```go
// registry/client.go:22-31
func NewClient() (*Client, error) {
    reg, err := modconfig.NewRegistry(nil)
    // ...
}
```

**Assessment:** The registry client uses CUE's modconfig.Registry which manages its own connection pool. No explicit cleanup required for HTTP connections.

---

## 6. I/O Efficiency Review

### File I/O

**Finding:** No buffered I/O used, but not needed for typical file sizes.

```go
// Current approach - fine for prompt files
content, err := os.ReadFile(path)
```

**Would only matter for:** Files >1MB, which prompt files should never be.

### Network I/O

**Finding:** Registry operations use CUE's built-in client with proper retry logic.

```go
// registry/client.go:54-76
for attempt := 0; attempt < c.retries; attempt++ {
    if attempt > 0 {
        wait := c.baseWait * time.Duration(1<<(attempt-1))
        // ... exponential backoff
    }
}
```

**Good:** Exponential backoff prevents thundering herd.

### Batching

**Finding:** No batching opportunities identified. All file reads are independent and sequential.

---

## 7. Algorithmic Complexity Review

### CUE Merging

```go
// loader.go:136-260 - mergeWithReplacement
func (l *Loader) mergeWithReplacement(values []cue.Value) (cue.Value, error)
```

**Complexity:** O(n * m) where n = config items, m = fields per item.

**Assessment:** Acceptable. Typical configs have <50 total items.

### Context Selection

```go
// composer.go:252-334 - selectContexts
for iter.Next() {  // O(n) contexts
    for _, tag := range ctx.Tags {  // O(t) tags per context
        if tagSet[tag] {  // O(1) lookup
```

**Complexity:** O(n * t) where n = contexts, t = tags per context.

**Assessment:** Optimal - map lookup for tag matching.

### Task Resolution

```go
// task.go:466-490 - mergeTaskMatches
installedNames := make(map[string]bool)  // O(1) lookup
for _, m := range installed {
    installedNames[m.Name] = true
}
// ...
sort.Slice(merged, ...)  // O(n log n)
```

**Complexity:** O(n log n) for merge and sort.

**Assessment:** Optimal.

### Module Path Resolution

```go
// composer.go:605-627 - resolveModulePath
entries, err := os.ReadDir(filepath.Dir(moduleBase))  // O(d) directory entries
for _, entry := range entries {
    if strings.HasPrefix(entry.Name(), baseName+"@v") {
```

**Complexity:** O(d) where d = entries in CUE cache directory.

**Assessment:** Could be slow if cache has many versions. Consider caching resolution result.

---

## 8. Caching Review

### Current Caching

**Finding:** No explicit caching implemented. Each `start` invocation:
1. Loads configuration from disk
2. Parses CUE files
3. Processes templates

### Caching Opportunities

| What | Benefit | Complexity | Recommendation |
|------|---------|------------|----------------|
| Parsed CUE config | Skip re-parsing | High | Not worth it - CLI tool, no persistent process |
| Compiled templates | Faster template execution | Medium | Not worth it - templates vary per context |
| Registry index | Avoid network call | Medium | Could cache with TTL, but registry is only fetched on-demand |

**Verdict:** Caching not recommended for a CLI tool. Each invocation is independent and caching would add state management complexity.

---

## 9. Concurrency Review

### Current Model

The application is single-threaded:
- No goroutines for parallel file loading
- No worker pools
- Sequential processing throughout

### Potential Parallelism

1. **Context resolution:** Could load multiple contexts in parallel.

```go
// Current (sequential)
for iter.Next() {
    resolved, err := c.resolveContext(cfg, ctx.Name)
}

// Potential parallel approach
var wg sync.WaitGroup
for iter.Next() {
    wg.Add(1)
    go func(name string) {
        defer wg.Done()
        resolved, err := c.resolveContext(cfg, name)
    }(ctx.Name)
}
wg.Wait()
```

**Recommendation:** Not worth it. Typical context count is 2-5, and file I/O is the bottleneck.

### Thread Safety

**Finding:** No shared mutable state. Package-level variables are:
- `regexp.MustCompile` results (read-only after init)
- `color.NoColor` flag (set once during startup)

**Verdict:** No concurrency issues possible.

---

## 10. Serialisation Review

### CUE Encoding/Decoding

**Primary serialisation:** CUE value extraction using CUE's reflection.

```go
// executor.go:314-336
if models := agentVal.LookupPath(cue.ParsePath("models")); models.Exists() {
    iter, err := models.Fields()
    for iter.Next() {
        // Field-by-field extraction
    }
}
```

**Assessment:** Uses CUE's native extraction. No JSON/YAML parsing overhead.

### Index Decoding

```go
// index.go:119
if err := iter.Value().Decode(&entry); err != nil {
```

**Finding:** Uses CUE's Decode method for struct population. This is optimal for CUE values.

### Template Execution

```go
// executor.go:148-157
tmpl, err := template.New("command").Parse(cfg.Agent.Command)
var buf bytes.Buffer
if err := tmpl.Execute(&buf, data); err != nil {
```

**Finding:** Go's text/template is used correctly. Template is parsed once per execution.

**Potential Optimisation:** Cache parsed templates by template string:

```go
var templateCache sync.Map

func getCachedTemplate(name, tmplStr string) (*template.Template, error) {
    if cached, ok := templateCache.Load(tmplStr); ok {
        return cached.(*template.Template), nil
    }
    tmpl, err := template.New(name).Parse(tmplStr)
    if err != nil {
        return nil, err
    }
    templateCache.Store(tmplStr, tmpl)
    return tmpl, nil
}
```

**Recommendation:** Not worth it for CLI. Template parsing is fast (~100μs).

---

## Recommendations Summary

### Actionable Items

| Priority | Issue | Location | Recommendation |
|----------|-------|----------|----------------|
| Low | Regex compiled per call | `temp/manager.go:114` | Move to package level |
| Low | Slice capacity hints | Various | Add capacity where size is predictable |

### Not Recommended

| Item | Reason |
|------|--------|
| Parallel context loading | Overhead exceeds benefit for typical usage |
| Template caching | Single-shot CLI, no benefit |
| Configuration caching | CLI tool, no persistent state |
| Buffered I/O | File sizes are small |

---

## Profiling Recommendations

If performance concerns arise, profile with:

```bash
# Build with profiling
go build -o start-profile ./cmd/start

# CPU profile
go test -cpuprofile=cpu.prof -bench=. ./internal/...

# Memory profile
go test -memprofile=mem.prof -bench=. ./internal/...

# Trace
go test -trace=trace.out ./internal/...
go tool trace trace.out
```

**Key areas to profile:**
1. CUE loading (`internal/cue/loader.go`) - likely dominant
2. Template processing (`internal/orchestration/template.go`)
3. File I/O in context resolution

---

## Benchmarking Suggestions

Add benchmarks for hot paths:

```go
// loader_test.go
func BenchmarkLoad(b *testing.B) {
    loader := NewLoader()
    dirs := []string{"testdata/global", "testdata/local"}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = loader.Load(dirs)
    }
}

// composer_test.go
func BenchmarkCompose(b *testing.B) {
    // Setup composer and config
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = composer.Compose(cfg, selection, "", "")
    }
}
```

---

## Conclusion

The `start` codebase has good performance characteristics for a CLI tool:

1. **Correct hot path identification:** CUE loading is the bottleneck, which is appropriate.
2. **No memory leaks:** Resource lifecycle is properly managed.
3. **Efficient algorithms:** Map lookups where appropriate, no O(n²) issues.
4. **Clean execution model:** Process replacement via `syscall.Exec` eliminates wrapper overhead.

The minor optimisations identified (regex caching, slice capacity hints) would provide marginal benefit. The tool's performance is primarily bounded by CUE's parsing speed and file I/O, which are external to this codebase.

**Performance grade: B+**

No critical issues. Minor improvements possible but not impactful for typical usage patterns.
