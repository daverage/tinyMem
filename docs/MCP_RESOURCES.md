# Ensuring `list_mcp_resources` finds tinyMem

The `list_mcp_resources` lookup that the Codex agent runs before mutating a repository pulls from the shared automation registry (`$CODEX_HOME/automations/*`). In this workspace we only ship the tinyMem code itself, so the registry is empty and the agent cannot find the MCP tool that knows how to validate mutations.

To keep agents honest when they act in this repo, add a tinyMem automation definition that registers the `tinymem mcp` server (and any helper commands) so `list_mcp_resources` returns a non-empty list.

## How to register tinyMem in the automation registry

1. Locate your Codex automation directory. By default it’s `$CODEX_HOME/automations/`; if that directory does not exist yet, create it.
2. Inside `automations/`, add a folder for this repo (for example, `tinymem`). Place a `automation.toml` file there.
3. Populate `automation.toml` with metadata that exposes the MCP entry point. A minimal example:

   ```toml
   title = "tinyMem MCP"
   description = "Exposes the tinyMem MCP server and diagnostics helpers for repository work."

   [server]
   command = "tinymem"
   args = ["mcp"]
   timeout = 60000
   trust = false

   [[tools]]
   name = "tinyMem health"
   description = "Verify the database before editing."
   command = "tinymem"
   args = ["health"]
   timeout = 15000

   [[tools]]
   name = "tinyMem doctor"
   description = "Run the doctor diagnostic from MCP context."
   command = "tinymem"
   args = ["doctor"]
   timeout = 30000
   ```

   Adjust the snippet as needed for your environment; the registry that backs `list_mcp_resources` may support additional fields such as `env`, `cwd`, or `trust`.

4. After the automation file is saved, restart the Codex agent (if necessary) so it notices the new resource. Running `list_mcp_resources` again should now show your tinyMem MCP server and any helper tools.

## Why this matters

- The root `docs/agents/AGENTS.md` contract mandates that repository mutations happen via the MCP tools; registering `tinymem` here keeps codex obeying that contract.
- Once `list_mcp_resources` returns our entry, we can use the MCP server to call `memory_query`, `memory_recent`, `memory_write`, etc., without manually editing `tinyTasks.md`.
- The automation can bundle other useful wrappers (`health`, `doctor`, `query`) so agents have quick diagnostics at hand before mutating files.

If you need a hand generating the TOML for your automation tool, look for other Codex automation directories on this machine (e.g., `~/.codex/automations/`) for examples, or copy the snippet above and adjust the tool metadata to match.
