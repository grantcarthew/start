# dr-007: UTD Error Handling by Context

- Date: 2025-12-02
- Status: Accepted
- Category: UTD

## Problem

UTD template resolution can fail in various ways (file missing, command fails, template syntax error). How should errors be handled?

The answer depends on WHERE the UTD is used:

- Tasks are user-initiated actions (user expects them to work)
- Contexts are optional background information (nice-to-have)
- Roles are optional system prompts (agent can run without them)

Different contexts need different error handling strategies.

## Decision

Error handling depends on the parent entity using UTD.

**Tasks: Fail hard**

- Any UTD error stops execution immediately
- Exit with code 1
- Display error message to stdout
- User must fix configuration

**Contexts: Warn and skip**

- Log warning to stdout with details
- Skip this context entirely
- Continue processing other contexts
- Session proceeds with available contexts

**Roles: Depends on optional flag (see dr-039)**

- Required roles (optional: false, default): Error and stop execution
- Optional roles (optional: true): Skip and try next role in definition order
- All roles exhausted: Error with "no valid roles found"
- Explicit --role flag: Always error if resolution fails

**General principle:** Fail only when critical (tasks), otherwise warn and continue (best effort).

## Why

**Tasks are user-initiated:**

User runs `start task code-review` expecting it to work. If task configuration is broken, failing immediately is correct:

- Clear feedback (task is broken)
- Prevents wasted time (agent running wrong task)
- Forces fix (can't proceed with broken task)

**Contexts are optional enrichment:**

Contexts provide background information. If one fails:

- Other contexts still valuable
- Agent can still run effectively
- Missing context reduces quality but doesn't prevent work
- Best effort: use what works

**Roles have configurable error handling (dr-039):**

Roles customize agent behavior. Error handling depends on the `optional` field:

- Required roles (default): Must resolve successfully or error
- Optional roles: Skip gracefully, try next role in chain
- Explicit selection (--role flag): Always error if broken
- All roles exhausted: Error (no silent "no role" state)

**Best effort maximizes utility:**

Multiple contexts, one fails:

- Warn about failed context
- Use remaining contexts
- Better than failing completely
- User gets work done, can fix later

## Error Types and Handling

### File Errors

**File not found:**

Task:

```
Error: file not found: ./PROMPT.md
Task 'code-review' failed
```

Exit code 1

Context/Role:

```
Warning: file not found: ~/reference/MISSING.md
Skipping context 'documentation'
```

Continue

**File not readable:**

Task:

```
Error: cannot read file: ./prompt.md (permission denied)
Task 'code-review' failed
```

Exit code 1

Context/Role:

```
Warning: cannot read file: ./PROJECT.md (permission denied)
Skipping context 'project'
```

Continue

### Command Errors

**Shell not found:**

Task:

```
Error: shell 'nonexistent' not found in PATH
Task 'analyze' failed
```

Exit code 1

Context/Role:

```
Warning: shell 'badshell' not found in PATH
Skipping context 'metrics'
```

Continue

**Command exits non-zero:**

Task:

```
Error: command failed with exit code 1
Command: git diff --staged
Stderr: fatal: not a git repository
Task 'code-review' failed
```

Exit code 1

Context/Role:

```
Warning: command failed with exit code 1
Command: git status
Stderr: fatal: not a git repository
Skipping context 'git-status'
```

Continue

**Command timeout:**

Task:

```
Error: command timeout after 30 seconds
Command: slow-analysis.sh
Partial output: "Processing..."
Task 'analyze' failed
```

Exit code 1

Context/Role:

```
Warning: command timeout after 30 seconds
Command: long-running-script
Using partial output: "..."
```

Continue with partial output

### Template Errors

**Invalid template syntax:**

Task:

```
Error: invalid template syntax
Template: "File: {{.file_contents"
Error: template: utd:1: unclosed action
Task 'review' failed
```

Exit code 1

Context/Role:

```
Warning: invalid template syntax
Template: "Status: {{.command_output"
Error: template: utd:1: unclosed action
Skipping context 'status'
```

Continue

**Unknown placeholder:**

Task:

```
Error: unknown placeholder in template
Template: "Output: {{.commnd_output}}"
Error: can't evaluate field commnd_output
Task 'check' failed
```

Exit code 1

Context/Role:

```
Warning: unknown placeholder in template
Template: "Data: {{.unknown}}"
Error: can't evaluate field unknown
Skipping role 'custom'
```

Continue

### Unused Field Warnings

**File defined but not used:**

All contexts (not an error, just a warning):

```
Warning: file defined but not referenced in template
File: ./unused.md
Template does not contain {{.file}} or {{.file_contents}}
```

Continue (use template as-is, ignore file)

**Command defined but not used:**

All contexts (not an error, just a warning):

```
Warning: command defined but not referenced in template
Command: git status
Template does not contain {{.command}} or {{.command_output}}
```

Continue (use template as-is, ignore command)

## Partial Failures

**File works, command fails:**

Task:

```
Error: command failed while resolving template
File read successfully: ./PROJECT.md
Command failed: git log (exit code 128)
Task 'report' failed
```

Exit code 1

Context/Role:

```
Warning: command failed while resolving template
File read successfully: ./PROJECT.md
Command failed: git log (exit code 128)
Using file contents only
```

Continue with file contents, empty command output

**Command works, file fails:**

Task:

```
Error: file not found while resolving template
Command succeeded: git status
File not found: ./missing.md
Task 'status' failed
```

Exit code 1

Context/Role:

```
Warning: file not found while resolving template
Command succeeded: git status
File not found: ./missing.md
Using command output only
```

Continue with command output, empty file contents

## Multiple Context Failures

**Scenario:** 5 contexts configured, 2 fail

Output:

```
Warning: context 'git-status' skipped (command failed)
Warning: context 'missing-doc' skipped (file not found)

Loaded contexts (3):
  - environment
  - index
  - agents
```

Session continues with 3 working contexts.

**Scenario:** All contexts fail

Output:

```
Warning: context 'env' skipped (file not found)
Warning: context 'project' skipped (command failed)
Warning: context 'git' skipped (shell not found)

No contexts loaded
Proceeding without context documents
```

Session continues without contexts (agent still works).

## Message Format

**Error messages (tasks):**

```
Error: <what-happened>
<relevant-details>
Task '<task-name>' failed
```

**Warning messages (contexts/roles):**

```
Warning: <what-happened>
<relevant-details>
Skipping <entity-type> '<entity-name>'
```

**Details include:**

- File paths (if relevant)
- Commands (if relevant)
- Exit codes (if relevant)
- Error output (stderr, exception messages)
- Partial data captured (if any)

**Be specific and actionable:**

Bad: `Error: file error`
Good: `Error: file not found: ./PROMPT.md`

Bad: `Warning: command problem`
Good: `Warning: command failed with exit code 1 (git diff)`

## Trade-offs

Accept:

- More complex error handling logic (parent-dependent)
- Multiple code paths for same error type
- Users must understand fail vs warn distinction
- Partial failures can be confusing

Gain:

- Tasks fail fast (user gets immediate feedback)
- Contexts are best-effort (maximize utility)
- Clear distinction between critical and optional
- Better user experience (work continues when possible)

## Alternatives

**Always fail:**

Any error stops execution:

- Pro: Simpler logic, single code path
- Con: Broken context stops entire session
- Con: Missing optional file fails everything
- Rejected: Too brittle, poor user experience

**Always warn:**

No errors stop execution:

- Pro: Maximum resilience
- Con: Broken tasks proceed silently
- Con: User wastes time on wrong task
- Con: No clear signal of critical failure
- Rejected: Tasks must fail when broken

**Strict mode flag:**

All errors fail, warnings disabled:

- Pro: Users can choose behavior
- Con: Adds complexity (two modes)
- Con: Users must remember to use flag
- Rejected: Parent-based handling is clearer

**Silent failures:**

Skip failures without warnings:

- Pro: Clean output
- Con: Hidden problems
- Con: User unaware of missing data
- Rejected: Warnings are valuable feedback

## Best Effort Philosophy

The tool tries to maximize utility:

**Use what works:**

- 5 contexts, 2 fail → use 3
- File fails, command works → use command output
- Command times out → use partial output

**Inform the user:**

- Clear warnings about what failed
- Explain why it failed
- Show what's being used instead

**Fail only when critical:**

- Task is broken → fail (user-initiated)
- Context is broken → skip (optional enrichment)
- Required role is broken → fail (must resolve)
- Optional role is broken → skip (try next role)

**Principle:** Try to run if we can, fail only when we must.

## Examples

**Task with file error:**

```
Error: file not found: ./review-prompt.md
Task 'code-review' failed

Fix: Create ./review-prompt.md or update task configuration
```

Exit code: 1

**Context with command error:**

```
Warning: command failed with exit code 128
Command: git log -5
Stderr: fatal: not a git repository
Skipping context 'recent-commits'

Loaded contexts (2):
  - environment
  - index
```

Continues with 2 contexts

**Optional role with template error:**

```
Role:
  custom-reviewer  ○  skipped
  fallback-role    ✓  fallback.md
```

Optional role skipped, next role used

**Required role with template error:**

```
Role:
  custom-reviewer  ○  template error

Error: role "custom-reviewer": invalid template syntax
```

Execution stops with error

**Partial failure in context:**

```
Warning: command timeout after 30 seconds
Command: npm run long-analysis
Using partial output (42 bytes)
Context 'analysis' loaded with incomplete data
```

Context loaded with whatever data available

## Validation

**At configuration load:**

Validate structure only (CUE schema):

- Field types correct
- Ranges valid (timeout 1-3600)
- Required fields present

Do NOT validate at load time:

- Files exist (may be created later)
- Commands work (may depend on runtime state)
- Templates valid (validated at execution)

**At execution time:**

Validate all runtime aspects:

- File existence and readability
- Shell binary in PATH
- Command execution
- Template parsing and execution
- Placeholder resolution

## Updates

- 2026-01-31: Role error handling updated by dr-039 (optional field). Optional roles skip on error, required roles now error instead of warn.
