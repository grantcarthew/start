# start Template Reference

Go template syntax: `{{.placeholder}}`. Two contexts with different placeholders.

Agent command templates (agents.cue `command` field). Do NOT quote `{{.prompt}}` — shell escaping is automatic:
```
claude --print {{.prompt}}
gemini --model {{.model}} --print {{.prompt}}
{{.bin}} --print {{.prompt}}
```

Placeholders: `{{.bin}}` `{{.model}}` `{{.role}}` `{{.role_file}}` `{{.prompt}}` `{{.date}}`

Note: `{{.role_file}}` is a path to a temp file containing the role content. Use it with whatever system-prompt flag your agent binary accepts.

UTD templates (roles, contexts, tasks — `prompt` field or file content):

Placeholders: `{{.instructions}}` `{{.file}}` `{{.file_contents}}` `{{.command}}` `{{.command_output}}` `{{.date}}`

Common mistakes:
- `claude "{{.prompt}}"` — wrong, causes double-quoting; use `claude {{.prompt}}`
- `{prompt}` — wrong, use `{{.prompt}}`
