package orchestration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockShellRunner implements ShellRunner for testing.
type mockShellRunner struct {
	output string
	err    error
	calls  []shellCall
}

type shellCall struct {
	command    string
	workingDir string
	shell      string
	timeout    int
}

func (m *mockShellRunner) Run(command, workingDir, shell string, timeout int) (string, error) {
	m.calls = append(m.calls, shellCall{command, workingDir, shell, timeout})
	return m.output, m.err
}

func TestTemplateProcessor_Process(t *testing.T) {
	tests := []struct {
		name         string
		fields       UTDFields
		instructions string
		fileContent  string // content to write to temp file
		shellOutput  string
		wantContains string
		wantErr      bool
		errContains  string
	}{
		{
			name: "simple prompt without placeholders",
			fields: UTDFields{
				Prompt: "Hello, world!",
			},
			wantContains: "Hello, world!",
		},
		{
			name: "prompt with date placeholder",
			fields: UTDFields{
				Prompt: "Today is {{.date}}",
			},
			wantContains: "Today is 20", // Partial match for year prefix
		},
		{
			name: "prompt with instructions placeholder",
			fields: UTDFields{
				Prompt: "Instructions: {{.instructions}}",
			},
			instructions: "focus on error handling",
			wantContains: "Instructions: focus on error handling",
		},
		{
			name: "prompt with file contents placeholder",
			fields: UTDFields{
				Prompt: "File content:\n{{.file_contents}}",
			},
			fileContent:  "This is the file content.",
			wantContains: "File content:\nThis is the file content.",
		},
		{
			name: "prompt with command output placeholder",
			fields: UTDFields{
				Prompt:  "Output: {{.command_output}}",
				Command: "echo hello",
			},
			shellOutput:  "hello\n",
			wantContains: "Output: hello\n",
		},
		{
			name:   "file only - content becomes template",
			fields: UTDFields{
				// File will be set in test
			},
			fileContent:  "Content from file only",
			wantContains: "Content from file only",
		},
		{
			name: "command only - output becomes template",
			fields: UTDFields{
				Command: "echo test",
			},
			shellOutput:  "test output",
			wantContains: "test output",
		},
		{
			name:        "no file, command, or prompt",
			fields:      UTDFields{},
			wantErr:     true,
			errContains: "UTD requires at least one of",
		},
		{
			name: "invalid template syntax",
			fields: UTDFields{
				Prompt: "Bad template {{.Unknown",
			},
			wantErr:     true,
			errContains: "parsing template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create file if content provided
			if tt.fileContent != "" && tt.fields.File == "" {
				filePath := filepath.Join(tmpDir, "test.md")
				if err := os.WriteFile(filePath, []byte(tt.fileContent), 0644); err != nil {
					t.Fatalf("writing test file: %v", err)
				}
				tt.fields.File = filePath
			}

			// Create mock shell runner
			var runner *mockShellRunner
			if tt.shellOutput != "" || tt.fields.Command != "" {
				runner = &mockShellRunner{output: tt.shellOutput}
			}

			processor := NewTemplateProcessor(nil, runner, tmpDir)
			result, err := processor.Process(tt.fields, tt.instructions)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !strings.Contains(result.Content, tt.wantContains) {
				t.Errorf("content = %q, want containing %q", result.Content, tt.wantContains)
			}
		})
	}
}

func TestTemplateProcessor_LazyEvaluation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	filePath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(filePath, []byte("file content"), 0644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	t.Run("file not read when file_contents not used", func(t *testing.T) {
		fields := UTDFields{
			File:   "/nonexistent/file.md", // Would error if read
			Prompt: "Simple prompt without file placeholder",
		}

		processor := NewTemplateProcessor(nil, nil, tmpDir)
		result, err := processor.Process(fields, "")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.FileRead {
			t.Error("file should not have been read")
		}
	})

	t.Run("command not executed when command_output not used", func(t *testing.T) {
		runner := &mockShellRunner{output: "output"}
		fields := UTDFields{
			Command: "echo hello",
			Prompt:  "Simple prompt without command placeholder",
		}

		processor := NewTemplateProcessor(nil, runner, tmpDir)
		result, err := processor.Process(fields, "")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.CommandExecuted {
			t.Error("command should not have been executed")
		}
		if len(runner.calls) > 0 {
			t.Errorf("expected 0 shell calls, got %d", len(runner.calls))
		}
	})

	t.Run("file read when file_contents used", func(t *testing.T) {
		fields := UTDFields{
			File:   filePath,
			Prompt: "Content: {{.file_contents}}",
		}

		processor := NewTemplateProcessor(nil, nil, tmpDir)
		result, err := processor.Process(fields, "")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !result.FileRead {
			t.Error("file should have been read")
		}
		if !strings.Contains(result.Content, "file content") {
			t.Errorf("content = %q, expected to contain file content", result.Content)
		}
	})

	t.Run("command executed when command_output used", func(t *testing.T) {
		runner := &mockShellRunner{output: "hello world"}
		fields := UTDFields{
			Command: "echo hello",
			Prompt:  "Output: {{.command_output}}",
		}

		processor := NewTemplateProcessor(nil, runner, tmpDir)
		result, err := processor.Process(fields, "")

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !result.CommandExecuted {
			t.Error("command should have been executed")
		}
		if len(runner.calls) != 1 {
			t.Errorf("expected 1 shell call, got %d", len(runner.calls))
		}
	})
}

func TestDefaultFileReader_Read(t *testing.T) {
	tmpDir := t.TempDir()
	reader := &DefaultFileReader{}

	t.Run("reads file successfully", func(t *testing.T) {
		filePath := filepath.Join(tmpDir, "test.txt")
		expected := "test content"
		if err := os.WriteFile(filePath, []byte(expected), 0644); err != nil {
			t.Fatalf("writing test file: %v", err)
		}

		content, err := reader.Read(filePath)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if content != expected {
			t.Errorf("content = %q, want %q", content, expected)
		}
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		_, err := reader.Read(filepath.Join(tmpDir, "nonexistent.txt"))
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("expands tilde in path", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("could not get home directory")
		}

		// Create a file in home directory for testing
		testFile := filepath.Join(home, ".start-test-file")
		if err := os.WriteFile(testFile, []byte("home content"), 0644); err != nil {
			t.Skip("could not write test file in home directory")
		}
		defer os.Remove(testFile)

		content, err := reader.Read("~/.start-test-file")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if content != "home content" {
			t.Errorf("content = %q, want %q", content, "home content")
		}
	})
}

func TestIsUTDValid(t *testing.T) {
	tests := []struct {
		name   string
		fields UTDFields
		want   bool
	}{
		{
			name:   "empty fields",
			fields: UTDFields{},
			want:   false,
		},
		{
			name:   "file only",
			fields: UTDFields{File: "test.md"},
			want:   true,
		},
		{
			name:   "command only",
			fields: UTDFields{Command: "echo hello"},
			want:   true,
		},
		{
			name:   "prompt only",
			fields: UTDFields{Prompt: "Hello"},
			want:   true,
		},
		{
			name:   "all fields",
			fields: UTDFields{File: "f", Command: "c", Prompt: "p"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUTDValid(tt.fields); got != tt.want {
				t.Errorf("IsUTDValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
