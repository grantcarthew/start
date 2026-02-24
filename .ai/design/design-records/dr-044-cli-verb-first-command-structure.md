# DR-044: CLI Verb-First Command Structure

- Date: 2026-02-23
- Status: Accepted
- Category: CLI
- Supersedes: dr-017 (CLI Show Command), dr-029 (CLI Configuration Editing Commands)

## Problem

`start config` uses a noun-first command structure where category subcommands group verbs underneath them:

```
start config agent edit claude
start config role add
start config context remove project
```

`start assets` uses verb-first where verbs are top-level and targets are arguments:

```
start assets add role/go-expert
start assets list
start assets search golang
```

The inconsistency makes the CLI feel awkward. Users must mentally switch patterns between `config` and `assets`. The noun-first structure also buries common operations (edit, remove) two levels deep and prevents natural search-by-name workflows.

`start show` has the same problem — noun subcommands (`show agent`, `show role`) that are redundant given `show` already does cross-category search by name.

## Decision

Restructure `start config` and `start show` to use verb-first commands. Remove all noun-group subcommands. Introduce search-by-name with interactive menus for commands that target existing items.

### start config command structure

```
start config                          # list effective config (unchanged)
start config add [category]           # add item; prompts for category if omitted
start config edit [query]             # search by name, menu if multiple, then edit; prompts if no query
start config remove [query]           # search by name, menu if multiple, confirm, delete; prompts if no query
start config list [category]          # list items; all categories if omitted
start config info [query]             # search by name, show raw config fields; prompts if no query
start config open [category]          # open .cue file in $EDITOR; prompts if omitted
start config order [category]         # reorder items; prompts if omitted (contexts/roles only)
start config search [query]           # search by keyword across names, descriptions, tags
start config settings [key] [value]   # manage settings (unchanged)
```

### start show command structure

```
start show [query]    # search by name, show resolved content; list all if no query
```

Noun subcommands (`show agent`, `show role`, `show context`, `show task`) removed.

### No-argument behaviour

| Command | No argument |
| --- | --- |
| `start config` | list all (unchanged) |
| `start config list` | list all |
| `start config info` | prompt category → item → display raw fields |
| `start config add` | prompt for category |
| `start config edit` | prompt interactively |
| `start config open` | prompt for category |
| `start config order` | prompt for category |
| `start config remove` | prompt category → item picker → confirmation → delete |
| `start show` | list all (unchanged) |

### Category arguments

Commands that accept a category (`add`, `list`, `open`, `order`) use singular as canonical with plural as alias:

```
start config add agent    # canonical
start config add agents   # alias
```

Valid categories: `agent`, `role`, `context`, `task` (and `setting`/`settings` for `open`).

### Removed commands

- `start config agent` and all subcommands (`add`, `default`, `edit`, `info`, `list`, `remove`)
- `start config role` and all subcommands
- `start config context` and all subcommands
- `start config task` and all subcommands
- `start config agent default` (use `start config settings default_agent` instead)
- `start show agent`, `start show role`, `start show context`, `start show task`

### add and edit are always interactive

`config add` and `config edit` have no field-specific flags. All input is gathered via interactive prompts. Users who need scripted or non-interactive config changes should use `config open` to edit the CUE file directly.

`config remove` retains `--yes` / `-y` to skip the confirmation prompt, as this is useful in scripts and is not a field flag.

### config order with non-orderable categories

Only `context` and `role` support ordering. If `config order agent` or `config order task` is supplied, present the same menu as `config order` with no argument — prompt the user to choose context or role.

### Distinction between config info and start show

- `start config info <query>` — raw stored config fields (name, file path, tags, command template)
- `start show <query>` — resolved content after global+local merge, file read, command execution

Both use search-by-name with an interactive menu on multiple matches.

## Why

Verb-first matches how users think about tasks. A user knows they want to edit something before they know the category. `start config edit` → guided from there is natural. `start config agent edit` → requires knowing the category upfront.

Verb-first also aligns `config` with `assets`, making the full CLI consistent. Users learn one pattern and apply it everywhere.

Search-by-name with menus (already used by `start task`) is more forgiving than requiring exact category prefixes. The menu provides the exact-match step without forcing the user to remember the category.

`start show` noun subcommands were redundant — `start show <name>` already searched cross-category. The subcommands only existed as category filters, which the menu on multiple matches handles naturally.

## Trade-offs

Accept:

- Breaking change — all existing `start config agent/role/context/task` command paths stop working
- Users familiar with the old structure must relearn
- More implementation work than a purely additive change

Gain:

- Single consistent pattern across the entire CLI
- Shallower command depth — most operations are one level deep
- Search-by-name is more discoverable than memorising category prefixes
- `start config edit` and `start show` become equivalent in interaction style

## Alternatives

Keep noun-first and add aliases:

- Pro: No breaking change
- Con: Doubles the command surface, both patterns coexist permanently
- Con: Documentation and completion must cover both
- Rejected: Aliases perpetuate the inconsistency

Noun-first for config, verb-first for assets (status quo):

- Pro: No change required
- Con: The inconsistency remains and is felt by every user switching between the two
- Rejected: The inconsistency is the problem being solved
