package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/registry"
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

	err = executeTask(stdout, stderr, strings.NewReader(""), flags, "test-task", "focus on testing")
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
			result, matches, err := findTask(cfg, tt.taskName)

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

			if len(matches) > 0 {
				t.Errorf("unexpected multiple matches: %v", matches)
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

	// Isolate from global config
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

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

	// Ambiguous prefix should return matches (per DR-015)
	result, matches, err := findTask(cfg, "review")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if result != "" {
		t.Errorf("expected empty result for multiple matches, got %q", result)
	}

	if len(matches) != 3 {
		t.Errorf("expected 3 matches, got %d: %v", len(matches), matches)
		return
	}

	// Should contain all matching tasks
	hasCode := false
	hasDocs := false
	hasTests := false
	for _, m := range matches {
		if m == "review-code" {
			hasCode = true
		}
		if m == "review-docs" {
			hasDocs = true
		}
		if m == "review-tests" {
			hasTests = true
		}
	}
	if !hasCode || !hasDocs || !hasTests {
		t.Errorf("matches should contain review-code, review-docs, review-tests: %v", matches)
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

	_, _, err = findTask(cfg, "anything")
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

	err = executeTask(stdout, stderr, strings.NewReader(""), flags, "./test-task.md", "")
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

	err = executeTask(stdout, stderr, strings.NewReader(""), flags, "./review-task.md", "focus on security")
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

	err = executeTask(stdout, stderr, strings.NewReader(""), flags, "./nonexistent.md", "")

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

// Tests for unified task resolution (DR-015 update)

func TestHasExactInstalledTask(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
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

	tests := []struct {
		name     string
		taskName string
		want     bool
	}{
		{
			name:     "exact match exists",
			taskName: "test-task",
			want:     true,
		},
		{
			name:     "partial match does not count",
			taskName: "test",
			want:     false,
		},
		{
			name:     "nonexistent task",
			taskName: "nonexistent",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasExactInstalledTask(cfg, tt.taskName)
			if got != tt.want {
				t.Errorf("hasExactInstalledTask(%q) = %v, want %v", tt.taskName, got, tt.want)
			}
		})
	}
}

func TestFindInstalledTasks(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	oldHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	config := `
agents: {
	echo: {
		bin: "echo"
		command: "{{.bin}} test"
	}
}

tasks: {
	"golang/debug": {
		prompt: "Debug Go code"
	}
	"golang/refactor": {
		prompt: "Refactor Go code"
	}
	"python/debug": {
		prompt: "Debug Python code"
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

	tests := []struct {
		name       string
		searchTerm string
		wantCount  int
		wantNames  []string
	}{
		{
			name:       "match golang tasks",
			searchTerm: "golang",
			wantCount:  2,
			wantNames:  []string{"golang/debug", "golang/refactor"},
		},
		{
			name:       "match debug tasks",
			searchTerm: "debug",
			wantCount:  2,
			wantNames:  []string{"golang/debug", "python/debug"},
		},
		{
			name:       "no matches",
			searchTerm: "nonexistent",
			wantCount:  0,
			wantNames:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := findInstalledTasks(cfg, tt.searchTerm)
			if len(matches) != tt.wantCount {
				t.Errorf("findInstalledTasks(%q) returned %d matches, want %d", tt.searchTerm, len(matches), tt.wantCount)
			}

			for _, m := range matches {
				if m.Source != TaskSourceInstalled {
					t.Errorf("match %q has source %q, want %q", m.Name, m.Source, TaskSourceInstalled)
				}
			}

			if tt.wantNames != nil {
				for _, wantName := range tt.wantNames {
					found := false
					for _, m := range matches {
						if m.Name == wantName {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected match %q not found in results", wantName)
					}
				}
			}
		})
	}
}

func TestFindRegistryTasks(t *testing.T) {
	// Create mock registry index
	index := &registry.Index{
		Tasks: map[string]registry.IndexEntry{
			"golang/debug": {
				Module:      "github.com/example/golang-debug@v0",
				Description: "Debug Go code",
			},
			"golang/review": {
				Module:      "github.com/example/golang-review@v0",
				Description: "Review Go code",
			},
			"python/debug": {
				Module:      "github.com/example/python-debug@v0",
				Description: "Debug Python code",
			},
		},
	}

	tests := []struct {
		name       string
		searchTerm string
		wantCount  int
		wantNames  []string
	}{
		{
			name:       "match golang tasks",
			searchTerm: "golang",
			wantCount:  2,
			wantNames:  []string{"golang/debug", "golang/review"},
		},
		{
			name:       "match debug tasks",
			searchTerm: "debug",
			wantCount:  2,
			wantNames:  []string{"golang/debug", "python/debug"},
		},
		{
			name:       "no matches",
			searchTerm: "nonexistent",
			wantCount:  0,
			wantNames:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := findRegistryTasks(index, tt.searchTerm)
			if len(matches) != tt.wantCount {
				t.Errorf("findRegistryTasks(%q) returned %d matches, want %d", tt.searchTerm, len(matches), tt.wantCount)
			}

			for _, m := range matches {
				if m.Source != TaskSourceRegistry {
					t.Errorf("match %q has source %q, want %q", m.Name, m.Source, TaskSourceRegistry)
				}
			}

			if tt.wantNames != nil {
				for _, wantName := range tt.wantNames {
					found := false
					for _, m := range matches {
						if m.Name == wantName {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected match %q not found in results", wantName)
					}
				}
			}
		})
	}
}

func TestMergeTaskMatches(t *testing.T) {
	installed := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "golang/refactor", Source: TaskSourceInstalled},
	}

	registry := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceRegistry},  // duplicate - should be excluded
		{Name: "golang/review", Source: TaskSourceRegistry}, // new - should be included
		{Name: "python/debug", Source: TaskSourceRegistry},  // new - should be included
	}

	merged := mergeTaskMatches(installed, registry)

	// Should have 4 unique tasks
	if len(merged) != 4 {
		t.Errorf("mergeTaskMatches returned %d results, want 4", len(merged))
	}

	// Check that golang/debug is from installed (not registry)
	for _, m := range merged {
		if m.Name == "golang/debug" {
			if m.Source != TaskSourceInstalled {
				t.Errorf("golang/debug should have source 'installed', got %q", m.Source)
			}
		}
	}

	// Check results are sorted alphabetically
	for i := 1; i < len(merged); i++ {
		if merged[i-1].Name > merged[i].Name {
			t.Errorf("results not sorted: %q > %q", merged[i-1].Name, merged[i].Name)
		}
	}
}

func TestPromptTaskSelection_NonTTY(t *testing.T) {
	matches := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "golang/refactor", Source: TaskSourceRegistry},
	}

	stdout := new(bytes.Buffer)
	stdin := strings.NewReader("")

	_, err := promptTaskSelection(stdout, stdin, matches, "golang")
	if err == nil {
		t.Error("expected error for non-TTY input")
		return
	}

	errMsg := err.Error()

	// Error should mention ambiguous
	if !strings.Contains(errMsg, "ambiguous") {
		t.Errorf("error should mention ambiguous: %v", err)
	}

	// Error should list task names
	if !strings.Contains(errMsg, "golang/debug") {
		t.Errorf("error should contain 'golang/debug': %v", err)
	}
	if !strings.Contains(errMsg, "golang/refactor") {
		t.Errorf("error should contain 'golang/refactor': %v", err)
	}
}

func TestPromptTaskSelection_NonTTY_ManyMatches(t *testing.T) {
	// Create 25 matches (more than maxTaskResults=20)
	var matches []TaskMatch
	for i := 1; i <= 25; i++ {
		matches = append(matches, TaskMatch{
			Name:   fmt.Sprintf("task/%02d", i),
			Source: TaskSourceInstalled,
		})
	}

	stdout := new(bytes.Buffer)
	stdin := strings.NewReader("")

	_, err := promptTaskSelection(stdout, stdin, matches, "task")
	if err == nil {
		t.Error("expected error for non-TTY input")
		return
	}

	errMsg := err.Error()

	// Error should mention ambiguous
	if !strings.Contains(errMsg, "ambiguous") {
		t.Errorf("error should mention ambiguous: %v", err)
	}

	// Error should list some task names
	if !strings.Contains(errMsg, "task/01") {
		t.Errorf("error should contain task names: %v", err)
	}
}

func TestMergeTaskMatches_Sorting(t *testing.T) {
	// Test that results are sorted alphabetically
	installed := []TaskMatch{
		{Name: "zebra/task", Source: TaskSourceInstalled},
		{Name: "alpha/task", Source: TaskSourceInstalled},
	}

	registry := []TaskMatch{
		{Name: "middle/task", Source: TaskSourceRegistry},
	}

	merged := mergeTaskMatches(installed, registry)

	if len(merged) != 3 {
		t.Fatalf("expected 3 results, got %d", len(merged))
	}

	// Check alphabetical order
	expected := []string{"alpha/task", "middle/task", "zebra/task"}
	for i, want := range expected {
		if merged[i].Name != want {
			t.Errorf("position %d: got %q, want %q", i, merged[i].Name, want)
		}
	}
}

func TestMergeTaskMatches_Empty(t *testing.T) {
	// Test with empty inputs
	merged := mergeTaskMatches(nil, nil)
	if len(merged) != 0 {
		t.Errorf("expected 0 results for empty inputs, got %d", len(merged))
	}

	// Test with only installed
	installed := []TaskMatch{{Name: "task1", Source: TaskSourceInstalled}}
	merged = mergeTaskMatches(installed, nil)
	if len(merged) != 1 {
		t.Errorf("expected 1 result, got %d", len(merged))
	}

	// Test with only registry
	registry := []TaskMatch{{Name: "task2", Source: TaskSourceRegistry}}
	merged = mergeTaskMatches(nil, registry)
	if len(merged) != 1 {
		t.Errorf("expected 1 result, got %d", len(merged))
	}
}
