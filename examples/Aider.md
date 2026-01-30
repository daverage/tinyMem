# tinyMem Aider Integration Guide

[Aider](https://aider.chat/) is a popular AI pair programmer in the terminal. It works excellently with tinyMem via **Proxy Mode**.

## Prerequisites

1.  **Aider** installed (`pip install aider-chat`).
2.  **tinyMem** installed and configured.
3.  A backend LLM running (Ollama, LM Studio) OR an API key for a cloud provider.

## Configuration

### 1. Configure tinyMem Backend
Ensure `.tinyMem/config.toml` points to your actual model provider.

**Example (Ollama):**
```toml
[proxy]
port = 8080
base_url = "http://localhost:11434/v1"
```

### 2. Start tinyMem Proxy
```bash
tinymem proxy
```

## Running Aider

Aider needs to be told to talk to `localhost:8080` instead of the real API.

### Option A: Command Line Flags (Recommended)

```bash
aider \
  --openai-api-base http://localhost:8080/v1 \
  --openai-api-key dummy \
  --model openai/qwen2.5-coder  # Prefix 'openai/' tells Aider to use generic client
```

> **Critical:** You MUST use the `openai/` prefix for the model name (e.g., `openai/qwen2.5-coder` or `openai/gpt-4`). This forces Aider to use its generic OpenAI client, which respects the custom API base. If you just say `--model gpt-4`, it might try to hit the official OpenAI API directly.

### Option B: Environment Variables

```bash
export OPENAI_API_BASE=http://localhost:8080/v1
export OPENAI_API_KEY=dummy

aider --model openai/your-model-name
```

## Configuration Reference

For full configuration options, see [Configuration.md](Configuration.md).

## Troubleshooting

### "Unknown context window size"
Aider might not know the context limit of a local model proxied through tinyMem. Create a `.aider.model.metadata.json` file in your project root:

```json
{
    "openai/qwen2.5-coder": {
        "max_tokens": 32768,
        "input_cost_per_token": 0.0,
        "output_cost_per_token": 0.0,
        "litellm_provider": "openai",
        "mode": "chat"
    }
}
```

Then run aider with:
```bash
aider --model-metadata-file .aider.model.metadata.json --model openai/qwen2.5-coder ...
```

### Connection Error
-   Ensure `tinymem proxy` is running.
-   Check `tinymem doctor`.
-   Try using `127.0.0.1` instead of `localhost` if on Windows/WSL.

