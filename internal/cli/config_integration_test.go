package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests for config commands that test the full workflow.
//
// Note: Tests below use os.Chdir (process-global state). Do not add t.Parallel()
// to any test that calls os.Chdir — it will cause data races on the working directory.

func TestConfigAgent_FullWorkflow(t *testing.T) {
	// Setup isolated environment
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	chdir(t, tmpDir)

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("add agent via flags", func(t *testing.T) {

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

	t.Run("info displays agent details", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "info", "claude"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("info failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "agents/claude") {
			t.Errorf("info output missing agent name: %s", output)
		}
		if !strings.Contains(output, "Default Model: sonnet") {
			t.Errorf("info output missing default model: %s", output)
		}
	})

	t.Run("add second agent", func(t *testing.T) {
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

	t.Run("unset default agent", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "default", "--unset"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unset failed: %v", err)
		}

		// Verify settings.cue no longer contains default_agent
		configPath := filepath.Join(globalDir, "settings.cue")
		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read settings.cue: %v", err)
		}
		if strings.Contains(string(content), "default_agent") {
			t.Errorf("settings.cue should not contain default_agent after unset: %s", content)
		}
	})

	t.Run("show after unset agent", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "default"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("default show failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "No default agent set.") {
			t.Errorf("expected 'No default agent set.', got: %s", output)
		}
	})

	t.Run("unset error with agent name", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "default", "--unset", "claude"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error when using --unset with a name")
		}
		if !strings.Contains(err.Error(), "cannot use --unset") {
			t.Errorf("expected 'cannot use --unset' error, got: %v", err)
		}
	})

	t.Run("remove agent with --yes", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "remove", "gemini", "--yes"})

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

	chdir(t, tmpDir)

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("add role with file", func(t *testing.T) {
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

	t.Run("info role details", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "role", "info", "reviewer"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("info failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "roles/reviewer") {
			t.Errorf("info missing role name: %s", output)
		}
		if !strings.Contains(output, "Prompt:") {
			t.Errorf("info missing prompt: %s", output)
		}
	})

}

func TestConfigContext_FullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	chdir(t, tmpDir)

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("add required context", func(t *testing.T) {
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
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "context", "list"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("list failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "[required]") {
			t.Errorf("list missing [required] marker: %s", output)
		}
		if !strings.Contains(output, "[default]") {
			t.Errorf("list missing [default] marker: %s", output)
		}
	})
}

func TestConfigTask_FullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	chdir(t, tmpDir)

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("add task with prompt and role", func(t *testing.T) {
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

	t.Run("info task details", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "info", "review"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("info failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "tasks/review") {
			t.Errorf("info missing task name: %s", output)
		}
		if !strings.Contains(output, "Role: code-reviewer") {
			t.Errorf("info missing role: %s", output)
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

	t.Run("add global agent", func(t *testing.T) {
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

func TestConfigTask_SubstringResolution(t *testing.T) {
	// Setup isolated environment
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	chdir(t, tmpDir)

	globalDir := filepath.Join(tmpDir, "start")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Add tasks with namespace-style names
	for _, args := range [][]string{
		{"config", "task", "add", "--name", "cwd/dotai/create-role", "--prompt", "Create a role"},
		{"config", "task", "add", "--name", "confluence/read-doc", "--prompt", "Read a doc"},
		{"config", "task", "add", "--name", "golang/review/architecture", "--prompt", "Review arch"},
		{"config", "task", "add", "--name", "golang/review/code", "--prompt", "Review code"},
	} {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("add task %v failed: %v", args, err)
		}
	}

	t.Run("info with unique substring", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "info", "create-role"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("info with substring failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "cwd/dotai/create-role") {
			t.Errorf("expected resolved name in output: %s", output)
		}
	})

	t.Run("info with exact match still works", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "info", "confluence/read-doc"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("info with exact name failed: %v", err)
		}

		output := stdout.String()
		if !strings.Contains(output, "confluence/read-doc") {
			t.Errorf("expected exact name in output: %s", output)
		}
	})

	t.Run("info with ambiguous substring errors", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "info", "review"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for ambiguous match")
		}
		if !strings.Contains(err.Error(), "ambiguous") {
			t.Errorf("expected 'ambiguous' error, got: %v", err)
		}
	})

	t.Run("info with no match errors", func(t *testing.T) {
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "info", "nonexistent"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for no match")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})

	t.Run("edit with substring via flags", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "edit", "read-doc", "--description", "Updated description"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("edit with substring failed: %v", err)
		}

		// Verify the update was applied to the correct task
		cmd2 := NewRootCmd()
		stdout2 := &bytes.Buffer{}
		cmd2.SetOut(stdout2)
		cmd2.SetErr(&bytes.Buffer{})
		cmd2.SetArgs([]string{"config", "task", "info", "confluence/read-doc"})
		if err := cmd2.Execute(); err != nil {
			t.Fatalf("info after edit failed: %v", err)
		}
		output := stdout2.String()
		if !strings.Contains(output, "Updated description") {
			t.Errorf("expected updated description in output: %s", output)
		}
	})

	t.Run("remove with substring", func(t *testing.T) {
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "remove", "create-role", "--yes"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("remove with substring failed: %v", err)
		}

		// Verify it was removed
		content, _ := os.ReadFile(filepath.Join(globalDir, "tasks.cue"))
		if strings.Contains(string(content), "create-role") {
			t.Errorf("create-role should be removed: %s", content)
		}
	})

}

func TestConfigRemove_MultipleArgs(t *testing.T) {
	t.Run("task remove ambiguous query with --yes expands all", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)
		chdir(t, tmpDir)

		globalDir := filepath.Join(tmpDir, "start")
		if err := os.MkdirAll(globalDir, 0755); err != nil {
			t.Fatal(err)
		}

		for _, args := range [][]string{
			{"config", "task", "add", "--name", "golang/review/architecture", "--prompt", "Arch"},
			{"config", "task", "add", "--name", "golang/review/code", "--prompt", "Code"},
			{"config", "task", "add", "--name", "golang/review/security", "--prompt", "Security"},
			{"config", "task", "add", "--name", "confluence/read-doc", "--prompt", "Read"},
		} {
			cmd := NewRootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("add %v failed: %v", args, err)
			}
		}

		// Ambiguous query with --yes should remove all three golang/review/* tasks
		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "remove", "golang/review", "--yes"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("ambiguous remove with --yes failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(globalDir, "tasks.cue"))
		s := string(content)
		if strings.Contains(s, "golang/review/architecture") {
			t.Errorf("architecture should be removed: %s", s)
		}
		if strings.Contains(s, "golang/review/code") {
			t.Errorf("code should be removed: %s", s)
		}
		if strings.Contains(s, "golang/review/security") {
			t.Errorf("security should be removed: %s", s)
		}
		if !strings.Contains(s, "confluence/read-doc") {
			t.Errorf("read-doc should still exist: %s", s)
		}
	})

	t.Run("agent remove multiple with --yes", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)
		chdir(t, tmpDir)

		globalDir := filepath.Join(tmpDir, "start")
		if err := os.MkdirAll(globalDir, 0755); err != nil {
			t.Fatal(err)
		}

		for _, args := range [][]string{
			{"config", "agent", "add", "--name", "alpha", "--bin", "alpha", "--command", `alpha "{{.prompt}}"`},
			{"config", "agent", "add", "--name", "beta", "--bin", "beta", "--command", `beta "{{.prompt}}"`},
			{"config", "agent", "add", "--name", "gamma", "--bin", "gamma", "--command", `gamma "{{.prompt}}"`},
		} {
			cmd := NewRootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("add %v failed: %v", args, err)
			}
		}

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "agent", "remove", "alpha", "beta", "--yes"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("multi-remove failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(globalDir, "agents.cue"))
		s := string(content)
		if strings.Contains(s, `"alpha"`) {
			t.Errorf("alpha should be removed: %s", s)
		}
		if strings.Contains(s, `"beta"`) {
			t.Errorf("beta should be removed: %s", s)
		}
		if !strings.Contains(s, `"gamma"`) {
			t.Errorf("gamma should still exist: %s", s)
		}

		out := stdout.String()
		if !strings.Contains(out, "alpha") {
			t.Errorf("output should mention alpha: %s", out)
		}
		if !strings.Contains(out, "beta") {
			t.Errorf("output should mention beta: %s", out)
		}
	})

	t.Run("context remove multiple with --yes", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)
		chdir(t, tmpDir)

		globalDir := filepath.Join(tmpDir, "start")
		if err := os.MkdirAll(globalDir, 0755); err != nil {
			t.Fatal(err)
		}

		for _, args := range [][]string{
			{"config", "context", "add", "--name", "project/alpha", "--prompt", "Alpha"},
			{"config", "context", "add", "--name", "project/beta", "--prompt", "Beta"},
			{"config", "context", "add", "--name", "project/gamma", "--prompt", "Gamma"},
		} {
			cmd := NewRootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("add %v failed: %v", args, err)
			}
		}

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "context", "remove", "project/alpha", "project/beta", "--yes"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("multi-remove failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(globalDir, "contexts.cue"))
		s := string(content)
		if strings.Contains(s, "project/alpha") {
			t.Errorf("project/alpha should be removed: %s", s)
		}
		if strings.Contains(s, "project/beta") {
			t.Errorf("project/beta should be removed: %s", s)
		}
		if !strings.Contains(s, "project/gamma") {
			t.Errorf("project/gamma should still exist: %s", s)
		}

		out := stdout.String()
		if !strings.Contains(out, "alpha") {
			t.Errorf("output should mention alpha: %s", out)
		}
		if !strings.Contains(out, "beta") {
			t.Errorf("output should mention beta: %s", out)
		}
	})

	t.Run("role remove multiple with --yes", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)
		chdir(t, tmpDir)

		globalDir := filepath.Join(tmpDir, "start")
		if err := os.MkdirAll(globalDir, 0755); err != nil {
			t.Fatal(err)
		}

		for _, args := range [][]string{
			{"config", "role", "add", "--name", "project/alpha", "--prompt", "Alpha"},
			{"config", "role", "add", "--name", "project/beta", "--prompt", "Beta"},
			{"config", "role", "add", "--name", "project/gamma", "--prompt", "Gamma"},
		} {
			cmd := NewRootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("add %v failed: %v", args, err)
			}
		}

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "role", "remove", "project/alpha", "project/beta", "--yes"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("multi-remove failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(globalDir, "roles.cue"))
		s := string(content)
		if strings.Contains(s, "project/alpha") {
			t.Errorf("project/alpha should be removed: %s", s)
		}
		if strings.Contains(s, "project/beta") {
			t.Errorf("project/beta should be removed: %s", s)
		}
		if !strings.Contains(s, "project/gamma") {
			t.Errorf("project/gamma should still exist: %s", s)
		}

		out := stdout.String()
		if !strings.Contains(out, "alpha") {
			t.Errorf("output should mention alpha: %s", out)
		}
		if !strings.Contains(out, "beta") {
			t.Errorf("output should mention beta: %s", out)
		}
	})

	t.Run("task remove multiple by substring with --yes", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)
		chdir(t, tmpDir)

		globalDir := filepath.Join(tmpDir, "start")
		if err := os.MkdirAll(globalDir, 0755); err != nil {
			t.Fatal(err)
		}

		for _, args := range [][]string{
			{"config", "task", "add", "--name", "confluence/read-doc", "--prompt", "Read a doc"},
			{"config", "task", "add", "--name", "golang/review/architecture", "--prompt", "Review arch"},
			{"config", "task", "add", "--name", "golang/review/code", "--prompt", "Review code"},
		} {
			cmd := NewRootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("add %v failed: %v", args, err)
			}
		}

		cmd := NewRootCmd()
		stdout := &bytes.Buffer{}
		cmd.SetOut(stdout)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "remove", "read-doc", "architecture", "--yes"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("multi-remove failed: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(globalDir, "tasks.cue"))
		s := string(content)
		if strings.Contains(s, "read-doc") {
			t.Errorf("read-doc should be removed: %s", s)
		}
		if strings.Contains(s, "architecture") {
			t.Errorf("architecture should be removed: %s", s)
		}
		if !strings.Contains(s, "golang/review/code") {
			t.Errorf("golang/review/code should still exist: %s", s)
		}

		out := stdout.String()
		if !strings.Contains(out, "read-doc") {
			t.Errorf("output should mention read-doc: %s", out)
		}
		if !strings.Contains(out, "architecture") {
			t.Errorf("output should mention architecture: %s", out)
		}
	})

	t.Run("task remove ambiguous query without --yes errors", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)
		chdir(t, tmpDir)

		globalDir := filepath.Join(tmpDir, "start")
		if err := os.MkdirAll(globalDir, 0755); err != nil {
			t.Fatal(err)
		}

		for _, args := range [][]string{
			{"config", "task", "add", "--name", "golang/review/architecture", "--prompt", "Arch"},
			{"config", "task", "add", "--name", "golang/review/code", "--prompt", "Code"},
			{"config", "task", "add", "--name", "confluence/read-doc", "--prompt", "Read"},
		} {
			cmd := NewRootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetIn(strings.NewReader(""))
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("add %v failed: %v", args, err)
			}
		}

		// Ambiguous arg alongside another arg without --yes must error
		cmd := NewRootCmd()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"config", "task", "remove", "golang/review", "confluence/read-doc"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for ambiguous multi-arg remove without --yes")
		}
		if !strings.Contains(err.Error(), "--yes") {
			t.Errorf("expected '--yes' hint in error, got: %v", err)
		}

		// No tasks should have been removed
		content, _ := os.ReadFile(filepath.Join(globalDir, "tasks.cue"))
		s := string(content)
		if !strings.Contains(s, "golang/review/architecture") {
			t.Errorf("architecture should still exist after failed remove: %s", s)
		}
		if !strings.Contains(s, "confluence/read-doc") {
			t.Errorf("read-doc should still exist after failed remove: %s", s)
		}
	})
}
