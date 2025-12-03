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
