# Todo: Fields Absent from Interactive Add

## Background

p-034 removed all field-specific flags from `config <type> add` and `config <type> edit`. The `add` commands were simplified to prompt only for the minimum fields needed to create a valid entity. Several optional fields that were previously settable via flags are now silently zeroed on creation with no indication to the user. This document describes the gap, its runtime impact, and the options for resolving it.

## Affected Fields

### config agent add

Fields NOT prompted during `add`:

- `Models` — map of alias to model ID (e.g. `sonnet: "claude-sonnet-4-5-20251001"`)
- `Tags` — string slice, used for filtering in interactive search menus

Fields that ARE prompted: `name`, `bin`, `command`, `default_model`, `description`

### config role add

Fields NOT prompted during `add`:

- `Optional` — bool; when true, a missing role file at runtime is silently skipped rather than causing an error
- `Tags` — string slice, used for filtering

Fields that ARE prompted: `name`, `description`, content source (file/command/prompt)

### config context add

Fields NOT prompted during `add`:

- `Tags` — string slice, used for filtering

Fields that ARE prompted: `name`, `description`, content source, `required`, `default`

### config task add

Fields NOT prompted during `add`:

- `Tags` — string slice, used for filtering

Fields that ARE prompted: `name`, `description`, content source, `role`

## Runtime Impact

### Agent: Models vs DefaultModel

`DefaultModel` is prompted during `add`. `Models` is not. At runtime, `executor.go` resolves the model like this:

1. Use the model provided at invocation time, or fall back to `DefaultModel`
2. If the resolved model string is a key in the `Models` map, replace it with the mapped value
3. Otherwise, pass the string through unchanged

This means `DefaultModel = "sonnet"` with an empty `Models` map is not a runtime error. The string `"sonnet"` is passed directly to the command template as `{{.model}}`. Whether that works depends entirely on whether the underlying AI CLI tool accepts `"sonnet"` as a valid model identifier.

The problem is agent-specific. The Claude CLI accepts aliases like `sonnet`. Other tools may require full model IDs like `claude-sonnet-4-5-20251001`. A user who sets `DefaultModel` without configuring `Models` may get silent failures or unexpected model selection until they run `config agent edit` to add mappings.

This is not a data integrity issue — the configuration is valid — but it creates a gap between what the user typed and what the agent actually uses that is not visible at add time.

### Role: Optional flag and missing files

`Optional` defaults to false. A role added with `optional` not set will cause a hard error at runtime if its file source is missing. This is the most operationally significant gap:

- A user who adds a role with `--file ~/.config/start/roles/go-expert.md` on a machine where that file exists may deploy to a second machine where it does not
- Without `optional: true`, the runtime fails with an error rather than skipping the role
- The user has no indication during `config role add` that this field exists or that it is relevant to their file-based role

### Tags

Tags affect only the interactive search menus within `start config`. They have no runtime impact on agent execution, role loading, context injection, or task dispatch. A role, agent, context, or task without tags functions identically to one with tags — it simply will not appear in tag-filtered search results.

Missing tags at add time is a minor UX inconvenience, not a correctness concern.

## Options

### Option 1: Do nothing

Accept the two-step workflow. Users who need models, optional, or tags run `config <type> edit <name>` after adding. The comment `// models and tags are empty on add; use edit to add them` in the source is the only documentation of this.

Pros:
- `add` remains short and simple
- Consistent with the p-034 design decision
- Does not delay p-032

Cons:
- Silent gap: users have no indication these fields exist
- The `DefaultModel` + empty `Models` combination is a footgun for non-Claude agents
- The `Optional` omission from role add is a latent runtime failure for file-based roles deployed across machines

### Option 2: Prompt for all fields during add

Add the missing prompts to each add function. `promptTags` already exists and is lightweight. For agent `add`, add `promptModels` (keep/clear/edit sub-menu). For role `add`, add a boolean prompt for `optional`.

Pros:
- `add` is comprehensive; nothing is hidden from the user
- Eliminates the `DefaultModel`/`Models` footgun at the point where it matters

Cons:
- Makes `add` significantly longer, especially for agents (models sub-menu is complex)
- `promptModels` with `current = nil` uses "keep/clear/edit" — "keep" and "clear" are meaningless on an empty map; the prompt would need a variant for the add case
- Tags during add is premature in most workflows; most users do not tag items at creation time

### Option 3: Contextual prompts only

Prompt for the missing fields only when they are contextually relevant:

- Prompt for `Optional` in `config role add` only when the user chooses a file content source (it is irrelevant for command or inline prompt sources)
- Prompt for `Models` in `config agent add` only when the user enters a non-empty `DefaultModel` (the footgun only exists when `DefaultModel` is set)
- Do not prompt for `Tags` in any add function; add a note in the completion message

This gives targeted coverage of the two operationally significant gaps without expanding the add flow for the majority of cases.

Completion message example for role add with file source:

```
Added role "go-expert" to global config.
Run 'start config role edit go-expert' to set optional or tags.
```

Completion message example for agent add with default model set:

```
Added agent "gemini" to global config.
Models map is empty — run 'start config agent edit gemini' to map model aliases.
```

Pros:
- Covers the two gaps that have runtime impact
- Does not add complexity for users who do not need these fields
- Tags are omitted intentionally (no runtime impact, premature)

Cons:
- Partial solution; tags remain absent from add with no prompt
- Requires conditional logic in add functions (role: check content source type; agent: check whether default model was entered)
- Completion message hints can be overlooked

### Option 4: Post-add hint only

No change to the add prompts. After a successful add, print a single-line hint that directs the user to edit for additional fields. No conditional logic required.

Agent add completion:
```
Added agent "claude" to global config.
To set model aliases or tags: start config agent edit claude
```

Role add completion (always shown, not conditional on file source):
```
Added role "go-expert" to global config.
To set optional or tags: start config role edit go-expert
```

Pros:
- Zero impact on add flow complexity
- Users are informed at exactly the right moment
- No conditional prompt logic required
- Easiest to implement

Cons:
- Does not prevent the `DefaultModel`/`Models` footgun — user still has to take action
- Does not surface the `Optional` risk specifically for file-based roles
- Hint may be overlooked or scrolled past

## Recommendation

Option 3 covers the two operationally significant gaps (agent models footgun, role optional) with minimal complexity. Option 4 is the lowest-effort path and good enough if the priority is to close p-032 first.

Tags are not worth prompting for during add regardless of which option is chosen. They serve filtering only, and asking for them upfront adds noise for new users who have not yet built a library of tagged items to filter.

The `Optional` field for roles is the highest-priority gap because it is a silent runtime failure mode for users who add file-based roles and deploy across multiple machines.

## Relationship to p-032

This gap exists in the current noun-first CLI and will carry forward into the verb-first CLI unless addressed. p-032 should note this as a pre-existing gap and either:

- Resolve it before the verb-first refactor (add a small p-035)
- Record it as a known limitation in the p-032 design notes and close it in a follow-up project

The fix is contained entirely within the four `runConfig*Add` functions and does not interact with the structural changes in p-032.

## slowReader Test Helper

`config_testhelpers_test.go` introduces `slowReader`, which limits each `Read` call to one byte. This prevents `bufio.NewReader` instances in sequential prompt functions from over-consuming the underlying reader and stealing input intended for the next prompt.

This is the correct solution for the current structure where each prompt function creates its own `bufio.NewReader`. Two future scenarios are worth noting:

- If prompt functions are refactored to share a single `bufio.NewReader`, `slowReader` becomes unnecessary dead code in tests.
- If a prompt function ever needs to read large input efficiently (e.g. paged input), `slowReader` would become a test bottleneck.

p-032 will restructure the prompt infrastructure and should revisit whether `slowReader` remains the right approach or whether shared reader management is preferable.
