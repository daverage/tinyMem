# tinyMem Crush/Rush Integration Guide

[Crush](https://github.com/charmbracelet/crush) (and its CLI `rush`) supports both **Model Context Protocol (MCP)** and standard **OpenAI-compatible APIs**, giving you two flexible ways to integrate with tinyMem.

## Option A: MCP Mode (Recommended)

This mode allows Rush to natively use tinyMem's tools (like `memory_query`, `memory_write`, `memory_health`) to actively manage context during your chat.

### 1. Configuration

Crush looks for a configuration file at `.crush.json` (project local) or `~/.config/crush/crush.json` (global).

**Create/Edit `.crush.json`:**

```json
{
  "mcp": {
    "tinymem": {
      "type": "stdio",
      "command": "tinymem",
      "args": ["mcp"],
      "timeout": 120,
      "env": {
        "TINYMEM_METRICS_ENABLED": "true"
      }
    }
  }
}
```

*If `tinymem` is not in PATH, replace `"command": "tinymem"` with `"command": "/absolute/path/to/tinymem"`.*

### 2. Usage (MCP)

Start Rush:
```bash
rush
```

Rush will now have access to tinyMem tools:
-   **Querying:** "What decisions did we make about the API structure?" (Rush calls `memory_query`)
-   **Writing:** "Remember that we decided to use Postgres for production." (Rush calls `memory_write`)
-   **Health:** "Check if the memory system is working." (Rush calls `memory_health`)

### 3. Advanced MCP Config

You can tune the MCP integration in `.crush.json`:

```json
{
  "mcp": {
    "tinymem": {
      "type": "stdio",
      "command": "tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "debug",
        "TINYMEM_RECALL_MAX_ITEMS": "20"
      },
      "disabled_tools": []
    }
  },
  "system_prompt_suffix": "\n\nUse tinyMem to check context before answering code questions."
}
```

---

## Option B: Proxy Mode

Use this mode if you want tinyMem to act as a transparent "middle-man," automatically injecting relevant memory context into every prompt before it reaches your LLM.

### 1. Configure tinyMem

Create a `config.toml` file (e.g., in your project root or home directory) to tell tinyMem where your actual LLM is running (e.g., Ollama, LM Studio, vLLM).

```toml
[agent]
contract = 'small'

[proxy]
port = 8080
base_url = "http://127.0.0.1:1234" # Your actual LLM backend (e.g., LM Studio, Ollama)

[llm]
model = "unsloth/rnj-1-instruct" # The model name your backend expects
timeout = 5000

[recall]
max_items = 10

[cove]
enabled = true
confidence_threshold = 0.6

[logging]
level = "info"
file = "tinymem.log"
```

### 2. Start tinyMem Proxy

Open a terminal and run:

```bash
tinymem proxy
```
*You should see logs indicating the proxy is listening on port 8080.*

### 3. Configure Crush

Edit your `.crush.json` to treat tinyMem as an OpenAI-compatible provider.

```json
{
  "providers": {
    "Ollama": {
      "name": "Ollama",
      "base_url": "http://localhost:8080/v1",
      "type": "openai-compat",
      "models": [
        {
          "name": "rnj",
          "id": "unsloth/rnj-1-instruct",
          "context_window": 256000,
          "default_max_tokens": 20000
        }
      ]
    }
  }
}
```

* **`base_url`**: Must point to tinyMem (`http://localhost:8080/v1`).
* **`id`**: Should match the model name configured in `config.toml` (or be compatible with your backend).

### 4. Usage (Proxy)

Start Rush:
```bash
rush
```

In this mode, interaction is implicit:
-   **Context Injection:** When you ask a question, tinyMem automatically searches its database and prepends relevant memories to your prompt.
-   **Transparency:** Crush thinks it's talking directly to the LLM, but tinyMem is enriching the conversation in the background.

---

## Configuration Reference

For full configuration options, see [Configuration.md](Configuration.md).

## Troubleshooting

-   **Tools Not Available (MCP):** Check `rush --version` to ensure you have a version with MCP support.
-   **Connection Refused (Proxy):** Ensure `tinymem proxy` is running and the port matches your `.crush.json` configuration.
-   **Logs:** Check `.tinyMem/logs/` or the terminal where you ran `tinymem proxy` to see if requests are being processed.