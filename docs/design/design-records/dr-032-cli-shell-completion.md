# DR-032: CLI Shell Completion

- Date: 2025-12-19
- Status: Accepted
- Category: CLI

## Problem

Modern CLI tools provide shell completion for improved usability. Users expect tab-completion for commands and flags. Without completion:

- Users must type full command and flag names
- No discoverability of available subcommands
- More typos and errors
- CLI feels outdated compared to modern tools

The CLI needs a completion strategy that addresses:

- Which shells to support
- Command structure for generating completion scripts
- Installation method (automatic vs manual)
- Scope of completions (static vs dynamic values)

## Decision

Implement shell completion for bash, zsh, and fish using Cobra's built-in completion generation. Provide manual installation via script output to stdout. Complete commands and flags only (static completion).

Command structure:

- `start completion bash` - Output bash completion script
- `start completion zsh` - Output zsh completion script
- `start completion fish` - Output fish completion script

Each command outputs the completion script to stdout with installation instructions in the command's help text.

## Why

Three shells cover most users:

- bash: Most common on Linux servers
- zsh: macOS default since Catalina, popular on Linux
- fish: Growing popularity among developers
- PowerShell excluded: Windows not supported (DR-006)

Cobra provides free implementation:

- Built-in completion generation for bash, zsh, fish
- Tested, maintained completion logic
- No custom code required for static completion
- Standard patterns that work across shells

Manual installation is sufficient:

- One-time setup per machine
- Clear instructions in help text
- Avoids complexity of path detection, OS differences, permissions
- Can add auto-install later if users request it

Static completion provides immediate value:

- All commands and subcommands completed
- All flags (long and short forms) completed
- No config loading overhead during completion
- Dynamic completion (agent names, task names) can be added later

## Trade-offs

Accept:

- Users must manually install completion scripts
- No dynamic completion for agent/task/role names
- Different installation steps per shell (documented in help)

Gain:

- Zero implementation complexity (Cobra handles everything)
- No config loading during completion (fast)
- No maintenance burden for completion logic
- Professional UX (tab completion works)
- Easy to extend later with dynamic completions

## Alternatives

No shell completion:

- Pro: No implementation required
- Con: Poor UX, feels unprofessional
- Con: Users must type full commands
- Rejected: Completion is expected in modern CLIs

Auto-install command (`start completion install bash`):

- Pro: Lower friction for users
- Pro: Handles OS-specific paths automatically
- Con: Complex path detection logic
- Con: Permission handling (user vs system install)
- Con: More code to maintain
- Rejected: Manual installation is acceptable for one-time setup. Can revisit if users request it.

Dynamic completion (agent names, task names):

- Pro: Complete `--agent <TAB>` with configured agents
- Pro: Complete `start task <TAB>` with task names
- Con: Requires config loading during completion
- Con: Must handle missing/invalid config gracefully
- Con: More complex implementation
- Rejected: Static completion provides sufficient value. Can add dynamic completion in future version.

Full dynamic completion (including model names):

- Pro: Complete `--model <TAB>` with agent's available models
- Con: Requires parsing nested agent config
- Con: Complex and fragile
- Rejected: Diminishing returns for implementation cost.

## Structure

Command hierarchy:

```
start completion
├── bash   # Output bash completion script
├── zsh    # Output zsh completion script
└── fish   # Output fish completion script
```

No flags on completion subcommands. Each outputs the appropriate script to stdout.

## Usage Examples

Generate and install bash completion:

```bash
# Generate script
start completion bash > ~/.bash_completion.d/start

# Add to .bashrc if not auto-sourced
echo 'source ~/.bash_completion.d/start' >> ~/.bashrc

# Reload
source ~/.bashrc
```

Generate and install zsh completion:

```bash
# Generate script
start completion zsh > ~/.zsh/completions/_start

# Ensure completions directory is in fpath (add to .zshrc)
fpath=(~/.zsh/completions $fpath)
autoload -Uz compinit && compinit

# Reload
source ~/.zshrc
```

Generate and install fish completion:

```bash
# Generate script (fish auto-loads from this directory)
start completion fish > ~/.config/fish/completions/start.fish
```

Using completion:

```bash
# Complete subcommands
start <TAB>
# Shows: assets config doctor prompt show task completion help

# Complete flags
start --<TAB>
# Shows: --agent --context --dry-run --help --model --quiet --role --verbose

# Complete subcommand flags
start config --<TAB>
# Shows available flags for config command
```
