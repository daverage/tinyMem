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

Aider uses LiteLLM internally. LiteLLM requires a provider prefix (like `openai/`) when using a custom API base.

### Option A: Command Line Arguments

```bash
aider --openai-api-base http://localhost:8080/v1 --model openai/qwen2.5-coder-7b-instruct
```

**CRITICAL**: If you omit `openai/` and only pass `--model qwen2.5-coder-7b-instruct`, LiteLLM will throw a `BadRequestError: LLM Provider NOT provided`. Always include the prefix.

## Troubleshooting Connection Errors

If you see `InternalServerError: OpenAIException - Connection error`:

1.  **Verify LM Studio**: Ensure LM Studio is open, the model is loaded, and the **Local Server is started** (usually on port 1234).
2.  **Verify tinyMem**: Ensure `tinymem proxy` is running in your project folder.
3.  **Check URLs**: If `localhost` doesn't work, try using `127.0.0.1` in your `.tinyMem/config.toml`:
    ```toml
    base_url = "http://127.0.0.1:1234/v1"
    ```
4.  **Run Diagnostics**: Use the tinyMem doctor to check connectivity:
    ```bash
    tinymem doctor
    ```
    Look for the "LLM backend reachability" check.

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

## Resolving "Unknown context window size" Warnings

Aider uses LiteLLM, which may not recognize local model strings like `openai/qwen2.5-coder-7b-instruct`. You might see a warning like:
`Warning for openai/qwen2.5-coder-7b-instruct: Unknown context window size and costs, using sane defaults.`

To resolve this, create a file named **`.aider.model.metadata.json`** in your project root:

```json
{
    "openai/qwen2.5-coder-7b-instruct": {
        "max_tokens": 32768,
        "input_cost_per_token": 0.0,
        "output_cost_per_token": 0.0,
        "litellm_provider": "openai",
        "mode": "chat"
    }
}
```

Aider will automatically detect this file in the current directory or your home directory. You can also specify it explicitly using:
```bash
aider --model-metadata-file .aider.model.metadata.json --model openai/qwen2.5-coder-7b-instruct
```
