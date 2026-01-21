# Multiple Agent Review Orchestration

## Goal

Create a prompt that instructs an AI agent to run all Go code review tasks as parallel background agents, with each agent saving its report to `.ai/reviews/yyyy-mm-dd-<type>.md`.

## Available Review Types

Found 13 review task types via `start assets search golang/review`:

| Type | Description |
|------|-------------|
| architecture | Deep review of Go system structure, design decisions, and component organisation |
| concurrency | Deep review identifying threading, parallelism, and asynchronous execution issues |
| correctness | Deep review verifying that code logic correctly implements the intended behaviour |
| dependency | Focused review of third-party package usage and dependency management |
| documentation | Review of external documentation, API documentation, and developer guides |
| duplication | Focused review to identify repeated code patterns that may benefit from consolidation |
| error-handling | Deep review of how failures are handled and whether edge cases are covered |
| holistic | Broad first-pass review of Go code covering all quality aspects at a high level |
| observability | Deep review assessing whether code can be understood and debugged in production |
| performance | Deep review analysing code efficiency and resource usage |
| readability | Focused review assessing whether code is clear and understandable to other developers |
| security | Deep review focused on identifying vulnerabilities, security weaknesses, and potential attack vectors |
| testing | Deep review of test quality, coverage, and production code testability |

## Command Pattern

```bash
# Search for available tasks
start assets search golang/review

# Run a specific review task
start task golang/review/<type>
```

## Observations from Test Run

Ran `start task golang/review/duplication` to understand the system.

### How `start task` Works

1. Launches a Claude agent (opus model) in the background
2. Provides context documents:
   - `repo/agents` → AGENTS.md (repository context)
   - `dotai/environment` → environment.md (local environment)
3. Assigns a role: `golang/assistant`
4. Provides a task prompt from `.start/temp/task-golang-review-<type>.md`
5. Output is buffered and written to `/private/tmp/claude/.../tasks/<id>.output`

### Task Prompt Structure

The task prompts (e.g., `task-golang-review-duplication.md`) contain:
- Task title and description
- Workflow steps
- Expected outcomes
- Detailed review criteria sections
- Tool recommendations (dupl, golangci-lint, jscpd)
- Key questions to consider

### Permission Issue Encountered

The background agent attempted to write to `.ai/reviews/` but encountered a pending permission:

```
I understand the permission is still pending. Let me wait for you to grant
the file write permission for the reviews directory.
```

The agent then output its summary to stdout instead. This is the key blocker for parallel background agents - they cannot receive interactive permission approvals.

### Test Output Summary

The duplication review completed successfully and found:
- 16 duplication issues (mostly acceptable CLI boilerplate)
- Epic/Sprint add commands share 116 lines (acceptable)
- Test patterns duplicated (acceptable - clarity over DRY)
- JSON output pattern could use `PrintSuccessJSON()` more consistently
- **Conclusion:** No immediate refactoring recommended

## Challenges for Multi-Agent Orchestration

1. **Permission blocking**: Background agents can't get interactive permission approval for file writes
2. **Output collection**: Need to collect reports from 13 parallel agents
3. **File naming**: Must dynamically determine today's date for filenames
4. **Directory preparation**: `.ai/reviews/` must exist before agents write

## Potential Solutions

### Option 1: Pre-approve Permissions
- Ensure `.ai/reviews/` directory exists
- Pre-approve file write permissions for that directory before launching agents

### Option 2: Permissive Mode
- Use `--dangerously-skip-permissions` or similar flag if supported
- Appropriate for trusted local operations

### Option 3: Post-collection
- Agents output to stdout (as the test did when blocked)
- Orchestrating agent collects all outputs after completion
- Writes files to `.ai/reviews/` centrally

### Option 4: Sequential Execution
- Run one agent at a time
- Each gets permission approval
- Less efficient but avoids permission issues

## Next Steps

1. Investigate `start` program's permission handling options
2. Determine if there's a way to pre-approve directory write access
3. Design the orchestration prompt based on chosen solution
4. Test with 2-3 agents before scaling to all 13

## Files of Interest

- `.start/temp/` - Contains generated task prompts
- `.start/temp/role-golang-assistant.md` - Role definition
- `.start/temp/context-repo-agents.md` - Context document
