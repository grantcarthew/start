package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests for config commands that test the full workflow.

func TestConfigAgent_FullWorkflow(t *testing.T) {
	// Setup isolated environment
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

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Reset package-level flags before each test
	resetAgentFlags := func() {
		agentName = ""
		agentBin = ""
		agentCommand = ""
		agentDefaultModel = ""
		agentDescription = ""
		agentModels = nil
		agentTags = nil
		configLocal = false
	}

	t.Run("add agent via flags", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "agent", "add",
			"--name", "claude",
			"--bin", "claude",
			"--command", `claude --model {{.model}} "{{.prompt}}"`,
			"--default-model", "sonnet",
			"--description", "Anthropic Claude",
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add failed: %v", err)
		}

		// Verify file exists
		agentsPath := filepath.Join(globalDir, "agents.cue")
		if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
			t.Fatal("agents.cue was not created")
		}

		content, _ := os.ReadFile(agentsPath)
		if !strings.Contains(string(content), `"claude"`) {
			t.Errorf("agents.cue missing claude: %s", content)
		}
	})

	t.Run("list shows added agent", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "claude") {
			t.Errorf("list output missing claude: %s", output)
		}
		if !strings.Contains(output, "Anthropic Claude") {
			t.Errorf("list output missing description: %s", output)
		}
	})

	t.Run("show displays agent details", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "show", "claude"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Agent: claude") {
			t.Errorf("show output missing agent name: %s", output)
		}
		if !strings.Contains(output, "Default Model: sonnet") {
			t.Errorf("show output missing default model: %s", output)
		}
	})

	t.Run("add second agent", func(t *testing.T) {
		resetAgentFlags()

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
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add gemini failed: %v", err)
		}
	})

	t.Run("list shows both agents", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "claude") {
			t.Errorf("list output missing claude: %s", output)
		}
		if !strings.Contains(output, "gemini") {
			t.Errorf("list output missing gemini: %s", output)
		}
	})

	t.Run("set default agent", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "default", "claude"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("default failed: %v", err)
		}

		// Verify settings.cue was created with default_agent
		configPath := filepath.Join(globalDir, "settings.cue")
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read settings.cue: %v", err)
		}
		if !strings.Contains(string(content), `default_agent:`) {
			t.Errorf("settings.cue missing default_agent: %s", content)
		}
	})

	t.Run("show default agent", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "default"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("default show failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Default agent: claude") {
			t.Errorf("expected 'Default agent: claude', got: %s", output)
		}
	})

	t.Run("remove agent with confirmation", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader("y\n"))
		cmd.SetArgs([]string{"config", "agent", "remove", "gemini"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("remove failed: %v", err)
		}

		// Verify gemini is removed
		content, _ := os.ReadFile(filepath.Join(globalDir, "agents.cue"))
		if strings.Contains(string(content), `"gemini"`) {
			t.Errorf("gemini should be removed: %s", content)
		}
		if !strings.Contains(string(content), `"claude"`) {
			t.Errorf("claude should still exist: %s", content)
		}
	})

	t.Run("add duplicate fails", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "agent", "add",
			"--name", "claude",
			"--bin", "claude",
			"--command", `claude "{{.prompt}}"`,
		})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for duplicate agent")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' error, got: %v", err)
		}
	})
}

func TestConfigRole_FullWorkflow(t *testing.T) {
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

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	resetRoleFlags := func() {
		roleName = ""
		roleDescription = ""
		roleFile = ""
		roleCommand = ""
		rolePrompt = ""
		roleTags = nil
		configLocal = false
	}

	t.Run("add role with file", func(t *testing.T) {
		resetRoleFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "role", "add",
			"--name", "go-expert",
			"--file", "~/.config/start/roles/go-expert.md",
			"--description", "Go programming expert",
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(globalDir, "roles.cue"))
		if !strings.Contains(string(content), `"go-expert"`) {
			t.Errorf("roles.cue missing go-expert: %s", content)
		}
	})

	t.Run("add role with prompt", func(t *testing.T) {
		resetRoleFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "role", "add",
			"--name", "reviewer",
			"--prompt", "You are a code reviewer. Review code for bugs and style issues.",
			"--description", "Code review expert",
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add failed: %v", err)
		}
	})

	t.Run("list shows roles", func(t *testing.T) {
		resetRoleFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "role", "list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "go-expert") {
			t.Errorf("list missing go-expert: %s", output)
		}
		if !strings.Contains(output, "reviewer") {
			t.Errorf("list missing reviewer: %s", output)
		}
	})

	t.Run("show role details", func(t *testing.T) {
		resetRoleFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "role", "show", "reviewer"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Role: reviewer") {
			t.Errorf("show missing role name: %s", output)
		}
		if !strings.Contains(output, "Prompt:") {
			t.Errorf("show missing prompt: %s", output)
		}
	})
}

func TestConfigContext_FullWorkflow(t *testing.T) {
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

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	resetContextFlags := func() {
		contextName = ""
		contextDescription = ""
		contextFile = ""
		contextCommand = ""
		contextPrompt = ""
		contextRequired = false
		contextDefault = false
		contextTags = nil
		configLocal = false
	}

	t.Run("add required context", func(t *testing.T) {
		resetContextFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "context", "add",
			"--name", "project",
			"--file", "PROJECT.md",
			"--description", "Project documentation",
			"--required",
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(globalDir, "contexts.cue"))
		if !strings.Contains(string(content), `"project"`) {
			t.Errorf("contexts.cue missing project: %s", content)
		}
		if !strings.Contains(string(content), "required: true") {
			t.Errorf("contexts.cue missing required flag: %s", content)
		}
	})

	t.Run("add default context", func(t *testing.T) {
		resetContextFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "context", "add",
			"--name", "readme",
			"--file", "README.md",
			"--default",
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add failed: %v", err)
		}
	})

	t.Run("list shows contexts with markers", func(t *testing.T) {
		resetContextFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "context", "list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "[R]") {
			t.Errorf("list missing [R] marker: %s", output)
		}
		if !strings.Contains(output, "[D]") {
			t.Errorf("list missing [D] marker: %s", output)
		}
	})
}

func TestConfigTask_FullWorkflow(t *testing.T) {
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

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	resetTaskFlags := func() {
		taskName = ""
		taskDescription = ""
		taskFile = ""
		taskCommand = ""
		taskPrompt = ""
		taskRole = ""
		taskTags = nil
		configLocal = false
	}

	t.Run("add task with prompt and role", func(t *testing.T) {
		resetTaskFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "task", "add",
			"--name", "review",
			"--prompt", "Review this code for bugs and improvements",
			"--role", "code-reviewer",
			"--description", "Code review task",
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(globalDir, "tasks.cue"))
		if !strings.Contains(string(content), `"review"`) {
			t.Errorf("tasks.cue missing review: %s", content)
		}
		if !strings.Contains(string(content), `role:`) {
			t.Errorf("tasks.cue missing role: %s", content)
		}
	})

	t.Run("list shows tasks", func(t *testing.T) {
		resetTaskFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "review") {
			t.Errorf("list missing review: %s", output)
		}
	})

	t.Run("show task details", func(t *testing.T) {
		resetTaskFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "show", "review"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("show failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "Task: review") {
			t.Errorf("show missing task name: %s", output)
		}
		if !strings.Contains(output, "Role: code-reviewer") {
			t.Errorf("show missing role: %s", output)
		}
	})
}

func TestConfigLocal_Isolation(t *testing.T) {
	// Test that --local flag properly isolates local and global configs
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()

	// Create a project directory
	projectDir := filepath.Join(tmpDir, "myproject")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}

	// Create global config
	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	resetAgentFlags := func() {
		agentName = ""
		agentBin = ""
		agentCommand = ""
		agentDefaultModel = ""
		agentDescription = ""
		agentModels = nil
		agentTags = nil
		configLocal = false
	}

	t.Run("add global agent", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "agent", "add",
			"--name", "global-agent",
			"--bin", "global",
			"--command", `global "{{.prompt}}"`,
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add global failed: %v", err)
		}
	})

	t.Run("add local agent", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs([]string{
			"config", "agent", "add",
			"--local",
			"--name", "local-agent",
			"--bin", "local",
			"--command", `local "{{.prompt}}"`,
		})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("add local failed: %v", err)
		}

		// Verify local config was created
		localPath := filepath.Join(projectDir, ".start", "agents.cue")
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			t.Fatal("local agents.cue was not created")
		}
	})

	t.Run("list shows both in merged view", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "global-agent") {
			t.Errorf("list missing global-agent: %s", output)
		}
		if !strings.Contains(output, "local-agent") {
			t.Errorf("list missing local-agent: %s", output)
		}
	})

	t.Run("list --local shows only local", func(t *testing.T) {
		resetAgentFlags()

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "list", "--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list failed: %v", err)
		}

		output := stdout.String()
		if strings.Contains(output, "global-agent") {
			t.Errorf("list --local should not show global-agent: %s", output)
		}
		if !strings.Contains(output, "local-agent") {
			t.Errorf("list --local missing local-agent: %s", output)
		}
	})
}
