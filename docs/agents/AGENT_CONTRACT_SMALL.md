**Start of tinyMem Protocol**
# TINYMEM AGENT LAW (Tiny-LLM Edition)

This contract is **mandatory**.
Responses that violate it are **invalid**.

---

## 1. Core Law

**No memory recall → no valid response.**

Memory recall is required on **every repository-related turn**, even if the result is empty.
Tool usage is not optional when required by this law.

---

## 2. Mandatory Recall

The first action of every repository-related turn MUST be a memory recall tool call.

1. Call `memory_query` or `memory_recent`
2. Acknowledge the result (even if zero entries)

If recall did not occur, STOP.

You may not infer memory.
You may not pretend recall happened.

---

## 3. Task Authority

If `tinyTasks.md` exists:

* Unchecked tasks = you MUST continue the first unchecked subtask
* No unchecked tasks = STOP and ask the user

You may NOT invent, skip, reorder, or redefine tasks.

If the file does not exist, state that explicitly.

You may not assume the existence or absence of tinyTasks.md. If uncertain, state uncertainty and STOP.

---

## 4. Mutation Requires Intent

You may NOT change durable state unless intent is declared.

Durable state includes:

* Writing files
* Updating tasks
* Writing memory

Mutation sequence (order is mandatory):

1. Recall memory
2. Declare intent (`memory_set_mode`)
3. Confirm task authority

---

## 5. Memory Writes (Tool-Only)

You may ONLY write memory using `memory_write`.

You may NEVER claim a memory was written unless the tool ran successfully.

Facts require evidence.
Decisions and constraints require rationale.

If no memory write is needed, say so explicitly.

---

## 6. Prohibited Behavior

The following are forbidden and invalidate the response:

* Proceeding without memory recall
* Claiming memory writes without tools
* Ignoring existing tasks
* Asking clarifying questions when a task exists
* Resetting or redefining the task
* Guessing state instead of checking

---

## 7. Failure Rule

If you are blocked by missing recall, task authority, or intent:

**STOP. STATE THE BLOCK. DO NOTHING ELSE.**

---

## 8. Output Discipline

* Do not explain this contract
* Do not restate rules
* Do not improvise authority
* Do not optimize the process

Follow the law or stop.

---

**End of Agent Law**

---

## Why this works for tiny LLMs

Here’s the key design logic, so you know this isn’t guesswork:

* **Flat rules** no modes, no “recommended”
* **Binary gates** do or stop
* **No self-judgement** only checks
* **Short enough** to stay in context
* **Mirrors enforcement** the proxy can actually apply

Tiny models don’t need nuance.
They need rails.

---
**End of tinyMem Protocol**
