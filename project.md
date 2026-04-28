# CLI Read Command

## 1. Goal

Add a `start read` subcommand that outputs asset content to stdout. Users can pipe curated asset content into other tools, paste it into existing agent sessions, or preview what an asset contains without launching a full agent run.

## 2. Scope

In scope:
- New `start read [name]` subcommand.
- UTD assets (roles, contexts, tasks): output the resolved file, prompt, or command result with template placeholders populated.
- Agent assets: output the partially rendered command template.
- Three-tier asset resolution with auto-install from registry, matching `start show`.
- Pure content output to stdout.
- `--verbose` adds metadata (asset name, type, origin, resolved path) on stderr.

Out of scope:
- Multiple asset arguments per invocation.
- New flags beyond the persistent set.
- Changes to existing commands or to the orchestration engine.

## 3. Current State

The CLI registers nine top-level commands. `start show` already performs cross-category asset search, resolution, and content display, but wraps output in metadata formatting unsuitable for piping.

Relevant infrastructure:
- `partialFillAgentCommand` substitutes static agent-template placeholders (`{{.bin}}`, `{{.model}}`) and leaves runtime placeholders (`{{.prompt}}`, `{{.role}}`, `{{.role_file}}`) untouched.
- `orchestration.ExtractUTDFields` returns a `UTDFields` struct containing `File`, `Prompt`, `Command`, `Shell`, `Timeout`.
- `orchestration.ExtractOrigin` and `orchestration.ResolveModulePath` together resolve `@module/` path prefixes against the CUE module cache.
- `orchestration.TemplateProcessor.Process` performs full UTD resolution: lazy file reading, command execution, template substitution. Source-field priority is Prompt then File then Command — the first non-empty source wins. `Shell` and `Timeout` configure command execution and apply regardless of which source was selected.
- `orchestration.DefaultFileReader` reads via `os.ReadFile` and does not resolve `@module/` prefixes; callers resolve those first.
- `shell.NewRunner` returns a runner implementing `orchestration.ShellRunner`.
- `internal/cli/cross_resolve.go` exposes `resolveCrossCategory`, the shared resolution flow extracted from `start show`. Its doc comment specifies that callers needing clean stdout (such as `read`) must construct the resolver with stderr in the stdout writer slot, otherwise selection menus, registry progress, and auto-install notices corrupt piped output.
- `start show` reads files via a helper (`resolveShowFile`) that returns raw bytes without template resolution. It is not reusable for `read`.

## 4. References

- `internal/cli/show.go` — the closest analogue command and the source of `partialFillAgentCommand`.
- `internal/cli/cross_resolve.go` — shared resolution flow with the doc comment on writer wiring for clean-pipe callers.
- `internal/orchestration/composer.go` — `ExtractUTDFields`, `ExtractOrigin`, `ResolveModulePath`.
- `internal/orchestration/template.go` — `UTDFields`, `TemplateProcessor`, source-field priority.
- `.ai/design/utd-pattern.md` — UTD specification.
- `.ai/design/cli-command-structure.md` — CLI structure conventions.

## 5. Requirements

1. `start read [name]` accepts zero or one positional argument. Without an argument: in non-TTY mode return an error; in TTY mode prompt for a search query, then resolve and output content (single match) or present a selection menu (multiple matches).
2. Asset name resolution uses the same cross-category strategy as `start show`: exact config match, registry search with auto-install, substring search, interactive selection on ambiguity.
3. For UTD assets, output is template-resolved and follows this source priority: file, then prompt, then command. The selected source is rendered (or executed, for command) and written to stdout.
4. For agent assets, output is the command template with static placeholders substituted and runtime placeholders left intact.
5. Stdout receives only the asset content. Selection menus, registry fetch progress, auto-install notices, `--verbose` metadata, and status messages all go to stderr.
6. If an asset has no source field that can produce content, return a configuration error naming the asset and the expected fields, leave stdout empty, and exit non-zero via the standard error path.
7. `--verbose` emits asset type, name, origin, and resolved file path to stderr before the content.
8. The command appears in `start --help` under the same group as the other top-level user commands.
9. `start read help` displays the command help, matching the convention used by other commands.

## 6. Implementation Plan

1. Define the command, register it, and write the help text. The help text must explain that output is template-resolved, and must warn that for UTD assets carrying both `file` and `prompt`, `read` outputs the file (priority: file > prompt > command) while the agent execution path renders the prompt and injects file contents via `{{.file_contents}}` — so for these mixed-field assets the `read` output will not match what the agent receives. Direct users to `start show` to inspect the prompt.

2. Implement the run function, resolving the asset via `resolveCrossCategory`. Construct the resolver so that selection menus, registry progress, and auto-install notices land on stderr — see the `cross_resolve.go` doc comment for the exact wiring requirement. Requirement 5's stdout-only rule applies to every write path, not just resolver-wrapped output: pass `cmd.ErrOrStderr()` to `promptSearchQuery` for the no-argument TTY prompt, and emit the `Query must be at least 3 characters` short-query fallback to stderr.

3. For UTD assets, extract the fields and resolve any `@module/` prefix on the file path using the asset's origin. Reject with a clear error if the prefix is present but origin is empty. Before calling `Process`, guard the empty-source case by calling `orchestration.IsUTDValid(fields)`; on false, return `fmt.Errorf("asset %q has no content fields (expected one of: file, prompt, command)", name)` per Requirement 6. When fields are valid, construct the processor with `NewTemplateProcessor(fr, sr, workingDir)` where `workingDir` is the result of `os.Getwd()`. Hand the work to `TemplateProcessor.Process`. The processor's source priority is the inverse of what `read` needs, so control priority by clearing the higher-priority source fields before calling `Process`. `Shell` and `Timeout` are execution config, not source fields — they must always pass through to `Process` regardless of which source was selected. Add a code comment at the trimming site that names `template.go` as the priority dependency.

4. For agent assets, partially render the command template with `partialFillAgentCommand`. If the agent has no command field, return `fmt.Errorf("agent %q has no command field", name)` per Requirement 6.

5. Write tests covering each path, including:
   - Each UTD source variant (file, prompt, command).
   - A UTD `command` asset that declares a custom `shell` and `timeout` — verify both flow through (regression cover for the trimming logic).
   - A UTD asset with both `file` and `prompt` — verify file wins.
   - Agent asset rendering.
   - Empty-content configuration error for agents (no command field): error names the agent, empty stdout, non-zero exit.
   - Empty-content configuration error for UTD assets (file, prompt, command all empty): error names the asset and lists expected fields, empty stdout, non-zero exit.
   - No-argument behaviour in non-TTY returns an error.
   - Ambiguous name in non-TTY returns an error listing the candidates.
   - `--verbose` writes metadata to stderr without polluting stdout.
   - Registry progress and auto-install notices land on stderr (resolver constructed with stderr in the stdout slot).
   - Wiring check: a unit-level test asserting that `read` calls `promptSearchQuery` with `cmd.ErrOrStderr()` and emits the short-query fallback on stderr. TTY-gated end-to-end paths (no-arg prompt, selection menu, TTY-stdin/piped-stdout) are intentionally not covered — see Implementation Guidance.

6. Update `.ai/design/cli-command-structure.md` to list the new command.

## 7. Constraints

- Do not modify existing commands or the orchestration engine.
- Reuse existing helpers where they apply: `partialFillAgentCommand`, `ExtractUTDFields`, `ExtractOrigin`, `ResolveModulePath`, `resolveCrossCategory`. `resolveShowFile` does not apply because it returns raw bytes without template resolution.
- Stdout is reserved for asset content. Every other write goes to stderr.

## 8. Implementation Guidance

- The `TemplateProcessor` source-priority dependency is the only non-obvious part of the UTD path. Trimming source fields before calling `Process` is the chosen mechanism because it preserves a single rendering path; the inline comment is the safety net that catches a future `Process` refactor.
- After `resolveCrossCategory` reports an install occurred, refresh the in-memory config before looking up the resolved asset's CUE value. The same pattern is used by `start` and `task`.
- `start read` operates on the merged (global + local) config, matching what an agent would see. Scope inspection remains the job of `start show` and `start config info`.
- The `IsUTDValid` guard in plan step 3 lives in `read.go` because `TemplateProcessor.Process` returns the generic error `UTD requires at least one of: file, command, or prompt` (`template.go:122`) when all source fields are empty. Without the guard, that raw error would surface instead of the descriptive configuration error Requirement 6 commits to. Keeping the guard local to `read.go` leaves `Process` and the existing `start`/`task` paths unchanged.
- `workingDir` in plan step 3 is resolved once via `os.Getwd()` because it feeds the `{{.cwd}}` and git template variables and is the directory where command-source UTDs execute (`template.go:115`). Command-source UTDs run in the user's cwd by design — not in the asset's origin or CUE module cache directory — so the user's environment (working tree, git state) is what the command sees. Passing `""` would let the shell command inherit the parent process's cwd at exec time rather than a stable resolved path. `start` and `task` use the same once-resolved pattern via `loadExecutionConfig` (`start.go:86-92`).
- The empty-content error path (Requirement 6, plan step 4) does not use a `Silent()` error type because `cmd/start/main.go` prefixes non-silent RunE errors with `Error: `, which is the correct surface for a configuration error. The descriptive error names the asset and the missing fields. A `Silent()` type (cf. `assets_validate.go:90`, `doctor.go:89`) would suppress that message — the opposite of what's wanted here.
- The wiring requirement in plan step 2 (route `promptSearchQuery` and the short-query fallback to stderr) departs from `show` because `show.go:99-112` writes both to `cmd.OutOrStdout()`. A literal copy of that pattern would corrupt `start read | bar` whenever stdin is a TTY but stdout is piped — a common interactive pipe case.
- The TTY-gated e2e tests dropped from plan step 5 are out of scope because `isTerminal` (`root.go:159`) only treats `*os.File` TTY descriptors as interactive, and the existing test suite has no pseudo-TTY helpers (no `creack/pty` or `/dev/tty` usage). The unit-level wiring check proves the writer-routing contract without a real TTY; revisit with `creack/pty` if interactive surface area grows.

## 9. Acceptance Criteria

- `start read <name>` resolves a known asset and writes its content to stdout with no decoration.
- `start read` with no argument prompts in TTY mode and errors in non-TTY mode.
- Queries shorter than the established minimum length are rejected (non-TTY) or re-prompted (TTY), matching `start show`.
- UTD file resolution succeeds for `@module/`, `~/`, and relative paths.
- Template placeholders are populated in the output.
- UTD `command` execution honours per-asset `shell` and `timeout`.
- Agent assets produce a partially rendered command template.
- Assets with no producible content return a configuration error naming the asset (and, for UTD, the expected fields), leave stdout empty, and exit non-zero via the standard error path.
- Selection menus, registry progress, auto-install notices, and `--verbose` metadata all appear on stderr; stdout remains pipe-clean in every case.
- Cross-category interactive selection works for ambiguous names.
- Registry auto-install works for previously uninstalled assets.
- Unknown asset names produce a clear error.
- `start read help` displays the command help.
- The command appears in `start --help` under the same group as the other top-level user commands.
- `.ai/design/cli-command-structure.md` lists the command.

## 10. Issues Discovered

None outstanding. Prior review pass identified five issues (writer wiring for
TTY prompts, TTY-test precedent, UTD empty-source guard, error-framing for
empty content, working-directory specification); all have been folded into
Requirements, the Implementation Plan, and Implementation Guidance above.
