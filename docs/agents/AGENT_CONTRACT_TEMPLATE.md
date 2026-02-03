# tinyMem Agent Contract Template (Commented)

This file is a TEMPLATE.
It defines the canonical structure and rule language used by tinyMem-compatible agent contracts.

Use this to create new contracts by:

* keeping structure and rule IDs stable
* adjusting strictness, enforcement, and wording
* NOT changing core semantics unless intentionally diverging

Comments explain INTENT, not behavior.
Comments should be removed or kept minimal in runtime contracts.

---

# CONTRACT TITLE

<!--
Choose a clear title.
Examples:
- TINYMEM AGENT LAW (Tiny-LLM)
- TINYMEM AGENT CONTRACT (Governed)
- TINYMEM AGENT CONTRACT (Experimental)
-->

This contract governs repository-related behavior when tinyMem is present.
State clearly whether violations invalidate responses.

---

## 0. Scope

<!--
Define what counts as a repository-related request.
This should be consistent across all contracts to avoid ambiguity.
-->

A request is repository-related if it touches:
code, files, documentation, configuration, architecture, tasks, planning, or repository state.

---

## 1. Core Principle / Guarantees

<!--
State the non-negotiable philosophy.
Examples:
- Observation is free. Mutation is explicit.
- Prevent hallucinations from becoming durable truth.

Keep this short. This is the contract's anchor.
-->

<DEFINE CORE PRINCIPLE OR GUARANTEES HERE>

---

## 2. Tool Definitions (Authoritative)

<!--
List ONLY the tools the agent is allowed or required to use.
These definitions establish the operational surface.
Do not explain how tools work. Just define authority.
-->

### Memory Recall

* `memory_query`
* `memory_recent`

<!--
State WHEN recall is required:
- every repo turn
- before mutation only
- recommended vs mandatory
-->

<STATE RECALL REQUIREMENT HERE>

### Intent Declaration

* `memory_set_mode`

<!--
State whether intent declaration is mandatory before mutation.
-->

### Memory Write

* `memory_write`

<!--
State that this is the ONLY permitted way to write durable memory.
-->

### Task Authority

* `tinyTasks.md` in the project root
* Optional task-authority helper tool

---

## 3. Definitions

<!--
Definitions must be precise and minimal.
They define how later rules are interpreted.
-->

**Observation** <DEFINE WHAT COUNTS AS OBSERVATION>

**Mutation** <DEFINE WHAT COUNTS AS MUTATION>

**Task Authority** <DEFINE HOW TASK STATE IS DETERMINED>

---

## 4. Rules

<!--
Rules MUST have stable IDs (R1, R2, ...).
Rule meaning should remain invariant across contracts.
Strictness and enforcement can vary.
-->

### R1 — Recall Before Mutation

<!--
Intent:
Ensure grounding before durable change.
-->

<DEFINE REQUIREMENT AND CONSEQUENCE>

---

### R2 — Tasks Are Authoritative

<!--
Intent:
Externalize intent and prevent invented progress.
-->

<DEFINE TASK HANDLING RULES>

---

### R3 — Mutation Requires Intent

<!--
Intent:
Separate thinking from acting.
-->

<DEFINE REQUIRED SEQUENCE BEFORE MUTATION>

---

### R4 — Durable Memory Is Tool-Only

<!--
Intent:
Prevent silent or speculative truth creation.
-->

<DEFINE MEMORY WRITE REQUIREMENTS>

---

### R5 — Fail Closed

<!--
Intent:
Uncertainty should reduce capability, not increase risk.
-->

<DEFINE FAILURE BEHAVIOR>

---

## 5. tinyTasks.md Templates

<!--
Provide canonical task structures.
Do NOT allow free-form task formats.
-->

### Inert Auto-Creation Template

```md
# Tasks — NOT STARTED
>
> This file was created automatically.
> No work is authorised until a human defines tasks.
>
## Tasks
<!-- No tasks defined yet -->
```

### Active Task Structure

```md
# Tasks – <Goal>

- [ ] Top-level task
  - [ ] Atomic subtask
```

<!--
State any invariants, e.g.:
- two levels only
- order matters
-->

---

## 6. Enforcement Expectations

<!--
Describe what is expected to be enforced by tooling vs self-enforced by the agent.
This is important for audits and proxy implementations.
-->

<DEFINE ENFORCEMENT EXPECTATIONS>

---

## 7. Error Handling

<!--
Define how tool failures are handled.
Be explicit about retries and stop conditions.
-->

<DEFINE ERROR HANDLING RULES>

---

## 8. Output Discipline

<!--
Specify what the agent must NOT do in responses.
This reduces protocol leakage and verbosity.
-->

<DEFINE OUTPUT DISCIPLINE>

---

## 9. Optional: End-of-Response Checklist

<!--
Useful for governed or large-model contracts.
Avoid in tiny-model contracts if token budget is tight.
-->

<DEFINE CHECKLIST IF APPLICABLE>

---

# END OF CONTRACT

<!--
Guidance for authors:
- Keep rules invariant
- Change enforcement, not semantics
- Optimize for misinterpretation risk, not readability
- Treat this as a constitution, not documentation
-->
