# tinyMem Qwen Integration Guide

tinyMem supports Qwen models through both **Proxy Mode** (for OpenAI-compatible tools like Ollama/LM Studio) and **MCP** (if using an MCP-compatible client with Qwen).

## Quick Start: Which Mode?

| Tool | Recommended Mode | Why |
|------|------------------|-----|
| **Qwen CLI** | **Proxy** | Most Qwen CLI tools support OpenAI-compatible API bases. |
| **Ollama** | **Proxy** | tinyMem proxies requests *to* Ollama, injecting memory context. |
| **LM Studio** | **Proxy** | tinyMem proxies requests *to* LM Studio. |

---

## 1. Qwen via Ollama (Proxy Mode)

In this setup, tinyMem sits between your client (e.g., a script, a terminal chat) and Ollama.

**Flow:** `Client -> tinyMem (Port 8080) -> Ollama (Port 11434)`

### Configuration

1.  **Configure tinyMem:**
    Create/Edit `.tinyMem/config.toml` in your project root:

    ```toml
    [proxy]
    port = 8080
    base_url = "http://localhost:11434/v1"  # Pointing to Ollama

    [llm]
    model = "qwen2.5-coder" # Verify this matches your `ollama list` name
    ```

2.  **Start tinyMem Proxy:**
    ```bash
    tinymem proxy
    ```

3.  **Configure Your Client:**
    Point your client to tinyMem instead of Ollama.
    -   **Base URL:** `http://localhost:8080/v1`
    -   **API Key:** `any-string` (Ollama ignores this)

### Example: `curl` Test

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dummy" \
  -d '{
    "model": "qwen2.5-coder",
    "messages": [
      {"role": "user", "content": "What was the decision on the database schema?"}
    ]
  }'
```
*tinyMem will intercept this, query memory for "database schema", inject context, and forward to Ollama.*

---

## 2. Qwen via LM Studio (Proxy Mode)

**Flow:** `Client -> tinyMem (Port 8080) -> LM Studio (Port 1234)`

### Configuration

1.  **LM Studio Setup:**
    -   Load a Qwen model (e.g., `Qwen2.5-Coder-7B-Instruct`).
    -   Start the **Local Server** in LM Studio (default port `1234`).

2.  **Configure tinyMem:**
    Edit `.tinyMem/config.toml`:

    ```toml
    [proxy]
    port = 8080
    base_url = "http://localhost:1234/v1"  # Pointing to LM Studio

    [llm]
    base_url = "http://localhost:1234/v1"
    # Tip: Use the exact string LM Studio shows in the "Model Identifier" field
    model = "qwen2.5-coder-7b-instruct"
    ```

3.  **Start Proxy:**
    ```bash
    tinymem proxy
    ```

4.  **Connect Client:**
    Configure your IDE or CLI tool to use `http://localhost:8080/v1` as the OpenAI Base URL.

---

## 3. Qwen CLI (Native)

If you are using a specific `qwen-cli` tool:

-   **If it supports OpenAI-compatible backends:**
    Set `OPENAI_BASE_URL=http://localhost:8080/v1` and run it.

-   **If it does NOT support OpenAI backends:**
    You cannot use tinyMem's proxy features. However, if the CLI supports **MCP**, you can register tinyMem as an MCP server (see IDEs guide).

---

## Tips for Qwen

-   **Context Window:** Qwen models often have large context windows (32k+). You can safely increase `max_tokens` in `.tinyMem/config.toml` to retrieve more memories.
    ```toml
    [recall]
    max_tokens = 8000
    ```
-   **CoVe (Chain of Verification):** Qwen 2.5 Coder is excellent at logic. Enable CoVe for better memory accuracy.
    ```toml
    [cove]
    enabled = true
    confidence_threshold = 0.7
    ```

## Configuration Reference

For full configuration options, see [Configuration.md](Configuration.md).

## Troubleshooting

-   **Connection Refused:** Ensure your backend (Ollama/LM Studio) is running *and* listening on the port specified in `config.toml`.
-   **Empty Responses:** Check if the model name in `config.toml` matches exactly what the backend expects.