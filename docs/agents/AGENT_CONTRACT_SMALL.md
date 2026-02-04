**Start of tinyMem Protocol**
You operate under TinyMem governance. This protocol is MANDATORY.

### 1. The Workflow
1. **Observe:** Call `memory_query` or `memory_recent` before any mutation to avoid duplicates or conflict.
2. **Sequence:** If a task has 2+ steps, externalize them to `tinyTasks.md` (use `task_add`).
3. **Mutate:** Use standard tools to edit files. Never assume state.
4. **Persist:** Call `memory_write` immediately after a successful mutation. **NEVER** claim a change until this tool succeeds.

### 2. Guardrails
- **Fail Closed:** If a tool fails or state is ambiguous, STOP and ask the user.
- **Tool Authority:** Prefer MCP tools (`memory_*`) over CLI commands when available.
- **Evidence:** Facts require evidence; decisions require rationale.

### 3. Tool Reference (MCP)
- `memory_query`: Search contextual memory. **Required** before mutations.
- `memory_recent`: Fetch latest entries for immediate context.
- `memory_write`: Persist facts/decisions. **Required** after mutations.
- `task_add`: Add/Update `tinyTasks.md` entries. **Required** for multi-step work.

### 4. Tool Reference (CLI)
- `tinymem health/doctor`: Verify DB/Config before starting work.
- `tinymem stats/dashboard`: Check progress and health metrics.
- `tinymem write/query/recent`: CLI equivalents of MCP tools.

**End of tinyMem Protocol**