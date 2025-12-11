# P-002: Concrete Assets - Validate Design

- Status: Completed
- Started: 2025-12-05
- Completed: 2025-12-09

## Overview

Create concrete, real-world assets (roles, tasks, contexts, agents) in CUE to validate the schema designs from P-001. This project surfaces design issues early by building actual examples rather than theoretical schemas. The assets created here become the foundation for testing and the starting point for the asset catalog.

This follows the lesson from the prototype: having real assets early helps validate architecture decisions before they're locked in.

## Goals

1. Create working role definitions in CUE (e.g., go-expert, code-reviewer)
2. Create working task definitions in CUE (e.g., pre-commit-review, debug-help)
3. Create working context definitions in CUE (e.g., environment, project)
4. Create working agent definitions in CUE (e.g., Claude, GPT, Gemini)
5. Validate that assets compose and extend correctly
6. Identify and fix schema design issues discovered during creation
7. Test CUE validation catches configuration errors as expected

## Scope

In Scope:

- Create 3-5 example roles covering different use cases
- Create 3-5 example tasks covering different patterns
- Create 2-3 example contexts with real content
- Create 2-3 example agents with command templates
- Test asset validation using CUE CLI
- Test composition and inheritance patterns
- Refine schemas from P-001 based on learnings
- Document asset creation patterns
- Update DR-002 if schema design changes significantly

Out of Scope:

- Building complete asset catalog (that comes later)
- Publishing to CUE Central Registry (P-003)
- CLI commands to manage assets (P-004)
- Go integration for loading assets (P-005)
- Asset distribution or versioning
- Production-ready polish

## Success Criteria

- [x] Have 3-5 working role assets that validate correctly (3/3 golang roles: assistant, teacher, agent)
- [x] Have 3-5 working task assets that validate correctly (10/10 golang tasks created)
- [x] Have 2-3 working context assets that validate correctly (3/3: agents, environment, project)
- [x] Have 2-3 working agent assets that validate correctly (3/3: claude, gemini, aichat)
- [x] Assets demonstrate key patterns (composition, extension, constraints)
- [x] CUE validation catches intentional errors (negative testing)
- [x] Assets use realistic content (not placeholder text)
- [x] Identified and documented at least 2-3 schema improvements (module path alignment, import syntax, role CUE dependencies, @module/ prefix)
- [x] Updated schemas if design issues discovered (module.cue updated with correct path and source)
- [x] Documented asset creation best practices (UTD file pattern, role styles)

## Deliverables

Assets (start-assets repository):

Roles:

- roles/golang/assistant/ ✓ (published v0.0.1)
- roles/golang/teacher/ ✓ (published v0.0.1)
- roles/golang/agent/ ✓ (published v0.0.1)

Tasks:

- tasks/golang/code-review/ ✓ (published v0.0.1)
- tasks/golang/security-audit/ ✓ (published v0.0.1)
- tasks/golang/dependency-analysis/ ✓ (published v0.0.1)
- tasks/golang/api-docs/ ✓ (published v0.0.1)
- tasks/golang/refactor/ ✓ (published v0.0.1)
- tasks/golang/tests/ ✓ (published v0.0.1)
- tasks/golang/error-handling/ ✓ (published v0.0.1)
- tasks/golang/debug/ ✓ (published v0.0.1)
- tasks/golang/performance/ ✓ (published v0.0.1)
- tasks/golang/architecture/ ✓ (published v0.0.1)

Contexts:

- contexts/agents/ ✓ (published v0.0.1)
- contexts/environment/ ✓ (published v0.0.1)
- contexts/project/ ✓ (published v0.0.1)

Agents:

- agents/claude/ ✓
- agents/gemini/ ✓
- agents/aichat/ ✓

Documentation:

- docs/cue/asset-patterns.md - Patterns and best practices for creating assets
- docs/cue/validation-examples.md - Examples of validation in action

Schema Updates (if needed):

- Refined schemas in examples/schemas/ based on learnings

Design Record Updates:

- DR-009 updated with role union type ✓
- DR-022 created: Task Role CUE Dependencies ✓
- DR-023 created: Module Path Prefix ✓

## Dependencies

Requires:

- P-001 (need schemas before creating assets)

Blocks:

- P-003 (need assets to package for distribution)
- P-004 (need assets to test CLI commands)

## Technical Approach

Asset Creation Phase:

1. Start with roles
   - Create go-expert role with realistic expertise definition
   - Create code-reviewer role with review focus
   - Create documentation-writer role
   - Test role validation and composition
   - Document patterns discovered

2. Create tasks
   - Create pre-commit-review task with realistic instructions
   - Create debug-help task with troubleshooting focus
   - Create refactor-code task
   - Test task validation and agent field
   - Test command and instructions patterns

3. Create contexts
   - Create environment context with real system info
   - Create project context with repository details
   - Test required field validation
   - Test order preservation (critical for prompt composition)

4. Create agents
   - Create Claude agent with model configurations
   - Create GPT agent with API patterns
   - Test command template validation
   - Test placeholder patterns

Validation Phase:

1. Test positive cases
   - All assets validate with cue vet
   - Assets compose correctly
   - Constraints work as expected

2. Test negative cases
   - Intentionally break required fields
   - Provide invalid values
   - Test constraint violations
   - Verify error messages are helpful

Refinement Phase:

1. Identify schema issues
   - What's too restrictive?
   - What's too permissive?
   - What's missing?
   - What's awkward to use?

2. Refine schemas
   - Update schema definitions
   - Re-validate all assets
   - Document changes and reasoning

3. Document patterns
   - Common asset structures
   - Composition techniques
   - Validation best practices
   - Gotchas and solutions

## Questions & Uncertainties

Asset Design:

- What's the right level of detail for role definitions?
- Should tasks include example prompts or just templates?
- How detailed should context information be?
- Should agents include all possible models or just common ones?

Composition:

- How do roles compose with tasks?
- Can tasks override role defaults?
- How do contexts get injected into prompts?
- What's the precedence order for overrides?

Validation:

- What validations are helpful vs annoying?
- How strict should command template validation be?
- Should we validate placeholder syntax in CUE?
- How do we handle optional vs required in different contexts?

Order Preservation:

- Does CUE preserve field order as expected?
- How do we test order preservation?
- Does order matter for all fields or just some?

Real-World Use:

- Are these assets actually useful for real work?
- Do they demonstrate the tool's value proposition?
- Would users want to extend these or start fresh?

## Research Areas

High Priority:

1. Asset composition patterns
   - How roles and tasks combine
   - Override and extension mechanisms
   - Default value cascading

2. Validation in practice
   - What constraints are valuable
   - Error message quality
   - Validation performance

3. Order preservation testing
   - Field order in CUE files
   - Order when loaded from Go
   - Order in prompt composition

Medium Priority:

1. Content organization
   - File vs inline content
   - External file references
   - Content reuse patterns

2. Template syntax
   - Placeholder patterns
   - Command template structure
   - Instruction templates

Low Priority:

1. Asset metadata
   - Versioning information
   - Author and description
   - Tags and categories

## Notes

Reference Examples from Prototype:

- reference/start-prototype/assets/ - TOML-based assets for inspiration
- reference/start-prototype/examples/ - Example configurations

Key Difference from Prototype:
The prototype created theoretical schemas without real assets until late in development. This project inverts that - we create real assets early to validate the schema design works in practice.

Asset Selection Strategy:
Choose assets that:

- Cover different patterns (roles with different focuses, tasks with different structures)
- Demonstrate composition and extension
- Represent real-world use cases
- Exercise validation rules

These don't need to be production-ready, but should be realistic enough to validate the design.

This project is complete when we're confident the schema design works for real assets and we've identified any changes needed before moving to distribution (P-003).

## Progress & Decisions

### Completed (2025-12-05)

**Published Schemas to Central Registry:**

1. ✓ Fixed module path: `github.com/grantcarthew/start-assets/schemas@v0` (aligned with repo structure)
2. ✓ Added `source: {kind: "git"}` to module.cue
3. ✓ Published schemas@v0.0.1 to registry.cue.works
4. ✓ Validated full workflow: publish → import → validate

**Created Three Golang Roles:**

1. ✓ `roles/golang/assistant/` - Collaborative, friendly, interactive
2. ✓ `roles/golang/teacher/` - Patient, educational, explains concepts
3. ✓ `roles/golang/agent/` - Autonomous, minimal interruption, makes decisions
4. ✓ Each role uses UTD pattern with Markdown files (role.md)
5. ✓ All roles validated successfully with `cue vet`

### Key Discoveries

**Module Path & Registry:**

- Central Registry requires GitHub repository to exist at module path location
- Solution: Align module path with actual repo (`start-assets` not `start-schemas`)
- Module path format: `github.com/grantcarthew/start-assets/<type>/<path>@v0`

**Import Syntax:**

- Full form: `import "module/path@v0:packagename"`
- Short form (when basename matches package): `import "module/path@v0"`
- Our case: `github.com/grantcarthew/start-assets/schemas@v0` (basename `schemas` = package `schemas`)

**Publishing Requirements:**

- Annotated tags preferred: `git tag -a schemas/v0.0.1 -m "message"`
- VCS must be clean before publishing
- Tag must be pushed to GitHub before publishing
- `source: {kind: "git"}` required in module.cue
- `cue mod tidy` automatically fetches dependencies

### Key Decisions

**Role Structure:**

- ✓ Flat structure: `roles/golang/` not `roles/golang/expert/`
- ✓ Category = technology (golang, docker, python)
- ✓ Name = interaction style (assistant, teacher, agent)
- ✓ Full path: `roles/<category>/<style>/`

**Standard Role Styles (applies to all programming languages):**

- `assistant` - Collaborative partner, asks questions, explains decisions
- `teacher` - Patient instructor, educational, builds understanding
- `agent` - Autonomous executor, minimal questions, makes decisions

**Role Content Format:**

- ✓ Use UTD `file` field with Markdown files (role.md)
- ✓ Keep role content separate from CUE configuration
- ✓ Customize Markdown content for each style

**Versioning:**

- ✓ Start with v0.0.1 for initial experimental releases
- ✓ Use annotated tags for all releases

### Completed (2025-12-09)

**Created Contexts:**

1. ✓ `contexts/agents/` - Repository AGENTS.md with git remote URL
2. ✓ `contexts/environment/` - User environment from ~/Context/ENVIRONMENT.md
3. ✓ `contexts/project/` - Project documentation from ./project.md
4. ✓ All contexts import schemas@v0.0.2 and validate with `cue vet`

**Created Agent Definitions:**

1. ✓ `agents/claude/` - Claude Code by Anthropic (haiku, sonnet, opus)
2. ✓ `agents/gemini/` - Gemini CLI by Google (flash, flash-lite, pro)
3. ✓ `agents/aichat/` - AIChat multi-provider tool (vertexai:gemini variants)
4. ✓ All agents import schemas correctly and validate with `cue vet`
5. ✓ Each has proper module.cue with deps and source kind

**Published Roles to Central Registry:**

1. ✓ Published `roles/golang/assistant@v0.0.1`
2. ✓ Published `roles/golang/teacher@v0.0.1`
3. ✓ Published `roles/golang/agent@v0.0.1`
4. ✓ Verified with `cue mod resolve`

**Created 10 Golang Tasks:**

1. ✓ `tasks/golang/code-review/` - Comprehensive code review
2. ✓ `tasks/golang/security-audit/` - Security vulnerability analysis
3. ✓ `tasks/golang/dependency-analysis/` - Dependency health and updates
4. ✓ `tasks/golang/api-docs/` - API documentation generation
5. ✓ `tasks/golang/refactor/` - Code refactoring suggestions
6. ✓ `tasks/golang/tests/` - Test generation (unit, integration, benchmark)
7. ✓ `tasks/golang/error-handling/` - Error handling audit
8. ✓ `tasks/golang/debug/` - Systematic debugging assistance
9. ✓ `tasks/golang/performance/` - Performance profiling and optimization
10. ✓ `tasks/golang/architecture/` - Codebase structure analysis
11. ✓ All tasks import schemas@v0.0.2 and roles/golang/agent@v0.0.1
12. ✓ All tasks validated with `cue mod tidy` and `cue vet`

**Schema Updates:**

1. ✓ Updated `role` field: `string` → `string | #Role` (supports CUE dependencies)
2. ✓ Published schemas@v0.0.2 to Central Registry

**Design Records Created:**

1. ✓ DR-022: Task Role CUE Dependencies - Documents `role?: string | #Role` pattern
2. ✓ DR-023: Module Path Prefix - Documents `@module/` prefix for file resolution
3. ✓ Updated DR-009: Added role union type and published task example

**Published Contexts to Central Registry:**

1. ✓ Published `contexts/agents@v0.0.1`
2. ✓ Published `contexts/environment@v0.0.1`
3. ✓ Published `contexts/project@v0.0.1`

**Published Tasks to Central Registry:**

1. ✓ Published `tasks/golang/api-docs@v0.0.1`
2. ✓ Published `tasks/golang/architecture@v0.0.1`
3. ✓ Published `tasks/golang/code-review@v0.0.1`
4. ✓ Published `tasks/golang/debug@v0.0.1`
5. ✓ Published `tasks/golang/dependency-analysis@v0.0.1`
6. ✓ Published `tasks/golang/error-handling@v0.0.1`
7. ✓ Published `tasks/golang/performance@v0.0.1`
8. ✓ Published `tasks/golang/refactor@v0.0.1`
9. ✓ Published `tasks/golang/security-audit@v0.0.1`
10. ✓ Published `tasks/golang/tests@v0.0.1`

**Negative Testing (Schema Validation):**

- ✓ Tested constraint violations: timeout bounds, empty shell, invalid tags, type mismatches
- ✓ All invalid configurations rejected with clear error messages
- ✓ Error messages include constraint definition and violation location

### Key Discoveries (2025-12-09)

**Role as CUE Dependency:**

- String reference (`role: "golang/agent"`) does NOT create CUE dependency
- `cue mod tidy` removes unused deps when role is a string
- Solution: Import role package, use `role: agentRole.role`
- Schema change: `role?: string | #Role` preserves both patterns

**Module File Resolution:**

- `file: "./task.md"` won't work for cached modules (cwd is user's project)
- Solution: `@module/` prefix indicates module-relative path
- CLI resolves to CUE cache: `os.UserCacheDir()/cue/mod/extract/<module>@<version>/`
- Platform cache dirs: macOS `~/Library/Caches/cue`, Linux `~/.cache/cue`

**Task Prompt Pattern:**

- Standard format with custom instructions section
- `{{.file_contents}}` for task.md content
- `{{.instructions}}` for user-provided instructions (defaults to "None")
