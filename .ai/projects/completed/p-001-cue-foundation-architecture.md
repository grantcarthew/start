# P-001: CUE Foundation & Architecture

- Status: Complete
- Started: 2025-12-01
- Completed: 2025-12-05

## Overview

Research CUE language capabilities and design the foundational architecture for the CUE-based start tool. This project establishes how we'll use CUE for configuration, validation, asset management, and schema definition.

This is the critical first project that determines whether CUE can fulfill the requirements that TOML could not: ordered configuration, built-in validation, type safety, and native package distribution.

## Decisions Made

### Foundation Technology

- **Pure CUE architecture** - No TOML, all configuration and schemas in CUE
- **CUE Central Registry** - Use standard OCI registries for module distribution
- **Git tags for versioning** - Idiomatic CUE approach, format: `<directory-path>/<version>`
- **Semantic versioning** - Required by CUE (v0.1.0, v0.2.0, etc.)

### Asset Distribution Architecture

- **Separate repository** - `start-assets` created at `./reference/start-assets`
- **Each asset = independent CUE module** - Own versioning, independent lifecycle
- **Lazy loading** - Download assets on first use (e.g., `start task code-review`)
- **Module path convention** - `github.com/grantcarthew/start-<type>-<name>@v<major>`
- **Module path prefix** - `github.com/grantcarthew/start` (clean branding, not `start-assets`)

### Repository Structure

- **Directory organization** - `tasks/category/item/` (e.g., `tasks/golang/code-review/`)
- **Module names include category** - `start-task-category-item@v0` (e.g., `start-task-golang-code-review@v0`)
- **Schemas in start-assets** - `start-assets/schemas/` (not separate repo)
- **Index format** - CUE not CSV (`index.cue`)
- **Index keys** - Use `category/item` format matching user input and directory structure
- **Index purpose** - Enable CLI search/discovery (OCI catalog API is disabled)

### Schema Hierarchy

- **Three layers**:
  1. Schemas (constraints) - `github.com/grantcarthew/start-schemas@v0`
  2. Assets (values conforming to schemas)
  3. User config (values + overrides)
- **Assets import schemas** - Validate against schemas during development
- **Publishing workflow** - Schemas published first, assets depend on them

### Context Ordering

- **Order matters** - ENVIRONMENT.md → INDEX.csv → AGENTS.md
- **Ordered lists in CUE** - Preserve context injection order
- **Sequential resolution** - Each config resolved and added in order

### Registry Configuration

- **Default registry** - Hard-coded to `github.com/grantcarthew/start-assets`
- **Asset repository** - `github.com/grantcarthew/start-assets` (source code location)
- **Module prefix** - `start` (module identity, not implementation detail)
- **User-configurable** - Can override registry location

### Repository Descriptions

- **start** - "Context-aware AI agent launcher"
- **start-assets** - "Official asset modules for the start AI agent launcher"

### CLI Commands (From Prototype)

- `start assets search <query>` - Search catalog
- `start assets browse` - Open GitHub catalog in browser
- `start assets add <query>` - Search and install
- `start assets update [query]` - Update cached assets
- `start assets info <name>` - Preview asset

### Tag Namespaces

- **CLI releases** - `v0.1.0` (no prefix, triggers Homebrew/GitHub releases)
- **Asset modules** - `tasks/golang/code-review/v0.1.0` (directory path prefix)
- **Schema module** - `schemas/v0.1.0`

### Template Strategy

- **Pattern constraints** - Use CUE native `[Name=_]: {...}` for schema defaults and field injection
- **Field name auto-injection** - Pattern alias `[Name=_]: {name: Name}` automatically injects field labels
- **Placeholder substitution** - Use `text/template.Execute` (Go template syntax) not custom placeholders
- **UTD pattern implementation** - Template strings with runtime data injection via text/template
- **Default values** - Use CUE's default operator `|*value` in pattern constraints

### Schema Philosophy: Pure Constraints

- **No defaults in schemas** - Schemas define "what is valid", not "what is typical"
- **User-controlled defaults** - Users set global defaults via pattern constraints in their config
- **Prevents conflicts** - Multiple defaults would conflict and cancel each other out in CUE
- **Flexibility** - Users can change one line to affect all tasks (e.g., timeout for slow local models)
- **No name field** - Map keys identify tasks, eliminating redundant name field
- **Searchable by key** - Task resolution and search use map keys, descriptions, tags, and module paths

### What CUE Eliminates (vs Prototype)

- ❌ Custom placeholder parser (`{file}`, `{command_output}`) - Replaced by text/template
- ❌ Custom validation logic - Replaced by CUE native constraints
- ❌ Custom default value management - Replaced by pattern constraints + `|*`
- ❌ Custom CSV index parsing - Replaced by native `index.cue`
- ❌ Manual field name duplication - Replaced by pattern alias auto-injection
- ❌ Runtime type errors - Replaced by compile-time CUE validation
- ❌ Custom GitHub asset metadata - Replaced by CUE module metadata

### Minimal Custom Layer (Go)

- **Runtime data injection** - Execute commands, read files, inject into CUE templates
- **Module resolution** - Map friendly names to module paths, use CUE's `mod get`
- **Orchestration** - Load CUE → Execute → Inject → Render → Launch agent
- **UTD validation** - Validate "at least one of file/command/prompt" at runtime

### UTD Schema Design

- **Reusable definition** - #UTD used by tasks, roles, and contexts
- **Pure schema (no defaults)** - Schema defines constraints only, users set defaults in their config
- **Content fields** - file, command, prompt (at least one required, validated by Go)
- **Shell configuration** - Optional shell and timeout overrides
- **Timeout field naming** - `timeout` not `command_timeout` (simpler, clearer)
- **Timeout constraint** - 1-3600 seconds (no default in schema, user provides via pattern)
- **Template syntax** - Go templates with `{{.placeholder}}` not `{placeholder}`
- **Placeholders** - `{{.file}}`, `{{.file_contents}}`, `{{.command}}`, `{{.command_output}}`, `{{.date}}`
- **Resolution priority** - prompt > file > command

### Task Schema Design

- **Embeds #UTD** - Inherits all UTD fields (file, command, prompt, shell, timeout)
- **No name field** - Map key IS the task name (e.g., `tasks["code-review"]`), no duplication needed
- **Pure schema (no defaults)** - Schema defines constraints only, users set defaults in their config
- **User-controlled defaults** - `tasks: [_]: #Task & {timeout: *120 | _}` applies global defaults in user config
- **No alias field** - Users can use shell aliases/history instead
- **Task-specific placeholder** - `{{.instructions}}` for CLI arguments
- **References not inline** - role/agent are string references, not embedded configs

## Goals

1. Understand CUE's type system, unification, and validation capabilities
2. Understand CUE's module and package hierarchy (schema, values, templates)
3. Design CUE schemas for core concepts (roles, tasks, contexts, agents)
4. Understand how CUE integrates with Go for runtime loading and validation
5. Document architectural decisions in design records
6. Validate that CUE can replace the custom GitHub asset system from prototype

## Scope

In Scope:

- Research CUE language features relevant to configuration and validation
- Study CUE module system and package structure
- Design initial schemas for roles, tasks, contexts, agents
- Understand CUE Central Registry and package format
- Explore CUE's Go API for loading and validating configurations
- Test concepts using tutorial at reference/cuelang-org/content/docs/tutorial/working-with-a-custom-module-registry/en.md
- Create DR-001: CUE-First Architecture Decision
- Create DR-002: Configuration Schema Design
- Document module hierarchy strategy

Out of Scope:

- Implementation of Go code (P-005)
- CLI command implementation (P-004)
- Publishing packages to registry (P-003)
- Creating production-ready assets (P-002)
- Performance optimization
- Error handling implementation

## Success Criteria

- [x] Can explain CUE's type system and how unification works
- [x] Can explain how CUE modules, packages, and hierarchy work (schema vs values vs templates)
- [x] Understand CUE's order preservation (lists maintain order, evaluation is order-independent)
- [x] Determined asset distribution strategy (CUE modules via OCI registry)
- [x] Determined versioning strategy (semver with git tags)
- [x] Determined repository structure (separate start-assets repo)
- [x] Have designed working CUE schema for tasks (validates with cue vet)
- [x] Have designed working CUE schema for index (validates with cue vet)
- [x] Task schema validates correctly using CUE CLI tools
- [x] Index schema validates correctly using CUE CLI tools
- [x] Understand pure constraints design (no defaults in schemas)
- [x] Understand user-controlled defaults pattern
- [x] Researched OCI registry capabilities and limitations
- [x] Have designed working CUE schemas for roles, contexts, agents
- [x] All schemas validate correctly using CUE CLI tools
- [x] Understand how to load and validate CUE from Go
- [x] Documented user-controlled defaults design (DR-001)
- [x] Documented no-name-field decision (DR-002)
- [x] Documented index category structure (DR-003)
- [x] Documented module naming convention (DR-004)
- [x] Documented UTD pattern with Go templates (DR-005)
- [x] Documented shell configuration approach (DR-006)
- [x] Documented UTD error handling strategy (DR-007)
- [x] Can articulate how CUE replaces the prototype's custom asset system
- [x] Have working examples that demonstrate key concepts

## Deliverables

Design Records:

- ✓ DR-001: User-Controlled Defaults in CUE Schemas
- ✓ DR-002: No Name Field in Task Schema
- ✓ DR-003: Index Category Structure
- ✓ DR-004: Module Naming Convention
- ✓ DR-005: Go Templates for UTD Pattern
- ✓ DR-006: Shell Configuration and Command Execution
- ✓ DR-007: UTD Error Handling by Context
- ✓ DR-008: Context Schema Design
- ✓ DR-009: Task Schema Design
- ✓ DR-010: Role Schema Design
- ✓ DR-011: Agent Schema Design

Documentation:

- ✓ docs/design/utd-pattern.md - UTD pattern documentation with Go templates
- docs/cue/schema-design.md - Schema design documentation (TODO)
- docs/cue/module-hierarchy.md - Module organization strategy (TODO)
- docs/cue/integration-notes.md - CUE-Go integration notes (TODO)

Schemas (reference/start-assets/schemas/):

- ✓ schemas/utd.cue - UTD schema definition (#UTD) - pure constraints, no defaults
- ✓ schemas/utd_example.cue - UTD examples with 11 usage patterns
- ✓ schemas/task.cue - Task schema definition (#Task) - embeds #UTD
- ✓ schemas/task_example.cue - Working examples demonstrating user config patterns
- ✓ schemas/index.cue - Index schema definition (#Index) for asset discovery
- ✓ schemas/index_example.cue - Index examples with resolution flows
- ✓ schemas/README.md - Schema documentation with design philosophy
- ✓ schemas/role.cue - Role schema definition (#Role) - embeds #UTD
- ✓ schemas/role_example.cue - Role examples with 7 usage patterns
- ✓ schemas/context.cue - Context schema definition (#Context) - embeds #UTD, adds required/default/tags
- ✓ schemas/context_example.cue - Context examples with 10 usage patterns
- ✓ schemas/agent.cue - Agent schema definition (#Agent) - command templates, no UTD
- ✓ schemas/agent_example.cue - Agent examples with 8 usage patterns

## Dependencies

None - this is the foundational project that all others depend on.

## Technical Approach

Research Phase:

1. Study CUE type system and unification
   - Read reference/cuelang-org/content/docs/concept/
   - Read reference/cuelang-org/content/docs/language-guide/
   - Understand how constraints and validation work
   - Understand how defaults and optional fields work

2. Study CUE module system
   - Read reference/cuelang-org/content/docs/tutorial/working-with-a-custom-module-registry/
   - Understand module hierarchy (schema, values, templates)
   - Understand package structure and imports
   - Understand how CUE Central Registry works

3. Study CUE-Go integration
   - Read reference/cue/doc/ for Go API documentation
   - Understand how to load CUE from Go
   - Understand how to validate and extract values
   - Understand error handling patterns

Design Phase:

1. Map prototype concepts to CUE
   - Review prototype design records (reference/start-prototype/docs/design/design-records/)
   - Identify what CUE handles natively vs what needs custom code
   - Design schema structure for roles, tasks, contexts, agents

2. Create schema definitions
   - Define role schema with constraints
   - Define task schema with validation
   - Define context schema with required fields
   - Define agent schema with command templates
   - Test schemas validate correctly

3. Design module hierarchy
   - Determine how schemas, values, and templates organize
   - Design package structure for distribution
   - Plan how users will import and extend schemas

Documentation Phase:

1. Write DR-001: CUE-First Architecture Decision
   - Why CUE vs TOML
   - Trade-offs and benefits
   - How CUE eliminates custom systems from prototype

2. Write DR-002: Configuration Schema Design
   - Schema patterns and structure
   - Validation rules and constraints
   - How schemas compose and extend

3. Document learnings
   - Module hierarchy strategy
   - Integration patterns
   - Best practices discovered

## Questions & Uncertainties

### Answered ✓

CUE Language:

- ✓ How does unification work in practice? - Types ARE values, lattice-based constraints
- ✓ What's the difference between concrete values and constraints? - All exist in value lattice
- ✓ How do we preserve configuration order? - Use ordered lists in CUE
- ✓ How do versions and dependencies work? - Semantic versioning + MVS algorithm
- ✓ What's the package format for CUE Central Registry? - OCI registry with standard manifest

Module System:

- ✓ What's the best way to structure schema vs values? - Three-layer hierarchy (schemas, assets, user config)
- ✓ How do packages compose? - Assets import schemas, validate during development
- ✓ Can users override our base schemas? - Yes, through CUE unification
- ✓ Package format? - OCI manifest + zip blob + module.cue blob

Architecture:

- ✓ How much simpler with CUE? - Eliminates custom GitHub asset system, index.csv becomes index.cue
- ✓ What concepts from prototype still apply? - Core concepts (roles, tasks, contexts, agents), CLI commands
- ✓ Asset distribution? - CUE modules replace GitHub catalog, lazy loading via registry

### Outstanding Questions

CUE Language:

- ✓ Can CUE handle placeholder substitution? - Yes, text/template.Execute
- ✓ Can CUE validate command template syntax? - Yes, via Go template validation in text/template
- ✓ How do we handle dynamic values from runtime? - Inject via Go, pass to text/template.Execute
- ✓ Should placeholder substitution happen in CUE or Go? - CUE (text/template.Execute)

Schema Design:

- ✓ How to model UTD pattern? - Template strings with text/template syntax
- ✓ How to validate "at least one field must be present" in UTD? - Go validates at runtime (CUE documents requirement)
- ✓ Should command templates be validated by CUE or Go? - CUE (text/template validates syntax)
- ✓ How to structure optional file/command/prompt fields? - All optional, Go validates at least one present
- ✓ Should tasks have alias field? - No, use shell aliases instead

Integration:

- What's the Go API surface for loading CUE?
- How do we handle validation errors from Go?
- Can we get structured error messages for user feedback?
- What's the performance of CUE validation?
- How do we handle CUE files that don't exist vs invalid CUE?

User Config:

- How do users define custom tasks/roles/contexts?
- How to override published assets?
- Config file organization in ~/.config/start/?

## Research Areas

High Priority:

1. CUE type system fundamentals
   - Constraints vs values
   - Unification algorithm
   - Validation and error messages
   - Optional and required fields

2. Module and package system
   - Package structure
   - Import and composition
   - Schema extension patterns
   - CUE Central Registry format

3. Go integration
   - Loading CUE configurations
   - Validation from Go
   - Error handling
   - Value extraction

Medium Priority:

1. Order preservation
   - How CUE maintains order
   - How to access ordered fields from Go
   - Implications for context injection

2. Command templating
   - Can CUE validate placeholder syntax?
   - Should substitution be in CUE or Go?
   - How to define template constraints

3. Configuration composition
   - Global vs local config merging
   - Override patterns
   - Default value strategies

Low Priority:

1. Advanced features
   - Scripting capabilities
   - Built-in functions
   - Custom validators
   - Code generation

## Notes

Reference Materials Available:

- reference/cue/ - CUE language source code and implementation
- reference/cuelang-org/ - Official documentation and tutorials
- reference/start-prototype/docs/design/design-records/ - 44 DRs documenting what we learned from TOML approach

Key Tutorial:

- reference/cuelang-org/content/docs/tutorial/working-with-a-custom-module-registry/en.md
  Use this to test concepts hands-on

Prototype Learnings to Carry Forward:

- Core concepts (roles, tasks, contexts, agents) are solid
- CLI interface from docs/cli/ is well-designed
- Process replacement execution model (DR-043) still applies
- Shell quote escaping (DR-044) still needed
- Many asset management DRs (DR-031-042) likely obsolete with CUE registry

This project sets the foundation for everything else. Take time to understand CUE deeply before moving to P-002.

## Progress & Next Steps

### Completed

1. ✓ Studied CUE documentation (type system, modules, registry)
2. ✓ Determined CUE is the right choice (replaces TOML completely)
3. ✓ Decided on asset distribution architecture (separate start-assets repo)
4. ✓ Determined versioning strategy (semver + git tags)
5. ✓ Designed repository structure (tasks/category/item/ pattern)
6. ✓ Established module path convention (github.com/grantcarthew/start-task-category-item@v0)
7. ✓ Created start-assets repository at ./reference/start-assets
8. ✓ Confirmed order preservation strategy (ordered lists in CUE)
9. ✓ Researched CUE template capabilities (pattern constraints + text/template)
10. ✓ Determined template strategy (text/template for runtime placeholders)
11. ✓ Identified what CUE eliminates from prototype (custom parsers, validation, defaults management)
12. ✓ Reviewed prototype Task design (DR-009, DR-019, DR-029)
13. ✓ Decided to remove alias field from tasks (shell already does this better)
14. ✓ Decided to remove name field from tasks (map key IS the name)
15. ✓ Created schemas module (github.com/grantcarthew/start-schemas@v0)
16. ✓ Implemented #Task schema with pure constraints (no defaults)
17. ✓ Implemented #Index schema for asset discovery
18. ✓ Created task examples demonstrating user config patterns with defaults
19. ✓ Created index examples with resolution flows
20. ✓ Validated schemas with cue vet (all pass)
21. ✓ Documented pure constraints design philosophy
22. ✓ Researched OCI registry API capabilities and limitations
23. ✓ Confirmed need for index.cue (OCI catalog API disabled)
24. ✓ Designed index key structure (category/item format)
25. ✓ Documented schema in schemas/README.md
26. ✓ Created DR-001: User-Controlled Defaults
27. ✓ Created DR-002: No Name Field
28. ✓ Created DR-003: Index Category Structure
29. ✓ Created DR-004: Module Naming Convention
30. ✓ Created UTD pattern documentation (docs/design/utd-pattern.md)
31. ✓ Created DR-005: Go Templates for UTD Pattern
32. ✓ Created DR-006: Shell Configuration and Command Execution
33. ✓ Created DR-007: UTD Error Handling by Context
34. ✓ UTD design complete (documentation + 3 design records)
35. ✓ Created #UTD schema (utd.cue) - pure constraints, no defaults
36. ✓ Refactored #Task schema to embed #UTD
37. ✓ Created UTD examples (utd_example.cue) with 11 usage patterns
38. ✓ Standardized timeout field naming (timeout not command_timeout)
39. ✓ Updated all documentation for timeout field consistency
40. ✓ Validated #UTD schema with cue vet (passes)
41. ✓ Updated schemas/README.md with #UTD documentation
42. ✓ Created #Role schema (role.cue) - embeds #UTD
43. ✓ Created role examples (role_example.cue) with 7 usage patterns
44. ✓ Created DR-008: Context Selection and Tagging
45. ✓ Created #Context schema (context.cue) - embeds #UTD, adds required/default/tags
46. ✓ Created context examples (context_example.cue) with 10 usage patterns
47. ✓ Created #Agent schema (agent.cue) - minimal, command templates only
48. ✓ Created agent examples (agent_example.cue) with 8 usage patterns
49. ✓ All schemas validated with cue vet
50. ✓ Created DR-009: Task Schema Design
51. ✓ Created DR-010: Role Schema Design
52. ✓ Created DR-011: Agent Schema Design
53. ✓ Verified CUE-Go integration approach (straightforward with standard API)
54. ✓ Confirmed runtime context injection pattern (ctx.CompileString + Fill)

### Completed

1. ✓ Design CUE schemas for all core concepts
   - ✓ #UTD schema
   - ✓ #Task schema
   - ✓ #Index schema
   - ✓ #Role schema
   - ✓ #Context schema
   - ✓ #Agent schema
2. ✓ Model UTD pattern (file/command/prompt) in CUE
3. ✓ Determine placeholder substitution strategy (text/template)
4. ✓ Understand runtime context injection (dynamic values)
5. ✓ Study CUE-Go API for loading and validation
6. ✓ Create working schema examples and validate with CUE CLI

### Deferred to Later Projects

- Implementation of Go code (P-005)
- CLI command implementation (P-004)
- Publishing packages to registry (P-003)
- Creating production-ready assets (P-002)
