# start

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/grantcarthew/start)](https://goreportcard.com/report/github.com/grantcarthew/start)
[![Go Reference](https://pkg.go.dev/badge/github.com/grantcarthew/start.svg)](https://pkg.go.dev/github.com/grantcarthew/start)
[![GitHub Release](https://img.shields.io/github/v/release/grantcarthew/start)](https://github.com/grantcarthew/start/releases)

Context-aware AI agent launcher powered by CUE.

## Why start?

**Stop re-explaining yourself to every AI session.**

Every time you open an AI coding session you provide the same background: what the project does, what role the agent should play, what you're working on today. start eliminates this by composing intelligent prompts from your project's context files and launching your configured AI agent — every time, consistently, with zero ceremony.

- **Role-based sessions** - Define agent expertise once, reuse across projects (`go-expert`, `code-reviewer`, `security-auditor`)
- **Reusable tasks** - Package common workflows as shareable prompts (`pre-commit-review`, `debug-help`, `write-docs`)
- **Automatic context injection** - Project files, environment info, and documentation included without manual setup
- **Multi-agent support** - Works with Claude, Gemini, aichat, opencode, or any AI CLI tool
- **CUE-powered configuration** - Type-safe, validated, order-preserving config with built-in schema enforcement
- **Registry packages** - Install curated roles, tasks, and contexts from the CUE Central Registry

**Perfect for:**

- Developers who run AI coding sessions daily and want consistent context
- Teams sharing prompt engineering patterns across projects
- Anyone tired of repeating themselves at the start of every session

## Quick Start

```bash
# Install
brew tap grantcarthew/tap
brew install grantcarthew/tap/start

# Auto-setup detects your installed AI agent and writes initial config
start doctor

# Launch an AI session with full project context
start

# Use a specific role
start --role go-expert

# Run a reusable task
start task pre-commit-review

# Send a one-off prompt (minimal context, focused output)
start prompt "Explain this error message"
```

## Installation

### Homebrew (Linux/macOS)

```bash
brew tap grantcarthew/tap
brew install grantcarthew/tap/start
```

### Go Install

```bash
go install github.com/grantcarthew/start/cmd/start@latest
```

### Build from Source

```bash
git clone https://github.com/grantcarthew/start.git
cd start
go build ./cmd/start
./start --version
```

## How It Works

start is built around four concepts: **agents**, **roles**, **tasks**, and **contexts**. These are all defined in CUE and distributed as packages through the CUE Central Registry.

### Agents

An agent is your AI CLI tool — Claude Code, Gemini, aichat, or anything else. You configure which agent to use, and start handles the command construction and process handoff.

```bash
# Use your default configured agent
start

# Switch to a different agent for this session
start --agent gemini
```

### Roles

A role defines how the AI agent should behave — its expertise, tone, and focus area. Roles become the system prompt for your session.

```bash
# Start with a Go expert role
start --role go-expert

# Use a role from a local file
start --role ./prompts/senior-reviewer.md
```

Roles are installed from the registry:

```bash
start assets add role/go-expert
start assets add role/code-reviewer
```

### Tasks

A task is a reusable, parameterisable prompt for a specific workflow. Run a task instead of typing the same instructions repeatedly.

```bash
# Run a configured task
start task pre-commit-review

# Pass instructions to a parameterised task
start task code-review "Focus on error handling"

# Run a task from a local file
start task ./tasks/my-review.md
```

Tasks only include required contexts by default, keeping prompts focused. Tasks are also available from the registry:

```bash
start assets add task/pre-commit-review
start assets add task/debug-help
```

### Contexts

Contexts are document fragments injected into the prompt — project overviews, environment details, coding standards, or anything else the agent needs to know. Contexts are tagged and selectively included.

```bash
# Include specific contexts by tag
start --context security,performance

# Include a context from a local file
start --context ./AGENTS.md
```

Your project's context files (like `AGENTS.md`, `README.md`, or `PROJECT.md`) are mapped to context definitions in config, so start knows exactly what to include and when.

### Configuration

Configuration is stored in CUE format at `~/.start/config.cue` (global) and `./.start/config.cue` (project-local). The `--local` flag targets project config instead of global.

```bash
# View effective configuration
start config

# Edit agent settings interactively
start config edit agent

# Edit with flags (non-interactive)
start config settings default_agent claude

# Use project-local config
start --local config
```

### Dry Run

Preview exactly what start would send to your agent before committing:

```bash
# See the composed prompt and command without launching
start --dry-run
start task code-review --dry-run
start prompt "My question" --dry-run
```

Dry run writes to `.start/temp/`:

```
.start/temp/
├── role.md       # System prompt (role content)
├── prompt.md     # Full composed prompt
└── command.txt   # Exact command that would execute
```

## Usage

### Core Commands

```bash
# Launch interactive session with full context
start [flags]

# Send a focused one-off prompt
start prompt [text] [flags]

# Run a reusable predefined task
start task <name> [instructions] [flags]
```

### Assets Management

```bash
# Browse available registry packages
start assets browse

# Search for packages
start assets search go

# Install a package
start assets add role/go-expert
start assets add task/pre-commit-review

# List installed assets
start assets list

# Update installed packages
start assets update
```

### Configuration

```bash
# Display current configuration
start config

# Edit specific sections interactively
start config edit agent
start config edit role
start config edit context
start config edit task
start config edit settings
```

### Search and Discovery

```bash
# Search across all installed and registry assets
start search go
start search review
```

### Diagnostics

```bash
# Diagnose setup, validate configuration, suggest fixes
start doctor
```

### Shell Completions

```bash
# Install tab-completion for your shell
start completion bash
start completion zsh
start completion fish
```

## CLI Reference

### Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--agent` | `-a` | Override agent for this session |
| `--role` | `-r` | Override role (config name or file path) |
| `--model` | `-m` | Override model selection |
| `--context` | `-c` | Select contexts (tags or file paths) |
| `--dry-run` | | Preview execution without launching |
| `--local` | `-l` | Use project-local config (`./.start/`) |
| `--quiet` | `-q` | Suppress output |
| `--verbose` | | Detailed output |
| `--debug` | | Debug output (implies `--verbose`) |
| `--no-color` | | Disable coloured output |

### File Path Support

The `--role`, `--context`, and task name arguments accept file paths alongside config names. Detected by prefix:

```bash
start --role ./roles/custom.md
start --context /absolute/path/context.md
start --context ~/shared/project-overview.md
start task ./tasks/my-workflow.md "Additional instructions"
```

### Task Resolution Order

1. Exact name match in configuration
2. Substring match (e.g., `review` matches `code-review`)
3. Registry lookup with auto-install prompt

## Contributing

Contributions welcome! Please:

1. Check existing issues: https://github.com/grantcarthew/start/issues
2. Create an issue for bugs or feature requests
3. Submit pull requests against the `main` branch

### Reporting Issues

Include:

- start version: `start --version`
- Operating system and version
- Full command and error message
- Output from `start doctor`

## License

`start` is licensed under the [Mozilla Public License 2.0](LICENSE).

## Author

Grant Carthew <grant@carthew.net>
