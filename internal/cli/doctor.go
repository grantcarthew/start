package cli

import (
	"context"
	"time"

	"github.com/grantcarthew/start/internal/assets"
	"github.com/grantcarthew/start/internal/config"
	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/doctor"
	"github.com/grantcarthew/start/internal/registry"
	"github.com/spf13/cobra"
)

// Build-time variables set via ldflags
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

// addDoctorCommand adds the doctor command to the parent command.
func addDoctorCommand(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "doctor",
		GroupID: "utilities",
		Short:   "Diagnose start installation and configuration",
		Long: `Performs comprehensive health check of start installation, configuration,
and environment. Reports issues, warnings, and suggestions.

Checks performed:
  - Version and build information
  - Configuration file validation (CUE syntax)
  - Agent binary availability
  - Context and role file existence
  - Environment (directory permissions)

Exit codes:
  0 - All checks passed
  1 - Issues found`,
		RunE: runDoctor,
	}

	parent.AddCommand(cmd)
}

// runDoctor executes the doctor command.
func runDoctor(cmd *cobra.Command, args []string) error {
	report, err := prepareDoctor()
	if err != nil {
		return err
	}

	flags := getFlags(cmd)
	reporter := doctor.NewReporter(cmd.OutOrStdout(), flags.Verbose, flags.Quiet)
	reporter.Print(report)

	if report.HasIssues() {
		// Return a silent error to set exit code 1
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		return errDoctorIssuesFound
	}

	return nil
}

// errDoctorIssuesFound is returned when doctor finds issues.
// It implements SilentError so main.go skips printing it.
var errDoctorIssuesFound = &doctorError{}

type doctorError struct{}

func (e *doctorError) Error() string { return "issues found" }
func (e *doctorError) Silent() bool  { return true }

// prepareDoctor runs all checks and builds the report.
func prepareDoctor() (doctor.Report, error) {
	var report doctor.Report

	// Intro section
	report.Sections = append(report.Sections, doctor.CheckIntro())

	// Version section
	buildInfo := doctor.BuildInfo{
		Version:      version,
		Commit:       commit,
		BuildDate:    buildDate,
		GoVersion:    doctor.DefaultBuildInfo().GoVersion,
		Platform:     doctor.DefaultBuildInfo().Platform,
		IndexVersion: resolveIndexVersion(),
	}
	report.Sections = append(report.Sections, doctor.CheckVersion(buildInfo))

	// Configuration section
	paths, err := config.ResolvePaths("")
	if err != nil {
		return report, err
	}
	report.Sections = append(report.Sections, doctor.CheckConfiguration(paths))

	// Load config for remaining checks (if possible)
	var cfgLoaded bool
	var cfgResult internalcue.LoadResult

	if paths.AnyExists() {
		loader := internalcue.NewLoader()
		dirs := paths.ForScope(config.ScopeMerged)
		if len(dirs) > 0 {
			cfgResult, err = loader.Load(dirs)
			if err == nil {
				cfgLoaded = true
			}
		}
	}

	// Agent checks
	if cfgLoaded {
		report.Sections = append(report.Sections, doctor.CheckAgents(cfgResult.Value))
	} else {
		report.Sections = append(report.Sections, doctor.SectionResult{
			Name: "Agents",
			Results: []doctor.CheckResult{
				{Status: doctor.StatusInfo, Label: "Skipped", Message: "no valid config"},
			},
		})
	}

	// Context checks
	if cfgLoaded {
		report.Sections = append(report.Sections, doctor.CheckContexts(cfgResult.Value))
	} else {
		report.Sections = append(report.Sections, doctor.SectionResult{
			Name: "Contexts",
			Results: []doctor.CheckResult{
				{Status: doctor.StatusInfo, Label: "Skipped", Message: "no valid config"},
			},
		})
	}

	// Role checks
	if cfgLoaded {
		report.Sections = append(report.Sections, doctor.CheckRoles(cfgResult.Value))
	} else {
		report.Sections = append(report.Sections, doctor.SectionResult{
			Name: "Roles",
			Results: []doctor.CheckResult{
				{Status: doctor.StatusInfo, Label: "Skipped", Message: "no valid config"},
			},
		})
	}

	// Task checks
	if cfgLoaded {
		report.Sections = append(report.Sections, doctor.CheckTasks(cfgResult.Value))
	} else {
		report.Sections = append(report.Sections, doctor.SectionResult{
			Name: "Tasks",
			Results: []doctor.CheckResult{
				{Status: doctor.StatusInfo, Label: "Skipped", Message: "no valid config"},
			},
		})
	}

	// Environment checks
	report.Sections = append(report.Sections, doctor.CheckEnvironment(paths))

	return report, nil
}

// resolveIndexVersion queries the registry for the latest index version.
// Returns the version string (e.g., "v0.3.2") or empty string on failure.
func resolveIndexVersion() string {
	client, err := registry.NewClient()
	if err != nil {
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolved, err := client.ResolveLatestVersion(ctx, registry.IndexModulePath)
	if err != nil {
		return ""
	}

	return assets.VersionFromOrigin(resolved)
}
