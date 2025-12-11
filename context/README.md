# Context Directory

AI reference documentation and source code for the `start` project. This directory contains cloned repositories, fetched documentation, and local assets that provide context for AI agents working on this codebase.

## Structure

| Directory       | Description                      | Source                                        |
| --------------- | -------------------------------- | --------------------------------------------- |
| aichat          | AIChat LLM CLI tool reference    | <https://github.com/sigoden/aichat>           |
| cobra           | Go CLI library                   | <https://github.com/spf13/cobra>              |
| cue             | CUE language source              | <https://github.com/cue-lang/cue>             |
| cuelang-org     | CUE official documentation       | <https://github.com/cue-lang/cuelang.org>     |
| docs            | Standalone docs fetched via snag | Various (Claude Code, etc.)                   |
| gemini-cli      | Gemini CLI AI agent              | <https://github.com/google-gemini/gemini-cli> |
| start-assets    | CUE-based asset definitions      | Local                                         |
| start-prototype | TOML prototype research          | Local                                         |

## Rebuild

Run the upsert script to clone missing repos, pull existing ones, and refresh fetched docs:

```bash
./scripts/upsert-context
```
