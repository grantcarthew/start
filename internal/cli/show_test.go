package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	internalcue "github.com/grantcarthew/start/internal/cue"
)

// setupTestConfig creates a temp directory with CUE config for testing.
//
// Note: Tests below use os.Chdir (process-global state). Do not add t.Parallel()
// to any test that calls os.Chdir — it will cause data races on the working directory.
func setupTestConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	startDir := filepath.Join(dir, ".start")
	if err := os.MkdirAll(startDir, 0755); err != nil {
		t.Fatalf("creating .start dir: %v", err)
	}

	config := `
agents: {
	claude: {
		bin:         "claude"
		command:     "{{.bin}} --model {{.model}} '{{.prompt}}'"
		description: "Claude by Anthropic"
		default_model: "sonnet"
		models: {
			sonnet: "claude-sonnet-4-20250514"
			opus:   "claude-opus-4-20250514"
		}
		tags: ["anthropic", "claude", "ai"]
	}
}

contexts: {
	environment: {
		required: true
		file:     "~/context/ENVIRONMENT.md"
		prompt:   "Environment context loaded."
		tags:     ["system", "environment"]
	}
	"git-status": {
		default: true
		tags:    ["git", "vcs"]
		command: "git status --short"
		prompt:  "Git status output."
	}
}

roles: {
	assistant: {
		description: "General assistant"
		prompt:      "You are a helpful assistant."
	}
	"code-reviewer": {
		description: "Code reviewer"
		prompt:      "You are an expert code reviewer."
	}
}

tasks: {
	review: {
		description: "Review changes"
		role:        "code-reviewer"
		command:     "git diff --staged"
		prompt:      "Review: {{.command_output}}"
	}
}
`
	configPath := filepath.Join(startDir, "settings.cue")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	chdir(t, dir)
	t.Setenv("HOME", dir)

	return dir
}

// TestPrepareShowAgent tests the prepareShowAgent logic function.
func TestPrepareShowAgent(t *testing.T) {
	setupTestConfig(t)

	tests := []struct {
		name           string
		agentName      string
		scope          string
		wantType       string
		wantName       string
		wantContent    []string
		wantAllNames   []string
		wantShowReason string
		wantErr        bool
	}{
		{
			name:           "no name shows first agent",
			agentName:      "",
			scope:          "",
			wantType:       "Agent",
			wantName:       "claude",
			wantAllNames:   []string{"claude"},
			wantShowReason: "first in config",
			wantErr:        false,
		},
		{
			name:      "named agent with all fields",
			agentName: "claude",
			scope:     "",
			wantType:  "Agent",
			wantName:  "claude",
			wantContent: []string{
				"Claude by Anthropic",
				"sonnet: claude-sonnet-4-20250514",
				"opus: claude-opus-4-20250514",
				"Tags: anthropic, claude, ai",
			},
			wantAllNames: []string{"claude"},
			wantErr:      false,
		},
		{
			name:      "substring match",
			agentName: "clau",
			scope:     "",
			wantType:  "Agent",
			wantName:  "claude",
			wantContent: []string{
				"Claude by Anthropic",
			},
			wantErr: false,
		},
		{
			name:      "nonexistent agent",
			agentName: "nonexistent",
			scope:     "",
			wantErr:   true,
		},
		{
			name:      "scope global no config",
			agentName: "",
			scope:     "global",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareShow(tt.agentName, tt.scope, internalcue.KeyAgents, "Agent")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ItemType != tt.wantType {
				t.Errorf("ItemType = %q, want %q", result.ItemType, tt.wantType)
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			for _, want := range tt.wantContent {
				if !strings.Contains(result.Content, want) {
					t.Errorf("Content missing %q\ngot: %s", want, result.Content)
				}
			}
			if tt.wantShowReason != "" && result.ShowReason != tt.wantShowReason {
				t.Errorf("ShowReason = %q, want %q", result.ShowReason, tt.wantShowReason)
			}
			// Check AllNames
			if len(tt.wantAllNames) > 0 {
				if len(result.AllNames) != len(tt.wantAllNames) {
					t.Errorf("AllNames length = %d, want %d", len(result.AllNames), len(tt.wantAllNames))
				}
			}
		})
	}
}

// TestPrepareShowRole tests the prepareShowRole logic function.
func TestPrepareShowRole(t *testing.T) {
	setupTestConfig(t)

	tests := []struct {
		name           string
		roleName       string
		wantType       string
		wantName       string
		wantContent    []string
		wantAllNames   []string
		wantShowReason string
		wantErr        bool
	}{
		{
			name:           "no name shows first role",
			roleName:       "",
			wantType:       "Role",
			wantName:       "assistant",
			wantAllNames:   []string{"assistant", "code-reviewer"},
			wantShowReason: "first in config",
			wantErr:        false,
		},
		{
			name:     "named role with hyphen",
			roleName: "code-reviewer",
			wantType: "Role",
			wantName: "code-reviewer",
			wantContent: []string{
				"Description: Code reviewer",
				"You are an expert code reviewer.",
			},
			wantErr: false,
		},
		{
			name:     "substring match",
			roleName: "code",
			wantType: "Role",
			wantName: "code-reviewer",
			wantContent: []string{
				"Description: Code reviewer",
			},
			wantErr: false,
		},
		{
			name:     "nonexistent role",
			roleName: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareShow(tt.roleName, "", internalcue.KeyRoles, "Role")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ItemType != tt.wantType {
				t.Errorf("ItemType = %q, want %q", result.ItemType, tt.wantType)
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			for _, want := range tt.wantContent {
				if !strings.Contains(result.Content, want) {
					t.Errorf("Content missing %q\ngot: %s", want, result.Content)
				}
			}
			if tt.wantShowReason != "" && result.ShowReason != tt.wantShowReason {
				t.Errorf("ShowReason = %q, want %q", result.ShowReason, tt.wantShowReason)
			}
			if len(tt.wantAllNames) > 0 {
				if len(result.AllNames) != len(tt.wantAllNames) {
					t.Errorf("AllNames length = %d, want %d", len(result.AllNames), len(tt.wantAllNames))
				}
			}
		})
	}
}

// TestPrepareShowContext tests the prepareShowContext logic function.
func TestPrepareShowContext(t *testing.T) {
	setupTestConfig(t)

	tests := []struct {
		name           string
		contextName    string
		wantType       string
		wantName       string
		wantContent    []string
		wantAllNames   []string
		wantShowReason string
		wantErr        bool
	}{
		{
			name:           "no name shows first context",
			contextName:    "",
			wantType:       "Context",
			wantName:       "environment",
			wantAllNames:   []string{"environment", "git-status"},
			wantShowReason: "first in config",
			wantErr:        false,
		},
		{
			name:        "context with file and required",
			contextName: "environment",
			wantType:    "Context",
			wantName:    "environment",
			wantContent: []string{
				"File: ~/context/ENVIRONMENT.md",
				"Environment context loaded.",
				"Required: true",
				"Tags: system, environment",
			},
			wantErr: false,
		},
		{
			name:        "context with command and default",
			contextName: "git-status",
			wantType:    "Context",
			wantName:    "git-status",
			wantContent: []string{
				"Command: git status --short",
				"Git status output.",
				"Default: true",
				"Tags: git, vcs",
			},
			wantErr: false,
		},
		{
			name:        "substring match",
			contextName: "git",
			wantType:    "Context",
			wantName:    "git-status",
			wantContent: []string{
				"Command: git status --short",
			},
			wantErr: false,
		},
		{
			name:        "nonexistent context",
			contextName: "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareShow(tt.contextName, "", internalcue.KeyContexts, "Context")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ItemType != tt.wantType {
				t.Errorf("ItemType = %q, want %q", result.ItemType, tt.wantType)
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			for _, want := range tt.wantContent {
				if !strings.Contains(result.Content, want) {
					t.Errorf("Content missing %q\ngot: %s", want, result.Content)
				}
			}
			if tt.wantShowReason != "" && result.ShowReason != tt.wantShowReason {
				t.Errorf("ShowReason = %q, want %q", result.ShowReason, tt.wantShowReason)
			}
			if len(tt.wantAllNames) > 0 {
				if len(result.AllNames) != len(tt.wantAllNames) {
					t.Errorf("AllNames length = %d, want %d", len(result.AllNames), len(tt.wantAllNames))
				}
			}
		})
	}
}

// TestPrepareShowTask tests the prepareShowTask logic function.
func TestPrepareShowTask(t *testing.T) {
	setupTestConfig(t)

	tests := []struct {
		name           string
		taskName       string
		wantType       string
		wantName       string
		wantContent    []string
		wantAllNames   []string
		wantShowReason string
		wantErr        bool
	}{
		{
			name:           "no name shows first task",
			taskName:       "",
			wantType:       "Task",
			wantName:       "review",
			wantAllNames:   []string{"review"},
			wantShowReason: "first in config",
			wantErr:        false,
		},
		{
			name:     "task with all fields",
			taskName: "review",
			wantType: "Task",
			wantName: "review",
			wantContent: []string{
				"Description: Review changes",
				"Command: git diff --staged",
				"Review: {{.command_output}}",
				"Role: code-reviewer",
			},
			wantErr: false,
		},
		{
			name:     "nonexistent task",
			taskName: "nonexistent",
			wantErr:  true,
		},
		{
			name:     "substring match",
			taskName: "rev",
			wantType: "Task",
			wantName: "review",
			wantContent: []string{
				"Description: Review changes",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareShow(tt.taskName, "", internalcue.KeyTasks, "Task")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.ItemType != tt.wantType {
				t.Errorf("ItemType = %q, want %q", result.ItemType, tt.wantType)
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			for _, want := range tt.wantContent {
				if !strings.Contains(result.Content, want) {
					t.Errorf("Content missing %q\ngot: %s", want, result.Content)
				}
			}
			if tt.wantShowReason != "" && result.ShowReason != tt.wantShowReason {
				t.Errorf("ShowReason = %q, want %q", result.ShowReason, tt.wantShowReason)
			}
			if len(tt.wantAllNames) > 0 {
				if len(result.AllNames) != len(tt.wantAllNames) {
					t.Errorf("AllNames length = %d, want %d", len(result.AllNames), len(tt.wantAllNames))
				}
			}
		})
	}
}

// TestPrintPreview tests the printPreview output function.
func TestPrintPreview(t *testing.T) {
	result := ShowResult{
		ItemType:   "Agent",
		Name:       "claude",
		Content:    "Line 1\nLine 2\nLine 3",
		AllNames:   []string{"claude", "gemini"},
		ShowReason: "first in config",
	}

	var buf bytes.Buffer
	printPreview(&buf, result)

	output := buf.String()

	// Check list of all items
	if !strings.Contains(output, "Agents: claude, gemini") {
		t.Errorf("output should contain agents list, got: %s", output)
	}

	// Check header with show reason
	if !strings.Contains(output, "Agent: claude") {
		t.Errorf("output should contain 'Agent: claude', got: %s", output)
	}
	if !strings.Contains(output, "first in config") {
		t.Errorf("output should contain 'first in config', got: %s", output)
	}

	// Check separator line
	if !strings.Contains(output, strings.Repeat("─", 79)) {
		t.Error("output should contain separator line")
	}

	// Check full content is shown
	if !strings.Contains(output, "Line 1") {
		t.Error("output should contain Line 1")
	}
	if !strings.Contains(output, "Line 3") {
		t.Error("output should contain Line 3")
	}
}

// TestShowCommandIntegration tests the full command flow via Cobra.
func TestShowCommandIntegration(t *testing.T) {
	setupTestConfig(t)

	tests := []struct {
		name       string
		args       []string
		wantOutput []string
		wantErr    bool
	}{
		{
			name:       "show agent no name shows first agent",
			args:       []string{"show", "agent"},
			wantOutput: []string{"Agents:", "Agent:", "claude", "first in config"},
			wantErr:    false,
		},
		{
			name:       "show agent with name shows content",
			args:       []string{"show", "agent", "claude"},
			wantOutput: []string{"Agent: claude", "Description:"},
			wantErr:    false,
		},
		{
			name:       "show role no name shows first role",
			args:       []string{"show", "role"},
			wantOutput: []string{"Roles:", "Role:", "assistant", "first in config"},
			wantErr:    false,
		},
		{
			name:       "show context no name shows first context",
			args:       []string{"show", "context"},
			wantOutput: []string{"Contexts:", "Context:", "environment", "first in config"},
			wantErr:    false,
		},
		{
			name:       "show context with name shows content",
			args:       []string{"show", "context", "environment"},
			wantOutput: []string{"Context: environment"},
			wantErr:    false,
		},
		{
			name:       "show task with name shows content",
			args:       []string{"show", "task", "review"},
			wantOutput: []string{"Task: review"},
			wantErr:    false,
		},
		{
			name:       "show task no name shows first task",
			args:       []string{"show", "task"},
			wantOutput: []string{"Tasks:", "Task:", "review", "first in config"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := NewRootCmd()
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\ngot: %s", want, output)
				}
			}
		})
	}
}
