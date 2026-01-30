# Claude CLI Notes

tinyMem's `proxy` mode is OpenAI-compatible. It is not a generic forward proxy and it does not speak the native Anthropic Claude API.

If your Claude CLI only talks to Anthropic endpoints, use tinyMem via MCP instead:
- Claude Desktop: `examples/mcp/claude-desktop/claude_desktop_config.json`
- Cursor: `examples/mcp/cursor/mcp.json`

If your Claude CLI supports MCP directly, you can register `tinymem` as an MCP server.

### Register `tinymem` with `claude mcp add`

Instead of hand-editing JSON, use the CLI command that Claude provides for managing MCP servers:

```bash
claude mcp add tinymem -- /absolute/path/to/tinymem mcp
```

The `--` marks the end of Claude CLI flags; everything after becomes the exact command Claude will launch. If you prefer a different name or need to pass extra flags, the command can look like this:

```bash
claude mcp add my-server -- /Users/you/bin/my-command --some-flag arg1
```

Once the command succeeds, `claude mcp list` will show the registered server, and you can open Claude’s configuration file if you want to adjust environment variables or the working directory.

Claude stores the following block in its JSON config (or paste it manually if you are editing the file yourself):

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "/absolute/path/to/tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_METRICS_ENABLED": "true",
        "TINYMEM_LOG_LEVEL": "debug"
      }
    }
  }
}
```

The `TINYMEM_METRICS_ENABLED` flag lets tinyMem emit recall metrics, and `TINYMEM_LOG_LEVEL=debug` keeps MCP logs verbose for troubleshooting.
