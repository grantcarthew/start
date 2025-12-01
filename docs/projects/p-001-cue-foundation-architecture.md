# P-001: CUE Foundation & Architecture

- Status: Proposed
- Started: -
- Completed: -

## Overview

Research CUE language capabilities and design the foundational architecture for the CUE-based start tool. This project establishes how we'll use CUE for configuration, validation, asset management, and schema definition.

This is the critical first project that determines whether CUE can fulfill the requirements that TOML could not: ordered configuration, built-in validation, type safety, and native package distribution.

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

- [ ] Can explain CUE's type system and how unification works
- [ ] Can explain how CUE modules, packages, and hierarchy work (schema vs values vs templates)
- [ ] Have designed working CUE schemas for roles, tasks, contexts, and agents
- [ ] Schemas validate correctly using CUE CLI tools
- [ ] Understand how to load and validate CUE from Go
- [ ] Documented why CUE is the right choice (DR-001)
- [ ] Documented schema design patterns and decisions (DR-002)
- [ ] Can articulate how CUE replaces the prototype's custom asset system
- [ ] Have working examples that demonstrate key concepts

## Deliverables

Design Records:
- DR-001: CUE-First Architecture Decision
- DR-002: Configuration Schema Design
- DR-003: Module Hierarchy and Organization (possibly)

Documentation:
- docs/cue/schema-design.md - Schema design documentation
- docs/cue/module-hierarchy.md - Module organization strategy
- docs/cue/integration-notes.md - CUE-Go integration notes

Examples (proof of concept):
- examples/schemas/role.cue - Role schema definition
- examples/schemas/task.cue - Task schema definition
- examples/schemas/context.cue - Context schema definition
- examples/schemas/agent.cue - Agent schema definition
- examples/test-validation.cue - Validation tests

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

4. Map prototype concepts to CUE
   - Review prototype design records (reference/start-prototype/docs/design/design-records/)
   - Identify what CUE handles natively vs what needs custom code
   - Design schema structure for roles, tasks, contexts, agents

5. Create schema definitions
   - Define role schema with constraints
   - Define task schema with validation
   - Define context schema with required fields
   - Define agent schema with command templates
   - Test schemas validate correctly

6. Design module hierarchy
   - Determine how schemas, values, and templates organize
   - Design package structure for distribution
   - Plan how users will import and extend schemas

Documentation Phase:

7. Write DR-001: CUE-First Architecture Decision
   - Why CUE vs TOML
   - Trade-offs and benefits
   - How CUE eliminates custom systems from prototype

8. Write DR-002: Configuration Schema Design
   - Schema patterns and structure
   - Validation rules and constraints
   - How schemas compose and extend

9. Document learnings
   - Module hierarchy strategy
   - Integration patterns
   - Best practices discovered

## Questions & Uncertainties

CUE Language:
- How does unification work in practice for our use case?
- What's the difference between concrete values and constraints?
- How do we handle optional vs required fields?
- Can CUE validate command template syntax (e.g., placeholder patterns)?
- How do we handle dynamic values that come from runtime?

Module System:
- What's the best way to structure schema vs values vs templates?
- How do packages compose and extend each other?
- Can users override or extend our base schemas?
- What's the package format for CUE Central Registry?
- How do versions and dependencies work?

Integration:
- What's the Go API surface for loading CUE?
- How do we handle validation errors from Go?
- Can we get structured error messages for user feedback?
- What's the performance of CUE validation?
- How do we handle CUE files that don't exist vs invalid CUE?

Architecture:
- Should we use CUE for everything or mix with Go?
- How do we preserve configuration order (critical for context injection)?
- Can CUE handle the placeholder substitution logic?
- Should command templates be validated by CUE or Go?

Migration from Prototype:
- What concepts from prototype DRs still apply?
- What can we delete because CUE handles it?
- How much simpler is the architecture with CUE?

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

4. Order preservation
   - How CUE maintains order
   - How to access ordered fields from Go
   - Implications for context injection

5. Command templating
   - Can CUE validate placeholder syntax?
   - Should substitution be in CUE or Go?
   - How to define template constraints

6. Configuration composition
   - Global vs local config merging
   - Override patterns
   - Default value strategies

Low Priority:

7. Advanced features
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
