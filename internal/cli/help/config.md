# start Configuration Reference

- Global: ~/.config/start/ (or $XDG_CONFIG_HOME/start/)
- Local: ./.start/
- Local overrides global; --local flag targets ./.start/ only

Files: agents.cue, roles.cue, contexts.cue, tasks.cue, settings.cue

Settings: `default_agent` `shell` `timeout` `assets_index`

Context inclusion: `--required` always included; `--default` included when no -c flag; `start prompt` excludes defaults unless `-c default`

```
start config agent list
start config agent add --name claude --bin claude --command 'claude {{.prompt}}'
start config agent edit claude --bin claude-code
start config agent default claude
start config agent remove claude
start config role list
start config role add --name go-expert --file path/to/role.md
start config role edit go-expert --description "Go expert"
start config role remove go-expert
start config context list
start config context add --name project --file PROJECT.md --default
start config context add --name env --command "env" --required
start config context edit project --required
start config context remove project
start config task list
start config task add --name review --prompt "Review for issues: {{.instructions}}"
start config task add --name commit --file tasks/commit.md --role git-expert
start config task edit review --role code-reviewer
start config task remove review
start config settings
start config settings default_agent claude
start config settings shell /bin/bash
start config settings timeout 120
start config settings assets_index --unset
```
