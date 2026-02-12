package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/grantcarthew/start/internal/orchestration"
)

// Color definitions per DR-036
var (
	colorError     = color.New(color.FgRed)
	colorWarning   = color.New(color.FgYellow)
	colorSuccess   = color.New(color.FgGreen)
	colorHeader    = color.New(color.FgGreen)
	colorSeparator = color.New(color.FgMagenta)
	colorDim       = color.New(color.Faint)
	colorCyan      = color.New(color.FgCyan)
	colorBlue      = color.New(color.FgBlue)

	// Asset category colours
	colorAgents    = color.New(color.FgBlue)
	colorRoles     = color.New(color.FgGreen)
	colorContexts  = color.New(color.FgCyan)
	colorTasks     = color.New(color.FgHiYellow)
	colorInstalled = color.New(color.FgHiGreen)
)

// categoryColor returns the colour for an asset category.
func categoryColor(category string) *color.Color {
	switch category {
	case "agents":
		return colorAgents
	case "roles":
		return colorRoles
	case "contexts":
		return colorContexts
	case "tasks":
		return colorTasks
	default:
		return colorDim
	}
}

// PrintError prints an error message in red.
func PrintError(w io.Writer, format string, args ...interface{}) {
	_, _ = colorError.Fprintf(w, "Error: ")
	_, _ = fmt.Fprintf(w, format, args...)
	_, _ = fmt.Fprintln(w)
}

// PrintWarning prints a warning message in yellow.
func PrintWarning(w io.Writer, format string, args ...interface{}) {
	_, _ = colorWarning.Fprintf(w, "Warning: ")
	_, _ = fmt.Fprintf(w, format, args...)
	_, _ = fmt.Fprintln(w)
}

// PrintSuccess prints a success marker (checkmark) in green.
func PrintSuccess(w io.Writer, text string) {
	_, _ = colorSuccess.Fprintf(w, "✓")
	_, _ = fmt.Fprintf(w, " %s\n", text)
}

// PrintHeader prints a header/title in green with a leading blank line.
func PrintHeader(w io.Writer, text string) {
	_, _ = fmt.Fprintln(w)
	_, _ = colorHeader.Fprintln(w, text)
}

// PrintSeparator prints a separator line in magenta.
func PrintSeparator(w io.Writer) {
	_, _ = colorSeparator.Fprintln(w, strings.Repeat("─", 79))
}

// PrintContextTable prints contexts in a table format.
// Shows all contexts (loaded and failed) with status indicator.
func PrintContextTable(w io.Writer, contexts []orchestration.Context) {
	if len(contexts) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "Context documents:")

	// Calculate column widths
	nameWidth := 4 // "Name" header
	tagsWidth := 4 // "Tags" header
	fileWidth := 4 // "File" header

	type row struct {
		name   string
		status string
		tags   string
		file   string
	}

	rows := make([]row, len(contexts))
	for i, ctx := range contexts {
		// Status: ✓ for loaded, ○ for failed
		status := "✓"
		if ctx.Error != "" {
			status = "○"
		}

		// Tags: combine required, default, and tags
		var tags []string
		if ctx.Required {
			tags = append(tags, "required")
		}
		if ctx.Default {
			tags = append(tags, "default")
		}
		tags = append(tags, ctx.Tags...)
		tagStr := strings.Join(tags, ", ")
		if tagStr == "" {
			tagStr = "-"
		}

		// File: show basename, add error info if failed
		file := ctx.File
		if file != "" {
			file = filepath.Base(file)
		} else {
			file = "-"
		}
		if ctx.Error != "" {
			file += " (not found)"
		}

		rows[i] = row{
			name:   ctx.Name,
			status: status,
			tags:   tagStr,
			file:   file,
		}

		// Update widths
		if len(ctx.Name) > nameWidth {
			nameWidth = len(ctx.Name)
		}
		if len(tagStr) > tagsWidth {
			tagsWidth = len(tagStr)
		}
		if len(file) > fileWidth {
			fileWidth = len(file)
		}
	}

	// Print header
	_, _ = fmt.Fprintf(w, "  %-*s  %s  %-*s  %s\n",
		nameWidth, "Name", "Status", tagsWidth, "Tags", "File")

	// Print rows
	for _, r := range rows {
		_, _ = fmt.Fprint(w, "  ")
		_, _ = fmt.Fprintf(w, "%-*s  ", nameWidth, r.name)
		if r.status == "✓" {
			_, _ = colorSuccess.Fprintf(w, "%s", r.status)
		} else {
			_, _ = fmt.Fprint(w, r.status)
		}
		_, _ = fmt.Fprintf(w, "       %-*s  %s\n", tagsWidth, r.tags, r.file)
	}
	_, _ = fmt.Fprintln(w)
}

// PrintRoleTable prints the role resolution chain in a table format.
// Shows status indicator: ✓ for loaded, ○ for skipped/error.
func PrintRoleTable(w io.Writer, resolutions []orchestration.RoleResolution) {
	if len(resolutions) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "Role:")

	// Calculate column widths
	nameWidth := 4 // "Name" header
	for _, r := range resolutions {
		if len(r.Name) > nameWidth {
			nameWidth = len(r.Name)
		}
	}

	// Print header
	_, _ = fmt.Fprintf(w, "  %-*s  %s  %s\n", nameWidth, "Name", "Status", "File")

	// Print rows
	for _, r := range resolutions {
		// Determine status symbol
		status := "○"
		if r.Status == "loaded" {
			status = "✓"
		}

		// Determine file display
		file := filepath.Base(r.File)
		if file == "" || file == "." {
			file = "-"
		}

		// Add status info for non-loaded roles
		switch r.Status {
		case "skipped":
			file = "skipped"
		case "error":
			if r.Error != "" {
				file = r.Error
			} else {
				file = "not found"
			}
		}

		// Print row
		_, _ = fmt.Fprint(w, "  ")
		_, _ = fmt.Fprintf(w, "%-*s  ", nameWidth, r.Name)
		if status == "✓" {
			_, _ = colorSuccess.Fprintf(w, "%s", status)
		} else {
			_, _ = fmt.Fprint(w, status)
		}
		_, _ = fmt.Fprintf(w, "       %s\n", file)
	}
	_, _ = fmt.Fprintln(w)
}
