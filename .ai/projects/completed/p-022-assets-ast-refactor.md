# p-022: Assets AST Refactor

- Status: Complete
- Started: 2026-02-19
- Completed: 2026-02-19

## Overview

Refactor asset installation code in `internal/assets/install.go` to use CUE's Abstract Syntax Tree (AST) APIs instead of fragile string manipulation. The current implementation uses string searching, manual parsing, and hardcoded indentation which is unreliable and difficult to maintain.

This project will replace ~205 lines of manual parser code with proper AST manipulation, eliminating the acknowledged hacks and limitations in the current implementation.

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
- Refactor `formatAssetStruct` to return `*ast.StructLit` instead of string
- Update callers of `formatAssetStruct`, `writeAssetToConfig`, `UpdateAssetInConfig` for new signatures

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
- [x] writeAssetToConfig builds proper AST nodes and uses format.Node()
- [x] UpdateAssetInConfig manipulates AST directly
- [x] FindAssetKey, FindMatchingBrace, FindOpeningBrace removed
- [x] cueParseState type and constants removed
- [x] Fuzz tests for deleted functions removed
- [x] All tests pass with AST-based implementation
- [x] Manual test: install, update, and check asset existence work correctly
- [x] No hardcoded indentation logic remains
- [x] Code review confirms no string manipulation of CUE content

## Current State

Verified: 2026-02-19

All existing tests pass (verified). CUE AST APIs confirmed available in cuelang.org/go v0.15.4. No external callers of Find* functions outside of install.go and test files.

Already Resolved:

- `AssetExists` (install.go:135-138): Uses CUE value lookup via `cfg.LookupPath()`. No changes needed.
- Code duplication in `assets_update.go`: Already resolved. Calls `assets.ExtractAssetContent()` and `assets.UpdateAssetInConfig()` directly.

Remaining Problems in `internal/assets/install.go` (751 lines total):

writeAssetToConfig (lines 398-483): Uses string manipulation to insert assets into config files. Calls `FindAssetKey` to detect duplicates, `FindOpeningBrace` and `FindMatchingBrace` to locate the category block, then manually builds the output with hardcoded `\t` indentation.

UpdateAssetInConfig (lines 692-751): Uses `FindAssetKey` + `FindOpeningBrace` + `FindMatchingBrace` to locate the asset entry, then does string splicing with hardcoded `\t` indentation to replace it.

Manual CUE Parser (lines 485-689): Three state-machine functions totalling ~205 lines (including doc comments):
- `FindAssetKey` (lines 485-555): Character-by-character scanner for finding keys
- `FindMatchingBrace` (lines 557-627): Character-by-character brace matcher
- `FindOpeningBrace` (lines 629-689): Character-by-character brace finder

Supporting types (lines 27-36): `cueParseState` type with four state constants used only by the Find* functions.

Hardcoded indentation in both `writeAssetToConfig` (lines 462-473) and `UpdateAssetInConfig` (lines 735-746): Uses `\t` prefix per line, with LIMITATION comment acknowledging this only works for flat structures.

Test files referencing Find* functions:
- `internal/assets/install_test.go` (1379 lines): 7 test functions to remove:
  - TestFindAssetKey (line 107)
  - TestFindMatchingBrace (line 168)
  - TestFindOpeningBrace (line 231)
  - TestFindAssetKey_EmptyKey (line 302)
  - TestFindAssetKey_EmptyContent (line 318)
  - TestFindMatchingBrace_MultiLineString (line 329)
  - TestFindOpeningBrace_MultiLineString (line 403)
- `internal/assets/fuzz_test.go` (154 lines): Entire file (FuzzFindAssetKey, FuzzFindMatchingBrace, FuzzFindOpeningBrace)
- `internal/cli/assets_test.go`: TestFindOpeningBrace (lines 670-822)

Tests that survive (may need updated expectations):
- TestWriteAssetToConfig (line 565): Uses `strings.Contains` checks, likely resilient to formatter output
- TestWriteAssetToConfig_NewCategory (line 448): Uses `strings.Contains` checks
- TestWriteAssetToConfig_BracesInStringValues (line 719): Uses `strings.Contains` plus position comparison
- TestUpdateAssetInConfig in install_test.go (line 502): Uses `strings.Contains` checks
- TestUpdateAssetInConfig in cli/assets_test.go (line 407): Uses `strings.Contains`, calls `UpdateAssetInConfig` at the function level

New imports needed: `cuelang.org/go/cue/ast`, `cuelang.org/go/cue/parser` (neither is currently imported in the project codebase; `cuelang.org/go/cue/format` is already imported in install.go)

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

formatAssetStruct:
```go
// Before: builds a string "{\n\torigin: ...\n}"
// After: builds *ast.StructLit using ast.NewStruct(), ast.NewString(), etc.
```

Remove: FindAssetKey, FindMatchingBrace, FindOpeningBrace, cueParseState type and constants.

Signature Changes:

- `formatAssetStruct` returns `*ast.StructLit` instead of `string`
- `writeAssetToConfig` accepts `ast.Expr` instead of `string` for content
- `UpdateAssetInConfig` accepts `ast.Expr` instead of `string` for newContent
- Callers updated accordingly

## Dependencies

Requires:

- p-021 (Auto-Setup Default Assets) - Completed

## Decision Points (Resolved)

1. Comment preservation during AST round-trip: A - Accept full reformat. Output is always canonical CUE. Comments stay associated with their AST node (inside the correct config entry).

2. Handling the content parameter: B - Change function signatures to accept `ast.Expr`. Full AST pipeline, no string-to-AST shims.

3. formatAssetStruct return type: A - Refactor to return `*ast.StructLit` directly. Build AST nodes with `ast.NewStruct`, `ast.NewString`, etc. End-to-end AST pipeline.

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

## Progress

Update this section as each stage completes. If a session runs out of context, the next session resumes from the last completed stage.

1. Refactor `formatAssetStruct` to build `*ast.StructLit` directly
   - Status: complete

2. Refactor `writeAssetToConfig` to use AST
   - Status: complete

3. Refactor `UpdateAssetInConfig` to use AST
   - Status: complete

4. Remove dead code
   - Status: complete

5. Remove obsolete tests
   - Status: complete

6. Update surviving tests if needed
   - Status: complete
   - Fixed alignment-sensitive assertions (CUE formatter aligns field values)
   - Fixed Simplify()-induced label changes (quoted to unquoted for valid identifiers)

7. Manual testing
   - Status: complete
   - Fresh install into empty config: pass
   - Append to existing category: pass
   - Install to new category (new file): pass
   - Update (current detection): pass
   - Force update (re-write preserving siblings): pass
   - Duplicate add (already installed report): pass
   - Update all (global + local configs): pass
   - Dry-run (preview without applying): pass
   - CUE vet validation on all generated files: pass
   - Local config (--local flag): pass
   - Supporting commands (list, search, info, index): pass

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

Lines to remove from install.go: ~295 (Find* functions + doc comments ~205, cueParseState ~10, writeAssetToConfig string logic ~85, UpdateAssetInConfig string logic ~60, hardcoded indentation ~20; some overlap as functions are fully replaced)
Lines to remove from tests: ~460 (fuzz_test.go 154, install_test.go Find* tests ~300, cli/assets_test.go TestFindOpeningBrace ~153)
Lines to add: ~80-100 (AST-based writeAssetToConfig and UpdateAssetInConfig, helper functions)
Net reduction: ~550+ lines across all files
