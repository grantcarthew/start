package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigAgentList_NoConfig(t *testing.T) {
	// Set up temp directory with no config
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Save and restore working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()

	// Change to temp directory (no local config)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "agent", "list", "--local"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No agents configured") {
		t.Errorf("expected 'No agents configured', got: %s", output)
	}
}

func TestConfigAgentList_WithAgents(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config with an agent
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	agentsContent := `agents: {
	"claude": {
		bin: "claude"
		command: "claude \"{{.prompt}}\""
		description: "Anthropic Claude"
	}
}`
	if err := os.WriteFile(filepath.Join(globalDir, "agents.cue"), []byte(agentsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Save and restore working directory
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "agent", "list"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "claude") {
		t.Errorf("expected output to contain 'claude', got: %s", output)
	}
	if !strings.Contains(output, "Anthropic Claude") {
		t.Errorf("expected output to contain description, got: %s", output)
	}
}

func TestConfigAgentInfo(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config with an agent
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	agentsContent := `agents: {
	"claude": {
		bin: "claude"
		command: "claude --model {{.model}} \"{{.prompt}}\""
		default_model: "sonnet"
		description: "Anthropic Claude"
		models: {
			"sonnet": "claude-sonnet-4-20250514"
			"opus": "claude-opus-4-20250514"
		}
	}
}`
	if err := os.WriteFile(filepath.Join(globalDir, "agents.cue"), []byte(agentsContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "agent", "info", "claude"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "agents/claude") {
		t.Errorf("expected 'agents/claude', got: %s", output)
	}
	if !strings.Contains(output, "Bin:") {
		t.Errorf("expected 'Bin:', got: %s", output)
	}
	if !strings.Contains(output, "Default Model:") {
		t.Errorf("expected 'Default Model:', got: %s", output)
	}
	if !strings.Contains(output, "opus ->") {
		t.Errorf("expected models to include 'opus ->', got: %s", output)
	}
}

func TestConfigAgentInfo_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create empty global config
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "agents.cue"), []byte("agents: {}"), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "agent", "info", "nonexistent"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestConfigAgentAdd_NonInteractive_MissingFlags(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config dir
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	// Provide empty stdin to simulate non-interactive
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"config", "agent", "add"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error when missing required flags in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestConfigAgentAdd_WithFlags(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config dir
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{
		"config", "agent", "add",
		"--name", "gemini",
		"--bin", "gemini",
		"--command", `gemini "{{.prompt}}"`,
		"--description", "Google Gemini",
	})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	agentsPath := filepath.Join(globalDir, "agents.cue")
	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read agents.cue: %v", err)
	}

	if !strings.Contains(string(content), `"gemini"`) {
		t.Errorf("expected agents.cue to contain 'gemini', got: %s", content)
	}
	if !strings.Contains(string(content), `bin:`) {
		t.Errorf("expected agents.cue to contain 'bin:', got: %s", content)
	}
}

func TestConfigAgentRemove(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config with an agent
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	agentsContent := `agents: {
	"claude": {
		bin: "claude"
		command: "claude \"{{.prompt}}\""
	}
	"gemini": {
		bin: "gemini"
		command: "gemini \"{{.prompt}}\""
	}
}`
	if err := os.WriteFile(filepath.Join(globalDir, "agents.cue"), []byte(agentsContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	// Simulate "y" confirmation
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetArgs([]string{"config", "agent", "remove", "gemini"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify gemini was removed
	content, err := os.ReadFile(filepath.Join(globalDir, "agents.cue"))
	if err != nil {
		t.Fatalf("failed to read agents.cue: %v", err)
	}

	if strings.Contains(string(content), `"gemini"`) {
		t.Errorf("expected gemini to be removed, but still present: %s", content)
	}
	if !strings.Contains(string(content), `"claude"`) {
		t.Errorf("expected claude to still be present: %s", content)
	}
}

func TestConfigAgentDefault_Show(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config with settings
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	configContent := `settings: {
	default_agent: "claude"
}`
	if err := os.WriteFile(filepath.Join(globalDir, "settings.cue"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "agent", "default"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Default agent: claude") {
		t.Errorf("expected 'Default agent: claude', got: %s", output)
	}
}

func TestConfigRoleList_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "role", "list", "--local"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No roles configured") {
		t.Errorf("expected 'No roles configured', got: %s", output)
	}
}

func TestConfigContextList_WithContexts(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config with a context
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	contextsContent := `contexts: {
	"project": {
		file: "PROJECT.md"
		description: "Project context"
		required: true
	}
}`
	if err := os.WriteFile(filepath.Join(globalDir, "contexts.cue"), []byte(contextsContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "context", "list"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "project") {
		t.Errorf("expected output to contain 'project', got: %s", output)
	}
	if !strings.Contains(output, "[required]") {
		t.Errorf("expected output to contain '[required]' marker, got: %s", output)
	}
}

func TestConfigContextList_PreservesDefinitionOrder(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config with multiple contexts in specific order
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Define contexts in a specific order: zebra, alpha, middle
	// If sorted alphabetically, order would be: alpha, middle, zebra
	contextsContent := `contexts: {
	"zebra": {
		file: "zebra.md"
		description: "Zebra context (defined first)"
	}
	"alpha": {
		file: "alpha.md"
		description: "Alpha context (defined second)"
	}
	"middle": {
		file: "middle.md"
		description: "Middle context (defined third)"
	}
}`
	if err := os.WriteFile(filepath.Join(globalDir, "contexts.cue"), []byte(contextsContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "context", "list"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()

	// Verify order is preserved (definition order, not alphabetical)
	zebraIdx := strings.Index(output, "zebra")
	alphaIdx := strings.Index(output, "alpha")
	middleIdx := strings.Index(output, "middle")

	if zebraIdx == -1 || alphaIdx == -1 || middleIdx == -1 {
		t.Fatalf("expected all contexts in output, got: %s", output)
	}

	// Definition order: zebra < alpha < middle
	if zebraIdx >= alphaIdx || alphaIdx >= middleIdx {
		t.Errorf("context order not preserved (expected zebra < alpha < middle): zebra=%d, alpha=%d, middle=%d\noutput: %s",
			zebraIdx, alphaIdx, middleIdx, output)
	}
}

func TestConfigTaskList_WithTasks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config with a task
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	tasksContent := `tasks: {
	"review": {
		prompt: "Review this code"
		description: "Code review task"
	}
}`
	if err := os.WriteFile(filepath.Join(globalDir, "tasks.cue"), []byte(tasksContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "task", "list"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "review") {
		t.Errorf("expected output to contain 'review', got: %s", output)
	}
	if !strings.Contains(output, "Code review task") {
		t.Errorf("expected output to contain description, got: %s", output)
	}
}

func TestWriteAgentsFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "agents.cue")

	agents := map[string]AgentConfig{
		"claude": {
			Name:         "claude",
			Bin:          "claude",
			Command:      `claude "{{.prompt}}"`,
			DefaultModel: "sonnet",
			Description:  "Anthropic Claude",
			Models: map[string]string{
				"sonnet": "claude-sonnet-4-20250514",
				"opus":   "claude-opus-4-20250514",
			},
		},
	}

	err := writeAgentsFile(path, agents)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"claude"`) {
		t.Errorf("expected file to contain 'claude', got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "bin:") {
		t.Errorf("expected file to contain 'bin:', got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "models:") {
		t.Errorf("expected file to contain 'models:', got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "Auto-generated") {
		t.Errorf("expected file to contain 'Auto-generated' header, got: %s", contentStr)
	}
}

func TestWriteRolesFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "roles.cue")

	roles := map[string]RoleConfig{
		"go-expert": {
			Name:        "go-expert",
			Description: "Go programming expert",
			File:        "~/.config/start/roles/go-expert.md",
		},
	}

	err := writeRolesFile(path, roles, []string{"go-expert"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"go-expert"`) {
		t.Errorf("expected file to contain 'go-expert', got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "file:") {
		t.Errorf("expected file to contain 'file:', got: %s", contentStr)
	}
}

func TestWriteContextsFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "contexts.cue")

	contexts := map[string]ContextConfig{
		"project": {
			Name:        "project",
			Description: "Project context",
			File:        "PROJECT.md",
			Required:    true,
		},
	}

	err := writeContextsFile(path, contexts, []string{"project"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"project"`) {
		t.Errorf("expected file to contain 'project', got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "required: true") {
		t.Errorf("expected file to contain 'required: true', got: %s", contentStr)
	}
}

func TestWriteTasksFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "tasks.cue")

	tasks := map[string]TaskConfig{
		"review": {
			Name:        "review",
			Description: "Code review",
			Prompt:      "Review this code for bugs",
			Role:        "code-reviewer",
		},
	}

	err := writeTasksFile(path, tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, `"review"`) {
		t.Errorf("expected file to contain 'review', got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "prompt:") {
		t.Errorf("expected file to contain 'prompt:', got: %s", contentStr)
	}
	if !strings.Contains(contentStr, "role:") {
		t.Errorf("expected file to contain 'role:', got: %s", contentStr)
	}
}

func TestTruncatePrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{
		{"short string unchanged", "short", 10, "short"},
		{"long string truncated with ellipsis", "this is a longer string", 10, "this is..."},
		{"newlines replaced with spaces", "with\nnewlines", 20, "with newlines"},
		{"empty string unchanged", "", 10, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncatePrompt(tc.input, tc.max)
			if result != tc.expected {
				t.Errorf("truncatePrompt(%q, %d) = %q, want %q", tc.input, tc.max, result, tc.expected)
			}
		})
	}
}

func TestScopeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		local bool
		want  string
	}{
		{"local scope", true, "local"},
		{"global scope", false, "global"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scopeString(tt.local)
			if got != tt.want {
				t.Errorf("scopeString(%v) = %q, want %q", tt.local, got, tt.want)
			}
		})
	}
}

// Settings command tests

func TestConfigSettingsList_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "settings"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "No settings configured") {
		t.Errorf("expected 'No settings configured', got: %s", output)
	}
}

func TestConfigSettingsList_WithSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	settingsContent := `settings: {
	default_agent: "claude"
	timeout: 120
}`
	if err := os.WriteFile(filepath.Join(globalDir, "settings.cue"), []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "settings"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "default_agent: claude") {
		t.Errorf("expected 'default_agent: claude', got: %s", output)
	}
	if !strings.Contains(output, "timeout: 120") {
		t.Errorf("expected 'timeout: 120', got: %s", output)
	}
}

func TestConfigSettingsShow_SingleKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	settingsContent := `settings: {
	default_agent: "gemini"
}`
	if err := os.WriteFile(filepath.Join(globalDir, "settings.cue"), []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "settings", "default_agent"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "default_agent: gemini") {
		t.Errorf("expected 'default_agent: gemini', got: %s", output)
	}
}

func TestConfigSettingsShow_NotSet(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	settingsContent := `settings: {
	default_agent: "claude"
}`
	if err := os.WriteFile(filepath.Join(globalDir, "settings.cue"), []byte(settingsContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "settings", "shell"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "shell: (not set)") {
		t.Errorf("expected 'shell: (not set)', got: %s", output)
	}
}

func TestConfigSettingsShow_InvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"config", "settings", "invalid_key"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid key")
	}

	if !strings.Contains(err.Error(), "unknown setting") {
		t.Errorf("expected 'unknown setting' error, got: %v", err)
	}
}

func TestConfigSettingsSet(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "settings", "default_agent", "claude"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, `Set default_agent to "claude"`) {
		t.Errorf("expected set confirmation, got: %s", output)
	}

	// Verify file was created
	settingsPath := filepath.Join(tmpDir, "start", "settings.cue")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}

	if !strings.Contains(string(content), `default_agent: "claude"`) {
		t.Errorf("settings file missing default_agent, content: %s", content)
	}
}

func TestConfigSettingsSet_Integer(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "settings", "timeout", "60"})

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file has integer value (no quotes)
	settingsPath := filepath.Join(tmpDir, "start", "settings.cue")
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings file: %v", err)
	}

	if !strings.Contains(string(content), "timeout: 60") {
		t.Errorf("settings file missing timeout as integer, content: %s", content)
	}
	// Make sure it's NOT quoted
	if strings.Contains(string(content), `timeout: "60"`) {
		t.Errorf("timeout should not be quoted, content: %s", content)
	}
}

func TestConfigSettingsSet_InvalidValue(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"config", "settings", "timeout", "not-a-number"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-integer timeout")
	}

	if !strings.Contains(err.Error(), "requires an integer value") {
		t.Errorf("expected integer value error, got: %v", err)
	}
}

func TestHasNonSettingsContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "settings only",
			content: "settings: {\n\tdefault_agent: \"claude\"\n}",
			want:    false,
		},
		{
			name:    "with agents",
			content: "settings: {\n\tdefault_agent: \"claude\"\n}\nagents: {\n\tclaude: {}\n}",
			want:    true,
		},
		{
			name:    "with roles",
			content: "roles: {\n\tdev: {}\n}",
			want:    true,
		},
		{
			name:    "with contexts",
			content: "contexts: {\n\tenv: {}\n}",
			want:    true,
		},
		{
			name:    "with tasks",
			content: "tasks: {\n\tbuild: {}\n}",
			want:    true,
		},
		{
			name:    "empty file",
			content: "",
			want:    false,
		},
		{
			name:    "comments only",
			content: "// Auto-generated by start config\n// Settings file\n",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasNonSettingsContent(tt.content)
			if got != tt.want {
				t.Errorf("hasNonSettingsContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigSettingsSet_RefusesOverwriteNonSettings(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create global config directory with mixed content
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a settings.cue file that also contains agents
	mixedContent := `settings: {
	default_agent: "claude"
}

agents: {
	claude: {
		bin: "claude"
		command: "{{.bin}}"
	}
}
`
	settingsPath := filepath.Join(globalDir, "settings.cue")
	if err := os.WriteFile(settingsPath, []byte(mixedContent), 0644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCmd()
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"config", "settings", "default_role", "dev"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected error when trying to overwrite file with non-settings content")
	}

	if !strings.Contains(err.Error(), "non-settings content") {
		t.Errorf("expected non-settings content error, got: %v", err)
	}

	// Verify original file is unchanged
	content, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "agents:") {
		t.Error("original file content should be preserved")
	}
}

// Tests for prompt helper functions

func TestPromptTags_KeepCurrent(t *testing.T) {
	current := []string{"tag1", "tag2"}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("\n") // Just press Enter

	result, err := promptTags(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 || result[0] != "tag1" || result[1] != "tag2" {
		t.Errorf("expected current tags to be preserved, got: %v", result)
	}

	output := stdout.String()
	if !strings.Contains(output, "tag1, tag2") {
		t.Errorf("expected current tags in output, got: %s", output)
	}
}

func TestPromptTags_Clear(t *testing.T) {
	current := []string{"tag1", "tag2"}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("-\n")

	result, err := promptTags(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty tags after clear, got: %v", result)
	}
}

func TestPromptTags_Replace(t *testing.T) {
	current := []string{"old1", "old2"}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("new1, new2, new3\n")

	result, err := promptTags(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 || result[0] != "new1" || result[1] != "new2" || result[2] != "new3" {
		t.Errorf("expected new tags, got: %v", result)
	}
}

func TestPromptTags_Empty(t *testing.T) {
	var current []string
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("first, second\n")

	result, err := promptTags(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 || result[0] != "first" || result[1] != "second" {
		t.Errorf("expected new tags, got: %v", result)
	}

	output := stdout.String()
	if !strings.Contains(output, "(none)") {
		t.Errorf("expected '(none)' for empty current tags, got: %s", output)
	}
}

func TestPromptDefaultModel_NoModels(t *testing.T) {
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("gpt-4\n")

	result, err := promptDefaultModel(stdout, stdin, "gpt-3", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "gpt-4" {
		t.Errorf("expected 'gpt-4', got: %s", result)
	}

	output := stdout.String()
	if !strings.Contains(output, "Default model") {
		t.Errorf("expected promptString fallback, got: %s", output)
	}
}

func TestPromptDefaultModel_NoModelsKeepCurrent(t *testing.T) {
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("\n")

	result, err := promptDefaultModel(stdout, stdin, "gpt-3", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "gpt-3" {
		t.Errorf("expected 'gpt-3', got: %s", result)
	}
}

func TestPromptDefaultModel_SelectByNumber(t *testing.T) {
	models := map[string]string{
		"haiku":  "claude-3-5-haiku-20241022",
		"sonnet": "claude-3-7-sonnet-20250219",
		"opus":   "claude-opus-4-20250514",
	}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("3\n")

	result, err := promptDefaultModel(stdout, stdin, "sonnet", models)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sorted: haiku=1, opus=2, sonnet=3
	if result != "sonnet" {
		t.Errorf("expected 'sonnet', got: %s", result)
	}

	output := stdout.String()
	if !strings.Contains(output, "1. haiku") {
		t.Errorf("expected numbered list, got: %s", output)
	}
	if !strings.Contains(output, "(current)") {
		t.Errorf("expected current marker, got: %s", output)
	}
}

func TestPromptDefaultModel_SelectByAlias(t *testing.T) {
	models := map[string]string{
		"haiku":  "claude-3-5-haiku-20241022",
		"sonnet": "claude-3-7-sonnet-20250219",
	}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("haiku\n")

	result, err := promptDefaultModel(stdout, stdin, "sonnet", models)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "haiku" {
		t.Errorf("expected 'haiku', got: %s", result)
	}
}

func TestPromptDefaultModel_SelectByAliasCaseInsensitive(t *testing.T) {
	models := map[string]string{
		"haiku":  "claude-3-5-haiku-20241022",
		"sonnet": "claude-3-7-sonnet-20250219",
	}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("HAIKU\n")

	result, err := promptDefaultModel(stdout, stdin, "sonnet", models)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "haiku" {
		t.Errorf("expected 'haiku', got: %s", result)
	}
}

func TestPromptDefaultModel_EnterKeepsCurrent(t *testing.T) {
	models := map[string]string{
		"haiku":  "claude-3-5-haiku-20241022",
		"sonnet": "claude-3-7-sonnet-20250219",
	}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("\n")

	result, err := promptDefaultModel(stdout, stdin, "sonnet", models)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "sonnet" {
		t.Errorf("expected 'sonnet', got: %s", result)
	}
}

func TestPromptDefaultModel_InvalidNumber(t *testing.T) {
	models := map[string]string{
		"haiku":  "claude-3-5-haiku-20241022",
		"sonnet": "claude-3-7-sonnet-20250219",
	}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("5\n")

	_, err := promptDefaultModel(stdout, stdin, "sonnet", models)
	if err == nil {
		t.Fatal("expected error for invalid number")
	}

	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' error, got: %v", err)
	}
}

func TestPromptDefaultModel_InvalidAlias(t *testing.T) {
	models := map[string]string{
		"haiku":  "claude-3-5-haiku-20241022",
		"sonnet": "claude-3-7-sonnet-20250219",
	}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("turbo\n")

	_, err := promptDefaultModel(stdout, stdin, "sonnet", models)
	if err == nil {
		t.Fatal("expected error for invalid alias")
	}

	if !strings.Contains(err.Error(), "not a known model alias") {
		t.Errorf("expected 'not a known model alias' error, got: %v", err)
	}
}

func TestPromptDefaultModel_NoCurrent(t *testing.T) {
	models := map[string]string{
		"haiku":  "claude-3-5-haiku-20241022",
		"sonnet": "claude-3-7-sonnet-20250219",
	}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("1\n")

	result, err := promptDefaultModel(stdout, stdin, "", models)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "haiku" {
		t.Errorf("expected 'haiku', got: %s", result)
	}

	output := stdout.String()
	if strings.Contains(output, "(current)") {
		t.Errorf("expected no current marker when no default set, got: %s", output)
	}
	if strings.Contains(output, "Enter to keep") {
		t.Errorf("expected no keep prompt when no default set, got: %s", output)
	}
}

func TestPromptModels_Keep(t *testing.T) {
	current := map[string]string{"fast": "gpt-4", "smart": "gpt-4-turbo"}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("k\n")

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 || result["fast"] != "gpt-4" || result["smart"] != "gpt-4-turbo" {
		t.Errorf("expected current models preserved, got: %v", result)
	}
}

func TestPromptModels_KeepDefault(t *testing.T) {
	current := map[string]string{"fast": "gpt-4"}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("\n") // Just Enter defaults to keep

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result["fast"] != "gpt-4" {
		t.Errorf("expected current models preserved, got: %v", result)
	}
}

func TestPromptModels_Clear(t *testing.T) {
	current := map[string]string{"fast": "gpt-4", "smart": "gpt-4-turbo"}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("c\n")

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty models after clear, got: %v", result)
	}
}

func TestPromptModels_Edit_KeepExisting(t *testing.T) {
	current := map[string]string{"fast": "gpt-4"}
	stdout := &bytes.Buffer{}
	// Edit mode, keep fast, don't add new
	stdin := strings.NewReader("e\n\n\n")

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result["fast"] != "gpt-4" {
		t.Errorf("expected model kept, got: %v", result)
	}
}

func TestPromptModels_Edit_UpdateExisting(t *testing.T) {
	current := map[string]string{"fast": "gpt-4"}
	stdout := &bytes.Buffer{}
	// Edit mode, update fast to gpt-4-turbo, don't add new
	stdin := strings.NewReader("e\ngpt-4-turbo\n\n")

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result["fast"] != "gpt-4-turbo" {
		t.Errorf("expected model updated, got: %v", result)
	}
}

func TestPromptModels_Edit_DeleteExisting(t *testing.T) {
	current := map[string]string{"fast": "gpt-4", "slow": "gpt-3"}
	stdout := &bytes.Buffer{}
	// Edit mode, keep fast, delete slow, don't add new
	stdin := strings.NewReader("e\n\n-\n\n")

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result["fast"] != "gpt-4" {
		t.Errorf("expected only fast model kept, got: %v", result)
	}
	if _, exists := result["slow"]; exists {
		t.Errorf("expected slow model deleted, got: %v", result)
	}
}

func TestPromptModels_Edit_AddNew(t *testing.T) {
	current := map[string]string{"fast": "gpt-4"}
	stdout := &bytes.Buffer{}
	// Edit mode, keep fast, add new model
	stdin := strings.NewReader("e\n\nreasoning=o1\n\n")

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 models, got: %v", result)
	}
	if result["fast"] != "gpt-4" {
		t.Errorf("expected fast model preserved, got: %v", result)
	}
	if result["reasoning"] != "o1" {
		t.Errorf("expected reasoning model added, got: %v", result)
	}
}

func TestPromptModels_Empty(t *testing.T) {
	var current map[string]string
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("e\nnew=model-id\n\n")

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 1 || result["new"] != "model-id" {
		t.Errorf("expected new model added, got: %v", result)
	}

	output := stdout.String()
	if !strings.Contains(output, "(none)") {
		t.Errorf("expected '(none)' for empty current models, got: %s", output)
	}
}

func TestPromptModels_InvalidChoice(t *testing.T) {
	current := map[string]string{"fast": "gpt-4"}
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("x\n")

	_, err := promptModels(stdout, stdin, current)
	if err == nil {
		t.Fatal("expected error for invalid choice")
	}
	if !strings.Contains(err.Error(), "invalid choice") {
		t.Errorf("expected 'invalid choice' error, got: %v", err)
	}
}

func TestPromptModels_Edit_InvalidFormat(t *testing.T) {
	var current map[string]string
	stdout := &bytes.Buffer{}
	// Try invalid format, then valid, then finish
	stdin := strings.NewReader("e\ninvalid-no-equals\nvalid=model\n\n")

	result, err := promptModels(stdout, stdin, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Invalid format") {
		t.Errorf("expected invalid format message, got: %s", output)
	}

	if len(result) != 1 || result["valid"] != "model" {
		t.Errorf("expected valid model added despite earlier invalid input, got: %v", result)
	}
}

func TestPromptText_MultiLine(t *testing.T) {
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("line one\nline two\nline three\n\n")

	result, err := promptText(stdout, stdin, "Prompt text", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "line one\nline two\nline three"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	output := stdout.String()
	if !strings.Contains(output, "Prompt text") {
		t.Errorf("expected label in output, got: %s", output)
	}
	if !strings.Contains(output, "blank line to finish") {
		t.Errorf("expected instructions in output, got: %s", output)
	}
}

func TestPromptText_SingleLine(t *testing.T) {
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("just one line\n\n")

	result, err := promptText(stdout, stdin, "Prompt text", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "just one line" {
		t.Errorf("expected %q, got %q", "just one line", result)
	}
}

func TestPromptText_EmptyOpensEditor_FallsBackToDefault(t *testing.T) {
	// When first line is empty, promptText tries to open $EDITOR.
	// With an invalid editor, it falls back to returning defaultVal.
	t.Setenv("EDITOR", "/nonexistent-editor")
	t.Setenv("VISUAL", "")

	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("\n")

	result, err := promptText(stdout, stdin, "Prompt text", "fallback value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "fallback value" {
		t.Errorf("expected %q, got %q", "fallback value", result)
	}
}

func TestPromptText_EmptyNoDefault(t *testing.T) {
	// Empty input with no default returns empty string
	t.Setenv("EDITOR", "/nonexistent-editor")
	t.Setenv("VISUAL", "")

	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("\n")

	result, err := promptText(stdout, stdin, "Prompt text", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestPromptText_ShowsMultiLineDefault(t *testing.T) {
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("new text\n\n")

	_, err := promptText(stdout, stdin, "Prompt text", "line 1\nline 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Current value:") {
		t.Errorf("expected 'Current value:' for multi-line default, got: %s", output)
	}
	if !strings.Contains(output, "line 1\nline 2") {
		t.Errorf("expected default value shown, got: %s", output)
	}
}

func TestPromptText_ShowsSingleLineDefault(t *testing.T) {
	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("new text\n\n")

	_, err := promptText(stdout, stdin, "Prompt text", "short default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "short default") {
		t.Errorf("expected single-line default in brackets, got: %s", output)
	}
	// Should NOT show "Current value:" for single-line defaults
	if strings.Contains(output, "Current value:") {
		t.Errorf("single-line default should not show 'Current value:', got: %s", output)
	}
}
