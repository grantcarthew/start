# DR-042: Terminal Colour Standard

- Date: 2026-02-10
- Status: Accepted
- Category: CLI
- Supersedes: [dr-036](superseded/dr-036-cli-terminal-colors.md)

## Problem

DR-036 established basic message-type colours (errors, warnings, success) but did not define colours for asset categories or other semantic elements. As the CLI grows, inconsistent colour usage makes output harder to scan and reduces the visual language available to users.

A comprehensive colour standard is needed so all commands use a consistent palette and new output can be added without ad-hoc colour choices.

## Decision

All terminal colour usage follows the standard below. Colours are defined centrally in `internal/cli/output.go` and accessed via named variables or helper functions.

### Message Types

| Element | Colour | Variable | Usage |
|---------|--------|----------|-------|
| Errors | Red | `colorError` | Error prefix and messages |
| Warnings | Yellow | `colorWarning` | Warning prefix and messages |
| Success markers | Green | `colorSuccess` | Checkmarks (✓), confirmations |
| Headers/titles | Green | `colorHeader` | Section headers |
| Separators | Magenta | `colorSeparator` | Horizontal rules (───) |
| Dim/secondary text | Faint | `colorDim` | Descriptions, metadata, de-emphasised text |

### Asset Categories

| Category | Colour | Variable |
|----------|--------|----------|
| agents | Blue | `colorAgents` |
| roles | Green | `colorRoles` |
| contexts | Cyan | `colorContexts` |
| tasks | HiYellow | `colorTasks` |

Access via `categoryColor(category)` helper which returns the appropriate `*color.Color`.

Apply to: category headers in search results, list output, and any command that groups output by asset type. The category name is coloured; trailing punctuation (e.g., `/`) is default colour.

### Markers

| Marker | Colour | Usage |
|--------|--------|-------|
| Installed `*` | HiGreen | Suffix on search results for installed assets |
| Version arrows `->` | Blue | Version transition in update output |
| Parenthetical delimiters | Cyan | Bracketing metadata (version info, status) |

### General Utility

| Variable | Colour | Usage |
|----------|--------|-------|
| `colorCyan` | Cyan | General-purpose accent |
| `colorBlue` | Blue | General-purpose accent |

### Formatting Rules

- Category names are coloured, trailing `/` is default
- Asset names are default colour
- Descriptions and metadata are dim (faint)
- Markers (installed `*`, success `✓`) use their assigned colour
- When colours would conflict in context, the more specific role wins

## Why

- Consistent colour palette reduces cognitive load when scanning output
- Asset category colours provide instant visual grouping across all commands
- Dim descriptions create clear hierarchy: category > name > description
- Centralised definitions in `output.go` prevent drift and simplify changes
- `categoryColor()` helper ensures new commands get correct colours without hardcoding
- HiYellow for tasks avoids collision with warning yellow while keeping the action-oriented feel
- HiGreen for installed marker distinguishes from regular green success markers

## Trade-offs

Accept:

- Asset category colours overlap with message-type colours (green for both roles and success)
- Context determines which meaning applies, so this is acceptable
- More colour variables to maintain

Gain:

- Four-colour asset type distinction at a glance
- Clear visual hierarchy in all asset-related output
- Reusable `categoryColor()` function for future commands
- Consistent experience across search, list, info, and update commands

## Alternatives

No asset category colours:

- Pro: Simpler, fewer colour definitions
- Con: All categories look the same, harder to scan grouped output
- Rejected: Visual distinction is worth the small complexity cost

Bright (Hi) variants for all categories:

- Pro: Maximum contrast on dark terminals
- Con: Clashes more with message-type colours
- Con: Can be garish with lots of output
- Rejected: Standard variants are sufficient for categories; Hi variants reserved for emphasis (HiYellow for tasks, HiGreen for markers)

## Usage Examples

Search output:

```
Found 5 matches:

roles/                    <- green "roles", default "/"
  cwd/dotai/default         Project-specific default role *
  cwd/role-md               Project-specific role from role.md

contexts/                 <- cyan "contexts", default "/"
  cwd/agents-md             Repository-specific AI agent guidelines *
  cwd/project               Project-specific documentation
```

- Category names: coloured per asset type
- Asset names: default terminal colour
- Descriptions: dim/faint
- Installed `*`: HiGreen

## Implementation Notes

All colour variables are defined in `internal/cli/output.go`.

The `categoryColor()` function maps category strings to their colour:

```
func categoryColor(category string) *color.Color
```

A reference script at `scripts/show-colours` displays all standard ANSI colours in the terminal for visual comparison during development.

Respect `--no-color` flag and `NO_COLOR` environment variable (handled by `fatih/color`).
