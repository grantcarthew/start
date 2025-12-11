# DR-005: Go Templates for UTD Pattern

- Date: 2025-12-02
- Status: Accepted
- Category: UTD

## Problem

The Unified Template Design (UTD) pattern needs a template syntax for placeholder substitution in prompt text.

Requirements:

- Support placeholder substitution (file paths, command output, dates)
- Allow both simple substitution and complex logic (conditionals, loops)
- Minimize custom code (validation, parsing, execution)
- Provide clear, learnable syntax
- Validate templates at runtime
- Support extensibility (custom functions, helpers)

## Decision

Use Go's `text/template` package with `{{.placeholder}}` syntax for all UTD template operations.

**Template syntax:**

- `{{.file}}` - File path
- `{{.file_contents}}` - File contents
- `{{.command}}` - Command string
- `{{.command_output}}` - Command output
- `{{.date}}` - Current timestamp
- `{{.instructions}}` - Task-specific instructions

**Full Go template support:**

- Conditionals: `{{if .command_output}}...{{end}}`
- Loops: `{{range .items}}...{{end}}`
- Functions: `{{.file | trim}}`
- Pipelines: `{{.command_output | upper | trim}}`
- Comments: `{{/* comment */}}`

**Runtime injection:**

- Go code builds template data map
- Executes `text/template.Execute` with data
- Returns rendered text or error

## Why

**Go templates are standard:**

- Part of Go standard library (`text/template`)
- Well-documented, widely known syntax
- Battle-tested in production systems
- Zero external dependencies

**Eliminates custom code:**

- No custom placeholder parser required
- No custom substitution logic required
- No custom validation (template validates itself)
- No custom error messages (Go provides them)

**Powerful yet simple:**

- Simple case: `{{.file_contents}}` (just substitution)
- Complex case: `{{if .command_output}}...{{else}}...{{end}}`
- Users choose complexity level
- Progressive disclosure (learn advanced features as needed)

**Extensibility:**

- Can add custom functions via `template.Funcs()`
- Can provide helpers (trim, upper, lower, etc.)
- Future-proof for advanced use cases

**Runtime validation:**

- Template syntax validated during execution
- Clear error messages for invalid syntax
- Fails fast on typos (`{{.commnd_output}}` detected)

## Trade-offs

Accept:

- More verbose syntax (`{{.file}}` vs simpler alternatives)
- Must escape template syntax in content (`{{` needs `{{"{{"}}`)
- Learning curve for Go template features
- Dot prefix required for all placeholders

Gain:

- Zero custom template code
- Standard, documented syntax
- Powerful features (conditionals, loops, functions)
- Runtime validation with clear errors
- Extensible via custom functions
- Future-proof

## Alternatives Considered

**1. Simple placeholder syntax (`{file}`):**

```
Prompt: "File: {file}, Date: {date}"
```

- Pro: Simpler syntax, less typing
- Pro: Easier to read
- Con: Must write custom parser
- Con: Must write custom validator
- Con: Must write custom executor
- Con: No advanced features without building them
- Con: More code to maintain
- Rejected: Too much custom code for marginal syntax benefit

**2. String substitution only:**

```go
strings.ReplaceAll(prompt, "{file}", file)
```

- Pro: Simplest implementation
- Pro: No parsing required
- Con: No conditionals (can't handle optional content)
- Con: No loops (can't iterate)
- Con: Must handle escaping manually
- Con: Limited flexibility
- Rejected: Too limiting for advanced use cases

**3. External template engine (Mustache, Handlebars):**

```
Prompt: "File: {{file}}, Date: {{date}}"
```

- Pro: Simple, well-known syntax
- Pro: Language-agnostic
- Con: External dependency
- Con: Non-standard in Go ecosystem
- Con: Limited features (no custom functions)
- Rejected: Go templates are standard library, no dependencies needed

**4. Jinja2-style templates:**

```
Prompt: "File: {{ file }}, Date: {{ date }}"
```

- Pro: Familiar to Python developers
- Con: Requires external library
- Con: Different from Go conventions
- Con: Additional learning curve
- Rejected: Go templates are idiomatic for Go projects

## Implementation

**Template data structure:**

Provide these fields to Go's template engine:

- File path (path to resolved temp file in `.start/temp/`)
- File contents (resolved template content, empty string if not read)
- Command string (always available as string)
- Command output (empty string if not executed)
- Current date (ISO 8601 timestamp)
- Instructions (task-specific, empty for roles/contexts)

**Template execution flow:**

1. Parse template using `text/template.Parse()`
2. Check for syntax errors, fail fast if invalid
3. Build data map with placeholder values
4. Execute template with data using `template.Execute()`
5. Return rendered text or error

**Lazy evaluation strategy:**

Before reading files or executing commands:

- Scan template for `{{.file_contents}}` placeholder
- Only read file if placeholder exists
- Scan template for `{{.command_output}}` placeholder
- Only execute command if placeholder exists
- File path and command string are always available (no I/O needed)

**Custom functions (optional enhancement):**

Template can be extended with custom functions:

- String manipulation: trim, upper, lower
- Text processing: truncate, indent, wrap
- Use Go's template.FuncMap to register functions
- Functions available in all templates

## Examples

**Simple substitution:**

```cue
prompt: "Current date: {{.date}}"
```

Result: `Current date: 2025-12-02T10:30:00+10:00`

**Conditional content:**

```cue
command: "git status --short"
prompt: """
{{if .command_output}}
Changes detected:
{{.command_output}}
{{else}}
Working tree clean.
{{end}}
"""
```

**Combined file and command:**

```cue
file:    "./PROJECT.md"
command: "git log -5 --oneline"
prompt: """
Project documentation:
{{.file_contents}}

{{if .command_output}}
Recent commits:
{{.command_output}}
{{end}}
"""
```

**Advanced: Custom function (future):**

```cue
command: "git status"
prompt:  "Status: {{.command_output | trim | upper}}"
```

## Validation

**At runtime:**

- Template syntax validated by `text/template.Parse()`
- Unknown placeholders detected (typo: `{{.commnd}}` → error)
- Template execution errors reported with clear messages

**CUE schema:**

- Fields are strings (CUE doesn't validate Go template syntax)
- Go runtime performs template validation
- Invalid templates fail at execution time

## Error Messages

**Invalid syntax:**

```
Error: invalid template syntax: template: utd:1: unexpected "}" in operand
```

**Unknown placeholder:**

```
Error: template execution failed: template: utd:1:5: executing "utd" at <.commnd_output>:
can't evaluate field commnd_output in type TemplateData
```

**Missing value (non-fatal):**

```
Warning: {{.file_contents}} requested but file not found, using empty string
```

## Security Considerations

**Template execution is safe:**

- Go templates cannot execute arbitrary code
- No file system access from templates
- No network access from templates
- No process execution from templates

**Command execution is controlled:**

- Commands executed by Go code, not templates
- Templates only receive command output (strings)
- Command execution subject to timeout limits
- Execution context controlled by application

## Future Enhancements

**Custom functions:**

- String manipulation: trim, upper, lower, truncate
- Formatting: indent, wrap, escape
- Date formatting: custom date formats
- Conditionals: isEmpty, isNotEmpty, contains

**Template helpers:**

- File path manipulation: basename, dirname, ext
- Text processing: lines, words, chars
- Encoding: base64, json, yaml

## Updates

- 2025-12-08: Clarified that `{{.file}}` returns resolved temp file path, `{{.file_contents}}` returns resolved content. All content files are template-processed and written to `.start/temp/`.

## See Also

- [UTD Pattern Documentation](../utd-pattern.md)
- [Go text/template package](https://pkg.go.dev/text/template)
