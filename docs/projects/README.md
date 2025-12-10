# Project Documents

This directory contains stand-alone project documents for building the `start` tool. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

### Active Projects

| Project | Title | Status | Started |
|---------|-------|--------|---------|
| [P-004](./p-004-cli-minimal-implementation.md) | CLI Minimal Implementation | Proposed | - |
| [P-005](./p-005-orchestration-core-engine.md) | Orchestration Core Engine | Proposed | - |

### Completed Projects

| Project | Title | Completed |
|---------|-------|-----------|
| [P-001](./completed/p-001-cue-foundation-architecture.md) | CUE Foundation & Architecture | 2025-12-05 |
| [P-002](./completed/p-002-assets-validation.md) | Assets Validation | 2025-12-08 |
| [P-003](./completed/p-003-registry-distribution.md) | Registry Distribution | 2025-12-10 |

---

## Project Overview

### Active

#### P-004: CLI Minimal Implementation

Build minimal CLI commands to interact with CUE configurations. Adapted from prototype CLI design but CUE-native.

**Key Deliverables:** `start init`, `start show`, CLI structure, additional DRs

**Dependencies:** P-001, P-002, P-003

#### P-005: Orchestration Core Engine

Implement core orchestration logic: load CUE from Go, compose prompts, execute agent commands. End-to-end integration.

**Key Deliverables:** Working orchestrator, Go-CUE integration, execution model

**Dependencies:** P-001, P-004

### Completed

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
