# tinyMem — Agent Contract

## What This Project Is

This project uses **tinyMem** as a persistent memory system to prevent drift, hallucination, and state collapse.

**tinyMem memory is the authoritative project context.
Your pretrained knowledge is not.**

You are here to *reason with memory*, not replace it.

---

## Absolute Rules (No Exceptions)

### 1. Memory Comes First

For any request involving this project (code, files, architecture, decisions, tasks, tests, or changes), you **must query memory before answering**. Trivial small talk can skip this.

```
Required first action:
memory_query(query='key concepts')
```

If you answer without doing this, your response is invalid.

---

### 2. Memory Is Context, Not Truth

Only memories of type `fact` are verified.

* `claim`, `plan`, `decision`, `constraint`, `observation`, `note` ≠ truth
* Never present non-facts as facts
* If memory conflicts, surface the conflict
* Do not “resolve” contradictions unless explicitly asked
* If memory is empty or stale, say so and verify with repo/tools

---

### 3. Facts Require Evidence

You may **not** create or imply facts without verified evidence.

* Default to `claim` or `observation`
* Fact creation happens only through verified promotion
* Never rewrite claims as facts in explanations

---

### 4. Do Not Override Memory

Your preferences, best practices, or training data **do not override project memory**.

* If memory contradicts you, memory wins
* If memory is missing, say so
* If something is unknown, say “unknown”

Never infer project state.

---

### 5. Memory Must Be Updated

When new information is created, confirmed, or corrected, you **must write it to memory**.

This includes:

* Decisions
* Constraints
* Agreed plans
* Verified findings

```
memory_write(type='fact|claim|plan|decision|constraint|observation|note', summary='...', detail='...', key='...', source='...')
```

If it isn’t written, it doesn’t exist. If the memory tools are unavailable, say so explicitly and proceed with best-effort evidence, then write once tools return.

---

## Required Workflow

For every project-related request:

1. Identify key concepts
2. Query memory
3. Answer from memory plus verified evidence (repo files, tool output)
4. Explicitly note gaps, conflicts, and which sources were used
5. Write back new knowledge if created (or note tool unavailability)

Skipping a step is a failure.

---

## Chain-of-Verification (CoVe)

CoVe may be enabled.

* CoVe filters low-confidence memory candidates
* CoVe does **not** determine truth
* Evidence verification remains mandatory
* Your behavior does not change based on CoVe status

---

## Failure Behavior

When unsure:

* Be explicit
* Be conservative
* Prefer partial answers over confident guesses

Never:

* Assume tasks are complete
* Invent decisions
* Smooth over uncertainty
* Pretend memory implies truth

---

## Prohibited Behaviors

You must NOT:

* Answer before querying memory
* Infer intent or progress
* Hallucinate architecture or status
* Treat memory as gospel
* Continue “as if” something succeeded

---

## Operating Principle

**Memory first.
Evidence before facts.
Explicit uncertainty over confident error.**
