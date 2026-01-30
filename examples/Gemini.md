# tinyMem Gemini Integration Guide

> **Current Status:** Google Gemini's API is **NOT** natively OpenAI-compatible. This affects how you can use it with tinyMem.

## Supported Use Cases

| Setup | Feasibility | Description |
|-------|-------------|-------------|
| **Gemini Agent with tinyMem** | ✅ **Recommended** | You are building an agent (using Vertex AI or Gemini API) that calls tinyMem via **MCP**. |
| **Gemini as tinyMem Backend** | ⚠️ **Requires Adapter** | You want tinyMem to use Gemini for its internal operations (summarization, CoVe). Requires an intermediate proxy like [lite-llm](https://github.com/BerriAI/litellm) to translate OpenAI -> Gemini. |
| **Gemini CLI Tool** | ✅ **Supported** | You have a CLI tool that uses Gemini but supports custom OpenAI-compatible proxies (rare, but possible). |

---

## 1. Using tinyMem with a Gemini Agent (MCP)

If you are building an autonomous agent using Gemini (e.g., via LangChain, Vertex AI Agent Builder, or custom code) that supports the **Model Context Protocol (MCP)**, you can register tinyMem as a tool.

### Configuration

Add tinyMem to your MCP client configuration:

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_METRICS_ENABLED": "true"
      }
    }
  }
}
```

### Usage

When the Gemini model needs information, your agent framework will route the `memory_query` or `memory_write` tool calls to tinyMem via stdio.

---

## 2. Using Gemini as the LLM Backend

tinyMem uses an internal LLM for tasks like summarizing memories and verifying facts (CoVe). Since tinyMem speaks the OpenAI protocol, you cannot point it directly at `generativelanguage.googleapis.com`.

**Workaround: Use LiteLLM Proxy**

1.  **Install LiteLLM:** `pip install litellm[proxy]`
2.  **Start LiteLLM Proxy:**
    ```bash
    export GEMINI_API_KEY=your_key
    litellm --model gemini/gemini-pro
    # Runs on http://0.0.0.0:4000
    ```
3.  **Configure tinyMem:**
    
    `.tinyMem/config.toml`:
    ```toml
    [llm]
    base_url = "http://localhost:4000" # Point to LiteLLM
    model = "gemini/gemini-pro"
    ```

---

## 3. Configuration Reference

For full configuration options, see [Configuration.md](Configuration.md).

## Troubleshooting

-   **"404 Not Found"**: If pointing tinyMem directly at Google's API, this is expected. Use the LiteLLM workaround.
-   **"Protocol Error"**: Ensure you are using MCP mode (`tinymem mcp`) only with MCP-compatible clients.