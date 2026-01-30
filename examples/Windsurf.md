# tinyMem + Windsurf Integration

[Windsurf](https://codeium.com/windsurf) is an AI-powered IDE by Codeium. It supports the **Model Context Protocol (MCP)** natively.

## Configuration

1.  **Install tinyMem** and ensure it's in your PATH.
2.  **Open Windsurf Settings**.
3.  Locate the **MCP Servers** configuration section (usually in `settings.json` or a dedicated MCP settings panel).
4.  Add tinyMem:

    ```json
    {
      "mcpServers": {
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

5.  **Restart Windsurf**.

## Usage

Windsurf's "Cascade" agent can now access your project memory.

> "Check project memory for our coding standards."

## Verification

If connected, Windsurf should list `tinymem` in its active context providers or tools list.
