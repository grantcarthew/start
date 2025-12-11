# DR Writing Guide

Creating and maintaining Design Records.

Location: `docs/design/design-records/dr-NNN-title.md`

Read when: Writing/updating DRs or reconciling docs.

---

## Markdown Formatting Note

IMPORTANT: Avoid using bold Markdown (`**text**`) in design records. Bold formatting adds no semantic value to large language models but significantly increases token count. Use section headers, lists, and clear structure instead.

Good: Use `## Section Name` and bullet points
Bad: Use `**Section Name:**` with bold

Note: CLI documentation (docs/cli/) is intended for human readers and should use bold for readability. This guidance applies only to DR and Project documents that are primarily consumed by AI agents.

---

## DR Schema

DR structure:

### Header

```markdown
# DR-NNN: Title

- Date: YYYY-MM-DD
- Status: Proposed | Accepted | Superseded | Deprecated
- Category: Configuration | Tasks | CLI | Agents | etc.
```

### Required Sections

Problem:

- What constraint or issue drove this decision?
- What forces are at play?
- What problem does this solve?

Decision:

- Clear, specific statement of what was decided
- Should be implementable and testable

Why:

- Core reasoning behind this choice
- Why is this the right solution for our context?
- Supporting details that explain the decision

Trade-offs:

- Accept: What costs, limitations, or complexity are we accepting?
- Gain: What benefits, simplicity, or capabilities do we get?

Alternatives:

- What other options were considered?
- Why were they rejected?
- What were their trade-offs?

### Optional Sections

Add as needed:

- Structure - Schema definitions, field descriptions
- Scope - Where/how this applies (global vs local, etc.)
- Rationale - Detailed reasoning and context
- Usage Examples - How to use the decision in practice
- Validation - Rules for correctness
- Execution Flow - Step-by-step behavior
- Implementation Notes - High-level guidance for implementers (not code)
- Security - Security considerations and threat model
- Breaking Changes - Updates from previous versions
- Updates - Historical changes with dates

---

## What Belongs in DRs

### ✅ Configuration Examples

Config structure and schema (TOML, JSON, etc.):

```toml
[agents.claude]
bin = "claude"
command = "{bin} --model {model} '{prompt}'"
default_model = "sonnet"

  [agents.claude.models]
  haiku = "claude-3-5-haiku-20241022"
  sonnet = "claude-3-7-sonnet-20250219"
```

This is NOT implementation code - it's the schema/structure being defined.

### ✅ Usage Examples

Behavior and command usage:

```bash
start --agent claude --model sonnet
start task code-review "focus on security"
```

### ✅ Field Descriptions

Field meanings and constraints:

role (string, optional):

- Name reference to a role defined in `[roles.<name>]`
- Example: `"code-reviewer"` (references `[roles.code-reviewer]`)
- If omitted: Uses `default_role` from settings

### ✅ Validation Rules

Validation requirements:

At configuration load:

- Task name matches pattern: `/^[a-z0-9]+(-[a-z0-9]+)*$/`
- At least one of `file`, `command`, or `prompt` present
- `role` field (if present) references existing role name

### ✅ Execution Flows

Step-by-step algorithms:

When `start task <name>` is executed:

1. Select role:
   - CLI `--role` flag → use it
   - Else task `role` field → use it
   - Else `default_role` setting → use it

### ✅ Tables and Matrices

Scope and behavior matrices:

| Placeholder | Agent Commands | Roles | Contexts | Tasks |
| ----------- | -------------- | ----- | -------- | ----- |
| {model}     | ✓              | ✓     | ✓        | ✓     |
| {prompt}    | ✓              | -     | -        | -     |

### ✅ Breaking Changes Notes

Historical changes:

Breaking Changes from Original:

1. Changed: `role` field now references role name (not file path)
2. Removed: `documents` array (auto-includes required contexts)
3. Added: `agent` field for agent selection

### ✅ Updates Section

Dated updates:

Updates:

- 2025-01-04: Changed from hardcoded tier names to flexible user-defined model names
- 2025-01-05: Changed from global-only to both global + local support

---

## What Does NOT Belong in DRs

### ❌ Implementation Code

Do not include actual source code (Go, Python, etc.):

```go
// Bad: Implementation code
func LoadConfig(path string) (*Config, error) {
    // ... implementation ...
}
```

Why: DRs capture decisions, not implementation.

Exception: Pseudocode in Why section for complex logic.

### ❌ Cross-Links to Other DRs

Do not link to other DR documents:

Why: Links break when DRs change.

Instead: Use `design-records/README.md` index.

Exception: Link when status changes (superseding/superseded):

```markdown
Status: Superseded by [DR-042](./dr-042-new-approach.md)
```

### ❌ User Documentation Duplication

Do not duplicate content from user docs:

Bad: Duplicating full CLI docs

Good: Brief example + reference to full docs

---

## When to Create a DR

Always Create a DR for:

- Architectural decisions (component structure, data flow)
- Algorithm specifications (resolution order, search, matching)
- Breaking changes or deprecations
- Data formats, schemas, or protocols (TOML structure, field definitions)
- Public API or CLI command structure
- Security or performance trade-offs
- Major UX decisions (flag names, command organization)
- Multi-file vs single-file config decisions

Never Create a DR for:

- Simple bug fixes
- Documentation corrections (typos, clarifications)
- Code refactoring without behavior change
- Cosmetic changes
- Internal implementation details that don't affect external behavior

When Unsure, Ask:

Would a future developer need to understand WHY we made this choice?

- Yes → Create a DR
- No → Just fix it

---

## DR Numbering and Lifecycle

### File Naming Convention

Format: `dr-<NNN>-<category>-<title>.md`

- `NNN` = Three-digit number with leading zeros (001, 002, 003...)
- `category` = Technology/component/area (cue, cli, index, module, task, config, etc.)
- `title` = KISS description of the decision
- All lowercase kebab-case (words separated by hyphens, no underscores or spaces)

Examples:

- `dr-001-cue-user-controlled-defaults.md`
- `dr-002-cue-no-name-field.md`
- `dr-003-index-category-structure.md`
- `dr-004-module-naming-convention.md`

### Numbering

- Sequential: DR-001, DR-002, DR-003, etc.
- Gaps are acceptable (superseded/deprecated DRs)
- Never reuse numbers
- Get next number from `design-records/README.md` index

### Status Values

- Proposed - Under consideration, not implemented
- Accepted - Approved and in use (or in progress)
- Superseded - Replaced by newer DR, link in header
- Deprecated - No longer recommended, may exist in legacy code

---

## Writing a Good DR

### Focus on "Why" Not "How"

Include detailed reasoning:

```markdown
## Decision

Use TOML for all configuration files

## Why

- Human-readable and editable
- No whitespace sensitivity (unlike YAML)
- Excellent Go support via BurntSushi/toml
- Supports comments and complex nested structures
```

### Be Specific

Include concrete details and behavior:

```markdown
## Decision

Single configuration file with global + local merge strategy

## Merge Behavior

- Local config merges with global
- Same keys in local override global values
- New keys in local are added
- Omitted keys use global defaults
```

### Document Trade-offs Honestly

List both costs and benefits:

```markdown
## Trade-offs

Accept:

- Users must learn TOML syntax
- More complex than simple key=value files
- Parsing requires external library

Gain:

- Comments support (critical for user guidance)
- Nested structures for complex config
- No whitespace errors (unlike YAML)
```

### Consider Alternatives Seriously

Analyze with pros, cons, and rejection reasoning:

```markdown
## Alternatives

YAML:

- Pro: Widely known, standard in DevOps
- Pro: Native Go support
- Con: Whitespace-sensitive, error-prone for hand-editing
- Con: Complex spec with surprising edge cases
- Rejected: Error-prone editing outweighs familiarity

JSON:

- Pro: Simple, universal
- Con: No comments (users can't document their config)
- Con: Less human-friendly (trailing commas, quoted keys)
- Rejected: Lack of comments is a dealbreaker
```

---

## Reconciliation Process

After 5-10 DRs or significant design changes:

1. Remove deprecated references: `rg "old-pattern" docs/`
2. Update DR index in `design-records/README.md` with current status
3. Check DR status accuracy (Proposed → Accepted, Deprecated, Superseded?)
4. Remove stale TODOs: `rg "TODO|TBD|to be written" docs/`
5. Verify examples match current schema
6. Remove any "Related Decisions" sections from DRs
