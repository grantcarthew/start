package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/start/internal/orchestration"
)

// resetFlags resets all global flags to their default values.
func resetFlags() {
	flagAgent = ""
	flagRole = ""
	flagModel = ""
	flagContext = nil
	flagDryRun = false
	flagQuiet = false
	flagVerbose = false
}

// setupStartTestConfig creates a minimal CUE config for start command testing.
func setupStartTestConfig(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	config := `
agents: {
	echo: {
		bin: "echo"
		command: "{{.Bin}} 'Agent executed'"
		default_model: "default"
		models: {
			default: "echo-model"
		}
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
	project: {
		default: true
		prompt: "Project context"
	}
}

tasks: {
	"test-task": {
		role: "assistant"
		prompt: """
			Test task prompt.
			Instructions: {{.Instructions}}
			"""
	}
}

settings: {
	default_agent: "echo"
	default_role: "assistant"
}
`
	configFile := filepath.Join(configDir, "config.cue")
	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return tmpDir
}

func TestExecuteStart_DryRun(t *testing.T) {
	resetFlags()
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Set flags directly
	flagDryRun = true

	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: true,
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeStart(stdout, stderr, selection, "")
	if err != nil {
		t.Fatalf("executeStart() error = %v", err)
	}

	output := stdout.String()

	// Should show dry run header
	if !strings.Contains(output, "Dry Run") {
		t.Errorf("Expected 'Dry Run' in output, got:\n%s", output)
	}

	// Should show agent
	if !strings.Contains(output, "echo") {
		t.Errorf("Expected agent 'echo' in output")
	}

	// Should show role
	if !strings.Contains(output, "assistant") {
		t.Errorf("Expected role 'assistant' in output")
	}
}

func TestExecuteStart_ContextSelection(t *testing.T) {
	resetFlags()
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	flagDryRun = true

	tests := []struct {
		name        string
		selection   orchestration.ContextSelection
		wantContext string
	}{
		{
			name: "required and default",
			selection: orchestration.ContextSelection{
				IncludeRequired: true,
				IncludeDefaults: true,
			},
			wantContext: "env",
		},
		{
			name: "required only",
			selection: orchestration.ContextSelection{
				IncludeRequired: true,
				IncludeDefaults: false,
			},
			wantContext: "env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := new(bytes.Buffer)
			stderr := new(bytes.Buffer)

			err := executeStart(stdout, stderr, tt.selection, "")
			if err != nil {
				t.Fatalf("executeStart() error = %v", err)
			}

			output := stdout.String()
			if !strings.Contains(output, tt.wantContext) {
				t.Errorf("Expected context %q in output", tt.wantContext)
			}
		})
	}
}

func TestExecuteTask_DryRun(t *testing.T) {
	resetFlags()
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer os.Chdir(origDir)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	flagDryRun = true

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeTask(stdout, stderr, "test-task", "focus on testing")
	if err != nil {
		t.Fatalf("executeTask() error = %v", err)
	}

	output := stdout.String()

	// Should show task name
	if !strings.Contains(output, "test-task") {
		t.Errorf("Expected task name in output")
	}

	// Should show instructions
	if !strings.Contains(output, "focus on testing") {
		t.Errorf("Expected instructions in output")
	}

	// Should show dry run header
	if !strings.Contains(output, "Dry Run") {
		t.Errorf("Expected 'Dry Run' in output")
	}
}

func TestPrintDryRunSummary(t *testing.T) {
	buf := new(bytes.Buffer)

	agent := orchestration.Agent{
		Name:         "test-agent",
		DefaultModel: "test-model",
	}

	result := orchestration.ComposeResult{
		Role:     "You are a test assistant.",
		RoleName: "test-role",
		Prompt:   "Test prompt content",
		Contexts: []orchestration.Context{
			{Name: "ctx1"},
			{Name: "ctx2"},
		},
	}

	printDryRunSummary(buf, agent, "", result, "/tmp/test-dir")

	output := buf.String()

	expectedStrings := []string{
		"Dry Run",
		"test-agent",
		"test-role",
		"ctx1, ctx2",
		"/tmp/test-dir",
		"role.md",
		"prompt.md",
		"command.txt",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected %q in output", expected)
		}
	}
}

func TestPrintPreviewLines(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		n       int
		wantLen int
	}{
		{
			name:    "fewer lines than limit",
			text:    "line1\nline2",
			n:       5,
			wantLen: 2,
		},
		{
			name:    "more lines than limit",
			text:    "line1\nline2\nline3\nline4\nline5\nline6",
			n:       3,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			printPreviewLines(buf, tt.text, tt.n)
			output := buf.String()

			lines := strings.Split(strings.TrimSpace(output), "\n")
			// Account for possible "... (X more lines)" line
			if len(lines) < tt.wantLen {
				t.Errorf("Expected at least %d lines, got %d", tt.wantLen, len(lines))
			}
		})
	}
}
