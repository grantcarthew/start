# CUE-Go Integration Notes

Research notes for integrating CUE with Go in the `start` CLI.

## CUE Go API

### Creating a Context

```go
import "cuelang.org/go/cue/cuecontext"

ctx := cuecontext.New()
```

The context is the entry point for all CUE operations. It manages the CUE runtime and should be reused across operations.

### Loading CUE

**From string/bytes (for testing or inline config):**

```go
v := ctx.CompileString(`
    tasks: "code-review": {
        command: "git diff --staged"
        prompt:  "Review: {{.command_output}}"
    }
`)
if err := v.Err(); err != nil {
    return err
}
```

**From files (with module support):**

```go
import "cuelang.org/go/cue/load"

insts := load.Instances([]string{"."}, &load.Config{
    Dir: "/path/to/config",
})
if insts[0].Err != nil {
    return insts[0].Err
}

v := ctx.BuildInstance(insts[0])
if err := v.Err(); err != nil {
    return err
}
```

### Validation

```go
// Check for structural errors
if err := v.Err(); err != nil {
    return err
}

// Validate completeness (all values concrete)
if err := v.Validate(cue.Concrete(true)); err != nil {
    return err
}
```

### Extracting Values

**Lookup by path:**

```go
field := v.LookupPath(cue.ParsePath("tasks.code-review.timeout"))

// Get typed values
s, err := field.String()
i, err := field.Int64()
b, err := field.Bool()
```

**Decode into Go struct:**

```go
type Task struct {
    File        string   `json:"file,omitempty"`
    Command     string   `json:"command,omitempty"`
    Prompt      string   `json:"prompt,omitempty"`
    Shell       string   `json:"shell,omitempty"`
    Timeout     int      `json:"timeout,omitempty"`
    Description string   `json:"description,omitempty"`
    Tags        []string `json:"tags,omitempty"`
    Role        string   `json:"role,omitempty"`
    Agent       string   `json:"agent,omitempty"`
}

var task Task
if err := v.LookupPath(cue.ParsePath("tasks.code-review")).Decode(&task); err != nil {
    return err
}
```

**Iterate over map fields:**

```go
tasksVal := v.LookupPath(cue.ParsePath("tasks"))
iter, err := tasksVal.Fields()
if err != nil {
    return err
}

for iter.Next() {
    name := iter.Selector().String()
    var task Task
    if err := iter.Value().Decode(&task); err != nil {
        return err
    }
    // process task...
}
```

### Value Injection

**Convert Go value to CUE:**

```go
data := map[string]any{
    "file_contents":  fileContent,
    "command_output": cmdOutput,
    "date":           time.Now().Format(time.RFC3339),
}
injected := ctx.Encode(data)
```

**Fill at path:**

```go
result := v.FillPath(cue.ParsePath("runtime"), injected)
```

**Unify values (merge):**

```go
combined := base.Unify(overlay)
```

## Configuration Hierarchy

Four layers, closest to working directory wins:

```
LOCAL      ./.start/              Project config + .md files
GLOBAL     ~/.config/start/       User config + .md files
CACHE      ~/.cache/cue/mod/...   Downloaded packages (.cue + .md)
REGISTRY   CUE Central Registry   Published packages (.cue + .md)
```

**Proximity order (highest priority first):**

```
Local > Global > Cache > Registry
```

### Merge Semantics

**Different names are additive (union):**

```
Global: { foo: {...}, bar: {...} }
Local:  { baz: {...} }
Result: { foo: {...}, bar: {...}, baz: {...} }
```

**Same name, closest wins (complete replacement):**

```
Global: { foo: { timeout: 30, shell: "bash" } }
Local:  { foo: { timeout: 10 } }
Result: { foo: { timeout: 10 } }  // local replaces entirely
```

### Loading Order

```go
func LoadConfig(workingDir string) (cue.Value, error) {
    ctx := cuecontext.New()

    // Load in priority order (lowest first, so higher can override)
    // Registry/Cache handled by CUE module imports

    // 1. Global config
    globalInsts := load.Instances([]string{"."}, &load.Config{
        Dir: expandPath("~/.config/start"),
    })
    global := ctx.BuildInstance(globalInsts[0])

    // 2. Local config
    localInsts := load.Instances([]string{"."}, &load.Config{
        Dir: filepath.Join(workingDir, ".start"),
    })
    local := ctx.BuildInstance(localInsts[0])

    // 3. Unify (CUE handles merge - later values win)
    merged := global.Unify(local)

    // 4. Validate
    if err := merged.Validate(); err != nil {
        return cue.Value{}, err
    }

    return merged, nil
}
```

### File Path Resolution

File paths in the `file:` field resolve relative to their source:

| Source | `file: "prompt.md"` resolves to |
|--------|--------------------------------|
| Local | `./.start/prompt.md` |
| Global | `~/.config/start/prompt.md` |
| Package | `~/.cache/cue/mod/.../prompt.md` |

Go must track the source location when extracting values to resolve paths correctly.

## UTD Runtime Flow

The Unified Template Design pattern requires Go to:

1. Load and validate CUE configuration
2. Extract UTD fields (file, command, prompt)
3. Execute file reads and commands (lazy, based on template usage)
4. Build template data map
5. Execute Go template
6. Return rendered text

```
┌─────────────────────────────────────────────────────────────────┐
│  1. LOAD CUE                                                    │
│     load.Instances() + ctx.BuildInstance()                      │
│     CUE validates: types, constraints, structure                │
└──────────────────────────────┬──────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. GO VALIDATION                                               │
│     - At least one of file/command/prompt (UTD constraint)      │
│     - role/agent references exist                               │
└──────────────────────────────┬──────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. LAZY EVALUATION                                             │
│     Scan prompt for placeholders:                               │
│     - {{.file_contents}} present → read file                    │
│     - {{.command_output}} present → execute command             │
└──────────────────────────────┬──────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. BUILD TEMPLATE DATA                                         │
│     map[string]any{                                             │
│         "file":           filePath,                             │
│         "file_contents":  fileContent,                          │
│         "command":        commandStr,                           │
│         "command_output": commandOutput,                        │
│         "date":           timestamp,                            │
│         "instructions":   userInstructions,                     │
│     }                                                           │
└──────────────────────────────┬──────────────────────────────────┘
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. EXECUTE TEMPLATE                                            │
│     text/template.Execute(prompt, data)                         │
│     Returns: rendered prompt string                             │
└─────────────────────────────────────────────────────────────────┘
```

### Template Data Fields

| Field | Source | Always Available |
|-------|--------|------------------|
| `file` | UTD `file:` field (path string) | Yes (empty if not defined) |
| `file_contents` | Content read from `file:` | Only if file read |
| `command` | UTD `command:` field (command string) | Yes (empty if not defined) |
| `command_output` | stdout from executing `command:` | Only if command executed |
| `date` | `time.Now().Format(time.RFC3339)` | Yes |
| `instructions` | CLI args for tasks | Tasks only |

## Error Handling

Error handling depends on the parent entity (per DR-007):

| Entity | On Error | Rationale |
|--------|----------|-----------|
| Task | Fail hard, exit 1 | User-initiated, expects it to work |
| Context | Warn and skip | Optional enrichment, best effort |
| Role | Warn and skip | Optional behavior, agent can run without |

### Error Types

**CUE validation errors:**

```go
if err := v.Validate(); err != nil {
    // Structured error with path information
    // e.g., "tasks.code-review.timeout: invalid value 0 (out of bound >=1)"
}
```

**File errors:**

```go
// File not found
"Error: file not found: ./PROMPT.md"

// Permission denied
"Error: cannot read file: ./prompt.md (permission denied)"
```

**Command errors:**

```go
// Command failed
"Error: command failed with exit code 1"
"Command: git diff --staged"
"Stderr: fatal: not a git repository"

// Timeout
"Error: command timeout after 30 seconds"
```

**Template errors:**

```go
// Syntax error
"Error: invalid template syntax: template: utd:1: unclosed action"

// Unknown placeholder
"Error: can't evaluate field commnd_output (did you mean command_output?)"
```

## CUE vs Go Responsibilities

| Responsibility | CUE | Go |
|----------------|-----|-----|
| Field types | ✓ | |
| Value constraints (e.g., timeout 1-3600) | ✓ | |
| Pattern defaults | ✓ | |
| At least one of file/command/prompt | | ✓ |
| File existence and readability | | ✓ |
| Command execution | | ✓ |
| Template rendering | | ✓ |
| Reference validation (role/agent exist) | | ✓ |
| Error handling by entity type | | ✓ |

## Package Structure

Published packages contain both `.cue` and `.md` files:

```
github.com/grantcarthew/start-task-golang-code-review@v0/
├── module.cue        # Package metadata
├── task.cue          # Task definition
└── prompt.md         # Prompt content (optional, can be inline)
```

Task definition can reference bundled files:

```cue
package review

tasks: "golang/code-review": {
    file:        "prompt.md"  // relative to package
    command:     "git diff --staged"
    description: "Review Go code changes"
}
```

Or use inline prompts:

```cue
package review

tasks: "golang/code-review": {
    command: "git diff --staged"
    prompt: """
Review these Go code changes:

{{.command_output}}

Focus on idiomatic patterns and error handling.
"""
}
```

## Open Questions

1. **Package path resolution:** How does Go determine the filesystem path to a file referenced in a CUE package from the registry cache?

2. **Incremental loading:** Can we load only the specific task/role/context needed, or must we load everything?

3. **Performance:** What's the overhead of CUE parsing and validation? Should we cache parsed values?

4. **Module imports:** How do user configs import from published packages? What's the `module.cue` and import syntax?

## See Also

- [UTD Pattern](../design/utd-pattern.md)
- [DR-005: Go Templates](../design/design-records/dr-005-utd-go-templates.md)
- [DR-007: Error Handling](../design/design-records/dr-007-utd-error-handling.md)
- [P-001: CUE Foundation](../projects/p-001-cue-foundation-architecture.md)
