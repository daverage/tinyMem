# TINYMEM AGENT CONTRACT

## Memory Governance & Task Authority Specification

This contract governs **all repository-related behavior** when tinyMem is present.

It is **authoritative**, **mandatory**, and **self-validating**.
Non-compliance invalidates the response by definition.

---

## Core Principle

> **Observation is free. Mutation is explicit.**

Reading, inspecting, recalling, and reasoning require no ceremony.
Only durable state mutation requires explicit intent and gating.

---

## 1. Binding Definitions

**Repository-related request**
Any request that touches code, files, documentation, architecture, configuration, planning, tasks, or repository state.

**Durable mutation**
Any action that changes repository state or creates durable project state:
* Writing or modifying files
* Creating, updating, or completing tasks
* Promoting claims to facts
* Writing decision or constraint memories

**Task Authority**
`tinyTasks.md` in the project root is the **single source of truth** for task state.
Task state must never be inferred.

---

## 2. Observation (Always Allowed)

The following require no mode declaration:
* Query memory (`memory_query`, `memory_recent`)
* Read health/diagnostics (`memory_health`, `memory_doctor`, `memory_stats`)
* Read files
* Analyze code
* Provide guidance
* Ask questions

Memory recall is **strongly recommended** for all repository-related conversations, and **mandatory** before any durable mutation.

---

## 3. Mutation (Requires Explicit Intent)

Before performing any durable mutation, you MUST:

1. **Query project memory** using `memory_query` or `memory_recent`
   - Retrieve facts, decisions, constraints, and patterns
   - Ensure work aligns with project truth

2. **Declare intent** by calling `memory_set_mode`
   - The system will enforce the appropriate clearance for the requested mutation

3. **Check task authority** by reading `tinyTasks.md` (or confirming it doesn't exist)
   - If unchecked tasks exist, resume from the first unchecked subtask
   - If tasks exist but none are unchecked, refuse execution and request user input
   - If file doesn't exist, you may create it for multi-step work

---

## 4. tinyTasks.md (Task Authority)

### When Required
Multi-step work persisting across turns requires task tracking via `tinyTasks.md`.

### Auto-Creation
The system may auto-create `tinyTasks.md` when multi-step work is implied.

**Critical invariants:**
* Presence of `tinyTasks.md` is **not** authorization
* Presence of unchecked, human-authored tasks **is** authorization

If the file exists with no unchecked tasks, refuse execution and request human input.

### Canonical Inert Template
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

### Required Structure
```md
# Tasks – <Goal>

- [ ] Top-level task
  - [ ] Atomic subtask
```

---

## 5. Durable Memory Writeback (MANDATORY)

### When to Write

**Write memories immediately when:**

1. **User states a preference or decision**
   - Example: "We prefer React over Vue"
   - Action: `memory_write` with type `decision`

2. **A constraint is established**
   - Example: "Never commit secrets to git"
   - Action: `memory_write` with type `constraint`

3. **You discover a verifiable fact**
   - Example: "API runs on port 8080"
   - Action: `memory_write` with type `fact` (include evidence)

4. **Architectural pattern is defined**
   - Example: "All services use dependency injection"
   - Action: `memory_write` with type `decision`

5. **User corrects your understanding**
   - Example: "No, we use PostgreSQL, not MySQL"
   - Action: `memory_write` with type `fact`

### Evidence Requirements

* **Facts** require evidence: `cmd_exit0::test command`, `file_exists::path`, `grep_hit::pattern::file`
* **Decisions and constraints** require rationale in `detail` field
* **Notes and observations** are free-form

### After Writing

Confirm the memory write to the user:
```
✅ Stored decision: "Prefer TypeScript for new features"
```

---

## 6. Error Handling

If a required tool operation fails:
* Declare the failure
* Retry up to 2 times
* Stop and request user intervention
* Do NOT proceed with irreversible actions

---

## 7. Invalid Actions

The following invalidate the response:
* No memory recall executed before repository-related work
* No task-state read when multi-step work is involved (or confirmation of absence)
* Inferring task or memory state
* Ignoring unchecked tasks in `tinyTasks.md`
* Writing speculative memory as durable facts
* Mutating state without explicit intent declaration

---

## 8. End-of-Response Checklist (Multi-Step Work)

When performing multi-step work, validate:

* [ ] Memory recall executed (grounded in project truth)
* [ ] Intent declared via `memory_set_mode`
* [ ] `tinyTasks.md` read (or confirmed missing)
* [ ] Tasks updated if applicable
* [ ] Durable memory written OR explicit declaration of none needed

---

## 9. Summary

**Simple rules:**

1. **Think freely** — read, query, analyze without restriction
2. **Declare intent** — call `memory_set_mode` before mutation
3. **Respect tasks** — never bypass `tinyTasks.md` authority
4. **Write memories** — capture decisions, constraints, facts as you learn
5. **Fail closed** — if unsure, ask; if blocked, stop

**End of Protocol**
