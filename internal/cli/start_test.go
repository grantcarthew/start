package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/start/internal/orchestration"
)

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
		command: "{{.bin}} 'Agent executed'"
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
	configFile := filepath.Join(configDir, "settings.cue")
	if err := os.WriteFile(configFile, []byte(config), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	return tmpDir
}

func TestExecuteStart_DryRun(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	flags := &Flags{DryRun: true}

	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: true,
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeStart(stdout, stderr, flags, selection, "")
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
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	flags := &Flags{DryRun: true}

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

			err := executeStart(stdout, stderr, flags, tt.selection, "")
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
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeTask(stdout, stderr, flags, "test-task", "focus on testing")
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
			{Name: "ctx1", File: "file1.md", Default: true},
			{Name: "ctx2", File: "file2.md"},
		},
	}

	printDryRunSummary(buf, agent, "", "", result, "/tmp/test-dir")

	output := buf.String()

	expectedStrings := []string{
		"Dry Run",
		"test-agent",
		"test-role",
		"Context documents:",
		"ctx1",
		"ctx2",
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

func TestPrintContentPreview(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		maxLines      int
		wantTruncated bool
	}{
		{
			name:          "fewer lines than limit shows no count",
			text:          "line1\nline2",
			maxLines:      5,
			wantTruncated: false,
		},
		{
			name:          "more lines than limit shows count",
			text:          "line1\nline2\nline3\nline4\nline5\nline6",
			maxLines:      3,
			wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			printContentPreview(buf, "Test", tt.text, tt.maxLines)
			output := buf.String()

			if tt.wantTruncated {
				if !strings.Contains(output, fmt.Sprintf("(%d lines)", tt.maxLines)) {
					t.Errorf("Expected truncated header with line count, got: %s", output)
				}
				if !strings.Contains(output, "... (") {
					t.Errorf("Expected '... (X more lines)' suffix, got: %s", output)
				}
			} else {
				if strings.Contains(output, "lines)") {
					t.Errorf("Expected no line count for short content, got: %s", output)
				}
			}
		})
	}
}

func TestFindTask(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Load config for testing
	cfg, err := loadMergedConfig()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	tests := []struct {
		name        string
		taskName    string
		wantMatch   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "exact match",
			taskName:  "test-task",
			wantMatch: "test-task",
			wantErr:   false,
		},
		{
			name:      "prefix match",
			taskName:  "test",
			wantMatch: "test-task",
			wantErr:   false,
		},
		{
			name:        "no match",
			taskName:    "nonexistent",
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := findTask(cfg, tt.taskName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.wantMatch {
				t.Errorf("findTask() = %q, want %q", result, tt.wantMatch)
			}
		})
	}
}

func TestFindTask_AmbiguousPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	// Create config with multiple tasks that share a prefix
	config := `
agents: {
	echo: {
		bin: "echo"
		command: "{{.bin}} test"
	}
}

tasks: {
	"review-code": {
		prompt: "Review code"
	}
	"review-docs": {
		prompt: "Review documentation"
	}
	"review-tests": {
		prompt: "Review tests"
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
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	cfg, err := loadMergedConfig()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	// Ambiguous prefix should return error
	_, err = findTask(cfg, "review")
	if err == nil {
		t.Error("expected error for ambiguous prefix")
		return
	}

	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error = %q, want containing 'ambiguous'", err.Error())
	}

	// Should list matching tasks
	if !strings.Contains(err.Error(), "review-code") || !strings.Contains(err.Error(), "review-docs") {
		t.Errorf("error should list matching tasks: %v", err)
	}
}

func TestFindTask_NoTasksDefined(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	// Create config without tasks
	config := `
agents: {
	echo: {
		bin: "echo"
		command: "{{.bin}} test"
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
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	cfg, err := loadMergedConfig()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	_, err = findTask(cfg, "anything")
	if err == nil {
		t.Error("expected error when no tasks defined")
		return
	}

	if !strings.Contains(err.Error(), "no tasks defined") {
		t.Errorf("error = %q, want containing 'no tasks defined'", err.Error())
	}
}

// File path integration tests (DR-038)

func TestExecuteStart_FilePathRole(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Create a role file
	roleContent := "You are a file-based role for testing."
	roleFile := filepath.Join(tmpDir, "test-role.md")
	if err := os.WriteFile(roleFile, []byte(roleContent), 0644); err != nil {
		t.Fatalf("writing role file: %v", err)
	}

	flags := &Flags{
		DryRun: true,
		Role:   "./test-role.md",
	}

	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: true,
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeStart(stdout, stderr, flags, selection, "")
	if err != nil {
		t.Fatalf("executeStart() error = %v", err)
	}

	output := stdout.String()

	// Should show file path as role name
	if !strings.Contains(output, "./test-role.md") {
		t.Errorf("Expected file path in role output, got:\n%s", output)
	}

	// Should include the role content
	if !strings.Contains(output, "file-based role") {
		t.Errorf("Expected role content in output, got:\n%s", output)
	}
}

func TestExecuteStart_FilePathContext(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Create a context file
	ctxContent := "File-based context content for testing."
	ctxFile := filepath.Join(tmpDir, "test-context.md")
	if err := os.WriteFile(ctxFile, []byte(ctxContent), 0644); err != nil {
		t.Fatalf("writing context file: %v", err)
	}

	flags := &Flags{
		DryRun:  true,
		Context: []string{"./test-context.md"},
	}

	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		Tags:            flags.Context,
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeStart(stdout, stderr, flags, selection, "")
	if err != nil {
		t.Fatalf("executeStart() error = %v", err)
	}

	output := stdout.String()

	// Should show file path as context name
	if !strings.Contains(output, "./test-context.md") {
		t.Errorf("Expected file path in context output, got:\n%s", output)
	}
}

func TestExecuteStart_MixedContextOrder(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Create context files
	if err := os.WriteFile(filepath.Join(tmpDir, "first.md"), []byte("First file context"), 0644); err != nil {
		t.Fatalf("writing first.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "last.md"), []byte("Last file context"), 0644); err != nil {
		t.Fatalf("writing last.md: %v", err)
	}

	// Mixed order: file, config tag (default), file
	flags := &Flags{
		DryRun:  true,
		Context: []string{"./first.md", "default", "./last.md"},
	}

	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		Tags:            flags.Context,
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeStart(stdout, stderr, flags, selection, "")
	if err != nil {
		t.Fatalf("executeStart() error = %v", err)
	}

	output := stdout.String()

	// Verify order is preserved: first.md should appear before project (default), which should appear before last.md
	firstIdx := strings.Index(output, "./first.md")
	projectIdx := strings.Index(output, "project")
	lastIdx := strings.Index(output, "./last.md")

	if firstIdx == -1 {
		t.Error("Expected ./first.md in output")
	}
	if projectIdx == -1 {
		t.Error("Expected project (default context) in output")
	}
	if lastIdx == -1 {
		t.Error("Expected ./last.md in output")
	}

	if firstIdx != -1 && projectIdx != -1 && lastIdx != -1 {
		if firstIdx >= projectIdx || projectIdx >= lastIdx {
			t.Errorf("Context order not preserved: first.md(%d) < project(%d) < last.md(%d)",
				firstIdx, projectIdx, lastIdx)
		}
	}
}

func TestExecuteTask_FilePathTask(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Create a task file
	taskContent := "File-based task prompt for testing."
	taskFile := filepath.Join(tmpDir, "test-task.md")
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("writing task file: %v", err)
	}

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeTask(stdout, stderr, flags, "./test-task.md", "")
	if err != nil {
		t.Fatalf("executeTask() error = %v", err)
	}

	output := stdout.String()

	// Should show file path as task name
	if !strings.Contains(output, "./test-task.md") {
		t.Errorf("Expected file path in task output, got:\n%s", output)
	}

	// Should include task content
	if !strings.Contains(output, "File-based task prompt") {
		t.Errorf("Expected task content in output, got:\n%s", output)
	}
}

func TestExecuteTask_FilePathWithInstructions(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Create a task file with instructions placeholder
	taskContent := "Review this code.\nInstructions: {{.instructions}}"
	taskFile := filepath.Join(tmpDir, "review-task.md")
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("writing task file: %v", err)
	}

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeTask(stdout, stderr, flags, "./review-task.md", "focus on security")
	if err != nil {
		t.Fatalf("executeTask() error = %v", err)
	}

	output := stdout.String()

	// Should have substituted instructions
	if !strings.Contains(output, "focus on security") {
		t.Errorf("Expected instructions to be substituted, got:\n%s", output)
	}

	// Should NOT contain the placeholder
	if strings.Contains(output, "{{.instructions}}") {
		t.Errorf("Template placeholder was not substituted, got:\n%s", output)
	}
}

func TestExecuteTask_FilePathMissing(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err = executeTask(stdout, stderr, flags, "./nonexistent.md", "")

	if err == nil {
		t.Error("Expected error for missing file")
		return
	}

	// Should include file path in error
	if !strings.Contains(err.Error(), "./nonexistent.md") {
		t.Errorf("Error should contain file path: %v", err)
	}
}

func TestExecuteStart_FilePathContextMissing(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting working dir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	flags := &Flags{
		DryRun:  true,
		Context: []string{"./missing-context.md"},
	}

	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		Tags:            flags.Context,
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	// Missing context files should not cause fatal error (per DR-038: show ○ status)
	err = executeStart(stdout, stderr, flags, selection, "")
	if err != nil {
		t.Fatalf("executeStart() should not fail for missing context file: %v", err)
	}

	output := stdout.String()

	// Should show the missing context with error indicator
	if !strings.Contains(output, "./missing-context.md") {
		t.Errorf("Expected missing file path in output, got:\n%s", output)
	}
}
