# Configuring OpenAI API Clients (formerly Codex) with tinyMem

This guide explains how to configure OpenAI API clients (which previously included access to Codex models, now deprecated) to use `tinyMem` as a proxy. `tinyMem` acts as an HTTP proxy, enabling it to inject context and manage memory for your AI interactions.

**Note on Codex:** The OpenAI Codex models have been deprecated. This guide focuses on configuring general OpenAI API clients, which can interact with current OpenAI models like GPT-3.5 or GPT-4, through `tinyMem`.

Before you begin, ensure `tinyMem` is running in proxy mode (e.g., `tinymem proxy`) and listening on `http://localhost:8080` (or your configured port).

The OpenAI Python client documentation (November 2025) explicitly shows how to pass a custom `httpx` client with proxy settings plus a `base_url`, and it calls out `OPENAI_BASE_URL` as the environment variable you can use instead of embedding a full URL in your code, so the examples below mirror that official guidance.

## 1. OpenAI Python Client Library (`openai`)

The OpenAI Python client library (using the `openai` package) can be configured to use a proxy in several ways.

### Using `http_client` (Explicit Configuration)

This method provides explicit control over the HTTP client, which is often the most reliable.

```python
from openai import OpenAI
import httpx
import os

# tinyMem proxy address
PROXY_ADDRESS = "http://localhost:8080"

# Configure httpx client to use tinyMem proxy
_http_client = httpx.Client(
    proxies={
        "http://": PROXY_ADDRESS,
        "https://": PROXY_ADDRESS,
    },
)

# Initialize the OpenAI client with the custom http_client
client = OpenAI(
    # Set your API key or ensure OPENAI_API_KEY environment variable is set
    # api_key="YOUR_OPENAI_API_KEY",
    http_client=_http_client,
    # If your LLM backend behind tinyMem supports it, you can also set base_url here
    # base_url="http://localhost:11434/v1" # Example for Ollama
)

# Example usage (e.g., chat completions)
try:
    response = client.chat.completions.create(
        model="gpt-3.5-turbo", # or your local model served via tinyMem
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Explain the concept of recursion."},
        ]
    )
    print(response.choices[0].message.content)
except Exception as e:
    print(f"Error making API call: {e}")

```

### Using Environment Variables

The OpenAI Python client (which uses `httpx` internally) can often pick up proxy settings from standard environment variables.

```bash
# Set the HTTP_PROXY and HTTPS_PROXY environment variables
export HTTP_PROXY="http://localhost:8080"
export HTTPS_PROXY="http://localhost:8080"

# (Optional) If you have hosts that should bypass the proxy, use NO_PROXY
# export NO_PROXY="localhost,127.0.0.1"

# Then run your Python script that uses the OpenAI client:
# python your_script.py

# To unset the proxy:
# unset HTTP_PROXY
# unset HTTPS_PROXY
# unset NO_PROXY
```
Ensure these environment variables are set in the terminal *before* you run your Python script.

## 2. Setting `OPENAI_BASE_URL` (for local LLMs)

If you are using `tinyMem` to proxy to a local LLM that provides an OpenAI-compatible API (like Ollama or LM Studio), you can direct your OpenAI client to `tinyMem` by setting the `OPENAI_BASE_URL` environment variable.

```bash
# Point your OpenAI client's base URL to tinyMem's proxy address
export OPENAI_BASE_URL="http://localhost:8080/v1"

# Then run your OpenAI client application.
# It will now send requests to tinyMem, which will forward them to your
# actual local LLM backend (as configured in tinyMem's config.toml).

# To unset:
# unset OPENAI_BASE_URL
```
**Note:** When using `OPENAI_BASE_URL`, `tinyMem` will typically expect the `api_key` in your client to be a dummy value (e.g., `sk-xxxxxxxxxxxxxxxxxxxxxxxx`).

## 3. MCP Integration (for supported Codex CLI versions)

Some versions of a `codex` CLI might support registering an external MCP server directly from the command line.

**Example command:**
```bash
codex mcp add tinymem \
    --env TINYMEM_LOG_LEVEL=info \
    -- tinymem mcp
```

**With CoVe disabled:**
```bash
codex mcp add tinymem \
    --env TINYMEM_LOG_LEVEL=info \
    --env TINYMEM_COVE_ENABLED=false \
    -- tinymem mcp
```
**How to use:**
1.  Run this command once to register `tinyMem` as an MCP server named `tinymem`. The `codex` CLI will store this configuration.
2.  The `codex` CLI will then be able to use `tinyMem`'s MCP tools (like `memory_query`, `memory_write`, etc.).
3.  The `--` separates the `codex mcp add` arguments from the actual command that `codex` will execute to start the MCP server (`tinymem mcp`).
4.  Ensure that `tinymem` is in your system's PATH.
