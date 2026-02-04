Prompt 1 — Make the server the sole authority for tinyTasks.md

Purpose:
Eliminate all LLM-driven file editing of tinyTasks.md. Tasks become server-managed state.

You are modifying the existing TinyMem server codebase.

OBJECTIVE (NON-NEGOTIABLE):
The server must be the sole authority over tinyTasks.md.

CURRENT STATE:
- The LLM may read or write tinyTasks.md via file tools.
- Task discipline partially relies on prompts and agent behavior.

TARGET STATE:
1. The LLM must never read or write tinyTasks.md directly.
2. All task mutations must go through exactly one server-owned code path.
3. MCP and proxy mode must both use this same path.
4. Any remaining direct file access to tinyTasks.md is a bug.

REQUIRED CHANGES:
- Introduce a TaskManager (or equivalent) that:
  - loads and parses tinyTasks.md
  - validates task structure
  - exposes explicit operations: add, update, complete, list
  - writes back deterministically
- Replace all existing tinyTasks.md read/write logic with TaskManager calls.
- Block or remove any file_write capability targeting tinyTasks.md.

CONSTRAINTS:
- Do NOT change the tinyTasks.md format unless strictly required.
- Do NOT rely on prompts or agent instructions for correctness.
- Do NOT leave old mutation paths in place.

VALIDATION:
- Identify all previous task mutation paths.
- Show how each is removed, blocked, or routed through TaskManager.

DELIVERABLE:
- Updated server code
- Explanation of what was replaced vs removed

Prompt 2 — Interpret LLM output as intent, not execution

Purpose:
Formalize how the server understands what the LLM is “trying to do”.

You are updating the TinyMem server request handling logic.

OBJECTIVE:
Treat all LLM-originated actions as intent declarations, never as execution.

CURRENT STATE:
- The LLM selects tools and indirectly executes mutations.
- Some intent semantics are implicit or prompt-based.

TARGET STATE:
1. The server interprets intent solely from:
   - MCP tool calls (in MCP mode)
   - structured proxy requests (in proxy mode)
2. Free-text output is never authoritative.
3. Execution only occurs after server validation.

REQUIRED CHANGES:
- Define a small, closed set of intent categories (file_write, task_update, memory_write, etc.).
- Map each tool or proxy request to exactly one intent category.
- Before execution:
  - validate intent category
  - validate scope
  - validate gating policy
  - validate prerequisites (recall, authority, evidence)
- If validation fails, reject with no side effects.

CONSTRAINTS:
- Do NOT infer intent from raw text.
- Do NOT trust the LLM to sequence correctly.
- Do NOT allow partial execution.

DELIVERABLE:
- Updated intent interpretation flow
- Code showing where intent is validated and enforced

Prompt 3 — Unify MCP and proxy enforcement (single gate)

Purpose:
Ensure gating logic lives in one place and applies everywhere.

You are refactoring the TinyMem server to unify enforcement logic.

OBJECTIVE:
MCP mode and proxy mode must share the same enforcement rules.

CURRENT STATE:
- MCP enforces rules at the tool boundary.
- Proxy mode relies partially on prompts and agent discipline.

TARGET STATE:
1. All mutation requests flow through one enforcement layer.
2. Gating decisions are server-side and deterministic.
3. Prompts are advisory only.

REQUIRED CHANGES:
- Extract a shared Enforcement / Validator component.
- Route:
  - MCP tool calls
  - proxy mutation requests
  through this component.
- Remove or bypass any proxy-only enforcement that duplicates logic.
- Ensure rejection behavior is identical in both modes.

CONSTRAINTS:
- Do NOT add new prompt text.
- Do NOT duplicate validation logic.
- Do NOT change client behavior unless necessary.

VALIDATION:
- Show how MCP and proxy reach the same enforcement code.
- Show that bypass paths no longer exist.

DELIVERABLE:
- Refactored server code
- Explanation of removed vs reused logic

Prompt 4 — Server-owned memory governance (upgrade, not rewrite)

Purpose:
Make memory a true server-side ledger.

You are upgrading TinyMem’s memory handling.

OBJECTIVE:
The server must be the sole writer of durable memory.

CURRENT STATE:
- The LLM proposes memory and may implicitly “write” it.
- Some rules are enforced via prompts.

TARGET STATE:
1. The LLM can only propose memory candidates.
2. The server decides whether memory is written.
3. The server assigns IDs, timestamps, and provenance.
4. Memory writes are blocked if prerequisites are not met.

REQUIRED CHANGES:
- Define a structured MemoryProposal format.
- Validate proposals against:
  - recall requirements
  - evidence requirements
  - duplication rules
- Persist memory only after validation.
- Return explicit confirmation on success.

CONSTRAINTS:
- Do NOT rely on prompt discipline.
- Do NOT store raw LLM output unvalidated.
- Do NOT allow silent memory writes.

DELIVERABLE:
- Updated memory handling code
- Description of old vs new flow

Prompt 5 — MCP metadata as protocol, not documentation

Purpose:
Push governance into machine-readable metadata.

You are refining TinyMem MCP tool metadata.

OBJECTIVE:
Tool metadata must encode protocol constraints, not explanations.

CURRENT STATE:
- Metadata may rely on descriptive text.
- Some rules are implicit.

TARGET STATE:
1. Each tool declares:
   - intent category
   - side effects
   - required prerequisites
   - allowed scope
2. Metadata is machine-readable and enforceable.
3. Descriptions are minimal and non-philosophical.

REQUIRED CHANGES:
- Define a metadata schema used by the server.
- Update existing tools to conform.
- Ensure the server uses metadata during validation.

CONSTRAINTS:
- Do NOT encode policy in prose.
- Do NOT duplicate enforcement logic in descriptions.

DELIVERABLE:
- Updated tool metadata
- Explanation of how metadata is consumed by enforcement

How you know these prompts succeeded

After Codex runs them, all of the following must be true:

tinyTasks.md cannot be modified by the LLM directly

No mutation occurs without server validation

MCP and proxy behave identically for enforcement

Gating is policy, not conversation state

Prompts can be deleted without breaking safety

Hallucinated success claims are inert

If any of those are false, the implementation is incomplete.

Final grounding statement

These prompts do not teach behavior.
They restructure authority.

That’s why they work, and why they don’t burn context.
