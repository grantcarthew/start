package doctor

import (
	"fmt"
	"io"
	"strings"
)

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
	_, _ = fmt.Fprintln(r.w, "start doctor")
	_, _ = fmt.Fprintln(r.w, strings.Repeat("═", 59))
	_, _ = fmt.Fprintln(r.w)
}

// printSection prints a single section.
func (r *Reporter) printSection(section SectionResult) {
	// Section header
	if section.Summary != "" {
		_, _ = fmt.Fprintf(r.w, "%s (%s)\n", section.Name, section.Summary)
	} else {
		_, _ = fmt.Fprintln(r.w, section.Name)
	}

	// Section results
	for _, result := range section.Results {
		r.printResult(result, section.NoIcons)
	}

	_, _ = fmt.Fprintln(r.w)
}

// printResult prints a single check result.
func (r *Reporter) printResult(result CheckResult, noIcons bool) {
	// Format based on content and icon mode
	if noIcons {
		// No icons - used for info-only sections like Version, Repository
		if result.Message == "" {
			_, _ = fmt.Fprintf(r.w, "  %s\n", result.Label)
		} else {
			_, _ = fmt.Fprintf(r.w, "  %-10s %s\n", result.Label+":", result.Message)
		}
		return
	}

	symbol := result.Status.Symbol()

	// Format based on content
	if result.Message == "" {
		_, _ = fmt.Fprintf(r.w, "  %s %s\n", symbol, result.Label)
	} else {
		_, _ = fmt.Fprintf(r.w, "  %s %s - %s\n", symbol, result.Label, result.Message)
	}

	// Print fix suggestion if present and there's an issue
	if result.Fix != "" && (result.Status == StatusFail || result.Status == StatusWarn) {
		_, _ = fmt.Fprintf(r.w, "    Fix: %s\n", result.Fix)
	}

	// Print details in verbose mode
	if r.verbose && len(result.Details) > 0 {
		for _, detail := range result.Details {
			_, _ = fmt.Fprintf(r.w, "    %s\n", detail)
		}
	}
}

// printSummary prints the summary section.
func (r *Reporter) printSummary(report Report) {
	_, _ = fmt.Fprintln(r.w, "Summary")
	_, _ = fmt.Fprintln(r.w, strings.Repeat("─", 59))

	errors := report.ErrorCount()
	warnings := report.WarnCount()

	if errors == 0 && warnings == 0 {
		_, _ = fmt.Fprintln(r.w, "  No issues found")
		_, _ = fmt.Fprintln(r.w)
		return
	}

	// Count summary
	var parts []string
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d error", errors))
		if errors > 1 {
			parts[len(parts)-1] += "s"
		}
	}
	if warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d warning", warnings))
		if warnings > 1 {
			parts[len(parts)-1] += "s"
		}
	}
	_, _ = fmt.Fprintf(r.w, "  %s found\n", strings.Join(parts, ", "))
	_, _ = fmt.Fprintln(r.w)

	// List issues
	issues := report.Issues()
	if len(issues) > 0 {
		_, _ = fmt.Fprintln(r.w, "Issues:")
		for _, issue := range issues {
			symbol := issue.Status.Symbol()
			if issue.Message != "" {
				_, _ = fmt.Fprintf(r.w, "  %s %s: %s\n", symbol, issue.Label, issue.Message)
			} else {
				_, _ = fmt.Fprintf(r.w, "  %s %s\n", symbol, issue.Label)
			}
		}
		_, _ = fmt.Fprintln(r.w)
	}

	// List recommendations (fixes)
	var recommendations []string
	for _, issue := range issues {
		if issue.Fix != "" {
			recommendations = append(recommendations, issue.Fix)
		}
	}
	if len(recommendations) > 0 {
		_, _ = fmt.Fprintln(r.w, "Recommendations:")
		for i, rec := range recommendations {
			_, _ = fmt.Fprintf(r.w, "  %d. %s\n", i+1, rec)
		}
		_, _ = fmt.Fprintln(r.w)
	}
}

// printQuiet prints minimal output for quiet mode.
func (r *Reporter) printQuiet(report Report) {
	issues := report.Issues()
	for _, issue := range issues {
		prefix := "Warning"
		if issue.Status == StatusFail {
			prefix = "Error"
		}

		if issue.Message != "" {
			_, _ = fmt.Fprintf(r.w, "%s: %s: %s\n", prefix, issue.Label, issue.Message)
		} else {
			_, _ = fmt.Fprintf(r.w, "%s: %s\n", prefix, issue.Label)
		}
	}
}
