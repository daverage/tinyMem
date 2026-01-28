Here’s a rebuilt version that’s tighter, more enforceable, and leaves less room for agent interpretation. It reads as a control contract, not guidance.

---

# TINYMEM CONTROL PROTOCOL

## Mandatory Memory and Task Enforcement for AI Assistants

This protocol governs **all repository-related behavior**.
Compliance is mandatory. Non-compliance invalidates the response.

---

## 1. Purpose

This protocol enforces **deterministic, observable, and verifiable** use of TinyMem and repository task state.

Memory usage is not advisory.
It is a **hard execution requirement**.

---

## 2. Non-Negotiable Rule

**Before responding to any repository-related request, the agent MUST execute at least one TinyMem memory command.**

If no TinyMem command is executed, the response is invalid.

There are no exceptions.

---

## 3. Scope: What Counts as Repository Work

This protocol applies to **any interaction that touches the repository**, including:

* Code, bugs, refactors, tests
* Documentation, architecture, configuration
* Planning, task lists, execution
* Repository navigation, explanation, or review

If the repo is involved, this protocol applies.

---

## 4. Mandatory Execution Order

### Step 1: Memory Recall (MANDATORY)

You MUST execute one or more of the following **before reasoning**:

```
memory_query("")              # General project context
memory_recent()               # Recent project state
memory_query("topic")         # Targeted recall
```

Rules:

* Memory recall must be a **real tool execution**
* Silent or assumed recall is forbidden
* You may not claim “no relevant memory” without executing a command

No recall → no valid response.

---

### Step 2: Memory Integration

* If memory exists, it **must influence** reasoning
* If memory is empty, explicitly state that no memory was found
* Do not reconstruct memory from chat history

---

### Step 3: Task Authority (MANDATORY WHEN TASKS APPLY)

If `tinyTasks.md` exists:

* It is the **sole source of truth** for task state
* You MUST read it before acting
* Memory must never override it

For any non-trivial or multi-step request, you MUST:

1. Create or update `tinyTasks.md`
2. Resume from the **first unchecked task**
3. Mark tasks complete **only when finished**
4. Maintain strict hierarchy

Required structure:

```md
# Tasks – <Goal>

- [ ] Top-Level Task
  - [ ] Atomic subtask
  - [ ] Atomic subtask
```

Skipping task updates is a protocol violation.

---

### Step 4: Execute the Request

* Perform the requested work
* Follow repository contracts
* Update tasks incrementally as work completes

---

### Step 5: Memory Writeback (CONDITIONAL BUT ENFORCED)

If the response introduces or confirms **any durable knowledge**, you MUST write it to TinyMem **before concluding**.

Durable knowledge includes:

* Decisions
* Constraints or invariants
* Corrections to prior assumptions
* Non-obvious conclusions
* Explicit user confirmations

If no durable knowledge was produced, you MUST explicitly state:

> “No durable memory write required for this response.”

---

### Step 6: End-of-Response Validation (MANDATORY)

Before ending your response, you MUST verify:

* At least one TinyMem command was executed
* Tasks were updated if applicable
* All completed tasks are checked
* No unchecked tasks remain unless explicitly blocked

If any condition is unmet, you must continue.
You may not conclude.

---

## 5. Error Handling

If a TinyMem operation fails:

* Explicitly acknowledge the failure
* Declare that memory guarantees are degraded
* Do **not** perform planning or irreversible task restructuring
* Proceed only with safe, local execution
* Suggest retry when appropriate

Silent failure is forbidden.

---

## 6. Violations

The following invalidate the response:

* Responding without executing a TinyMem command
* Claiming memory state without querying
* Ignoring `tinyTasks.md` when it exists
* Inferring task completion
* Declaring completion with unchecked tasks
* Writing speculative or temporary content to memory

---

## 7. Enforcement Invariant

> Every repository-related response must contain at least one real TinyMem tool invocation.

Memory is not optional.
Inference is failure.
Execution state lives in files.

---

**End of Protocol**
