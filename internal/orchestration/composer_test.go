package orchestration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
)

func TestComposer_Compose(t *testing.T) {
	t.Parallel()
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
		{
			name: "unmatched tag adds warning",
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
				}
			`,
			selection: ContextSelection{
				IncludeRequired: true,
				Tags:            []string{"nonexistent"},
			},
			wantPrompt:  "Environment",
			wantCtxs:    []string{"env"},
			wantWarning: true,
		},
		{
			name: "multiple unmatched tags add multiple warnings",
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
				Tags:            []string{"invalid1", "invalid2"},
			},
			wantPrompt:  "Environment",
			wantCtxs:    []string{"env"},
			wantWarning: true,
		},
		{
			name: "mix of valid and invalid tags warns only for invalid",
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
				}
			`,
			selection: ContextSelection{
				IncludeRequired: true,
				Tags:            []string{"security", "invalidtag"},
			},
			wantPrompt:  "Environment\n\nSecurity context",
			wantCtxs:    []string{"env", "security"},
			wantWarning: true,
		},
		{
			name: "default pseudo-tag does not warn",
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
				Tags:            []string{"default"},
			},
			wantPrompt:  "Environment",
			wantCtxs:    []string{"env"},
			wantWarning: false,
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

			// Check warnings
			hasWarning := len(result.Warnings) > 0
			if hasWarning != tt.wantWarning {
				t.Errorf("Warnings present = %v, want %v (warnings: %v)", hasWarning, tt.wantWarning, result.Warnings)
			}
		})
	}
}

func TestComposer_ComposeWithRole(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	ctx := cuecontext.New()

	tmpDir := t.TempDir()

	config := `
		tasks: {
			"code-review": {
				command: "echo staged changes"
				prompt: """
					Review the following:

					{{.command_output}}

					Instructions: {{.instructions}}
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	ctx := cuecontext.New()

	tests := []struct {
		name        string
		config      string
		contextName string
		wantErr     string
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
	t.Parallel()
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

			_, _, err := composer.resolveRole(cfg, tt.roleName)
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

func TestComposer_ResolveTask_TempFile(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()

	// Create two separate temp directories:
	// - workingDir: the project working directory
	// - externalDir: simulates CUE cache (outside working directory)
	workingDir := t.TempDir()
	externalDir := t.TempDir()

	// Create a source file in the external directory (simulating CUE cache)
	sourceFile := filepath.Join(externalDir, "task.md")
	if err := os.WriteFile(sourceFile, []byte("Task content here"), 0644); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	config := fmt.Sprintf(`
		tasks: {
			"test-task": {
				file: %q
				prompt: "Read {{.file}} for instructions."
			}
		}
	`, sourceFile)

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	result, err := composer.ResolveTask(cfg, "test-task", "")
	if err != nil {
		t.Fatalf("ResolveTask() error = %v", err)
	}

	// Verify temp file was created (because source is outside working directory)
	expectedTempPath := filepath.Join(workingDir, ".start", "temp", "task-test-task.md")
	if result.TempFile != expectedTempPath {
		t.Errorf("TempFile = %q, want %q", result.TempFile, expectedTempPath)
	}

	// Verify temp file exists and has correct content
	content, err := os.ReadFile(result.TempFile)
	if err != nil {
		t.Fatalf("reading temp file: %v", err)
	}
	if string(content) != "Task content here" {
		t.Errorf("temp file content = %q, want %q", string(content), "Task content here")
	}

	// Verify {{.file}} in prompt contains temp file path (for external files).
	// External files (outside working directory) use temp path because the
	// original location may be inaccessible to AI agents (e.g., CUE cache).
	if !strings.Contains(result.Content, expectedTempPath) {
		t.Errorf("Content should contain temp file path %q, got: %s", expectedTempPath, result.Content)
	}
}

func TestComposer_ResolveTask_TempFile_WithSlashInName(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()

	// Create two separate temp directories:
	// - workingDir: the project working directory
	// - externalDir: simulates CUE cache (outside working directory)
	workingDir := t.TempDir()
	externalDir := t.TempDir()

	sourceFile := filepath.Join(externalDir, "task.md")
	if err := os.WriteFile(sourceFile, []byte("Nested task content"), 0644); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	config := fmt.Sprintf(`
		tasks: {
			"start/create-task": {
				file: %q
				prompt: "Read {{.file}} for instructions."
			}
		}
	`, sourceFile)

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	result, err := composer.ResolveTask(cfg, "start/create-task", "")
	if err != nil {
		t.Fatalf("ResolveTask() error = %v", err)
	}

	// Verify filename derivation handles slashes (converted to dashes)
	expectedTempPath := filepath.Join(workingDir, ".start", "temp", "task-start-create-task.md")
	if result.TempFile != expectedTempPath {
		t.Errorf("TempFile = %q, want %q", result.TempFile, expectedTempPath)
	}
}

func TestComposer_ResolveTask_NoFile_NoTempFile(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()
	tmpDir := t.TempDir()

	config := `
		tasks: {
			"prompt-only": {
				prompt: "This task has no file."
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, tmpDir)
	composer := NewComposer(processor, tmpDir)

	result, err := composer.ResolveTask(cfg, "prompt-only", "")
	if err != nil {
		t.Fatalf("ResolveTask() error = %v", err)
	}

	// Verify no temp file for prompt-only tasks
	if result.TempFile != "" {
		t.Errorf("TempFile should be empty for prompt-only task, got %q", result.TempFile)
	}
}

func TestComposer_ResolveContext_TempFile(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()

	// Create two separate temp directories:
	// - workingDir: the project working directory
	// - externalDir: simulates CUE cache (outside working directory)
	workingDir := t.TempDir()
	externalDir := t.TempDir()

	sourceFile := filepath.Join(externalDir, "context.md")
	if err := os.WriteFile(sourceFile, []byte("Context content"), 0644); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	config := fmt.Sprintf(`
		contexts: {
			"project-info": {
				required: true
				file: %q
			}
		}
	`, sourceFile)

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	result, err := composer.resolveContext(cfg, "project-info")
	if err != nil {
		t.Fatalf("resolveContext() error = %v", err)
	}

	// Verify temp file was created (because source is outside working directory)
	expectedTempPath := filepath.Join(workingDir, ".start", "temp", "context-project-info.md")
	if result.TempFile != expectedTempPath {
		t.Errorf("TempFile = %q, want %q", result.TempFile, expectedTempPath)
	}

	// Verify content
	if result.Content != "Context content" {
		t.Errorf("Content = %q, want %q", result.Content, "Context content")
	}
}

func TestComposer_LocalFile_NoTempFile(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()
	workingDir := t.TempDir()

	// Create a local file within the working directory
	sourceFile := filepath.Join(workingDir, "AGENTS.md")
	if err := os.WriteFile(sourceFile, []byte("Local file content"), 0644); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	config := fmt.Sprintf(`
		contexts: {
			"agents": {
				required: true
				file: %q
			}
		}
	`, sourceFile)

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compile config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	result, err := composer.resolveContext(cfg, "agents")
	if err != nil {
		t.Fatalf("resolveContext() error = %v", err)
	}

	// Verify NO temp file was created (local files don't need temp copies)
	if result.TempFile != "" {
		t.Errorf("TempFile should be empty for local file, got %q", result.TempFile)
	}

	// Verify temp directory was not created
	tempDir := filepath.Join(workingDir, ".start", "temp")
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("temp directory should not exist for local files, but found: %s", tempDir)
	}

	// Verify content was still read correctly
	if result.Content != "Local file content" {
		t.Errorf("Content = %q, want %q", result.Content, "Local file content")
	}
}

func TestComposer_resolveFileToTemp(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(tmpDir, "source.md")
	if err := os.WriteFile(sourceFile, []byte("Source content"), 0644); err != nil {
		t.Fatalf("writing source file: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, tmpDir)
	composer := NewComposer(processor, tmpDir)

	t.Run("creates temp file with correct content", func(t *testing.T) {
		tempPath, err := composer.resolveFileToTemp("task", "test", sourceFile)
		if err != nil {
			t.Fatalf("resolveFileToTemp() error = %v", err)
		}

		content, err := os.ReadFile(tempPath)
		if err != nil {
			t.Fatalf("reading temp file: %v", err)
		}
		if string(content) != "Source content" {
			t.Errorf("content = %q, want %q", string(content), "Source content")
		}
	})

	t.Run("returns empty string for empty path", func(t *testing.T) {
		tempPath, err := composer.resolveFileToTemp("task", "test", "")
		if err != nil {
			t.Fatalf("resolveFileToTemp() error = %v", err)
		}
		if tempPath != "" {
			t.Errorf("tempPath = %q, want empty", tempPath)
		}
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := composer.resolveFileToTemp("task", "test", "/nonexistent/file.md")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestComposer_isLocalFile(t *testing.T) {
	t.Parallel()
	workingDir := "/home/user/project"
	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "empty path",
			filePath: "",
			want:     false,
		},
		{
			name:     "relative path - simple",
			filePath: "AGENTS.md",
			want:     true,
		},
		{
			name:     "relative path - with dot prefix",
			filePath: "./AGENTS.md",
			want:     true,
		},
		{
			name:     "relative path - subdirectory",
			filePath: "docs/guide.md",
			want:     true,
		},
		{
			name:     "relative path - with dot prefix subdirectory",
			filePath: "./docs/guide.md",
			want:     true,
		},
		{
			name:     "absolute path - child of working dir",
			filePath: "/home/user/project/docs/guide.md",
			want:     true,
		},
		{
			name:     "absolute path - deeply nested child",
			filePath: "/home/user/project/a/b/c/file.md",
			want:     true,
		},
		{
			name:     "absolute path - outside working dir",
			filePath: "/tmp/cache/file.md",
			want:     false,
		},
		{
			name:     "absolute path - sibling directory",
			filePath: "/home/user/other-project/file.md",
			want:     false,
		},
		{
			name:     "absolute path - parent directory",
			filePath: "/home/user/file.md",
			want:     false,
		},
		{
			name:     "absolute path - similar prefix but not child",
			filePath: "/home/user/project-other/file.md",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := composer.isLocalFile(tt.filePath)
			if got != tt.want {
				t.Errorf("isLocalFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestComposer_TildeExpansion_Context(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()

	// Get home directory and create a temp file there
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	// Create a temp file in home directory
	testFile := filepath.Join(home, ".start-test-context.md")
	if err := os.WriteFile(testFile, []byte("Tilde context content"), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	defer func() { _ = os.Remove(testFile) }()

	workingDir := t.TempDir()

	// Config uses tilde path
	config := `
		contexts: {
			"tilde-test": {
				file: "~/.start-test-context.md"
				prompt: "Content: {{.file_contents}}"
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compiling config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	result, err := composer.resolveContext(cfg, "tilde-test")
	if err != nil {
		t.Fatalf("resolveContext() error = %v", err)
	}

	if !strings.Contains(result.Content, "Tilde context content") {
		t.Errorf("Content = %q, want to contain %q", result.Content, "Tilde context content")
	}
}

func TestComposer_TildeExpansion_Role(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()

	// Get home directory and create a temp file there
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	// Create a temp file in home directory
	testFile := filepath.Join(home, ".start-test-role.md")
	if err := os.WriteFile(testFile, []byte("Tilde role content"), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	defer func() { _ = os.Remove(testFile) }()

	workingDir := t.TempDir()

	// Config uses tilde path
	config := `
		roles: {
			"tilde-test": {
				file: "~/.start-test-role.md"
				prompt: "Role: {{.file_contents}}"
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compiling config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	content, _, err := composer.resolveRole(cfg, "tilde-test")
	if err != nil {
		t.Fatalf("resolveRole() error = %v", err)
	}

	if !strings.Contains(content, "Tilde role content") {
		t.Errorf("Content = %q, want to contain %q", content, "Tilde role content")
	}
}

func TestComposer_TildeExpansion_Task(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()

	// Get home directory and create a temp file there
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}

	// Create a temp file in home directory
	testFile := filepath.Join(home, ".start-test-task.md")
	if err := os.WriteFile(testFile, []byte("Tilde task content"), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	defer func() { _ = os.Remove(testFile) }()

	workingDir := t.TempDir()

	// Config uses tilde path
	config := `
		tasks: {
			"tilde-test": {
				file: "~/.start-test-task.md"
				prompt: "Task: {{.file_contents}}"
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compiling config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	result, err := composer.ResolveTask(cfg, "tilde-test", "test instructions")
	if err != nil {
		t.Fatalf("ResolveTask() error = %v", err)
	}

	if !strings.Contains(result.Content, "Tilde task content") {
		t.Errorf("Content = %q, want to contain %q", result.Content, "Tilde task content")
	}
}

func TestComposer_TildeExpansion_FileNotFound(t *testing.T) {
	t.Parallel()
	ctx := cuecontext.New()
	workingDir := t.TempDir()

	// Config uses tilde path to nonexistent file
	config := `
		contexts: {
			"missing": {
				file: "~/.start-nonexistent-file-12345.md"
				prompt: "Content: {{.file_contents}}"
			}
		}
	`

	cfg := ctx.CompileString(config)
	if err := cfg.Err(); err != nil {
		t.Fatalf("compiling config: %v", err)
	}

	processor := NewTemplateProcessor(nil, nil, workingDir)
	composer := NewComposer(processor, workingDir)

	_, err := composer.resolveContext(cfg, "missing")
	if err == nil {
		t.Error("expected error for nonexistent tilde path file")
	}
}
