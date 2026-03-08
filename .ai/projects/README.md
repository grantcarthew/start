# Project Documents

This directory contains stand-alone project documents for building the `start` tool. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

### Active Projects

None

### Development Status

All planned projects complete. Rich `--json` output implemented across all data commands (issue #69) via p-039 and p-040.

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
| [p-034](./completed/p-034-cli-config-add-edit-flags-removal.md) | CLI Config Add/Edit Flags Removal | 2026-02-24 |
| [p-035](./completed/p-035-cli-config-open-command.md) | CLI Config Open Command | 2026-02-24 |
| [p-036](./completed/p-036-cli-config-types-migration.md) | CLI Config Types Migration | 2026-02-24 |
| [p-037](./completed/p-037-cli-config-order-category-arg.md) | CLI Config Order Category Argument | 2026-02-24 |
| [p-032](./completed/p-032-cli-config-verb-first-refactor.md) | CLI Config Verb-First Refactor | 2026-02-24 |
| [p-038](./completed/p-038-cli-index-cache.md) | CLI Index Cache | 2026-03-02 |
| [p-039](./completed/p-039-cli-json-output-assets.md) | CLI JSON Output - Shared Prep and Assets Commands | 2026-03-06 |
| [p-040](./completed/p-040-cli-json-output-config-doctor.md) | CLI JSON Output - Config, Search, and Doctor Commands | 2026-03-06 |
| [p-041](./completed/p-041-cli-unified-asset-resolution.md) | Unified Asset Resolution | 2026-03-08 |

---

## Status Values

- **Proposed** - Project defined, not yet started
- **In Progress** - Currently being worked on
- **Completed** - All success criteria met, deliverables created
- **Blocked** - Waiting on external dependency or decision
- **Deferred** - Intentionally postponed

---

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
