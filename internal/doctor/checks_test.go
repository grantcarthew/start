package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/grantcarthew/start/internal/config"
)

func TestCheckIntro(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	info := BuildInfo{
		Version:      "v1.0.0",
		Commit:       "abc123",
		BuildDate:    "2025-01-01",
		GoVersion:    "go1.23.0",
		Platform:     "linux/amd64",
		IndexVersion: "v0.3.2",
	}

	section := CheckVersion(info)

	if section.Name != "Version" {
		t.Errorf("CheckVersion().Name = %q, want %q", section.Name, "Version")
	}
	if !section.NoIcons {
		t.Error("CheckVersion().NoIcons should be true")
	}
	if len(section.Results) != 6 {
		t.Fatalf("CheckVersion() should have 6 results, got %d", len(section.Results))
	}

	// Check version label includes version
	if section.Results[0].Label != "start v1.0.0" {
		t.Errorf("Version label = %q, want %q", section.Results[0].Label, "start v1.0.0")
	}

	// Check index version
	indexResult := section.Results[5]
	if indexResult.Label != "Index" {
		t.Errorf("Index label = %q, want %q", indexResult.Label, "Index")
	}
	if indexResult.Message != "v0.3.2" {
		t.Errorf("Index message = %q, want %q", indexResult.Message, "v0.3.2")
	}
	if indexResult.Status != StatusInfo {
		t.Errorf("Index status = %v, want StatusInfo", indexResult.Status)
	}
}

func TestCheckVersion_IndexUnavailable(t *testing.T) {
	t.Parallel()
	info := BuildInfo{
		Version:   "v1.0.0",
		Commit:    "abc123",
		BuildDate: "2025-01-01",
		GoVersion: "go1.23.0",
		Platform:  "linux/amd64",
	}

	section := CheckVersion(info)

	if len(section.Results) != 6 {
		t.Fatalf("CheckVersion() should have 6 results, got %d", len(section.Results))
	}

	indexResult := section.Results[5]
	if indexResult.Status != StatusWarn {
		t.Errorf("Index status = %v, want StatusWarn", indexResult.Status)
	}
	if indexResult.Message != "unavailable" {
		t.Errorf("Index message = %q, want %q", indexResult.Message, "unavailable")
	}
}

func TestCheckVersion_WithCustomIndexPath(t *testing.T) {
	t.Parallel()
	info := BuildInfo{
		Version:      "v1.0.0",
		Commit:       "abc123",
		BuildDate:    "2025-01-01",
		GoVersion:    "go1.23.0",
		Platform:     "linux/amd64",
		IndexVersion: "v0.3.2",
		IndexPath:    "github.com/example/custom-assets/index@v0",
	}

	section := CheckVersion(info)

	var sourceResult *CheckResult
	for i := range section.Results {
		if section.Results[i].Label == "Index Source" {
			sourceResult = &section.Results[i]
			break
		}
	}
	if sourceResult == nil {
		t.Fatal("CheckVersion() missing 'Index Source' result")
	}
	if sourceResult.Message != "github.com/example/custom-assets/index@v0" {
		t.Errorf("Index Source message = %q, want %q", sourceResult.Message, "github.com/example/custom-assets/index@v0")
	}
	if sourceResult.Status != StatusInfo {
		t.Errorf("Index Source status = %v, want StatusInfo", sourceResult.Status)
	}
}

func TestCheckVersion_NoIndexPath(t *testing.T) {
	t.Parallel()
	info := BuildInfo{
		Version:      "v1.0.0",
		Commit:       "abc123",
		BuildDate:    "2025-01-01",
		GoVersion:    "go1.23.0",
		Platform:     "linux/amd64",
		IndexVersion: "v0.3.2",
		// IndexPath not set — default behaviour
	}

	section := CheckVersion(info)

	for _, r := range section.Results {
		if r.Label == "Index Source" {
			t.Errorf("CheckVersion() without IndexPath should not include 'Index Source' result")
		}
	}
}

func TestCheckConfiguration_NoConfig(t *testing.T) {
	t.Parallel()
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

	// Both should be info status with message containing "Not found"
	for _, r := range section.Results {
		if r.Status != StatusInfo {
			t.Errorf("Result status should be StatusInfo, got %v", r.Status)
		}
		if !strings.Contains(r.Message, "Not found") {
			t.Errorf("Result message should contain 'Not found', got %q", r.Message)
		}
	}
}

func TestCheckConfiguration_ValidConfig(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// --- CheckAgents tests ---

func TestCheckAgents_NoneConfigured(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString("{}")

	section := CheckAgents(v)

	if section.Name != "Agents" {
		t.Errorf("Name = %q, want %q", section.Name, "Agents")
	}
	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusInfo {
		t.Errorf("status = %v, want StatusInfo", section.Results[0].Status)
	}
	if section.Results[0].Label != "None configured" {
		t.Errorf("label = %q, want %q", section.Results[0].Label, "None configured")
	}
}

func TestCheckAgents_ValidBinary(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`agents: { myagent: { bin: "go" } }`)

	section := CheckAgents(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
	if section.Results[0].Label != "myagent" {
		t.Errorf("label = %q, want %q", section.Results[0].Label, "myagent")
	}
	if section.Summary != "1 configured" {
		t.Errorf("summary = %q, want %q", section.Summary, "1 configured")
	}
}

func TestCheckAgents_MissingBinary(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`agents: { broken: { bin: "nonexistent-binary-xyz-123" } }`)

	section := CheckAgents(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusFail {
		t.Errorf("status = %v, want StatusFail", section.Results[0].Status)
	}
	if section.Results[0].Fix == "" {
		t.Error("expected a fix suggestion for missing binary")
	}
}

func TestCheckAgents_NoBinField(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`agents: { nobin: { description: "no bin" } }`)

	section := CheckAgents(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusWarn {
		t.Errorf("status = %v, want StatusWarn", section.Results[0].Status)
	}
	if section.Results[0].Message != "No bin field" {
		t.Errorf("message = %q, want %q", section.Results[0].Message, "No bin field")
	}
}

// --- CheckContexts tests ---

func TestCheckContexts_NoneConfigured(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString("{}")

	section := CheckContexts(v)

	if section.Name != "Contexts" {
		t.Errorf("Name = %q, want %q", section.Name, "Contexts")
	}
	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Label != "None configured" {
		t.Errorf("label = %q, want %q", section.Results[0].Label, "None configured")
	}
}

func TestCheckContexts_FileExists(t *testing.T) {
	t.Parallel()
	tmpFile := filepath.Join(t.TempDir(), "context.md")
	if err := os.WriteFile(tmpFile, []byte("context content"), 0644); err != nil {
		t.Fatal(err)
	}

	cctx := cuecontext.New()
	v := cctx.CompileString(`contexts: { myctx: { file: "` + tmpFile + `" } }`)

	section := CheckContexts(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
}

func TestCheckContexts_FileMissingRequired(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`contexts: { reqctx: { file: "/nonexistent/path/file.md", required: true } }`)

	section := CheckContexts(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusFail {
		t.Errorf("status = %v, want StatusFail for required missing file", section.Results[0].Status)
	}
}

func TestCheckContexts_FileMissingOptional(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`contexts: { optctx: { file: "/nonexistent/path/file.md", required: false } }`)

	section := CheckContexts(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusWarn {
		t.Errorf("status = %v, want StatusWarn for optional missing file", section.Results[0].Status)
	}
}

func TestCheckContexts_InlinePrompt(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`contexts: { promptctx: { prompt: "You are helpful" } }`)

	section := CheckContexts(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
	if section.Results[0].Message != "(inline prompt)" {
		t.Errorf("message = %q, want %q", section.Results[0].Message, "(inline prompt)")
	}
}

func TestCheckContexts_Command(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`contexts: { cmdctx: { command: "echo hello" } }`)

	section := CheckContexts(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
	if section.Results[0].Message != "(command)" {
		t.Errorf("message = %q, want %q", section.Results[0].Message, "(command)")
	}
}

// --- CheckRoles tests ---

func TestCheckRoles_NoneConfigured(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString("{}")

	section := CheckRoles(v)

	if section.Name != "Roles" {
		t.Errorf("Name = %q, want %q", section.Name, "Roles")
	}
	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Label != "None configured" {
		t.Errorf("label = %q, want %q", section.Results[0].Label, "None configured")
	}
}

func TestCheckRoles_FileExists(t *testing.T) {
	t.Parallel()
	tmpFile := filepath.Join(t.TempDir(), "role.md")
	if err := os.WriteFile(tmpFile, []byte("role content"), 0644); err != nil {
		t.Fatal(err)
	}

	cctx := cuecontext.New()
	v := cctx.CompileString(`roles: { myrole: { file: "` + tmpFile + `" } }`)

	section := CheckRoles(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
}

func TestCheckRoles_FileMissing(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`roles: { badrole: { file: "/nonexistent/path/role.md" } }`)

	section := CheckRoles(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	// Roles do NOT downgrade to warn - missing file stays as StatusFail
	if section.Results[0].Status != StatusFail {
		t.Errorf("status = %v, want StatusFail (roles don't downgrade)", section.Results[0].Status)
	}
}

func TestCheckRoles_PromptFallback(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`roles: { prole: { prompt: "You are a code reviewer" } }`)

	section := CheckRoles(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
	if section.Results[0].Message != "(inline prompt)" {
		t.Errorf("message = %q, want %q", section.Results[0].Message, "(inline prompt)")
	}
}

func TestCheckRoles_NoFileOrPrompt(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`roles: { emptyrole: { description: "nothing useful" } }`)

	section := CheckRoles(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusWarn {
		t.Errorf("status = %v, want StatusWarn", section.Results[0].Status)
	}
	if section.Results[0].Message != "No file, prompt, or command" {
		t.Errorf("message = %q, want %q", section.Results[0].Message, "No file, prompt, or command")
	}
}

// --- CheckTasks tests ---

func TestCheckTasks_NoneConfigured(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString("{}")

	section := CheckTasks(v)

	if section.Name != "Tasks" {
		t.Errorf("Name = %q, want %q", section.Name, "Tasks")
	}
	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Label != "None configured" {
		t.Errorf("label = %q, want %q", section.Results[0].Label, "None configured")
	}
}

func TestCheckTasks_FileExists(t *testing.T) {
	t.Parallel()
	tmpFile := filepath.Join(t.TempDir(), "task.md")
	if err := os.WriteFile(tmpFile, []byte("task content"), 0644); err != nil {
		t.Fatal(err)
	}

	cctx := cuecontext.New()
	v := cctx.CompileString(`tasks: { mytask: { file: "` + tmpFile + `" } }`)

	section := CheckTasks(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
	if section.Summary != "1 configured" {
		t.Errorf("summary = %q, want %q", section.Summary, "1 configured")
	}
}

func TestCheckTasks_InlinePrompt(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`tasks: { prompttask: { prompt: "Do the thing" } }`)

	section := CheckTasks(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
	if section.Results[0].Message != "(inline prompt)" {
		t.Errorf("message = %q, want %q", section.Results[0].Message, "(inline prompt)")
	}
}

func TestCheckTasks_ModulePath(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`tasks: { modtask: { file: "@module/task.md" } }`)

	section := CheckTasks(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusPass {
		t.Errorf("status = %v, want StatusPass", section.Results[0].Status)
	}
	if section.Results[0].Message != "(registry module)" {
		t.Errorf("message = %q, want %q", section.Results[0].Message, "(registry module)")
	}
}

func TestCheckTasks_FileMissing(t *testing.T) {
	t.Parallel()
	cctx := cuecontext.New()
	v := cctx.CompileString(`tasks: { badtask: { file: "/nonexistent/path/task.md" } }`)

	section := CheckTasks(v)

	if len(section.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(section.Results))
	}
	if section.Results[0].Status != StatusFail {
		t.Errorf("status = %v, want StatusFail", section.Results[0].Status)
	}
}
