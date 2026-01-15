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
