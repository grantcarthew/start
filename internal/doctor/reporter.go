package doctor

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// Colour definitions per DR-042
var (
	colorHeader    = color.New(color.FgGreen)
	colorSeparator = color.New(color.FgMagenta)
	colorSuccess   = color.New(color.FgGreen)
	colorError     = color.New(color.FgRed)
	colorWarning   = color.New(color.FgYellow)
	colorDim       = color.New(color.Faint)
	colorCyan      = color.New(color.FgCyan)

	// Asset category colours per DR-042
	colorAgents   = color.New(color.FgBlue)
	colorRoles    = color.New(color.FgGreen)
	colorContexts = color.New(color.FgCyan)
	colorTasks    = color.New(color.FgHiYellow)
)

// fprintDim writes text with dim styling and cyan parenthetical delimiters per DR-042.
// Text is dim, ( and ) are cyan. Byte iteration is safe here as delimiters are ASCII.
func fprintDim(w io.Writer, s string) {
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '(' || s[i] == ')' {
			if i > start {
				_, _ = colorDim.Fprint(w, s[start:i])
			}
			_, _ = colorCyan.Fprintf(w, "%c", s[i])
			start = i + 1
		}
	}
	if start < len(s) {
		_, _ = colorDim.Fprint(w, s[start:])
	}
}

// sectionColor returns the colour for a section name.
func sectionColor(name string) *color.Color {
	switch name {
	case "Agents":
		return colorAgents
	case "Roles":
		return colorRoles
	case "Contexts":
		return colorContexts
	case "Tasks":
		return colorTasks
	default:
		return nil
	}
}

// Reporter handles output formatting for doctor results.
type Reporter struct {
	w       io.Writer
	verbose bool
	quiet   bool
}

// NewReporter creates a new reporter.
func NewReporter(w io.Writer, verbose, quiet bool) *Reporter {
	return &Reporter{
		w:       w,
		verbose: verbose,
		quiet:   quiet,
	}
}

// Print outputs the complete report.
func (r *Reporter) Print(report Report) {
	if r.quiet {
		r.printQuiet(report)
		return
	}

	r.printHeader()

	for _, section := range report.Sections {
		r.printSection(section)
	}

	r.printSummary(report)
}

// printHeader prints the doctor header.
func (r *Reporter) printHeader() {
	_, _ = fmt.Fprintln(r.w)
	_, _ = colorHeader.Fprintln(r.w, "start doctor")
	_, _ = colorSeparator.Fprintln(r.w, strings.Repeat("═", 59))
	_, _ = fmt.Fprintln(r.w)
}

// printSection prints a single section.
func (r *Reporter) printSection(section SectionResult) {
	// Section header
	sc := sectionColor(section.Name)
	if sc != nil {
		_, _ = sc.Fprintf(r.w, "%s", section.Name)
	} else {
		_, _ = fmt.Fprint(r.w, section.Name)
	}
	if section.Summary != "" {
		_, _ = fmt.Fprint(r.w, " ")
		fprintDim(r.w, "("+section.Summary+")")
	}
	_, _ = fmt.Fprintln(r.w)

	// Section results
	for _, result := range section.Results {
		r.printResult(result, section.NoIcons)
	}

	_, _ = fmt.Fprintln(r.w)
}

// statusColor returns the colour for a status.
func statusColor(s Status) *color.Color {
	switch s {
	case StatusPass:
		return colorSuccess
	case StatusFail:
		return colorError
	case StatusWarn:
		return colorWarning
	default:
		return colorDim
	}
}

// printResult prints a single check result.
func (r *Reporter) printResult(result CheckResult, noIcons bool) {
	// Format based on content and icon mode
	if noIcons {
		// No icons - used for info-only sections like Version, Repository
		if result.Message == "" {
			_, _ = fmt.Fprintf(r.w, "  %s\n", result.Label)
		} else {
			_, _ = fmt.Fprintf(r.w, "  %-10s ", result.Label+":")
			fprintDim(r.w, result.Message)
			_, _ = fmt.Fprintln(r.w)
		}
		return
	}

	sc := statusColor(result.Status)
	symbol := result.Status.Symbol()

	// Format based on content
	_, _ = fmt.Fprint(r.w, "  ")
	_, _ = sc.Fprintf(r.w, "%s", symbol)
	if result.Message == "" {
		_, _ = fmt.Fprintf(r.w, " %s\n", result.Label)
	} else {
		_, _ = fmt.Fprintf(r.w, " %s - ", result.Label)
		fprintDim(r.w, result.Message)
		_, _ = fmt.Fprintln(r.w)
	}

	// Print fix suggestion if present and there's an issue
	if result.Fix != "" && (result.Status == StatusFail || result.Status == StatusWarn) {
		_, _ = fmt.Fprint(r.w, "    ")
		fprintDim(r.w, "Fix: "+result.Fix)
		_, _ = fmt.Fprintln(r.w)
	}

	// Print details in verbose mode
	if r.verbose && len(result.Details) > 0 {
		for _, detail := range result.Details {
			_, _ = colorDim.Fprintf(r.w, "    %s\n", detail)
		}
	}
}

// printSummary prints the summary section.
func (r *Reporter) printSummary(report Report) {
	_, _ = colorHeader.Fprintln(r.w, "Summary")
	_, _ = colorSeparator.Fprintln(r.w, strings.Repeat("─", 59))

	errors := report.ErrorCount()
	warnings := report.WarnCount()

	if errors == 0 && warnings == 0 {
		_, _ = colorSuccess.Fprintln(r.w, "  No issues found")
		_, _ = fmt.Fprintln(r.w)
		return
	}

	// Count summary
	_, _ = fmt.Fprint(r.w, "  ")
	if errors > 0 {
		label := "error"
		if errors > 1 {
			label = "errors"
		}
		_, _ = colorError.Fprintf(r.w, "%d %s", errors, label)
		if warnings > 0 {
			_, _ = fmt.Fprint(r.w, ", ")
		}
	}
	if warnings > 0 {
		label := "warning"
		if warnings > 1 {
			label = "warnings"
		}
		_, _ = colorWarning.Fprintf(r.w, "%d %s", warnings, label)
	}
	_, _ = fmt.Fprintln(r.w, " found")
	_, _ = fmt.Fprintln(r.w)

	// List issues
	issues := report.Issues()
	if len(issues) > 0 {
		_, _ = fmt.Fprintln(r.w, "Issues:")
		for _, issue := range issues {
			sc := statusColor(issue.Status)
			symbol := issue.Status.Symbol()
			_, _ = fmt.Fprint(r.w, "  ")
			_, _ = sc.Fprintf(r.w, "%s", symbol)
			if issue.Message != "" {
				_, _ = fmt.Fprintf(r.w, " %s: ", issue.Label)
				fprintDim(r.w, issue.Message)
				_, _ = fmt.Fprintln(r.w)
			} else {
				_, _ = fmt.Fprintf(r.w, " %s\n", issue.Label)
			}
		}
	}
}

// printQuiet prints minimal output for quiet mode.
func (r *Reporter) printQuiet(report Report) {
	issues := report.Issues()
	for _, issue := range issues {
		sc := statusColor(issue.Status)
		prefix := "Warning"
		if issue.Status == StatusFail {
			prefix = "Error"
		}

		_, _ = sc.Fprintf(r.w, "%s: ", prefix)
		if issue.Message != "" {
			_, _ = fmt.Fprintf(r.w, "%s: %s\n", issue.Label, issue.Message)
		} else {
			_, _ = fmt.Fprintf(r.w, "%s\n", issue.Label)
		}
	}
}
