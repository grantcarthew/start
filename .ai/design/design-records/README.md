# Design Records

Design Records (DRs) document architectural decisions, algorithms, breaking changes, and API/CLI structure for the start project.

## Active Design Records

| Number | Title | Category | Status | Date |
|--------|-------|----------|--------|------|
| dr-001 | User-Controlled Defaults in CUE Schemas | CUE | Accepted | 2025-12-02 |
| dr-002 | No Name Field in Task Schema | CUE | Accepted | 2025-12-02 |
| dr-003 | Index Category Structure | Index | Accepted | 2025-12-02 |
| dr-004 | Module Naming Convention | Module | Accepted | 2025-12-02 |
| dr-005 | Go Templates for UTD Pattern | UTD | Accepted | 2025-12-02 |
| dr-006 | Shell Configuration and Command Execution | UTD | Accepted | 2025-12-02 |
| dr-007 | UTD Error Handling by Context | UTD | Accepted | 2025-12-02 |
| dr-008 | Context Schema Design | CUE | Accepted | 2025-12-03 |
| dr-009 | Task Schema Design | CUE | Accepted | 2025-12-03 |
| dr-010 | Role Schema Design | CUE | Accepted | 2025-12-03 |
| dr-011 | Agent Schema Design | CUE | Accepted | 2025-12-03 |
| dr-012 | CLI Global Flags | CLI | Accepted | 2025-12-03 |
| dr-013 | CLI Start Command | CLI | Accepted | 2025-12-03 |
| dr-014 | CLI Prompt Command | CLI | Accepted | 2025-12-03 |
| dr-015 | CLI Task Command | CLI | Accepted | 2025-12-03 |
| dr-016 | CLI Dry Run Flag | CLI | Accepted | 2025-12-04 |
| dr-017 | CLI Show Command | CLI | Accepted | 2025-12-04 |
| dr-018 | CLI Auto-Setup | CLI | Accepted | 2025-12-04 |
| dr-019 | Index Bin Field for Agent Detection | Index | Accepted | 2025-12-04 |
| dr-020 | Template Processing and File Resolution | UTD | Accepted | 2025-12-08 |
| dr-021 | Module and Package Naming Conventions | CUE | Accepted | 2025-12-08 |
| dr-022 | Task Role CUE Dependencies | CUE | Accepted | 2025-12-09 |
| dr-023 | Module Path Prefix for File Resolution | CUE | Accepted | 2025-12-09 |
| dr-024 | Testing Strategy | Testing | Accepted | 2025-12-11 |
| dr-025 | Configuration Merge Semantics | Config | Accepted | 2025-12-11 |
| dr-026 | CLI Logic and I/O Separation | CLI | Accepted | 2025-12-12 |
| dr-027 | Registry Module Fetching | Registry | Accepted | 2025-12-15 |
| dr-028 | CLI Assets Command | CLI | Accepted | 2025-12-17 |
| dr-029 | CLI Configuration Editing Commands | CLI | Accepted | 2025-12-18 |
| dr-030 | Settings Schema | Config | Accepted | 2025-12-18 |
| dr-031 | CLI Doctor Command | CLI | Accepted | 2025-12-19 |
| dr-032 | CLI Shell Completion | CLI | Accepted | 2025-12-19 |
| dr-033 | Additional CLI Global Flags | CLI | Accepted | 2025-12-19 |
| dr-034 | CLI Parent Command Defaults | CLI | Accepted | 2025-12-23 |
| dr-035 | CLI Debug Logging | CLI | Proposed | 2025-12-23 |
| dr-036 | CLI Terminal Colors | CLI | Superseded by dr-042 | 2025-12-24 |
| dr-037 | Base Schema for Common Asset Fields | CUE | Proposed | 2026-01-05 |
| dr-038 | CLI File Path Inputs | CLI | Accepted | 2026-01-16 |
| dr-039 | Role Optional Field | Role | Accepted | 2026-01-31 |
| dr-040 | CLI Config Reorder Command | CLI | Accepted | 2026-02-08 |
| dr-041 | CLI Flag Asset Search | CLI | Accepted | 2026-02-08 |
| dr-042 | Terminal Colour Standard | CLI | Accepted | 2026-02-10 |

## Process

1. Create DRs for significant design decisions
2. Use format: `dr-<NNN>-<category>-<title>.md`
3. Follow template in dr-writing-guide.md
4. List directory to find next number: `ls .ai/design/design-records/dr-*.md`
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
