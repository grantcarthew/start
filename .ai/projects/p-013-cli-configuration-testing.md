# p-013: CLI Configuration Commands Testing

- Status: Proposed
- Started: -

## Overview

End-to-end testing of the `start` CLI configuration management commands and config merging behavior. This project covers the `config` command hierarchy, the `show` command, and global/local configuration merging.

Part 2 of 3 in CLI testing series:
- p-012: Core Commands
- p-013: Configuration Commands (this project)
- p-014: Supporting Commands (Assets, Doctor, Completion)

## Goals

1. Test all config subcommands (agent, role, context, task, settings)
2. Test all CRUD actions (list, add, info, edit, remove, default)
3. Test show command for inspecting resolved configuration
4. Test global/local configuration merging
5. Fix all issues discovered during testing

## Scope

In Scope:
- `start config agent` (list, add, info, edit, remove, default)
- `start config role` (list, add, info, edit, remove, default)
- `start config context` (list, add, info, edit, remove)
- `start config task` (list, add, info, edit, remove)
- `start config settings` (info, edit)
- `start show` (agent, role, context, task)
- `--local` flag for config commands
- `--scope` flag for show command
- Configuration merging (global + local)

Out of Scope:
- Core execution commands (p-012)
- Assets, Doctor, Completion (p-014)

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

### 1. Config Agent Commands

#### 1.1 config agent list

Description: List all configured agents with default indicator.

Test:
```bash
./start config agent list
```

Expected: Lists agents with names, shows which is default.

Result: PASS

Notes: Shows agents with * indicator for default, includes (global) source.

---

#### 1.2 config agent add (Flags)

Description: Add a new agent using flags.

Test:
```bash
./start config agent add --name test-agent --bin echo --command "{role}"
./start config agent list
./start config agent remove test-agent
```

Expected: Agent added, appears in list, then removed.

Result: PASS

Notes: Agent added successfully, appeared in list, removed with --yes flag.

---

#### 1.3 config agent add (Interactive)

Description: Add agent with interactive prompts for missing fields.

Test:
```bash
./start config agent add
# Follow prompts
```

Expected: Prompts for required fields (name, bin, command).

Result: SKIP

Notes: Requires TTY for interactive prompts.

---

#### 1.4 config agent info

Description: Show detailed agent information.

Test:
```bash
./start config agent info <agent-name>
```

Expected: Shows all agent fields (bin, command, models, default_model, description).

Result: PASS

Notes: Shows Source, Bin, Command, Default Model, Description, and Models list.

---

#### 1.5 config agent edit (With Name)

Description: Edit existing agent interactively.

Test:
```bash
./start config agent edit <agent-name>
# Prompts show current values
```

Expected: Shows current values, allows changes.

Result: SKIP

Notes: Requires TTY for interactive prompts.

---

#### 1.6 config agent edit (No Name - $EDITOR)

Description: Edit agents file in $EDITOR.

Test:
```bash
EDITOR=cat ./start config agent edit
```

Expected: Opens agents.cue file content.

Result: PASS

Notes: Opens agents.cue content in $EDITOR with full CUE structure.

---

#### 1.7 config agent edit (Flags)

Description: Edit agent using flags (non-interactive).

Test:
```bash
./start config agent add --name temp --bin temp --command "temp"
./start config agent edit temp --default-model "new-model"
./start config agent info temp
./start config agent remove temp
```

Expected: Agent updated, change visible in info.

Result: FAIL

Notes: Flag-based editing not implemented. Edit command only supports interactive prompts or $EDITOR mode. Created p-017 to implement this feature.

---

#### 1.8 config agent default (Show)

Description: Show current default agent.

Test:
```bash
./start config agent default
```

Expected: Shows name of default agent.

Result: ____

Notes:

---

#### 1.9 config agent default (Set)

Description: Set default agent.

Test:
```bash
./start config agent default <agent-name>
./start config agent default
```

Expected: Sets new default, shown on subsequent query.

Result: ____

Notes:

---

#### 1.10 config agent remove

Description: Remove an agent with confirmation.

Test:
```bash
./start config agent add --name temp --bin temp --command "temp"
./start config agent remove temp
./start config agent list
```

Expected: Prompts for confirmation, removes agent.

Result: ____

Notes:

---

#### 1.11 config agent aliases

Description: Plural alias works (agents = agent).

Test:
```bash
./start config agents list
```

Expected: Same as `config agent list`.

Result: ____

Notes:

---

### 2. Config Role Commands

#### 2.1 config role list

Description: List all configured roles.

Test:
```bash
./start config role list
```

Expected: Lists roles with names.

Result: ____

Notes:

---

#### 2.2 config role add (Flags)

Description: Add a new role using flags.

Test:
```bash
./start config role add --name test-role --content "You are a test assistant"
./start config role list
./start config role remove test-role
```

Expected: Role added, appears in list, then removed.

Result: ____

Notes:

---

#### 2.3 config role add (File)

Description: Add role with file reference.

Test:
```bash
echo "# Test Role" > /tmp/test-role.md
./start config role add --name file-role --file /tmp/test-role.md
./start config role info file-role
./start config role remove file-role
```

Expected: Role references file path.

Result: ____

Notes:

---

#### 2.4 config role info

Description: Show detailed role information.

Test:
```bash
./start config role info <role-name>
```

Expected: Shows role content or file reference.

Result: ____

Notes:

---

#### 2.5 config role edit

Description: Edit existing role.

Test:
```bash
./start config role add --name temp-role --content "Original"
./start config role edit temp-role --content "Updated"
./start config role info temp-role
./start config role remove temp-role
```

Expected: Role updated with new content.

Result: ____

Notes:

---

#### 2.6 config role default

Description: Set/show default role.

Test:
```bash
./start config role default
./start config role default <role-name>
./start config role default
```

Expected: Shows and sets default role.

Result: ____

Notes:

---

#### 2.7 config role remove

Description: Remove a role.

Test:
```bash
./start config role add --name temp --content "temp"
./start config role remove temp
```

Expected: Role removed.

Result: ____

Notes:

---

### 3. Config Context Commands

#### 3.1 config context list

Description: List all configured contexts.

Test:
```bash
./start config context list
```

Expected: Lists contexts with names, required/default/tags indicators.

Result: ____

Notes:

---

#### 3.2 config context add (File)

Description: Add context with file reference.

Test:
```bash
./start config context add --name test-ctx --file "/tmp/test.md"
./start config context list
./start config context remove test-ctx
```

Expected: Context added with file reference.

Result: ____

Notes:

---

#### 3.3 config context add (Command)

Description: Add context with command.

Test:
```bash
./start config context add --name cmd-ctx --command "echo hello"
./start config context info cmd-ctx
./start config context remove cmd-ctx
```

Expected: Context added with command.

Result: ____

Notes:

---

#### 3.4 config context add (Required/Default/Tags)

Description: Add context with selection fields.

Test:
```bash
./start config context add --name tagged-ctx --file "/tmp/test.md" --tags "test,example" --default
./start config context info tagged-ctx
./start config context remove tagged-ctx
```

Expected: Context has tags and default flag set.

Result: ____

Notes:

---

#### 3.5 config context info

Description: Show detailed context information.

Test:
```bash
./start config context info <context-name>
```

Expected: Shows file/command, tags, required/default flags.

Result: ____

Notes:

---

#### 3.6 config context edit

Description: Edit existing context.

Test:
```bash
./start config context add --name temp --file "/tmp/a.md"
./start config context edit temp --file "/tmp/b.md"
./start config context info temp
./start config context remove temp
```

Expected: Context file updated.

Result: ____

Notes:

---

#### 3.7 config context remove

Description: Remove a context.

Test:
```bash
./start config context add --name temp --file "/tmp/test.md"
./start config context remove temp
```

Expected: Context removed.

Result: ____

Notes:

---

### 4. Config Task Commands

#### 4.1 config task list

Description: List all configured tasks.

Test:
```bash
./start config task list
```

Expected: Lists tasks with names.

Result: ____

Notes:

---

#### 4.2 config task add

Description: Add a new task.

Test:
```bash
./start config task add --name test-task --prompt "Do something"
./start config task list
./start config task remove test-task
```

Expected: Task added and removed.

Result: ____

Notes:

---

#### 4.3 config task add (With Role)

Description: Add task with role reference.

Test:
```bash
./start config task add --name role-task --prompt "Test" --role <role-name>
./start config task info role-task
./start config task remove role-task
```

Expected: Task references specified role.

Result: ____

Notes:

---

#### 4.4 config task add (With Command)

Description: Add task with command for dynamic content.

Test:
```bash
./start config task add --name cmd-task --command "git status" --prompt "Review: {{.command_output}}"
./start config task info cmd-task
./start config task remove cmd-task
```

Expected: Task has command field.

Result: ____

Notes:

---

#### 4.5 config task info

Description: Show detailed task information.

Test:
```bash
./start config task info <task-name>
```

Expected: Shows prompt, role, command, file fields.

Result: ____

Notes:

---

#### 4.6 config task edit

Description: Edit existing task.

Test:
```bash
./start config task add --name temp --prompt "Original"
./start config task edit temp --prompt "Updated"
./start config task info temp
./start config task remove temp
```

Expected: Task updated.

Result: ____

Notes:

---

#### 4.7 config task remove

Description: Remove a task.

Test:
```bash
./start config task add --name temp --prompt "temp"
./start config task remove temp
```

Expected: Task removed.

Result: ____

Notes:

---

### 5. Config Settings Commands

#### 5.1 config settings info

Description: Show current settings.

Test:
```bash
./start config settings info
```

Expected: Shows default_agent, default_role, other settings.

Result: ____

Notes:

---

#### 5.2 config settings edit

Description: Edit settings.

Test:
```bash
./start config settings edit --default-agent <agent>
./start config settings info
```

Expected: Setting updated.

Result: ____

Notes:

---

### 6. Show Commands

#### 6.1 show agent (List)

Description: Show agent list.

Test:
```bash
./start show agent
```

Expected: Lists all agents.

Result: ____

Notes:

---

#### 6.2 show agent (Named)

Description: Show specific agent details.

Test:
```bash
./start show agent <agent-name>
```

Expected: Shows agent configuration.

Result: ____

Notes:

---

#### 6.3 show role (List)

Description: Show role list.

Test:
```bash
./start show role
```

Expected: Lists all roles.

Result: ____

Notes:

---

#### 6.4 show role (Named with UTD)

Description: Show resolved role content after UTD processing.

Test:
```bash
./start show role <role-name>
```

Expected: Shows processed role content (templates resolved).

Result: ____

Notes:

---

#### 6.5 show context (List)

Description: Show context list.

Test:
```bash
./start show context
```

Expected: Lists all contexts with tags/required/default.

Result: ____

Notes:

---

#### 6.6 show context (Named with UTD)

Description: Show resolved context content.

Test:
```bash
./start show context <context-name>
```

Expected: Shows processed context (file read, command executed).

Result: ____

Notes:

---

#### 6.7 show task (List)

Description: Show task list.

Test:
```bash
./start show task
```

Expected: Lists all tasks.

Result: ____

Notes:

---

#### 6.8 show task (Named)

Description: Show task template.

Test:
```bash
./start show task <task-name>
```

Expected: Shows task prompt template.

Result: ____

Notes:

---

#### 6.9 show --scope global

Description: Show from global config only.

Test:
```bash
./start show agent --scope global
```

Expected: Shows only global config (ignores local).

Result: ____

Notes:

---

#### 6.10 show --scope local

Description: Show from local config only.

Test:
```bash
mkdir -p ./.start
./start config agent add --name local-test --bin local --command "local" --local
./start show agent --scope local
./start config agent remove local-test --local
rmdir ./.start 2>/dev/null || rm -rf ./.start
```

Expected: Shows only local config.

Result: ____

Notes:

---

### 7. Local Config Flag

#### 7.1 --local Flag (Short -l)

Description: -l short form works.

Test:
```bash
./start config agent list -l
```

Expected: Same as --local.

Result: ____

Notes:

---

#### 7.2 --local Creates Directory

Description: --local creates .start/ if needed.

Test:
```bash
rm -rf ./.start
./start config agent add --name local-agent --bin local --command "cmd" --local
ls ./.start/
./start config agent remove local-agent --local
rm -rf ./.start
```

Expected: Creates .start/ directory with agents.cue.

Result: ____

Notes:

---

#### 7.3 --local Scope Isolation

Description: Local config doesn't affect global.

Test:
```bash
./start config agent add --name unique-local --bin local --command "cmd" --local
./start config agent list  # Should show both
./start show agent --scope global  # Should not show unique-local
./start config agent remove unique-local --local
rm -rf ./.start
```

Expected: Local agent visible in merged, not in global scope.

Result: ____

Notes:

---

### 8. Configuration Merging

#### 8.1 Global Only

Description: Works with only global config.

Test:
```bash
rm -rf ./.start
./start show agent
```

Expected: Shows global config.

Result: ____

Notes:

---

#### 8.2 Local Only

Description: Works with only local config.

Test:
```bash
mv ~/.config/start ~/.config/start.bak
mkdir -p ./.start
echo 'agents: { local: { bin: "echo", command: "{role}" } }' > ./.start/config.cue
./start show agent
rm -rf ./.start
mv ~/.config/start.bak ~/.config/start
```

Expected: Shows local config only.

Result: ____

Notes:

---

#### 8.3 Merged (Both)

Description: Local overrides global for same names.

Test:
```bash
mkdir -p ./.start
./start config agent add --name test-merge --bin global --command "global"
./start config agent add --name test-merge --bin local --command "local" --local
./start show agent test-merge  # Should show local version
./start config agent remove test-merge
./start config agent remove test-merge --local
rm -rf ./.start
```

Expected: Local version takes precedence.

Result: ____

Notes:

---

#### 8.4 Additive Merge

Description: Different names from global and local both appear.

Test:
```bash
./start config agent add --name global-only --bin global --command "cmd"
./start config agent add --name local-only --bin local --command "cmd" --local
./start config agent list  # Should show both
./start config agent remove global-only
./start config agent remove local-only --local
rm -rf ./.start
```

Expected: Both agents visible in merged view.

Result: ____

Notes:

---

## Issues Log

| ID | Feature | Description | Status | Fix | DR |
|----|---------|-------------|--------|-----|-----|
| 1 | 1.7 | Flag-based editing not implemented for edit commands | Open | p-017 | - |

---

## Deliverables

- This project document with completed checklist
- All issues fixed and verified

---

## Notes

