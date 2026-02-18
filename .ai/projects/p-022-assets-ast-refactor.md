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

- Refactor `writeAssetToConfig` to use AST parsing and manipulation
- Refactor `UpdateAssetInConfig` to use AST manipulation
- Remove manual parser functions (FindAssetKey, FindMatchingBrace, FindOpeningBrace)
- Remove `cueParseState` type and constants
- Remove fuzz tests for deleted functions (`internal/assets/fuzz_test.go`)
- Remove `TestFindOpeningBrace` from `internal/cli/assets_test.go` (lines 669-818)
- Update tests to work with AST-based implementation
- Maintain backward compatibility for existing callers

Already Done (no longer in scope):

- `AssetExists` already uses CUE value lookup (not string matching)
- Code duplication in `assets_update.go` already resolved (calls `assets.ExtractAssetContent` and `assets.UpdateAssetInConfig` directly)

Out of Scope:

- Changing the external API of the assets package
- Modifying test expectations beyond necessary format adjustments
- Refactoring other parts of the codebase that don't use these functions
- Adding new functionality to asset installation

## Success Criteria

- [x] AssetExists uses CUE value lookup (already done)
- [ ] writeAssetToConfig builds proper AST nodes and uses format.Node()
- [ ] UpdateAssetInConfig manipulates AST directly
- [ ] FindAssetKey, FindMatchingBrace, FindOpeningBrace removed
- [ ] cueParseState type and constants removed
- [ ] Fuzz tests for deleted functions removed
- [ ] All tests pass with AST-based implementation
- [ ] Manual test: install, update, and check asset existence work correctly
- [ ] No hardcoded indentation logic remains
- [ ] Code review confirms no string manipulation of CUE content

## Current State

Verified: 2026-02-18

Already Resolved:

- `AssetExists` (install.go:135-138): Already uses CUE value lookup via `cfg.LookupPath()`. No string matching. No changes needed.
- Code duplication in `assets_update.go`: Already resolved. The update command calls `assets.ExtractAssetContent()` and `assets.UpdateAssetInConfig()` directly. File is 292 lines with no duplicated functions.

Remaining Problems in `internal/assets/install.go`:

writeAssetToConfig (lines 398-483): Uses string manipulation to insert assets into config files. Calls `FindAssetKey` to detect duplicates, `FindOpeningBrace` and `FindMatchingBrace` to locate the category block, then manually builds the output with hardcoded `\t` indentation.

UpdateAssetInConfig (lines 692-751): Uses `FindAssetKey` + `FindOpeningBrace` + `FindMatchingBrace` to locate the asset entry, then does string splicing with hardcoded `\t` indentation to replace it.

Manual CUE Parser (lines 485-689): Three state-machine functions totalling ~170 lines:
- `FindAssetKey` (lines 496-555): Character-by-character scanner for finding keys
- `FindMatchingBrace` (lines 568-627): Character-by-character brace matcher
- `FindOpeningBrace` (lines 639-689): Character-by-character brace finder

Supporting types (lines 28-36): `cueParseState` type with four state constants used only by the Find* functions.

Hardcoded indentation in both `writeAssetToConfig` (lines 462-473) and `UpdateAssetInConfig` (lines 735-746): Uses `\t` prefix per line, with LIMITATION comment acknowledging this only works for flat structures.

Test files referencing Find* functions:
- `internal/assets/install_test.go`: TestFindAssetKey, TestFindMatchingBrace, TestFindOpeningBrace, TestFindAssetKey_EmptyKey, TestFindAssetKey_EmptyContent, TestFindMatchingBrace_MultiLineString, TestFindOpeningBrace_MultiLineString
- `internal/assets/fuzz_test.go`: FuzzFindAssetKey, FuzzFindMatchingBrace, FuzzFindOpeningBrace (entire file)
- `internal/cli/assets_test.go`: TestFindOpeningBrace (lines 669-818)

## Deliverables

Files:

- Updated `internal/assets/install.go` - AST-based writeAssetToConfig and UpdateAssetInConfig; Find* functions and cueParseState removed
- Updated `internal/assets/install_test.go` - Find* tests removed; writeAssetToConfig and UpdateAssetInConfig tests updated for formatter output
- Deleted `internal/assets/fuzz_test.go` - All fuzz tests target removed functions
- Updated `internal/cli/assets_test.go` - TestFindOpeningBrace removed

Documentation:

- Updated comments explaining AST-based approach
- Removed "LIMITATION" and "simple approach" comments

Tests:

- All existing tests pass or have expectations updated for formatter output
- Manual verification that install, update, and check operations work

## Technical Approach

CUE AST APIs (cuelang.org/go v0.15.4):

1. `parser.ParseFile("file.cue", source, parser.ParseComments)` - Parse CUE source into `*ast.File`
2. `ast.LabelName(field.Label)` - Extract field name from any label type (returns name, isQuoted, err)
3. `ast.NewStruct(fields ...interface{})` - Build structs; accepts `*ast.Field`, `string`/`ast.Label` + `ast.Expr` pairs
4. `ast.NewString(s)`, `ast.NewBool(b)`, `ast.NewList(exprs...)` - Build literal nodes
5. `ast.NewStringLabel(name)` - Creates quoted or unquoted label as needed (handles `cwd/agents-md` style names)
6. `format.Node(node, format.Simplify())` - Format any AST node to `[]byte` with proper indentation

Key Changes:

writeAssetToConfig:
```go
// Before: String search for category block, manual indentation
// After: parser.ParseFile() -> find/create category struct -> append new field -> format.Node()
// For new files: build ast.File with comment group + category field -> format.Node()
```

UpdateAssetInConfig:
```go
// Before: FindAssetKey + FindMatchingBrace + string splicing
// After: parser.ParseFile() -> walk to find field by LabelName() -> replace field.Value -> format.Node()
```

Asset content (from formatAssetStruct) needs to be parsed into an `ast.Expr` before inserting:
```go
// Parse the content string "{...}" into an AST expression
expr, err := parser.ParseExpr("asset.cue", content)
```

Remove: FindAssetKey, FindMatchingBrace, FindOpeningBrace, cueParseState type and constants.

Backward Compatibility:

Keep the same function signatures. `writeAssetToConfig` and `UpdateAssetInConfig` accept string content; internally parse it to AST. Callers are unaffected.

## Dependencies

Requires:

- p-021 (Auto-Setup Default Assets) - Completed

## Decision Points

1. Comment preservation during AST round-trip

`format.Node()` reformats the entire file when writing back. This means existing files will be normalised to CUE formatter style. Changes may include whitespace adjustments and comment positioning.

Options:
- A. Accept full reformat - simpler implementation, output is always canonical CUE
- B. Preserve original formatting for unmodified sections - significantly more complex, would need to splice AST output into original text for modified sections only

2. Handling the `content` parameter

`writeAssetToConfig` and `UpdateAssetInConfig` receive `content` as a string (e.g., `"{\n\torigin: ...\n}"`). This needs to become an AST node.

Options:
- A. Parse content string with `parser.ParseExpr()` inside the function - no caller changes needed
- B. Change function signatures to accept `ast.Expr` - cleaner but breaks callers (ExtractAssetContent, checkAndUpdate)

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

CUE AST Key Details (verified 2026-02-18):

- `parser.ParseFile` with `parser.ParseComments` preserves comments in AST
- `ast.NewStruct` accepts `*ast.Field` directly or label/value pairs as `...interface{}`
- `ast.NewStringLabel(name)` handles quoting automatically (uses `StringLabelNeedsQuoting`)
- File-level declarations are in `file.Decls []ast.Decl`; top-level fields are `*ast.Field`
- Struct elements are in `structLit.Elts []ast.Decl`; fields are `*ast.Field`
- `format.Node` works on `*ast.File`, `ast.Expr`, or any `ast.Node`

Alternative Considered:

Continue with string manipulation and fix the edge cases individually.

Rejected because: Band-aid approach. Each fix creates new complexity. The root problem is not using the right tool (AST) for the job.

## Estimation

Lines to remove: ~230 (Find* functions ~170, cueParseState ~10, hardcoded indentation ~20, fuzz_test.go ~155, cli TestFindOpeningBrace ~150)
Lines to add: ~80-100 (AST-based writeAssetToConfig and UpdateAssetInConfig, helper functions)
Net reduction: ~280-300 lines
