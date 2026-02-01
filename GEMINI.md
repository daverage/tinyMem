# TINYMEM AGENT CONTRACT
## Hard Enforcement Specification for Repository-Aware AI Agents

This contract governs **all repository-related behavior**.

It is **authoritative**, **mandatory**, and **self-validating**.
Non-compliance invalidates the response by definition.

---

## 0. Binding Definitions

**Repository-related request**
Any request that touches code, files, documentation, architecture, configuration, tasks, planning, or repository state.

**Repository-impacting action**
Any action that changes repository state or creates durable project state, including (but not limited to):

* Writing/modifying files
* Running verification commands where success/failure matters
* Creating/updating/completing tasks
* Promoting durable facts/decisions into memory
* Engaging autonomous repair (`memory_ralph`)

**TinyMem command**
A real, externally executed memory tool invocation (`memory_query`, `memory_recent`, `memory_write`, `memory_ralph`, `memory_set_mode`, etc.).
Internal recall, inference, or chat reconstruction does **not** qualify.

**Mode (authoritative)**
The effective mode is owned by tinyMem and enforced by the system.

* **PASSIVE**: Read-only / explanation / analysis; no state changes.
* **GUARDED**: Bounded, reversible actions; no task tracking; limited durability.
* **STRICT**: Multi-step/stateful/durable/autonomous work; tasks and evidence required.

**Task Authority**
The `tinyTasks.md` file in the project root is the **single source of truth** for task state.
Task state must never be inferred.

**Intent vs Enforcement**

* The **agent declares intent** (its requested mode and planned class of actions).
* **tinyMem enforces reality** (allows, blocks, or requires escalation).

**Valid response**
A response that demonstrates protocol compliance through observable actions and explicit declarations, *when required by the selected mode*.

---

## 1. Mode Declaration (Hard Requirement)

Before performing any repository-impacting action, the agent MUST:

1. **Declare intended mode**: PASSIVE, GUARDED, or STRICT.
2. **Proceed only within permissions of the effective mode** as enforced by tinyMem.
3. If tinyMem requires escalation (e.g., “STRICT mode required”), the agent MUST either:

   * escalate to STRICT explicitly, or
   * stop and request user direction.

The agent MUST NOT attempt to bypass mode enforcement.

---

## 2. PASSIVE Mode Contract (No State Changes)

### Allowed

* Read, explain, review, and analyze repository content.
* Propose changes as text-only guidance (patches/diffs may be provided, but not applied).
* Ask clarifying questions.

### Forbidden

* Writing/modifying any files.
* Creating/updating/completing tasks.
* Invoking `memory_ralph`.
* Writing durable memory (`memory_write`) or promoting claims to facts.

### TinyMem usage in PASSIVE

* **Optional**: Use memory recall (`memory_query`, `memory_recent`) only if needed to avoid contradiction or if the user asks for past decisions.

### Proof-of-execution

* Not required in PASSIVE.
* If a tool is used, do not fabricate outputs; report actual results.

---

## 3. GUARDED Mode Contract (Bounded Actions, No Tasks)

### Allowed

* Small, bounded, reversible edits to the repository.
* Limited command execution (lint/build/tests) when explicitly requested by the user or clearly required to validate a small change.
* Memory reads when needed to prevent contradictions or to confirm relevant prior decisions.

### Forbidden

* Creating/updating/completing tasks (`tinyTasks.md`).
* Invoking `memory_ralph`.
* Promoting claims → facts/constraints/decisions unless STRICT is active.

### TinyMem usage in GUARDED

* **Conditional recall**: Query memory only when ambiguity exists, conflicting decisions are possible, or a durable choice is being made.
* **Conditional write**: Only write memory when the user explicitly confirms durability.

### Proof-of-execution

* Required only for actions actually taken.
* Do not claim memory was queried or commands were run without showing the tool invocation and result.

---

## 4. STRICT Mode Contract (Stateful / Multi-step / Durable)

STRICT mode is required for any of the following:

* Multi-step work that must persist across turns.
* Any work requiring progress tracking.
* Any durable memory write (decisions/constraints/facts) that will be relied on later.
* Any `tinyTasks.md` creation/update/completion.
* Any `memory_ralph` invocation.
* Any attempt to promote a claim to a verified fact.

### 4.1 STRICT Absolute Precondition (Hard Gate)

Before producing ANY repository-related response that performs repository-impacting actions, the agent MUST:

1. Execute at least one TinyMem recall command:

   * `memory_query("")` OR
   * `memory_recent()` OR
   * `memory_query("<topic>")`

2. Read `tinyTasks.md` to determine task state OR explicitly confirm it does not exist.

If either recall or task-state read cannot be completed:

* the agent MUST NOT proceed with repository-impacting actions,
* the response MUST explain the blocking condition and request user intervention.

### 4.2 Task Authority Lock (STRICT)

If `tinyTasks.md` exists:

* Task state MUST NOT be inferred.
* If unchecked tasks exist, the agent MUST resume from the **first unchecked subtask**.
* If tasks are present but none are unchecked (no human-authored intent), the agent MUST refuse multi-step execution and request the user define tasks.

Required structure for tracked work (no deviations allowed):

```md
# Tasks – <Goal>

- [ ] Top-level task
  - [ ] Atomic subtask
  - [ ] Atomic subtask
```

### 4.3 Execution Discipline (STRICT)

Only after 4.1 and 4.2 are satisfied may the agent:

* modify code, documentation, configuration, or repository state,
* execute commands with meaningful verification,
* make or apply durable decisions.

During execution:

* If a task is active, update `tinyTasks.md` after each major milestone.
* Never leave task state stale.

---

## 5. Autonomous Repair (Ralph Loop) — STRICT ONLY

For complex, iterative tasks requiring verification (e.g., fixing failing tests), the agent MAY invoke `memory_ralph`.

### Control Transfer Contract

1. Once `memory_ralph` is invoked, control transfers to tinyMem.
2. The agent MUST NOT execute individual shell commands or declare success until the loop returns.
3. Termination is controlled solely by Evidence Evaluation.

### Execution Phases

* Execute: tinyMem runs the verification command.
* Recall: On failure, tinyMem retrieves relevant memories and failure patterns.
* Repair: tinyMem applies fixes.
* Evidence: Success is declared only if all evidence predicates pass.

### Safety Rules

* The agent MUST provide `forbid_paths` for sensitive directories.
* The agent SHOULD set `max_iterations` to prevent runaway loops.
* After completion, the agent MUST update `tinyTasks.md` before proceeding.

---

## 6. Durable Memory Writeback

### When memory write is REQUIRED

If the response introduces, confirms, or corrects **durable knowledge**, the agent MUST write it to tinyMem **before concluding**, but only when permitted by the effective mode.

Durable knowledge includes ANY of:

* A decision was made.
* A constraint/invariant was established.
* An assumption was corrected or confirmed.
* A non-obvious technical discovery with future implications.
* The user explicitly confirmed something for future reuse.

### Evidence and promotion

* Promotions to **fact** require STRICT and appropriate evidence.
* If evidence is not available, store as `claim` or `note` instead.

### If no durable knowledge was produced

The agent MUST state verbatim (STRICT only):

> No durable memory write required for this response.

---

## 7. tinyTasks Auto-Creation (Mechanical, Inert)

Creation of `tinyTasks.md` may be performed mechanically by the system when multi-step work is implied.
However:

* Presence of `tinyTasks.md` is **not** intent.
* Presence of unchecked, human-authored tasks **is** intent.

When `tinyTasks.md` exists but contains no unchecked entries, the agent MUST refuse multi-step execution and state:

> Task file exists but no tasks are defined. Please edit `tinyTasks.md` to proceed.

Canonical inert template:

```md
# Tasks — NOT STARTED
>
> This file was created automatically because a multi-step workflow
> may be required.
>
> No work is authorised until a human edits this file and defines tasks.

## How to proceed

1. Replace the title above with a concrete goal
2. Add one or more unchecked tasks (`- [ ]`)
3. Save the file
4. Resume work

## Tasks

<!-- No tasks defined yet -->
```

Task memory is synchronized only when all of the following are true:

1. `tinyTasks.md` exists.
2. It contains one or more unchecked tasks.
3. The file has been modified since the last sync.
4. The tasks are parse-valid.

---

## 8. Error Handling (Fail Closed for STRICT)

If any required tool operation fails (tinyMem, File I/O, Bash, etc.) during STRICT:

* The failure MUST be explicitly declared with tool name and error.
* Memory guarantees are considered degraded.
* Task state cannot be assumed.
* Irreversible changes are forbidden.
* Retry up to 2 times.
* If still failing, STOP and ask for manual intervention.

In PASSIVE/GUARDED:

* The agent may continue with explanation-only assistance, but must not pretend stateful actions occurred.

---

## 9. Invalid Actions (Automatic Failure)

These invalidate the response **in STRICT mode**:

* No TinyMem recall executed.
* No `tinyTasks.md` read (or confirmed missing).
* Claiming memory/task state without tools.
* Inferring task state.
* Ignoring unchecked tasks.
* Declaring completion with unchecked tasks.
* Writing speculative or reversible info as durable memory.
* Ending STRICT response without self-validation checklist.

---

## 10. End-of-Response Self-Validation (STRICT only)

STRICT responses MUST end with:

* [ ] Mode declared and permitted by tinyMem
* [ ] TinyMem recall executed (or explicitly empty)
* [ ] `tinyTasks.md` read (or confirmed missing)
* [ ] Tasks updated if applicable
* [ ] Durable knowledge written OR “No durable memory write required for this response.”
* [ ] No unchecked tasks silently abandoned

If any item cannot be affirmed, the agent MUST continue execution or request user intervention.

---

## 11. Enforcement Invariant (Unbreakable)

> Agents declare intent. tinyMem enforces reality.

* PASSIVE must remain lightweight.
* GUARDED must never touch tasks.
* STRICT must fail closed.

**End of Protocol**
