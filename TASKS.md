# tinyMem vNext – Agentic AI Task List

This task list is written for an agentic AI or human engineer to implement tinyMem **end‑to‑end**, with no hidden assumptions. Tasks are ordered to preserve correctness, truth discipline, and streaming guarantees.

Each task includes **intent**, **implementation notes**, and **acceptance criteria**.

---

## 0. Global invariants (read before coding)

These invariants must hold everywhere in the codebase:

* Memory is **not gospel**. Model output is never trusted by default.
* All memory entries are **typed** (fact, claim, plan, decision, constraint, observation, note).
* **Facts require evidence**. No evidence → not a fact.
* Prompt injection is **deterministic and bounded**.
* Streaming is **mandatory** where supported.
* Everything is **project‑scoped** under `.tinyMem/`.
* Single standalone Go executable.

Violating any invariant is a correctness bug, not a feature gap.

---

## 1. Repository & build foundation

### 1.1 Create Go module and skeleton

**Intent**: Establish a stable structure that prevents architectural drift.

**Tasks**:

* Initialize Go module (Go 1.22+).
* Create directory structure:

  * `cmd/tinymem/`
  * `internal/config/`
  * `internal/logging/`
  * `internal/storage/`
  * `internal/memory/`
  * `internal/evidence/`
  * `internal/recall/`
  * `internal/semantic/`
  * `internal/inject/`
  * `internal/extract/`
  * `internal/llm/`
  * `internal/server/proxy/`
  * `internal/server/mcp/`
  * `internal/doctor/`

**Acceptance**:

* `go test ./...` runs cleanly.
* `tinymem --help` builds and runs.

---

## 2. Project root & `.tinyMem` lifecycle

### 2.1 Project root detection

**Intent**: Ensure all state is local and explicit.

**Tasks**:

* Detect project root as current working directory.
* Resolve `.tinyMem/` path relative to root.

### 2.2 Directory initialization

**Tasks**:

* On startup, ensure:

  * `.tinyMem/`
  * `.tinyMem/logs/` (if logging enabled)
  * `.tinyMem/run/`
* Never write outside `.tinyMem/`.

**Acceptance**:

* Deleting `.tinyMem/` and restarting recreates a clean state.

---

## 3. Configuration system

### 3.1 Default configuration

**Intent**: Tool must run with zero config.

**Tasks**:

* Define default config in code (proxy port, search enabled, streaming on).

### 3.2 Config loading & overrides

**Tasks**:

* Load `.tinyMem/config.toml` if present.
* Apply environment variable overrides.
* Validate config (ports, modes, incompatible flags).

**Acceptance**:

* Invalid config fails fast with human‑readable error.

---

## 4. SQLite storage & schema

### 4.1 Database creation & migrations

**Intent**: Truth persistence with inspectability.

**Tasks**:

* Create `.tinyMem/store.sqlite3`.
* Apply schema migrations automatically.

### 4.2 Core tables

**Tasks**:

* `memories` table with:

  * id, project_id, type, summary, detail
  * key (optional), source
  * created_at, updated_at
  * superseded_by
* `evidence` table linked to memories.
* FTS5 virtual table over summary/detail.

**Acceptance**:

* Insert + search via FTS works deterministically.

---

## 5. Memory truth model

### 5.1 Memory types & validation

**Intent**: Prevent lies from becoming institutional memory.

**Tasks**:

* Define enum for memory types.
* Implement validation rules:

  * fact requires evidence
  * plan/claim never auto‑promote
  * decision/constraint require confirmation or repo‑derived proof

### 5.2 Supersession handling

**Tasks**:

* Detect conflicts (key collision or explicit contradiction markers).
* Mark old memory as `superseded_by` new one.

**Acceptance**:

* Superseded memories are ignored during recall.

---

## 6. Evidence system (reality checks)

### 6.1 Evidence verification engine

**Intent**: Check reality locally without token cost.

**Tasks**:

* Implement verifiers:

  * `file_exists`
  * `grep_hit`
  * `cmd_exit0`
  * `test_pass`
* All verifiers run locally.

**Acceptance**:

* Evidence verification never calls an LLM.
* Failed evidence blocks fact promotion.

---

## 7. Recall engine (token governor)

### 7.1 Lexical recall (baseline)

**Intent**: Deterministic, debuggable memory recall.

**Tasks**:

* Build FTS query from user prompt.
* Rank by BM25.
* Always consider constraints and decisions.

### 7.2 Token budgeting

**Tasks**:

* Track approximate token count of injected memory.
* Enforce `max_items` and `max_tokens`.

**Acceptance**:

* Recall never exceeds configured limits.

---

## 8. Semantic recall (optional enhancer)

### 8.1 Embedding backend

**Intent**: Improve phrasing flexibility, not correctness.

**Tasks**:

* Implement embedding client (Ollama or local model).
* Cache embeddings per memory entry.

### 8.2 Vector storage & hybrid scoring

**Tasks**:

* Store vectors in SQLite (extension or fallback).
* Combine FTS + vector scores with weights.
* Fail gracefully if unavailable.

**Acceptance**:

* System remains fully functional with semantic disabled.

---

## 9. Prompt injection

### 9.1 Memory block renderer

**Intent**: Stable, small‑model‑friendly context.

**Tasks**:

* Render memory as bounded system message:

  * Explicit types
  * Evidence markers for facts

**Acceptance**:

* Injection format never changes dynamically.

---

## 10. LLM backend client

### 10.1 OpenAI‑compatible client

**Intent**: Maximize compatibility.

**Tasks**:

* Forward chat completions to upstream base URL.
* Support streaming and non‑streaming.

**Acceptance**:

* Works with Ollama / LM Studio OpenAI‑compatible APIs.

---

## 11. Proxy server (deterministic path)

### 11.1 HTTP proxy

**Intent**: Guarantee long‑memory behavior.

**Tasks**:

* Implement `/v1/chat/completions`.
* Intercept prompt → recall → inject → forward.
* Stream responses immediately.

### 11.2 Post‑response capture

**Tasks**:

* Maintain bounded rolling buffer.
* Trigger extraction after stream ends.

**Acceptance**:

* Large responses do not spike memory usage.

---

## 12. Automatic memory extraction

### 12.1 Candidate extraction

**Intent**: Capture useful context without trusting the model.

**Tasks**:

* Rule‑based extraction of claims, plans, decisions.
* Default all extracted items to non‑fact types.

### 12.2 Truth enforcement

**Tasks**:

* Apply validation + evidence checks.
* Store safely.

**Acceptance**:

* Model claims never become facts without evidence.

---

## 13. MCP server

### 13.1 MCP stdio lifecycle

**Intent**: Clean IDE integration.

**Tasks**:

* Implement MCP handshake.
* Expose tools:

  * memory.query
  * memory.recent
  * memory.write
  * memory.stats
  * memory.health
  * memory.doctor

### 13.2 MCP streaming

**Tasks**:

* Stream large responses incrementally.

**Acceptance**:

* No large JSON blobs returned by default.

---

## 14. Health, stats, doctor

### 14.1 Health checks

**Tasks**:

* DB connectivity
* FTS availability
* Semantic availability
* LLM backend reachability

### 14.2 Doctor diagnostics

**Tasks**:

* Filesystem permissions
* Port conflicts
* Streaming mode status

**Acceptance**:

* `tinymem doctor` explains failures clearly.

---

## 15. Logging & transparency

### 15.1 Logging system

**Intent**: Invisible when healthy, explicit when not.

**Tasks**:

* Log levels: off/error/warn/info/debug.
* File logging to `.tinyMem/logs/`.

### 15.2 CLI log controls

**Tasks**:

* `tinymem logs tail`
* `tinymem logs level`

---

## 16. CLI commands

**Tasks**:

* `tinymem proxy`
* `tinymem mcp`
* `tinymem run`
* `tinymem health`
* `tinymem stats`
* `tinymem doctor`
* `tinymem recent`
* `tinymem query`

**Acceptance**:

* All commands work from any project root.

---

## 17. Streaming enforcement (hard gate)

**Tasks**:

* Ensure proxy always streams when upstream supports it.
* Ensure MCP tools stream large outputs.
* Track streaming metrics.

**Acceptance**:

* No code path buffers full LLM responses unnecessarily.

---

## 18. Cross‑platform build & smoke tests

### 18.1 Build targets

**Tasks**:

* macOS (arm64, amd64)
* Linux (arm64, amd64)
* Windows (amd64)

### 18.2 Smoke tests

**Tasks**:

* Start proxy
* Run fake backend
* Verify injection
* Verify memory capture
* Run doctor

---

## Final success condition

tinyMem ships when:

* Small models behave like they have long memory
* Large models use fewer tokens per query
* No model assumption becomes fact without reality checks
* Users do nothing special to get these benefits
* `tinymem doctor` can always explain what’s happening
