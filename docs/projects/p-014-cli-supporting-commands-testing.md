# P-014: CLI Supporting Commands Testing

- Status: Proposed
- Started: -

## Overview

End-to-end testing of the `start` CLI supporting commands: assets management, doctor diagnostics, and shell completion. These commands support the core workflow but don't execute agents.

Part 3 of 3 in CLI testing series:
- P-012: Core Commands
- P-013: Configuration Commands
- P-014: Supporting Commands (this project)

## Goals

1. Test all assets subcommands (search, add, list, info, update, browse, index)
2. Test doctor diagnostics and health checks
3. Test shell completion generation (bash, zsh, fish)
4. Fix all issues discovered during testing

## Scope

In Scope:
- `start assets` (search, add, list, info, update, browse, index)
- `start doctor`
- `start completion` (bash, zsh, fish)

Out of Scope:
- Core execution commands (P-012)
- Configuration commands (P-013)

## Success Criteria

- [ ] All features tested and marked complete in checklist below
- [ ] All discovered issues fixed and verified
- [ ] No blocking issues remain

## Testing Workflow

For each feature:

1. Read the feature description and test steps
2. Execute the test commands
3. Record the result: PASS, FAIL, PARTIAL, or SKIP
4. If FAIL/PARTIAL:
   - Document the issue in Issues Log
   - Fix the issue
   - If fix involves design decisions: update related DR or create new DR
   - Retest and verify
5. Mark the feature as tested with brief notes

Design Record Updates:
- Bug fixes that change documented behaviour → update existing DR
- New design decisions during fixes → create new DR
- Reference DRs in Issues Log when applicable

---

## Feature Checklist

### 1. Assets Commands

#### 1.1 assets search

Description: Search registry for assets by keyword.

Test:
```bash
./start assets search role
./start assets search golang
```

Expected: Returns matching assets grouped by type.

Result: ____

Notes:

---

#### 1.2 assets search --verbose

Description: Verbose search shows tags and module paths.

Test:
```bash
./start assets search role --verbose
```

Expected: Shows additional detail (tags, full module paths).

Result: ____

Notes:

---

#### 1.3 assets search (Minimum 3 chars)

Description: Search requires minimum 3 characters.

Test:
```bash
./start assets search ab
```

Expected: Error about minimum query length.

Result: ____

Notes:

---

#### 1.4 assets search (No Results)

Description: No matches shows helpful message.

Test:
```bash
./start assets search xyznonexistent123
```

Expected: Message about no matches found.

Result: ____

Notes:

---

#### 1.5 assets add

Description: Install asset from registry.

Test:
```bash
./start assets add <package-query>
# Follow prompts if multiple matches
./start config role list  # or appropriate type
```

Expected: Asset installed and available in config.

Result: ____

Notes:

---

#### 1.6 assets add --local

Description: Install asset to local config.

Test:
```bash
./start assets add <package-query> --local
ls ./.start/
# Clean up
```

Expected: Asset installed to ./.start/.

Result: ____

Notes:

---

#### 1.7 assets add (Direct Path)

Description: Install by direct module path.

Test:
```bash
./start assets add golang/code-review
```

Expected: Installs specific asset without search.

Result: ____

Notes:

---

#### 1.8 assets list

Description: List installed registry assets.

Test:
```bash
./start assets list
```

Expected: Shows installed assets with versions and update status.

Result: ____

Notes:

---

#### 1.9 assets list --type

Description: Filter list by asset type.

Test:
```bash
./start assets list --type roles
./start assets list --type tasks
```

Expected: Shows only assets of specified type.

Result: ____

Notes:

---

#### 1.10 assets info

Description: Show detailed asset information.

Test:
```bash
./start assets info <package-name>
```

Expected: Shows description, tags, version, install status.

Result: ____

Notes:

---

#### 1.11 assets info (Search then Show)

Description: Info can search by query.

Test:
```bash
./start assets info "code review"
```

Expected: Searches then shows details.

Result: ____

Notes:

---

#### 1.12 assets update

Description: Update installed assets to latest.

Test:
```bash
./start assets update
```

Expected: Checks and applies updates, reports results.

Result: ____

Notes:

---

#### 1.13 assets update (Specific)

Description: Update specific assets.

Test:
```bash
./start assets update golang
```

Expected: Updates only matching assets.

Result: ____

Notes:

---

#### 1.14 assets update --dry-run

Description: Preview updates without applying.

Test:
```bash
./start assets update --dry-run
```

Expected: Shows what would be updated, no changes made.

Result: ____

Notes:

---

#### 1.15 assets update --force

Description: Force re-fetch even if current.

Test:
```bash
./start assets update --force
```

Expected: Re-fetches all assets regardless of version.

Result: ____

Notes:

---

#### 1.16 assets browse

Description: Open asset repository in browser.

Test:
```bash
./start assets browse
```

Expected: Opens browser to GitHub repository.

Result: ____

Notes:

---

#### 1.17 assets browse (Specific)

Description: Browse specific asset.

Test:
```bash
./start assets browse <package-name>
```

Expected: Opens browser to specific package.

Result: ____

Notes:

---

#### 1.18 assets index

Description: Regenerate index.cue in asset repo.

Test:
```bash
./start assets index
```

Expected: Error (not in asset repo) or regenerates index.

Result: ____

Notes:

---

#### 1.19 assets index (Not Asset Repo)

Description: Error when not in asset repository.

Test:
```bash
cd /tmp && /path/to/start assets index
```

Expected: Error about not being in asset repository.

Result: ____

Notes:

---

### 2. Doctor Command

#### 2.1 doctor (All Pass)

Description: Run diagnostics with valid configuration.

Test:
```bash
./start doctor
echo "Exit code: $?"
```

Expected: All checks pass, exit code 0.

Result: ____

Notes:

---

#### 2.2 doctor (Version Info)

Description: Shows version and build information.

Test:
```bash
./start doctor
```

Expected: Shows version at start of output.

Result: ____

Notes:

---

#### 2.3 doctor (Config Validation)

Description: Validates CUE configuration syntax.

Test:
```bash
./start doctor
```

Expected: Reports config file validation status.

Result: ____

Notes:

---

#### 2.4 doctor (Agent Binary Check)

Description: Checks if agent binaries exist.

Test:
```bash
./start doctor
```

Expected: Reports which agent binaries are available.

Result: ____

Notes:

---

#### 2.5 doctor (Missing Binary)

Description: Reports missing agent binary.

Test:
```bash
./start config agent add bad-agent --bin nonexistent-binary-xyz --command "cmd"
./start doctor
./start config agent remove bad-agent
```

Expected: Warning about missing binary, exit code 1.

Result: ____

Notes:

---

#### 2.6 doctor (Context File Check)

Description: Checks if context files exist.

Test:
```bash
./start config context add bad-ctx --file /nonexistent/file.md
./start doctor
./start config context remove bad-ctx
```

Expected: Warning about missing context file.

Result: ____

Notes:

---

#### 2.7 doctor (Role File Check)

Description: Checks if role files exist.

Test:
```bash
./start config role add bad-role --file /nonexistent/role.md
./start doctor
./start config role remove bad-role
```

Expected: Warning about missing role file.

Result: ____

Notes:

---

#### 2.8 doctor (Exit Code on Issues)

Description: Exit code 1 when issues found.

Test:
```bash
./start config agent add bad --bin nonexistent --command "cmd"
./start doctor
echo "Exit code: $?"
./start config agent remove bad
```

Expected: Exit code 1.

Result: ____

Notes:

---

#### 2.9 doctor (Suggestions)

Description: Provides fix suggestions.

Test:
```bash
./start config agent add bad --bin nonexistent --command "cmd"
./start doctor
./start config agent remove bad
```

Expected: Suggests how to fix issues.

Result: ____

Notes:

---

### 3. Completion Commands

#### 3.1 completion bash

Description: Generate bash completion script.

Test:
```bash
./start completion bash > /tmp/start.bash
head -5 /tmp/start.bash
```

Expected: Generates valid bash script.

Result: ____

Notes:

---

#### 3.2 completion zsh

Description: Generate zsh completion script.

Test:
```bash
./start completion zsh > /tmp/start.zsh
head -5 /tmp/start.zsh
```

Expected: Generates valid zsh script.

Result: ____

Notes:

---

#### 3.3 completion fish

Description: Generate fish completion script.

Test:
```bash
./start completion fish > /tmp/start.fish
head -5 /tmp/start.fish
```

Expected: Generates valid fish script.

Result: ____

Notes:

---

#### 3.4 completion bash --help

Description: Shows installation instructions.

Test:
```bash
./start completion bash --help
```

Expected: Shows how to install bash completion.

Result: ____

Notes:

---

#### 3.5 completion zsh --help

Description: Shows installation instructions.

Test:
```bash
./start completion zsh --help
```

Expected: Shows how to install zsh completion.

Result: ____

Notes:

---

#### 3.6 completion fish --help

Description: Shows installation instructions.

Test:
```bash
./start completion fish --help
```

Expected: Shows how to install fish completion.

Result: ____

Notes:

---

#### 3.7 completion (Bash Integration)

Description: Completion works in bash.

Test:
```bash
source <(./start completion bash)
./start <TAB><TAB>  # Should show commands
```

Expected: Tab completion shows available commands.

Result: ____

Notes:

---

### 4. Help and Discoverability

#### 4.1 assets --help

Description: Shows assets command help.

Test:
```bash
./start assets --help
```

Expected: Lists all assets subcommands.

Result: ____

Notes:

---

#### 4.2 doctor --help

Description: Shows doctor command help.

Test:
```bash
./start doctor --help
```

Expected: Describes checks performed.

Result: ____

Notes:

---

#### 4.3 completion --help

Description: Shows completion command help.

Test:
```bash
./start completion --help
```

Expected: Lists shell options.

Result: ____

Notes:

---

## Issues Log

| ID | Feature | Description | Status | Fix | DR |
|----|---------|-------------|--------|-----|-----|

---

## Deliverables

- This project document with completed checklist
- All issues fixed and verified

---

## Notes

