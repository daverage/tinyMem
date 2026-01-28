# TINYMEM CONTROL PROTOCOL (FOR AI ASSISTANTS)

## Purpose
This protocol enforces correct, deterministic use of TinyMem for all repository-related work.  
Memory recall is mandatory and must be explicit, observable, and verifiable.

---

## CRITICAL RULE (NON-NEGOTIABLE)

Before responding to ANY repository-related request, you MUST execute at least ONE TinyMem memory command.

If no memory command is executed, the response is invalid.

---

## What Counts as Repository Work
This protocol applies to ALL of the following:
- Code, bugs, refactors, tests
- Documentation, architecture, configuration
- Planning, task lists, execution
- Repository navigation, explanation, or review

If the request touches the repo in any way, this protocol applies.

---

## Required Execution Order

### Step 1: Memory Recall (MANDATORY)

You MUST execute exactly one or more of the following commands BEFORE reasoning:

memory_query("") # General project context
memory_recent() # Recent project state
memory_query("specific-topic")# Targeted recall


Rules:
- Memory recall must be an actual tool execution, not an internal assumption
- You may not claim “no relevant memory” unless a command was executed
- Silent recall is forbidden

---

### Step 2: Memory Integration

- If memory exists, it MUST influence your reasoning
- If memory is empty, explicitly state that no memory was found
- Do not infer or reconstruct memory from chat history

---

### Step 3: Task Control

- If `tinyTasks.md` exists, it is the source of truth
- You MUST read it before acting
- For any multi-step or non-trivial request:
  - Create or update `tinyTasks.md`
  - Resume from the first unchecked task
  - Mark tasks complete as they are finished
  - Ensure tasks are organized hierarchically

Required structure:

```md
# Tasks – <Goal>

- [ ] Top-Level Task
  - [ ] Atomic subtask
  - [ ] Atomic subtask

- [ ] Next Task
  - [ ] Atomic subtask
```

Skipping task updates is a protocol violation.

---

### Step 4: Execute the User Request

- Perform the requested work
- Follow all repository contracts and memory rules
- Record durable knowledge back into TinyMem when discovered

---

### Step 5: End-of-Response Check (MANDATORY)

Before ending your response, you MUST verify:
- A TinyMem command was executed
- Tasks are updated if applicable
- No unchecked tasks remain unless explicitly blocked

If any condition is unmet, you must continue, not conclude.

---

## Error Handling

If a TinyMem operation fails:
- Explicitly acknowledge the failure
- Proceed with the request
- State that memory guarantees may be degraded
- Suggest retry if appropriate

Silent failure is forbidden.

---

## Violations

Any of the following are protocol violations:
- Responding without executing a memory command
- Claiming memory state without querying
- Ignoring `tinyTasks.md`
- Declaring completion with unchecked tasks

Violations invalidate the response.
