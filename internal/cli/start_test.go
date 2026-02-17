package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalcue "github.com/grantcarthew/start/internal/cue"
	"github.com/grantcarthew/start/internal/orchestration"
	"github.com/grantcarthew/start/internal/registry"
)

// chdir changes to the given directory and registers a cleanup to restore the original.
func chdir(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to dir %s: %v", dir, err)
	}
}

// setupStartTestConfig creates a minimal CUE config for start command testing.
func setupStartTestConfig(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

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
			Instructions: {{.instructions}}
			"""
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
	return tmpDir
}

// Note: Tests below use os.Chdir (process-global state). Do not add t.Parallel()
// to any test that calls os.Chdir — it will cause data races on the working directory.

func TestExecuteStart_DryRun(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	chdir(t, tmpDir)

	flags := &Flags{DryRun: true}

	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: true,
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := executeStart(stdout, stderr, strings.NewReader(""), flags, selection, "")
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

func TestExecuteStart_NoRole(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	chdir(t, tmpDir)

	flags := &Flags{DryRun: true, NoRole: true}

	selection := orchestration.ContextSelection{
		IncludeRequired: true,
		IncludeDefaults: true,
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := executeStart(stdout, stderr, strings.NewReader(""), flags, selection, "")
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

	// Should still show contexts
	if !strings.Contains(output, "env") {
		t.Errorf("Expected context 'env' in output")
	}

	// Should NOT contain role content
	if strings.Contains(output, "You are a helpful assistant") {
		t.Errorf("Expected no role content in output, got:\n%s", output)
	}

	// Role name should be empty
	if strings.Contains(output, "assistant") {
		t.Errorf("Expected no role name 'assistant' in output, got:\n%s", output)
	}
}

func TestExecuteTask_NoRole(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	chdir(t, tmpDir)

	flags := &Flags{DryRun: true, NoRole: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := executeTask(stdout, stderr, strings.NewReader(""), flags, "test-task", "focus on testing")
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

	// Should NOT contain role content (task has role: "assistant" configured)
	if strings.Contains(output, "You are a helpful assistant") {
		t.Errorf("Expected no role content with --no-role, got:\n%s", output)
	}
}

func TestExecuteTask_MissingTaskRole(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	// Config with a task referencing a role that does NOT exist
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

roles: {}

tasks: {
	"test-task": {
		role: "missing-role"
		prompt: "Test task prompt."
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

	chdir(t, tmpDir)

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := executeTask(stdout, stderr, strings.NewReader(""), flags, "test-task", "")
	if err == nil {
		t.Fatal("Expected error for missing task role, got nil")
	}

	// The resolver should have been invoked and failed (no registry in tests).
	// Error should mention the role name.
	if !strings.Contains(err.Error(), "missing-role") {
		t.Errorf("Expected error to mention %q, got: %v", "missing-role", err)
	}
}

func TestExecuteStart_ContextSelection(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	chdir(t, tmpDir)

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

			err := executeStart(stdout, stderr, strings.NewReader(""), flags, tt.selection, "")
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
	chdir(t, tmpDir)

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := executeTask(stdout, stderr, strings.NewReader(""), flags, "test-task", "focus on testing")
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
		RoleResolutions: []orchestration.RoleResolution{
			{Name: "test-role", Status: "loaded", File: "test-role.md"},
		},
	}

	printDryRunSummary(buf, agent, "", "", result, "/tmp/test-dir")

	output := buf.String()

	expectedStrings := []string{
		"Dry Run",
		"test-agent",
		"test-role",
		"Context:",
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
			name:          "lines within threshold shows all",
			text:          "line1\nline2\nline3\nline4\nline5\nline6",
			maxLines:      3,
			wantTruncated: false,
		},
		{
			name:          "lines beyond threshold truncates",
			text:          "line1\nline2\nline3\nline4\nline5\nline6\nline7",
			maxLines:      3,
			wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			printContentPreview(buf, "Test", colorDim, tt.text, tt.maxLines)
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

func TestTaskResolution(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	chdir(t, tmpDir)

	// Load config for testing
	cfg, err := loadMergedConfig()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	t.Run("exact match", func(t *testing.T) {
		got, err := findExactInstalledName(cfg.Value, internalcue.KeyTasks, "test-task")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "test-task" {
			t.Errorf("findExactInstalledName() = %q, want %q", got, "test-task")
		}
	})

	t.Run("exact match not found", func(t *testing.T) {
		got, err := findExactInstalledName(cfg.Value, internalcue.KeyTasks, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("findExactInstalledName() = %q, want empty", got)
		}
	})

	t.Run("substring match", func(t *testing.T) {
		matches, err := findInstalledTasks(cfg, "test")
		if err != nil {
			t.Fatalf("findInstalledTasks() error: %v", err)
		}
		if len(matches) != 1 {
			t.Fatalf("findInstalledTasks() returned %d results, want 1", len(matches))
		}
		if matches[0].Name != "test-task" {
			t.Errorf("findInstalledTasks() name = %q, want %q", matches[0].Name, "test-task")
		}
	})

	t.Run("no match", func(t *testing.T) {
		matches, err := findInstalledTasks(cfg, "nonexistent")
		if err != nil {
			t.Fatalf("findInstalledTasks() error: %v", err)
		}
		if len(matches) != 0 {
			t.Errorf("findInstalledTasks() returned %d results, want 0", len(matches))
		}
	})
}

func TestTaskResolution_AmbiguousPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

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

	chdir(t, tmpDir)

	cfg, err := loadMergedConfig()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	// Ambiguous prefix should return multiple matches
	matches, err := findInstalledTasks(cfg, "review")
	if err != nil {
		t.Fatalf("findInstalledTasks() error: %v", err)
	}
	if len(matches) != 3 {
		t.Fatalf("expected 3 matches, got %d: %v", len(matches), matches)
	}

	// Should contain all matching tasks
	names := make(map[string]bool)
	for _, m := range matches {
		names[m.Name] = true
	}
	for _, want := range []string{"review-code", "review-docs", "review-tests"} {
		if !names[want] {
			t.Errorf("matches missing %q, got %v", want, matches)
		}
	}

	// Multi-term AND should narrow results
	matches, err = findInstalledTasks(cfg, "review,code")
	if err != nil {
		t.Fatalf("findInstalledTasks() error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match for 'review,code', got %d: %v", len(matches), matches)
	}
	if matches[0].Name != "review-code" {
		t.Errorf("match name = %q, want %q", matches[0].Name, "review-code")
	}
}

func TestTaskResolution_ExactMatchFallsThrough(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// CUE registry extracts modules with read-only permissions which
	// breaks t.TempDir() cleanup. Fix permissions before removal.
	t.Cleanup(func() {
		_ = filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			_ = os.Chmod(path, 0755)
			return nil
		})
	})

	configDir := filepath.Join(tmpDir, ".start")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("creating config dir: %v", err)
	}

	// Config where "review" is both an exact task name AND a substring
	// matching "start/review". The multipleInstalled path should cause
	// executeTask to fall through to selection instead of silently
	// running the exact match.
	config := `
agents: {
	echo: {
		bin: "echo"
		command: "{{.bin}} test"
	}
}

tasks: {
	"review": {
		prompt: "General review"
	}
	"start/review": {
		prompt: "Start review"
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

	chdir(t, tmpDir)

	cfg, err := loadMergedConfig()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	// Precondition: exact match exists
	exact, err := findExactInstalledName(cfg.Value, internalcue.KeyTasks, "review")
	if err != nil {
		t.Fatalf("findExactInstalledName error: %v", err)
	}
	if exact != "review" {
		t.Fatalf("expected exact match %q, got %q", "review", exact)
	}

	// Precondition: multiple regex matches exist
	matches, err := findInstalledTasks(cfg, "review")
	if err != nil {
		t.Fatalf("findInstalledTasks error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %v", len(matches), matches)
	}

	// executeTask should NOT silently use the exact match.
	// With non-TTY stdin, multiple matches produce an ambiguous error.
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	flags := &Flags{Quiet: true}
	err = executeTask(stdout, stderr, strings.NewReader(""), flags, "review", "")
	if err == nil {
		t.Fatal("expected ambiguous task error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected error containing %q, got: %v", "ambiguous", err)
	}
}

func TestTaskResolution_NoTasksDefined(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

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

	chdir(t, tmpDir)

	cfg, err := loadMergedConfig()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}

	if got, _ := findExactInstalledName(cfg.Value, internalcue.KeyTasks, "anything"); got != "" {
		t.Errorf("findExactInstalledName() = %q, want empty for missing tasks", got)
	}

	matches, err := findInstalledTasks(cfg, "anything")
	if err != nil {
		t.Fatalf("findInstalledTasks() error: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("findInstalledTasks() returned %d results, want 0", len(matches))
	}
}

// File path integration tests (DR-038)

func TestExecuteStart_FilePathRole(t *testing.T) {
	tmpDir := setupStartTestConfig(t)
	chdir(t, tmpDir)

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

	err := executeStart(stdout, stderr, strings.NewReader(""), flags, selection, "")
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
	chdir(t, tmpDir)

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

	err := executeStart(stdout, stderr, strings.NewReader(""), flags, selection, "")
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
	chdir(t, tmpDir)

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

	err := executeStart(stdout, stderr, strings.NewReader(""), flags, selection, "")
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
	chdir(t, tmpDir)

	// Create a task file
	taskContent := "File-based task prompt for testing."
	taskFile := filepath.Join(tmpDir, "test-task.md")
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("writing task file: %v", err)
	}

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := executeTask(stdout, stderr, strings.NewReader(""), flags, "./test-task.md", "")
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
	chdir(t, tmpDir)

	// Create a task file with instructions placeholder
	taskContent := "Review this code.\nInstructions: {{.instructions}}"
	taskFile := filepath.Join(tmpDir, "review-task.md")
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("writing task file: %v", err)
	}

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := executeTask(stdout, stderr, strings.NewReader(""), flags, "./review-task.md", "focus on security")
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
	chdir(t, tmpDir)

	flags := &Flags{DryRun: true}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	err := executeTask(stdout, stderr, strings.NewReader(""), flags, "./nonexistent.md", "")

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
	chdir(t, tmpDir)

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
	err := executeStart(stdout, stderr, strings.NewReader(""), flags, selection, "")
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

func TestFindInstalledTasks(t *testing.T) {
	tmpDir := t.TempDir()

	// Isolate from global config
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

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

	chdir(t, tmpDir)

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
			matches, err := findInstalledTasks(cfg, tt.searchTerm)
			if err != nil {
				t.Fatalf("findInstalledTasks() error: %v", err)
			}
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
			matches, err := findRegistryTasks(index, tt.searchTerm)
			if err != nil {
				t.Fatalf("findRegistryTasks() error: %v", err)
			}
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

func TestPromptTaskSelection_ByNumber(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("2\n"))

	matches := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "golang/review", Source: TaskSourceRegistry},
		{Name: "python/debug", Source: TaskSourceInstalled},
	}

	selected, err := promptTaskSelection(&buf, reader, matches, "debug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name != "golang/review" {
		t.Errorf("expected %q, got %q", "golang/review", selected.Name)
	}
	if selected.Source != TaskSourceRegistry {
		t.Errorf("expected source %q, got %q", TaskSourceRegistry, selected.Source)
	}
}

func TestPromptTaskSelection_ByExactName(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("golang/review\n"))

	matches := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "golang/review", Source: TaskSourceRegistry},
	}

	selected, err := promptTaskSelection(&buf, reader, matches, "golang")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name != "golang/review" {
		t.Errorf("expected %q, got %q", "golang/review", selected.Name)
	}
}

func TestPromptTaskSelection_ByExactNameCaseInsensitive(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("GOLANG/DEBUG\n"))

	matches := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "golang/review", Source: TaskSourceRegistry},
	}

	selected, err := promptTaskSelection(&buf, reader, matches, "golang")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name != "golang/debug" {
		t.Errorf("expected %q, got %q", "golang/debug", selected.Name)
	}
}

func TestPromptTaskSelection_BySubstring(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("review\n"))

	matches := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "golang/review", Source: TaskSourceRegistry},
	}

	selected, err := promptTaskSelection(&buf, reader, matches, "golang")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name != "golang/review" {
		t.Errorf("expected %q, got %q", "golang/review", selected.Name)
	}
}

func TestPromptTaskSelection_InvalidNumber(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("5\n"))

	matches := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "golang/review", Source: TaskSourceRegistry},
	}

	_, err := promptTaskSelection(&buf, reader, matches, "golang")
	if err == nil {
		t.Fatal("expected error for out-of-range number")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' in error, got: %v", err)
	}
}

func TestPromptTaskSelection_AmbiguousSubstring(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("golang\n"))

	matches := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "golang/review", Source: TaskSourceRegistry},
	}

	_, err := promptTaskSelection(&buf, reader, matches, "go")
	if err == nil {
		t.Fatal("expected error for ambiguous substring")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' in error, got: %v", err)
	}
}

func TestPromptTaskSelection_Truncation(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("1\n"))

	// Create more than maxTaskResults matches
	matches := make([]TaskMatch, 25)
	for i := range matches {
		matches[i] = TaskMatch{
			Name:   fmt.Sprintf("task-%02d", i+1),
			Source: TaskSourceInstalled,
		}
	}

	selected, err := promptTaskSelection(&buf, reader, matches, "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.Name != "task-01" {
		t.Errorf("expected %q, got %q", "task-01", selected.Name)
	}

	output := buf.String()
	if !strings.Contains(output, "25 tasks") {
		t.Errorf("expected total count in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Showing 20 of 25") {
		t.Errorf("expected truncation message, got:\n%s", output)
	}
}

func TestPromptTaskSelection_TruncationRejectsOutOfRange(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	// Try to select item 21 when only 20 are displayed
	reader := bufio.NewReader(strings.NewReader("21\n"))

	matches := make([]TaskMatch, 25)
	for i := range matches {
		matches[i] = TaskMatch{
			Name:   fmt.Sprintf("task-%02d", i+1),
			Source: TaskSourceInstalled,
		}
	}

	_, err := promptTaskSelection(&buf, reader, matches, "task")
	if err == nil {
		t.Fatal("expected error for selecting beyond displayed range")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' in error, got: %v", err)
	}
}

func TestPromptTaskSelection_DisplayFormat(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("1\n"))

	matches := []TaskMatch{
		{Name: "golang/debug", Source: TaskSourceInstalled},
		{Name: "python/review", Source: TaskSourceRegistry},
	}

	_, err := promptTaskSelection(&buf, reader, matches, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should show source labels
	if !strings.Contains(output, string(TaskSourceInstalled)) {
		t.Errorf("expected %q source label in output, got:\n%s", TaskSourceInstalled, output)
	}
	if !strings.Contains(output, string(TaskSourceRegistry)) {
		t.Errorf("expected %q source label in output, got:\n%s", TaskSourceRegistry, output)
	}
	// Should show match count
	if !strings.Contains(output, "2 tasks") {
		t.Errorf("expected '2 tasks' in output, got:\n%s", output)
	}
}

func TestGetConfiguredAgents(t *testing.T) {
	t.Parallel()
	cfg := buildTestCfg(t, `{
		agents: {
			claude: {
				bin: "claude"
				command: "{{.bin}}"
				description: "Anthropic Claude"
			}
			copilot: {
				bin: "gh"
				command: "{{.bin}} copilot"
				description: "GitHub Copilot"
			}
			aider: {
				bin: "aider"
				command: "{{.bin}}"
			}
		}
	}`)

	choices := getConfiguredAgents(cfg.Value)
	if len(choices) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(choices))
	}
	if choices[0].Name != "claude" {
		t.Errorf("expected first agent 'claude', got %q", choices[0].Name)
	}
	if choices[0].Description != "Anthropic Claude" {
		t.Errorf("expected description 'Anthropic Claude', got %q", choices[0].Description)
	}
	if choices[1].Name != "copilot" {
		t.Errorf("expected second agent 'copilot', got %q", choices[1].Name)
	}
	if choices[2].Name != "aider" {
		t.Errorf("expected third agent 'aider', got %q", choices[2].Name)
	}
	if choices[2].Description != "" {
		t.Errorf("expected empty description for aider, got %q", choices[2].Description)
	}
}

func TestGetConfiguredAgents_Empty(t *testing.T) {
	t.Parallel()
	cfg := buildTestCfg(t, `{
		roles: {
			assistant: { prompt: "hello" }
		}
	}`)

	choices := getConfiguredAgents(cfg.Value)
	if len(choices) != 0 {
		t.Errorf("expected 0 agents, got %d", len(choices))
	}
}

func TestPromptAgentSelection_ByNumber(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("2\n"))

	choices := []agentChoice{
		{Name: "claude", Description: "Anthropic Claude"},
		{Name: "copilot", Description: "GitHub Copilot"},
	}

	selected, err := promptAgentSelection(&buf, reader, choices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != "copilot" {
		t.Errorf("expected 'copilot', got %q", selected)
	}
}

func TestPromptAgentSelection_ByExactName(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("copilot\n"))

	choices := []agentChoice{
		{Name: "claude", Description: "Anthropic Claude"},
		{Name: "copilot", Description: "GitHub Copilot"},
	}

	selected, err := promptAgentSelection(&buf, reader, choices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != "copilot" {
		t.Errorf("expected 'copilot', got %q", selected)
	}
}

func TestPromptAgentSelection_ByExactNameCaseInsensitive(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("CLAUDE\n"))

	choices := []agentChoice{
		{Name: "claude", Description: "Anthropic Claude"},
		{Name: "copilot", Description: "GitHub Copilot"},
	}

	selected, err := promptAgentSelection(&buf, reader, choices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != "claude" {
		t.Errorf("expected 'claude', got %q", selected)
	}
}

func TestPromptAgentSelection_BySubstring(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("cop\n"))

	choices := []agentChoice{
		{Name: "claude", Description: "Anthropic Claude"},
		{Name: "copilot", Description: "GitHub Copilot"},
	}

	selected, err := promptAgentSelection(&buf, reader, choices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != "copilot" {
		t.Errorf("expected 'copilot', got %q", selected)
	}
}

func TestPromptAgentSelection_InvalidNumber(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("5\n"))

	choices := []agentChoice{
		{Name: "claude", Description: "Anthropic Claude"},
		{Name: "copilot", Description: "GitHub Copilot"},
	}

	_, err := promptAgentSelection(&buf, reader, choices)
	if err == nil {
		t.Fatal("expected error for out-of-range number")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' in error, got: %v", err)
	}
}

func TestPromptAgentSelection_AmbiguousSubstring(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("c\n"))

	choices := []agentChoice{
		{Name: "claude", Description: "Anthropic Claude"},
		{Name: "copilot", Description: "GitHub Copilot"},
	}

	_, err := promptAgentSelection(&buf, reader, choices)
	if err == nil {
		t.Fatal("expected error for ambiguous substring")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' in error, got: %v", err)
	}
}

func TestPromptAgentSelection_EmptyInput(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("\n"))

	choices := []agentChoice{
		{Name: "claude", Description: "Anthropic Claude"},
		{Name: "copilot", Description: "GitHub Copilot"},
	}

	_, err := promptAgentSelection(&buf, reader, choices)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "no selection provided") {
		t.Errorf("expected 'no selection provided' in error, got: %v", err)
	}
}

func TestPromptSetDefault_Yes(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("y\n"))

	result := promptSetDefault(&buf, reader, "claude")
	if !result {
		t.Error("expected true for 'y' input")
	}
}

func TestPromptSetDefault_No(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	reader := bufio.NewReader(strings.NewReader("n\n"))

	result := promptSetDefault(&buf, reader, "claude")
	if result {
		t.Error("expected false for 'n' input")
	}
}

func TestBuildExecutionEnv_SingleAgent_AutoSelect(t *testing.T) {
	t.Parallel()
	cfg := buildTestCfg(t, `{
		agents: {
			echo: {
				bin: "echo"
				command: "{{.bin}} hello"
			}
		}
	}`)

	flags := &Flags{}
	var buf bytes.Buffer
	r := strings.NewReader("")

	env, err := buildExecutionEnv(cfg, t.TempDir(), "", flags, &buf, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Agent.Name != "echo" {
		t.Errorf("expected agent 'echo', got %q", env.Agent.Name)
	}
}

func TestBuildExecutionEnv_DefaultAgentSet(t *testing.T) {
	t.Parallel()
	cfg := buildTestCfg(t, `{
		settings: {
			default_agent: "copilot"
		}
		agents: {
			claude: {
				bin: "claude"
				command: "{{.bin}}"
			}
			copilot: {
				bin: "gh"
				command: "{{.bin}} copilot"
			}
		}
	}`)

	flags := &Flags{}
	var buf bytes.Buffer
	r := strings.NewReader("")

	env, err := buildExecutionEnv(cfg, t.TempDir(), "", flags, &buf, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Agent.Name != "copilot" {
		t.Errorf("expected agent 'copilot', got %q", env.Agent.Name)
	}
}

func TestBuildExecutionEnv_MultipleAgents_NonTTY(t *testing.T) {
	t.Parallel()
	cfg := buildTestCfg(t, `{
		agents: {
			claude: {
				bin: "claude"
				command: "{{.bin}}"
			}
			copilot: {
				bin: "gh"
				command: "{{.bin}} copilot"
			}
		}
	}`)

	flags := &Flags{}
	var buf bytes.Buffer
	r := strings.NewReader("") // non-TTY: falls back to first agent

	env, err := buildExecutionEnv(cfg, t.TempDir(), "", flags, &buf, r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Agent.Name != "claude" {
		t.Errorf("expected first agent 'claude', got %q", env.Agent.Name)
	}
	if !strings.Contains(buf.String(), "Using agent") {
		t.Errorf("expected non-TTY fallback message, got: %q", buf.String())
	}
}
