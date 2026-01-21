# Go Code Duplication Review

Date: 2026-01-21

## Summary

This review analyses code duplication in the `start` CLI codebase using `golangci-lint` with the `dupl` linter enabled. The tool identified **28 duplication issues** across the codebase, primarily concentrated in the CLI configuration commands.

## Tool Output

```
golangci-lint run --enable dupl
```

## Findings by Category

### 1. High-Priority: Config Command Handlers (Structural Duplication)

The most significant duplication exists in the config subcommands (`config_agent.go`, `config_role.go`, `config_context.go`, `config_task.go`). These files follow nearly identical patterns for CRUD operations on different asset types.

#### 1.1 List Command Duplication

**Files:**
- `internal/cli/config_agent.go:71-123`
- `internal/cli/config_role.go:69-121`

**Pattern:** Both `runConfigAgentList` and `runConfigRoleList` share identical structure:
1. Load items for scope
2. Check if empty
3. Get default item
4. Sort and print with default marker

**Recommendation:** Extract a generic list function parameterised by item type. Consider:

```go
type ListableConfig interface {
    GetName() string
    GetDescription() string
    GetSource() string
    GetOrigin() string
}

func runConfigList[T ListableConfig](
    cmd *cobra.Command,
    loadFn func(bool) (map[string]T, error),
    getDefaultFn func(cue.Value) string,
    itemType string,
) error
```

**Priority:** Medium - the duplication is clear but the code is stable and changes are localised.

#### 1.2 Remove Command Duplication

**Files:**
- `internal/cli/config_agent.go:565-631`
- `internal/cli/config_context.go:661-727`
- `internal/cli/config_role.go:604-670`
- `internal/cli/config_task.go:613-679`

**Pattern:** All four remove handlers follow identical flow:
1. Resolve config paths
2. Load items from directory
3. Check existence
4. Confirm removal (TTY check, prompt)
5. Delete and write file

**Recommendation:** Extract a shared `runConfigRemove` function:

```go
func runConfigRemove[T any](
    cmd *cobra.Command,
    args []string,
    itemType string,
    loadFn func(string) (map[string]T, error),
    writeFn func(string, map[string]T) error,
    filename string,
) error
```

**Priority:** High - four near-identical implementations with ~67 lines each.

#### 1.3 Default Command Duplication

**Files:**
- `internal/cli/config_agent.go:650-711`
- `internal/cli/config_role.go:689-750`

**Pattern:** Both handle showing/setting defaults identically:
1. Resolve paths
2. Show current default if no args
3. Verify item exists
4. Write setting

**Recommendation:** Extract shared logic, parameterised by setting key and loader function.

**Priority:** Medium - only two instances, ~62 lines each.

#### 1.4 LoadForScope Duplication

**Files:**
- `internal/cli/config_agent.go:727-772`
- `internal/cli/config_context.go:744-787`
- `internal/cli/config_role.go:765-808`
- `internal/cli/config_task.go:695-738`

**Pattern:** All `loadXXXForScope` functions follow identical merging logic:
1. Resolve paths
2. If localOnly, load from local
3. Else, load global first, then overlay local

**Recommendation:** This is the strongest candidate for generics:

```go
func loadConfigForScope[T any](
    localOnly bool,
    loadFromDir func(string) (map[string]T, error),
    setSource func(*T, string),
) (map[string]T, error)
```

**Priority:** High - four identical implementations, clear type parameterisation.

### 2. Medium-Priority: Show Command Handlers

**Files:**
- `internal/cli/show.go:211-256` (`prepareShowRole`)
- `internal/cli/show.go:351-396` (`prepareShowAgent`)

**Pattern:** Both functions:
1. Load config
2. Look up item collection
3. Iterate to collect names
4. Determine which to show
5. Format content

**Recommendation:** Extract shared logic. The `prepareShowContext` and `prepareShowTask` functions have slight variations (list-only mode, substring matching) so full consolidation requires care.

**Priority:** Medium - only two highly similar functions; others have different behaviour.

### 3. Low-Priority: Test Duplication

**Files:**
- `internal/cli/config_test.go:760-800` vs `config_test.go:802-842` vs `config_test.go:340-381`
- Multiple similar test setup blocks across `config_test.go`

**Pattern:** Test functions repeat the same setup:
```go
tmpDir := t.TempDir()
t.Setenv("XDG_CONFIG_HOME", tmpDir)
globalDir := filepath.Join(tmpDir, "start")
os.MkdirAll(globalDir, 0755)
os.WriteFile(filepath.Join(globalDir, "settings.cue"), content, 0644)
origWd, _ := os.Getwd()
defer os.Chdir(origWd)
os.Chdir(tmpDir)
```

**Recommendation:** Create test helper:

```go
func setupTestConfig(t *testing.T, files map[string]string) (cleanup func())
```

However, test clarity often benefits from explicit setup. The Go testing philosophy prefers duplication over abstraction in tests.

**Priority:** Low - test duplication is generally acceptable for readability.

### 4. Acceptable Duplication

The following duplications were identified but are considered acceptable:

#### 4.1 Write File Functions

- `writeAgentsFile`, `writeRolesFile`, `writeContextsFile`, `writeTasksFile`

These write different CUE structures with type-specific field handling. While structurally similar, they handle different fields and output formats. Consolidation would require complex templating or reflection.

**Verdict:** Keep separate - each asset type has different fields.

#### 4.2 LoadFromDir Functions

- `loadAgentsFromDir`, `loadRolesFromDir`, `loadContextsFromDir`, `loadTasksFromDir`

These read different CUE paths and populate type-specific structs. Consolidation would require reflection or code generation.

**Verdict:** Keep separate - the CUE field extraction is type-specific.

## Consolidation Approaches

### Recommended: Generics for Core Patterns

Go 1.18+ generics are ideal for the identified patterns. Key candidates:

1. **`loadConfigForScope[T any]`** - The merging logic is identical across all four types
2. **`runConfigRemove[T any]`** - Confirmation and deletion flow is type-agnostic
3. **`runConfigList[T ListableConfig]`** - List display with interface constraint

### Not Recommended: Interface Abstraction for Write/Load

The `writeXXXFile` and `loadXXXFromDir` functions handle type-specific fields. Abstracting these would require:
- Reflection (runtime cost, type safety loss)
- Code generation (build complexity)
- Interface with many methods (over-engineering)

The current approach keeps field handling explicit and type-safe.

## Metrics

| Category | Files | Lines Duplicated | Instances |
|----------|-------|------------------|-----------|
| List handlers | 2 | ~53 | 2 |
| Remove handlers | 4 | ~67 | 4 |
| Default handlers | 2 | ~62 | 2 |
| LoadForScope | 4 | ~45 | 4 |
| Show handlers | 2 | ~46 | 2 |
| Test setup | 3+ | ~42 | 10+ |

**Total duplication issues:** 28 (per golangci-lint)

## Action Items

### Should Address

1. **Extract `loadConfigForScope[T]` generic helper** - Clear win, type-safe, reduces ~180 lines to ~45
2. **Extract `runConfigRemove[T]` generic helper** - Four identical implementations, reduces ~268 lines to ~70
3. **Consider test helper for config setup** - Optional, balance against test readability

### May Address (Optional)

1. **Consolidate list handlers** - Lower priority, only two instances
2. **Consolidate default handlers** - Lower priority, only two instances
3. **Consolidate show handlers** - Requires care due to behavioural differences

### Keep as Is

1. **Write functions** - Type-specific field handling justifies separation
2. **LoadFromDir functions** - CUE field extraction is inherently type-specific
3. **Test duplication** - Explicit test setup aids debugging

## Conclusion

The codebase has moderate duplication concentrated in the CLI config commands. The most impactful consolidation targets are:

1. `loadConfigForScope` (4 instances, perfect generics candidate)
2. `runConfigRemove` (4 instances, straightforward extraction)

These two refactors would eliminate approximately 400 lines of duplicate code while maintaining type safety through Go generics. The remaining duplication is either acceptable (type-specific logic) or low-priority (tests).

The current duplication does not indicate a maintenance problem - the code is consistent and changes would be localised. Consolidation should be driven by maintenance needs rather than theoretical concerns.
