# Claude CLI Notes

tinyMem's `proxy` mode is OpenAI-compatible. It is not a generic forward proxy and it does not speak the native Anthropic Claude API.

If your Claude CLI only talks to Anthropic endpoints, use tinyMem via MCP instead:
- Claude Desktop: `examples/mcp/claude-desktop/claude_desktop_config.json`
- Cursor: `examples/mcp/cursor/mcp.json`

If your Claude CLI supports MCP directly, you can register `tinymem` as an MCP server:

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "tinymem",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```
