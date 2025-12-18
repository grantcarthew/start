# start - AI Agent CLI Orchestrator

`start` is a command-line orchestrator for AI agents built on CUE. It manages prompt composition, context injection, and workflow automation by wrapping AI CLI tools (Claude, Gemini, GPT, etc.) with configurable roles, reusable tasks, and project-aware context documents.

## Project Status

**Fresh start, design phase.** This project builds on extensive research from [start-prototype](https://github.com/grantcarthew/start-prototype), which validated the core concepts through a TOML-based implementation. The prototype revealed that TOML's lack of table ordering and limited validation made it unsuitable for this use case. This version is rebuilt from the ground up using CUE.

Avoid references to the prototype in new documentation. This is a new design.

### Active Project

Projects are stored in the docs/projects/ directory. Continue by reading the active project.

Active Project: docs/projects/p-008-configuration-editing.md

## Why CUE?

CUE (Configure Unify Execute) provides exactly what this project needs:

- **Order preservation**: Configuration order matters for context injection and prompt composition
- **Built-in validation**: Schema definition and validation are native features
- **Type safety**: Strong typing prevents configuration errors
- **Packages and modules**: CUE Central Registry provides proper package distribution
- **Templating**: Native support for constraints, defaults, and composition
- **Data and logic together**: Configuration can include validation rules and transformations

This eliminates the need for custom asset management, GitHub-based distribution, and hand-rolled validation that the prototype required.

## Core Concepts

- **Roles**: Define AI agent behavior and expertise (e.g., `go-expert`, `code-reviewer`)
- **Tasks**: Reusable prompts for common workflows (e.g., `pre-commit-review`, `debug-help`)
- **Contexts**: Environment-specific information loaded at runtime and injected into prompts
- **Agents**: AI model configurations (Claude, GPT, Gemini, etc.) with command templates
- **Packages**: Roles, tasks, and configurations distributed via CUE Central Registry

## Expected Usage

```bash
start                           # Start interactive session with default role
start --role go-expert          # Start with specific role
start task pre-commit-review    # Run a specific task
start init                      # Initialize CUE configuration
```

## Architecture Principles

- **CUE-native**: All configuration, schemas, and validation in CUE
- **Registry-driven**: Packages distributed via CUE Central Registry, not custom GitHub system
- **Order-aware**: Configuration order preserved for context injection
- **Type-safe**: CUE schemas prevent configuration errors
- **Simple**: Let CUE handle complexity instead of building custom systems

## What Changed From Prototype

The prototype explored the design space and validated the core concepts. Key architectural changes:

| Aspect             | Prototype (TOML)         | This Version (CUE)   |
| ------------------ | ------------------------ | -------------------- |
| Config format      | TOML (unordered tables)  | CUE (ordered, typed) |
| Asset distribution | Custom GitHub API system | CUE Central Registry |
| Validation         | Custom Go code           | CUE schemas          |
| Package management | Custom catalog/cache     | CUE modules          |
| Schema definition  | Documentation only       | Enforced by CUE      |
| Order preservation | Failed assumption        | Native support       |

## Documentation

Documentation will be structured as:

- `docs/design/` - Design decisions and architecture (Design Records)
- `docs/cue/` - CUE schema reference and examples
- `docs/cli/` - Command reference (adapted from prototype)

The prototype's CLI documentation (`../start-prototype/docs/cli/*.md`) provides a starting point for the user interface, though commands will evolve as CUE capabilities are integrated.

## Document Driven Development (DDD)

This project uses Document Driven Development. Design decisions are documented in Design Records (DRs) before implementation.

**Location:** `docs/design/design-records/`

**Process:**

- Create DRs for architectural decisions, algorithms, breaking changes, API/CLI structure
- Get next DR number from `docs/design/design-records/README.md`
- Follow reconciliation process after 5-10 DRs

Design Records will start fresh at DR-001, as the CUE architecture is fundamentally different from the prototype.

## Testing

When implementing code, ensure it is testable. Read `docs/design/design-records/dr-024-testing-strategy.md` for the testing approach.

Key principles:

- Test real behaviour over mocks (use actual CUE validation, real files via `t.TempDir()`)
- Design functions to accept interfaces/parameters rather than reaching for globals
- Use table-driven tests for multiple cases
- Run tests via `scripts/invoke-tests`

## References

- **Prototype research**: [start-prototype](https://github.com/grantcarthew/start-prototype) - TOML-based implementation with 44 design records documenting the exploration
- **CUE language**: [cuelang.org](https://cuelang.org)
- **CUE Central Registry**: [registry.cuelang.org](https://registry.cuelang.org)

## Local Reference Repositories

The `./Context` directory contains cloned source code and documentation for development reference. Each directory has an INDEX.csv file cataloging its contents:

- **Context/cue** - CUE language source code, standard library, and implementation
  - Use for: Understanding CUE internals, package structure, encoding implementations, module system
  - See: `Context/cue/INDEX.csv` for detailed catalog

- **Context/cuelang-org** - CUE official documentation site
  - Use for: Tutorials, concepts, how-to guides, language reference, examples
  - See: `Context/cuelang-org/INDEX.csv` for detailed catalog
  - Key content: `content/docs/tutorial/` (especially working-with-a-custom-module-registry)

- **Context/start-prototype** - TOML-based prototype with complete design records
  - Use for: Design decisions, CLI interface research, architectural patterns explored
  - See: `Context/start-prototype/INDEX.csv` for detailed catalog
  - Key content: `docs/design/design-records/` (44 DRs), `docs/cli/` (command reference)

- **Context/cobra** - Cobra CLI framework source code
  - Use for: CLI implementation patterns, command structure, flag handling, shell completions
  - See: `Context/cobra/INDEX.csv` for detailed catalog

## Development Status

**Completed:**

- P-001: CUE Foundation & Architecture (schemas designed and published)
- P-002: Concrete Assets Validation (17 modules published to CUE Central Registry)
- P-003: Registry Distribution (20 modules published, prototype comparison documented)
- P-004: Minimal CLI Implementation (CUE loading, show command, global flags)
- P-005: Orchestration Core Engine (template processing, shell execution, composition, CLI commands)
- P-006: Auto-Setup (registry interaction, agent detection, TTY prompts, config writing)
- P-008: Configuration Editing (config agent/role/context/task commands with list/add/show/edit/remove)

**Current:** P-007 Package Management

The prototype validated that this tool solves a real problem. This version will implement it properly.
