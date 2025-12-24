package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// Color definitions per DR-036
var (
	colorError   = color.New(color.FgRed)
	colorWarning = color.New(color.FgYellow)
	colorSuccess = color.New(color.FgGreen)
	colorHeader  = color.New(color.FgGreen)
	colorSeparator = color.New(color.FgMagenta)
)

// PrintError prints an error message in red.
func PrintError(w io.Writer, format string, args ...interface{}) {
	colorError.Fprintf(w, "Error: ")
	fmt.Fprintf(w, format, args...)
	fmt.Fprintln(w)
}

// PrintWarning prints a warning message in yellow.
func PrintWarning(w io.Writer, format string, args ...interface{}) {
	colorWarning.Fprintf(w, "Warning: ")
	fmt.Fprintf(w, format, args...)
	fmt.Fprintln(w)
}

// PrintSuccess prints a success marker (checkmark) in green.
func PrintSuccess(w io.Writer, text string) {
	colorSuccess.Fprintf(w, "✓")
	fmt.Fprintf(w, " %s\n", text)
}

// PrintHeader prints a header/title in green with a leading blank line.
func PrintHeader(w io.Writer, text string) {
	fmt.Fprintln(w)
	colorHeader.Fprintln(w, text)
}

// PrintSeparator prints a separator line in magenta.
func PrintSeparator(w io.Writer) {
	colorSeparator.Fprintln(w, strings.Repeat("─", 79))
}
