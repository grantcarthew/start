## Diff Review Summary

Scope: 7 files changed (5 modified, 2 new), +497 insertions, -34 deletions
Intent: Add three-tier asset resolution (installed config, registry, substring search) to `--agent`, `--role`, `--model`, and `--context` CLI flags, per DR-041. Splits `prepareExecutionEnv` into `loadExecutionConfig` + `buildExecutionEnv` so the resolver can search installed config between loading and building.
Verdict: SAFE

## Critical Findings

None.

## High Findings

None.

## Medium Findings

**1. Warning message changed -- may affect user-facing documentation or scripts** (`composer.go:210`)

The warning text changed from `tag %q matched no contexts` to `context %q not found`. While no tests assert the old string, any user scripts or docs referencing the old message will break. This is a reasonable change (the message is now more accurate since the value may be a context name rather than a tag), but worth noting.

**2. `findExactInRegistry` short name ambiguity** (`resolve.go:332-340`)

When searching by short name (e.g. `"assistant"`), if multiple entries share the same short name (e.g. `"golang/assistant"` and `"python/assistant"`), the function returns the first match found from map iteration, which is non-deterministic. The existing `findExactTaskInRegistry` in `task.go:721` has the same pattern, so this is consistent but potentially surprising. Low probability in practice since short-name collisions across categories are unlikely.

**3. Duplicate `findExact` functions** (`resolve.go:322` vs `task.go:719`)

`findExactInRegistry` (resolve.go) and `findExactTaskInRegistry` (task.go) have nearly identical logic. Task resolution still uses its own function and separate `TaskMatch`/`TaskSource` types rather than the new `AssetMatch`/`AssetSource` types from the resolver. This is a known incremental approach (tasks already had their own resolution before this change), but it creates two parallel systems. Not a bug, but creates maintenance burden.

## Low / Info

**4. `prepareExecutionEnv` retained for backward compatibility** (`start.go:124-133`)

The wrapper function is retained but only used... nowhere now. Both `executeStart` and `executeTask` call `loadExecutionConfig` + `buildExecutionEnv` directly. The function isn't exported, so if nothing else calls it, it's dead code. Check whether anything outside these files uses it; if not, it can be removed. (Grep shows it's only in `start.go` and some project docs.)

**5. `cfg` variable reassignment after reload** (`start.go:184`, `task.go:114`)

After `r.reloadConfig(workingDir)`, the local `cfg` variable is reassigned from `r.cfg`. This works correctly but is slightly fragile -- if someone later adds code between the reload and `buildExecutionEnv` that uses `r.cfg` directly, it would work, but code using the local `cfg` before the reassignment would use stale data. Current code is correct.

**6. `resolveContexts` calls `ensureIndex` per term** (`resolve.go:257`)

For each context term that doesn't match installed config, `ensureIndex()` is called inside the loop. Due to the lazy-fetch caching (`didFetch`), this is safe and only hits the network once. Just noting the pattern is correct.

**7. Context resolution installs but doesn't error on install failure** (`resolve.go:296-303`)

When installing registry context matches, `autoInstall` errors are silently `continue`d past, and the context name is not added to resolved. This means a failed install silently drops the context. For contexts (multi-select semantics), this is a reasonable design choice -- the user may still get other matched contexts. Contrast with agent/role resolution where install errors are fatal.

**8. Good test coverage**

The new `resolve_test.go` covers exact match, substring match, no match, multiple matches (non-TTY error), empty input, file path bypass, model resolution (exact/substring/passthrough/multiple/nil/empty), context resolution (exact/file/default/search/no-match/mixed), score threshold filtering, and merging. The `search_test.go` additions cover `SearchCategoryEntries`, `SearchInstalledConfig`, and `extractIndexEntryFromCUE`. All tests use `t.Parallel()` and `skipRegistry: true` appropriately.

## Verdict

The changes are a solid improvement. The three-tier resolution architecture is well-designed with clear separation of concerns: `loadExecutionConfig` (phase 1), resolver (phase 2), `buildExecutionEnv` (phase 3). The resolver struct with lazy registry fetching, deduplication, and auto-install is clean. Error handling is appropriate (fatal for agent/role, graceful for context). Test coverage is thorough. The code builds, vets, and all tests pass.

Safe to commit.
