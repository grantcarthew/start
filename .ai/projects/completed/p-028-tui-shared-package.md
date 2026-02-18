# P-028: TUI Shared Package

- Status: Complete
- Started: 2026-02-18
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

## Implementation Progress

### Completed

1. Created `internal/tui/tui.go` with:
   - All 15 exported colour variables: `ColorError`, `ColorWarning`, `ColorSuccess`, `ColorHeader`, `ColorSeparator`, `ColorDim`, `ColorCyan`, `ColorBlue`, `ColorAgents`, `ColorRoles`, `ColorContexts`, `ColorTasks`, `ColorPrompts`, `ColorInstalled`, `ColorRegistry`
   - `Annotate(format string, a ...any) string` — returns `(text)` styled
   - `Bracket(format string, a ...any) string` — returns `[text]` styled
   - `CategoryColor(category string) *color.Color` — case-insensitive via `strings.ToLower`

2. Migrated `internal/cli/output.go`:
   - Removed the 15-colour var block and `categoryColor()` function
   - Replaced `"github.com/fatih/color"` import with `"github.com/grantcarthew/start/internal/tui"`
   - All colour refs updated to `tui.ColorXxx`
   - Sprint annotation patterns in `printContextTable` and `printAgentModel` replaced with `tui.Annotate()`

3. Migrated all 18 `internal/cli/*.go` files:
   - `config_helpers.go`, `config.go`, `config_order.go`, `config_context.go`, `start.go` (prior session)
   - `config_task.go`, `config_role.go`, `config_agent.go`, `config_settings.go`
   - `show.go`, `search.go`, `assets_search.go`, `task.go`
   - `resolve.go`, `assets_info.go`, `assets_add.go`, `assets_list.go`, `assets_index.go`, `assets_update.go`
   - `start_test.go` updated for `tui.ColorDim` reference
   - All `colorXxx` refs replaced with `tui.ColorXxx`
   - All three-call annotation/bracket patterns replaced with `tui.Annotate()`/`tui.Bracket()`
   - All `categoryColor()` calls replaced with `tui.CategoryColor()`
   - Note: `start.go` and `doctor/reporter.go` retain `"github.com/fatih/color"` import for `*color.Color` parameter/return types

4. Migrated `internal/doctor/reporter.go`:
   - Removed 11-colour var block and `sectionColor()` function
   - All `colorXxx` refs replaced with `tui.ColorXxx`
   - `fprintDim` updated to use `tui.ColorDim` and `tui.ColorCyan`
   - `statusColor` returns `*color.Color` so `"github.com/fatih/color"` import retained

5. Testing:
   - All tests pass (`scripts/invoke-tests`)
   - Zero `go vet` warnings (non-constant format strings fixed with `"%s"` wrapper)

## Decision Points

1. Exported colour variable naming convention — decided: A (PascalCase: `ColorCyan`, `ColorDim`, `ColorHeader`)

2. Doctor's `sectionColor()` uses capitalised category names ("Agents") while CLI's `categoryColor()` uses lowercase ("agents") — decided: B (case-insensitive via `strings.ToLower` inside `tui.CategoryColor()`)

## Success Criteria

- [x] `internal/tui` package exists with colour definitions and helpers
- [x] Zero duplicate colour definitions between cli and doctor
- [x] All three-call annotation/bracket patterns replaced with helper calls
- [x] All tests pass (`scripts/invoke-tests`)
- [x] No visual output changes (verified via dry-run and doctor output)

## Deliverables

- `internal/tui/tui.go` — shared colour definitions and formatting helpers
- Updated `internal/cli/output.go` — imports from tui, no local colours
- Updated `internal/cli/*.go` — annotation/bracket helpers used throughout
- Updated `internal/doctor/reporter.go` — imports from tui, no local colours
