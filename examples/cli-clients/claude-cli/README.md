# Configuring Claude CLI (Claude Code) with tinyMem

This guide explains how to configure Anthropic's Claude CLI (often referred to as "Claude Code") to use `tinyMem` as a proxy. `tinyMem` acts as an HTTP proxy, enabling it to inject context and manage memory for your AI interactions with Claude.

Before you begin, ensure `tinyMem` is running in proxy mode (e.g., `tinymem proxy`) and listening on `http://localhost:8080` (or your configured port).

## Using Environment Variables (Recommended)

Claude Code respects standard proxy environment variables. The simplest way to direct its traffic through `tinyMem` is to set `HTTPS_PROXY`.

```bash
# Set the HTTPS_PROXY environment variable to tinyMem's address
export HTTPS_PROXY="http://localhost:8080"

# (Optional) If you have hosts that should bypass the proxy, use NO_PROXY
# export NO_PROXY="localhost,127.0.0.1,api.anthropic.com" # You might need to adjust NO_PROXY

# Now, when you run Claude Code, its API requests will go through tinyMem
# Example:
# claude-code

# To unset the proxy:
# unset HTTPS_PROXY
# unset NO_PROXY
```
Anthropic's corporate proxy documentation (November 2025) reiterates that Claude Code expects an `http://` URL even when you set `HTTPS_PROXY` and currently ignores `NO_PROXY`, so clearing the proxy environment variables is the only reliable way to bypass tinyMem for specific hosts.

**Important Note:** Claude Code generally expects an `http://` schema even for `HTTPS_PROXY` when connecting to a simple HTTP proxy like `tinyMem`.

## Using the In-App `/config` Command

If you are using the interactive REPL of Claude Code, you can configure network settings using the `/config` command.

1.  Start Claude Code:
    ```bash
    claude-code
    ```
2.  In the Claude Code REPL, type `/config` and press Enter.
3.  This will open a settings interface. Look for `network.httpProxyPort` and set it to `8080` (or your `tinyMem` proxy port).
4.  Save and exit the configuration.

Anthropic's Claude Code settings documentation (late 2025) describes `/config` as the entry point for changing `network.httpProxyPort`, so following these steps matches the way the app exposes proxy settings today.

## MCP Integration (for supported Claude CLI versions)

Some versions or variants of Claude CLI might support MCP integration through a JSON configuration file, similar to Claude Desktop/Cursor. If your CLI uses a `claude_config.json` or similar file, you can configure `tinyMem` as an MCP server.

**Example `claude_config.json`:**
```json
{
  "mcpServers": {
    "tinymem": {
      "command": "tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "info"
      }
    }
  }
}
```

**With CoVe disabled:**
```json
{
  "mcpServers": {
    "tinymem": {
      "command": "tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "info",
        "TINYMEM_COVE_ENABLED": "false"
      }
    }
  }
}
```
**How to use:**
1.  Find your Claude CLI's configuration directory.
2.  Locate the JSON configuration file for MCP servers.
3.  Add the `tinymem` server configuration to the `mcpServers` object.
4.  Ensure `tinymem` is in your system's PATH, or provide the absolute path to the executable in the `command` field.

## Considerations

*   **`NO_PROXY`:** Be aware that Claude Code has had reported issues honoring the `NO_PROXY` environment variable. If you experience issues, you might need to adjust your `NO_PROXY` settings or temporarily clear proxy environment variables when interacting with services that shouldn't go through `tinyMem`.
*   **Authentication:** If your `tinyMem` setup requires authentication (which it typically doesn't by default), you would need to include credentials in the proxy URL (e.g., `export HTTPS_PROXY="http://username:password@localhost:8080"`). However, for a standard `tinyMem` setup, this is not usually necessary.

Choose the configuration method that best fits your workflow.
