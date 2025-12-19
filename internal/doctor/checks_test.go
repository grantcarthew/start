package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grantcarthew/start/internal/config"
)

func TestCheckIntro(t *testing.T) {
	section := CheckIntro()

	if section.Name != "Repository" {
		t.Errorf("CheckIntro().Name = %q, want %q", section.Name, "Repository")
	}
	if !section.NoIcons {
		t.Error("CheckIntro().NoIcons should be true")
	}
	if len(section.Results) != 2 {
		t.Fatalf("CheckIntro() should have 2 results, got %d", len(section.Results))
	}
	if section.Results[0].Label != RepoURL {
		t.Errorf("First result should be repo URL, got %q", section.Results[0].Label)
	}
	if section.Results[1].Label != IssuesURL {
		t.Errorf("Second result should be issues URL, got %q", section.Results[1].Label)
	}
}

func TestCheckVersion(t *testing.T) {
	info := BuildInfo{
		Version:   "v1.0.0",
		Commit:    "abc123",
		BuildDate: "2025-01-01",
		GoVersion: "go1.23.0",
		Platform:  "linux/amd64",
	}

	section := CheckVersion(info)

	if section.Name != "Version" {
		t.Errorf("CheckVersion().Name = %q, want %q", section.Name, "Version")
	}
	if !section.NoIcons {
		t.Error("CheckVersion().NoIcons should be true")
	}
	if len(section.Results) != 5 {
		t.Fatalf("CheckVersion() should have 5 results, got %d", len(section.Results))
	}

	// Check version label includes version
	if section.Results[0].Label != "start v1.0.0" {
		t.Errorf("Version label = %q, want %q", section.Results[0].Label, "start v1.0.0")
	}
}

func TestCheckConfiguration_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	paths := config.Paths{
		Global:       filepath.Join(tmpDir, "global"),
		Local:        filepath.Join(tmpDir, "local"),
		GlobalExists: false,
		LocalExists:  false,
	}

	section := CheckConfiguration(paths)

	if section.Name != "Configuration" {
		t.Errorf("CheckConfiguration().Name = %q, want %q", section.Name, "Configuration")
	}

	// Should have 2 results (global not found, local not found)
	if len(section.Results) != 2 {
		t.Fatalf("CheckConfiguration() should have 2 results, got %d", len(section.Results))
	}

	// Both should be info status with "Not found" message
	for _, r := range section.Results {
		if r.Status != StatusInfo {
			t.Errorf("Result status should be StatusInfo, got %v", r.Status)
		}
		if r.Message != "Not found" {
			t.Errorf("Result message should be 'Not found', got %q", r.Message)
		}
	}
}

func TestCheckConfiguration_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write valid CUE file
	cueContent := `settings: { default_agent: "test" }`
	if err := os.WriteFile(filepath.Join(globalDir, "settings.cue"), []byte(cueContent), 0644); err != nil {
		t.Fatal(err)
	}

	paths := config.Paths{
		Global:       globalDir,
		Local:        filepath.Join(tmpDir, "local"),
		GlobalExists: true,
		LocalExists:  false,
	}

	section := CheckConfiguration(paths)

	// Should have results for global (header + file), local, and validation
	hasPass := false
	for _, r := range section.Results {
		if r.Status == StatusPass {
			hasPass = true
		}
	}
	if !hasPass {
		t.Error("Valid config should have at least one StatusPass result")
	}
}

func TestCheckConfiguration_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write invalid CUE file
	cueContent := `this is not valid cue {{{`
	if err := os.WriteFile(filepath.Join(globalDir, "bad.cue"), []byte(cueContent), 0644); err != nil {
		t.Fatal(err)
	}

	paths := config.Paths{
		Global:       globalDir,
		Local:        filepath.Join(tmpDir, "local"),
		GlobalExists: true,
		LocalExists:  false,
	}

	section := CheckConfiguration(paths)

	// Should have a failure result
	hasFail := false
	for _, r := range section.Results {
		if r.Status == StatusFail {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("Invalid config should have StatusFail result")
	}
}

func TestCheckEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}

	paths := config.Paths{
		Global:       globalDir,
		Local:        filepath.Join(tmpDir, "local"),
		GlobalExists: true,
		LocalExists:  false,
	}

	section := CheckEnvironment(paths)

	if section.Name != "Environment" {
		t.Errorf("CheckEnvironment().Name = %q, want %q", section.Name, "Environment")
	}

	// Should have results for config directory and working directory
	if len(section.Results) < 2 {
		t.Errorf("CheckEnvironment() should have at least 2 results, got %d", len(section.Results))
	}

	// Config directory should be writable (we just created it)
	hasWritable := false
	for _, r := range section.Results {
		if r.Label == "Config directory" && r.Status == StatusPass {
			hasWritable = true
		}
	}
	if !hasWritable {
		t.Error("Config directory should be writable")
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := expandPath(tt.input); got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		input string
		want  string
	}{
		{filepath.Join(home, "test"), "~/test"},
		{"/other/path", "/other/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := shortenPath(tt.input); got != tt.want {
				t.Errorf("shortenPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
