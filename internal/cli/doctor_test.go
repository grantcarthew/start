package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorCommand_Exists(t *testing.T) {
	cmd := NewRootCmd()

	// Find the doctor command
	doctorCmd, _, err := cmd.Find([]string{"doctor"})
	if err != nil {
		t.Fatalf("doctor command not found: %v", err)
	}

	if doctorCmd.Use != "doctor" {
		t.Errorf("Use = %q, want %q", doctorCmd.Use, "doctor")
	}

	if doctorCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}

func TestDoctorCommand_NoConfig(t *testing.T) {
	// Create isolated temp directory with no config
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"doctor"})

	// Should complete without panic, may return error for issues
	_ = cmd.Execute()

	output := stdout.String()

	// Should show intro section
	if !strings.Contains(output, "start") {
		t.Errorf("output should contain 'start', got: %s", output)
	}
}

func TestDoctorCommand_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create local .start config
	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	config := `
agents: {
	echo: {
		bin: "echo"
		command: "{{.Bin}} test"
		default_model: "default"
	}
}

roles: {
	assistant: {
		prompt: "You are a helpful assistant."
	}
}

contexts: {
	env: {
		required: true
		prompt: "Environment context"
	}
}

settings: {
	default_agent: "echo"
}
`
	configFile := filepath.Join(configDir, "settings.cue")
	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"doctor"})

	_ = cmd.Execute()

	output := stdout.String()

	// Should show various sections
	expectedSections := []string{
		"Version",
		"Configuration",
		"Agents",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("output should contain section %q, got: %s", section, output)
		}
	}
}

func TestDoctorCommand_Verbose(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"doctor", "--verbose"})

	_ = cmd.Execute()

	// Verbose mode should produce output
	if stdout.Len() == 0 {
		t.Error("verbose mode should produce output")
	}
}

func TestPrepareDoctor(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	report, err := prepareDoctor()
	if err != nil {
		t.Fatalf("prepareDoctor() error = %v", err)
	}

	// Should have multiple sections
	if len(report.Sections) == 0 {
		t.Error("report should have sections")
	}

	// Check for expected section names
	sectionNames := make(map[string]bool)
	for _, s := range report.Sections {
		sectionNames[s.Name] = true
	}

	expectedSections := []string{"Repository", "Version", "Configuration", "Environment"}
	for _, name := range expectedSections {
		if !sectionNames[name] {
			t.Errorf("missing section %q", name)
		}
	}
}

func TestPrepareDoctor_WithValidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create local config
	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	config := `
agents: {
	testAgent: {
		bin: "echo"
		command: "{{.Bin}} test"
	}
}

roles: {
	testRole: {
		prompt: "Test role"
	}
}

contexts: {
	testContext: {
		prompt: "Test context"
	}
}
`
	configFile := filepath.Join(configDir, "settings.cue")
	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	report, err := prepareDoctor()
	if err != nil {
		t.Fatalf("prepareDoctor() error = %v", err)
	}

	// Should have agent, role, context sections when config is loaded
	sectionNames := make(map[string]bool)
	for _, s := range report.Sections {
		sectionNames[s.Name] = true
	}

	// These sections should be present when config loads successfully
	if !sectionNames["Agents"] {
		t.Error("missing Agents section")
	}
	if !sectionNames["Roles"] {
		t.Error("missing Roles section")
	}
	if !sectionNames["Contexts"] {
		t.Error("missing Contexts section")
	}
}

func TestDoctorError(t *testing.T) {
	err := &doctorError{}

	if err.Error() != "issues found" {
		t.Errorf("Error() = %q, want %q", err.Error(), "issues found")
	}

	// Verify it's the same as the package-level error
	if errDoctorIssuesFound.Error() != "issues found" {
		t.Errorf("errDoctorIssuesFound.Error() = %q, want %q", errDoctorIssuesFound.Error(), "issues found")
	}
}
