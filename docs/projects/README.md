# Project Documents

This directory contains stand-alone project documents for building the `start` tool. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

### Active Projects

| Project | Title | Started |
|---------|-------|---------|
| [P-013](./p-013-cli-configuration-testing.md) | CLI Configuration Testing | - |
| [P-014](./p-014-cli-supporting-commands-testing.md) | CLI Supporting Commands Testing | - |

### Completed Projects

| Project | Title | Completed |
|---------|-------|-----------|
| [P-001](./completed/p-001-cue-foundation-architecture.md) | CUE Foundation & Architecture | 2025-12-05 |
| [P-002](./completed/p-002-assets-validation.md) | Assets Validation | 2025-12-08 |
| [P-003](./completed/p-003-registry-distribution.md) | Registry Distribution | 2025-12-10 |
| [P-004](./completed/p-004-cli-minimal-implementation.md) | CLI Minimal Implementation | 2025-12-12 |
| [P-005](./completed/p-005-orchestration-core-engine.md) | Orchestration Core Engine | 2025-12-15 |
| [P-006](./completed/p-006-auto-setup.md) | Auto-Setup | 2025-12-17 |
| [P-007](./completed/p-007-package-management.md) | Package Management | 2025-12-18 |
| [P-008](./completed/p-008-configuration-editing.md) | Configuration Editing | 2025-12-19 |
| [P-009](./completed/p-009-doctor-diagnostics.md) | Doctor & Diagnostics | 2025-12-19 |
| [P-010](./completed/p-010-shell-completion.md) | Shell Completion | 2025-12-19 |
| [P-011](./completed/p-011-cli-refinements.md) | CLI Refinements | 2025-12-22 |
| [P-015](./completed/p-015-schema-base-origin-tracking.md) | Schema Base and Origin Tracking | 2026-01-05 |
| [P-016](./completed/p-016-file-placeholder-temp-path.md) | File Placeholder Temp Path | 2026-01-13 |
| [P-012](./completed/p-012-cli-end-to-end-testing.md) | CLI Core Commands Testing | 2026-01-13 |

---

## Project Overview

### Completed

#### P-012: CLI Core Commands Testing

End-to-end testing of CLI core commands (start, prompt, task) from a user's perspective. Tested first-run experience, global flags, and error handling. Fixed 20 issues discovered during testing.

**Key Deliverables:** Completed test checklist, fixed issues, DR-012 updated for --verbose scope

**Dependencies:** All prior projects

#### P-016: File Placeholder Temp Path

Fixed bug where `{{.file}}` template placeholder returned CUE cache path instead of local `.start/temp/` path. This caused permission issues when AI agents attempted to read files from external cache directories.

**Key Deliverables:** Updated Composer to pre-write temp files, added `@module/` resolution to roles/contexts

**Dependencies:** P-005, P-015

#### P-015: Schema Base and Origin Tracking

Added `#Base` schema with origin field for tracking registry provenance of assets. Enables `@module/` path resolution for file-based UTDs.

**Key Deliverables:** `#Base` schema, origin field, validation script

**Dependencies:** P-001, P-003

#### P-011: CLI Refinements

Addressed CLI usability issues: resolved naming collision between `config <type> show` and `show <type>` by renaming to `info`, and established unified exit code policy (0 success, 1 failure) across all commands.

**Key Deliverables:** Renamed config subcommands, updated 9 design records with exit code policy

**Dependencies:** All prior projects

#### P-010: Shell Completion

Shell completion for bash, zsh, and fish using Cobra's built-in completion generation. Static completion for commands and flags.

**Key Deliverables:** `start completion` commands for bash/zsh/fish, DR-032

**Dependencies:** All prior projects

#### P-009: Doctor & Diagnostics

Health checks, configuration validation, and diagnostics to help users identify and fix issues.

**Key Deliverables:** `start doctor` command with validation checks and fix suggestions, DR-031

**Dependencies:** P-004, P-006

#### P-008: Configuration Editing

Configuration editing commands for managing agents, roles, contexts, tasks, and settings without manually editing CUE files.

**Key Deliverables:** `start config` commands for all entity types, DR-029, DR-030

**Dependencies:** P-004, P-006

#### P-007: Package Management

Package management commands for discovering, adding, and updating assets from CUE Central Registry.

**Key Deliverables:** `start assets` commands (search, add, list, info, update, browse, index), DR-028

**Dependencies:** P-006

#### P-006: Auto-Setup

First-run experience: detect installed AI CLI tools, fetch configuration from registry, write to user config. Enables zero-config to agent launch workflow.

**Key Deliverables:** Registry interaction, agent detection, auto-setup flow, E2E validation, DR-027

**Dependencies:** P-001, P-002, P-003, P-004, P-005

#### P-005: Orchestration Core Engine

Core orchestration logic: UTD template processing, shell execution, prompt composition, agent execution, CLI commands (start, prompt, task).

**Key Deliverables:** Template processor, shell runner, composer, executor, CLI commands

**Dependencies:** P-001, P-004

#### P-004: CLI Minimal Implementation

Built minimal CLI infrastructure to validate CUE-based architecture end-to-end. Implemented CUE loading/validation, `start show` command with subcommands, and global flags.

**Key Deliverables:** CUE loader/validator, show command, DR-025 (merge semantics), DR-026 (I/O separation)

**Dependencies:** P-001, P-002, P-003

#### P-001: CUE Foundation & Architecture

Research CUE capabilities and design the foundational architecture. Establishes how CUE will be used for configuration, validation, schemas, and modules.

**Key Deliverables:** Schema designs, DR-001 through DR-011, architecture documentation

#### P-002: Assets Validation

Create real assets (roles, tasks, contexts, agents) in CUE to validate the schema designs from P-001. 17 modules published to CUE Central Registry.

**Key Deliverables:** Example assets, refined schemas, published modules

#### P-003: Registry Distribution

Define how assets are distributed using CUE Central Registry. Validated that CUE registry replaces prototype's custom GitHub asset system (DR-031-042).

**Key Deliverables:** 20 published modules, publishing guide, prototype comparison

---

## Status Values

- **Proposed** - Project defined, not yet started
- **In Progress** - Currently being worked on
- **Completed** - All success criteria met, deliverables created
- **Blocked** - Waiting on external dependency or decision
- **Deferred** - Intentionally postponed

---

## Projects vs Design Records

**Projects** are work packages that define **what to build** and **how to validate it**.

**Design Records (DRs)** document **why we chose** a specific approach and the trade-offs.

A single project may generate multiple DRs. Projects describe the work; DRs document the decisions made during that work.

See [p-writing-guide.md](./p-writing-guide.md) for detailed guidance.

---

## Contributing

When creating a new project:

1. List directory to find next number: `ls docs/projects/p-*.md`
2. Use format: `p-<NNN>-<category>-<title>.md`
3. Follow the structure in [p-writing-guide.md](./p-writing-guide.md)
4. Define clear, measurable success criteria
5. Update this README with project entry
6. Link dependencies to other projects
