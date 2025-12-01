# Project Documents

This directory contains stand-alone project documents for building the `start` tool. Each project represents a focused effort with clear goals, scope, and success criteria.

See [p-writing-guide.md](./p-writing-guide.md) for guidelines on creating and maintaining project documents.

---

## Quick Reference

| Project | Title | Status | Started | Completed |
|---------|-------|--------|---------|-----------|
| [P-001](./p-001-cue-foundation-architecture.md) | CUE Foundation & Architecture | Proposed | - | - |
| [P-002](./p-002-concrete-assets.md) | Concrete Assets - Validate Design | Proposed | - | - |
| [P-003](./p-003-distribution-strategy.md) | Distribution Strategy | Proposed | - | - |
| [P-004](./p-004-minimal-cli.md) | Minimal CLI Implementation | Proposed | - | - |
| [P-005](./p-005-orchestration-core.md) | Orchestration Core | Proposed | - | - |

Next Number: P-006

---

## Project Overview

### P-001: CUE Foundation & Architecture

Research CUE capabilities and design the foundational architecture. Establishes how CUE will be used for configuration, validation, schemas, and modules.

**Key Deliverables:** Schema designs, DR-001, DR-002, architecture documentation

**Dependencies:** None (starting point)

### P-002: Concrete Assets - Validate Design

Create real assets (roles, tasks, contexts, agents) in CUE to validate the schema designs from P-001. This provides concrete examples and surfaces design issues early.

**Key Deliverables:** Example assets, refined schemas, DR-003

**Dependencies:** P-001 (need schemas first)

### P-003: Distribution Strategy

Define how assets are distributed and consumed using CUE Central Registry. Research packaging, publishing, versioning, and dependency management.

**Key Deliverables:** Package structure, registry strategy, DR-004

**Dependencies:** P-002 (need concrete assets to package)

### P-004: Minimal CLI Implementation

Build minimal CLI commands to interact with CUE configurations. Adapted from prototype CLI design but CUE-native.

**Key Deliverables:** `start init`, `start show`, CLI structure, DR-005

**Dependencies:** P-001 (need architecture), P-002 (need examples to test)

### P-005: Orchestration Core

Implement core orchestration logic: load CUE from Go, compose prompts, execute agent commands. End-to-end integration.

**Key Deliverables:** Working orchestrator, Go-CUE integration, execution model

**Dependencies:** P-001, P-004 (need architecture and CLI)

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

1. Use next sequential number: `p-NNN-short-name.md`
2. Follow the structure in [p-writing-guide.md](./p-writing-guide.md)
3. Define clear, measurable success criteria
4. Update this README with project entry
5. Link dependencies to other projects
