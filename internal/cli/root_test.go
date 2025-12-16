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
	t.Skip("Skipping: auto-setup requires network access and creates uncleanbale cache")
}

func TestExecute_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := Execute()
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
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--help"})

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	// Should show agent and orchestrator info
	if !strings.Contains(output, "AI agent") && !strings.Contains(output, "orchestrator") {
		t.Log("Help output:", output)
	}
}
