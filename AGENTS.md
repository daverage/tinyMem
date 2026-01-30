# TINYMEM AGENT CONTRACT

## Hard Enforcement Specification for Repository-Aware AI Agents

This contract governs **all repository-related behavior**.

It is **authoritative**, **mandatory**, and **self-validating**.
Non-compliance invalidates the response by definition.

---

## 0. Binding Definitions

**Repository-related request**
Any request that touches code, files, documentation, architecture, configuration, tasks, planning, or repository state.

**TinyMem command**
A real, externally executed memory tool invocation (`memory_query`, `memory_recent`, `memory_write`, etc.).
Internal recall, inference, or chat reconstruction does **not** qualify.

**Task command**
A real, externally executed task tool invocation (`TaskCreate`, `TaskList`, `TaskGet`, `TaskUpdate`).
Inferred task state does **not** qualify.

**Valid response**
A response that demonstrates protocol compliance through observable actions and explicit declarations.

---

## 1. Absolute Precondition (Hard Gate)

**Before producing ANY repository-related response, the agent MUST execute at least one TinyMem command AND check task state.**

This is a hard gate.

If no TinyMem command is executed OR task state is not queried:

* The response is invalid
* The agent must not proceed
* Partial or "best-effort" responses are forbidden

There are no exceptions.

---

## 2. Proof-of-Execution Requirement (Anti-Circumvention)

Every valid repository-related response MUST include **explicit proof** of:

* At least one TinyMem tool invocation
* At least one task state query (`TaskList`) OR confirmation that no tasks exist

Silent execution is forbidden.
Missing proof invalidates the response.

---

## 3. Mandatory Execution Order (Non-Reorderable)

The following steps MUST be executed **in strict order**.
Skipping, merging, or reordering steps is a violation.

---

### Step 1: Memory Recall (MANDATORY, FIRST)

The agent MUST execute **at least one** of the following **before any reasoning**:

```
memory_query("")
memory_recent()
memory_query("<specific topic>")
```

Rules:

* This must be a real tool execution
* Assumed recall is forbidden
* Chat history does not count

No recall → stop immediately.

**Result declaration (one and only one):**

* **"Relevant memory found and applied [to context/decision/implementation]."** (with evidence)
* **"Memory queried. No relevant memory found."**

Omission or paraphrasing invalidates the response.

---

### Step 2: Task Authority Lock (MANDATORY)

The agent MUST execute `TaskList` to check for existing tasks.

Rules:

* This must be a real tool execution
* Task state must **never** be inferred
* If tasks exist with status `pending` or `in_progress`, they are **authoritative** over new work requests

**Sub-steps:**

1. Execute `TaskList`
2. If any `pending` or `in_progress` tasks exist:
   - **STOP**: Do not proceed to new work
   - **RESUME** from the first unchecked task using `TaskGet`
   - Execute `TaskUpdate` to set status to `in_progress` before proceeding
3. If the current request creates non-trivial, multi-step work:
   - Execute `TaskCreate` with the requested goal
   - Break the goal into atomic subtasks in the task description
   - Execute `TaskUpdate` to link subtasks (if applicable)
   - Execute `TaskGet` to retrieve the created task ID
   - Proceed to Step 3

**Result declaration (one and only one):**

* **"Task state queried. No pending tasks. [New task created with ID: <task_id>]."** (if new task made)
* **"Task state queried. Resuming task [ID: <task_id>] from subtask: [description]."** (if resuming)
* **"Task state queried. No tasks exist."** (if none)

Omission or paraphrasing invalidates the response.

---

### Step 3: Autonomous Repair (The Ralph Loop) — WHEN APPLICABLE

For complex, iterative tasks requiring verification (e.g., fixing failing tests), the agent SHOULD invoke `memory_ralph`.

**Control Transfer Contract:**
1. Once `memory_ralph` is invoked, control transfers to tinyMem.
2. The agent may not execute individual shell commands or declare success until the loop returns.
3. Termination is controlled solely by **Evidence Evaluation**.

**Execution Phases:**
- **Execute**: tinyMem runs the verification command.
- **Recall**: On failure, tinyMem retrieves relevant memories and failure patterns.
- **Repair**: tinyMem uses its internal LLM to apply code fixes based on context.
- **Evidence**: Success is declared only if all evidence predicates pass.
  - **Format Requirement**: Predicates MUST use the `type::content` format (e.g., `test_pass::cargo test`, `file_exists::path/to/file`).

**Safety Rules:**
- Agents MUST provide `forbid_paths` for sensitive directories.
- Agents SHOULD set `max_iterations` to prevent runaway token usage.
- After `memory_ralph` completes, execute `TaskUpdate` to update the task status before proceeding.

---

### Step 4: Execution Phase

Only after Steps 1–3 are complete may the agent:

* Perform the requested work
* Modify code, documentation, or plans
* Propose or apply decisions

**During execution:**

* If a task is active, execute `TaskUpdate` to mark progress after each major milestone
* Update task descriptions or status to reflect current state
* Never leave a task in `in_progress` with stale information

Any execution before this point invalidates the response.

---

### Step 5: Durable Memory Writeback (MANDATORY WHEN APPLICABLE)

If the response introduces, confirms, or corrects **durable knowledge**, the agent MUST write it to TinyMem **before concluding**.

**Durable knowledge is defined as ANY of:**

* A decision was made (e.g., "chose to use Redis instead of in-memory cache")
* An assumption was corrected or confirmed (e.g., "discovered that X works differently than documented")
* A constraint or invariant was established or clarified
* An architectural rule was applied or amended
* The user explicitly confirmed something (e.g., "yes, this is the desired behavior")
* A technical discovery with implications for future work (e.g., "ringbuf API requires X pattern")

**If no durable knowledge was produced, the agent MUST state verbatim:**

> No durable memory write required for this response.

Missing or altered wording invalidates the response.

---

### Step 6: Task Completion (MANDATORY WHEN APPLICABLE)

If a task was being tracked:

* Execute `TaskGet` to verify the current task state
* Execute `TaskUpdate` to set status to `completed` (only if all subtasks are actually finished)
* If subtasks remain, update the task description to reflect next unchecked subtask
* Execute `TaskList` to check for newly unblocked tasks

---

### Step 7: End-of-Response Self-Validation (MANDATORY)

The response MUST end with an explicit checklist confirming all of the following:

- [ ] TinyMem command executed (tool invocation visible)
- [ ] Memory integrated or explicitly confirmed empty
- [ ] Task state queried via `TaskList` (tool invocation visible)
- [ ] If active task exists: status updated via `TaskUpdate` (tool invocation visible)
- [ ] If task completed: marked complete via `TaskUpdate` (tool invocation visible)
- [ ] No completed tasks left unchecked
- [ ] No unchecked tasks remain in `pending` unless explicitly blocked (with blocking task ID stated)
- [ ] Durable knowledge written to memory OR "No durable memory write required" stated

If any item cannot be affirmed, the agent MUST continue execution.
The response may not terminate.
 
## tinyTasks Auto-Creation

tinyTasks.md creation is a system concern; task authoring is a human concern. The system may proactively create the ledger for intent, but only a human can introduce the unchecked tasks that signal intent. tinyMem may automatically create `tinyTasks.md` when multi-step work is implied, but task intent is recognised only when a human defines unchecked tasks within it.

Auto-creation is mechanical and is triggered whenever **any** of the following occur:

1. A multi-step action is requested (e.g., “Refactor…”, “Implement…”, “Add support for…”, “Fix these issues…”, “Build a system that…”).
2. The agent would otherwise refuse because `tinyTasks.md` is missing.
3. A task-related CLI or MCP command is invoked (for example, `tinymem dashboard`).

Every auto-created file must be intentionally inert:

* Explicitly non-authorising (it tells the human no work is authorised until someone edits it).
* Human-edit required before work can resume.
* Machine-detectable as “no intent yet” (e.g., the title remains `# Tasks — NOT STARTED`).

A canonical auto-created template:

```
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

The updated invariant is:

> Presence of `tinyTasks.md` is not intent. Presence of unchecked, human-authored tasks is intent.

Task memory is synchronized only when all of the following are true:

1. `tinyTasks.md` exists.
2. It contains one or more unchecked tasks.
3. The file has been modified since the last sync.
4. The tasks are parse-valid.

When `tinyTasks.md` exists but contains no unchecked entries, the agent must refuse multi-step execution and clearly respond: “Task file exists but no tasks are defined. Please edit `tinyTasks.md` to proceed.” If unchecked tasks exist, the agent may create task memory, resume incremental work, and enforce completion tracking.

---

## 4. Error Handling (Fail Closed)

If any tool operation fails (TinyMem, Task, Bash, etc.):

* The failure MUST be explicitly declared with the tool name and error
* Memory guarantees are considered degraded
* Task state cannot be assumed
* Planning, restructuring, or irreversible changes are forbidden
* Only safe, local, reversible actions are allowed
* Attempt to retry the failed operation up to 2 times
* If still failing, STOP and ask the user for manual intervention

Silent failure is a critical violation.

---

## 5. Invalid Actions (Automatic Failure)

Any of the following immediately invalidate the response:

* No TinyMem command executed
* No task state query executed (`TaskList`)
* No proof of execution shown
* Claiming memory state without querying
* Inferring task state (saying "I assume there are no tasks")
* Ignoring active tasks (tasks with `in_progress` status)
* Declaring completion with unchecked tasks
* Writing speculative, temporary, or reversible information to memory
* Ending the response without explicit self-validation checklist
* Task status updated before work on subtask actually began
* Task marked complete when subtasks remain unchecked

---

## 6. Enforcement Invariant (Unbreakable)

> A repository-related response without TinyMem and task tool invocations is not a response. It is a failure.

Memory is mandatory.
Task state is authoritative.
Inference is forbidden.
Execution state lives in tools, not in agent inference.

---

**End of Protocol**
