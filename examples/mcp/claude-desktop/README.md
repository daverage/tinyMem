# Claude Desktop MCP Example

This folder contains an example `claude_desktop_config.json` snippet for registering `tinymem` as an MCP server.

How to use:
1. Find your Claude Desktop config file (platform-specific location per Anthropic docs).
2. Merge the `mcpServers.tinymem` entry from `claude_desktop_config.json` into your config.
3. Replace `/absolute/path/to/tinymem` with your actual binary path (or use `tinymem` if it is on your `PATH`).
4. Restart Claude Desktop.

Notes:
- `type: "stdio"` is the typical MCP server mode for local executables.
