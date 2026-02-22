# Start CLI Command Tests

Manual testing reference for the start CLI.

## Help & Version

```bash
# Root help
start --help
start -h

# Version
start --version

# Subcommand help
start doctor --help
start completion --help
start config --help
start config agent --help
start config role --help
start config context --help
start config task --help
start config settings --help
start task --help
```

## Global Flags

```bash
# Verbose output (no short form for --verbose)
start --verbose --dry-run

# Debug output (implies verbose)
start --debug --dry-run

# Quiet mode
start --quiet --dry-run
start -q --dry-run

```

## Dry Run Mode

Dry run creates files in `.start/temp/` without executing the agent.

```bash
# Basic dry run
start --dry-run

# With agent selection
start --dry-run --agent claude
start --dry-run --agent gemini
start --dry-run --agent aichat
start --dry-run -a claude

# With model selection
start --dry-run --model sonnet
start --dry-run --model opus
start --dry-run --model flash
start --dry-run -m sonnet

# With role selection
start --dry-run --role golang-agent
start --dry-run -r golang-agent

# With context selection
start --dry-run --context environment
start --dry-run --context project
start --dry-run -c environment -c project

# Combined flags
start --dry-run --agent claude --model sonnet --role golang-agent
start --dry-run -a claude -m opus -r golang-agent -c environment
start --dry-run --verbose --agent claude --model sonnet
start --dry-run --debug --agent gemini --model flash
```

## Config Commands

### Agent Configuration

```bash
# List all agents
start config agent list

# Show specific agent info
start config agent info claude
start config agent info gemini
start config agent info aichat

# Set default agent
start config agent default claude
```

### Role Configuration

```bash
# List all roles
start config role list

# Show specific role info
start config role info golang-agent
start config role info golang-assistant

# Set default role
start config role default golang-agent
```

### Context Configuration

```bash
# List all contexts
start config context list

# Show specific context info
start config context info environment
start config context info project
```

### Task Configuration

```bash
# List all tasks
start config task list

# Show specific task info
start config task info code-review
start config task info debug
```

### Settings

```bash
# Show current settings
start config settings
```

## Task Execution

```bash
# Task dry run
start task code-review --dry-run
start task debug --dry-run

# Task with instructions
start task code-review --dry-run "focus on security"
start task debug --dry-run "investigate memory leak"

# Task with flags
start task code-review --dry-run --verbose
start task code-review --dry-run --agent claude
start task code-review --dry-run --model opus
start task debug --dry-run --agent gemini --model flash

# Task prefix matching (if unambiguous)
start task code --dry-run
start task deb --dry-run
```

## Doctor Command

```bash
# Run all checks
start doctor

# Verbose output
start doctor --verbose
start doctor -v
```

## Completion

```bash
# Generate shell completions (PowerShell not supported per DR-032)
start completion bash
start completion zsh
start completion fish

# Install completions (example for bash)
start completion bash > ~/.bash_completion.d/start

# Install completions (example for zsh)
start completion zsh > "${fpath[1]}/_start"
```

## Error Cases

Test that these produce appropriate errors:

```bash
# Unknown agent
start --dry-run --agent nonexistent

# Unknown task
start task nonexistent --dry-run

# Missing task name
start task --dry-run
```

## Live Execution

These execute real agents - ensure the agent is installed and configured.

```bash
# Execute with default agent
start

# Execute with specific agent
start --agent claude
start --agent gemini

# Execute with model override
start --agent claude --model opus

# Execute task
start task code-review "review the recent changes"
start task debug "fix the failing test"

# Execute with all options
start --agent claude --model sonnet --role golang-agent --context environment
```

## Temp File Verification

After dry run, check the generated files:

```bash
# Run dry run
start --dry-run

# Check temp directory
ls -la .start/temp/

# View generated files
cat .start/temp/role.md
cat .start/temp/prompt.md
cat .start/temp/command.txt
```

## Template Placeholder Testing

Verify placeholders are correctly escaped (no manual quotes needed):

```bash
# Check command output in dry run
start --dry-run --debug 2>&1 | grep "Final command"

# Verify single quotes around values
# Expected: 'value' not ''value'' (double-quoted)
```

## Context Selection Testing

```bash
# Required contexts only
start --dry-run

# Specific context by name
start --dry-run --context environment

# Multiple contexts
start --dry-run --context environment --context project

# Context with verbose to see which are included
start --dry-run --verbose
```

## Model Resolution Testing

```bash
# Friendly name (should resolve)
start --dry-run --agent claude --model sonnet
start --dry-run --agent claude --model opus

# Full model ID (should pass through)
start --dry-run --agent claude --model claude-sonnet-4-20250514

# Default model (no --model flag)
start --dry-run --agent claude
```
