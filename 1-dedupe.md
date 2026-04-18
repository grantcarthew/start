# Extract Shared Cross-Category Resolution Logic

Prerequisite for `2-read-feature.md`. Complete this project first to provide the shared `resolveCrossCategory()` function that the `read` command depends on.

## 1. Goal

Extract the cross-category asset resolution flow from `runShowSearch()` into a reusable function so that both `show` and the new `read` command can share the same resolution logic without duplication.

## 2. Scope

In scope:
- Extract resolution logic from `runShowSearch()` into a shared function
- Refactor `runShowSearch()` to call the shared function
- Design the shared function so `read` (and future commands) can use it with their own output handling

Out of scope:
- Implementing the `read` command (covered by `2-read-feature.md`)
- Changes to the resolver, asset matching, or registry logic in `resolve.go`
- Changes to `promptAssetSelection` interactive prompt

## 3. Current State

`runShowSearch()` in `internal/cli/show.go:234-373` (verified) contains two interleaved concerns:

1. Resolution: exact match across categories, ambiguous handling, substring search, registry search with auto-install, match merging (~80 lines of logic)
2. Output: calling `showVerboseItem()`, `displayShowMatch()`, or `promptShowSelection()` at each branch endpoint

The resolution logic is the same flow needed by `read` and any future command that resolves an asset by name across categories. The output handling is show-specific.

The resolution flow has these decision points, each terminating with an output action:

| Step | Condition | Current action |
|------|-----------|----------------|
| 1 | Ambiguous short name matches | `promptShowSelection` |
| 2 | Single exact match (no extra substring matches) | `showVerboseItem` |
| 3 | Multiple exact matches | `promptShowSelection` |
| 4 | Single substring match | `showVerboseItem` |
| 5 | Exact registry match (no installed matches) | Auto-install then `showVerboseItem` |
| 6 | Combined search, single merged match | `displayShowMatch` |
| 7 | Combined search, multiple merged matches | `promptShowSelection` |
| 8 | No matches | Error |

Steps 1-5 and 8 can resolve to a single `AssetMatch` or an error. Steps 6-7 also resolve to a single match (directly or via selection). The output action is always applied to a single resolved match.

Supporting functions already in `resolve.go` and `show.go`:
- `findExactInstalledName()` - exact/short name match in installed config
- `searchInstalled()` - substring search in installed config
- `registryEntries()` - get registry entries for a category
- `findExactInRegistry()` - exact name match in registry
- `searchRegistryCategory()` - substring search in registry
- `mergeAssetMatches()` - deduplicate and sort installed + registry matches
- `newResolver()` / `resolver.ensureIndex()` / `resolver.autoInstall()` - registry operations
- `showCategoryFor()` - look up category metadata
- `promptShowSelection()` - interactive selection from multiple matches
- `displayShowMatch()` - auto-install if needed, then display

## 4. References

- `internal/cli/show.go` - `runShowSearch()`, `showVerboseItem()`, `displayShowMatch()`, `promptShowSelection()`
- `internal/cli/resolve.go` - `AssetMatch`, `resolver`, matching and merging functions
- `internal/cli/show_test.go` - Existing tests for cross-category search behaviour
- `2-read-feature.md` - The `read` command project that will consume the extracted function

## 5. Requirements

1. Extract a function `resolveCrossCategory(query string, r *resolver) (AssetMatch, error)` that performs cross-category asset resolution. The function auto-installs when the resolved match originates from the registry, setting `r.didInstall = true`. Callers read `r.didInstall` after the call to decide whether to flip scope to `config.ScopeMerged` for downstream operations
2. The function must handle interactive selection when multiple matches exist (TTY) and return an error when multiple matches exist (non-TTY)
3. `runShowSearch()` must be refactored to call the extracted function, then compute `effectiveScope` from `r.didInstall` (merged when true, else the input scope), then call `showVerboseItem(w, match.Name, effectiveScope, cat.key, cat.itemType)` on the returned match. `displayShowMatch()` is deleted
4. All existing `show` tests must pass without modification
5. The extracted function must be usable by `read` and future commands with no show-specific dependencies
6. `promptShowSelection()` is refactored to return `(AssetMatch, error)` instead of calling display functions directly. Auto-install of a registry match selected by the user happens in `resolveCrossCategory()` after the prompt returns

## 6. Implementation Plan

1. Create a `resolveCrossCategory()` function in `internal/cli/cross_resolve.go` with the signature in requirement 1. The function uses `r.cfg`, `r.stdout`, `r.stderr`, `r.stdin`, `r.flags`, and `r.didInstall` directly from the passed-in resolver — no re-loading of config. Callers are responsible for loading `r.cfg` at the correct scope before calling the function. The function's doc comment must specify the writer routing contract:

   ```
   // resolveCrossCategory resolves an asset query across all categories via
   // three-tier search (exact installed → substring installed → registry) with
   // interactive selection on ambiguity. Auto-installs registry matches and
   // sets r.didInstall = true when an install occurs.
   //
   // Writes:
   //   r.stdout — registry fetch progress, install notices, interactive selection prompts
   //   r.stderr — debug output only
   //
   // Callers needing clean stdout (e.g. `start read` piping content) should construct
   // the resolver with stderr in the stdout slot: newResolver(cfg, flags, stderr, stderr, stdin).
   //
   // Post-call contract: if r.didInstall is true and the caller subsequently reads
   // r.cfg.Value (e.g. to look up the resolved asset's CUE value), the caller must
   // first call r.reloadConfig(workingDir) — the installed asset is written to disk
   // but r.cfg is not refreshed in place. See start.go:342 and task.go:128 for the
   // established pattern. show is exempt because showVerboseItem loads config
   // independently via prepareShow.
   ```

2. Move the resolution logic from `runShowSearch()` into `resolveCrossCategory()`. This includes steps 1-8 from the Current State table. Replace output calls (`showVerboseItem`, `displayShowMatch`) with returning the match. Replace `promptShowSelection` calls with the refactored selection prompt that returns the chosen match. Preserve the `r.ensureIndex()` call position: it must remain after step 2 (installed substring search) and only execute when installed-only resolution fails to produce a single match. Do not hoist `ensureIndex()` to the top of the function — doing so regresses offline behaviour and triggers unnecessary network calls on every fast-path resolution. The existing gate at show.go:322-324 must be preserved in the extracted function

3. Move `promptShowSelection()` from `show.go` to `cross_resolve.go` and rename it to `promptCrossCategorySelection()` with the signature `promptCrossCategorySelection(r *resolver, matches []AssetMatch, query string) (AssetMatch, error)`. The function reads writer/reader from `r.stdout` and `r.stdin`, matching the pattern established by `promptAssetSelection` at resolve.go:517. The `scope` parameter is dropped (it was only used to pass through to `displayShowMatch`, which is being deleted). The call to `displayShowMatch()` is removed — selection is purely about choosing a match. Co-locating with `resolveCrossCategory()` keeps the whole cross-category flow in one file and prevents a reverse dependency from the shared layer to `show.go`

4. Centralise auto-install in `resolveCrossCategory()` via a helper `r.installIfRegistry(match AssetMatch) error` that auto-installs when `match.Source == AssetSourceRegistry` by calling `r.autoInstall(...)`. The helper takes no scope argument and returns no scope — `r.autoInstall` already sets `r.didInstall = true` on success (resolve.go:631), and callers read `r.didInstall` after `resolveCrossCategory()` returns to flip scope to `ScopeMerged`. This matches the pattern at `start.go:342` and `task.go:286`. Apply this helper at all three registry-match branches: step 5 (exact registry match), step 6 (combined-search single match), and step 7 (post-selection)

5. Delete `displayShowMatch()`. Refactor `runShowSearch()` to: keep the leading `fmt.Fprintln(w)` separator, construct the resolver via `newResolver()`, call `resolveCrossCategory()`, compute `effectiveScope := scope; if r.didInstall { effectiveScope = config.ScopeMerged }`, look up the category with `cat := showCategoryFor(match.Category); if cat == nil { return fmt.Errorf("unknown category %q", match.Category) }`, then call `showVerboseItem(w, match.Name, effectiveScope, cat.key, cat.itemType)` on the returned match. The same defensive nil-check must be applied in `read.go` when it consumes `resolveCrossCategory`. Remove the now-unused `github.com/grantcarthew/start/internal/assets` import from `show.go` — its only use was the `assets.SearchResult` literal in the deleted `displayShowMatch()`; the construction moves with the helper to `resolve.go` / `cross_resolve.go`

6. Verify all existing tests in `show_test.go` pass without modification. Create `internal/cli/cross_resolve_test.go` with direct unit tests that construct the resolver via `newTestResolver()` (which sets `r.skipRegistry = true`) and exercise the non-registry branches of `resolveCrossCategory()`:
   - zero matches across all categories returns a "no matches" error
   - single installed exact match returns the match without prompting
   - ambiguous short-name exact match in non-TTY returns an ambiguity error
   - single installed substring match returns the match without prompting
   - combined-search multiple matches in non-TTY returns an ambiguity error
   - fall-through: single exact match plus additional substring matches across categories returns an ambiguity error in non-TTY (and surfaces selection in TTY)

   These tests are fast and deterministic because `skipRegistry = true` bypasses the network path. Registry-dependent branches (exact-registry auto-install, combined-search with registry results) remain covered by the manual verification accepted in resolved issue 19

## 7. Constraints

- Follow existing CLI patterns in `internal/cli/` for function signatures and error handling
- Do not change the external behaviour of `start show` (output format, error messages, interactive prompts)
- Do not modify `resolve.go` — the extracted function composes existing resolution primitives
- The leading `fmt.Fprintln(w)` at show.go:236 must remain in `runShowSearch()` before the `resolveCrossCategory()` call. It is show-specific cosmetics and must not move into the extracted function, otherwise `start read` would emit a stray leading newline on stdout
- Pure Go, no cgo
- Tests use `setupTestConfig(t)` with `os.Chdir` isolation (no `t.Parallel()`)

## 8. Implementation Guidance

- When a utility function or module already exists, use it. Do not reimplement the same logic.
- All new functions must have corresponding tests unless they are thin wrappers.
- Update `2-read-feature.md` section 8 (Implementation Guidance) to reference the extracted function instead of advising duplication. Remove the guidance about duplicating `runShowSearch` in `runReadSearch`.
- Tests that touch `resolveCrossCategory()` set `r.skipRegistry = true` on the constructed resolver before the call, matching the pattern in `resolve_test.go:256`.
- For consumers needing clean stdout (e.g. `start read`), construct the resolver with stderr in the stdout slot: `newResolver(cfg, flags, stderr, stderr, stdin)`. This routes registry fetch/install progress lines to stderr. `show` keeps the existing `newResolver(cfg, flags, stdout, stderr, stdin)` wiring. This is a consumer-side choice — `resolveCrossCategory()` itself makes no assumptions about which stream is "user-visible".

## 9. Issues Discovered

25. Unit test coverage and placement for `resolveCrossCategory` (gap) — Resolved: new `cross_resolve_test.go` with six direct unit tests.
    Acceptance criteria updated to require `internal/cli/cross_resolve_test.go` containing direct unit tests that construct the resolver via `newTestResolver()` (skipRegistry = true) and cover the non-registry branches: zero matches, single installed exact, ambiguous short-name exact (non-TTY), single installed substring, combined-search multiple (non-TTY), and the fall-through case. Fast, deterministic, no registry dependency. Implementation plan step 6 updated to enumerate the tests. Registry-dependent branches (exact-registry auto-install, combined-search with registry results) remain covered by manual verification per resolved issue 19. Mirrors the file layout of `cross_resolve.go` and isolates unit tests from the cobra integration tests in `show_test.go`.

24. Stale `r.cfg` after auto-install inside `resolveCrossCategory` (gap) — Resolved: document the caller's reload contract.
    `resolveCrossCategory()` writes the installed asset to disk but does not refresh `r.cfg`. `show` avoids this because `showVerboseItem → prepareShow → loadConfig` loads fresh config. Consumers that read `r.cfg.Value` directly after the call (e.g. `read`) must call `r.reloadConfig(workingDir)` when `r.didInstall` is true, matching the existing pattern at `start.go:342-348` and `task.go:128-133`. Implementation plan step 1 doc comment extended with a "Post-call contract" bullet making this explicit. Cross-reference added to `2-read-feature.md` section 8 so the read implementation performs the reload before looking up the CUE value. No change to `resolveCrossCategory()` internals — the existing codebase convention is followed.

23. `promptShowSelection` file placement after refactor (design) — Resolved: move to `cross_resolve.go` and rename.
    After the refactor the function is purely generic (returns `AssetMatch`, no display side-effects). Leaving it in `show.go` would create a reverse dependency from the shared layer to a consumer. Implementation plan step 3 updated to move `promptShowSelection()` into `cross_resolve.go` alongside `resolveCrossCategory()` and rename it to `promptCrossCategorySelection()`. Keeps the whole cross-category flow in one file and sheds the now-misleading `Show` prefix.

20. Orphaned `assets` import in `show.go` after `displayShowMatch` deletion (gap) — Resolved: cleanup step added to plan.
    Implementation plan step 5 updated with: "Remove the now-unused `github.com/grantcarthew/start/internal/assets` import from `show.go` — its only use was the `assets.SearchResult` literal in the deleted `displayShowMatch()`; the construction moves with the helper to `resolve.go` / `cross_resolve.go`." Prevents a build break during incremental implementation.

21. `installIfRegistry` helper signature contradicts scope-free design (design) — Resolved: scope dropped from helper.
    Implementation plan step 4 updated with the signature `r.installIfRegistry(match AssetMatch) error`. No scope argument, no scope return. `r.autoInstall(...)` sets `r.didInstall = true` on success (resolve.go:631), and callers read `r.didInstall` to flip scope to `ScopeMerged` — matching the existing pattern at `start.go:342` and `task.go:286`. Removes the inconsistency hazard issue 16 already removed from the public API.

22. `2-read-feature.md` resolved issue 3 contradicts dedupe-first plan (gap) — Resolved: read doc updated.
    `2-read-feature.md` resolved issue 3 originally read "Duplicate in `read.go`." Text replaced with "Superseded by issues 13 and 16" and a pointer to the dedupe extraction, so future readers do not implement duplication. Numbering preserved. Cross-reference cleanup only — no code impact.

1. File placement for `resolveCrossCategory()` (decision) — Resolved: new file.
   Create `internal/cli/cross_resolve.go` for the shared cross-category resolution function. Signals shared purpose and keeps `show.go` focused on show-specific concerns.

2. Selection prompt strategy for cross-category matches (decision) — Resolved: modify in place.
   Refactor `promptShowSelection()` to return `(AssetMatch, error)` instead of calling `displayShowMatch()` directly. `runShowSearch()` receives the match and applies show-specific output. Scope section updated to allow `promptShowSelection` changes.

3. Scope out-of-scope contradiction for `promptShowSelection` (gap) — Resolved: fixed.
   Removed `promptShowSelection` from the out-of-scope list. The extraction requires changing its return type, which was correctly identified in section 8.

4. `resolveCrossCategory()` needs `stdin` for TTY detection (gap) — Resolved: acknowledged.
   The function signature already includes stdin. TTY detection is handled by `promptShowSelection` via `isTerminal(stdin)`. No action needed beyond ensuring stdin is passed through.

5. Exact-match-with-substring-check logic is subtle (risk) — Resolved: acknowledged.
   Lines 282-301 of `runShowSearch()` fall through to selection when a single exact match has additional substring matches across categories. Must be preserved in the extracted function with a dedicated test case.

6. Self-reference in scope section (gap) — Resolved: fixed.
   Changed "covered by `project.md`" to "covered by `2-read-feature.md`" in section 2.

7. Double auto-install after extraction (design) — Resolved: centralise in `resolveCrossCategory()`.
   Auto-install lives solely inside `resolveCrossCategory()`. The function returns the resolved match and the effective scope (`ScopeMerged` when auto-install occurred, otherwise the input scope). `runShowSearch()` calls `showVerboseItem(w, match.Name, effectiveScope, cat.key, cat.itemType)` unconditionally — no branching on source. `displayShowMatch()` is deleted. Section 6 step 4 and `runShowSearch()` refactor updated accordingly.

8. Function signature not explicitly specified (gap) — Resolved: resolver-parameter variant adopted. Superseded in part by issue 16.
   Original resolution: `resolveCrossCategory(query string, scope config.Scope, r *resolver) (AssetMatch, config.Scope, error)`. Issue 16 later dropped the scope parameter and the scope return value, yielding the final signature `resolveCrossCategory(query string, r *resolver) (AssetMatch, error)`. The resolver still carries cfg, flags, stdout, stderr, stdin — caller constructs it via `newResolver()` before the call. Auto-install is signalled via `r.didInstall`. Test pattern of `r.skipRegistry = true` before the call remains unchanged.

9. Auto-install responsibility after `promptShowSelection` refactor (gap) — Resolved: centralised in `resolveCrossCategory()`.
   All three "registry match" branches inside `resolveCrossCategory()` (step 5 exact registry, step 6 combined-search single match, and step 7 post-selection) funnel through a common auto-install block: `if selected.Source == AssetSourceRegistry { r.autoInstall(...); scope = config.ScopeMerged }`. A small helper (e.g. `r.installIfRegistry(match)` returning the effective scope) keeps the logic DRY. `promptShowSelection()` only returns the chosen match — it no longer installs.

10. Leading blank line placement (risk) — Resolved: keep in `runShowSearch()`.
    The `fmt.Fprintln(w)` at show.go:236 is show-specific cosmetics and stays in `runShowSearch()` before the `resolveCrossCategory()` call. Not moved into the extracted function. Constraint to be added to section 7.

11. Registry skip path for tests (gap) — Resolved: dissolved by issue 8 resolution.
    With resolver passed in, tests set `r.skipRegistry = true` on the constructed resolver before calling `resolveCrossCategory()`, matching the pattern in `resolve_test.go:256`. No new test hook needed.

12. Fall-through test case not in acceptance criteria (gap) — Resolved: added.
    Section 10 updated with: "A test case covers the fall-through where a single exact match coexists with additional substring matches across categories, asserting that selection is presented (TTY) or an ambiguity error returned (non-TTY) rather than the exact match auto-shown."

19. Registry auto-install paths lack unit-test coverage (gap) — Resolved: accepted.
    `show_test.go` has no tests for auto-install (`rg 'autoInstall|AssetSourceRegistry|ensureIndex' internal/cli/show_test.go` returns nothing). The dedupe refactor moves the existing `r.autoInstall(...)` calls verbatim into a centralised helper inside `resolveCrossCategory`; it does not change auto-install behaviour. Risk of extraction regressing auto-install is low, and adding test injection points for a mocked registry client is disproportionate to the risk. Gap accepted; manual verification covers the auto-install paths for this project. Future work adding auto-install logic changes should add coverage then.

18. `showCategoryFor` nil-return hazard in new callers (risk) — Resolved: defensive nil-check at both call sites.
    Implementation plan step 5 updated with `if cat == nil { return fmt.Errorf("unknown category %q", match.Category) }` after `showCategoryFor(match.Category)`. Same guard required in `read.go`. Cross-reference added to `2-read-feature.md` implementation guidance.

17. Writer routing contract for `resolveCrossCategory` undocumented (gap) — Resolved: contract added to doc comment.
    Implementation plan step 1 updated with the required doc comment block specifying r.stdout receives fetch progress, install notices, and interactive prompts; r.stderr receives debug output only. Guidance for clean-stdout callers (e.g. `read`) included inline.

16. Scope parameter consistency with `r.cfg` not enforced (design) — Resolved: scope parameter dropped.
    Signature simplified to `resolveCrossCategory(query string, r *resolver) (AssetMatch, error)`. The function signals auto-install via `r.didInstall` (existing resolver field at resolve.go:59). Callers read `r.didInstall` after the call and flip their own scope to `ScopeMerged` if needed. Removes the inconsistency hazard entirely — no scope argument to mismatch. Requirements 1 and 3, plus implementation plan steps 1 and 5, updated.

15. `ensureIndex()` call ordering must be preserved (risk) — Resolved: ordering pinned in plan.
    Section 6 step 2 updated to require `r.ensureIndex()` stays positioned after step 2 (installed substring search) and only fires when installed-only resolution fails to produce a single match. The show.go:322-324 gate is preserved verbatim. Prevents offline regression and unnecessary network calls on fast-path resolutions.

14. Post-refactor `promptShowSelection` signature (gap) — Resolved: resolver-sourced writers.
    New signature: `promptShowSelection(r *resolver, matches []AssetMatch, query string) (AssetMatch, error)`. Uses `r.stdout` and `r.stdin`, matching `promptAssetSelection` at resolve.go:517. The `scope` parameter is dropped — it was only used to pass to the deleted `displayShowMatch`. Section 6 step 3 updated with the explicit signature.

13. Resolver progress-writer routing for `read` (gap) — Resolved: per-command wiring documented.
    `read` constructs its resolver with stderr in the stdout slot: `newResolver(cfg, flags, stderr, stderr, stdin)`. This routes registry fetch/install progress lines to stderr and keeps stdout reserved for asset content. `show` keeps the existing `newResolver(cfg, flags, stdout, stderr, stdin)` wiring. No change to `resolve.go` or `tui.go`. Guidance belongs in `2-read-feature.md` section 8; cross-referenced here because `resolveCrossCategory()` is the consumer.

## 10. Acceptance Criteria

- [ ] A shared `resolveCrossCategory()` function (or equivalent) exists that resolves a query to a single `AssetMatch`
- [ ] `runShowSearch()` calls the shared function and applies show-specific output
- [ ] All existing tests in `show_test.go` pass without modification
- [ ] The shared function is callable from `read.go` without importing show-specific types or functions
- [ ] `go build ./...` succeeds
- [ ] `scripts/invoke-tests` passes
- [ ] `internal/cli/cross_resolve_test.go` exists with direct unit tests using `newTestResolver()` (skipRegistry = true) covering: zero matches, single installed exact match, ambiguous short-name exact (non-TTY), single installed substring match, combined-search multiple matches (non-TTY), and the fall-through where a single exact match coexists with additional substring matches across categories (asserting that selection is presented in TTY or an ambiguity error returned in non-TTY rather than the exact match auto-shown)
