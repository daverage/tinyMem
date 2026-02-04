# Exposing tinyMem MCP tools to Codex

Codex discovers MCP servers from config files — there is no separate automation
registry or resource-listing step.  Two scopes are supported:

| File | Scope |
|---|---|
| `~/.codex/config.toml` | global (all projects) |
| `.codex/config.toml` | project-scoped (can be checked into the repo) |

## Adding tinymem to the config

Add the following block to whichever config file applies:

```toml
[mcp_servers.tinymem]
command = "tinymem"
args = ["mcp"]
enabled = true
startup_timeout_sec = 15
```

That is the complete entry.  Codex starts the server on session launch and
exposes every tool the server advertises (`memory_query`, `memory_recent`,
`memory_write`, `task_add`, `artifact_create`, etc.) alongside its built-in
tools.

## Project-scoped setup (recommended for this repo)

Check a `.codex/config.toml` into the repo root with the block above.  Any
Codex session opened inside the repo will pick it up automatically — no
per-machine global config needed.

## CLI alternative

If you prefer not to hand-edit the file:

```bash
codex mcp add tinymem -- tinymem mcp
```

This appends the equivalent entry to `~/.codex/config.toml`.

## Filtering tools (optional)

If you only want a subset of the tools exposed, use `enabled_tools`:

```toml
[mcp_servers.tinymem]
command = "tinymem"
args = ["mcp"]
enabled = true
enabled_tools = ["memory_query", "memory_recent", "memory_write", "task_add"]
```

## Why the tools weren't showing up

The server was already running, but no config entry existed — Codex had no
way to know about it.  Once the config block above is in place, the tools
appear in the next session without any restart.
