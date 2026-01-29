# Configuring Zed IDE with tinyMem (MCP)

Zed supports connecting to MCP servers via `context_servers`. This lets Zed use `tinymem` as an MCP server (stdio).

## Register tinyMem through `context_servers`

Add or merge an entry like the following into your Zed `settings.json`:

```json
{
  "context_servers": {
    "tinymem": {
      "command": "/path/to/tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "info"
      }
    }
  }
}
```

The `command` value should point to the `tinymem` binary (absolute path, or a command on your `PATH`). `args` must include `["mcp"]` so the binary runs in MCP server mode.

## Proxy mode (when you want an OpenAI-compatible API base)

If you want Zed (or a Zed extension/tooling) to send OpenAI-compatible requests through tinyMem's proxy instead, point the client at `http://localhost:8080/v1` (or your configured port) and keep your actual LLM backend configured inside `.tinyMem/config.toml`.
