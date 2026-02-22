# Project Documents

This directory contains stand-alone project documents for building the `start` tool. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

### Active Projects

None

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
| [p-021](./completed/p-021-auto-setup-default-assets.md) | Auto-Setup Default Assets | 2026-02-08 |
| [p-023](./completed/p-023-cli-config-reorder.md) | CLI Config Reorder | 2026-02-08 |
| [p-024](./completed/p-024-cli-flag-asset-search.md) | CLI Flag Asset Search | 2026-02-13 |
| [p-025](./completed/p-025-terminal-colour-standard.md) | Terminal Colour Standard | 2026-02-13 |
| [p-026](./completed/p-026-cli-config-loader-return-type.md) | CLI Config Loader Return Type Alignment | 2026-02-19 |
| [p-027](./completed/p-027-cli-content-source-menu.md) | CLI Content Source Menu Extraction | 2026-02-19 |
| [p-028](./completed/p-028-tui-shared-package.md) | TUI Shared Package | 2026-02-19 |
| [p-029](./completed/p-029-cli-show-verbose-inspection.md) | CLI Show Verbose Inspection | 2026-02-20 |
| [p-022](./completed/p-022-assets-ast-refactor.md) | Assets AST Refactor | 2026-02-20 |
| [p-030](./completed/p-030-assets-index-setting.md) | Assets Index Setting | 2026-02-20 |

---

## Project Overview

### Active

None

### Completed

#### p-030: Assets Index Setting

Made the CUE module path for the start-assets index configurable via `assets_index` in `#Settings`. All `assets` subcommands and `doctor` read this setting and fall back to the hardcoded `IndexModulePath` constant when absent.

**Key Deliverables:** `assets_index` in settings schema and `validSettingsKeys`, `EffectiveIndexPath` helper, updated `FetchIndex` signature, updated asset commands and doctor

**Dependencies:** None

#### p-022: Assets AST Refactor

Refactored asset installation code to use CUE's AST APIs instead of fragile string manipulation. Replaced 160+ lines of manual parsing with proper AST operations.

**Key Deliverables:** AST-based `internal/assets/install.go`, removed string manipulation, updated tests

**Dependencies:** p-021

#### p-029: CLI Show Verbose Inspection

Enhanced `start show` into a comprehensive resource inspection tool with cross-category search, verbose dump (CUE definitions, file contents, origin/cache metadata), and enhanced listing with descriptions.

**Key Deliverables:** Rewritten `show.go` with verbose dump and cross-category search, exported wrappers in `composer.go`, updated tests

**Dependencies:** p-024, p-025

#### p-028: TUI Shared Package

Extracted shared colour definitions and `Annotate`/`Bracket` helpers into `internal/tui` package. Migrated CLI and doctor output to use the shared package.

**Key Deliverables:** `internal/tui` package, migrated cli and doctor packages

**Dependencies:** None

#### p-027: CLI Content Source Menu Extraction

Normalised and extracted `promptContentSource` helper to reduce duplication across config commands.

**Key Deliverables:** Shared `promptContentSource` helper, updated config commands

**Dependencies:** None

#### p-026: CLI Config Loader Return Type Alignment

Updated agents and tasks config loaders to return definition order alongside the map, matching the pattern established for contexts and roles.

**Key Deliverables:** Updated loader return types, updated consumers

**Dependencies:** None

#### p-025: Terminal Colour Standard

Applied DR-042 terminal colour standard across all CLI output. Audited every command visually, added category colours to info commands, styled all interactive prompts with cyan delimiters and dim metadata, coloured execution output, and added magenta as the prompt category colour.

**Key Deliverables:** Coloured info/add/edit/remove/default/order commands, styled execution output, `colorPrompts`, updated DR-042, content preview threshold

**Dependencies:** p-004, p-008

#### p-024: CLI Flag Asset Search

Extended substring search to `--agent`, `--role`, `--model`, and `--context` flags. Three-tier resolution (exact config, exact registry, substring search) with auto-install. Changed `--context` from tag-only to unified search.

**Key Deliverables:** `resolve.go` resolver, `SearchInstalledConfig()`, flag integration in start.go/task.go, DR-041

**Dependencies:** p-021

#### p-023: CLI Config Reorder

Added `order`/`reorder` command to context and role configuration for interactive move-up reordering. Refactored write functions to preserve definition order instead of alphabetically sorting.

**Key Deliverables:** `config_order.go`, refactored write/load functions, DR-040

**Dependencies:** None

#### p-021: Auto-Setup Default Assets

Extracted asset installation logic to shared `internal/assets` package and enhanced auto-setup to install commonly-needed contexts (`cwd/agents-md`) during first-run configuration.

**Key Deliverables:** `internal/assets` package, updated auto-setup, updated DR-018

**Dependencies:** p-006, p-007

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
