# tinyMem + Cline Integration

[Cline](https://github.com/cline/cline) is a powerful autonomous coding agent for VS Code. It supports MCP, allowing it to read/write memory during tasks.

## Configuration

1.  **Install the Cline Extension** in VS Code.

2.  **Configure MCP via UI:**
    - Click the **MCP Servers** icon at the top navigation bar of the Cline pane
    - Select the **Configure** tab
    - Click the **"Configure MCP Servers"** button at the bottom

    This opens the `cline_mcp_settings.json` configuration file.

3.  **Add tinyMem** to the configuration:

    ```json
    {
      "mcpServers": {
        "tinymem": {
          "command": "/usr/local/bin/tinymem",
          "args": ["mcp"]
        }
      }
    }
    ```

    *Tip: Ensure `tinymem` is in your system PATH, or use the absolute path as shown above.*

4.  **Save and restart VS Code** (or reload window).

For more details on Cline MCP configuration, see the [official Cline MCP documentation](https://docs.cline.bot/mcp/configuring-mcp-servers).

## Usage

Cline will now see `memory_query`, `memory_write`, etc., as available tools.

> **Prompt:** "Read the memory to understand how we handle authentication, then implement the login route."

Cline will call `memory_query` first, then proceed with the task using that context.
