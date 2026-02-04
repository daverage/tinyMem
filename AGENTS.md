**Start of tinyMem Protocol**

# TINYMEM AGENT CONTRACT (Governed — Task-Externalised)

This contract governs all repository-related behavior when tinyMem is present.
Non-compliance invalidates the response.

---

## 0. Scope

A request is **repository-related** if it touches:

* code
* files
* documentation
* configuration
* architecture
* tasks
* planning
* repository state

---

## 1. Core Principle

Observation is free.
Sequencing is authority.
Mutation is explicit.

---

## 2. Tool Definitions (Authoritative)

### Memory Recall

* `memory_query`
* `memory_recent`

Available in ALL modes (implementing "Observation is free").
Required before any mutation in GUARDED/STRICT modes.

### Intent Declaration

* `memory_set_mode`

Required before any mutation.

### Memory Write

* `memory_write`

The **only** permitted mechanism for durable memory.

### Task Authority

* `tinyTasks.md` in the project root
* Optional task-authority helper tool

---

## 3. Definitions

### Observation

Reading, inspecting, analyzing, summarizing, or asking questions.

### Mutation

Any durable state change, including:

* writing or modifying files
* creating, updating, or completing tasks
* writing memory
* promoting a claim to a fact, decision, or constraint

### Task Authority

`tinyTasks.md` is the single source of truth for task state.
Task state must never be inferred.

### Task Identification

The moment the agent identifies, implies, or sequences more than one actionable step.

This includes:

* plans
* approaches
* checklists
* ordered bullets
* “first / then / next”
* step-by-step reasoning

---

## 4. Modes (Intent)

You operate in exactly one mode:

* **PASSIVE** — observation only
* **GUARDED** — bounded, reversible mutation
* **STRICT** — maximum caution, full enforcement

Mode MUST be declared via `memory_set_mode` before mutation.

---

## 5. Rule Set (Stable IDs)

### R1 — Recall Before Mutation

Memory recall tools (`memory_query`, `memory_recent`) are available in ALL modes (implementing "Observation is free").

Before any mutation in GUARDED/STRICT modes, you MUST:

* call `memory_query` or `memory_recent`
* acknowledge the result (even if empty)

---

### R2 — Task Externalisation Is Mandatory

The agent may NOT hold a task list internally.

If **Task Identification** occurs:

1. All steps MUST be externalised into `tinyTasks.md`
2. No mutation may occur until task authority is resolved

If `tinyTasks.md` does NOT exist:

* Create the inert template
* Populate it with a proposed task list
* STOP
* Request the human to review, edit, reorder, or approve the proposed tasks

Creation or population of `tinyTasks.md` does NOT authorize work.

Planning in the response body is prohibited once this rule triggers.

#### Task Proposal Allowance

The agent MAY populate `tinyTasks.md` with a proposed task list.

Proposed tasks are NOT authorized until a human:
- confirms them explicitly, or
- edits or reorders them, or
- states approval in plain language

The agent MUST stop after proposing tasks and wait for human authorization.

---

### R3 — Tasks Are Authoritative

If `tinyTasks.md` exists:

* Continue the **first unchecked subtask**
* If no unchecked subtasks exist, STOP and request user input

The agent may NOT:

* skip tasks
* reorder tasks
* redefine tasks
* invent progress

---

### R4 — Mutation Requires Intent

Before any mutation, ALL of the following MUST be true, in order:

1. R1 satisfied (memory recall in GUARDED/STRICT modes)
2. Intent declared via `memory_set_mode`
3. R2 satisfied (task externalised if required)
4. R3 satisfied (task authority confirmed)

---

### R5 — Durable Memory Is Tool-Only

* Use `memory_write` only
* Facts require evidence
* Decisions and constraints require rationale
* Never claim a memory write unless the tool succeeded

---

### R6 — Fail Closed

If recall, intent, task authority, or enforcement is uncertain:

* Continue with observation only, OR
* STOP and request user input

Never guess. Never proceed optimistically.

---

## 6. tinyTasks.md Templates

### Inert Auto-Creation Template

```md
# Tasks — PROPOSED
>
> These tasks were proposed by the agent.
> No work is authorised until a human reviews and confirms them.
>
## Tasks
<!-- No tasks defined yet -->
```

### Active Task Structure

```md
# Tasks – <Goal>

- [ ] Top-level task
  - [ ] Atomic subtask
    - [ ] Task definition and details
```

Rules:

* Two levels only
* Order matters
* Unchecked == authorized *after human confirmation*

---

## 7. Enforcement Expectations

Expected to be enforceable at the boundary:

* block mutation without recall
* block mutation without intent
* block mutation when tasks are required but missing
* block mutation when tasks exist but none are unchecked
* track violations for audit

If enforcement is unavailable, self-enforce and fail closed (R6).

---

## 8. Error Handling

If a required tool fails:

1. State the failure
2. Retry up to 2 times
3. STOP and request human intervention

---

## 9. End-of-Response Checklist (When Mutation Occurs)

Confirm explicitly:

* recall completed in GUARDED/STRICT modes (R1)
* mode declared (R4)
* task authority resolved (R2, R3)
* memory writes completed or not required (R5)

Do not restate this contract.

---

## 10. Server-Governed Prompts (Phase 1–5)

tinyMem now runs a fixed series of server-controlled prompts so agent output becomes an intent ledger, not an execution trace.

### Prompt 1 — TaskManager Ownership
* `tinyTasks.md` is the server's sole task ledger; the LLM may not read or write it directly.
* All task mutations flow through the shared TaskManager, which loads/parses the file, validates structure, and exposes add, update, complete, and list operations.
* MCP and proxy mode call the same TaskManager path, so any other file access to `tinyTasks.md` is rejected.
*Implementation evidence: `internal/tasks/manager.go` implements the TaskManager APIs and both `internal/server/mcp/server.go` (lines 90-106) and `internal/server/proxy/server.go` (lines 81-110) instantiate that shared manager so every mutation path is server-owned.

### Prompt 2 — Intent Interpretation
* Each tool call or proxy mutation maps to exactly one intent category (file_write, task_update, memory_write, diagnostics, mode declaration, etc.) so the server always knows what the LLM is formally asking for.
* The server now exposes machine-readable intent metadata (category, minimum mode, recall requirement, scope, and side effects) for every tool so both MCP and proxy layers load the same contract instead of inferring intent from prose.
* Validation uses that metadata and the shared intent gate (`ensureIntent`) to confirm the declared category exists, the requested mode meets the minimum, recall/authority/evidence prerequisites are satisfied, and any scope constraints (e.g., fact writes needing evidence or tinyTasks edits needing strict mode) hold; failed validation rejects the request with zero side effects.
* MCP and proxy both consult this registry, so no mode-determining logic lives in prompts—the LLM is treated as making intent declarations, not executing actions.
*Implementation evidence: `internal/intent/definition.go` defines every tool's metadata, `internal/server/tool_definitions.go` attaches it to each MCP tool, and `internal/server/mcp/server.go#ensureIntent` validates category, mode, and recall before every tool executes, so intent is derived from metadata, not agent prose.

### Prompt 3 — Unified Enforcement
* All mutating requests—MCP tool calls and proxy mutations alike—flow through a single enforcement gate.
* Enforcement decisions are deterministic, policy-driven, and executed on the server; prompt text is advisory, not authoritative.
*Implementation evidence: `internal/server/mcp/server.go#ensureIntent`, `internal/execution/controller.go`, and `internal/enforcement/recorder.go` record mode compliance and enforcement events, and proxy mode reuses the same `execution.Controller`, so MCP and proxy share one deterministic gate.

### Prompt 4 — Memory Governance
* Agents submit structured memory proposals; the server decides what gets persisted.
* The server validates recall/evidence/duplication rules, then assigns IDs, timestamps, and provenance before writing.
* No memory write occurs unless all prerequisites are satisfied.
*Implementation evidence: `internal/server/mcp/server.go#handleMemoryWrite` parses the structured JSON proposal, enforces recall/mode/evidence via `requireMode`/`ensureRecallBeforeMutation`, and only then persists to `memory.Service`, guaranteeing the server owns every memory mutation.

### Prompt 5 — Metadata as Protocol
* Every tool carries machine-readable metadata that states its intent category, side effects, prerequisites, and allowed scope.
* Enforcement consumes that metadata directly, not prose, so behavior is deterministic and auditable.
* Tool descriptions stay short and focus on capabilities rather than policy.
*Implementation evidence: `intent.Definition.Metadata` plus `server.ToolMetadata` publish the machine-readable intent schema, so enforcement consumes structured metadata while the tool descriptions in `internal/server/tool_definitions.go` remain concise.

Together these prompts guarantee that `tinyTasks.md` cannot be modified by the LLM directly, no mutation occurs without server validation, MCP and proxy enforcement behave identically, gating is policy rather than conversation state, prompts can be deleted without breaking safety, and hallucinated success claims remain inert.


**End of tinyMem Protocol**
