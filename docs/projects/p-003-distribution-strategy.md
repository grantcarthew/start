# P-003: Distribution Strategy

- Status: Proposed
- Started: -
- Completed: -

## Overview

Define how assets are distributed and consumed using CUE Central Registry. This project replaces the entire custom GitHub asset management system from the prototype (DR-031 through DR-042) with CUE's native package distribution mechanism.

This is a critical architectural decision - if CUE Central Registry works as expected, it eliminates thousands of lines of custom code and complex asset management logic from the prototype.

## Goals

1. Understand CUE Central Registry package format and requirements
2. Understand how to publish packages to the registry
3. Design package structure for roles, tasks, contexts, and agents
4. Understand versioning and dependency management in CUE
5. Test publishing a package to the registry
6. Define how users discover and import packages
7. Document the distribution architecture in design records

## Scope

In Scope:

- Research CUE Central Registry package format
- Understand publishing process and requirements
- Design package structure for asset distribution
- Test publishing one or more example packages
- Understand dependency management and versioning
- Define how users import and use published packages
- Create DR-004: Asset Distribution via CUE Registry
- Document package creation and publishing process

Out of Scope:

- Building complete asset catalog (comes after validation)
- CLI commands for package management (P-004)
- Go integration for package loading (P-005)
- Automated publishing or CI/CD
- Package hosting infrastructure
- Production package releases

## Success Criteria

- [ ] Understand CUE Central Registry package format requirements
- [ ] Successfully published at least one test package to registry
- [ ] Can import and use published package in CUE files
- [ ] Understand versioning strategy for packages
- [ ] Documented package structure and organization
- [ ] Created DR-004: Asset Distribution Strategy
- [ ] Can articulate how this replaces prototype's DR-031-042 complexity
- [ ] Documented publishing process and requirements

## Deliverables

Design Records:
- DR-004: Asset Distribution via CUE Central Registry

Documentation:
- docs/cue/package-structure.md - How packages are organized
- docs/cue/publishing-guide.md - How to publish packages to registry
- docs/cue/versioning-strategy.md - Versioning and dependency management

Test Package:
- Test package published to CUE Central Registry
- Example of importing and using the package
- Validation that it works end-to-end

Package Design:
- Package structure template
- Naming conventions
- Organization strategy

## Dependencies

Requires:
- P-001 (need schema design)
- P-002 (need concrete assets to package)

Blocks:
- P-004 (CLI needs to know how packages work)
- Future asset creation and distribution

## Technical Approach

Research Phase:

1. Study CUE Central Registry
   - Read reference/cuelang-org/content/docs/tutorial/working-with-a-custom-module-registry/
   - Understand registry requirements and constraints
   - Understand authentication and publishing process
   - Study existing packages for patterns

2. Study CUE modules and packages
   - Understand module.cue format
   - Understand package imports and exports
   - Study versioning (semantic versioning requirements)
   - Understand dependency resolution

3. Review prototype asset system
   - Read DR-031 through DR-042 in reference/start-prototype
   - Identify what CUE registry replaces
   - Identify what still needs custom handling
   - Calculate complexity reduction

Design Phase:

4. Design package structure
   - How to organize roles, tasks, contexts, agents
   - One package vs multiple packages?
   - Package naming conventions
   - Directory structure within packages

5. Design versioning strategy
   - Semantic versioning approach
   - Breaking change policy
   - Backwards compatibility strategy
   - How users specify versions

6. Design package discovery
   - How users find available packages
   - Package documentation and metadata
   - Search and browsing strategy

Testing Phase:

7. Create test package
   - Package one or more assets from P-002
   - Create module.cue with metadata
   - Follow CUE registry requirements
   - Test locally first

8. Publish to registry
   - Set up registry authentication
   - Publish test package
   - Verify package appears in registry
   - Test importing from another project

9. Validate end-to-end
   - Import package in new CUE file
   - Use assets from package
   - Validate composition works
   - Test version constraints

Documentation Phase:

10. Write DR-004
    - Why CUE Central Registry vs custom GitHub system
    - Trade-offs and benefits
    - How it simplifies architecture
    - What prototype DRs become obsolete

11. Document publishing process
    - Step-by-step guide
    - Requirements and prerequisites
    - Common issues and solutions

12. Document package structure
    - Organization patterns
    - Naming conventions
    - Best practices

## Questions & Uncertainties

Registry Mechanics:
- What are the exact requirements for publishing?
- How does authentication work?
- Are there package size limits?
- How long does publishing take?
- Can packages be unpublished or only deprecated?

Package Organization:
- Should each asset type be a separate package?
- Or one monolithic package with everything?
- How granular should packages be?
- What's the naming convention for start-related packages?

Versioning:
- How do we handle breaking changes to schemas?
- Can users pin to specific versions?
- How does dependency resolution work?
- What happens with version conflicts?

Discovery:
- How do users discover available packages?
- Is there a registry web UI?
- Can we provide our own catalog/directory?
- How do we document what's available?

User Experience:
- How easy is it for users to import packages?
- Do they need CUE CLI installed?
- Can start CLI abstract package management?
- What's the learning curve?

Migration:
- Can we support both CUE packages and local assets?
- How do users migrate from prototype (if they existed)?
- What's the transition story?

## Research Areas

High Priority:

1. CUE Central Registry mechanics
   - Publishing process
   - Authentication
   - Requirements and constraints
   - Package format

2. Package structure patterns
   - Organization strategies
   - Naming conventions
   - Best practices from existing packages

3. Versioning and dependencies
   - Semantic versioning in CUE
   - Dependency resolution
   - Version constraints

Medium Priority:

4. Package discovery
   - Registry UI and search
   - Documentation and metadata
   - How users find packages

5. User workflow
   - How users import and use packages
   - CUE CLI requirements
   - Integration with start CLI

6. Prototype comparison
   - Complexity of DR-031-042 asset system
   - What CUE registry replaces
   - Architecture simplification

Low Priority:

7. Advanced scenarios
   - Private packages
   - Forking and customization
   - Multiple registries

## Notes

Prototype Asset System to Replace:
- DR-031: Catalog-Based Asset Architecture
- DR-032: Asset Metadata Schema
- DR-033: Asset Resolution Algorithm
- DR-034: GitHub Catalog API Strategy
- DR-035: Interactive Asset Browsing
- DR-036: Cache Management
- DR-037: Asset Update Mechanism
- DR-039: Catalog Index File
- DR-040: Substring Matching Algorithm
- DR-042: Missing Asset Restoration

If CUE Central Registry works as expected, all of this complexity becomes unnecessary. That's the hypothesis to validate in this project.

Key Tutorial:
- reference/cuelang-org/content/docs/tutorial/working-with-a-custom-module-registry/en.md

Success Measure:
This project is successful if publishing and consuming CUE packages is simpler and more reliable than the custom GitHub asset system from the prototype. The goal is architectural simplification through better primitives.

Open Question:
Do we need our own catalog/directory of available packages, or is the CUE registry sufficient? This may depend on registry UI/UX.
