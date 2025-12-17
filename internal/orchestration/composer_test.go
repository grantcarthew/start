package orchestration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestComposer_Compose(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name        string
		config      string
		selection   ContextSelection
		customText  string
		wantPrompt  string
		wantCtxs    []string
		wantWarning bool
	}{
		{
			name: "required contexts only",
			config: `
				contexts: {
					env: {
						required: true
						prompt: "Environment info"
					}
					project: {
						default: true
						prompt: "Project info"
					}
				}
			`,
			selection: ContextSelection{
				IncludeRequired: true,
				IncludeDefaults: false,
			},
			wantPrompt: "Environment info",
			wantCtxs:   []string{"env"},
		},
		{
			name: "required and default contexts",
			config: `
				contexts: {
					env: {
						required: true
						prompt: "Environment info"
					}
					project: {
						default: true
						prompt: "Project info"
					}
				}
			`,
			selection: ContextSelection{
				IncludeRequired: true,
				IncludeDefaults: true,
			},
			wantPrompt: "Environment info\n\nProject info",
			wantCtxs:   []string{"env", "project"},
		},
		{
			name: "tagged contexts",
			config: `
				contexts: {
					env: {
						required: true
						prompt: "Environment"
					}
					security: {
						tags: ["security"]
						prompt: "Security context"
					}
					performance: {
						tags: ["performance"]
						prompt: "Performance context"
					}
				}
			`,
			selection: ContextSelection{
				IncludeRequired: true,
				Tags:            []string{"security"},
			},
			wantPrompt: "Environment\n\nSecurity context",
			wantCtxs:   []string{"env", "security"},
		},
		{
			name: "default pseudo-tag",
			config: `
				contexts: {
					env: {
						required: true
						prompt: "Environment"
					}
					project: {
						default: true
						prompt: "Project"
					}
					debug: {
						tags: ["debug"]
						prompt: "Debug"
					}
				}
			`,
			selection: ContextSelection{
				IncludeRequired: true,
				Tags:            []string{"default", "debug"},
			},
			wantPrompt: "Environment\n\nProject\n\nDebug",
			wantCtxs:   []string{"env", "project", "debug"},
		},
		{
			name: "custom text appended",
			config: `
				contexts: {
					env: {
						required: true
						prompt: "Environment"
					}
				}
			`,
			selection: ContextSelection{
				IncludeRequired: true,
			},
			customText: "Please review this code",
			wantPrompt: "Environment\n\nPlease review this code",
			wantCtxs:   []string{"env"},
		},
		{
			name:   "no contexts defined",
			config: `{}`,
			selection: ContextSelection{
				IncludeRequired: true,
				IncludeDefaults: true,
			},
			wantPrompt: "",
			wantCtxs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ctx.CompileString(tt.config)
			if err := cfg.Err(); err != nil {
				t.Fatalf("compile config: %v", err)
			}

			processor := NewTemplateProcessor(nil, nil, "")
			composer := NewComposer(processor, "")

			result, err := composer.Compose(cfg, tt.selection, tt.customText, "")
			if err != nil {
				t.Fatalf("Compose() error = %v", err)
			}

			if result.Prompt != tt.wantPrompt {
				t.Errorf("Prompt = %q, want %q", result.Prompt, tt.wantPrompt)
			}

			// Check contexts
			var gotCtxs []string
			for _, ctx := range result.Contexts {
				gotCtxs = append(gotCtxs, ctx.Name)
			}
			if len(gotCtxs) != len(tt.wantCtxs) {
				t.Errorf("Contexts = %v, want %v", gotCtxs, tt.wantCtxs)
			} else {
				for i, name := range tt.wantCtxs {
					if gotCtxs[i] != name {
						t.Errorf("Contexts[%d] = %q, want %q", i, gotCtxs[i], name)
					}
				}
			}
		})
	}
}

func TestComposer_ComposeWithRole(t *testing.T) {
	ctx := cuecontext.New()

	config := `
		settings: {
			default_role: "assistant"
		}
		contexts: {
			env: {
				required: true
				prompt: "Environment"
			}
		}
		roles: {
			assistant: {
				prompt: "You are a helpful assistant."
			}
			reviewer: {
				prompt: "You are a code reviewer."
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, "")
	composer := NewComposer(processor, "")

	t.Run("uses default role", func(t *testing.T) {
		result, err := composer.ComposeWithRole(cfg, ContextSelection{IncludeRequired: true}, "", "", "")
		if err != nil {
			t.Fatalf("ComposeWithRole() error = %v", err)
		}

		if result.RoleName != "assistant" {
			t.Errorf("RoleName = %q, want 'assistant'", result.RoleName)
		}
		if result.Role != "You are a helpful assistant." {
			t.Errorf("Role = %q, want 'You are a helpful assistant.'", result.Role)
		}
	})

	t.Run("uses specified role", func(t *testing.T) {
		result, err := composer.ComposeWithRole(cfg, ContextSelection{IncludeRequired: true}, "reviewer", "", "")
		if err != nil {
			t.Fatalf("ComposeWithRole() error = %v", err)
		}

		if result.RoleName != "reviewer" {
			t.Errorf("RoleName = %q, want 'reviewer'", result.RoleName)
		}
		if result.Role != "You are a code reviewer." {
			t.Errorf("Role = %q, want 'You are a code reviewer.'", result.Role)
		}
	})

	t.Run("nonexistent role adds warning", func(t *testing.T) {
		result, err := composer.ComposeWithRole(cfg, ContextSelection{IncludeRequired: true}, "nonexistent", "", "")
		if err != nil {
			t.Fatalf("ComposeWithRole() error = %v", err)
		}

		if len(result.Warnings) == 0 {
			t.Error("expected warning for nonexistent role")
		}
		if result.Role != "" {
			t.Errorf("Role should be empty for nonexistent role, got %q", result.Role)
		}
	})
}

func TestComposer_ResolveTask(t *testing.T) {
	ctx := cuecontext.New()

	tmpDir := t.TempDir()

	config := `
		tasks: {
			"code-review": {
				command: "echo staged changes"
				prompt: """
					Review the following:

					{{.CommandOutput}}

					Instructions: {{.Instructions}}
					"""
			}
			"simple": {
				prompt: "Simple task prompt"
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	runner := &mockShellRunner{output: "diff output here"}
	processor := NewTemplateProcessor(nil, runner, tmpDir)
	composer := NewComposer(processor, tmpDir)

	t.Run("task with command and instructions", func(t *testing.T) {
		result, err := composer.ResolveTask(cfg, "code-review", "focus on security")
		if err != nil {
			t.Fatalf("ResolveTask() error = %v", err)
		}

		if !strings.Contains(result.Content, "diff output here") {
			t.Errorf("Content should contain command output")
		}
		if !strings.Contains(result.Content, "focus on security") {
			t.Errorf("Content should contain instructions")
		}
	})

	t.Run("simple task", func(t *testing.T) {
		result, err := composer.ResolveTask(cfg, "simple", "")
		if err != nil {
			t.Fatalf("ResolveTask() error = %v", err)
		}

		if result.Content != "Simple task prompt" {
			t.Errorf("Content = %q, want 'Simple task prompt'", result.Content)
		}
	})

	t.Run("nonexistent task", func(t *testing.T) {
		_, err := composer.ResolveTask(cfg, "nonexistent", "")
		if err == nil {
			t.Error("expected error for nonexistent task")
		}
	})
}

func TestComposer_ContextWithFile(t *testing.T) {
	ctx := cuecontext.New()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "context.md")
	if err := os.WriteFile(filePath, []byte("File-based context content"), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	config := `
		contexts: {
			file_ctx: {
				required: true
				file: "` + filePath + `"
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, tmpDir)
	composer := NewComposer(processor, tmpDir)

	result, err := composer.Compose(cfg, ContextSelection{IncludeRequired: true}, "", "")
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if result.Prompt != "File-based context content" {
		t.Errorf("Prompt = %q, want 'File-based context content'", result.Prompt)
	}
}

func TestGetTaskRole(t *testing.T) {
	ctx := cuecontext.New()

	config := `
		tasks: {
			"with-role": {
				role: "reviewer"
				prompt: "Review code"
			}
			"without-role": {
				prompt: "Simple task"
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	t.Run("task with role", func(t *testing.T) {
		role := GetTaskRole(cfg, "with-role")
		if role != "reviewer" {
			t.Errorf("GetTaskRole() = %q, want 'reviewer'", role)
		}
	})

	t.Run("task without role", func(t *testing.T) {
		role := GetTaskRole(cfg, "without-role")
		if role != "" {
			t.Errorf("GetTaskRole() = %q, want empty", role)
		}
	})

	t.Run("nonexistent task", func(t *testing.T) {
		role := GetTaskRole(cfg, "nonexistent")
		if role != "" {
			t.Errorf("GetTaskRole() = %q, want empty", role)
		}
	})
}

func TestExtractUTDFields(t *testing.T) {
	ctx := cuecontext.New()

	config := `
		item: {
			file: "test.md"
			command: "echo hello"
			prompt: "Test prompt"
			shell: "bash -c"
			timeout: 60
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	item := cfg.LookupPath(cue.ParsePath("item"))
	fields := extractUTDFields(item)

	if fields.File != "test.md" {
		t.Errorf("File = %q, want 'test.md'", fields.File)
	}
	if fields.Command != "echo hello" {
		t.Errorf("Command = %q, want 'echo hello'", fields.Command)
	}
	if fields.Prompt != "Test prompt" {
		t.Errorf("Prompt = %q, want 'Test prompt'", fields.Prompt)
	}
	if fields.Shell != "bash -c" {
		t.Errorf("Shell = %q, want 'bash -c'", fields.Shell)
	}
	if fields.Timeout != 60 {
		t.Errorf("Timeout = %d, want 60", fields.Timeout)
	}
}

func TestGetDefaultRole(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name     string
		config   string
		wantRole string
	}{
		{
			name: "uses settings.default_role",
			config: `
				settings: {
					default_role: "expert"
				}
				roles: {
					assistant: { prompt: "You are an assistant." }
					expert: { prompt: "You are an expert." }
				}
			`,
			wantRole: "expert",
		},
		{
			name: "falls back to first role when no default",
			config: `
				roles: {
					first: { prompt: "First role." }
					second: { prompt: "Second role." }
				}
			`,
			wantRole: "first",
		},
		{
			name:     "returns empty when no roles defined",
			config:   `{}`,
			wantRole: "",
		},
		{
			name: "returns empty when settings exists but no default_role",
			config: `
				settings: {
					default_agent: "claude"
				}
			`,
			wantRole: "",
		},
		{
			name: "returns empty when roles is empty",
			config: `
				roles: {}
			`,
			wantRole: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ctx.CompileString(tt.config)
			if err := cfg.Err(); err != nil {
				t.Fatalf("compile config: %v", err)
			}

			processor := NewTemplateProcessor(nil, nil, "")
			composer := NewComposer(processor, "")

			result := composer.getDefaultRole(cfg)
			if result != tt.wantRole {
				t.Errorf("getDefaultRole() = %q, want %q", result, tt.wantRole)
			}
		})
	}
}

func TestComposer_ResolveContext_Errors(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name       string
		config     string
		contextName string
		wantErr    string
	}{
		{
			name: "context not found",
			config: `
				contexts: {
					env: { prompt: "Environment" }
				}
			`,
			contextName: "nonexistent",
			wantErr:     "context not found",
		},
		{
			name: "invalid UTD - no file, command, or prompt",
			config: `
				contexts: {
					invalid: {
						description: "This context has no UTD fields"
					}
				}
			`,
			contextName: "invalid",
			wantErr:     "invalid UTD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ctx.CompileString(tt.config)
			if err := cfg.Err(); err != nil {
				t.Fatalf("compile config: %v", err)
			}

			processor := NewTemplateProcessor(nil, nil, "")
			composer := NewComposer(processor, "")

			_, err := composer.resolveContext(cfg, tt.contextName)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestComposer_ResolveRole_Errors(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name     string
		config   string
		roleName string
		wantErr  string
	}{
		{
			name: "role not found",
			config: `
				roles: {
					assistant: { prompt: "You are an assistant." }
				}
			`,
			roleName: "nonexistent",
			wantErr:  "role not found",
		},
		{
			name: "invalid UTD - no file, command, or prompt",
			config: `
				roles: {
					invalid: {
						description: "This role has no UTD fields"
					}
				}
			`,
			roleName: "invalid",
			wantErr:  "invalid UTD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ctx.CompileString(tt.config)
			if err := cfg.Err(); err != nil {
				t.Fatalf("compile config: %v", err)
			}

			processor := NewTemplateProcessor(nil, nil, "")
			composer := NewComposer(processor, "")

			_, err := composer.resolveRole(cfg, tt.roleName)
			if err == nil {
				t.Error("expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}
