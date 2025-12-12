package orchestration

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
)

func TestExecutor_BuildCommand(t *testing.T) {
	executor := NewExecutor("")

	tests := []struct {
		name        string
		config      ExecuteConfig
		wantContain string
		wantErr     bool
	}{
		{
			name: "simple command with bin",
			config: ExecuteConfig{
				Agent: Agent{
					Bin:     "claude",
					Command: "{{.bin}} chat",
				},
			},
			wantContain: "claude chat",
		},
		{
			name: "command with model",
			config: ExecuteConfig{
				Agent: Agent{
					Bin:     "claude",
					Command: "{{.bin}} --model {{.model}}",
					Models: map[string]string{
						"sonnet": "claude-sonnet-4-20250514",
					},
					DefaultModel: "sonnet",
				},
			},
			wantContain: "claude-sonnet-4-20250514",
		},
		{
			name: "command with model override",
			config: ExecuteConfig{
				Agent: Agent{
					Bin:     "claude",
					Command: "{{.bin}} --model {{.model}}",
					Models: map[string]string{
						"sonnet": "claude-sonnet-4-20250514",
						"opus":   "claude-opus-4-20250514",
					},
					DefaultModel: "sonnet",
				},
				Model: "opus",
			},
			wantContain: "claude-opus-4-20250514",
		},
		{
			name: "command with role",
			config: ExecuteConfig{
				Agent: Agent{
					Bin:     "claude",
					Command: "{{.bin}} --system '{{.role}}'",
				},
				Role: "You are a code reviewer.",
			},
			wantContain: "You are a code reviewer",
		},
		{
			name: "command with role file",
			config: ExecuteConfig{
				Agent: Agent{
					Bin:     "claude",
					Command: "{{.bin}} --system-file {{.role_file}}",
				},
				RoleFile: "/tmp/role.md",
			},
			wantContain: "/tmp/role.md",
		},
		{
			name: "conditional model in template",
			config: ExecuteConfig{
				Agent: Agent{
					Bin:     "claude",
					Command: "{{.bin}}{{if .model}} --model {{.model}}{{end}}",
				},
			},
			wantContain: "claude",
		},
		{
			name: "invalid template",
			config: ExecuteConfig{
				Agent: Agent{
					Command: "{{.Invalid",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := executor.BuildCommand(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !strings.Contains(cmd, tt.wantContain) {
				t.Errorf("command = %q, want containing %q", cmd, tt.wantContain)
			}
		})
	}
}

func TestEscapeForShell(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple text", "simple text"},
		{"it's a test", "it'\"'\"'s a test"},
		{"no quotes here", "no quotes here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeForShell(tt.input)
			if got != tt.want {
				t.Errorf("escapeForShell(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractAgent(t *testing.T) {
	ctx := cuecontext.New()

	config := `
		agents: {
			claude: {
				bin: "claude"
				command: "{{.bin}} --model {{.model}}"
				default_model: "sonnet"
				description: "Anthropic Claude"
				models: {
					sonnet: "claude-sonnet-4-20250514"
					opus: "claude-opus-4-20250514"
				}
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	t.Run("extracts all fields", func(t *testing.T) {
		agent, err := ExtractAgent(cfg, "claude")
		if err != nil {
			t.Fatalf("ExtractAgent() error = %v", err)
		}

		if agent.Name != "claude" {
			t.Errorf("Name = %q, want 'claude'", agent.Name)
		}
		if agent.Bin != "claude" {
			t.Errorf("Bin = %q, want 'claude'", agent.Bin)
		}
		if agent.DefaultModel != "sonnet" {
			t.Errorf("DefaultModel = %q, want 'sonnet'", agent.DefaultModel)
		}
		if agent.Description != "Anthropic Claude" {
			t.Errorf("Description = %q, want 'Anthropic Claude'", agent.Description)
		}
		if len(agent.Models) != 2 {
			t.Errorf("Models count = %d, want 2", len(agent.Models))
		}
		if agent.Models["sonnet"] != "claude-sonnet-4-20250514" {
			t.Errorf("Models[sonnet] = %q, want 'claude-sonnet-4-20250514'", agent.Models["sonnet"])
		}
	})

	t.Run("nonexistent agent", func(t *testing.T) {
		_, err := ExtractAgent(cfg, "nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent agent")
		}
	})
}

func TestGetDefaultAgent(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("from settings", func(t *testing.T) {
		config := `
			settings: {
				default_agent: "gemini"
			}
			agents: {
				claude: { bin: "claude", command: "claude" }
				gemini: { bin: "gemini", command: "gemini" }
			}
		`
		cfg := ctx.CompileString(config)
		if err := cfg.Err(); err != nil {
			t.Fatalf("compile config: %v", err)
		}

		name := GetDefaultAgent(cfg)
		if name != "gemini" {
			t.Errorf("GetDefaultAgent() = %q, want 'gemini'", name)
		}
	})

	t.Run("falls back to first agent", func(t *testing.T) {
		config := `
			agents: {
				claude: { bin: "claude", command: "claude" }
				gemini: { bin: "gemini", command: "gemini" }
			}
		`
		cfg := ctx.CompileString(config)
		if err := cfg.Err(); err != nil {
			t.Fatalf("compile config: %v", err)
		}

		name := GetDefaultAgent(cfg)
		if name != "claude" {
			t.Errorf("GetDefaultAgent() = %q, want 'claude'", name)
		}
	})

	t.Run("no agents defined", func(t *testing.T) {
		config := `{}`
		cfg := ctx.CompileString(config)
		if err := cfg.Err(); err != nil {
			t.Fatalf("compile config: %v", err)
		}

		name := GetDefaultAgent(cfg)
		if name != "" {
			t.Errorf("GetDefaultAgent() = %q, want empty", name)
		}
	})
}

func TestGenerateDryRunCommand(t *testing.T) {
	agent := Agent{
		Name: "claude",
	}
	contexts := []string{"env", "project"}
	workingDir := "/home/user/project"
	cmdStr := "claude --model sonnet 'hello'"

	result := GenerateDryRunCommand(agent, "sonnet", "code-reviewer", contexts, workingDir, cmdStr)

	expectedContains := []string{
		"# Agent: claude",
		"# Model: sonnet",
		"# Role: code-reviewer",
		"# Contexts: env, project",
		"# Working Directory: /home/user/project",
		"claude --model sonnet",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result, expected) {
			t.Errorf("result should contain %q", expected)
		}
	}
}

func TestExecutor_ExecuteWithoutReplace(t *testing.T) {
	executor := NewExecutor("")

	t.Run("simple echo command", func(t *testing.T) {
		config := ExecuteConfig{
			Agent: Agent{
				Bin:     "echo",
				Command: "{{.bin}} hello world",
			},
		}

		output, err := executor.ExecuteWithoutReplace(config)
		if err != nil {
			t.Fatalf("ExecuteWithoutReplace() error = %v", err)
		}

		if !strings.Contains(output, "hello world") {
			t.Errorf("output = %q, want containing 'hello world'", output)
		}
	})
}
