# DR-040: CLI Config Reorder Command

- Date: 2026-02-08
- Status: Accepted
- Category: CLI

## Problem

Users cannot reorder configuration items without manually editing CUE files. The display and injection order of config items is determined by:

1. CUE field definition order in the file (read order)
2. Alphabetical sorting in write functions (write order)

This creates two issues:

- Context injection order (which affects AI behaviour per DR-008) cannot be controlled via CLI
- All write operations (add, edit, remove) on contexts and roles reset order to alphabetical, destroying any manual reordering the user has done in the CUE file

Order matters for:

- Contexts: injection order to the AI agent (general instructions before specific)
- Roles: display and selection order (optional roles typically listed first)

Order does not matter for:

- Agents: selection is explicit (interactive selection planned in issue #18) or via default setting
- Tasks: no behaviour depends on task order

## Decision

Add an `order` action to context and role commands following the existing noun-verb pattern from DR-029.

Command structure:

```
start config context order [--local]
start config context reorder [--local]   # alias
start config role order [--local]
start config role reorder [--local]      # alias
start config order                       # prompts user to choose contexts or roles
start config reorder                     # alias
```

Interaction model: simple numbered list with repeated move-up operations using bufio.Reader. No TUI library, no new dependencies.

Prerequisites (applied to contexts and roles only):

- Write functions accept an explicit order parameter and stop alphabetically sorting
- Roles load function updated to return definition order (contexts already does this)

Agents and tasks are unaffected. Their write functions continue to sort alphabetically.

Scope handling:

- Without --local: reorders items in global config (~/.config/start/)
- With --local: reorders items in local config (./.start/)
- Heading displays scope and file path so the user knows which file is being modified

## Why

Simple numbered list interaction:

- No new dependencies required (uses existing bufio.Reader pattern)
- Fits existing interactive patterns in the codebase
- Move-up-one-at-a-time is easy to understand and produces no input errors
- Repeated moves achieve any desired order

Contexts and roles only:

- These are the only entity types where order has semantic meaning
- Agents will use interactive selection (issue #18), removing the need for order-based fallback
- Tasks have no order-dependent behaviour
- Reduces scope of write function changes

Explicit order parameter on write functions:

- Every caller passes the order they want, keeping write functions simple and stateless
- Testable: order is an input, not hidden state read from the file system
- Add appends new item to end of order, remove drops from order, order command sets new order

Per-scope reordering (not merged):

- Global and local files are edited independently (DR-029)
- Simple mental model: reorder items within one file

## Trade-offs

Accept:

- Move-up-only interaction requires more steps than typing a full new order for large reorderings
- Write function signature change for contexts and roles touches multiple call sites
- Roles load function signature change ripples through role callers

Gain:

- Users can control context injection order and role display order via CLI
- Zero new dependencies
- Order preserved across add/edit/remove operations (no more alphabetical reset)
- Simple, low-risk interaction model with clear undo (just keep moving)

## Alternatives

charmbracelet/bubbletea TUI:

- Pro: Rich interactive experience with real-time cursor navigation
- Pro: Industry-standard Go TUI library
- Con: New dependency tree for a single use case
- Con: No pre-built reorderable list component; custom code still needed
- Rejected: Disproportionate dependency for the interaction required

Raw ANSI terminal handling without bubbletea:

- Pro: No new dependency
- Pro: Interactive cursor-based navigation
- Con: Significant custom terminal code to write and maintain
- Con: Must handle terminal resize, raw mode, signal cleanup manually
- Rejected: Complexity and maintenance burden not justified

Type full new order (e.g. "3 1 5 2 4"):

- Pro: Single input to achieve any order
- Con: Error-prone with many items (must type every number)
- Con: Harder to validate and explain
- Rejected: Move-up-one-at-a-time is simpler and less error-prone

## Execution Flow

When `start config context order [--local]` is executed:

1. Determine scope from --local flag (default: global)
2. Load contexts and definition order from the target scope directory
3. If no items found, display message and exit
4. Display numbered list with scope and file path in heading
5. Prompt: `Move up (number), Enter to save, q to cancel: `
6. Process input (case insensitive):
   - Empty input (Enter): write file with current order, display confirmation, exit
   - `q`, `quit`, or `exit`: discard changes, display cancellation message, exit
   - Valid number (2 or greater, within range): swap item with the one above it, re-display list, re-prompt
   - Number 1: display "already at top" message, re-prompt
   - Invalid input: display error message, re-prompt
7. After writing: parse generated CUE to verify syntax

When `start config order` is executed:

1. Prompt user to choose between contexts or roles
2. Run the reorder flow for the selected type

## Structure

### Display Format

```
Reorder Contexts (global - ~/.config/start/contexts.cue):

  1. cwd/agents-md             [required] [default]
  2. cwd/project               [default]
  3. dotai/cwd/context/index
  4. dotai/home/context/index
  5. dotai/home/environment    [required] [default]

Move up (number), Enter to save, q to cancel: 3

  1. cwd/agents-md             [required] [default]
  2. dotai/cwd/context/index
  3. cwd/project               [default]
  4. dotai/home/context/index
  5. dotai/home/environment    [required] [default]

Move up (number), Enter to save, q to cancel:
```

### Write Function Changes

Current signature pattern (contexts):

```
writeContextsFile(path string, contexts map[string]ContextConfig) error
```

New signature pattern:

```
writeContextsFile(path string, contexts map[string]ContextConfig, order []string) error
```

Same change for writeRolesFile. Agents and tasks write functions unchanged.

Behaviour:

- Write fields in the provided order
- All callers must provide order:
  - `add`: read existing order, append new item
  - `remove`: read existing order, drop removed item
  - `edit`: read existing order, pass through unchanged
  - `order`: pass user's reordered list

### Load Function Changes

Current:

| Function | Returns |
|----------|---------|
| loadRolesForScope | (map[string]RoleConfig, error) |
| loadContextsForScope | (map[string]ContextConfig, []string, error) |

Updated:

| Function | Returns |
|----------|---------|
| loadRolesForScope | (map[string]RoleConfig, []string, error) |
| loadContextsForScope | (map[string]ContextConfig, []string, error) |

Agents and tasks load functions unchanged.

### Input Validation

All text input is case insensitive. Accepted inputs at the prompt:

| Input | Action |
|-------|--------|
| Empty (Enter) | Save current order and exit |
| `q`, `quit`, `exit` | Discard changes and exit |
| Number 2 to N | Move that item up one position |
| Number 1 | Display "already at top" message |
| Anything else | Display error, re-prompt |

### Command Aliases

- `order` (primary)
- `reorder` (alias)
