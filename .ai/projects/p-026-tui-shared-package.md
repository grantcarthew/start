# P-026: TUI Shared Package

- Status: Pending
- Started:
- Completed:

## Overview

Extract shared terminal UI definitions and helpers into `internal/tui`. Currently, colour definitions are duplicated between `internal/cli/output.go` and `internal/doctor/reporter.go`, and the three-call annotation pattern (`colorCyan("(") + colorDim(text) + colorCyan(")")`) is repeated 80+ times across the CLI package. This project consolidates both into a single shared package.

Prompted by an external readability review (`.start/reviews/2026-02-15-readability.md`, findings M1, and the duplicate colour definitions in doctor).

## Goals

1. Create `internal/tui` package with shared colour definitions (per DR-042)
2. Add `Annotate` and `Bracket` formatting helpers to replace the three-call pattern
3. Migrate `internal/cli` to use `tui` colours and helpers
4. Migrate `internal/doctor` to use `tui` colours
5. Remove duplicate colour definitions from both packages

## Scope

In Scope:
- Shared colour variable definitions (DR-042 standard)
- `Annotate(text)` helper returning `(text)` with cyan delimiters and dim content
- `Bracket(text)` helper returning `[text]` with cyan delimiters and dim content
- `CategoryColor(category)` function (duplicated between cli and doctor)
- Migration of all call sites in `internal/cli` and `internal/doctor`

Out of Scope:
- Changing any visual output (colours, formatting must remain identical)
- Moving non-colour output helpers (printHeader, printSeparator, printContextTable, etc.) — these depend on domain types and belong in their current packages
- The `fprintDim` function in doctor/reporter.go — it has unique behaviour (parsing parens from a plain string) that is specific to doctor output formatting

## Current State

`internal/cli/output.go` defines 12 colour variables and `categoryColor()`:
- `colorError`, `colorWarning`, `colorSuccess`, `colorHeader`, `colorSeparator`, `colorDim`, `colorCyan`, `colorBlue`
- `colorAgents`, `colorRoles`, `colorContexts`, `colorTasks`, `colorPrompts`, `colorInstalled`

`internal/doctor/reporter.go` defines 11 of the same colours:
- `colorHeader`, `colorSeparator`, `colorSuccess`, `colorError`, `colorWarning`, `colorDim`, `colorCyan`
- `colorAgents`, `colorRoles`, `colorContexts`, `colorTasks`

Both reference DR-042 in comments.

The annotation pattern appears ~80+ times across 11 files in `internal/cli/`:
- Paren variant: `colorCyan.Sprint("(") + colorDim.Sprint(text) + colorCyan.Sprint(")")`
- Bracket variant: `colorCyan.Sprint("[") + colorDim.Sprint(text) + colorCyan.Sprint("]")`

## Technical Approach

1. Create `internal/tui/tui.go` with:
   - Exported colour variables matching DR-042 (e.g., `ColorCyan`, `ColorDim`, `ColorError`)
   - `Annotate(format string, a ...any) string` — returns `(text)` styled
   - `Bracket(format string, a ...any) string` — returns `[text]` styled
   - `CategoryColor(category string) *color.Color`

2. Update `internal/cli/output.go`:
   - Remove colour variable block, import from `tui`
   - Remove `categoryColor()`, use `tui.CategoryColor()`

3. Update all `internal/cli/*.go` files:
   - Replace three-call annotation patterns with `tui.Annotate()` / `tui.Bracket()`
   - Replace direct colour references (e.g., `colorCyan` to `tui.ColorCyan`)

4. Update `internal/doctor/reporter.go`:
   - Remove colour variable block, import from `tui`
   - Remove `sectionColor()`, use `tui.CategoryColor()` (with name mapping if needed)

## Decision Points

1. Exported colour variable naming convention

- A: PascalCase matching current names (`ColorCyan`, `ColorDim`, `ColorHeader`)
- B: Grouped under a struct (`Colors.Cyan`, `Colors.Dim`)

2. Doctor's `sectionColor()` uses capitalised category names ("Agents") while CLI's `categoryColor()` uses lowercase ("agents")

- A: Normalise to lowercase in `tui.CategoryColor()`, have doctor lowercase before calling
- B: Make `tui.CategoryColor()` case-insensitive

## Success Criteria

- [ ] `internal/tui` package exists with colour definitions and helpers
- [ ] Zero duplicate colour definitions between cli and doctor
- [ ] All three-call annotation/bracket patterns replaced with helper calls
- [ ] All tests pass (`scripts/invoke-tests`)
- [ ] No visual output changes (verified via dry-run and doctor output)

## Deliverables

- `internal/tui/tui.go` — shared colour definitions and formatting helpers
- Updated `internal/cli/output.go` — imports from tui, no local colours
- Updated `internal/cli/*.go` — annotation/bracket helpers used throughout
- Updated `internal/doctor/reporter.go` — imports from tui, no local colours
