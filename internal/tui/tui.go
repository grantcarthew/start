package tui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Colour definitions per DR-042
var (
	ColorError     = color.New(color.FgRed)
	ColorWarning   = color.New(color.FgYellow)
	ColorSuccess   = color.New(color.FgGreen)
	ColorHeader    = color.New(color.FgGreen)
	ColorSeparator = color.New(color.FgMagenta)
	ColorDim       = color.New(color.Faint)
	ColorCyan      = color.New(color.FgCyan)
	ColorBlue      = color.New(color.FgBlue)

	// Asset category colours
	ColorAgents    = color.New(color.FgBlue)
	ColorRoles     = color.New(color.FgGreen)
	ColorContexts  = color.New(color.FgCyan)
	ColorTasks     = color.New(color.FgHiYellow)
	ColorPrompts   = color.New(color.FgMagenta)
	ColorInstalled = color.New(color.FgHiGreen)
	ColorRegistry  = color.New(color.FgYellow)
)

// CategoryColor returns the colour for an asset category.
// Matching is case-insensitive.
func CategoryColor(category string) *color.Color {
	switch strings.ToLower(category) {
	case "agents":
		return ColorAgents
	case "roles":
		return ColorRoles
	case "contexts":
		return ColorContexts
	case "tasks":
		return ColorTasks
	default:
		return ColorDim
	}
}

// Annotate returns text wrapped in cyan parentheses with dim content: (text)
func Annotate(format string, a ...any) string {
	text := fmt.Sprintf(format, a...)
	return ColorCyan.Sprint("(") + ColorDim.Sprint(text) + ColorCyan.Sprint(")")
}

// Bracket returns text wrapped in cyan square brackets with dim content: [text]
func Bracket(format string, a ...any) string {
	text := fmt.Sprintf(format, a...)
	return ColorCyan.Sprint("[") + ColorDim.Sprint(text) + ColorCyan.Sprint("]")
}
