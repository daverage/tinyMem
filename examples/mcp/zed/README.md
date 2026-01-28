# Configuring Zed IDE with tinyMem (MCP)

Zed IDE exposes an `agent_servers` section in its configuration that can launch external executables in addition to the built-in LLM providers, so you can now register `tinyMem` directly as an MCP-aware agent. The Zed documentation (late 2025) describes this `agent_servers` structure, including `command`, `args`, and optional `env` blocks for each entry.

## Register tinyMem through `agent_servers`

Add or merge an entry like the following into the `agent_servers` map inside your Zed `settings.json` or the Agent Panel configuration file:

```json
{
  "agent_servers": {
    "tinymem_mcp": {
      "command": "/path/to/tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "info"
      }
    }
  }
}
```

The `command` value should point to the `tinymem` binary (absolute path or a directory on your `PATH`), and `args` must include `["mcp"]` so the binary runs in MCP server mode. You can also set any necessary environment variables inside `env`, just like other MCP integrations documented throughout this repo.

## When agent_servers is not an option

If your Zed build still exposes only custom API base URLs (e.g., inside `settings.json`'s `language_models` entries), you can send the provider requests through tinyMem's proxy instead. Point the provider's API base URL at `http://localhost:8080/v1` (or whatever port you choose) and keep your actual LLM backend configured inside `tinyMem`'s `config.toml`.

## Where to find updates

Monitor the Zed release notes and configuration docs for new MCP/agent-server capabilities (`agent_servers`, `context_servers`, etc.), and follow the same procedure above whenever a new mechanism is added.
