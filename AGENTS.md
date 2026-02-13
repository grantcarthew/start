# start - AI Agent CLI Orchestrator

`start` is a command-line orchestrator for AI agents built on CUE. It manages prompt composition, context injection, and workflow automation by wrapping AI CLI tools (Claude, Gemini, GPT, etc.) with configurable roles, reusable tasks, and project-aware context documents.

## Project Status

**Fresh start, design phase.** This project builds on extensive research from [start-prototype](https://github.com/grantcarthew/start-prototype), which validated the core concepts through a TOML-based implementation. The prototype revealed that TOML's lack of table ordering and limited validation made it unsuitable for this use case. This version is rebuilt from the ground up using CUE.

Avoid references to the prototype in new documentation. This is a new design.

### Active Project

Projects are stored in `.ai/projects/`. Continue by reading the active project.

Active Project: None

Other Projects (paused):
- None

Project Workflow:
- Active projects are in `.ai/projects/`
- When a project is complete, move it to `.ai/projects/completed/`
- Update this file to point to the next active project
- Update the Development Status section below

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
start doctor                    # Diagnose installation and configuration
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

Documentation is structured as:

- `.ai/` - AI agent working files (projects, design records, tasks)
- `.ai/design/` - Design decisions and architecture (Design Records)
- `.ai/projects/` - Project documents
- `.ai/tasks/` - Task prompts
- `docs/` - Human-facing documentation
- `docs/cli/` - Command reference
- `docs/cue/` - CUE schema reference and examples

The prototype's CLI documentation (`../start-prototype/docs/cli/*.md`) provides a starting point for the user interface, though commands will evolve as CUE capabilities are integrated.

## Document Driven Development (DDD)

This project uses Document Driven Development. Design decisions are documented in Design Records (DRs) before implementation.

**Location:** `.ai/design/design-records/`

**Process:**

- Create DRs for architectural decisions, algorithms, breaking changes, API/CLI structure
- Get next DR number from `.ai/design/design-records/README.md`
- Follow reconciliation process after 5-10 DRs

Design Records will start fresh at dr-001, as the CUE architecture is fundamentally different from the prototype.

## Testing

When implementing code, ensure it is testable. Read `.ai/design/design-records/dr-024-testing-strategy.md` for the testing approach.

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

## Start Assets Repository

The `./start-assets/` directory contains the cloned [start-assets](https://github.com/grantcarthew/start-assets) repository for local development and testing. This directory is git-ignored.

Use for: Developing and testing new assets, schemas, and registry content before publishing.

```
start-assets/
├── agents/          # Agent definitions (claude, gemini, aichat)
├── contexts/        # Context definitions (agents, environment, project)
├── index/           # Registry index module
├── roles/           # Role definitions (golang, start-assets)
├── schemas/         # CUE schema definitions for all asset types
└── tasks/           # Task definitions (golang, start)
```

## Development Status

**Completed:**

- p-001: CUE Foundation & Architecture (schemas designed and published)
- p-002: Concrete Assets Validation (17 modules published to CUE Central Registry)
- p-003: Registry Distribution (20 modules published, prototype comparison documented)
- p-004: Minimal CLI Implementation (CUE loading, show command, global flags)
- p-005: Orchestration Core Engine (template processing, shell execution, composition, CLI commands)
- p-006: Auto-Setup (registry interaction, agent detection, TTY prompts, config writing)
- p-007: Package Management (assets commands for browsing/installing registry packages)
- p-008: Configuration Editing (config agent/role/context/task/settings commands)
- p-009: Doctor Diagnostics (health checks, configuration validation, fix suggestions)
- p-010: Shell Completion (bash, zsh, fish tab-completion, documentation guides)
- p-011: CLI Refinements (command naming, exit codes)
- p-015: Schema Base and Origin Tracking (#Base schema, origin field for registry provenance)
- p-016: File Placeholder Temp Path ({{.file}} now uses .start/temp/ instead of CUE cache path)
- p-012: CLI Core Commands Testing (start, prompt, task commands, global flags, error handling)
- p-017: CLI Config Edit Flags (non-interactive flag-based editing for config edit commands)
- p-013: CLI Configuration Commands Testing (config commands, show commands, merging, --local flag)
- p-014: CLI Supporting Commands Testing (assets, doctor, completion commands)
- p-019: CLI File Path Inputs (file path support for --role, --context, task, and prompt commands)
- p-018: CLI Interactive Edit Completeness (models and tags editing in interactive mode)
- p-020: Role Optional Field (optional field for discovery-based roles, graceful fallback)
- p-021: Auto-Setup Default Assets (internal/assets package, default context installation)
- p-023: CLI Config Reorder (interactive reordering, definition order preservation)
- p-024: CLI Flag Asset Search (three-tier resolution for --agent, --role, --model, --context flags)
- p-025: Terminal Colour Standard (DR-042 applied across all CLI output, added prompt colour)

The prototype validated that this tool solves a real problem. This version will implement it properly.
