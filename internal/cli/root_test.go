package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	// Capture output by setting buffers on rootCmd
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{})

	// Call the exported Execute function
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
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
