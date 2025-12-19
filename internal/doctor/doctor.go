// Package doctor provides health check diagnostics for start configuration.
package doctor

import (
	"io"
	"runtime"
)

// Status represents the result status of a check.
type Status int

const (
	StatusPass Status = iota
	StatusWarn
	StatusFail
	StatusInfo
)

// String returns the string representation of a Status.
func (s Status) String() string {
	switch s {
	case StatusPass:
		return "pass"
	case StatusWarn:
		return "warn"
	case StatusFail:
		return "fail"
	case StatusInfo:
		return "info"
	default:
		return "unknown"
	}
}

// Symbol returns the display symbol for a Status.
func (s Status) Symbol() string {
	switch s {
	case StatusPass:
		return "✓"
	case StatusWarn:
		return "⚠"
	case StatusFail:
		return "✗"
	case StatusInfo:
		return "-"
	default:
		return "?"
	}
}

// CheckResult holds the result of a single check item.
type CheckResult struct {
	Status  Status
	Label   string   // Short label (e.g., "claude", "agents.cue")
	Message string   // Detail message (e.g., "/usr/local/bin/claude", "NOT FOUND")
	Fix     string   // Suggested fix action
	Details []string // Additional details for verbose mode
}

// SectionResult holds the results for a check section.
type SectionResult struct {
	Name    string        // Section name (e.g., "Agents", "Configuration")
	Results []CheckResult // Individual check results
	Summary string        // Optional summary (e.g., "2 configured")
	NoIcons bool          // If true, don't show status icons (for info-only sections)
}

// Report holds the complete diagnostic report.
type Report struct {
	Sections []SectionResult
}

// HasIssues returns true if the report contains any failures or warnings.
func (r Report) HasIssues() bool {
	for _, s := range r.Sections {
		for _, c := range s.Results {
			if c.Status == StatusFail || c.Status == StatusWarn {
				return true
			}
		}
	}
	return false
}

// Issues returns all check results that are failures or warnings.
func (r Report) Issues() []CheckResult {
	var issues []CheckResult
	for _, s := range r.Sections {
		for _, c := range s.Results {
			if c.Status == StatusFail || c.Status == StatusWarn {
				issues = append(issues, c)
			}
		}
	}
	return issues
}

// ErrorCount returns the number of failures in the report.
func (r Report) ErrorCount() int {
	count := 0
	for _, s := range r.Sections {
		for _, c := range s.Results {
			if c.Status == StatusFail {
				count++
			}
		}
	}
	return count
}

// WarnCount returns the number of warnings in the report.
func (r Report) WarnCount() int {
	count := 0
	for _, s := range r.Sections {
		for _, c := range s.Results {
			if c.Status == StatusWarn {
				count++
			}
		}
	}
	return count
}

// BuildInfo holds version and build information.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
	Platform  string
}

// DefaultBuildInfo returns build info with runtime defaults.
func DefaultBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   "dev",
		Commit:    "unknown",
		BuildDate: "unknown",
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// Options configures the doctor run.
type Options struct {
	Verbose bool
	Quiet   bool
	Stdout  io.Writer
	Stderr  io.Writer
}

// RepoURL is the repository URL for the project.
const RepoURL = "https://github.com/grantcarthew/start"

// IssuesURL is the issues URL for the project.
const IssuesURL = "https://github.com/grantcarthew/start/issues"
