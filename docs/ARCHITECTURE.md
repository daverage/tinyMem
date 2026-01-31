# tinyMem Architecture

## Executive Summary

tinyMem is a multi-modal persistent memory system tailored to large language models. It combines a proxy front-end, Model Context Protocol (MCP) server, CLI tooling, scheduled background services, and a SQLite-backed evidence store to keep conversations honest, traceable, and actionable over long-lived workflows.

## Deployment Topology

```
               +-------------------------+
               |      LLM / Agent         |
               +-----------+-------------+
                           |
                 Proxy Mode |  MCP Mode (stdin/stdout)
                           ▼
   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
   │  Proxy Server │   │  MCP Server  │   │  CLI / Tools │
   │  (/v1/chat)   │   │  (tinyMem)   │   │  (tinymem,   │
   │               │   │              │   │   dashboards)
   └───────┬──────┘   └──────┬───────┘   └──────────────┘
           │                 │                  │
           ▼                 ▼                  ▼
   ┌──────────────────────────────────────────────┐
   │             tinyMem Application               │
   │  - CoreModule   - ProjectModule   - ServerModule │
   │  - Service Layer (Memory, Evidence, Recall)   │
   └──────────────────┬───────────────────────────┘
                      ▼
             +------------------+
             | SQLite Memory DB |
             +------------------+
                      ▼
              +---------------+
              | tinyTasks.md  |
              | (ledger file) |
              +---------------+
```

Proxy and MCP transports enqueue requests into the same core process, which routes them through shared services and the SQLite store. The CLI tools and dashboards hook into the same API surface, exposing diagnostics, `tinyTasks` synchronization, and manual memory operations.

## Core Modules and Service Layer

- **CoreModule**: Loads configuration, bootstraps structured logging, and opens the SQLite database used for persistent memories and task state.
- **ProjectModule**: Knows the project root, project ID, and file-system layout so services can locate `tinyTasks.md`, sample data, and per-project metadata.
- **ServerModule**: Understands the execution mode (proxy, MCP, standalone) and wires transports to MCP tool handlers, health endpoints, and the Ralph repair loop.

The service layer sits atop the modules:

- **Memory Service** stores, indexes, and deletes memories while conforming to truth states (verified/asserted/tentative) and allows custom recall tiers.
- **Evidence Service** ensures evidence predicates are evaluated before admitting facts, constraining claims, or closing a Ralph loop.
- **Recall Engine** decides which memories to inject back into prompts based on tiers, tokens, and freshness heuristics.

## Memory Pipeline and Persistence

1. **Ingestion** – Agents issue `memory_write` or CLI commands, specifying type (constraint, plan, decision, observation, note, task, claim). Facts require external evidence references while other types may be more loosely defined.
2. **Evidence gating** – Evidence predicates such as `file_exists`, `cmd_exit0`, or `test_pass` are checked before a discovery is marked `verified`. Unverified assertions stay `asserted` or `tentative` until more proof arrives.
3. **Storage** – SQLite tables store the memory payload, metadata, recall tier, and hash for deduplication. The same store backs tinyTasks-derived state so everything stays in lockstep.
4. **Recall** – The recall engine filters memories by tier (`always`, `contextual`, `opportunistic`), truth state, workspace tags, and temporal relevance before injecting them into responses.
5. **Verification & Updates** – The CLI and MCP expose `memory_stats`, `memory_health`, and `memory_doctor` so operators can inspect database health, run diagnostics, or re-verify stored entries.

## tinyTasks System

`tinyTasks.md` remains the single source of truth for work. tinyMem keeps this ledger synchronized with memory entries and enforces strict intent semantics.

```
┌────────────────────────────────────────────────────────────┐
│ tinyTasks System                                           │
├──────────────┬──────────────┬──────────────┬───────────────┤
│ Parser       │ Task Memory  │ Synchronizer│ Guardrails    │
│ - Reads file │ - Mirrors    │ - Watches   │ - Intent      │
│ - Validates  │   unchecked  │   file      │   policies    │
└──────────────┴──────────────┴──────────────┴───────────────┘
```

### Intent semantics
- The system may automatically create `tinyTasks.md` when multi-step work is implied (multi-step requests, missing file detections, or task-related CLI/MCP commands). The auto-created file is intentionally inert and clearly marked `# Tasks — NOT STARTED` so no work is authorized yet.
- Real intent exists only when a human edits the file, replaces the title with a concrete goal, and adds unchecked task list entries (`- [ ]`). The earlier policy ensures deterministic behavior while keeping humans in control.
- When the file contains unchecked human-authored tasks, tinyMem parses, validates, and surfaces them to memory commands; completed tasks (`- [x]`) are ignored unless re-opened.

### File features
- Format: hierarchical Markdown lists with GitHub-style checkboxes for top-level tasks and atomic subtasks.
- Synchronization: CLI/MCP commands read and write `tinyTasks.md`, regenerate memory entries, and drive agent workflows.
- Safety: Dormant unchecked tasks stay out of recall unless explicitly requested, preventing stale instructions from triggering new work.

## Ralph Mode (Autonomous Repair)

Ralph orchestrates evidence-gated repair loops when tasks require verification.

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│ Execute  │ -> │ Evidence │ -> │ Recall   │ -> │ Repair   │
│ Phase    │    │ Phase    │    │ Phase    │    │ Phase    │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
       ↑                                                    │
       └───────────────────── Loop control ──────────────────┘
```

- **Execute**: Runs diagnostic commands or tests (via `memory_ralph` API) capturing stdout/stderr.
- **Evidence**: Enforces predicates like `cmd_exit0`, `test_pass`, or `file_exists` before declaring success.
- **Recall**: Fetches relevant memories, including tinyTasks state, to inform repairs.
- **Repair**: Applies fixes (patches, file edits) while respecting forbidden paths, iteration caps, and human approval gates.

Safety measures include iteration limits, forbidden-path enforcement, diagnostics logging, and human gating for high-risk repairs.

## Tooling and Interfaces

tinyMem exposes capabilities through CLI commands, MCP tool calls, and the dashboard.

| Interface | Purpose |
| --- | --- |
| `tinymem` CLI | Launches commands such as `memory_query`, `memory_recent`, `memory_write`, `memory_stats`, `memory_doctor`, `dashboard`, `proxy`, and `mcp` for dev workflows. |
| MCP tools | `memory_query`, `memory_recent`, `memory_write`, `memory_stats`, `memory_health`, `memory_doctor`, `memory_ralph`, and others feed LLM agents. |
| Proxy endpoints | `/v1/chat/completions` (OpenAI-compatible) forwards requests through tinyMem to enforce recall and filtering. |
| Dashboard | Visualizes memory state, tinyTasks progress, and health signals to operators. |

All interfaces observe the same architecture, retrieving memories, verifying evidence, and updating tinyTasks, so every agent or operator experiences a consistent view of project state.

## Operational Notes

- Versioned container images (e.g., GHCR) package the tinyMem binary along with schema migrations and dashboards.
- Diagnostics (`tinyMem doctor`, `memory_doctor`) report on schema health, memory integrity, and task sync status.
- The architecture ensures persistence, transparency, and evidence-backed memory retrieval for long-lived multi-agent workflows.
