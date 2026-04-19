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
6. If an asset has no source field that can produce content, write `No content available` to stderr, leave stdout empty, and exit non-zero.
7. `--verbose` emits asset type, name, origin, and resolved file path to stderr before the content.
8. The command appears in `start --help` under the same group as the other top-level user commands.
9. `start read help` displays the command help, matching the convention used by other commands.

## 6. Implementation Plan

1. Define the command, register it, and write the help text. The help text must explain that output is template-resolved, and must warn that for UTD assets carrying both `file` and `prompt`, `read` outputs the file (priority: file > prompt > command) while the agent execution path renders the prompt and injects file contents via `{{.file_contents}}` — so for these mixed-field assets the `read` output will not match what the agent receives. Direct users to `start show` to inspect the prompt.

2. Implement the run function, resolving the asset via `resolveCrossCategory`. Construct the resolver so that selection menus, registry progress, and auto-install notices land on stderr — see the `cross_resolve.go` doc comment for the exact wiring requirement. Requirement 5's stdout-only rule applies to every write path, not just resolver-wrapped output: pass `cmd.ErrOrStderr()` to `promptSearchQuery` for the no-argument TTY prompt, and emit the `Query must be at least 3 characters` short-query fallback to stderr.

3. For UTD assets, extract the fields, resolve any `@module/` prefix on the file path (using the asset's origin; reject with a clear error if the prefix is present but origin is empty), and hand the work to `TemplateProcessor.Process`. The processor's source priority is the inverse of what `read` needs, so control priority by clearing the higher-priority source fields before calling `Process`. `Shell` and `Timeout` are execution config, not source fields — they must always pass through to `Process` regardless of which source was selected. Add a code comment at the trimming site that names `template.go` as the priority dependency, so a future refactor of `Process` priority surfaces the coupling.

4. For agent assets, partially render the command template with `partialFillAgentCommand`. If the agent has no command field, treat it as the empty-content case from Requirement 6.

5. Write tests covering each path, including:
   - Each UTD source variant (file, prompt, command).
   - A UTD `command` asset that declares a custom `shell` and `timeout` — verify both flow through (regression cover for the trimming logic).
   - A UTD asset with both `file` and `prompt` — verify file wins.
   - Agent asset rendering.
   - Empty-content behaviour: stderr message, empty stdout, non-zero exit.
   - No-argument behaviour in non-TTY returns an error.
   - Ambiguous name in non-TTY returns an error listing the candidates.
   - `--verbose` writes metadata to stderr without polluting stdout.
   - Registry progress and auto-install notices land on stderr (resolver constructed with stderr in the stdout slot).
   - Wiring check: a unit-level test asserting that `read` calls `promptSearchQuery` with `cmd.ErrOrStderr()` and emits the short-query fallback on stderr. TTY-gated end-to-end paths (no-arg prompt, selection menu, TTY-stdin/piped-stdout) are not covered by an e2e test — see Issue 2 for the rationale.

6. Update `.ai/design/cli-command-structure.md` to list the new command.

## 7. Constraints

- Do not modify existing commands or the orchestration engine.
- Reuse existing helpers where they apply: `partialFillAgentCommand`, `ExtractUTDFields`, `ExtractOrigin`, `ResolveModulePath`, `resolveCrossCategory`. `resolveShowFile` does not apply because it returns raw bytes without template resolution.
- Stdout is reserved for asset content. Every other write goes to stderr.

## 8. Implementation Guidance

- The `TemplateProcessor` source-priority dependency is the only non-obvious part of the UTD path. Trimming source fields before calling `Process` is the chosen mechanism because it preserves a single rendering path; the inline comment is the safety net that catches a future `Process` refactor.
- After `resolveCrossCategory` reports an install occurred, refresh the in-memory config before looking up the resolved asset's CUE value. The same pattern is used by `start` and `task`.
- `start read` operates on the merged (global + local) config, matching what an agent would see. Scope inspection remains the job of `start show` and `start config info`.

## 9. Acceptance Criteria

- `start read <name>` resolves a known asset and writes its content to stdout with no decoration.
- `start read` with no argument prompts in TTY mode and errors in non-TTY mode.
- Queries shorter than the established minimum length are rejected (non-TTY) or re-prompted (TTY), matching `start show`.
- UTD file resolution succeeds for `@module/`, `~/`, and relative paths.
- Template placeholders are populated in the output.
- UTD `command` execution honours per-asset `shell` and `timeout`.
- Agent assets produce a partially rendered command template.
- Assets with no producible content write `No content available` to stderr, leave stdout empty, and exit non-zero.
- Selection menus, registry progress, auto-install notices, and `--verbose` metadata all appear on stderr; stdout remains pipe-clean in every case.
- Cross-category interactive selection works for ambiguous names.
- Registry auto-install works for previously uninstalled assets.
- Unknown asset names produce a clear error.
- `start read help` displays the command help.
- The command appears in `start --help` under the same group as the other top-level user commands.
- `.ai/design/cli-command-structure.md` lists the command.

## 10. Issues Discovered

1. TTY prompt and short-query feedback need stderr wiring, not just the resolver (gap) — Resolved: plan step 2 and test list updated.

   Requirement 5 reserves stdout for asset content. Plan step 2 covers the resolver
   writer-wiring (via `cross_resolve.go`) for registry progress, install notices,
   and selection menus. But the no-argument TTY prompt (Requirement 1) and the
   short-query fallback (Acceptance Criterion 3) are separate code paths. In
   `show.go`, both paths write to `cmd.OutOrStdout()` — `promptSearchQuery` takes
   an `io.Writer`, and the "Query must be at least 3 characters" fallback uses
   stdout directly (`show.go:104,112`). An implementer copying the `show` pattern
   would send these messages to stdout, corrupting `start read | bar` when stdin
   is a TTY but stdout is piped (a common interactive pipe use case).

   Resolution: plan step 2 now states that Requirement 5's stdout-only rule
   applies to every write path, instructing the implementer to pass
   `cmd.ErrOrStderr()` to `promptSearchQuery` and to emit the short-query
   fallback on stderr. Plan step 5 adds a test case asserting that, with a TTY
   stdin and piped stdout, the prompt and the fallback message land on stderr
   and stdout stays empty until content is produced.

2. TTY-dependent tests have no precedent in this test suite (gap) — Resolved: accept reduced coverage (Option A).

3. UTD empty-source path bypasses Requirement 6 (gap)

   Plan step 4 explicitly maps the empty-content case for agents to
   Requirement 6 ("If the agent has no command field, treat it as the
   empty-content case from Requirement 6"). Plan step 3 (UTD assets) is
   silent on the equivalent path. `TemplateProcessor.Process` returns
   the error `UTD requires at least one of: file, command, or prompt`
   when all source fields are empty (`template.go:122`). An implementer
   following plan step 3 literally would call `Process`, receive that
   error, and surface it as a regular error — leaving stdout empty and
   exiting non-zero, but emitting Process's raw error string instead of
   the `No content available` message Requirement 6 mandates and
   Acceptance Criterion 8 verifies. The asymmetry between steps 3 and 4
   is the smoking gun: the plan author considered empty-content for
   agents and did not extend the same treatment to UTD assets.

   Suggested resolution: extend plan step 3 with a pre-`Process` check
   using `orchestration.IsUTDValid(fields)`; on false, write
   `No content available` to stderr, leave stdout empty, and return a
   non-zero exit, matching the agent path. Add a test case to plan
   step 5 covering an empty-source UTD asset (analogous to the existing
   "Empty-content behaviour" entry, but for UTD rather than agents).

   Plan step 5 lists three tests that require a real TTY on stdin:
   the no-argument TTY prompt, the selection menu landing on stderr, and the
   TTY-stdin/piped-stdout pipe-cleanliness assertion added by Issue 1. The
   codebase's `isTerminal` (`root.go:159`) only returns true when stdin is an
   `*os.File` whose fd is a TTY, and the existing test suite contains no
   pseudo-TTY helpers or `/dev/tty` usage (no `pty`, `creack/pty`, `openpty`
   matches anywhere). Every existing `show`/`resolve` test asserts the non-TTY
   branch, so an implementer copying established patterns has no template for
   the TTY branch. Without direction, they will either skip these tests —
   leaving Requirement 5's headline guarantee (pipe-clean stdout with a TTY
   stdin) unverified — or introduce a new test dependency mid-feature.

   Resolution: Option A — the three TTY-gated e2e tests are dropped from plan
   step 5. Coverage shifts to a unit-level wiring check asserting that `read`
   passes `cmd.ErrOrStderr()` to `promptSearchQuery` and emits the short-query
   fallback on stderr. `promptSearchQuery` and the selection helper in
   `cross_resolve.go` are already internally testable with an arbitrary writer,
   so a unit test proves the writer-routing contract without needing a real
   TTY. The CI suite stays pty-free; if future interactive surface area grows,
   revisit with `creack/pty` (Option C) as a deliberate investment.
