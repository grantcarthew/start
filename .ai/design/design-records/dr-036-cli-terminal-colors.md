# dr-036: CLI Terminal Colors

- Date: 2025-12-24
- Status: Accepted
- Category: CLI

## Problem

Terminal output lacks visual distinction. Errors, warnings, success messages, and headers all appear in the same default color, making it harder to quickly identify issues and scan output.

## Decision

Add colored terminal output using the `github.com/fatih/color` library with central output helper functions.

Color scheme:

| Message Type | Color |
|--------------|-------|
| Errors | Red |
| Warnings | Yellow |
| Success markers (✓) | Green |
| Headers/titles | Green |
| Separators (───) | Magenta |

Control:

- Add global `--no-color` flag to disable colors
- Also respect `NO_COLOR` environment variable (handled automatically by fatih/color)

Implementation:

- Create central output helper functions in `internal/cli/output.go`
- Functions: `PrintError`, `PrintWarning`, `PrintSuccess`, `PrintHeader`, `PrintSeparator`
- All output functions respect the `--no-color` flag

## Why

- Visual distinction helps users quickly identify errors and warnings
- Consistent color scheme across all commands improves UX
- Central helpers ensure consistency and simplify future changes
- `fatih/color` is battle-tested, handles TTY detection, and respects `NO_COLOR` standard
- Global flag provides explicit control for users who prefer plain output

## Trade-offs

Accept:

- External dependency (`github.com/fatih/color`)
- Slightly more complex output code paths

Gain:

- Clear visual hierarchy in terminal output
- Faster issue identification for users
- Consistent appearance across all commands
- Automatic TTY detection (no colors when piped)

## Alternatives

Raw ANSI codes:

- Pro: No external dependency
- Con: Manual TTY detection required
- Con: No automatic `NO_COLOR` support
- Con: Verbose, error-prone
- Rejected: Too much boilerplate for robust implementation

`charmbracelet/lipgloss`:

- Pro: More powerful styling options
- Pro: Modern, used by Charm ecosystem
- Con: Heavier dependency
- Con: More complex API for simple color needs
- Rejected: Overkill for basic coloring

## Usage Examples

Command output before:

```
Warning: context "project": reading file README.md: no such file or directory
Starting AI Agent
───────────────────────────────────────────────────────────────────────────────
Agent: claude (model: sonnet)

Context documents:
  ✓ codebase

Error: command template does not start with a valid executable
```

Command output after (with colors):

- "Warning:" in yellow
- "Starting AI Agent" in green
- Separator line in magenta
- "✓" in green
- "Error:" in red

Disabling colors:

```bash
start --no-color
NO_COLOR=1 start
```

## Implementation Notes

The `--no-color` flag should be added to the persistent flags in root.go and stored in the Flags struct. The flag value should be passed to `color.NoColor` early in command execution.

Helper functions should accept an io.Writer for testability:

```
PrintError(w io.Writer, format string, args ...interface{})
PrintWarning(w io.Writer, format string, args ...interface{})
PrintSuccess(w io.Writer, text string)
PrintHeader(w io.Writer, text string)
PrintSeparator(w io.Writer)
```
