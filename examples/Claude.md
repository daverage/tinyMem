# tinyMem Claude Integration Guide

> [!IMPORTANT]
> This guide covers both **Claude Desktop** and **Claude Code (CLI)**. 
> While both support MCP, the configuration methods for environment variables differ.

tinyMem connects to the Claude ecosystem primarily through the **Model Context Protocol (MCP)**, which is natively supported by Claude Desktop and Claude Code.

## Quick Start: Which Mode?

| Tool | Recommended Mode | Why |
|------|------------------|-----|
| **Claude Desktop** | **MCP** | Native integration; allows Claude to query memory on demand. |
| **Claude Code** (`claude`) | **MCP** | Native CLI integration via `claude mcp` commands. |
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

---

## 2. Claude Code CLI (MCP)

The `claude` CLI tool (Claude Code) supports MCP and provides a command-line interface for adding servers.

### Registration

You can register tinyMem using the `claude mcp add` command.

**Method 1: Using `--env` flags (Claude Code CLI only)**
This is the recommended way to pass configuration to tinyMem in the CLI.

```bash
claude mcp add tinymem \
  --env TINYMEM_LOG_LEVEL=debug \
  --env TINYMEM_METRICS_ENABLED=true \
  -- tinymem mcp
```

**Method 2: Using shell exports**
Environment variables exported in your current shell session will be picked up when you add the server.

```bash
export TINYMEM_LOG_LEVEL=debug
claude mcp add tinymem -- tinymem mcp
```

**Method 3: Manual JSON editing**
You can also manually edit the `claude_desktop_config.json` (which Claude Code also uses for its MCP settings).

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