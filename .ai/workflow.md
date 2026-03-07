# Feature Development Workflow

An interactive, step-by-step workflow for developing features collaboratively with AI agents.

## Principles

- Interactive: Each step requires user involvement before proceeding
- Collaborative: Agent presents findings and options; user decides
- Flexible: User can skip steps or revisit earlier ones
- Conversational: This is a dialogue, not autonomous execution

## When to Use

Use this workflow for non-trivial features requiring design decisions. Skip it for simple bug fixes, typos, or trivial changes.

## The Workflow

### 1. Identify the Feature

Read AGENTS.md to find the active project. Read the project document to identify the next feature to implement.

Present the feature scope to the user for confirmation.

### 2. Read Required Documentation

Read all relevant documentation:

- Project document (goals, scope, success criteria)
- Relevant standards in `.ai/design/`
- Existing code that will be modified

Summarise findings for the user.

### 3. Discuss Options

Have an interactive discussion about:

- Implementation approaches
- Trade-offs between options
- Questions that need answers

If decisions cannot be made now, add them to the project's Decision Points section for later resolution.

Do not proceed until the user is satisfied with the direction.

### 4. Implement

Implement the feature:

- Write the code
- Write tests (unit, integration, e2e as appropriate)
- Follow existing code patterns

Present implementation to user for feedback.

### 5. Test and Fix

Run the test suite. Fix any failures.

Continue until tests pass and implementation is solid.

### 6. Code Review (Optional)

If the changes are significant, get an external code review.

Fix any reported issues.

### 7. Update Project

Update the project document:

- Mark completed items in success criteria
- Add progress notes
- Identify next feature

If all success criteria are met:

- Move project to `.ai/projects/completed/`
- Update `.ai/projects/README.md` with completion date
- Update `AGENTS.md` to set next project as active

### 8. Repeat

Return to step 1 for the next feature.

## Step Flexibility

Users may:

- Combine steps when appropriate
- Revisit earlier steps if new information emerges
- End the session at any step and resume later

Always confirm with the user before proceeding to the next step.
