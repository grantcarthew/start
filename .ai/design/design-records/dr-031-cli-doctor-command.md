# dr-031: CLI Doctor Command

- Date: 2025-12-19
- Status: Accepted
- Category: CLI

## Problem

Users need a way to diagnose issues with their `start` installation and configuration. Without a diagnostic tool, troubleshooting requires manual inspection of config files, PATH checking, and file existence verification. This creates friction when something goes wrong.

## Decision

Implement a `start doctor` command that performs comprehensive health checks and reports issues with clear pass/fail status and actionable fix suggestions.

## Why

Single diagnostic entry point:

- One command to check everything
- No need to manually inspect config files
- Clear reporting of what's working and what's broken

Actionable output:

- Issues include fix suggestions
- Users know exactly what to do
- Reduces support burden

CI/CD friendly:

- Exit code 0 for healthy, 1 for issues
- Quiet mode for automation
- Can gate deployments on health status

## Checks

The doctor command performs these checks in order:

### 1. Intro (informational)

Display repository information for support and issue reporting:

- Repository URL: `https://github.com/grantcarthew/start`
- Issues URL: `https://github.com/grantcarthew/start/issues`

### 2. Version Information (informational)

Display build metadata:

- CLI version
- Git commit hash
- Build date
- Go version
- Platform (OS/arch)

No network check for latest version (explicit user action via separate command if implemented later).

### 3. Configuration Validation

Check CUE configuration files:

- Global config directory exists (`~/.config/start/`)
- Local config directory exists (`./.start/`)
- CUE files parse without syntax errors
- Merged configuration is valid
- Report specific parse errors with file and line number

Severity:

- Error: CUE syntax errors, invalid configuration
- Info: Directory not found (may be intentional)

### 4. Agent Checks

For each configured agent:

- Check if `bin` exists in PATH using `exec.LookPath()`
- Report full path to found binaries
- Report "NOT FOUND" for missing binaries

Severity:

- Error: Configured agent binary not found

### 5. Context File Checks

For each configured context with a `file` field:

- Verify the referenced file exists
- Handle `~` expansion and relative paths
- Check file is readable

Severity:

- Error: Required context file missing
- Warning: Optional context file missing (if schema distinguishes)

### 6. Role File Checks

For each configured role with a `file` field:

- Verify the referenced file exists
- Handle path expansion

Severity:

- Error: Role file missing

### 7. Environment Checks

Verify runtime environment:

- Config directory writable (if exists)
- Working directory accessible

Severity:

- Error: Directory not writable/accessible

## Exit Codes

Binary exit codes following prototype dr-024 pattern:

- `0`: Healthy (no errors or warnings)
- `1`: Issues found (any error or warning)

Exit code does not distinguish error severity. Output categorises issues; users read output for details.

## Output Format

Follow prototype format with ✓/✗/⚠ indicators:

### Normal Mode

```
start doctor
═══════════════════════════════════════════════════════════

Repository
  https://github.com/grantcarthew/start
  Issues: https://github.com/grantcarthew/start/issues

Version
  start v0.1.0
  Commit:    abc1234
  Built:     2025-12-19T10:30:00Z
  Go:        go1.23.0
  Platform:  darwin/arm64

Configuration
  Global (~/.config/start/):
    ✓ agents.cue
    ✓ settings.cue
  Local (./.start/):
    - Not found

Agents (2 configured)
  ✓ claude - /usr/local/bin/claude
  ✗ gemini - NOT FOUND
    Install: https://github.com/google/generative-ai-cli

Contexts (2 configured)
  ✓ environment - ~/context/environment.md
  ✗ project - ./PROJECT.md (not found)

Roles (1 configured)
  ✓ go-expert - ~/.config/start/roles/go-expert.md

Environment
  ✓ Config directory writable
  ✓ Working directory accessible

Summary
───────────────────────────────────────────────────────────
  1 error, 1 warning found

Issues:
  ✗ Agent 'gemini' binary not found
  ⚠ Context 'project' file missing

Recommendations:
  1. Install gemini CLI or remove from config
  2. Create PROJECT.md or remove context
```

### Quiet Mode

```bash
start doctor --quiet
```

No output on success (exit 0). Only issues on failure:

```
Error: Agent 'gemini' binary not found
Warning: Context 'project' file missing
```

### Verbose Mode

```bash
start doctor --verbose
```

Adds detailed information:

- Full paths for all files
- File sizes and modification times
- Detailed CUE validation output

## Flags

Standard global flags apply:

- `--quiet` / `-q`: Suppress output, only exit code
- `--verbose`: Detailed output

No doctor-specific flags initially.

## Trade-offs

Accept:

- No automatic fixing (suggestions only)
- No network checks (version, connectivity)
- No agent-specific health (API keys, quotas)
- Binary exit code loses severity granularity

Gain:

- Simple, focused diagnostic tool
- Fast execution (no network)
- Clear actionable output
- CI/CD compatible
- Matches user mental model of "doctor" command

## Alternatives

Integrate into other commands:

- Pro: No new command to learn
- Con: Mixes concerns (e.g., `start config validate` vs full health)
- Rejected: Doctor is a distinct use case

Multiple exit codes by severity:

- Pro: Scripts can distinguish warnings from errors
- Con: More complex, users must memorise codes
- Rejected: Binary is simpler, output has details

Automatic fixing with `--fix`:

- Pro: One-step resolution
- Con: Risky without user review, scope creep
- Rejected: Out of scope, suggestions are safer
