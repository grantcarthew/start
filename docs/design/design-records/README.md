# Design Records

Design Records (DRs) document architectural decisions, algorithms, breaking changes, and API/CLI structure for the start project.

## Active Design Records

| Number | Title | Category | Status | Date |
|--------|-------|----------|--------|------|
| DR-001 | User-Controlled Defaults in CUE Schemas | CUE | Accepted | 2025-12-02 |
| DR-002 | No Name Field in Task Schema | CUE | Accepted | 2025-12-02 |
| DR-003 | Index Category Structure | Index | Accepted | 2025-12-02 |
| DR-004 | Module Naming Convention | Module | Accepted | 2025-12-02 |
| DR-005 | Go Templates for UTD Pattern | UTD | Accepted | 2025-12-02 |
| DR-006 | Shell Configuration and Command Execution | UTD | Accepted | 2025-12-02 |
| DR-007 | UTD Error Handling by Context | UTD | Accepted | 2025-12-02 |
| DR-008 | Context Schema Design | CUE | Accepted | 2025-12-03 |
| DR-009 | Task Schema Design | CUE | Accepted | 2025-12-03 |
| DR-010 | Role Schema Design | CUE | Accepted | 2025-12-03 |
| DR-011 | Agent Schema Design | CUE | Accepted | 2025-12-03 |
| DR-012 | CLI Global Flags | CLI | Accepted | 2025-12-03 |
| DR-013 | CLI Start Command | CLI | Accepted | 2025-12-03 |
| DR-014 | CLI Prompt Command | CLI | Accepted | 2025-12-03 |
| DR-015 | CLI Task Command | CLI | Accepted | 2025-12-03 |
| DR-016 | CLI Dry Run Flag | CLI | Accepted | 2025-12-04 |
| DR-017 | CLI Show Command | CLI | Accepted | 2025-12-04 |
| DR-018 | CLI Auto-Setup | CLI | Accepted | 2025-12-04 |
| DR-019 | Index Bin Field for Agent Detection | Index | Accepted | 2025-12-04 |
| DR-020 | Template Processing and File Resolution | UTD | Accepted | 2025-12-08 |
| DR-021 | Module and Package Naming Conventions | CUE | Accepted | 2025-12-08 |
| DR-022 | Task Role CUE Dependencies | CUE | Accepted | 2025-12-09 |
| DR-023 | Module Path Prefix for File Resolution | CUE | Accepted | 2025-12-09 |
| DR-024 | Testing Strategy | Testing | Accepted | 2025-12-11 |
| DR-025 | Configuration Merge Semantics | Config | Accepted | 2025-12-11 |
| DR-026 | CLI Logic and I/O Separation | CLI | Accepted | 2025-12-12 |
| DR-027 | Registry Module Fetching | Registry | Accepted | 2025-12-15 |
| DR-028 | CLI Assets Command | CLI | Accepted | 2025-12-17 |
| DR-029 | CLI Configuration Editing Commands | CLI | Accepted | 2025-12-18 |

## Process

1. Create DRs for significant design decisions
2. Use format: `dr-<NNN>-<category>-<title>.md`
3. Follow template in dr-writing-guide.md
4. List directory to find next number: `ls docs/design/design-records/dr-*.md`
5. After 5-10 DRs, perform reconciliation to update core documentation

## Categories

Common categories used in DRs:

- CUE - CUE language and schema decisions
- UTD - Unified Template Design pattern
- CLI - Command-line interface design
- Index - Asset discovery and search
- Module - Module organization and naming
- Task - Task-specific design
- Config - Configuration structure
- Agent - Agent configuration
- Role - Role definitions
- Context - Context document configuration
- Testing - Testing strategy and patterns
- Registry - CUE registry interaction and module fetching
