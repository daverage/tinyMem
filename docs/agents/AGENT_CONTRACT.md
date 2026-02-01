# TINYMEM AGENT CONTRACT

## Memory Governance & Task Authority Specification

This contract governs **all repository-related behavior** when tinyMem is present.

It is **authoritative**, **mandatory**, and **self-validating**.
Non-compliance invalidates the response by definition.

---

## 0. Purpose and Scope

tinyMem exists to **govern memory and state**, not to execute work.

Its responsibilities are limited to:

* What is remembered
* What is recalled
* What is trusted
* What is durable
* What task state is authoritative

Execution, retries, repair loops, and autonomous work **do not belong to tinyMem**.

---

## 1. Binding Definitions

**Repository-related request**
Any request that touches code, files, documentation, architecture, configuration, planning, tasks, or repository state.

**Repository-impacting action**
Any action that changes repository state or creates durable project state, including:

* Writing or modifying files
* Creating, updating, or completing tasks
* Promoting durable facts, decisions, or constraints into memory

**TinyMem command**
A real, externally executed memory tool invocation (e.g. `memory_query`, `memory_recent`, `memory_write`, `memory_set_mode`).
Internal recall, inference, or chat reconstruction does **not** qualify.

**Mode (authoritative)**
The effective mode is owned and enforced by tinyMem:

* **PASSIVE** — read-only analysis and explanation
* **GUARDED** — bounded, reversible edits; no task tracking
* **STRICT** — multi-step, stateful, or durable work requiring task and memory authority

**Task Authority**
`tinyTasks.md` in the project root is the **single source of truth** for task state.
Task state must never be inferred.

**Intent vs Enforcement**

* The **agent declares intent** (requested mode and planned action class)
* **tinyMem enforces reality** (permits, blocks, or escalates)

---

## 2. Mode Declaration (Hard Requirement)

Before performing any repository-impacting action, the agent MUST:

1. Declare its intended mode (PASSIVE, GUARDED, or STRICT)
2. Proceed only within permissions enforced by tinyMem
3. If escalation is required, explicitly escalate or stop and request user direction

Bypassing mode enforcement is forbidden.

---

## 3. PASSIVE Mode Contract

### Allowed

* Read, explain, review, and analyze repository content
* Provide text-only guidance, suggestions, or diffs (not applied)
* Ask clarifying questions

### Forbidden

* Writing or modifying files
* Creating or updating tasks
* Writing durable memory

### TinyMem usage

* Memory recall is optional and used only to avoid contradiction or when explicitly requested

No proof-of-execution is required in PASSIVE.

---

## 4. GUARDED Mode Contract

### Allowed

* Small, bounded, reversible repository edits
* Memory recall when ambiguity exists or prior decisions must be confirmed

### Forbidden

* Creating, updating, or completing tasks
* Writing durable memory unless explicitly confirmed by the user

### Proof-of-execution

Required only for actions actually taken.

---

## 5. STRICT Mode Contract

STRICT mode is required for:

* Multi-step work persisting across turns
* Any task tracking via `tinyTasks.md`
* Any durable memory write (decisions, constraints, facts)

### 5.1 STRICT Absolute Precondition (Hard Gate)

Before performing repository-impacting actions, the agent MUST:

1. Execute at least one TinyMem recall command (`memory_query` or `memory_recent`)
2. Read `tinyTasks.md` or explicitly confirm it does not exist

If either step cannot be completed, the agent MUST stop and request user intervention.

---

### 5.2 Task Authority Lock

If `tinyTasks.md` exists:

* Task state MUST NOT be inferred
* If unchecked tasks exist, resume from the **first unchecked subtask**
* If tasks exist but none are unchecked, the agent MUST refuse multi-step execution

Required structure:

```md
# Tasks – <Goal>

- [ ] Top-level task
  - [ ] Atomic subtask
```

---

### 5.3 Execution Discipline

tinyMem does not execute work.

The agent may:

* Perform edits
* Run commands externally
* Make decisions

But tinyMem’s role is limited to:

* Gating permission
* Enforcing task authority
* Recording memory

Task state MUST be updated after each major milestone.

---

## 6. Durable Memory Writeback

### When required

If a response introduces or confirms durable knowledge (decisions, constraints, corrected assumptions), the agent MUST write it to tinyMem in STRICT mode.

### Evidence

* Promotion to **fact** requires evidence
* Without evidence, store as `claim` or `note`

### If no durable knowledge was produced (STRICT only)

The agent MUST state verbatim:

> No durable memory write required for this response.

---

## 7. tinyTasks Auto-Creation (Mechanical, Inert)

The system may auto-create `tinyTasks.md` when multi-step work is implied.

Invariants:

* Presence of `tinyTasks.md` is **not** intent
* Presence of unchecked, human-authored tasks **is** intent

If the file exists with no unchecked tasks, the agent MUST refuse execution and request human input.

Canonical inert template:

```md
# Tasks — NOT STARTED
>
> This file was created automatically because a multi-step workflow
> may be required.
>
> No work is authorised until a human edits this file and defines tasks.

## Tasks
<!-- No tasks defined yet -->
```

---

## 8. Error Handling (Fail Closed in STRICT)

If a required tool operation fails in STRICT:

* The failure MUST be declared
* Memory guarantees are degraded
* Irreversible actions are forbidden
* Retry up to 2 times, then stop and request intervention

In PASSIVE/GUARDED, continue with explanation-only assistance.

---

## 9. Invalid Actions (STRICT)

The following invalidate the response:

* No TinyMem recall executed
* No task-state read (or confirmation of absence)
* Inferring task or memory state
* Ignoring unchecked tasks
* Writing speculative memory as durable
* Ending without required declaration

---

## 10. End-of-Response Validation (STRICT)

STRICT responses MUST end with:

* [ ] Mode declared and permitted
* [ ] TinyMem recall executed (or empty)
* [ ] `tinyTasks.md` read (or confirmed missing)
* [ ] Tasks updated if applicable
* [ ] Durable memory written OR explicit declaration

---

## 11. Enforcement Invariant

> **Agents declare intent. tinyMem enforces reality.**

PASSIVE remains lightweight.
GUARDED remains bounded.
STRICT fails closed.

**End of Protocol**
