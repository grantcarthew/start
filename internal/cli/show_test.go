package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestConfig creates a temp directory with CUE config for testing.
// Returns the directory path and a cleanup function.
func setupTestConfig(t *testing.T) (string, func()) {
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

	// Change to the temp directory so .start is found
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getting cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("changing to temp dir: %v", err)
	}

	// Also set temp HOME for global scope isolation
	origHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", dir)

	cleanup := func() {
		_ = os.Chdir(origDir)
		_ = os.Setenv("HOME", origHome)
	}

	return dir, cleanup
}

// TestPrepareShowAgent tests the prepareShowAgent logic function.
func TestPrepareShowAgent(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

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
			result, err := prepareShowAgent(tt.agentName, tt.scope)

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
	_, cleanup := setupTestConfig(t)
	defer cleanup()

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
			name:     "nonexistent role",
			roleName: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareShowRole(tt.roleName, "")

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
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	tests := []struct {
		name                 string
		contextName          string
		wantType             string
		wantName             string
		wantContent          []string
		wantAllNames         []string
		wantDefaultContexts  []string
		wantRequiredContexts []string
		wantListOnly         bool
		wantErr              bool
	}{
		{
			name:                 "no name returns list only",
			contextName:          "",
			wantType:             "Context",
			wantAllNames:         []string{"environment", "git-status"},
			wantDefaultContexts:  []string{"git-status"},
			wantRequiredContexts: []string{"environment"},
			wantListOnly:         true,
			wantErr:              false,
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
			wantListOnly: false,
			wantErr:      false,
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
			wantListOnly: false,
			wantErr:      false,
		},
		{
			name:        "nonexistent context",
			contextName: "nonexistent",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareShowContext(tt.contextName, "")

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
			if result.ListOnly != tt.wantListOnly {
				t.Errorf("ListOnly = %v, want %v", result.ListOnly, tt.wantListOnly)
			}
			if tt.wantListOnly {
				if len(result.AllNames) != len(tt.wantAllNames) {
					t.Errorf("AllNames = %v, want %v", result.AllNames, tt.wantAllNames)
				}
				if len(result.DefaultContexts) != len(tt.wantDefaultContexts) {
					t.Errorf("DefaultContexts = %v, want %v", result.DefaultContexts, tt.wantDefaultContexts)
				}
				if len(result.RequiredContexts) != len(tt.wantRequiredContexts) {
					t.Errorf("RequiredContexts = %v, want %v", result.RequiredContexts, tt.wantRequiredContexts)
				}
			} else {
				if result.Name != tt.wantName {
					t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
				}
				for _, want := range tt.wantContent {
					if !strings.Contains(result.Content, want) {
						t.Errorf("Content missing %q\ngot: %s", want, result.Content)
					}
				}
			}
		})
	}
}

// TestPrepareShowTask tests the prepareShowTask logic function.
func TestPrepareShowTask(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	tests := []struct {
		name         string
		taskName     string
		wantType     string
		wantName     string
		wantContent  []string
		wantAllNames []string
		wantListOnly bool
		wantErr      bool
	}{
		{
			name:         "no name returns list only",
			taskName:     "",
			wantType:     "Task",
			wantAllNames: []string{"review"},
			wantListOnly: true,
			wantErr:      false,
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
			wantListOnly: false,
			wantErr:      false,
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
			wantListOnly: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := prepareShowTask(tt.taskName, "")

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
			if result.ListOnly != tt.wantListOnly {
				t.Errorf("ListOnly = %v, want %v", result.ListOnly, tt.wantListOnly)
			}
			if tt.wantListOnly {
				if len(result.AllNames) != len(tt.wantAllNames) {
					t.Errorf("AllNames = %v, want %v", result.AllNames, tt.wantAllNames)
				}
			} else {
				if result.Name != tt.wantName {
					t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
				}
				for _, want := range tt.wantContent {
					if !strings.Contains(result.Content, want) {
						t.Errorf("Content missing %q\ngot: %s", want, result.Content)
					}
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
	if !strings.Contains(output, "Showing: claude (first in config)") {
		t.Errorf("output should contain 'Showing: claude (first in config)', got: %s", output)
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

// TestPrintListOnly tests list-only output.
func TestPrintListOnly(t *testing.T) {
	result := ShowResult{
		ItemType:         "Context",
		AllNames:         []string{"environment", "project", "git-status"},
		DefaultContexts:  []string{"project"},
		RequiredContexts: []string{"environment"},
		AllTags:          []string{"git", "system"},
		ListOnly:         true,
	}

	var buf bytes.Buffer
	printPreview(&buf, result)

	output := buf.String()

	if !strings.Contains(output, "Contexts: environment, project, git-status") {
		t.Errorf("output should contain contexts list, got: %s", output)
	}
	if !strings.Contains(output, "Default: project") {
		t.Errorf("output should contain default contexts, got: %s", output)
	}
	if !strings.Contains(output, "Required: environment") {
		t.Errorf("output should contain required contexts, got: %s", output)
	}
	if !strings.Contains(output, "Tags: git, system") {
		t.Errorf("output should contain tags, got: %s", output)
	}
}

// TestShowCommandIntegration tests the full command flow via Cobra.
func TestShowCommandIntegration(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	tests := []struct {
		name       string
		args       []string
		wantOutput []string
		wantErr    bool
	}{
		{
			name:       "show agent no name shows first agent",
			args:       []string{"show", "agent"},
			wantOutput: []string{"Agents:", "Showing:", "claude", "first in config"},
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
			wantOutput: []string{"Roles:", "Showing:", "assistant", "first in config"},
			wantErr:    false,
		},
		{
			name:       "show context no name lists contexts",
			args:       []string{"show", "context"},
			wantOutput: []string{"Contexts:", "environment", "git-status"},
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
			name:       "show task no name lists tasks",
			args:       []string{"show", "task"},
			wantOutput: []string{"Tasks:", "review"},
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
