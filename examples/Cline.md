# tinyMem + Cline Integration

[Cline](https://github.com/cline/cline) is a powerful autonomous coding agent for VS Code. It supports MCP, allowing it to read/write memory during tasks.

## Configuration

1.  **Install the Cline Extension** in VS Code.
2.  **Configure MCP:**
    Create or edit `~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/mcp.json` (Mac) or `%APPDATA%\Code\User\globalStorage\saoudrizwan.claude-dev\settings\mcp.json` (Windows).
    
    *Note: The path might vary. Check Cline's settings for "MCP Servers" button to open the config file directly.*

3.  Add tinyMem:

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

4.  **Restart VS Code** (or reload window).

## Usage

Cline will now see `memory_query`, `memory_write`, etc., as available tools.

> **Prompt:** "Read the memory to understand how we handle authentication, then implement the login route."

Cline will call `memory_query` first, then proceed with the task using that context.
