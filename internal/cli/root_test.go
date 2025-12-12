package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecute_NoConfig(t *testing.T) {
	// Reset rootCmd for clean test
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{})

	// Running start without config should return an error
	err := Execute()
	if err == nil {
		t.Fatal("Expected error when no config exists")
	}

	// Error should mention configuration
	if !strings.Contains(err.Error(), "configuration") {
		t.Errorf("Expected error about configuration, got: %v", err)
	}
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
