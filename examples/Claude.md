# tinyMem Claude Integration Guide

tinyMem connects to the Claude ecosystem primarily through the **Model Context Protocol (MCP)**, which is natively supported by Claude Desktop and Claude CLI.

## Quick Start: Which Mode?

| Tool | Recommended Mode | Why |
|------|------------------|-----|
| **Claude Desktop** | **MCP** | Native integration; allows Claude to query memory on demand. |
| **Claude CLI** (`claude`) | **MCP** | Native integration via `claude mcp` commands. |
| **Claude SDKs** | **Proxy** | If using Anthropic SDKs manually, standard MCP isn't automatic; consider Proxy or custom MCP client. |

---

## 1. Claude Desktop (MCP)

Claude Desktop can connect to tinyMem to read and write project memories.

### Configuration

1.  Locate your Claude Desktop configuration file:
    -   **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
    -   **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

2.  Add `tinymem` to the `mcpServers` section.

    **Basic Configuration:**
    ```json
    {
      "mcpServers": {
        "tinymem": {
          "command": "tinymem",
          "args": ["mcp"]
        }
      }
    }
    ```
    *(Ensure `tinymem` is in your system PATH. If not, use the absolute path, e.g., `/usr/local/bin/tinymem`)*

    **Advanced Configuration (Recommended):**
    ```json
    {
      "mcpServers": {
        "tinymem": {
          "command": "/usr/local/bin/tinymem",
          "args": ["mcp"],
          "env": {
            "TINYMEM_LOG_LEVEL": "info",
            "TINYMEM_METRICS_ENABLED": "true"
          }
        }
      }
    }
    ```

3.  Restart Claude Desktop.

### Usage
Once connected, you can ask Claude Desktop:
-   "What do we know about the database schema?"
-   "Remember that we are using Go 1.25."
-   "Check for any active tasks."

---

## 2. Claude CLI (MCP)

The `claude` CLI tool also supports MCP.

### Registration

You can register tinyMem with a single command:

```bash
claude mcp add tinymem -- tinymem mcp
```

Or with specific flags/env vars:

```bash
claude mcp add tinymem -- /usr/local/bin/tinymem mcp
```

### Verification

Check if it's running:
```bash
claude mcp list
```

### Usage

Start a session:
```bash
claude
```
In the chat:
> "Read the project memory and summarize recent decisions."

---

## 3. Configuration Reference

You can customize tinyMem's behavior via environment variables in your MCP config.

For a full list of configuration options, see [Configuration.md](Configuration.md).

| Variable | Description | Default |
|----------|-------------|---------|
| `TINYMEM_LOG_LEVEL` | Log verbosity (`debug`, `info`, `error`) | `info` |
| `TINYMEM_METRICS_ENABLED` | Track recall stats | `false` |
| `TINYMEM_RECALL_MAX_ITEMS` | Max memories to retrieve per query | `10` |

---

## Troubleshooting

-   **"command not found"**: Ensure `tinymem` is installed and in your PATH, or use the absolute path in the config.
-   **Connection Refused**: MCP runs over stdio, so network ports aren't usually the issue. Check `tinymem doctor` for internal health.
-   **No Memories Found**: Ensure you are launching Claude from the root of your project (where `.tinyMem/` resides), or that you have initialized memory with `tinymem health`.