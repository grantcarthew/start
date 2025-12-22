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
	if !strings.Contains(output, "Agent: claude") {
		t.Errorf("expected 'Agent: claude', got: %s", output)
	}
	if !strings.Contains(output, "Bin: claude") {
		t.Errorf("expected 'Bin: claude', got: %s", output)
	}
	if !strings.Contains(output, "Default Model: sonnet") {
		t.Errorf("expected 'Default Model: sonnet', got: %s", output)
	}
	if !strings.Contains(output, "opus:") {
		t.Errorf("expected models to include 'opus:', got: %s", output)
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

	// Reset package-level flags
	agentName = ""
	agentBin = ""
	agentCommand = ""
	agentDefaultModel = ""
	agentDescription = ""
	agentModels = nil
	agentTags = nil

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
	if !strings.Contains(output, "[R]") {
		t.Errorf("expected output to contain '[R]' for required, got: %s", output)
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

	err := writeRolesFile(path, roles)
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

	err := writeContextsFile(path, contexts)
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
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a longer string", 10, "this is..."},
		{"with\nnewlines", 20, "with newlines"},
		{"", 10, ""},
	}

	for _, tc := range tests {
		result := truncatePrompt(tc.input, tc.max)
		if result != tc.expected {
			t.Errorf("truncatePrompt(%q, %d) = %q, want %q", tc.input, tc.max, result, tc.expected)
		}
	}
}

func TestScopeString(t *testing.T) {
	if scopeString(true) != "local" {
		t.Errorf("scopeString(true) = %q, want 'local'", scopeString(true))
	}
	if scopeString(false) != "global" {
		t.Errorf("scopeString(false) = %q, want 'global'", scopeString(false))
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
