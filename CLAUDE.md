**Start of tinyMem Protocol**
You operate under TinyMem governance.

If a request would create files, change files, tasks, or memory:
- use TinyMem MCP tools when available
- do not assume state
- do not claim changes unless a tool call succeeds

Tool calls are the only way to affect reality.
If a tool call is blocked or rejected, adapt and retry.
If MCP tools are not available in your context, proceed with standard file operations; server-side enforcement still applies.
Otherwise, respond normally.

**TinyMem tool reference**
- `tinymem health` — verify the database, schema, and connectivity before any mutation.
- `tinymem doctor` — run a deeper diagnostic that reports configuration or health issues with storage, recall, or authorities.
- `tinymem stats` — check current memory/task counts and pass/fail rates to decide whether to update or run additional tooling.
- `tinymem dashboard` — view or refresh the visual task/memory status when you need awareness of progress or blockers.
- `tinymem query "<term>"` — search the existing memory store for a term that might affect your next edit.
- `tinymem recent` — fetch the latest entries so you can reference what tinyMem just learned.
- `tinymem write --type <type>` — persist a decision, note, or claim once you have collated the facts.
- `tinymem mcp` — start the stdio server that accepts the `memory_*` tool calls described below.

- `memory_query` — search contextual memory; required before mutations only in GUARDED/STRICT modes. In PASSIVE mode, mutations proceed without prior recall.
- `memory_recent` — fetch the most recent memories if you need to reference immediately preceding work.
- `memory_write` — persist a fact/decision/note; never claim a repository change without this tool concluding successfully.
- `task_add` — create tinyTasks.md if missing and append a task in one call; use this instead of creating the file manually.

**End of tinyMem Protocol**
