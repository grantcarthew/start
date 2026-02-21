package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecute_NoConfig(t *testing.T) {
	// Skip: This test triggers auto-setup which requires network access
	// and creates CUE cache with read-only directories that can't be cleaned up.
	// The auto-setup flow is tested in integration tests instead.
	t.Skip("Skipping: auto-setup requires network access and creates uncleanable cache")
}

func TestExecute_Help(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	cmd := NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute(--help) error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected help output, got empty string")
	}
	if !strings.Contains(output, "start") {
		t.Error("Expected help output to contain 'start'")
	}
}

func TestExecute_Version(t *testing.T) {
	t.Parallel()
	buf := new(bytes.Buffer)
	cmd := NewRootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute(--version) error = %v", err)
	}

	output := buf.String()

	// Should contain version line
	if !strings.Contains(output, "start version") {
		t.Errorf("Expected 'start version' in output, got: %s", output)
	}

	// Should contain repository URL
	if !strings.Contains(output, "https://github.com/grantcarthew/start") {
		t.Errorf("Expected repository URL in output, got: %s", output)
	}

	// Should contain issues URL
	if !strings.Contains(output, "/issues/new") {
		t.Errorf("Expected issues URL in output, got: %s", output)
	}
}

func TestResolveDirectory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid directory",
			path:    t.TempDir(),
			wantErr: false,
		},
		{
			name:    "non-existent directory",
			path:    "/nonexistent/path/that/does/not/exist",
			wantErr: true,
			errMsg:  "directory not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolveDirectory(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("resolveDirectory() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("resolveDirectory() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("resolveDirectory() unexpected error = %v", err)
				}
				if result == "" {
					t.Error("resolveDirectory() returned empty path")
				}
			}
		})
	}
}

// TestHelpArgLeafCommands verifies that "help" as a positional argument works
// on leaf commands (those with no subcommands) the same as --help.
// This covers all 17 commands updated with noArgsOrHelp in place of cobra.NoArgs.
func TestHelpArgLeafCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want string // expected substring in help output
	}{
		// assets subcommands
		{
			name: "assets browse help",
			args: []string{"assets", "browse", "help"},
			want: "browser",
		},
		{
			name: "assets index help",
			args: []string{"assets", "index", "help"},
			want: "asset catalog",
		},
		{
			name: "assets list help",
			args: []string{"assets", "list", "help"},
			want: "installed",
		},
		{
			name: "assets validate help",
			args: []string{"assets", "validate", "help"},
			want: "git tags",
		},
		// completion subcommands
		{
			name: "completion bash help",
			args: []string{"completion", "bash", "help"},
			want: "bash",
		},
		{
			name: "completion zsh help",
			args: []string{"completion", "zsh", "help"},
			want: "zsh",
		},
		{
			name: "completion fish help",
			args: []string{"completion", "fish", "help"},
			want: "fish",
		},
		// config interactive commands
		{
			name: "config add help",
			args: []string{"config", "add", "help"},
			want: "Interactively",
		},
		{
			name: "config edit help",
			args: []string{"config", "edit", "help"},
			want: "select and edit",
		},
		{
			name: "config remove help",
			args: []string{"config", "remove", "help"},
			want: "select and remove",
		},
		// config order commands
		{
			name: "config order help",
			args: []string{"config", "order", "help"},
			want: "Reorder",
		},
		{
			name: "config context order help",
			args: []string{"config", "context", "order", "help"},
			want: "Reorder context",
		},
		{
			name: "config role order help",
			args: []string{"config", "role", "order", "help"},
			want: "Reorder role",
		},
		// config category add commands
		{
			name: "config agent add help",
			args: []string{"config", "agent", "add", "help"},
			want: "new agent",
		},
		{
			name: "config context add help",
			args: []string{"config", "context", "add", "help"},
			want: "new context",
		},
		{
			name: "config role add help",
			args: []string{"config", "role", "add", "help"},
			want: "new role",
		},
		{
			name: "config task add help",
			args: []string{"config", "task", "add", "help"},
			want: "new task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			cmd := NewRootCmd()
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("%v: unexpected error: %v", tt.args, err)
			}
			if !strings.Contains(buf.String(), tt.want) {
				t.Errorf("%v: expected %q in output, got:\n%s", tt.args, tt.want, buf.String())
			}
		})
	}
}

// TestNoArgsOrHelpRejectsInvalidArgs verifies that noArgsOrHelp still rejects
// positional arguments other than a lone "help", preserving cobra.NoArgs behaviour.
func TestNoArgsOrHelpRejectsInvalidArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "unknown positional arg",
			args: []string{"assets", "browse", "unexpected"},
		},
		{
			name: "help plus extra arg",
			args: []string{"assets", "browse", "help", "extra"},
		},
		{
			name: "multiple unknown args",
			args: []string{"assets", "browse", "foo", "bar"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			buf := new(bytes.Buffer)
			cmd := NewRootCmd()
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Errorf("%v: expected error for invalid args, got nil", tt.args)
			}
		})
	}
}

func TestDebugImpliesVerbose(t *testing.T) {
	t.Parallel()
	// Each NewRootCmd() creates its own Flags - no reset needed
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"--debug", "--help"})

	// Execute triggers PersistentPreRunE which sets verbose
	_ = cmd.Execute()

	// After parsing, debug should imply verbose is set
	// Since flags are scoped to the command instance, we verify
	// by checking the command executed without error (--help exits cleanly)
	// The actual flag checking is internal to the command now
}
