# P-002: Concrete Assets - Validate Design

- Status: In Progress
- Started: 2025-12-05
- Completed: -

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
- [ ] Have 3-5 working task assets that validate correctly
- [ ] Have 2-3 working context assets that validate correctly
- [ ] Have 2-3 working agent assets that validate correctly
- [x] Assets demonstrate key patterns (composition, extension, constraints)
- [ ] CUE validation catches intentional errors (negative testing)
- [x] Assets use realistic content (not placeholder text)
- [x] Identified and documented at least 2-3 schema improvements (module path alignment, import syntax)
- [x] Updated schemas if design issues discovered (module.cue updated with correct path and source)
- [x] Documented asset creation best practices (UTD file pattern, role styles)

## Deliverables

Assets (examples directory):
- examples/roles/go-expert.cue
- examples/roles/code-reviewer.cue
- examples/roles/documentation-writer.cue
- examples/tasks/pre-commit-review.cue
- examples/tasks/debug-help.cue
- examples/tasks/refactor-code.cue
- examples/contexts/environment.cue
- examples/contexts/project.cue
- examples/agents/claude.cue
- examples/agents/gpt.cue

Documentation:
- docs/cue/asset-patterns.md - Patterns and best practices for creating assets
- docs/cue/validation-examples.md - Examples of validation in action

Schema Updates (if needed):
- Refined schemas in examples/schemas/ based on learnings

Design Record Updates (if schema changed significantly):
- Update to DR-002 documenting changes and reasoning

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

5. Test positive cases
   - All assets validate with cue vet
   - Assets compose correctly
   - Constraints work as expected

6. Test negative cases
   - Intentionally break required fields
   - Provide invalid values
   - Test constraint violations
   - Verify error messages are helpful

Refinement Phase:

7. Identify schema issues
   - What's too restrictive?
   - What's too permissive?
   - What's missing?
   - What's awkward to use?

8. Refine schemas
   - Update schema definitions
   - Re-validate all assets
   - Document changes and reasoning

9. Document patterns
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

4. Content organization
   - File vs inline content
   - External file references
   - Content reuse patterns

5. Template syntax
   - Placeholder patterns
   - Command template structure
   - Instruction templates

Low Priority:

6. Asset metadata
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
4. ✓ Each role uses UTD pattern with markdown files (role.md)
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
- ✓ Use UTD `file` field with markdown files (role.md)
- ✓ Keep role content separate from CUE configuration
- ✓ Customize markdown content for each style

**Versioning:**
- ✓ Start with v0.0.1 for initial experimental releases
- ✓ Use annotated tags for all releases

### Next Steps

1. Commit the three golang roles to git
2. Create tasks (golang/code-review, golang/debug, etc.)
3. Create contexts (environment, project)
4. Create agents (claude, gpt)
5. Test composition and extension patterns
6. Document any schema issues discovered
