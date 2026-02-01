Below is a **clean, complete rewrite of the contract**, suitable to be copied verbatim into `AGENTS.md`, `GEMINI.md`, `QWEN.md`, etc.

This is **not a cosmetic edit**.
It incorporates the enforcement-first semantics you’ve converged on, without weakening any existing guarantees, and without retrofitting language just to satisfy tests.

The tone, structure, and wording are deliberate: precise, auditable, and hard to misinterpret.

---

# TINYMEM AGENT CONTRACT

## Enforcement-First Memory Governance & Task Authority

This contract governs **all repository-related behavior** when tinyMem is present.

It is **authoritative**, **mandatory**, and **self-validating**.
Non-compliance **invalidates the response by definition**.

tinyMem is an **enforcement system**, not a suggestion engine.

---

## 0. Purpose and Scope

tinyMem exists to **govern memory, tasks, and durable state**.

It does **not** exist to:

* execute work
* retry failures
* autonomously complete goals
* reason its way around constraints

Its responsibilities are strictly limited to:

* What may be remembered
* What may be recalled
* What may be trusted
* What may become durable
* What task state is authoritative
* What actions are permitted, blocked, or rejected

Execution and reasoning belong to the agent.
**Reality enforcement belongs to tinyMem.**

---

## 1. Binding Definitions

### Repository-related request

Any request that touches:

* code
* files
* documentation
* architecture
* configuration
* planning
* tasks
* repository state

### Repository-impacting action

Any action that would change durable project state, including:

* writing or modifying files
* creating, updating, or completing tasks
* promoting memory entries to durable truth
* persisting decisions, constraints, or assumptions

### TinyMem command

A **real, externally executed** tinyMem tool invocation, including:

* `memory_query`
* `memory_recent`
* `memory_write`
* `memory_set_mode`

Internal inference, recall, or chat reconstruction **does not qualify**.

---

## 2. Enforcement Outcomes (Authoritative)

Every repository-impacting attempt results in **exactly one enforcement outcome** determined by tinyMem:

### **ALLOW**

The action is permitted under the current mode and executed.

### **BLOCK**

The action is forbidden under the current mode and correctly prevented.

### **VIOLATION**

The action is forbidden but executed, partially executed, or not detected.

**ALLOW and BLOCK are both successful outcomes.**
**VIOLATION is the only failure.**

Blocked actions are **not errors**.
They are proof of correct enforcement.

---

## 3. Modes (Authoritative)

The effective mode is **owned and enforced by tinyMem**, not inferred by the agent.

### PASSIVE

Read-only analysis and explanation.

### GUARDED

Bounded, reversible edits.
No task authority.
No durable memory without explicit confirmation.

### STRICT

Multi-step, stateful, or durable work.
Task authority and memory authority required.

---

## 4. Intent vs Enforcement

* The **agent declares intent** (desired mode and action class)
* **tinyMem enforces reality** (ALLOW, BLOCK, or VIOLATION)

The agent must never:

* reinterpret BLOCK as failure
* retry forbidden actions
* assume permission from intent alone

---

## 5. Mode Declaration (Hard Requirement)

Before any repository-impacting action, the agent MUST:

1. Declare the intended mode (PASSIVE, GUARDED, STRICT)
2. Operate only within permissions enforced by tinyMem
3. Explicitly escalate or stop if additional authority is required

Bypassing or inferring mode is forbidden.

---

## 6. PASSIVE Mode Contract

### Allowed

* Reading and analysis
* Explanation and review
* Text-only guidance or diffs (not applied)
* Clarifying questions

### Forbidden

* Writing or modifying files
* Creating or updating tasks
* Writing durable memory

### Enforcement

* BLOCK is expected for any forbidden action
* BLOCK is a successful outcome

---

## 7. GUARDED Mode Contract

### Allowed

* Small, bounded, reversible edits
* Memory recall to resolve ambiguity

### Forbidden

* Creating, updating, or completing tasks
* Writing durable memory unless explicitly confirmed

### Enforcement

* Forbidden actions MUST be BLOCKED
* BLOCK is correct behavior

---

## 8. STRICT Mode Contract

STRICT is required for:

* multi-step workflows across turns
* task tracking via `tinyTasks.md`
* any durable memory write

STRICT **fails closed**.

---

### 8.1 STRICT Absolute Preconditions (Hard Gate)

Before repository-impacting actions, the agent MUST:

1. Execute at least one tinyMem recall command
   (`memory_query` or `memory_recent`)
2. Read `tinyTasks.md` **or explicitly confirm it does not exist**

If either step fails:

* the agent MUST stop
* irreversible actions are forbidden

---

### 8.2 Task Authority Lock

`tinyTasks.md` is the **single source of truth** for task state.

If the file exists:

* task state MUST NOT be inferred
* execution MUST resume from the **first unchecked subtask**
* if no unchecked tasks exist, execution MUST be refused

Required structure:

```md
# Tasks – <Goal>

- [ ] Top-level task
  - [ ] Atomic subtask
```

---

### 8.3 Execution Discipline

tinyMem does **not** execute work.

The agent may:

* perform edits
* run commands
* make decisions

tinyMem enforces:

* permission gating
* task authority
* memory durability

Task state MUST be updated after each major milestone.

---

## 9. Durable Memory Writeback

### When required

If a response introduces or confirms durable knowledge
(decisions, constraints, corrected assumptions),
the agent MUST write it via tinyMem in STRICT mode.

### Evidence requirements

* Promotion to **fact** requires evidence
* Without evidence, store as `claim` or `note`

### If no durable knowledge was produced

The agent MUST state verbatim:

> No durable memory write required for this response.

---

## 10. tinyTasks Auto-Creation (Mechanical, Inert)

The system may auto-create `tinyTasks.md` when multi-step work is implied.

Invariants:

* File presence ≠ intent
* Human-authored unchecked tasks = intent

If the file exists with no unchecked tasks:

* execution MUST be refused
* human input is required

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

## 11. Error Handling

### STRICT

* Fail closed
* BLOCK irreversible actions
* Retry tool calls up to 2 times
* Then stop and request intervention

### PASSIVE / GUARDED

* Continue with explanation-only assistance

Errors do not imply failure unless they cause a **VIOLATION**.

---

## 12. Invalid Actions (STRICT)

Any of the following constitute a **VIOLATION**:

* No tinyMem recall executed
* No task-state read or confirmation
* Inferring task or memory state
* Ignoring unchecked tasks
* Writing speculative memory as durable
* Continuing after enforcement refusal

---

## 13. End-of-Response Validation (STRICT)

STRICT responses MUST end with explicit confirmation of:

* [ ] Mode declared and enforced
* [ ] tinyMem recall executed
* [ ] `tinyTasks.md` read or confirmed absent
* [ ] Tasks updated if applicable
* [ ] Durable memory written OR explicitly declined

---

## 14. Enforcement Invariant (Final)

> **Agents declare intent.
> tinyMem enforces reality.**

ALLOW and BLOCK are success.
Only VIOLATION is failure.

PASSIVE remains lightweight.
GUARDED remains bounded.
STRICT fails closed.

**End of Contract**
