# Code Change Review Report

**Commit**: 6793ca8d6ef0c6de2e537bb3bc812025c3c43aab
**Date**: 2026-01-16
**Feature**: CLI File Path Support for Roles, Contexts, Tasks, and Prompts
**Reviewer**: AI Code Analysis
**Review Date**: 2026-01-18
**Status**: ⚠️ **PRODUCTION BLOCKER IDENTIFIED**

---

## Executive Summary

This commit implements file path support for CLI inputs (roles, contexts, tasks, prompts) as specified in DR-038. The implementation includes 971 additions and 91 deletions across 12 files, with comprehensive test coverage for the new functionality.

**Critical Finding**: A breaking change in context selection logic was introduced that alters existing behavior when users specify context tags with the `start` command. This is a **production blocker** that must be addressed before release.

### Quality Assessment

| Category | Status | Notes |
|----------|--------|-------|
| Implementation | ✅ Good | Clean, well-structured code |
| Test Coverage | ✅ Excellent | Comprehensive tests for new features |
| Documentation | ✅ Complete | Three new CLI docs created |
| Regression Risk | ❌ **HIGH** | Breaking change in core functionality |
| DR-038 Compliance | ✅ Full | All requirements met |

---

## Scope of Changes

### Files Modified (12 files, +971/-91 lines)

**Implementation Files**:
- `internal/orchestration/filepath.go` (new, 54 lines)
- `internal/orchestration/filepath_test.go` (new, 147 lines)
- `internal/orchestration/composer.go` (+124/-91 lines)
- `internal/cli/task.go` (+100/-91 lines)
- `internal/cli/prompt.go` (+15/-9 lines)
- `internal/cli/root.go` (+4/-4 lines)

**Test Files**:
- `internal/cli/start_test.go` (+307 lines)

**Documentation Files**:
- `docs/cli/start.md` (new, 77 lines)
- `docs/cli/task.md` (new, 85 lines)
- `docs/cli/prompt.md` (new, 64 lines)

**Project Tracking**:
- `.ai/projects/p-019-cli-file-path-inputs.md` (+84 lines)
- `docs/thoughts.md` (+1 line)

---

## Test Results

All tests pass successfully:

```
✓ internal/orchestration - 48 tests PASS (0.026s)
✓ internal/cli - 61 tests PASS (0.148s)
✓ New file path tests - 8 tests PASS
  - TestIsFilePath (18 cases)
  - TestExpandFilePath (5 cases)
  - TestReadFilePath (2 cases)
  - TestExecuteStart_FilePathRole
  - TestExecuteStart_FilePathContext
  - TestExecuteStart_MixedContextOrder
  - TestExecuteTask_FilePathTask
  - TestExecuteTask_FilePathWithInstructions
  - TestExecuteTask_FilePathMissing
  - TestExecuteStart_FilePathContextMissing
```

**Note**: While all tests pass, the test suite does not cover the regression scenario identified below.

---

## Critical Issue: Context Selection Regression

### Problem Statement

The refactoring of `internal/orchestration/composer.go:Compose()` introduces a **breaking change** in how contexts are selected when users provide explicit tags to the `start` command.

### Behavior Change

**Previous Behavior** (before this commit):
```bash
$ start -c security
# Includes: required contexts + default contexts + security-tagged contexts
```

**New Behavior** (after this commit):
```bash
$ start -c security
# Includes: required contexts + security-tagged contexts
# MISSING: default contexts are excluded!
```

### Root Cause Analysis

**Location**: `internal/orchestration/composer.go:125-135`

```go
// Second: add default contexts if IncludeDefaults and no explicit tags
if selection.IncludeDefaults && len(selection.Tags) == 0 {
    defaultSelection := ContextSelection{IncludeDefaults: true}
    contexts, err := c.selectContexts(cfg, defaultSelection)
    // ... add contexts
}
```

The condition `len(selection.Tags) == 0` means default contexts are **only** included when **no tags** are provided. This differs from the original `selectContexts()` logic which included default contexts whenever `IncludeDefaults` was true, regardless of whether tags were present.

**Original Logic** (HEAD~1):
```go
// Default contexts included if IncludeDefaults is set
if selection.IncludeDefaults && ctx.Default {
    include = true
}
```

This was an **OR** condition - contexts could be included by being default OR by matching a tag.

**New Logic**:
```go
// Defaults only if IncludeDefaults AND Tags is empty
if selection.IncludeDefaults && len(selection.Tags) == 0 {
    // ... add defaults
}
```

This is now a **mutually exclusive** condition - defaults are skipped when tags are present.

### Impact Assessment

**Affected Command**: `start` (internal/cli/start.go:124-128)

```go
orchestration.ContextSelection{
    IncludeRequired: true,
    IncludeDefaults: true,  // This is now conditionally ignored!
    Tags:            flags.Context,
}
```

**User Impact**:
- Users who run `start -c <tag>` will no longer get default contexts
- This may break existing workflows that depend on default contexts always being present
- Users must now explicitly add `-c default,<tag>` to get previous behavior

**Severity**: **HIGH** - This is a breaking change in core functionality that affects the primary user-facing command.

### Why Tests Didn't Catch This

The test suite has no coverage for the case where `IncludeDefaults=true` **AND** `Tags` is non-empty.

Existing test in `composer_test.go:70-93`:
```go
{
    name: "tagged contexts",
    selection: ContextSelection{
        IncludeRequired: true,
        Tags:            []string{"security"},
        // IncludeDefaults is NOT set in this test
    },
}
```

**Missing Test Case**:
```go
{
    name: "tagged contexts with defaults",
    selection: ContextSelection{
        IncludeRequired: true,
        IncludeDefaults: true,  // This combination is not tested!
        Tags:            []string{"security"},
    },
}
```

---

## Implementation Quality Review

### File Path Detection (`filepath.go`)

**Quality**: ✅ **Excellent**

```go
func IsFilePath(s string) bool {
    return strings.HasPrefix(s, "./") ||
           strings.HasPrefix(s, "/") ||
           strings.HasPrefix(s, "~")
}
```

- Simple, efficient detection rule per DR-038
- Well-tested with 18 test cases covering edge cases
- Clear documentation

**Potential Concern**: Tilde expansion uses `filepath.Join(home, path[1:])` which works correctly but is subtle. When `path="~/file"`, `path[1:]="/file"`, and `filepath.Join` intelligently handles the leading slash to produce `/home/user/file`. This is correct behavior but could benefit from a comment explaining the non-obvious interaction.

### Task File Path Support (`task.go`)

**Quality**: ✅ **Good**

Changes cleanly separate file-based and config-based task resolution:

```go
if orchestration.IsFilePath(taskName) {
    // File path handling
    content, err := orchestration.ReadFilePath(taskName)
    taskResult, err = env.Composer.ProcessContent(content, instructions)
} else {
    // Config-based task handling (existing logic)
}
```

**Strengths**:
- Error messages include file paths for debugging
- Template processing preserves `{{.instructions}}` support for file-based tasks
- Proper integration with registry fallback logic

### Role and Context File Path Support (`composer.go`)

**Quality**: ⚠️ **Mixed**

**Good**:
- File path support cleanly integrated via `IsFilePath()` checks
- Error handling consistent with existing patterns (roles→warnings, contexts→○ status)
- Order preservation for mixed file/config contexts

**Problematic**:
- Context selection refactoring introduced regression (see Critical Issue above)
- The new three-pass context selection algorithm (required, defaults, tags) is more complex than necessary
- Lack of comments explaining the new ordering logic

### Test Coverage

**Quality**: ✅ **Excellent** (for new features)

New functionality has comprehensive test coverage:

- **Unit tests**: `IsFilePath` (18 cases), `ExpandFilePath` (5 cases), `ReadFilePath` (2 cases)
- **Integration tests**: 8 end-to-end tests covering:
  - File path roles
  - File path contexts (including missing files)
  - Mixed file/config context ordering
  - File path tasks with template substitution
  - Error handling for missing files

**Gap**: No test coverage for the regression scenario (IncludeDefaults=true with non-empty Tags).

---

## Documentation Review

### New Documentation Files

All three CLI documentation files are well-written:

**`docs/cli/start.md`** (77 lines):
- Clear examples of file path usage
- Explains file path detection rule
- Documents mixed context syntax

**`docs/cli/task.md`** (85 lines):
- Good coverage of file vs config task usage
- Explains template placeholder support for file-based tasks
- Examples are clear and practical

**`docs/cli/prompt.md`** (64 lines):
- Concise explanation of file path support
- Good examples of both inline and file-based prompts

### Help Text Updates

Flag descriptions updated in `root.go`:
```go
--role, -r    "Override role (config name or file path)"    // ✅ Clear
--context, -c "Select contexts (tags or file paths)"        // ✅ Clear
```

Task and prompt command long descriptions updated with file path information. ✅

---

## Design Specification Compliance (DR-038)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Detection rule: `./`, `/`, `~` prefixes | ✅ | `filepath.go:15-17` |
| Mixed context inputs | ✅ | `composer.go:138-173` |
| Order preservation | ✅ | Test: `TestExecuteStart_MixedContextOrder` |
| File paths shown as-is | ✅ | `Context.Name = tag` for file paths |
| Missing files: contexts→○, tasks/roles→error | ✅ | Error handling verified in tests |
| No file content validation | ✅ | Raw text used without parsing |
| Support for roles | ✅ | `composer.go:197-211` |
| Support for contexts | ✅ | `composer.go:139-154` |
| Support for tasks | ✅ | `task.go:59-77` |
| Support for prompts | ✅ | `prompt.go:30-38` |

**Compliance**: ✅ **100%** - All DR-038 requirements are met.

---

## Additional Observations

### Code Style

✅ **Excellent**:
- Consistent error messages
- Proper use of `debugf()` for tracing
- Clear variable names
- Good separation of concerns

### Error Handling

✅ **Good**:
- File read errors include file path in message
- Missing context files don't cause fatal errors (per DR-007)
- Task file errors are fatal with clear messages

### Security

✅ **No concerns**:
- No path traversal vulnerabilities (uses `filepath` package)
- Tilde expansion uses `os.UserHomeDir()` correctly
- No shell injection risks

### Performance

✅ **No concerns**:
- File reads are synchronous and appropriate for CLI context
- No unnecessary file operations
- Caching not needed for one-time CLI operations

---

## Recommendations

### 1. **CRITICAL - Fix Context Selection Regression** (Priority: P0)

**Issue**: Default contexts excluded when tags are provided to `start` command.

**Solution Option A - Minimal Change** (Recommended):

Modify `composer.go:125-135` to restore original behavior:

```go
// Second: add default contexts if IncludeDefaults
// (Previously: only if Tags is empty, now: always when requested)
if selection.IncludeDefaults {
    // Check if defaults were already added by tag processing
    defaultSelection := ContextSelection{IncludeDefaults: true}
    contexts, err := c.selectContexts(cfg, defaultSelection)
    if err != nil {
        return result, fmt.Errorf("selecting contexts: %w", err)
    }
    for _, ctx := range contexts {
        addConfigContext(ctx)  // This already checks for duplicates
    }
}
```

Remove the `len(selection.Tags) == 0` condition.

**Solution Option B - More Invasive**:

Revert the entire `Compose()` refactoring and implement file path support with minimal changes to the original context selection logic. This would reduce regression risk but require rewriting the file path integration.

**Recommended**: Option A - It's a one-line fix that restores the original behavior while preserving all the new file path functionality.

### 2. **Add Test Coverage for Regression Case** (Priority: P0)

Add test to `composer_test.go`:

```go
{
    name: "tagged contexts still include defaults",
    config: `
        contexts: {
            env: {
                required: true
                prompt: "Environment"
            }
            project: {
                default: true
                prompt: "Project"
            }
            security: {
                tags: ["security"]
                prompt: "Security"
            }
        }
    `,
    selection: ContextSelection{
        IncludeRequired: true,
        IncludeDefaults: true,
        Tags:            []string{"security"},
    },
    wantPrompt: "Environment\n\nProject\n\nSecurity",
    wantCtxs:   []string{"env", "project", "security"},
},
```

This test would fail with the current code and pass after the fix.

### 3. **Add Comment for Tilde Expansion** (Priority: P2 - Nice to have)

In `filepath.go:33`, add explanatory comment:

```go
// Join handles the leading slash in path[1:] correctly.
// E.g., home="/home/user", path[1:]="/file" -> "/home/user/file"
path = filepath.Join(home, path[1:])
```

### 4. **Consider Adding Integration Test** (Priority: P2)

Add a full integration test for `start -c <tag>` to verify default contexts are included. This would catch regressions at a higher level than unit tests.

### 5. **Document Context Selection Behavior** (Priority: P3)

Consider adding a design record or section in documentation that clearly explains context selection ordering and what contexts are included in different scenarios. This would help prevent similar issues in future refactorings.

---

## Code Smells and Technical Debt

### Minor Issues (Non-blocking)

1. **`ProcessContent()` method** (`composer.go:504-512`):
   - New method with minimal documentation
   - Consider adding example usage in godoc comment

2. **Duplicate selectContexts calls** (`composer.go`):
   - The refactoring calls `selectContexts()` multiple times
   - Could be optimized by caching results
   - Not a performance concern in CLI context, but adds complexity

3. **Mixed error handling**:
   - Some calls to `selectContexts()` ignore errors (`contexts, _ := ...`)
   - While non-fatal per DR-007, should be explicitly handled or documented

---

## Conclusion

This commit delivers high-quality implementation of file path support with excellent test coverage and complete documentation. However, it contains a **critical regression** in context selection logic that breaks existing behavior for a core command.

### Release Recommendation: ⛔ **DO NOT RELEASE**

**Rationale**:
- The regression affects the primary `start` command
- Users who specify context tags will lose default contexts
- Breaking change was not intentional or documented
- Simple fix available (remove one condition)

### Before Release Checklist:

- [ ] Fix context selection regression (remove `len(selection.Tags) == 0` condition)
- [ ] Add test case for `IncludeDefaults=true` with non-empty `Tags`
- [ ] Verify all existing tests still pass
- [ ] Run full integration test suite
- [ ] Manual testing of `start -c <tag>` to verify default contexts included

### After Fix:

Once the regression is fixed and tests added, the code quality is **excellent** and ready for production. The implementation is clean, well-tested, properly documented, and fully compliant with DR-038.

---

## Appendix: Test Commands

To verify the fix:

```bash
# Run all tests
GOROOT="" go test ./...

# Run specific regression test (after adding it)
GOROOT="" go test ./internal/orchestration -run "TestComposer_Compose/tagged_contexts_still_include_defaults"

# Manual verification
start -c security  # Should include default contexts
```

---

**Report Generated**: 2026-01-18
**Next Review**: After regression fix is applied
