# completion

Generate shell completion scripts for tab-completion of commands and flags.

## Usage

```
start completion <shell>
start completion bash
start completion zsh
start completion fish
```

## Description

Shell completion lets you press Tab to auto-complete commands, subcommands, and flags. Set it up once and save yourself a lot of typing.

The command outputs a completion script to stdout. You redirect it to a file, then configure your shell to load it. After that, tab completion works automatically in new shell sessions.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `bash` | Generate bash completion script |
| `zsh` | Generate zsh completion script |
| `fish` | Generate fish completion script |

### bash

Generates a bash completion script. Requires the `bash-completion` package.

```bash
# Install to user directory
start completion bash > ~/.bash_completion.d/start

# Or install system-wide (Linux)
sudo start completion bash > /etc/bash_completion.d/start

# Or install system-wide (macOS with Homebrew)
start completion bash > $(brew --prefix)/etc/bash_completion.d/start
```

Restart your shell or run `source ~/.bashrc` to activate.

### zsh

Generates a zsh completion script.

```bash
# Install to user directory
mkdir -p ~/.zsh/completions
start completion zsh > ~/.zsh/completions/_start

# Add to your .zshrc if not already present
echo 'fpath=(~/.zsh/completions $fpath)' >> ~/.zshrc
echo 'autoload -Uz compinit && compinit' >> ~/.zshrc

# Or install system-wide (macOS with Homebrew)
start completion zsh > $(brew --prefix)/share/zsh/site-functions/_start
```

Restart your shell or run `source ~/.zshrc` to activate.

### fish

Generates a fish completion script. Fish automatically loads completions from `~/.config/fish/completions/`.

```bash
# Install to user directory (auto-loaded by fish)
start completion fish > ~/.config/fish/completions/start.fish
```

Restart your shell or run `source ~/.config/fish/config.fish` to activate.

## Flags

No command-specific flags. Global flags are available but not typically needed.

## Examples

```bash
# Quick test - load completion in current session only
source <(start completion bash)

# Verify completion is working
start <TAB>
# Shows: assets completion config doctor prompt show task

# Complete flags
start --<TAB>
# Shows: --agent --context --dry-run --help --model --quiet --role --verbose

# Complete subcommand flags
start config <TAB>
# Shows: agent context role settings task
```

## Troubleshooting

**Completion not working after installation**

Make sure you restarted your shell or sourced your shell config file.

**bash-completion not found (bash)**

Install the bash-completion package:

- macOS: `brew install bash-completion@2`
- Ubuntu/Debian: `apt install bash-completion`
- Fedora: `dnf install bash-completion`

**compinit errors (zsh)**

Run `rm -f ~/.zcompdump*` then restart your shell. This clears the completion cache.

## See Also

- [config](config.md) - Configure agents, roles, and tasks
- [doctor](doctor.md) - Diagnose configuration issues
