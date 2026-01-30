# tinyMem IDE Integration Guide

tinyMem integrates with modern IDEs primarily through the **Model Context Protocol (MCP)** or via **Proxy Mode** for copilot-style extensions.

## 1. VS Code (via MCP Extension)

Currently, VS Code supports MCP through extensions like the **MCP Extension** or specific agent extensions that implement the protocol.

1.  Install an MCP-compatible extension.
2.  Configure the extension settings to register `tinymem`:
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

## 2. Cursor

Cursor supports MCP configuration via a dedicated file.

1.  Locate/Create your MCP config (check Cursor docs for latest location, typically in project settings or global settings).
2.  Add the configuration:
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
3.  Restart Cursor. You can now reference memory in Composer or Chat.

## 3. Zed

Zed supports "context servers" (MCP).

1.  Open Zed Settings (`cmd-,`).
2.  Add to `context_servers`:

    ```json
    {
      "context_servers": {
        "tinymem": {
          "command": "/usr/local/bin/tinymem",
          "args": ["mcp"],
          "env": {
            "TINYMEM_LOG_LEVEL": "info"
          }
        }
      }
    }
    ```
3.  Restart Zed.

## 4. Continue (VS Code Extension)

[Continue](https://continue.dev/) allows you to configure custom LLM providers. Use **Proxy Mode**.

1.  **Start Proxy:** `tinymem proxy`
2.  **Edit `config.json`** in `~/.continue/` (or `.continue/` in project):

    ```json
    {
      "models": [
        {
          "title": "tinyMem Proxy",
          "provider": "openai",
          "model": "AUTODETECT",
          "apiBase": "http://localhost:8080/v1",
          "apiKey": "dummy"
        }
      ]
    }
    ```
3.  Select "tinyMem Proxy" in the Continue dropdown.

## 5. Windsurf / Other Agents

Most agentic IDEs now support MCP. Look for "MCP Servers" or "Context Providers" in their settings and register the stdio command:

-   **Command:** `tinymem`
-   **Args:** `mcp`

## Configuration Reference

For full configuration options, see [Configuration.md](Configuration.md).

## Troubleshooting

-   **Tool Not Found:** Ensure you have restarted your IDE after editing configuration files.
-   **Path Issues:** If `tinymem` command is not found by the IDE, always use the absolute path (e.g., `/usr/local/bin/tinymem`).