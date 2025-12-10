# DR-006: Shell Configuration and Command Execution

- Date: 2025-12-02
- Status: Accepted
- Category: UTD

## Problem

UTD commands need to execute in various shells and interpreters (bash, sh, zsh, node, python, ruby, etc.). How should shell selection and command execution be configured?

Requirements:

- Support multiple shells and interpreters
- Allow per-command shell override
- Provide sensible defaults
- Give users full control over shell flags
- Control command execution timeout
- Unix-only (macOS, Linux - no Windows support)

## Decision

Two-level shell configuration with user-specified flags and auto-detection fallback.

**Global settings:**

```cue
settings: {
    shell:   "bash -c"  // Shell with flags
    timeout: 30         // Default timeout in seconds
}
```

**Per-section override:**

```cue
contexts: {
    "node-version": {
        command: "console.log(process.version)"
        shell:   "node -e"
        timeout: 5
        prompt:  "Node: {{.command_output}}"
    }
}
```

**Shell specification format:**

User provides shell binary and flags as single string:

- `"bash -c"` - Standard bash
- `"bash -euo pipefail -c"` - Bash strict mode
- `"sh -c"` - POSIX shell
- `"zsh -c"` - Z shell
- `"node -e"` - Node.js eval
- `"python3 -c"` - Python command
- `"ruby -e"` - Ruby eval

Implementation splits on whitespace: first element is binary, rest are flags.

**Resolution priority:**

1. Section-specific `shell` field (highest priority)
2. Global `settings.shell` setting
3. Auto-detected shell (bash if found in PATH, otherwise sh)

**Auto-detection:**

Only triggered when no shell configured:

1. Use `exec.LookPath("bash")` to check for bash in PATH
2. If found: use `"bash -c"`
3. If not found: use `exec.LookPath("sh")` to check for sh
4. If found: use `"sh -c"`
5. If neither found: error

**Timeout behavior:**

- Commands killed after timeout expires
- SIGTERM sent first, wait 1 second
- SIGKILL sent if still running
- Output captured up to timeout point
- Warning emitted
- Execution continues with partial output

## Why

**User-specified flags provide flexibility:**

Users control exact shell invocation:

- Standard mode: `bash -c`
- Strict mode: `bash -euo pipefail -c`
- Debugging: `sh -x -c`
- Unbuffered Python: `python3 -u -c`

No need for tool to know shell-specific flags.

**No magic mapping table:**

Tool doesn't need to know that:

- bash uses `-c` flag
- node uses `-e` flag
- deno uses `eval` command

Users specify what they need.

**Global default eliminates repetition:**

Set `shell: "bash -c"` once in settings, used everywhere. Override only when needed (node, python, etc.).

**Auto-detection provides good defaults:**

Works out-of-box without configuration. Most Unix systems have bash. Fallback to sh ensures something works.

**Two-level configuration balances convenience and flexibility:**

- Most commands use global default
- Special cases override per-section
- No need to specify shell for every command

**Timeout protection prevents hangs:**

Protects against infinite loops, slow commands, blocked I/O. Allows partial output to be useful. User can adjust timeout for slow operations.

## Trade-offs

Accept:

- Users must know shell flag syntax (`-c` vs `-e`)
- More verbose than implicit flag mapping
- Shell string parsing could fail on complex quoting
- Auto-detection adds startup check

Gain:

- Maximum flexibility (any flags, any shell)
- No hardcoded shell knowledge needed
- Future-proof (works with new shells)
- Simpler implementation (no mapping table)
- Explicit (users see exactly what runs)

## Alternatives

**Automatic flag mapping:**

Tool knows bash → `-c`, node → `-e`:

```cue
shell: "bash"  // Tool adds -c automatically
shell: "node"  // Tool adds -e automatically
```

- Pro: Less verbose, users don't need flag knowledge
- Con: Magic behavior, limited flexibility
- Con: Must maintain flag mapping table
- Con: Doesn't support custom flags
- Rejected: Too limiting, hides what's happening

**No global default:**

Require shell on every command:

```cue
contexts: {
    "status": {
        shell:   "bash -c"
        command: "git status"
    }
}
```

- Pro: Completely explicit
- Con: Extremely repetitive
- Con: Verbose for common case (bash)
- Rejected: Unnecessary repetition

**Shell detection from command:**

Detect shell from command syntax:

```cue
command: "console.log('hello')"  // Auto: JavaScript
command: "print('hello')"        // Auto: Python
```

- Pro: No shell field needed
- Con: Ambiguous (syntax overlaps)
- Con: Fragile, error-prone
- Con: Magic behavior
- Rejected: Too unreliable

**No timeout:**

- Pro: Simple, no limits
- Con: Can hang forever
- Con: No protection from infinite loops
- Rejected: Protection is essential

**Single binary name, separate flags field:**

```cue
shell:       "bash"
shell_flags: ["-c"]
```

- Pro: Structured data
- Con: More complex schema
- Con: Unnecessary separation
- Rejected: String format is simpler

## Implementation

**Command execution:**

1. Get shell string (section field → global setting → auto-detect)
2. Split shell string on whitespace: `strings.Fields(shell)`
3. First element is binary, rest are flags
4. Build command: `exec.Command(binary, flags..., userCommand)`
5. Set working directory (PWD or --directory flag)
6. Set timeout using context.WithTimeout
7. Execute command, capture stdout and stderr
8. Return output or error

**Shell string parsing:**

```
"bash -c"                → ["bash", "-c"]
"bash -euo pipefail -c"  → ["bash", "-euo", "pipefail", "-c"]
"python3 -u -c"          → ["python3", "-u", "-c"]
```

Simple space-split works for common cases. Complex quoting in shell field is not supported (use flags in command instead).

**Auto-detection:**

```
1. Check settings.shell → use if present
2. Check exec.LookPath("bash")
   - Found → use "bash -c"
3. Check exec.LookPath("sh")
   - Found → use "sh -c"
4. Neither found → error
```

## Working Directory

Commands execute in:

- Current working directory (PWD where `start` was invoked)
- Or directory specified by `--directory` (or `-d`) flag

Commands do NOT execute in:

- User's home directory
- Config file location
- Tool installation directory

Rationale: Users expect commands to run where they invoked `start`, matching standard CLI tool behavior (git, make, npm, etc.).

## Timeout Configuration

**Range:** 1 to 3600 seconds (1 second to 1 hour)

**Default:** 30 seconds (if not specified anywhere)

**Validation:**

- CUE schema enforces range: `int & >=1 & <=3600`
- Go runtime enforces timeout at execution

**Behavior on timeout:**

1. Send SIGTERM to process
2. Wait 1 second for graceful shutdown
3. If still running, send SIGKILL
4. Capture output produced before timeout
5. Emit warning: "Command timeout after N seconds"
6. Continue execution with partial output (if any)

**Resolution order:**

1. Section `timeout` field (highest priority)
2. Global `settings.timeout`
3. Hardcoded default: 30 seconds

## Security Considerations

**Command execution is powerful:**

- Commands run with user's permissions (no privilege escalation)
- Commands can read/write file system
- Commands can make network requests
- Commands inherit user's environment variables
- Commands can spawn other processes

**Protection mechanisms:**

- Timeout limits prevent infinite loops
- No automatic sudo or privilege escalation
- Commands run in working directory (not system paths)
- Shell configured by user (not auto-discovered from untrusted sources)

**User responsibilities:**

- Review commands before execution
- Understand command behavior
- Use trusted configurations
- Avoid sensitive data in command output
- Use `start show` to preview before executing

**Preview mode:**

The `start show` command resolves all UTD templates and displays final text without executing agent. Shows:

- Resolved shell and command
- File paths and preview of contents
- Commands that would execute
- Timeout values
- Final rendered prompt text

Users can verify behavior before actual execution.

## Unix Only

No Windows support. Design assumes:

- Unix-like shells (bash, sh, zsh, fish)
- Unix process model (fork/exec, signals)
- Unix PATH resolution
- Forward slashes in paths

Windows-specific shells (cmd.exe, PowerShell) are not supported.

## Examples

**Default bash:**

```cue
contexts: {
    "git-status": {
        command: "git status --short"
    }
}
```

Resolves to: `bash -c "git status --short"` (using global default or auto-detect)

**Bash strict mode:**

```cue
settings: {
    shell: "bash -euo pipefail -c"
}
```

All commands now run with strict error handling.

**Node.js interpreter:**

```cue
contexts: {
    "package-version": {
        shell:   "node -e"
        command: "console.log(require('./package.json').version)"
        prompt:  "Version: {{.command_output}}"
    }
}
```

Resolves to: `node -e "console.log(...)"`

**Python with custom timeout:**

```cue
contexts: {
    "python-analysis": {
        shell:   "python3 -c"
        timeout: 120
        command: """
import os
print(len([f for f in os.listdir('.') if f.endswith('.py')]))
"""
        prompt: "Python files: {{.command_output}}"
    }
}
```

Resolves to: `python3 -c "..."` with 120 second timeout

**Multi-line script:**

```cue
contexts: {
    "project-stats": {
        command: """
echo "Files: $(find . -type f | wc -l)"
echo "Lines: $(find . -name '*.go' -exec wc -l {} + | tail -1)"
"""
        timeout: 10
    }
}
```

Resolves to: `bash -c "multi-line script"` with 10 second timeout

**Custom shell with debugging:**

```cue
contexts: {
    "debug-script": {
        shell:   "sh -x -c"  // Trace execution
        command: "ls -la | grep .md"
    }
}
```

Resolves to: `sh -x -c "ls -la | grep .md"`

## Validation

**At configuration load:**

- Shell field must be non-empty string (if specified)
- Timeout must be integer 1-3600 (if specified)
- CUE schema validates types and ranges

**At execution time:**

- Shell binary (first word of shell string) must be in PATH
- If binary not found, emit error and skip command
- Timeout enforced by Go runtime context
- Command failure handled gracefully (warn + continue, or fail - see DR-007)

## Error Handling

**Shell binary not found:**

```
Error: shell 'nonexistent' not found in PATH
Skipping command execution
```

**Command timeout:**

```
Warning: command timeout after 30 seconds
Using partial output: "..."
```

**Command failed (non-zero exit):**

```
Warning: command exited with code 1
Stderr: "..."
```

Specific error handling behavior depends on UTD usage context (task/role/context).

## Updates

None yet.
