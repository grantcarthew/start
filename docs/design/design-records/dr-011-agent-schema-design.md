# DR-011: Agent Schema Design

- Date: 2025-12-03
- Status: Accepted
- Category: CUE

## Problem

Agents are command templates that launch AI CLI tools. The agent schema must:

- Support various AI CLI tools (Claude, Gemini, GPT, Ollama, etc.)
- Handle different command-line interfaces and flag styles
- Support model selection and mapping
- Enable auto-detection of installed agents
- Be flexible enough for custom scripts and wrappers

## Decision

Agents define command templates with placeholders for runtime substitution. Only the `command` field is required. Agents do NOT use UTD - they execute commands, not generate content.

Schema structure:

```cue
#Agent: {
    command: string & !=""           // Required: command template
    bin?: string & !=""              // Optional: binary for auto-detection
    description?: string             // Optional: human-readable description
    tags?: [...string]               // Optional: for search
    default_model?: string           // Optional: default model name
    models?: [string]: string & !="" // Optional: friendly name → identifier
}
```

## Why

Command-only requirement:

- Maximum flexibility for agent definitions
- Works with any executable, script, or wrapper
- No artificial constraints on command structure
- User responsibility to provide valid commands

No command validation:

- Different agents have different flag styles
- Validating placeholder usage is fragile
- If command fails, shell provides clear error
- Simpler implementation, fewer edge cases

Optional `bin` field:

- Enables auto-detection ("is claude installed?")
- Provides `{{.bin}}` placeholder for DRY
- Not required - user can put full path in command
- Useful for `start doctor` to check installations

No UTD embedding:

- Agents execute commands, not generate content
- Content generation happens in tasks, roles, contexts
- Agent receives composed prompt via `{{.prompt}}` placeholder
- Clear separation of concerns

Model mapping:

- Friendly names map to full model identifiers
- `--model sonnet` → `claude-3-7-sonnet-20250219`
- `default_model` specifies fallback when `--model` not provided
- Both optional - simple agents need neither

## Trade-offs

Accept:

- No validation of command correctness
- Invalid commands fail at runtime, not load time
- User must ensure command template is valid
- No enforcement of placeholder usage

Gain:

- Works with any command or script
- Extremely flexible agent definitions
- Simple schema, easy to understand
- No false negatives from over-strict validation
- Debug agents like `echo "{{.prompt}}"` work naturally

## Structure

Agent configuration:

```cue
agents: {
    "claude": {
        bin:         "claude"
        command:     "{{.bin}} --model {{.model}} --append-system-prompt '{{.role}}' '{{.prompt}}'"
        description: "Claude Code by Anthropic"
        default_model: "sonnet"
        models: {
            haiku:  "claude-3-5-haiku-20241022"
            sonnet: "claude-3-7-sonnet-20250219"
            opus:   "claude-opus-4-20250514"
        }
        tags: ["anthropic", "claude", "ai"]
    }
}
```

Field definitions:

- `command` (string, required) - Command template with placeholders
- `bin` (string, optional) - Binary name for detection and `{{.bin}}` placeholder
- `description` (string, optional) - Human-readable summary
- `tags` ([]string, optional) - Keywords for search
- `default_model` (string, optional) - Model name when `--model` not specified
- `models` (map, optional) - Friendly names to full model identifiers

## Placeholders

Agent command templates support these placeholders:

- `{{.bin}}` - The bin field value
- `{{.model}}` - Resolved model identifier (from models map or direct)
- `{{.prompt}}` - Composed prompt from task/UTD resolution
- `{{.role}}` - Role content (inline)
- `{{.role_file}}` - Role file path (for file-based agents)

## Usage Examples

Full-featured agent:

```cue
agents: "claude": {
    bin:         "claude"
    command:     "{{.bin}} --model {{.model}} --append-system-prompt '{{.role}}' '{{.prompt}}'"
    description: "Claude Code by Anthropic"
    default_model: "sonnet"
    models: {
        haiku:  "claude-3-5-haiku-20241022"
        sonnet: "claude-3-7-sonnet-20250219"
        opus:   "claude-opus-4-20250514"
    }
}
```

Minimal agent:

```cue
agents: "simple": {
    command: "my-ai-tool '{{.prompt}}'"
}
```

Custom script wrapper:

```cue
agents: "custom": {
    command:     "./scripts/ai-wrapper.sh --role '{{.role}}' --prompt '{{.prompt}}'"
    description: "Project-specific AI wrapper"
}
```

Debug/test agent:

```cue
agents: "echo": {
    command:     "echo 'Would send: {{.prompt}}'"
    description: "Debug agent that echoes the prompt"
}
```

Local LLM:

```cue
agents: "ollama": {
    bin:         "ollama"
    command:     "{{.bin}} run {{.model}} '{{.prompt}}'"
    description: "Ollama local LLM runner"
    default_model: "llama3"
    models: {
        llama3:    "llama3.2"
        mistral:   "mistral"
        codellama: "codellama"
    }
}
```

## Model Resolution

When `--model` flag is provided:

1. Check if value exists in agent's `models` map
2. If found: use mapped identifier
3. If not found: use value directly (allows full identifiers)

When `--model` flag not provided:

1. Use `default_model` if specified
2. If no default: error (model required but not specified)

Examples:

```bash
start --agent claude --model sonnet    # → claude-3-7-sonnet-20250219
start --agent claude --model gpt-4     # → gpt-4 (passed through)
start --agent claude                   # → claude-3-7-sonnet-20250219 (default)
```
