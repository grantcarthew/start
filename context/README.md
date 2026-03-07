# Context Directory

AI reference documentation and source code for the `start` project. This directory contains cloned repositories, fetched documentation, and local assets that provide context for AI agents working on this codebase.

## Structure

| Directory    | Description                      | Source                                    |
| ------------ | -------------------------------- | ----------------------------------------- |
| cue          | CUE language source              | <https://github.com/cue-lang/cue>         |
| cuelang-org  | CUE official documentation       | <https://github.com/cue-lang/cuelang.org> |
| docs         | Standalone docs fetched via snag | Various (Claude Code, etc.)               |
| start-assets | CUE-based asset definitions      | Local                                     |

## Rebuild

Run the upsert script to clone missing repos, pull existing ones, and refresh fetched docs:

```bash
./scripts/upsert-context
```
