# P-031: CLI workingDir Threading

- Status: Cancelled
- Started: 2026-02-22
- Completed: 2026-02-22
- Reason: The --directory flag was removed from the CLI (partially implemented across 7/40 ResolvePaths call sites, completing it was not justified for a rarely-used feature). Without --directory, the workingDir threading has no production purpose. The loadMergedConfig() dead code wrapper was deleted separately.

## Overview

Thread `workingDir` through CLI config functions to eliminate `os.Chdir` in tests. This removes a process-global hazard that permanently bars affected tests from running in parallel. Currently the `chdir` helpers in test files enforce sequential execution via a code comment — a social convention with no tooling enforcement. If a contributor adds `t.Parallel()` to any affected test, silent working-directory corruption occurs without the race detector flagging it.

The infrastructure is already partially in place: `config.ResolvePaths(workingDir string)` and `loadMergedConfigFromDir(workingDir string)` both accept the parameter. The refactor completes the threading by updating several production functions that currently pass `""` and by migrating all affected tests to pass `tmpDir` directly.

Addresses GitHub issue #51.

## Goals

1. Delete `loadMergedConfig()` (test-only wrapper with no production callers); replace all test call sites with `loadMergedConfigFromDir(tmpDir)` directly
2. Add `workingDir string` to `loadConfig`, `prepareShow`, `showVerboseItem`, `displayShowMatch`, `promptShowSelection`, `printVerboseDump`, `findConfigSource`, `reorderContexts`, and `reorderRoles`
3. Update all cobra RunE callers to pass `getFlags(cmd).Directory` (preserves existing cwd fallback in `ResolvePaths`)
4. Remove all `os.Chdir` and `chdir` helper usage from test files (see Decision Points for `TestConfigLocal_Isolation`)
5. Add `t.Parallel()` to all previously blocked test functions
6. Remove all `// Note: Do not add t.Parallel()` comments
7. Pass cleanly under `go test -race ./internal/cli/ ./test/integration/`

## Scope

In Scope:

- `internal/cli/start.go` — `loadMergedConfig()` signature
- `internal/cli/show.go` — `loadConfig`, `prepareShow`, `showVerboseItem`, `displayShowMatch`, `promptShowSelection`, `printVerboseDump`, `findConfigSource`; cobra handlers resolve and pass `workingDir`
- `internal/cli/config_order.go` — `reorderContexts`, `reorderRoles`, `runConfigOrder`, `runConfigContextOrder`, `runConfigRoleOrder`
- All five affected test files (see Current State)
- Deletion of `chdir` helpers in `start_test.go` and `test/integration/cli_test.go`

Out of Scope:

- `internal/config/paths.go` — `ResolvePaths` already accepts the parameter; no change needed
- `loadMergedConfigFromDir` and `loadMergedConfigWithIO` — already accept `workingDir`; no change needed
- `loadExecutionConfig` — already correctly threads `workingDir` from `flags.Directory`; serves as the reference pattern
- Any new features or behavioural changes beyond the structural refactor

## Success Criteria

- [ ] `config.ResolvePaths` is never called with `""` from any function that a test exercises without the test having an explicit opportunity to inject `workingDir`
- [ ] The `chdir` helpers in `start_test.go` and `test/integration/cli_test.go` are deleted
- [ ] `os.Chdir` does not appear in any `_test.go` file (subject to Decision Point 1)
- [ ] All previously blocked tests carry `t.Parallel()` (subject to Decision Point 1 for `TestConfigLocal_Isolation`)
- [ ] `go test -race ./internal/cli/ ./test/integration/` passes
- [ ] `scripts/invoke-tests` passes

## Deliverables

- Updated `internal/cli/start.go` — `loadMergedConfig()` deleted (was a test-only wrapper around `loadMergedConfigFromDir`)
- Updated `internal/cli/show.go` — `loadConfig`, `prepareShow`, `showVerboseItem`, `displayShowMatch`, `promptShowSelection`, `printVerboseDump`, `findConfigSource` all accept `workingDir string`; cobra handlers (`runShowListing`, `runShowSearch`, `runShowItem`) resolve `workingDir = getFlags(cmd).Directory` and pass it down
- Updated `internal/cli/config_order.go` — `reorderContexts` and `reorderRoles` accept `workingDir string`; cobra RunE handlers resolve and pass it
- Updated `internal/cli/start_test.go` — `chdir` helper deleted, `loadMergedConfig` calls replaced with `loadMergedConfigFromDir(tmpDir)`, `t.Parallel()` added
- Updated `internal/cli/show_test.go` — setup helpers return dir without calling `chdir`, `prepareShow` and `printVerboseDump` calls pass `dir`, cobra command tests pass `--directory dir` in args, `t.Parallel()` added
- Updated `internal/cli/config_integration_test.go` — `chdir` calls removed, `t.Parallel()` added, `TestConfigLocal_Isolation` raw `os.Chdir` migrated
- Updated `internal/cli/config_order_test.go` — `chdir` calls removed, `t.Parallel()` added
- Updated `test/integration/cli_test.go` — `chdir` helper deleted, `t.Parallel()` added

## Current State

### Production Functions Requiring Change

`internal/cli/start.go:578` — `loadMergedConfig()` wraps `loadMergedConfigFromDir("")`. Called only from test files (`start_test.go:449,542,636,730,809,1163`), not from production cobra handlers.

`internal/cli/show.go:540` — `loadConfig(scope config.Scope)` calls `config.ResolvePaths("")` directly. Called from `runShowListing` (line 150), `runShowSearch` (line 218), and `prepareShow` (line 454).

`internal/cli/show.go:453` — `prepareShow(name, scope, cueKey, itemType)` calls `loadConfig(scope)`. Called from `showVerboseItem` (line 420) and the `runShowItem` closure (line 440).

`internal/cli/show.go:419` — `showVerboseItem(w, name, scope, cueKey, itemType)` calls `prepareShow`. Called from `displayShowMatch` (lines 345, 347) and directly from `runShowSearch` (lines 258, 277, 300).

`internal/cli/show.go:334` — `displayShowMatch(w, scope, m, r)` calls `showVerboseItem`. Called from `runShowSearch` (line 327) and `promptShowSelection` (lines 344, 347).

`internal/cli/show.go:351` — `promptShowSelection(w, stdin, scope, matches, query, r)` calls `displayShowMatch`. Called from `runShowSearch` (line 329).

`internal/cli/show.go:563` — `printVerboseDump(w, r)` calls `findConfigSource`. Called from `showVerboseItem` (line 424) and the `runShowItem` closure (line 445). All tests that call `printVerboseDump` directly (e.g., `TestVerboseDumpConfigSource`, `TestVerboseDumpCUEDefinition`, `TestVerboseDumpOriginCache`, `TestVerboseDumpFileContent`, `TestVerboseDumpFileError`, `TestVerboseDumpCommand`, `TestVerboseDumpSeparators`) reach this function.

`internal/cli/show.go:638` — `findConfigSource(cueKey, name)` calls `config.ResolvePaths("")` directly. Called only from `printVerboseDump` (line 580). Determines which config file defines an item for the verbose dump output.

`internal/cli/config_order.go:93` — `reorderContexts(stdout, stdin, local)` calls `config.ResolvePaths("")`.

`internal/cli/config_order.go:176` — `reorderRoles(stdout, stdin, local)` calls `config.ResolvePaths("")`. Both called from `runConfigOrder`, `runConfigContextOrder`, and `runConfigRoleOrder`.

### Reference Pattern

`internal/cli/start.go:86` — `loadExecutionConfig(flags *Flags)` is the correct pattern. It resolves `workingDir` from `flags.Directory` (or `os.Getwd()` if empty) and passes it to `loadMergedConfigFromDirWithDebug(workingDir, flags)`. Production callers of `loadConfig` and `reorderContexts`/`reorderRoles` should follow the same approach: resolve at the cobra RunE level, pass down.

For tests, `flags.Directory` is set to the `tmpDir` created by `t.TempDir()` — no `chdir` needed.

### Affected Test Files

`internal/cli/show_test.go` (1140 lines):
- `chdir` called inside `setupTestConfig` (line 82), `setupTestConfigWithFiles` (line 148), `setupTestConfigWithOrigin` (line 179)
- Additional direct `chdir` calls in `TestPrepareShowLocalNoConfig`, `TestPrepareShowGlobalNoConfig`, `TestShowGlobalFlag`, `TestShowListingNoDescriptions`, `TestShowCrossCategoryMultipleExact`, `TestVerboseDumpFileError`

`internal/cli/start_test.go` (1887 lines):
- `chdir` helper defined at line 19
- Called by `TestExecuteStart_DryRun`, `TestExecuteStart_NoRole`, `TestExecuteTask_DryRun`, `TestExecuteTask_NoRole`, `TestExecuteTask_MissingTaskRole`, `TestExecuteStart_ContextSelection`, `TestTaskResolution`, `TestTaskResolution_AmbiguousPrefix`, `TestTaskResolution_ExactMatchFallsThrough`, `TestTaskResolution_ExactMatchTagFilter`, `TestTaskResolution_NoTasksDefined`, `TestExecuteStart_FilePathRole`, `TestExecuteStart_FilePathContext`, `TestExecuteStart_MixedContextOrder`, `TestExecuteTask_FilePathTask`, `TestExecuteTask_FilePathWithInstructions`, `TestExecuteTask_FilePathMissing`, `TestExecuteStart_FilePathContextMissing`, `TestFindInstalledTasks`
- Direct `loadMergedConfig()` calls at lines 449, 542, 636, 730, 809, 1163 — should become `loadMergedConfigFromDir(tmpDir)`

`internal/cli/config_integration_test.go` (1095 lines):
- `chdir` called in `TestConfigAgent_FullWorkflow`, `TestConfigRole_FullWorkflow`, `TestConfigContext_FullWorkflow`, `TestConfigTask_FullWorkflow`, `TestConfigTask_SubstringResolution`, `TestConfigRemove_MultipleArgs` (and sub-tests)
- These tests only operate on global scope (no `--local`); global config dir is found via `XDG_CONFIG_HOME` (already set via `t.Setenv`), not CWD — `chdir` is vestigial and safe to remove
- `TestConfigLocal_Isolation` uses raw `os.Chdir` + `defer os.Chdir(origWd)` — see Decision Points; this test has a structural dependency on CWD for `--local` flag resolution in out-of-scope config write commands

`internal/cli/config_order_test.go` (774 lines):
- `chdir` called in `TestConfigContextOrder_Command`, `TestConfigRoleOrder_Command`, `TestConfigContextOrder_Cancel`, `TestConfigContextOrder_NoContexts`, `TestConfigContextOrder_SingleItem` — these call `reorderContexts`/`reorderRoles` directly; after refactor they pass `tmpDir` as `workingDir` parameter
- `TestConfigContextAdd_PreservesOrder`, `TestConfigRoleAdd_PreservesOrder`, `TestConfigRoleList_PreservesDefinitionOrder` — use cobra commands for global config only; global dir found via `XDG_CONFIG_HOME`; `chdir` is vestigial and safe to remove

`test/integration/cli_test.go` (365 lines):
- `chdir` helper defined at line 17
- Called in `TestIntegration_CUELoaderWithComposer`, `TestIntegration_CUELoaderWithComposer_DefaultContexts`, `TestIntegration_CUELoaderWithComposer_TaggedContexts`, `TestIntegration_ComposeWithRole`, `TestIntegration_ResolveTask`, `TestIntegration_ExecutorBuildCommand`, `TestIntegration_GetTaskRole`
- Integration tests use the `integration` build tag; run with `go test -tags integration ./test/integration/`

### Test Helper Migration Pattern

Setup helpers currently call `chdir(t, dir)` to make tests work. After migration:

- Setup helpers return only the `dir` without calling `chdir`
- `t.Setenv("HOME", dir)` and `t.Setenv("XDG_CONFIG_HOME", dir)` are preserved (already test-safe)
- Functions under test receive `dir` as `workingDir` parameter
- Tests call `t.Parallel()` at the top

For `executeStart` and `executeTask`, the `Flags` struct already has `Directory string`. Tests set `flags.Directory = tmpDir` instead of calling `chdir`.

For `reorderContexts`/`reorderRoles`, tests pass the `tmpDir` directly as `workingDir`.

For `prepareShow` and `printVerboseDump`, tests pass `dir` as the new `workingDir` parameter.

For cobra command tests that used `chdir` to establish local config CWD (e.g., `TestShowGlobalFlag`, `TestShowListingNoDescriptions`, `TestShowCrossCategoryMultipleExact`, `TestShowCrossCategory`), pass `"--directory", dir` in `cmd.SetArgs(...)` instead of calling `chdir`.

## Decision Points

1. `TestConfigLocal_Isolation` in `config_integration_test.go` uses `os.Chdir(projectDir)` so that cobra commands with `--local` write to `projectDir/.start`. The config write functions (`config_agent.go`, `config_role.go`, `config_context.go`, `config_task.go`) call `config.ResolvePaths("")` and rely on `os.Getwd()` for local path resolution. These files are out of scope for p-031. As a result, this test cannot have `os.Chdir` removed without one of the following:

   A. Keep `TestConfigLocal_Isolation` as a documented exception: replace `defer os.Chdir(origWd)` with `t.Cleanup(func() { _ = os.Chdir(origWd) })` for test-safety, but accept that this test cannot carry `t.Parallel()` and revise the Success Criteria to exclude it

   B. Expand scope to thread `workingDir` through config write commands (`config_agent.go`, `config_role.go`, `config_context.go`, `config_task.go`) so that `TestConfigLocal_Isolation` can pass `--directory projectDir` instead of calling `os.Chdir`

## Technical Approach

Propagate `workingDir` parameter top-down through the call chain without changing any semantics:

1. Delete `loadMergedConfig()`; update all test call sites to `loadMergedConfigFromDir(tmpDir)` directly
2. Add `workingDir string` to `loadConfig`, `prepareShow`, `showVerboseItem`, `displayShowMatch`, `promptShowSelection`, `printVerboseDump`, `findConfigSource`, `reorderContexts`, `reorderRoles`
3. Cobra RunE handlers (`runShowListing`, `runShowSearch`, `runShowItem`, `runConfigOrder`, `runConfigContextOrder`, `runConfigRoleOrder`) resolve `workingDir = getFlags(cmd).Directory` (may be `""`) and pass it down; `config.ResolvePaths("")` continues to fall back to `os.Getwd()` for production
4. Test helpers stop calling `chdir`; tests supply `tmpDir` via `flags.Directory` (for start/task) or as a direct parameter (for show/order functions)
5. Delete `chdir` helpers, remove `// Note: Do not add t.Parallel()` comments, add `t.Parallel()` to all top-level tests

## Dependencies

None. All required infrastructure is already in place.

## Related DRs

- DR-024: Testing strategy (`test real behaviour over mocks`, `accept parameters rather than reaching for globals`)
