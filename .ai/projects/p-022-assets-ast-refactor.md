# p-022: Assets AST Refactor

- Status: Pending
- Started: (not yet started)
- Completed: (not yet completed)

## Overview

Refactor asset installation code in `internal/assets/install.go` to use CUE's Abstract Syntax Tree (AST) APIs instead of fragile string manipulation. The current implementation uses string searching, manual parsing, and hardcoded indentation which is unreliable and difficult to maintain.

This project will replace 160+ lines of manual parser code with proper AST manipulation, eliminating the acknowledged hacks and limitations in the current implementation.

## Goals

1. Replace string-based config file manipulation with AST parsing and manipulation
2. Eliminate manual CUE parser implementation (FindAssetKey, FindMatchingBrace, FindOpeningBrace)
3. Remove hardcoded indentation assumptions
4. Use CUE's formatter for all output generation
5. Handle all CUE syntax correctly (strings, comments, escapes, nested structures)

## Scope

In Scope:

- Refactor `AssetExists` to parse and check AST structure
- Refactor `writeAssetToConfig` to build and manipulate AST nodes
- Refactor `UpdateAssetInConfig` to use AST manipulation
- Remove manual parser functions (FindAssetKey, FindMatchingBrace, FindOpeningBrace)
- Update tests to work with AST-based implementation
- Maintain backward compatibility for existing callers

Out of Scope:

- Changing the external API of the assets package
- Modifying test expectations beyond necessary format adjustments
- Refactoring other parts of the codebase that don't use these functions
- Adding new functionality to asset installation

## Success Criteria

- [ ] AssetExists parses CUE files and checks AST (no string matching)
- [ ] writeAssetToConfig builds proper AST nodes and uses format.Node()
- [ ] UpdateAssetInConfig manipulates AST directly
- [ ] FindAssetKey, FindMatchingBrace, FindOpeningBrace removed
- [ ] All tests pass with AST-based implementation
- [ ] Manual test: install, update, and check asset existence work correctly
- [ ] No hardcoded indentation logic remains
- [ ] Code review confirms no string manipulation of CUE content

## Current State

Verified: 2026-02-03

The current implementation in `internal/assets/install.go` has several problematic patterns:

String-Based Existence Check (lines 67-80):

```go
func AssetExists(configDir, category, name string) bool {
    data, err := os.ReadFile(configPath)
    // ...
    return strings.Contains(existingContent, fmt.Sprintf("%q:", assetKey)) ||
        strings.Contains(existingContent, assetKey+":")
}
```

Issues: False positives if asset name appears in comments or descriptions.

Acknowledged Hack in writeAssetToConfig (lines 286-306):

```go
// This is a simple approach - for complex files might need proper parsing
closingBrace := strings.LastIndex(existingContent, "}")
```

Issues: Will break with nested structures, uses naive string search for brace matching.

Manual CUE Parser (lines 383-607):

- FindAssetKey: 70 lines of state machine for finding keys
- FindMatchingBrace: 67 lines for brace matching
- FindOpeningBrace: 58 lines for finding opening braces

Issues: Reimplements CUE parsing, doesn't handle all escape sequences, will have edge cases.

Hardcoded Indentation (lines 647-667):

```go
// LIMITATION: This assumes a flat config structure where assets are direct children
// of the category (e.g., tasks: { "name": { ... } }). It adds exactly one tab to
// each line (except the first).
lines := strings.Split(newContent, "\n")
for i, line := range lines {
    if i == 0 {
        sb.WriteString(line)
    } else {
        if line != "" {
            sb.WriteString("\n\t" + line)  // Hardcoded single tab
```

Issues: Explicitly documented limitation, won't work with nested structures.

Code Duplication (internal/cli/assets_update.go:255-408):

150+ lines of extractAssetContent, formatAssetStruct, formatFieldValue duplicated from assets package.

Issues: Violates DRY, creates maintenance burden when logic changes.

## Deliverables

Files:

- Updated `internal/assets/install.go` - AST-based implementation
- Updated `internal/assets/install_test.go` - Tests for AST approach
- Updated `internal/cli/assets_update.go` - Use exported functions, remove duplication

Documentation:

- Updated comments explaining AST-based approach
- Removed "LIMITATION" and "simple approach" comments

Tests:

- All existing tests pass or have expectations updated for formatter output
- Manual verification that install, update, and check operations work

## Technical Approach

Replace String Operations with AST:

1. Use `parser.ParseFile()` to read existing config files
2. Use `ast.Field` and `ast.StructLit` to build/modify structures
3. Use `format.Node()` to generate output (handles indentation automatically)
4. Use `ast.LabelName()` to extract field labels safely

Key Changes:

AssetExists:
```go
// Before: strings.Contains(content, assetKey+":")
// After: Parse file, iterate AST nodes, check LabelName()
```

writeAssetToConfig:
```go
// Before: String concatenation with manual indentation
// After: Build ast.File with ast.Field nodes, format.Node()
```

UpdateAssetInConfig:
```go
// Before: FindAssetKey + FindMatchingBrace + string replacement
// After: Parse, find field in AST, replace field.Value, format.Node()
```

Remove Manual Parsers:

Delete FindAssetKey, FindMatchingBrace, FindOpeningBrace entirely - AST operations don't need them.

Backward Compatibility:

Keep the same function signatures where possible. Internal implementation changes should be transparent to callers.

## Dependencies

Requires:

- p-021 (Auto-Setup Default Assets) - Must be completed and committed first

## Decision Points

None currently - approach is clear from code review.

## Testing Strategy

Unit Tests:

- Update tests to parse string fixtures into AST structures before passing to functions
- Verify AST-based functions produce correct CUE output
- Test edge cases: nested structures, comments, special characters

Integration Tests:

- Install asset to new config file
- Install asset to existing config file with other assets
- Update existing asset
- Check asset existence (positive and negative cases)
- Verify formatted output is valid CUE

Manual Tests:

1. Fresh install: `start assets add cwd/agents-md` into empty config
2. Update: `start assets update cwd/agents-md` with newer version
3. Duplicate check: Try to add same asset twice (should error)
4. Validation: Run `cue vet` on generated config files

## Notes

Why AST is Better:

- Robust: Handles all CUE syntax correctly (CUE's parser does this)
- Maintainable: No manual parsing state machines
- Reliable: No false positives from string matching
- Correct indentation: Formatter handles spacing automatically
- Future-proof: Works with new CUE features automatically

Previous Attempt (2026-02-03):

Initial refactoring attempt revealed:
- Need to properly understand CUE's AST structure for file comments
- Formatter produces slightly different output than tests expected
- Some tests may need updated expectations for formatter output

These are minor issues to resolve during implementation, not blockers.

Alternative Considered:

Continue with string manipulation and fix the edge cases individually.

Rejected because: Band-aid approach. Each fix creates new complexity. The root problem is not using the right tool (AST) for the job.
