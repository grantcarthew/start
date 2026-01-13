# P-012: CLI Core Commands Testing

- Status: In Progress
- Started: 2025-12-23

## Overview

End-to-end testing of the `start` CLI core commands from a user's perspective. This project covers first-run experience, the three main execution commands (start, prompt, task), global flags, and error handling.

Part 1 of 3 in CLI testing series:
- P-012: Core Commands (this project)
- P-013: Configuration Commands
- P-014: Supporting Commands (Assets, Doctor, Completion)

## Goals

1. Test first-run and auto-setup flows
2. Test core execution commands (start, prompt, task)
3. Test all global flags across applicable commands
4. Test error scenarios and edge cases
5. Fix all issues discovered during testing

## Scope

In Scope:
- Auto-setup flow
- `start` command (default execution)
- `start prompt` command
- `start task` command
- Global flags (--agent, --role, --context, --model, --directory, --dry-run, --quiet, --verbose, --debug, --version)
- Error handling and messages

Out of Scope:
- Configuration commands (P-013)
- Assets, Doctor, Completion (P-014)

## Success Criteria

- [ ] All features tested and marked complete in checklist below
- [ ] All discovered issues fixed and verified
- [ ] No blocking issues remain

## Testing Workflow

For each feature:

1. Read the feature description and test steps
2. Execute the test commands
3. Record the result: PASS, FAIL, PARTIAL, or SKIP
4. If FAIL/PARTIAL:
   - Document the issue in Issues Log
   - Fix the issue
   - If fix involves design decisions: update related DR or create new DR
   - Retest and verify
5. Mark the feature as tested with brief notes

Design Record Updates:
- Bug fixes that change documented behaviour → update existing DR
- New design decisions during fixes → create new DR
- Reference DRs in Issues Log when applicable

---

## Feature Checklist

### 1. First-Run Experience

#### 1.1 Auto-Setup (No Config)

Description: When no configuration exists, start should trigger auto-setup.

Test:
```bash
rm -rf ~/.config/start && rm -rf ./.start
./start
```

Expected: Auto-setup flow triggers, detects AI CLI tools, prompts for selection (if interactive), creates config.

Result: PASS

Notes: Auto-setup triggered, detected aichat/claude/gemini. Non-TTY correctly errors with guidance.

---

#### 1.2 Auto-Setup (Empty Directory)

Description: Empty config directory should trigger auto-setup, not error.

Test:
```bash
rm -rf ~/.config/start && mkdir ~/.config/start
./start
```

Expected: Auto-setup triggers (empty directory treated as no config).

Result: PASS

Notes: Empty directory triggers auto-setup (previously fixed issue #1).

---

#### 1.3 Auto-Setup (Non-TTY)

Description: Auto-setup in non-interactive mode with multiple agents.

Test:
```bash
rm -rf ~/.config/start
echo "" | ./start
```

Expected: Error message about multiple agents, suggests running interactively.

Result: PASS

Notes: Non-TTY correctly errors with "multiple AI CLI tools detected" message.

---

#### 1.4 Invalid CUE File Error

Description: Invalid CUE syntax should produce clear, actionable error message.

Test:
```bash
mkdir -p ~/.config/start
echo 'agents: { test: { bin: "test" command: "cmd" } }' > ~/.config/start/config.cue
./start
```

Expected: Error shows file path, line, column, source context with pointer.

Result: PASS

Notes: Clear error with file path, line 1, column 31, source context with pointer.

---

#### 1.5 Invalid CUE Multi-line Error

Description: Error in multi-line file shows context lines.

Test:
```bash
cat > ~/.config/start/config.cue << 'EOF'
// Config
agents: {
    claude: {
        bin: "claude"
        command: "--print {role}
    }
}
EOF
./start
```

Expected: Shows 2 lines before/after error with line numbers.

Result: PASS

Notes: Shows context lines before and after error with line numbers.

---

### 2. Start Command (Default)

#### 2.1 Start with Dry-Run

Description: Running `start --dry-run` previews execution without launching agent.

Test:
```bash
./start --dry-run
```

Expected: Shows agent, role, contexts, writes temp files, does not execute.

Result: PASS

Notes: Shows agent, role, contexts, role content, prompt preview, temp file paths.

---

#### 2.2 Start Default Contexts

Description: Plain `start` includes required AND default contexts.

Test:
```bash
./start --dry-run --debug
```

Expected: Debug output shows required and default contexts included.

Result: PASS

Notes: Debug shows `required=true, defaults=true, tags=[]`. Warning for missing "project" context file is expected (README.md doesn't exist in cwd).

---

#### 2.3 Start Execution

Description: Start actually launches the agent.

Test:
```bash
./start
# (will launch agent - verify it starts correctly, then exit)
```

Expected: Agent launches with composed prompt.

Result: PASS

Notes: Agent launches correctly. Initial 23-second wait was Claude processing the long AGENTS.md context - not a hang.

---

### 3. Prompt Command

#### 3.1 Prompt with Text

Description: Launch agent with custom prompt.

Test:
```bash
./start prompt "Hello world" --dry-run
```

Expected: Shows prompt content including "Hello world".

Result: PASS

Notes: Shows "Hello World" in prompt. Short content correctly displays without "(5 lines)" label.

---

#### 3.2 Prompt Excludes Defaults

Description: Prompt command excludes default contexts (per DR-014).

Test:
```bash
./start prompt "test" --dry-run --debug
```

Expected: Only required contexts, not default contexts.

Result: PASS

Notes: Debug shows `defaults=false` and `0 contexts`. Warning for missing "project" context is expected (required context with missing file).

---

#### 3.3 Prompt with Context Tag

Description: Can include tagged contexts with prompt.

Test:
```bash
./start prompt "test" -c test --dry-run
```

Expected: Includes required + tagged contexts.

Result: PASS

Notes: Context "testing" (tagged with "test") loaded successfully. Required fix for template placeholder case mismatch (issue #9).

---

#### 3.4 Prompt with Default Pseudo-tag

Description: Can include defaults using -c default.

Test:
```bash
./start prompt "test" -c default --dry-run
```

Expected: Includes required + default contexts.

Result: PASS

Notes: The `-c default` pseudo-tag includes the "codebase" context (which has `default: true`).

---

### 4. Task Command

#### 4.1 Task Execution

Description: Run a predefined task.

Test:
```bash
./start task <task-name> --dry-run
```

Expected: Task resolves, shows composed prompt.

Result: PASS

Notes: Task auto-installs from registry when not found locally (fixed isTaskNotFoundError to handle "no tasks defined" case). Full execution tested with golang/code-review - took 8min due to comprehensive prompt and --print mode waiting for complete response.

---

#### 4.2 Task with Instructions

Description: Instructions fill {{.instructions}} placeholder.

Test:
```bash
./start task <task-name> "focus on error handling" --dry-run
```

Expected: Instructions appear in composed prompt.

Result: PASS

Notes: Instructions displayed in dry-run summary and appear in prompt.md under "## Custom Instructions".

---

#### 4.3 Task Substring Match

Description: Task names can be matched by substring (per DR-015).

Test:
```bash
# If task "code-review" exists
./start task review --dry-run
```

Expected: Matches task containing "review" if unique.

Result: PASS

Notes: Substring "review" matched "golang/code-review" successfully.

---

#### 4.4 Task Ambiguous Match

Description: Multiple substring matches produce error or selection.

Test:
```bash
# Create two tasks with similar names
./start task <ambiguous-prefix> --dry-run
```

Expected: Error listing matching tasks (non-TTY) or interactive selection (TTY).

Result: PASS

Notes: Prefix "golang" matched golang/code-review and golang/refactor. Error: `ambiguous task prefix "golang" matches: golang/code-review, golang/refactor`.

---

#### 4.5 Task Not Found

Description: Unknown task produces clear error.

Test:
```bash
./start task nonexistent-task-xyz
```

Expected: Error message listing available tasks.

Result: PASS

Notes: Shows "Task not found locally, checking registry..." then `task "nonexistent-task-xyz" not found`.

---

#### 4.6 Task Context Selection

Description: Tasks include required contexts only by default.

Test:
```bash
./start task <task-name> --dry-run --debug
```

Expected: Only required contexts included.

Result: PASS

Notes: Debug shows `Selection: required=true, defaults=false, tags=[]`. Only "project" context (required) included.

---

### 5. Global Flags

#### 5.1 --agent Flag

Description: Override agent selection.

Test:
```bash
./start --agent <agent-name> --dry-run
./start prompt "test" --agent <agent-name> --dry-run
./start task <task> --agent <agent-name> --dry-run
```

Expected: Uses specified agent instead of default.

Result: PASS

Notes: Agent flag works. Invalid agent name gives clear error: `agent "nonexistent" not found`.

---

#### 5.2 --role Flag

Description: Override role selection.

Test:
```bash
./start --role <role-name> --dry-run
```

Expected: Uses specified role.

Result: PASS

Notes: Role flag works. Warning shown for undefined role but continues execution.

---

#### 5.3 --model Flag

Description: Override model selection.

Test:
```bash
./start --model "custom-model" --dry-run
```

Expected: Uses specified model (shown in output).

Result: PASS

Notes: Model flag works. Shows "opus (via --model)" indicating source.

---

#### 5.4 --context Flag (Single)

Description: Select contexts by single tag.

Test:
```bash
./start --context <tag> --dry-run
```

Expected: Includes required + tagged contexts.

Result: PASS

Notes: Single tag selection works. `-c test` includes required + default + tagged "testing" context.

---

#### 5.5 --context Flag (Multiple Comma)

Description: Select multiple context tags with comma.

Test:
```bash
./start -c <tag1>,<tag2> --dry-run
```

Expected: Includes contexts matching either tag.

Result: PASS

Notes: Comma-separated tags work. `-c test,docs` includes contexts with either tag.

---

#### 5.6 --context Flag (Multiple Flags)

Description: Select multiple context tags with repeated flag.

Test:
```bash
./start -c <tag1> -c <tag2> --dry-run
```

Expected: Includes contexts matching either tag.

Result: PASS

Notes: Repeated -c flags work. `-c test -c docs` includes contexts with either tag.

---

#### 5.7 --context default Pseudo-tag

Description: Combine default contexts with tagged.

Test:
```bash
./start -c default,<tag> --dry-run
```

Expected: Includes required + default + tagged contexts.

Result: PASS

Notes: Default pseudo-tag works. `-c default,test` includes required + default "codebase" + tagged "testing".

---

#### 5.8 --context No Match Warning

Description: Warning when tag matches no contexts.

Test:
```bash
./start -c nonexistent-tag-xyz --dry-run
```

Expected: Warning about no matching contexts.

Result: PASS

Notes: Initially failed - no warning shown. Fixed by adding unmatched tag check in composer.go. Now shows: `Warning: tag "nonexistent-tag-xyz" matched no contexts`.

---

#### 5.9 --directory Flag

Description: Override working directory.

Test:
```bash
./start --directory /tmp --dry-run --debug
```

Expected: Uses /tmp as working directory for context detection.

Result: PASS

Notes: Directory flag works. Debug shows `Working directory (from --directory): /tmp`.

---

#### 5.10 --dry-run Flag

Description: Preview without executing.

Test:
```bash
./start --dry-run
ls /tmp/start-dry-run-*  # Check temp files created
```

Expected: Shows preview, creates temp files with role.md, prompt.md, command.txt.

Result: PASS

Notes: Dry-run shows preview and creates temp files: role.md, prompt.md, command.txt.

---

#### 5.11 --quiet Flag

Description: Suppress output.

Test:
```bash
./start --quiet --dry-run
```

Expected: Minimal or no output (temp files still created).

Result: PARTIAL

Notes: Quiet suppresses execution info but dry-run output still shows (may be intentional - dry-run's purpose is to preview).

---

#### 5.12 --verbose Flag

Description: Detailed output.

Test:
```bash
./start --verbose --dry-run
```

Expected: Additional detail about context resolution, paths.

Result: PARTIAL

Notes: Verbose flag defined but no additional output in start/dry-run commands. Used by doctor and assets commands.

---

#### 5.13 --debug Flag

Description: Debug output (implies --verbose).

Test:
```bash
./start --debug --dry-run
```

Expected: Debug messages with [DEBUG] prefix.

Result: PASS

Notes: Debug flag shows [DEBUG] prefixed messages with detailed config, agent, context info.

---

#### 5.14 --version Flag

Description: Show version information.

Test:
```bash
./start --version
```

Expected: Shows version, repo URL, issue URL.

Result: PASS

Notes: Shows version (dev), GitHub repo URL, and issues URL.

---

#### 5.15 --help Flag

Description: Show help for all commands.

Test:
```bash
./start --help
./start prompt --help
./start task --help
```

Expected: Shows usage, flags, subcommands.

Result: PASS

Notes: Help works for root and all subcommands. Shows usage, available flags, and subcommands.

---

### 6. Error Handling

#### 6.1 Unknown Command

Description: Unknown command shows helpful error.

Test:
```bash
./start unknowncommand
```

Expected: Error message, no usage spam (SilenceUsage enabled).

Result: PASS

Notes: Shows `unknown command "unknowncommand" for "start"` - no usage spam.

---

#### 6.2 Invalid Flag

Description: Invalid flag shows error.

Test:
```bash
./start --invalid-flag
```

Expected: Error about unknown flag.

Result: PASS

Notes: Shows `unknown flag: --invalid-flag`.

---

#### 6.3 Invalid Directory

Description: Invalid directory path shows clear error.

Test:
```bash
./start --directory /nonexistent/path --dry-run
```

Expected: Error about directory not found.

Result: PASS

Notes: Shows `directory not found: /nonexistent/path`.

---

#### 6.4 Agent Binary Not Found

Description: Missing agent binary handled gracefully.

Test:
```bash
# Configure agent with nonexistent binary, then try to run
./start --agent <agent-with-bad-binary>
```

Expected: Clear error about missing binary.

Result: SKIP

Notes: Requires config modification. Validation exists in validateCommandExecutable (executor.go).

---

#### 6.5 No Agent Configured

Description: No agent in config shows helpful error.

Test:
```bash
# Remove all agents from config
./start
```

Expected: Error about no agent configured.

Result: SKIP

Notes: Auto-setup triggers when no config exists. Requires isolated environment to test.

---

#### 6.6 Exit Codes

Description: Exit code 0 on success, 1 on any error.

Test:
```bash
./start --dry-run; echo "Exit: $?"
./start --invalid-flag; echo "Exit: $?"
```

Expected: First returns 0, second returns 1.

Result: PASS

Notes: Success exits 0, error exits 1.

---

## Issues Log

| ID | Feature | Description | Status | Fix | DR |
|----|---------|-------------|--------|-----|-----|
| 1 | 1.2 | Empty directory caused error instead of auto-setup | Fixed | Added ValidateConfig with CUE file checking | - |
| 2 | - | `start config` showed help instead of listing config | Fixed | Added RunE to list all config | DR-034 |
| 3 | - | Parent commands didn't support `help` subcommand | Fixed | Added checkHelpArg helper, RunE to all parent commands | DR-034 |
| 4 | - | Unknown subcommands silently ran parent action | Fixed | Added unknownCommandError helper | DR-034 |
| 5 | 2.1 | Role config used `content` field instead of `prompt` | Fixed | Test config error - changed to `prompt` | - |
| 6 | 2.3 | `{prompt}` syntax not detected, caused silent bash failure | Fixed | Added singleBracePlaceholderPattern validation in ValidateCommandTemplate | - |
| 7 | 2.3 | Missing `{{.bin}}` in template caused cryptic bash error | Fixed | Added validateCommandExecutable to check first token is executable | - |
| 8 | - | Test templates used `{{.Bin}}` (uppercase) instead of `{{.bin}}` | Fixed | Updated all test files to use lowercase | - |
| 9 | 3.3 | Template placeholders used PascalCase but UTD docs specify snake_case | Fixed | Changed TemplateData from struct to map[string]string with lowercase keys; added `missingkey=zero` option | - |
| 10 | 2.3 | Model display showed empty "(model: )" when not specified | Fixed | Added resolveModel() to track source; hide model line when empty, show source when set | - |
| 11 | 2.3 | PrintHeader lacked visual spacing | Fixed | Added blank line before headers in PrintHeader() | - |
| 12 | 3.1 | Content preview showed "(5 lines)" even for short content | Fixed | Created printContentPreview() that only shows line count when truncated | - |
| 13 | 4.1 | No indication whether asset is from package vs user-defined | Deferred | Requires schema change - see DR-037, P-015 | DR-037 |
| 14 | 4.1 | Task not found doesn't auto-fetch from registry per DR-015 step 3 | Fixed | Integrate registry fetch into findTask; also handle "no tasks defined" case in isTaskNotFoundError | DR-015 |
| 15 | 4.1 | getAssetKey strips prefix causing collisions (golang/code-review → code-review) | Fixed | Preserve full path per DR-003 | DR-003 |
| 16 | 4.1 | @module/ path prefix not resolved, files not found | Fixed | Added @module/ resolution to resolveRole/resolveContext (P-016) | DR-023 |
| 17 | 4.1 | "Executing..." message unclear during long agent waits | Fixed | Changed to "Starting <agent> - awaiting response..." | - |
| 18 | 4.3 | `show task` doesn't support substring matching like `start task` | Fixed | Added substring matching to `prepareShowTask` per DR-015 | DR-015 |
| 19 | 4.1 | {{.file}} placeholder returns CUE cache path instead of local temp | Fixed | Pre-write temp files in Composer before template processing (P-016) | DR-020 |
| 20 | 5.8 | No warning when context tag matches no contexts | Fixed | Added unmatched tag check in Compose() with warning in ComposeResult.Warnings | - |

---

## Deliverables

- This project document with completed checklist
- All issues fixed and verified

---

## Notes

Testing started: 2025-12-23

2025-12-24: During testing, implemented terminal colors (DR-036) to improve error/warning visibility. Added `--no-color` global flag. Tests 1.1-2.2 complete, test 2.3 blocked by config issue (now fixed).

2025-12-24: Completed tests 2.3 and 3.1-3.4. Fixed 4 issues during testing:
- Issue #9: Major fix - template placeholder case mismatch (PascalCase vs snake_case)
- Issue #10: Model display improvements (hide when empty, show source)
- Issue #11: Added blank line before headers for visual spacing
- Issue #12: Content preview only shows line count when truncated

2025-01-07: Fixed Issue #14 properly - `isTaskNotFoundError` now handles "no tasks defined" case. Task 4.1 passes. Changed "Executing..." message to "Starting <agent> - awaiting response..." for better UX during long-running agent calls. Noted that --print mode waits for complete response (8min for comprehensive code review task).

2025-01-13: Completed all remaining tests (4.2-6.6). Fixed Issue #20 - added warning when context tag matches no contexts (composer.go). Tests 5.11 (--quiet) and 5.12 (--verbose) marked PARTIAL - quiet doesn't suppress dry-run output (may be intentional), verbose has no effect on start command. Tests 6.4 and 6.5 marked SKIP - require config modifications to test. All other tests PASS.
