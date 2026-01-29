# Configuring Aider to use tinyMem

This guide explains how to configure [Aider](https://aider.chat/) to use `tinyMem` as a proxy.

## 1. Configure tinyMem for LM Studio

Ensure your `.tinyMem/config.toml` is set up to point to LM Studio:

```toml
[proxy]
port = 8080
base_url = "http://localhost:1234/v1"

[llm]
base_url = "http://localhost:1234/v1"
```

## 2. Start tinyMem Proxy

```bash
tinymem proxy
```

## 3. Run Aider through tinyMem

Aider uses LiteLLM internally. To route Aider through tinyMem, you should point Aider's API base to tinyMem's proxy.

### Option A: Command Line Arguments

```bash
aider --openai-api-base http://localhost:8080/v1 --model openai/qwen2.5-coder-7b-instruct
```

*Note: The `openai/` prefix might be required by LiteLLM to treat the local endpoint as OpenAI-compatible.*

### Option B: Environment Variables

```bash
export OPENAI_API_BASE="http://localhost:8080/v1"
export OPENAI_API_KEY="not-needed"
aider --model openai/qwen2.5-coder-7b-instruct
```

## Using Qwen2.5-Coder

Qwen2.5-Coder is highly recommended for use with Aider. When using it via LM Studio and tinyMem:

1.  **Start LM Studio** and load `qwen2.5-coder-7b-instruct`.
2.  **Enable the Local Server** in LM Studio (default port 1234).
3.  **Chat Templates**: Ensure LM Studio is configured to use the "Auto-detect" or "Qwen2" chat template to ensure correct message formatting.
4.  Ensure tinyMem's `base_url` matches LM Studio's address.
4.  Run Aider pointing to tinyMem.

tinyMem will automatically inject relevant project context into Aider's requests and extract new memories from its responses.
