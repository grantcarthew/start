# Project Documents

This directory contains stand-alone project documents for building the `start` tool. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

### Active Projects

| Project | Title | Started |
|---------|-------|---------|
| [p-021](./p-021-auto-setup-default-assets.md) | Auto-Setup Default Assets | Pending |

### Completed Projects

| Project | Title | Completed |
|---------|-------|-----------|
| [p-001](./completed/p-001-cue-foundation-architecture.md) | CUE Foundation & Architecture | 2025-12-05 |
| [p-002](./completed/p-002-assets-validation.md) | Assets Validation | 2025-12-08 |
| [p-003](./completed/p-003-registry-distribution.md) | Registry Distribution | 2025-12-10 |
| [p-004](./completed/p-004-cli-minimal-implementation.md) | CLI Minimal Implementation | 2025-12-12 |
| [p-005](./completed/p-005-orchestration-core-engine.md) | Orchestration Core Engine | 2025-12-15 |
| [p-006](./completed/p-006-auto-setup.md) | Auto-Setup | 2025-12-17 |
| [p-007](./completed/p-007-package-management.md) | Package Management | 2025-12-18 |
| [p-008](./completed/p-008-configuration-editing.md) | Configuration Editing | 2025-12-19 |
| [p-009](./completed/p-009-doctor-diagnostics.md) | Doctor & Diagnostics | 2025-12-19 |
| [p-010](./completed/p-010-shell-completion.md) | Shell Completion | 2025-12-19 |
| [p-011](./completed/p-011-cli-refinements.md) | CLI Refinements | 2025-12-22 |
| [p-015](./completed/p-015-schema-base-origin-tracking.md) | Schema Base and Origin Tracking | 2026-01-05 |
| [p-016](./completed/p-016-file-placeholder-temp-path.md) | File Placeholder Temp Path | 2026-01-13 |
| [p-012](./completed/p-012-cli-end-to-end-testing.md) | CLI Core Commands Testing | 2026-01-13 |
| [p-013](./completed/p-013-cli-configuration-testing.md) | CLI Configuration Testing | 2026-01-15 |
| [p-014](./completed/p-014-cli-supporting-commands-testing.md) | CLI Supporting Commands Testing | 2026-01-15 |
| [p-017](./completed/p-017-cli-config-edit-flags.md) | CLI Config Edit Flags | 2026-01-16 |
| [p-019](./completed/p-019-cli-file-path-inputs.md) | CLI File Path Inputs | 2026-01-16 |
| [p-018](./completed/p-018-cli-interactive-edit-completeness.md) | CLI Interactive Edit Completeness | 2025-01-19 |
| [p-020](./completed/p-020-role-optional-field.md) | Role Optional Field | 2026-01-31 |

---

## Project Overview

### Active

#### p-021: Auto-Setup Default Assets

Extract asset installation logic to shared package and enhance auto-setup to install commonly-needed contexts (starting with `cwd/agents-md`) during first-run configuration.

**Key Deliverables:** `internal/assets` package, updated auto-setup, updated DR-018

**Dependencies:** p-006, p-007

### Completed

#### p-020: Role Optional Field

Added `optional` field to role schema enabling discovery-based roles (like dotai roles) that gracefully skip when their files don't exist. Roles iterate in definition order; optional roles skip to next, required roles error. Updated error display to show UI before errors.

**Key Deliverables:** Schema update (schemas/v0.2.0), composer role resolution, `--optional` CLI flag, dr-039, updated dr-007

**Dependencies:** None

#### p-018: CLI Interactive Edit Completeness

Enhanced interactive edit mode for `config <type> edit <name>` commands to support editing all fields (models, tags) that were previously only available via add commands.

**Key Deliverables:** `promptTags` and `promptModels` helper functions, updated edit commands for agent/role/context/task, 14 unit tests

**Dependencies:** p-008

#### p-012: CLI Core Commands Testing

End-to-end testing of CLI core commands (start, prompt, task) from a user's perspective. Tested first-run experience, global flags, and error handling. Fixed 20 issues discovered during testing.

**Key Deliverables:** Completed test checklist, fixed issues, dr-012 updated for --verbose scope

**Dependencies:** All prior projects

#### p-016: File Placeholder Temp Path

Fixed bug where `{{.file}}` template placeholder returned CUE cache path instead of local `.start/temp/` path. This caused permission issues when AI agents attempted to read files from external cache directories.

**Key Deliverables:** Updated Composer to pre-write temp files, added `@module/` resolution to roles/contexts

**Dependencies:** p-005, p-015

#### p-015: Schema Base and Origin Tracking

Added `#Base` schema with origin field for tracking registry provenance of assets. Enables `@module/` path resolution for file-based UTDs.

**Key Deliverables:** `#Base` schema, origin field, validation script

**Dependencies:** p-001, p-003

#### p-011: CLI Refinements

Addressed CLI usability issues: resolved naming collision between `config <type> show` and `show <type>` by renaming to `info`, and established unified exit code policy (0 success, 1 failure) across all commands.

**Key Deliverables:** Renamed config subcommands, updated 9 design records with exit code policy

**Dependencies:** All prior projects

#### p-010: Shell Completion

Shell completion for bash, zsh, and fish using Cobra's built-in completion generation. Static completion for commands and flags.

**Key Deliverables:** `start completion` commands for bash/zsh/fish, dr-032

**Dependencies:** All prior projects

#### p-009: Doctor & Diagnostics

Health checks, configuration validation, and diagnostics to help users identify and fix issues.

**Key Deliverables:** `start doctor` command with validation checks and fix suggestions, dr-031

**Dependencies:** p-004, p-006

#### p-008: Configuration Editing

Configuration editing commands for managing agents, roles, contexts, tasks, and settings without manually editing CUE files.

**Key Deliverables:** `start config` commands for all entity types, dr-029, dr-030

**Dependencies:** p-004, p-006

#### p-007: Package Management

Package management commands for discovering, adding, and updating assets from CUE Central Registry.

**Key Deliverables:** `start assets` commands (search, add, list, info, update, browse, index), dr-028

**Dependencies:** p-006

#### p-006: Auto-Setup

First-run experience: detect installed AI CLI tools, fetch configuration from registry, write to user config. Enables zero-config to agent launch workflow.

**Key Deliverables:** Registry interaction, agent detection, auto-setup flow, E2E validation, dr-027

**Dependencies:** p-001, p-002, p-003, p-004, p-005

#### p-005: Orchestration Core Engine

Core orchestration logic: UTD template processing, shell execution, prompt composition, agent execution, CLI commands (start, prompt, task).

**Key Deliverables:** Template processor, shell runner, composer, executor, CLI commands

**Dependencies:** p-001, p-004

#### p-004: CLI Minimal Implementation

Built minimal CLI infrastructure to validate CUE-based architecture end-to-end. Implemented CUE loading/validation, `start show` command with subcommands, and global flags.

**Key Deliverables:** CUE loader/validator, show command, dr-025 (merge semantics), dr-026 (I/O separation)

**Dependencies:** p-001, p-002, p-003

#### p-001: CUE Foundation & Architecture

Research CUE capabilities and design the foundational architecture. Establishes how CUE will be used for configuration, validation, schemas, and modules.

**Key Deliverables:** Schema designs, dr-001 through dr-011, architecture documentation

#### p-002: Assets Validation

Create real assets (roles, tasks, contexts, agents) in CUE to validate the schema designs from p-001. 17 modules published to CUE Central Registry.

**Key Deliverables:** Example assets, refined schemas, published modules

#### p-003: Registry Distribution

Define how assets are distributed using CUE Central Registry. Validated that CUE registry replaces prototype's custom GitHub asset system (dr-031-042).

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

1. List directory to find next number: `ls .ai/projects/p-*.md`
2. Use format: `p-<NNN>-<category>-<title>.md`
3. Follow the structure in [p-writing-guide.md](./p-writing-guide.md)
4. Define clear, measurable success criteria
5. Update this README with project entry
6. Link dependencies to other projects
