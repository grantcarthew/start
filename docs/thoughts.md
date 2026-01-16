# Observations and Thoughts

- ~~start show agent does not show the default model~~ (fixed)
- Model aliases may not resolve correctly on Vertex AI / Bedrock providers
  - Claude CLI has provider-specific default model mappings
  - See: https://github.com/anthropics/claude-code/issues/18447
  - Workaround: Use explicit model IDs in config or set ANTHROPIC_DEFAULT_OPUS_MODEL
  - Auto-setup now warns users about this
