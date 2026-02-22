# start Agent Reference

CLI orchestrator for AI agents. Manages prompt, role, and context injection.

- Global config: ~/.config/start/ (or $XDG_CONFIG_HOME/start/)
- Local config: ./.start/ (--local flag targets this)
- See `start help config` for config structure
- See `start help templates` for template placeholder syntax

```
start
start --role go-expert
start --role go-expert --model sonnet
start --agent gemini --model flash
start --context project,readme
start --no-role
start --dry-run
start prompt "Fix the bug in main.go"
start prompt ./notes.md
start prompt "Explain this" --role teacher
start prompt "Quick question" -c default
start task pre-commit-review
start task review "focus on error handling"
start task --tag golang
start task ./custom-task.md
start show
start show go-expert
start show role go-expert
start show context project
start show agent claude
start show task review
start show --global
start show --local
start config
start config agent
start config agent list
start config agent add --name claude --bin claude --command 'claude --print {{.prompt}}'
start config agent default claude
start config role
start config role list
start config role add --name go-expert --file ~/.config/start/roles/go-expert.md
start config context
start config context list
start config context add --name project --file PROJECT.md --default
start config task
start config task list
start config task add --name review --prompt "Review this code for issues"
start config settings
start config settings default_agent claude
start config settings timeout 120
start search golang
start search --tag review
start assets search golang
start assets add golang/code-review
start assets list
start assets update
start doctor
```
