# CLI Read Command

This project follows `1-dedupe.md`, which extracts the shared `resolveCrossCategory()` function used by the `read` command for cross-category asset resolution.

## 1. Goal

Add a `start read` subcommand that outputs asset file content directly to stdout. This enables users to pull curated asset content into existing agent sessions, pipe it to other tools, or preview what a document contains without launching a full agent session.

## 2. Scope

In scope:
- New `start read [name]` subcommand
- UTD assets (roles, contexts, tasks): resolve the `file` field and output file contents; fall back to rendered prompt template, then command output
- Agent assets: partially render the command template (resolve `bin` and `default_model`, leave runtime placeholders as-is)
- Three-tier asset resolution with auto-install from registry
- Pure content output to stdout (no decoration, no headers)
- `--verbose` flag adds metadata (asset name, type, origin, resolved path) before content

Out of scope:
- Multiple asset arguments (one asset per invocation)
- New flags beyond what the persistent flags already provide
- Changes to existing commands
- Changes to the orchestration or UTD resolution engine

## 3. Current State

The CLI has nine top-level commands registered in `internal/cli/root.go` via `NewRootCmd()`: `show`, `prompt`, `task`, `assets`, `config`, `search`, `doctor`, `completion`, `help`. The `show` command (`internal/cli/show.go`) already performs cross-category asset search, resolution, and content display, but wraps output in verbose metadata formatting (headers, separators, CUE definitions, labels).

Key existing infrastructure:
- `showCategories` slice defines the four asset categories (agents, roles, contexts, tasks)
- `resolveShowFile()` in `show.go` resolves `@module/` paths via CUE cache and `~/` paths via home directory expansion, then reads file contents
- `partialFillAgentCommand()` in `show.go` substitutes `{{.bin}}` and `{{.model}}` placeholders in agent command templates
- `orchestration.ExtractUTDFields()` extracts `file`, `command`, `prompt`, `shell`, `timeout` from a CUE value
- `orchestration.ExtractOrigin()` gets the origin field for module path resolution
- Three-tier resolution logic in `internal/cli/resolve.go` handles exact match, registry search, substring search, and auto-install
- Cross-category search in `runShowSearch()` handles ambiguous matches and interactive selection
- `orchestration.TemplateProcessor.Process()` handles full UTD template resolution with lazy file reading and command execution
- `shell.NewRunner()` creates a shell runner implementing `orchestration.ShellRunner` for command execution
- `orchestration.NewTemplateProcessor()` creates a processor that accepts a `FileReader`, `ShellRunner`, and working directory

## 4. References

- `internal/cli/root.go` - Command registration pattern
- `internal/cli/show.go` - Asset search, resolution, file reading, agent command rendering
- `internal/cli/resolve.go` - Three-tier resolution, `AssetMatch`, `resolver`, auto-install
- `internal/orchestration/composer.go` - `ExtractUTDFields()`, `ExtractOrigin()`, `ResolveModulePath()`
- `internal/orchestration/template.go` - `UTDFields` struct, template resolution
- `internal/orchestration/filepath.go` - `ExpandFilePath()`, `IsFilePath()`
- `.ai/design/utd-pattern.md` - UTD specification
- `.ai/design/cli-command-structure.md` - CLI structure conventions
- `internal/cli/show_test.go` - Test patterns using `setupTestConfig()`

## 5. Requirements

1. Add a `start read [name]` subcommand that accepts zero or one positional argument. Without an argument: in non-TTY mode return an error; in TTY mode prompt for a search query, then resolve and output content (single match) or present a selection menu (multiple matches)
2. Resolve the asset name using cross-category search following the same pattern as `start show`: exact config match, registry search with auto-install, substring search, interactive selection on ambiguity
3. For UTD assets (roles, contexts, tasks), produce output in this priority order. All paths resolve `{{}}` template placeholders (environment variables, git info, etc.) in the output:
   a. If the asset has a `file` field: resolve the path (`@module/`, `~/`, relative), read the file, resolve template placeholders in the file contents, and output
   b. If no `file` field but has a `prompt` field: resolve template placeholders and output the rendered prompt
   c. If no `file` or `prompt` but has a `command` field: execute the command and output the result
4. For agent assets: partially render the command template by substituting `{{.bin}}` and `{{.model}}` with resolved values, leaving runtime placeholders (`{{.prompt}}`, `{{.role}}`, `{{.role_file}}`) as-is
5. Output pure content to stdout with no decoration, headers, or formatting. Suitable for piping
6. If an asset has no content to produce (no file, no prompt, no command for UTD; no command for agent), output "No content available" to stdout
7. When `--verbose` is set, print metadata to stderr before the content: asset type, name, origin (if present), and resolved file path (if applicable)
8. Register the command in `NewRootCmd()` under the "commands" group
9. Support the `help` argument pattern used by other commands (`checkHelpArg`)

## 6. Implementation Plan

1. Create `internal/cli/read.go` with the `addReadCommand()` function following the pattern established by `addShowCommand()` in `show.go`. The command `Long` description should note that: (a) for UTD assets, file content takes priority over prompt when both are present (unlike the orchestration engine which prioritises prompt), and (b) output is template-resolved — `{{}}` placeholders are populated with environment values

2. Implement the `runRead()` function:
   - If no argument: in non-TTY mode return an error; in TTY mode prompt for a search query (same pattern as `runShow` in show.go:95-127)
   - Enforce 3-character minimum query length (same as `show`)
   - Delegate to a search function that reuses the cross-category resolution pattern from `runShowSearch()`
   - On resolution, call a content extraction function

3. Implement the content extraction function for UTD assets:
   - Call `orchestration.ExtractUTDFields()` on the resolved CUE value
   - If `fields.File` starts with `@module/`: call `orchestration.ExtractOrigin()` on the CUE value. If origin is empty, return an error: "@module/ path requires origin field". Otherwise resolve with `orchestration.ResolveModulePath(fields.File, origin)` (matching the pattern in `composer.go:439-449`)
   - Construct a `TemplateProcessor` with `orchestration.NewTemplateProcessor(&orchestration.DefaultFileReader{}, shell.NewRunner(), workingDir)` where `workingDir` comes from `os.Getwd()`
   - Determine priority and pass modified fields to `processor.Process()`:
     - If `fields.File` is not empty: pass `UTDFields{File: fields.File, Command: fields.Command}` (strip Prompt so Process uses file as template source)
     - Else if `fields.Prompt` is not empty: pass `UTDFields{Prompt: fields.Prompt, Command: fields.Command}` (strip File)
     - Else if `fields.Command` is not empty: pass `UTDFields{Command: fields.Command}`
     - Else: output "No content available"
   - Output `result.Content` to stdout

4. Implement the content extraction for agent assets:
   - Extract the `command` field from the CUE value
   - If command is empty: output "No content available" and return
   - Call `partialFillAgentCommand()` (already exists in `show.go`) to substitute static placeholders
   - Output the partially rendered command string

5. Add `--verbose` metadata output:
   - When verbose, write asset type, name, origin, and resolved path to stderr (not stdout, to keep stdout clean for piping)

6. Register the command in `root.go` by adding `addReadCommand(cmd)` to `NewRootCmd()`

7. Write tests in `internal/cli/read_test.go` following the patterns in `show_test.go`:
   - Asset with `file` field outputs file contents with template placeholders resolved
   - Asset with `prompt` only outputs rendered prompt
   - Asset with `command` only outputs command result
   - Agent asset outputs partially rendered command
   - Asset with no content outputs "No content available"
   - Unknown asset name returns error
   - Ambiguous name in non-TTY returns error with match list
   - No argument in non-TTY returns error
   - `--verbose` outputs metadata to stderr

## 7. Constraints

- Follow the existing CLI patterns in `internal/cli/` for command structure, flag access, and error handling
- Use `cmd.OutOrStdout()` for content output and `cmd.ErrOrStderr()` for verbose metadata
- Reuse existing functions where they exist (`resolveShowFile`, `partialFillAgentCommand`, `ExtractUTDFields`, `ExtractOrigin`)
- Do not modify existing commands or packages
- Follow Go formatting conventions (gofmt)
- Pure Go, no cgo
- Tests use `setupTestConfig(t)` with `.start/` directory and `os.Chdir` isolation (no `t.Parallel()`)

## 8. Implementation Guidance

- When a utility function or module already exists, use it. Do not reimplement the same logic. In particular, `resolveShowFile()` and `partialFillAgentCommand()` are in `show.go` but can be called directly since `read.go` is in the same package.
- Use the shared `resolveCrossCategory()` function (extracted by the dedupe project) for cross-category asset resolution. Call it from `runRead()` and apply read-specific content extraction to the returned `AssetMatch`. After the call, look up the category with `cat := showCategoryFor(match.Category)` and guard against nil: `if cat == nil { return fmt.Errorf("unknown category %q", match.Category) }`. This matches the defensive check applied in `runShowSearch()` per `1-dedupe.md` issue 18.
- When `r.didInstall` is true after `resolveCrossCategory()` returns, call `r.reloadConfig(workingDir)` before looking up the resolved asset's CUE value via `r.cfg.Value.LookupPath(...)`. The function installs to disk but does not refresh `r.cfg` in place, so a direct lookup against the pre-call config would miss the newly installed asset. Matches the existing pattern at `start.go:342-348` and `task.go:128-133`. See `1-dedupe.md` resolved issue 24 and the `resolveCrossCategory()` doc comment.
- For template resolution, use `orchestration.NewTemplateProcessor(&orchestration.DefaultFileReader{}, shell.NewRunner(), workingDir)` and call `processor.Process(fields, "")`. Control source priority by stripping fields before passing to `Process()`.
- All new modules, functions, or components must have corresponding tests.
- Update `.ai/design/cli-command-structure.md` to include the `start read` command.

## 9. Open Issues

17. Constraint lists `resolveShowFile` but implementation does not use it (gap)

    Section 7 says "Reuse existing functions where they exist (`resolveShowFile`, `partialFillAgentCommand`, `ExtractUTDFields`, `ExtractOrigin`)". The implementation plan step 3 does not use `resolveShowFile()` — it resolves `@module/` paths manually and passes to `TemplateProcessor.Process()`, which reads the file internally via `DefaultFileReader`. This is correct because `resolveShowFile()` returns raw file content without template resolution, and `read` needs template resolution. The constraint should drop `resolveShowFile` to avoid confusion.

18. `TemplateProcessor.Process()` priority is Prompt > File > Command (risk)

    `Process()` (template.go:100-123) checks `fields.Prompt` first. The plan handles this by stripping fields before passing to `Process()`, but this is a subtle coupling. If `Process()` priority changes in a future refactor, `read`'s file-first priority could silently break. The plan correctly handles this today, but the coupling should be documented in a code comment.

### Resolved Issues

1. UTD priority inversion for file vs prompt (design) — Documented in help text.
   File > prompt > command priority is intentional for `read`. Implementation plan step 1 updated to include notes in the command `Long` description.

2. Template resolution requires TemplateProcessor and ShellRunner setup (gap) — Option A adopted.
   Implementation plan step 3 now specifies constructing `TemplateProcessor` with `DefaultFileReader`, `shell.NewRunner()`, and `os.Getwd()`. All UTD paths use `processor.Process()` with field manipulation for priority control.

3. Cross-category search extraction from `runShowSearch` (design) — Superseded by issues 13 and 16.
   Original resolution: duplicate the resolution flow in `runReadSearch()` with read-specific output and leave `show.go` untouched. Superseded once the dedupe project (`1-dedupe.md`) was scheduled ahead of `read`. `read` now consumes the shared `resolveCrossCategory()` function extracted by dedupe — no duplication, no private `runReadSearch()` copy. See resolved issues 13 (dedupe extracts shared logic) and 16 (implementation order: dedupe first, read second).

4. Scope flag support (gap) — Merged scope only.
   No `--global` flag for `read`. Content output uses merged config (global + local), matching what the agent sees. Scope inspection is served by `show` and `config info`.

5. No-argument behaviour unspecified (gap) — Follow `show` pattern.
   No argument in non-TTY returns error. In TTY, prompt for a query, then resolve: single match outputs content, multiple matches present a selection menu. Requirement 1 and implementation plan step 2 updated.

6. `setupTestConfig` vs `setupStartTestConfig` naming (gap) — No action needed.
   Two separate helpers for separate test files. Project correctly references `setupTestConfig(t)` from `show_test.go`.

7. Reference to non-existent `orchestration.ResolveTemplate()` (gap) — Guidance corrected.
   Section 8 updated to reference `orchestration.NewTemplateProcessor()` and `processor.Process()` instead.

8. TemplateProcessor.Process() resolves all Go template syntax in file content (risk) — Accepted.
   Consistent with the orchestration engine. `read` output matches what the agent sees. Document in `read` help text that output is template-resolved.

9. Working directory for TemplateProcessor (gap) — Use `os.Getwd()`.
   Already specified in implementation plan step 3 when issue 2 was resolved.

10. Minimum query length not addressed (gap) — 3-character minimum.
    Already specified in implementation plan step 2 when issue 5 was resolved.

11. `@module/` path resolution not handled by TemplateProcessor (gap) — Added to implementation plan.
    `DefaultFileReader.Read()` does not resolve `@module/` prefixes. Implementation plan step 3 updated to include explicit `@module/` resolution via `ExtractOrigin()` and `ResolveModulePath()` before passing to `Process()`, matching `composer.go:439-449`.

12. Empty origin guard for `@module/` paths (gap) — Added to implementation plan.
    Implementation plan step 3 updated to check for empty origin before calling `ResolveModulePath()`, matching the guard in `resolveShowFile()` and `composer.go:440-441`.

13. `runReadSearch` duplication size and maintenance cost (risk) — Resolved by ordering.
    The dedupe project (`1-dedupe.md`) extracts shared resolution logic first. `read` will use the shared `resolveCrossCategory()` function instead of duplicating `runShowSearch()`.

14. `ResolveModulePath` reference location incorrect (gap) — Fixed.
    Section 4 moved `ResolveModulePath()` from the `filepath.go` entry to the `composer.go` entry where the function actually lives.

15. Agent with no command field not handled in step 4 (gap) — Guard added.
    Implementation plan step 4 updated with an empty-command check that outputs "No content available" before calling `partialFillAgentCommand()`.

16. Implementation order with dedupe (decision) — Dedupe first, read second.
    The dedupe project extracts shared `resolveCrossCategory()` from `runShowSearch()` before `read` is implemented. `read` uses the shared function from the start, avoiding duplication. Section 8 guidance updated.

## 10. Acceptance Criteria

- [ ] `start read <name>` resolves an asset by name and outputs its content to stdout
- [ ] `start read` with no argument prompts for a query in TTY mode, returns error in non-TTY mode
- [ ] Queries under 3 characters are rejected (non-TTY) or re-prompted (TTY)
- [ ] UTD file resolution works for `@module/` paths, `~/` paths, and relative paths
- [ ] Template placeholders (`{{.cwd}}`, `{{.git_branch}}`, etc.) are resolved in output
- [ ] When no file field exists, falls back to rendered prompt, then command output
- [ ] Agent assets output a partially rendered command template
- [ ] Assets with no content output "No content available"
- [ ] Output is pure content with no decoration (suitable for piping)
- [ ] `--verbose` writes metadata to stderr without polluting stdout
- [ ] Cross-category search with interactive selection works for ambiguous names
- [ ] Registry auto-install works for uninstalled assets
- [ ] Unknown asset names produce a clear error
- [ ] `start read help` displays command help
- [ ] Command appears in `start --help` output under the Commands group
- [ ] All new code has corresponding tests
- [ ] `cli-command-structure.md` updated with the new command
- [ ] `go build ./...` succeeds
- [ ] `scripts/invoke-tests` passes
