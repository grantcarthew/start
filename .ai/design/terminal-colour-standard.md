# Terminal Colour Standard

All terminal colour usage in `start` follows this standard. Colours are defined centrally in `internal/cli/output.go` and accessed via named variables or the `categoryColor()` helper function.

## Message Types

| Element | Colour | Variable | Usage |
|---------|--------|----------|-------|
| Errors | Red | `colorError` | Error prefix and messages |
| Warnings | Yellow | `colorWarning` | Warning prefix and messages |
| Success markers | Green | `colorSuccess` | Checkmarks (✓), confirmations |
| Headers/titles | Green | `colorHeader` | Section headers |
| Separators | Magenta | `colorSeparator` | Horizontal rules (───) |
| Dim/secondary text | Faint | `colorDim` | Descriptions, metadata, de-emphasised text |

## Asset Categories

| Category | Colour | Variable |
|----------|--------|----------|
| agents | Blue | `colorAgents` |
| roles | Green | `colorRoles` |
| contexts | Cyan | `colorContexts` |
| tasks | HiYellow | `colorTasks` |
| prompts | Magenta | `colorPrompts` |

Access via `categoryColor(category)` which returns the appropriate `*color.Color`. Apply to category headers in search results, list output, and any command that groups output by asset type.

## Markers

| Marker | Colour | Usage |
|--------|--------|-------|
| Installed `★` | HiGreen | Left-side prefix on installed assets |
| Default `→` | HiGreen | Left-side prefix on default agent/role |
| Version arrows `->` | Blue | Version transition in update output |
| Delimiters `()` `[]` | Cyan | Bracketing metadata (version info, status, flags) |

## General Utility

| Variable | Colour | Usage |
|----------|--------|-------|
| `colorCyan` | Cyan | General-purpose accent |
| `colorBlue` | Blue | General-purpose accent |

## Formatting Rules

- Category names are coloured; trailing `/` is default colour
- Asset names are default terminal colour
- Descriptions and metadata are dim (faint)
- Markers (`★`, `✓`) use their assigned colour
- When colours conflict in context, the more specific role wins
- Respect `--no-color` flag and `NO_COLOR` environment variable (handled by `fatih/color`)

## Implementation

All colour variables are defined in `internal/cli/output.go`.

```go
func categoryColor(category string) *color.Color
```

A reference script at `scripts/show-colours` displays all standard ANSI colours in the terminal for visual comparison during development.

## Output Example

```
Found 5 matches:

roles/                    <- green "roles", default "/"
  ★ cwd/dotai/default       Project-specific default role
    cwd/role-md              Project-specific role from role.md

contexts/                 <- cyan "contexts", default "/"
  ★ cwd/agents-md           Repository-specific AI agent guidelines
    cwd/project              Project-specific documentation
```

- Category names: coloured per asset type
- Asset names: default terminal colour
- Descriptions: dim/faint
- Installed `★`: HiGreen, left-side prefix
- Default `→`: HiGreen, left-side prefix
