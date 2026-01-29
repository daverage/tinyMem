# Gemini CLI Notes

tinyMem's `proxy` mode exposes an OpenAI-compatible API. Most Gemini clients use Google's native Gemini APIs, which are not OpenAI-compatible.

Use tinyMem in one of these ways:
- If your tool can talk to an OpenAI-compatible endpoint: point its API base URL to `http://localhost:8080/v1`.
- If your tool supports MCP: register `tinymem` as an MCP server and use `tinymem mcp`.

Example MCP server registration:

```json
{
  "mcpServers": {
    "tinymem": {
      "type": "stdio",
      "command": "tinymem",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```
