package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// setupReadTestConfig writes a CUE config covering each read code path.
func setupReadTestConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Isolate from the user's real config and registry cache.
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	// CUE module cache writes read-only files; chmod before TempDir cleanup.
	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return os.Chmod(path, 0o755)
			}
			return os.Chmod(path, 0o644)
		})
	})

	startDir := filepath.Join(dir, ".start")
	if err := os.MkdirAll(startDir, 0o755); err != nil {
		t.Fatalf("creating .start dir: %v", err)
	}

	// File-source role (real on-disk file, expanded via DefaultFileReader).
	roleFile := filepath.Join(dir, "role.md")
	if err := os.WriteFile(roleFile, []byte("Role file contents.\n"), 0o644); err != nil {
		t.Fatalf("writing role file: %v", err)
	}

	// File+prompt UTD: file must win over prompt for read.
	mixedFile := filepath.Join(dir, "mixed.md")
	if err := os.WriteFile(mixedFile, []byte("MIXED FILE CONTENT"), 0o644); err != nil {
		t.Fatalf("writing mixed file: %v", err)
	}

	// Tilde-path UTD: referenced as "~/tilde-role.md" in CUE; lives at $HOME
	// (== dir, set above). Exercises ExpandFilePath via DefaultFileReader.
	tildeFile := filepath.Join(dir, "tilde-role.md")
	if err := os.WriteFile(tildeFile, []byte("Tilde file contents."), 0o644); err != nil {
		t.Fatalf("writing tilde file: %v", err)
	}

	// Origin-bearing role: a fake origin string is fine — printReadVerbose only
	// reads the value from CUE; @module/ resolution is not triggered for an
	// absolute file path.
	tracedFile := filepath.Join(dir, "traced.md")
	if err := os.WriteFile(tracedFile, []byte("traced contents"), 0o644); err != nil {
		t.Fatalf("writing traced file: %v", err)
	}

	// Relative-path role: file written into dir (the test's cwd after chdir
	// below), referenced as "./relative-role.md" in CUE. Exercises
	// ExpandFilePath's filepath.Abs branch via DefaultFileReader.
	relativeFile := filepath.Join(dir, "relative-role.md")
	if err := os.WriteFile(relativeFile, []byte("relative contents"), 0o644); err != nil {
		t.Fatalf("writing relative file: %v", err)
	}

	// File whose content references {{.command_output}}, paired with a non-empty
	// command in CUE. Used to assert readUTD's trim block suppresses
	// TemplateProcessor.Process's lazy command execution
	// (template.go: needsCommandOutput && fields.Command != "").
	fcCmdRefFile := filepath.Join(dir, "fc-cmd-ref.md")
	if err := os.WriteFile(fcCmdRefFile, []byte("before {{.command_output}} after"), 0o644); err != nil {
		t.Fatalf("writing fc-cmd-ref file: %v", err)
	}

	cueConfig := `
agents: {
	claude: {
		bin:           "claude"
		command:       "{{.bin}} --model {{.model}} '{{.prompt}}'"
		description:   "Claude"
		default_model: "sonnet"
		models: {
			sonnet: "claude-sonnet-4"
			haiku:  "claude-haiku-4"
		}
	}
	bare: {
		bin:         "bare"
		description: "Agent with no command"
	}
}

roles: {
	"role-file": {
		description: "File-source role"
		file:        "` + roleFile + `"
	}
	"role-prompt": {
		description: "Prompt-source role"
		prompt:      "Hello {{.user}}"
	}
	"role-mixed": {
		description: "File and prompt; file should win"
		file:        "` + mixedFile + `"
		prompt:      "PROMPT WINS"
	}
	"role-empty": {
		description: "No source fields"
	}
	"role-tilde": {
		description: "Tilde-path file source"
		file:        "~/tilde-role.md"
	}
	"role-relative": {
		description: "Relative-path file source"
		file:        "./relative-role.md"
	}
	"pc-priority": {
		description: "Prompt and command; prompt should win"
		prompt:      "PROMPT VALUE"
		command:     "echo COMMAND VALUE"
	}
	"role-module-no-origin": {
		description: "@module/ file path without origin (error guard)"
		file:        "@module/anywhere.md"
	}
	"role-traced": {
		description: "File source with origin (verbose metadata)"
		file:        "` + tracedFile + `"
		origin:      "github.com/example/start-assets/roles/traced@v1.2.3"
	}
	"fc-cmd-ref": {
		description: "File source whose content references {{.command_output}}; command must not run"
		file:        "` + fcCmdRefFile + `"
		command:     "echo SHOULD-NOT-APPEAR"
	}
	"pc-cmd-ref": {
		description: "Prompt referencing {{.command_output}} with command; command must not run"
		prompt:      "before {{.command_output}} after"
		command:     "echo SHOULD-NOT-APPEAR"
	}
}

contexts: {
	"ctx-cmd": {
		description: "Command-source context with custom shell and timeout"
		command:     "printf 'cmd-output'"
		shell:       "bash -c"
		timeout:     5
	}
}

tasks: {
	"task-prompt": {
		description: "Prompt-source task"
		prompt:      "Task body"
	}
	"task-cmd": {
		description: "Command-source task"
		command:     "printf 'task-cmd-output'"
	}
}
`

	if err := os.WriteFile(filepath.Join(startDir, "settings.cue"), []byte(cueConfig), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	chdir(t, dir)
	return dir
}

// runReadCmd runs `start read` with the given args and a non-TTY stdin.
// Returns stdout, stderr, and any error from cmd.Execute().
func runReadCmd(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(append([]string{"read"}, args...))
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// TestReadUTDPromptSource verifies read renders a prompt-source role and the
// rendered template variables (e.g. {{.user}}) are substituted.
func TestReadUTDPromptSource(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "role-prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	if !strings.HasPrefix(stdout, "Hello ") {
		t.Errorf("expected stdout to start with rendered prompt, got: %q", stdout)
	}
	if strings.Contains(stdout, "{{.user}}") {
		t.Errorf("template placeholder should be substituted, got: %q", stdout)
	}
}

// TestReadUTDFileSource verifies a file-source role outputs the file contents.
func TestReadUTDFileSource(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "role-file")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	if stdout != "Role file contents.\n" {
		t.Errorf("stdout = %q, want %q", stdout, "Role file contents.\n")
	}
}

// TestReadUTDFileWinsOverPrompt verifies the file > prompt > command priority:
// when both file and prompt are defined, read outputs the file.
func TestReadUTDFileWinsOverPrompt(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "role-mixed")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "MIXED FILE CONTENT") {
		t.Errorf("expected file content in stdout, got: %q", stdout)
	}
	if strings.Contains(stdout, "PROMPT WINS") {
		t.Errorf("prompt should not appear when file is set, got: %q", stdout)
	}
}

// TestReadUTDCommandSource verifies a command-source UTD asset executes the
// command and that custom shell/timeout flow through to the runner. The trim
// block in readUTD must preserve Shell and Timeout — they are execution
// config, not source fields.
func TestReadUTDCommandSourceWithShellTimeout(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "ctx-cmd")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	if stdout != "cmd-output\n" {
		t.Errorf("stdout = %q, want %q", stdout, "cmd-output\n")
	}
}

// TestReadAgent verifies an agent's command template is partially rendered:
// {{.bin}} and {{.model}} are substituted; runtime placeholders remain.
func TestReadAgent(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "claude")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "claude --model claude-sonnet-4") {
		t.Errorf("expected resolved bin and model, got: %q", stdout)
	}
	if !strings.Contains(stdout, "{{.prompt}}") {
		t.Errorf("runtime {{.prompt}} placeholder should remain, got: %q", stdout)
	}
}

// TestReadAgentNoCommand verifies an agent with no command field returns a
// configuration error naming the agent and leaves stdout empty.
func TestReadAgentNoCommand(t *testing.T) {
	setupReadTestConfig(t)

	stdout, _, err := runReadCmd(t, "bare")
	if err == nil {
		t.Fatal("expected error for agent with no command field")
	}
	if !strings.Contains(err.Error(), "bare") {
		t.Errorf("error should name the agent, got: %v", err)
	}
	if !strings.Contains(err.Error(), "command") {
		t.Errorf("error should mention command field, got: %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty on error, got: %q", stdout)
	}
}

// TestReadUTDEmptyFields verifies a UTD asset with no file, prompt, or command
// returns a configuration error naming the asset and listing the expected
// fields. Stdout stays empty.
func TestReadUTDEmptyFields(t *testing.T) {
	setupReadTestConfig(t)

	stdout, _, err := runReadCmd(t, "role-empty")
	if err == nil {
		t.Fatal("expected error for UTD asset with no source fields")
	}
	if !strings.Contains(err.Error(), "role-empty") {
		t.Errorf("error should name the asset, got: %v", err)
	}
	for _, want := range []string{"file", "prompt", "command"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should list %q as expected field, got: %v", want, err)
		}
	}
	if stdout != "" {
		t.Errorf("stdout should be empty on error, got: %q", stdout)
	}
}

// TestReadNoArgNonTTY verifies that running `read` with no argument in a
// non-interactive environment returns an error rather than blocking on a
// prompt.
func TestReadNoArgNonTTY(t *testing.T) {
	setupReadTestConfig(t)

	_, _, err := runReadCmd(t)
	if err == nil {
		t.Fatal("expected error for no argument in non-TTY mode")
	}
	if !strings.Contains(err.Error(), "non-interactive") {
		t.Errorf("error should mention non-interactive mode, got: %v", err)
	}
}

// TestReadAmbiguousNonTTY verifies that an ambiguous name in non-TTY mode
// returns an error listing the candidate matches.
func TestReadAmbiguousNonTTY(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return os.Chmod(path, 0o755)
			}
			return os.Chmod(path, 0o644)
		})
	})

	startDir := filepath.Join(dir, ".start")
	if err := os.MkdirAll(startDir, 0o755); err != nil {
		t.Fatalf("creating .start dir: %v", err)
	}

	cueConfig := `
roles: {
	helper: {
		prompt: "role helper"
	}
}
tasks: {
	helper: {
		prompt: "task helper"
	}
}
`
	if err := os.WriteFile(filepath.Join(startDir, "settings.cue"), []byte(cueConfig), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	chdir(t, dir)

	_, _, err := runReadCmd(t, "helper")
	if err == nil {
		t.Fatal("expected ambiguity error in non-TTY mode")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error should mention ambiguity, got: %v", err)
	}
	for _, want := range []string{"roles/helper", "tasks/helper"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should list candidate %q, got: %v", want, err)
		}
	}
}

// TestReadVerboseCommandSource verifies --verbose against a command-source
// UTD asset emits a "Command: ..." line on stderr alongside Type/Name.
// Without this metadata, a user piping `start read --verbose ctx-cmd | ...`
// has no visibility into the shell-out that produced stdout.
func TestReadVerboseCommandSource(t *testing.T) {
	setupReadTestConfig(t)

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--verbose", "read", "ctx-cmd"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	stderrStr := stderr.String()
	for _, want := range []string{"Type: Context", "Name: ctx-cmd", "Command: printf 'cmd-output'"} {
		if !strings.Contains(stderrStr, want) {
			t.Errorf("stderr missing %q\ngot: %s", want, stderrStr)
		}
	}
	if stdout.String() != "cmd-output\n" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "cmd-output\n")
	}
}

// TestReadVerboseToStderr verifies --verbose writes metadata to stderr without
// polluting stdout.
func TestReadVerboseToStderr(t *testing.T) {
	setupReadTestConfig(t)

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--verbose", "read", "role-prompt"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	stderrStr := stderr.String()
	for _, want := range []string{"Type: Role", "Name: role-prompt"} {
		if !strings.Contains(stderrStr, want) {
			t.Errorf("stderr missing %q\ngot: %s", want, stderrStr)
		}
	}

	stdoutStr := stdout.String()
	for _, banned := range []string{"Type:", "Name:"} {
		if strings.Contains(stdoutStr, banned) {
			t.Errorf("stdout should not contain %q metadata, got: %q", banned, stdoutStr)
		}
	}
	if !strings.HasPrefix(stdoutStr, "Hello ") {
		t.Errorf("stdout should still contain rendered content, got: %q", stdoutStr)
	}
}

// TestReadQuietSuppressesStderr verifies that --quiet leaves stdout holding
// only the asset content with stderr empty. Three independent stderr-write
// paths converge in runRead and read* helpers — autoInstall progress
// (resolve.go), notifyScopeWidenedIfLocal (show.go), and printReadVerbose
// (read.go) — and a regression in any single Quiet/Verbose gate would leak
// metadata into a `start read --quiet | bar` pipeline. The autoInstall arm
// of that contract is unit-tested in resolve.go's tests; the widen-notice
// arm in TestNotifyScopeWidenedIfLocal. This test covers the verbose-path
// gate and the integration shape (no flag combination produces stderr on the
// happy path).
func TestReadQuietSuppressesStderr(t *testing.T) {
	setupReadTestConfig(t)

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--quiet", "read", "role-prompt"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	if stderr.Len() != 0 {
		t.Errorf("--quiet must leave stderr empty on happy path, got: %q", stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "Hello ") {
		t.Errorf("stdout should carry rendered content, got: %q", stdout.String())
	}
}

// TestReadResolveQueryRoutesToStderr asserts the wiring contract from the
// implementation plan: `read` must invoke promptSearchQuery with stderr (not
// stdout) and emit the short-query fallback to stderr. This keeps `start read
// | bar` pipe-clean when stdin is a TTY but stdout is piped.
func TestReadResolveQueryRoutesToStderr(t *testing.T) {
	setupReadTestConfig(t)

	t.Run("no-arg non-TTY surfaces error without writing stderr", func(t *testing.T) {
		stderr := new(bytes.Buffer)
		_, err := readResolveQuery(nil, stderr, strings.NewReader(""))
		if err == nil {
			t.Fatal("expected error for no-arg non-TTY")
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr should be untouched on non-TTY error path, got: %q", stderr.String())
		}
	})

	t.Run("short-query non-TTY surfaces error without writing stderr", func(t *testing.T) {
		stderr := new(bytes.Buffer)
		_, err := readResolveQuery([]string{"ab"}, stderr, strings.NewReader(""))
		if err == nil {
			t.Fatal("expected error for short query in non-TTY")
		}
		if !strings.Contains(err.Error(), "3 characters") {
			t.Errorf("error should mention minimum length, got: %v", err)
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr should be untouched on non-TTY error path, got: %q", stderr.String())
		}
	})
}

// TestReadCommandHelp verifies `start read help` prints the command help and
// that the read command registers --global plus inherits --local from root.
// Help-string assertions are limited to text that only lives in help (the
// stdout-routing contract, the start-show pointer, the auto-install widening
// note); flag presence is asserted via direct flag lookup so cosmetic
// help-formatter changes (heading order, line wrapping, colour) cannot
// false-fail it. Source priority is pinned by TestReadUTDFileWinsOverPrompt
// and TestReadUTDPromptWinsOverCommand, so help wording is not the place to
// re-assert it. No config isolation is needed: `help` short-circuits in
// checkHelpArg before any config is loaded.
func TestReadCommandHelp(t *testing.T) {
	stdout, _, err := runReadCmd(t, "help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"read", "stdout", "start show", "Auto-installed"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("help output missing %q\ngot: %s", want, stdout)
		}
	}

	root := NewRootCmd()
	var readCmd *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "read" {
			readCmd = c
			break
		}
	}
	if readCmd == nil {
		t.Fatal("read command not registered on root")
	}
	if readCmd.Flag("global") == nil {
		t.Error("read command missing --global flag")
	}
	if readCmd.Flag("local") == nil {
		t.Error("read command missing inherited --local flag (expected via root persistent flags)")
	}
}

// TestReadAppearsInRootHelp verifies the read command is registered on the
// root with GroupID "commands" so it lands in the Commands section of help
// output. Asserting the structural property avoids fragility against Cobra
// help-formatter changes (heading order, colour codes, line wrapping).
func TestReadAppearsInRootHelp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cmd := NewRootCmd()
	for _, c := range cmd.Commands() {
		if c.Name() == "read" {
			if c.GroupID != "commands" {
				t.Errorf("read.GroupID = %q, want %q", c.GroupID, "commands")
			}
			return
		}
	}
	t.Fatal("read command not registered on root")
}

// TestReadUnknownName verifies that a name with no installed or registry
// matches surfaces a clear error and leaves stdout empty. Acceptance criterion
// "Unknown asset names produce a clear error".
func TestReadUnknownName(t *testing.T) {
	setupReadTestConfig(t)

	stdout, _, err := runReadCmd(t, "definitely-not-a-real-asset-zzz")
	if err == nil {
		t.Fatal("expected error for unknown asset name")
	}
	if !strings.Contains(err.Error(), "definitely-not-a-real-asset-zzz") {
		t.Errorf("error should name the missing asset, got: %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty on error, got: %q", stdout)
	}
}

// TestReadUTDTildePath verifies that a UTD asset whose `file` field uses a
// `~/`-prefixed path resolves through DefaultFileReader's tilde expansion and
// outputs the file's contents. Acceptance criterion: "UTD file resolution
// succeeds for @module/, ~/, and relative paths".
func TestReadUTDTildePath(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "role-tilde")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if stdout != "Tilde file contents.\n" {
		t.Errorf("stdout = %q, want %q", stdout, "Tilde file contents.\n")
	}
}

// TestReadVerboseFileAndOrigin verifies that --verbose against a UTD asset
// with both `file` and `origin` emits Type, Name, Origin, and Path metadata
// lines to stderr, while stdout still receives the raw file contents.
func TestReadVerboseFileAndOrigin(t *testing.T) {
	setupReadTestConfig(t)

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--verbose", "read", "role-traced"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	stderrStr := stderr.String()
	wants := []string{
		"Type: Role",
		"Name: role-traced",
		"Origin: github.com/example/start-assets/roles/traced@v1.2.3",
		"Path: ", // Path: <absolute> — only assert the prefix; absolute path varies by tempdir
	}
	for _, want := range wants {
		if !strings.Contains(stderrStr, want) {
			t.Errorf("stderr missing %q\ngot: %s", want, stderrStr)
		}
	}
	// Path line should reference the actual file basename so we know it's the
	// resolved path, not the literal config string.
	if !strings.Contains(stderrStr, "traced.md") {
		t.Errorf("stderr Path line missing resolved file name 'traced.md'\ngot: %s", stderrStr)
	}

	if stdout.String() != "traced contents\n" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "traced contents\n")
	}
}

// TestReadShortQueryNonTTYEndToEnd is the cobra-level counterpart to the
// readResolveQuery unit test: a short query in a non-TTY environment must
// return the descriptive error and never write to stdout. Together with
// TestReadResolveQueryRoutesToStderr this proves the runRead → readResolveQuery
// wiring keeps stdout pipe-clean on the failure path. The TTY-mode re-prompt
// on stderr is not covered here because the project has no pseudo-TTY helpers
// (see project plan, Implementation Guidance).
func TestReadShortQueryNonTTYEndToEnd(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "ab")
	if err == nil {
		t.Fatal("expected error for short query in non-TTY")
	}
	if !strings.Contains(err.Error(), "3 characters") {
		t.Errorf("error should mention minimum length, got: %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on short-query failure, got: %q", stdout)
	}
	// Sanity check that the non-TTY path did not also dump the TTY-mode
	// "Query must be at least 3 characters" line to stderr — that fallback is
	// only meaningful when the user can be re-prompted. Failing to gate it
	// would clutter scripted callers' error output.
	if strings.Contains(stderr, "Query must be at least 3 characters") {
		t.Errorf("non-TTY path should not emit TTY re-prompt notice, stderr: %q", stderr)
	}
}

// TestReadUTDFileSourceSuppressesCommand pins the safety property of
// readUTD's source-priority trim block for the file branch. With both `file`
// and `command` set, and the file's content referencing {{.command_output}},
// the asset's command must not execute. TemplateProcessor.Process's lazy
// {{.command_output}} expansion (template.go: needsCommandOutput &&
// fields.Command != "") would otherwise shell out — readUTD's trim block
// (read.go: file != "" → fields.Command = "") is what prevents it. If
// Process's source-selection or lazy-eval semantics ever change so the trim
// block stops protecting against this, this test fails. The companion
// TestReadUTDFileWinsOverPrompt covers the externally observable behaviour
// (file content wins) but not the no-shell-out invariant.
func TestReadUTDFileSourceSuppressesCommand(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "fc-cmd-ref")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	if strings.Contains(stdout, "SHOULD-NOT-APPEAR") {
		t.Errorf("command output leaked into file-source render — readUTD trim block did not suppress command execution\nstdout: %q\nstderr: %s", stdout, stderr)
	}
	for _, marker := range []string{"before", "after"} {
		if !strings.Contains(stdout, marker) {
			t.Errorf("file content marker %q missing from stdout: %q", marker, stdout)
		}
	}
}

// TestReadUTDPromptSourceSuppressesCommand is the prompt-branch counterpart
// to TestReadUTDFileSourceSuppressesCommand. With `prompt` set and the prompt
// referencing {{.command_output}} alongside a non-empty `command`, the
// command must not execute. Guards the prompt-branch arm of readUTD's trim
// block (read.go: prompt != "" → fields.Command = "") against the same
// TemplateProcessor.Process lazy-eval regression.
func TestReadUTDPromptSourceSuppressesCommand(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "pc-cmd-ref")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}

	if strings.Contains(stdout, "SHOULD-NOT-APPEAR") {
		t.Errorf("command output leaked into prompt-source render — readUTD trim block did not suppress command execution\nstdout: %q\nstderr: %s", stdout, stderr)
	}
	for _, marker := range []string{"before", "after"} {
		if !strings.Contains(stdout, marker) {
			t.Errorf("prompt content marker %q missing from stdout: %q", marker, stdout)
		}
	}
}

// TestReadUTDPromptWinsOverCommand covers the second branch of readUTD's
// source-priority trim. With file empty and both prompt and command set, the
// prompt must win (file > prompt > command). Without this test the
// `else if fields.Prompt != ""` branch is uncovered.
func TestReadUTDPromptWinsOverCommand(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "pc-priority")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if !strings.Contains(stdout, "PROMPT VALUE") {
		t.Errorf("stdout should contain prompt content, got: %q", stdout)
	}
	if strings.Contains(stdout, "COMMAND VALUE") {
		t.Errorf("command output must not appear when prompt is set, got: %q", stdout)
	}
}

// TestReadUTDRelativePath verifies a UTD asset whose `file` field is a
// relative path (e.g. "./role.md") resolves through ExpandFilePath's
// filepath.Abs branch and outputs the file's contents. Acceptance criterion:
// "UTD file resolution succeeds for @module/, ~/, and relative paths".
func TestReadUTDRelativePath(t *testing.T) {
	setupReadTestConfig(t)

	stdout, stderr, err := runReadCmd(t, "role-relative")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if stdout != "relative contents\n" {
		t.Errorf("stdout = %q, want %q", stdout, "relative contents\n")
	}
}

// TestReadUTDModuleNoOrigin verifies the error guard in readUTD: an asset
// with an @module/ file path but no origin field returns a descriptive error
// naming the asset, and stdout stays empty.
func TestReadUTDModuleNoOrigin(t *testing.T) {
	setupReadTestConfig(t)

	stdout, _, err := runReadCmd(t, "role-module-no-origin")
	if err == nil {
		t.Fatal("expected error for @module/ path without origin")
	}
	if !strings.Contains(err.Error(), "role-module-no-origin") {
		t.Errorf("error should name the asset, got: %v", err)
	}
	if !strings.Contains(err.Error(), "@module/") {
		t.Errorf("error should mention @module/, got: %v", err)
	}
	if !strings.Contains(err.Error(), "origin") {
		t.Errorf("error should mention origin field, got: %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty on error, got: %q", stdout)
	}
}

// TestReadTooManyArgs verifies that more than one positional argument is
// rejected (Requirement 5.1: "accepts zero or one positional argument"). The
// Args validator runs before RunE, so this is a cobra-level rejection and
// stdout never gets touched.
func TestReadTooManyArgs(t *testing.T) {
	setupReadTestConfig(t)

	stdout, _, err := runReadCmd(t, "role-prompt", "extra-arg")
	if err == nil {
		t.Fatal("expected error for two positional arguments")
	}
	if stdout != "" {
		t.Errorf("stdout should be empty when args validation fails, got: %q", stdout)
	}
}

// TestReadUTDModulePath verifies @module/ resolution end-to-end: the file is
// looked up in $CUE_CACHE_DIR/mod/extract/<dir(modulePath)>/<base(modulePath)+version>/
// (see ResolveModulePath in composer.go). The test fabricates that directory
// layout and reads through readUTD's @module/ branch and DefaultFileReader.
// Acceptance criterion: "UTD file resolution succeeds for @module/, ~/, and
// relative paths".
func TestReadUTDModulePath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)

	cacheDir := filepath.Join(dir, "cue-cache")
	t.Setenv("CUE_CACHE_DIR", cacheDir)

	// CUE module cache writes read-only files; chmod before TempDir cleanup.
	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return os.Chmod(path, 0o755)
			}
			return os.Chmod(path, 0o644)
		})
	})

	// Origin layout: <dir(modulePath)>/<base(modulePath)+version>
	// → github.com/example/test-mod / sub@v1.0.0
	origin := "github.com/example/test-mod/sub@v1.0.0"
	versionedDir := filepath.Join(cacheDir, "mod", "extract",
		"github.com", "example", "test-mod", "sub@v1.0.0")
	if err := os.MkdirAll(versionedDir, 0o755); err != nil {
		t.Fatalf("creating versioned cache dir: %v", err)
	}
	moduleFile := filepath.Join(versionedDir, "module-content.md")
	if err := os.WriteFile(moduleFile, []byte("module file content"), 0o644); err != nil {
		t.Fatalf("writing module file: %v", err)
	}

	startDir := filepath.Join(dir, ".start")
	if err := os.MkdirAll(startDir, 0o755); err != nil {
		t.Fatalf("creating .start dir: %v", err)
	}

	cueConfig := `
roles: {
	"role-module": {
		description: "Module-prefixed file"
		file:        "@module/module-content.md"
		origin:      "` + origin + `"
	}
}
`
	if err := os.WriteFile(filepath.Join(startDir, "settings.cue"), []byte(cueConfig), 0o644); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	chdir(t, dir)

	stdout, stderr, err := runReadCmd(t, "role-module")
	if err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
	}
	if stdout != "module file content\n" {
		t.Errorf("stdout = %q, want %q", stdout, "module file content\n")
	}
}

// TestReadMergedScopeFindsGlobalAndLocal pins the scope constant in
// runRead's loadConfig call from both directions. The two existing scopes
// (Global at $HOME/.config/start/, Local at ./.start/) each contain a unique
// role; both must be reachable through `start read`. A regression that flipped
// runRead to ScopeLocal would break the global sub-test; flipping to
// ScopeGlobal would break the local sub-test. Pattern follows
// TestShowGlobalFlag in show_test.go: HOME is set but XDG_CONFIG_HOME is not,
// so globalConfigDir resolves to $HOME/.config/start/.
func TestReadMergedScopeFindsGlobalAndLocal(t *testing.T) {
	dir := t.TempDir()

	globalStartDir := filepath.Join(dir, ".config", "start")
	if err := os.MkdirAll(globalStartDir, 0o755); err != nil {
		t.Fatalf("creating global config dir: %v", err)
	}
	globalCueConfig := `
roles: {
	"global-only-role": {
		description: "Lives only in global config"
		prompt:      "from global"
	}
}
`
	if err := os.WriteFile(filepath.Join(globalStartDir, "settings.cue"), []byte(globalCueConfig), 0o644); err != nil {
		t.Fatalf("writing global config: %v", err)
	}

	localStartDir := filepath.Join(dir, ".start")
	if err := os.MkdirAll(localStartDir, 0o755); err != nil {
		t.Fatalf("creating local config dir: %v", err)
	}
	localCueConfig := `
roles: {
	"local-only-role": {
		description: "Lives only in local config"
		prompt:      "from local"
	}
}
`
	if err := os.WriteFile(filepath.Join(localStartDir, "settings.cue"), []byte(localCueConfig), 0o644); err != nil {
		t.Fatalf("writing local config: %v", err)
	}

	// globalConfigDir prefers XDG_CONFIG_HOME over $HOME/.config. Force the
	// $HOME/.config branch so this test is deterministic regardless of the
	// caller's environment.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", dir)
	chdir(t, dir)

	t.Run("global-only role visible via merged scope", func(t *testing.T) {
		stdout, stderr, err := runReadCmd(t, "global-only-role")
		if err != nil {
			t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
		}
		if stdout != "from global\n" {
			t.Errorf("stdout = %q, want %q", stdout, "from global\n")
		}
	})

	t.Run("local-only role visible via merged scope", func(t *testing.T) {
		stdout, stderr, err := runReadCmd(t, "local-only-role")
		if err != nil {
			t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
		}
		if stdout != "from local\n" {
			t.Errorf("stdout = %q, want %q", stdout, "from local\n")
		}
	})
}

// TestReadAgentModelOverrideExact verifies that --model with a key in the
// agent's models map produces the resolved id, not the agent's default_model.
// Regression guard for the rendering contract: `start --model haiku read claude`
// and `start --model haiku claude` must agree on the substituted model id.
func TestReadAgentModelOverrideExact(t *testing.T) {
	setupReadTestConfig(t)

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--model", "haiku", "read", "claude"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	stdoutStr := stdout.String()
	if !strings.Contains(stdoutStr, "claude-haiku-4") {
		t.Errorf("expected --model haiku to resolve to %q, got: %q", "claude-haiku-4", stdoutStr)
	}
	if strings.Contains(stdoutStr, "claude-sonnet-4") {
		t.Errorf("default_model should not appear when --model is set, got: %q", stdoutStr)
	}
}

// TestReadAgentModelOverrideSubstring verifies the multi-term substring path
// of resolveModelName: --model "hai" should match "haiku" since it is the
// only key containing that substring. Pins parity with `start`'s --model
// resolution rather than just exact-match.
func TestReadAgentModelOverrideSubstring(t *testing.T) {
	setupReadTestConfig(t)

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--model", "hai", "read", "claude"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	stdoutStr := stdout.String()
	if !strings.Contains(stdoutStr, "claude-haiku-4") {
		t.Errorf("expected --model 'hai' to resolve to haiku id %q, got: %q", "claude-haiku-4", stdoutStr)
	}
}

// TestReadAgentModelOverridePassthrough verifies that a --model value not
// present in the agent's models map is substituted verbatim. This lets users
// pass arbitrary model identifiers without having to register them in CUE.
func TestReadAgentModelOverridePassthrough(t *testing.T) {
	setupReadTestConfig(t)

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--model", "claude-opus-4-7", "read", "claude"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	stdoutStr := stdout.String()
	if !strings.Contains(stdoutStr, "claude-opus-4-7") {
		t.Errorf("expected literal --model value to pass through, got: %q", stdoutStr)
	}
}

// TestReadVerboseTildePathExpanded verifies --verbose reports the resolved
// absolute path for a tilde-prefixed file source, not the literal "~/..."
// string from the CUE config. The file is read from the expanded location
// regardless; this pins the metadata reported to the user.
func TestReadVerboseTildePathExpanded(t *testing.T) {
	dir := setupReadTestConfig(t)

	cmd := NewRootCmd()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"--verbose", "read", "role-tilde"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
	}

	expectedPath := filepath.Join(dir, "tilde-role.md")
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "Path: "+expectedPath) {
		t.Errorf("stderr should report expanded path %q, got: %s", expectedPath, stderrStr)
	}
	if strings.Contains(stderrStr, "Path: ~/tilde-role.md") {
		t.Errorf("stderr should not contain the unexpanded literal path, got: %s", stderrStr)
	}
}

// setupReadDualScopeConfig writes a global config at $HOME/.config/start/ and
// a local config at ./.start/. Both define a "shared-role" with distinct
// content so --local vs --global resolution can be told apart; each scope also
// defines a scope-only role to exercise the not-found path of the other scope.
// Mirrors TestReadMergedScopeFindsGlobalAndLocal's environment setup.
func setupReadDualScopeConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Not-found lookups fall through to the registry, which writes read-only
	// files into HOME/.cache/cue. Re-chmod before TempDir cleanup so the
	// teardown unlink calls can succeed.
	t.Cleanup(func() {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return os.Chmod(path, 0o755)
			}
			return os.Chmod(path, 0o644)
		})
	})

	globalStartDir := filepath.Join(dir, ".config", "start")
	if err := os.MkdirAll(globalStartDir, 0o755); err != nil {
		t.Fatalf("creating global config dir: %v", err)
	}
	globalCueConfig := `
roles: {
	"global-only-role": {
		description: "Lives only in global config"
		prompt:      "from global"
	}
	"shared-role": {
		description: "Defined in both scopes"
		prompt:      "shared from global"
	}
}
`
	if err := os.WriteFile(filepath.Join(globalStartDir, "settings.cue"), []byte(globalCueConfig), 0o644); err != nil {
		t.Fatalf("writing global config: %v", err)
	}

	localStartDir := filepath.Join(dir, ".start")
	if err := os.MkdirAll(localStartDir, 0o755); err != nil {
		t.Fatalf("creating local config dir: %v", err)
	}
	localCueConfig := `
roles: {
	"local-only-role": {
		description: "Lives only in local config"
		prompt:      "from local"
	}
	"shared-role": {
		description: "Defined in both scopes"
		prompt:      "shared from local"
	}
}
`
	if err := os.WriteFile(filepath.Join(localStartDir, "settings.cue"), []byte(localCueConfig), 0o644); err != nil {
		t.Fatalf("writing local config: %v", err)
	}

	// globalConfigDir prefers XDG_CONFIG_HOME over $HOME/.config. Force the
	// $HOME/.config branch so this test is deterministic regardless of the
	// caller's environment.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", dir)
	chdir(t, dir)
	return dir
}

// TestReadLocalScope verifies --local restricts resolution to the local
// config by asserting a global-only role is not visible under --local.
//
// This test has only one assertion (compared to TestReadGlobalScope's two)
// because merged-scope CUE resolution makes local override global on field
// conflict — so a "shared role under --local returns local content" check
// would pass even if --local were silently ignored and merged scope used
// instead. The only discriminating assertion for --local wiring is the
// not-found path on a global-only asset: under --local-respected the asset
// is invisible, under --local-ignored merged scope finds it. See
// TestReadGlobalScope for the symmetric test, which has two assertions
// because merged scope returns the local value for shared roles, making the
// "global wins under --global" assertion discriminating.
func TestReadLocalScope(t *testing.T) {
	setupReadDualScopeConfig(t)

	stdout, _, err := runReadCmd(t, "--local", "global-only-role")
	if err == nil {
		t.Fatal("expected not-found error for global-only role under --local")
	}
	if !strings.Contains(err.Error(), "global-only-role") {
		t.Errorf("error should name the missing asset, got: %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty on not-found error, got: %q", stdout)
	}
}

// TestReadGlobalScope verifies --global restricts resolution to the global
// config: the shared role resolves to the global definition, and a local-only
// name fails with a not-found error and empty stdout.
func TestReadGlobalScope(t *testing.T) {
	setupReadDualScopeConfig(t)

	t.Run("shared role resolves to global definition", func(t *testing.T) {
		stdout, stderr, err := runReadCmd(t, "--global", "shared-role")
		if err != nil {
			t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr)
		}
		if stdout != "shared from global\n" {
			t.Errorf("stdout = %q, want %q", stdout, "shared from global\n")
		}
	})

	t.Run("local-only role not found under --global", func(t *testing.T) {
		stdout, _, err := runReadCmd(t, "--global", "local-only-role")
		if err == nil {
			t.Fatal("expected not-found error for local-only role under --global")
		}
		if !strings.Contains(err.Error(), "local-only-role") {
			t.Errorf("error should name the missing asset, got: %v", err)
		}
		if stdout != "" {
			t.Errorf("stdout should be empty on not-found error, got: %q", stdout)
		}
	})
}

// TestReadLocalAndGlobalMutuallyExclusive verifies that passing both --local
// and --global returns the same mutual-exclusion error as `start show` and
// writes nothing to stdout. No fixture is required: showScopeFromCmd errors
// before runRead reaches loadConfig, so the cwd and HOME contents are
// irrelevant on this path.
func TestReadLocalAndGlobalMutuallyExclusive(t *testing.T) {
	stdout, _, err := runReadCmd(t, "--local", "--global", "any-name")
	if err == nil {
		t.Fatal("expected mutual-exclusion error when both --local and --global are set")
	}
	if !strings.Contains(err.Error(), "--local and --global are mutually exclusive") {
		t.Errorf("error should match show's mutual-exclusion text, got: %v", err)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty on mutual-exclusion error, got: %q", stdout)
	}
}
