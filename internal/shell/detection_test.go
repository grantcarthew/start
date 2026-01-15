package shell

import (
	"strings"
	"testing"
)

func TestDetectShell(t *testing.T) {
	shell, err := DetectShell()
	if err != nil {
		t.Fatalf("DetectShell() error = %v", err)
	}

	// Should return a path ending with shell name and -c flag
	if !strings.HasSuffix(shell, " -c") {
		t.Errorf("DetectShell() = %q, want ending with ' -c'", shell)
	}

	// Should contain bash or sh
	if !strings.Contains(shell, "bash") && !strings.Contains(shell, "sh") {
		t.Errorf("DetectShell() = %q, want containing 'bash' or 'sh'", shell)
	}
}

func TestIsShellAvailable(t *testing.T) {
	tests := []struct {
		shell string
		want  bool
	}{
		{"bash", true}, // Usually available
		{"sh", true},   // Always available on Unix
		{"nonexistent_shell_12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			got := IsShellAvailable(tt.shell)
			// bash may not be available on all systems, so we skip if it fails
			if tt.shell == "bash" && !got {
				t.Skip("bash not available on this system")
			}
			if got != tt.want {
				t.Errorf("IsShellAvailable(%q) = %v, want %v", tt.shell, got, tt.want)
			}
		})
	}
}
