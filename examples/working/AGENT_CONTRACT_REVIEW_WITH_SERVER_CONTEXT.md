# AGENT_CONTRACT Review (WITH MCP/PROXY CONTEXT)

**Revised Understanding:** The contract is not just for agents to follow—it's **enforced by the tinyMem MCP/Proxy server itself**.

**New Context:** tinyMem can:
- ✅ Intercept tool calls
- ✅ Validate compliance
- ✅ Reject non-compliant patterns
- ✅ Force ordering
- ✅ Track state across turns

This is **fundamentally different** from asking agents to self-comply.

---

## New Verdict: 🟢 **Excellent. With Minor Refinements.**

With the MCP/Proxy enforcing compliance, your contract is **actually pragmatic** and **very well-designed**.

**Why this changes everything:**

### Before (My Initial Review)
❌ "7B models will fail to follow 7 steps"

### After (With Enforcement)
✅ "7B models don't need to follow 7 steps—tinyMem enforces them"

The agent gets:
```
Agent: "Let me fix this bug"
        (doesn't call memory_query)
tinyMem Proxy: ❌ BLOCKED
        "Repository-related request detected. 
         You must call memory_query() first."
Agent: "Oh, right. Calling memory_query..."
```

This is **brilliant**.

---

## What This Changes in Your Contract

### 1. **The 7 Steps Are Now Server-Enforced**

Instead of relying on agent discipline:

```markdown
# How It Works: Server-Enforced Protocol

The tinyMem Proxy/MCP server validates compliance:

Step 1: Agent attempts repository-related work
Step 2: Server checks: "Did they call memory_query/memory_recent?"
Step 3: If NO → Server rejects the request with guidance
Step 4: If YES → Server proceeds to next validation step
Step 5: Server checks: "Is tinyTasks.md state being respected?"
Step 6: If NO → Server rejects and shows task state
Step 7: If YES → Server allows completion

The agent doesn't need perfect discipline. 
The server enforces it.
```

This is **far more practical** than asking agents to self-regulate.

---

## What This Means For Your Contract

### ✅ What's NOW Viable (With Server Enforcement)

1. **Step Enforcement:** The server can track which steps have run
   ```
   Request received at T0: memory_query() not called → REJECT
   Request received at T1: memory_query() called ✓ → Check next step
   Request received at T2: tinyTasks.md not checked → REJECT
   ```

2. **Durable Knowledge Validation:** The server can review what the agent tries to write
   ```
   Agent tries: memory_write --type fact "User asked about API"
   Server review: "This is an observation, not a constraint/decision."
   Server: ❌ REJECTED. Suggest rephrasing as a decision or constraint.
   ```

3. **Task Authority Enforcement:** Server owns the task state
   ```
   Agent marks task complete without checking subtasks?
   Server: ❌ "Subtask 'Fix auth' is still unchecked. 
            Update it first."
   ```

4. **Self-Validation Checklist:** Server can validate automatically
   ```
   Agent ends response with incomplete checklist items?
   Server: "Items unchecked: [memory_query, task_update]. 
            Please address before response ends."
   ```

5. **Error Handling:** Server intercepts failures gracefully
   ```
   memory_query() times out?
   Server: ✅ "Memory tool failed. Continuing with reduced guarantees."
   Agent: Proceeds without memory (server tracked it failed)
   ```

---

## Specific Server-Side Validation Opportunities

Now that I know tinyMem enforces this, here's what should be **server-validated**:

### Validation 1: Memory Query Enforcement

**Server tracks:**
```python
# pseudocode
def validate_repository_request(request):
    if is_repository_related(request):
        if not memory_query_called_in_session():
            return REJECT("Must call memory_query first")
        return ALLOW
```

**Implementation:**
- Server maintains session state
- Tracks which tools have been called
- Rejects requests before memory is queried

### Validation 2: Task State Integrity

**Server tracks:**
```python
def validate_task_update(update):
    current_tasks = read_tinyTasks_md()
    
    # Don't allow marking task complete if subtasks exist
    if update["action"] == "complete":
        subtasks = get_unchecked_subtasks(update["task"])
        if subtasks:
            return REJECT(f"Unchecked subtasks remain: {subtasks}")
    
    # Don't allow resuming a different task
    if update["action"] == "start" and has_active_task():
        active = get_active_task()
        if update["task"] != active:
            return REJECT(f"Task '{active}' is already active")
    
    return ALLOW
```

**Implementation:**
- Server reads/validates `tinyTasks.md` before accepting updates
- Prevents race conditions and inconsistent state
- Guides agents toward correct task structure

### Validation 3: Durable Knowledge Filtering

**Server validates memory writes:**
```python
def validate_memory_write(write_request):
    memory_type = write_request["type"]
    content = write_request["summary"]
    
    # Facts and decisions should be specific, not observational
    if memory_type in ["fact", "decision"]:
        
        # Flag observations (bad)
        bad_patterns = ["user asked", "i noticed", "seems like"]
        if any(pattern in content.lower() for pattern in bad_patterns):
            return REJECT(
                f"Cannot write '{memory_type}' with observational language. "
                f"Did you mean 'claim' or 'plan'?"
            )
        
        # Require specificity (good)
        if len(content) < 10:
            return REJECT("Memory too vague. Add details.")
        
        # Evidence check for facts
        if memory_type == "fact" and "evidence" not in write_request:
            return WARN(
                "Fact has no evidence reference. "
                "Add: --evidence 'file_exists::config.toml'"
            )
    
    return ALLOW
```

**Implementation:**
- Server validates memory writes client-side
- Rejects overly broad/observational claims
- Guides toward high-quality memory

### Validation 4: Ralph Loop Safety

**Server enforces Ralph Loop rules:**
```python
def validate_ralph_invocation(invocation):
    # Require predicates (evidence-based)
    if not invocation.get("predicates"):
        return REJECT(
            "Ralph Loop requires evidence predicates. "
            "Example: test_pass::cargo test auth"
        )
    
    # Enforce safety options
    if not invocation.get("forbid_paths"):
        return WARN(
            "No forbidden paths specified. "
            "Recommend forbidding: .git, node_modules"
        )
    
    max_iter = invocation.get("max_iterations", 5)
    if max_iter > 10:
        return REJECT("max_iterations too high. Max 10.")
    
    return ALLOW
```

**Implementation:**
- Server validates Ralph Loop is properly configured
- Prevents runaway loops
- Ensures safety constraints are set

---

## What This Means For Your Contract Document

Your contract should be **positioned as**:

```markdown
# TINYMEM AGENT CONTRACT

## Purpose

This contract defines how agents SHOULD work with tinyMem.
The tinyMem MCP/Proxy server ENFORCES this contract.

Agents don't need perfect compliance—tinyMem guides them.

---

## For Agents

Follow these steps. If you forget, tinyMem will remind you:

1. Call memory_query() or memory_recent()
2. Check tinyTasks.md
3. Perform work
4. Write durable memory if applicable
5. Update tasks
6. End with validation

---

## For Servers (tinyMem Developers)

The server should validate:

- Memory query called before repo work
- Task state not modified inconsistently
- Memory writes are high-quality (not observational)
- Ralph Loop safety constraints set
- Concurrent access doesn't cause race conditions
- Tool failures are handled gracefully

When validation fails, server should:
- ❌ Reject the request
- 📝 Show exactly why
- 💡 Suggest the correct action
```

---

## Revised Assessment of Each Section

### Section 1: Absolute Precondition ✅
**Status:** GOOD for server enforcement

```markdown
✅ Before producing ANY repository-related response, 
   the agent MUST execute at least one TinyMem command 
   AND check task state.
```

**Server Implementation:**
```python
def intercept_response(response):
    # Check: was memory_query or memory_recent called?
    if not has_memory_call(response):
        return REJECT_RESPONSE(
            "Response missing required memory tool call. "
            "Did you call memory_query() or memory_recent()?"
        )
```

### Section 2: Proof of Execution ✅
**Status:** GOOD for server validation

Server can check the response contains actual tool output:

```python
def validate_proof(response):
    # Proof = actual output, not inference
    acceptable_proofs = [
        "memory_query",
        "memory_recent", 
        "Result:",
        "Found X memories",
        "tinyTasks.md",
    ]
    
    if not any(proof in response for proof in acceptable_proofs):
        return WARN("No visible proof of tool execution shown")
```

### Section 3: Mandatory Execution Order ✅
**Status:** EXCELLENT for server enforcement

Server **owns the state machine**:

```python
session_state = {
    "memory_queried": False,
    "tasks_checked": False,
    "work_done": False,
    "memory_written": False,
    "validated": False,
}

# Server enforces order
def process_agent_request(request):
    if not session_state["memory_queried"]:
        return ENFORCE_STEP_1()
    elif not session_state["tasks_checked"]:
        return ENFORCE_STEP_2()
    # ... etc
```

### Section 4: Error Handling ✅
**Status:** GOOD, clearer with server context

Server can catch and report failures:

```python
try:
    result = memory_query()
except TimeoutError:
    return GRACEFUL_DEGRADE(
        "Memory tool timed out. "
        "Continuing without memory (reduced guarantees). "
        "This was noted."
    )
```

### Section 5: Invalid Actions ✅
**Status:** EXCELLENT for server enforcement

Server as gatekeeper:

```python
INVALID_ACTIONS = [
    "no_memory_tool_called",
    "task_state_inferred_not_checked",
    "declaring_completion_with_unchecked_tasks",
    "writing_observational_memory",
]

for invalid in INVALID_ACTIONS:
    if detected_in_response(invalid):
        return REJECT(f"Invalid: {invalid}")
```

### Section 6: Enforcement Invariant ✅
**Status:** PERFECT

> A repository-related response without TinyMem and `tinyTasks.md` operations is not a response. It is a failure.

**Server enforces:**
```python
def is_valid_response(response):
    has_memory = has_memory_tool(response)
    has_task_check = has_task_file_read(response)
    
    return has_memory and has_task_check
```

---

## What's Now Clearable vs Still Unclear

### ✅ NOW CLEAR (Server Can Enforce)

1. **Step ordering** — Server tracks state machine
2. **Memory tool execution** — Server intercepts calls
3. **Task authority** — Server owns file state
4. **Error handling** — Server gracefully degrades
5. **Invalid actions** — Server blocks them
6. **Proof of execution** — Server validates output format

### 🟡 STILL NEEDS CLARITY (Even With Server)

1. **Durable Knowledge Definition** — Too broad still
   - Server can filter, but needs clear rules
   - Suggest: Update definition with concrete examples

2. **Repository-Related Detection** — What qualifies?
   - Server needs decision logic
   - Suggest: Add decision tree to contract

3. **Ralph Loop Predicate Format** — Not standardized
   - Server needs to validate format
   - Suggest: Formalize as `type::value` (already mentioned, good)

4. **Concurrent Agent Handling** — Not addressed
   - Server can implement task locking
   - Suggest: Add to contract as server-enforced

5. **Graceful Degradation Semantics** — Under-specified
   - When is it OK to skip memory?
   - Suggest: Define degradation levels (full, reduced, minimal)

---

## Simplified Contract (FOR SERVER ENFORCEMENT)

With server enforcement context, here's what your contract should become:

```markdown
# TINYMEM AGENT CONTRACT (Server-Enforced)

## How This Works

tinyMem MCP/Proxy server enforces this contract.
Agents don't need perfect compliance—server guides them.

---

## For Agents: 6 Steps

1. **Query Memory** (Required first)
   ```
   memory_query("<topic>") or memory_recent()
   ```
   Server rejects repository work without memory query.

2. **Check Tasks** (Required, before major work)
   ```
   cat tinyTasks.md
   ```
   Server tracks task state and prevents inconsistency.

3. **Perform Work**
   Update code, documentation, or plans as requested.

4. **Write Durable Memory** (If applicable)
   ```
   memory_write --type decision "We decided X because Y"
   ```
   Server validates memory isn't observational garbage.

5. **Update Tasks**
   Mark subtasks complete as you finish them.

6. **Validate**
   End with: "✓ Memory queried ✓ Tasks checked ✓ Work done"
   Server confirms all steps were executed.

---

## For Servers: What to Enforce

### Validation Layer 1: Request Routing
- Detect repository-related requests (use decision tree)
- Route to enforcement pipeline

### Validation Layer 2: Memory Check
- Require memory_query/memory_recent before repo work
- Allow graceful degradation if tool fails
- Track memory query was performed

### Validation Layer 3: Task Check
- Require tinyTasks.md to be read
- Prevent task state from being inferred
- Enforce task structure (top-level + subtasks)

### Validation Layer 4: Work Execution
- Allow agent to perform requested work
- Track changes against task state
- Suggest task updates if needed

### Validation Layer 5: Memory Writeback
- Validate memory_write calls
- Reject observational/broad claims
- Require evidence for facts
- Accept decisions, constraints, discoveries

### Validation Layer 6: Task Completion
- Ensure subtasks are updated as work completes
- Prevent marking parent complete with unchecked subtasks
- Maintain single source of truth

### Validation Layer 7: Response Validation
- Confirm all steps executed
- Show proof of execution to user
- Flag incomplete compliance (but allow if explained)

---

## Graceful Degradation

If memory tool fails:
```
✓ Memory tool called
✗ Memory tool failed (timeout/offline)
→ Continue without memory injection
→ Declare: "Memory unavailable. Proceeding with reduced context."
```

If task file missing:
```
✓ Task check executed
✗ tinyTasks.md doesn't exist
→ Create it with new tasks
→ Declare: "No prior tasks. Created new task list."
```

---

## What NOT to Accept

- Repository work without memory query
- Task modifications that create inconsistency
- Observational memory writes ("User asked about X")
- Completed tasks with unchecked subtasks
- Silent failures (always declare issues)

---

## Model-Specific Guidance

All models (7B to 405B) can comply because:
- ✅ Server enforces the hard requirements
- ✅ Agents just need to call the tools
- ✅ Server provides guidance when they forget
- ✅ Server validates compliance, not agents

Simpler for agents = better for all.
```

---

## Implementation Notes for tinyMem Server

### What the Server Should Track Per Session

```python
class SessionState:
    def __init__(self):
        self.repository_context_detected = False
        self.memory_query_called = False
        self.memory_call_succeeded = False
        self.tasks_checked = False
        self.tasks_up_to_date = True
        self.work_performed = False
        self.memory_written = []
        self.tasks_updated = []
        self.validation_complete = False
        
    def can_perform_repository_work(self) -> bool:
        return self.memory_query_called
    
    def can_write_memory(self) -> bool:
        return self.tasks_checked
    
    def can_conclude(self) -> bool:
        return (
            self.memory_query_called and 
            self.tasks_checked and 
            self.work_performed
        )
```

### Middleware That Should Run

```python
# Pseudocode for server middleware
class TinyMemProtocolMiddleware:
    
    def intercept_request(self, request):
        """Incoming request from IDE/Agent"""
        
        # Step 1: Is this repository-related?
        if self.is_repository_related(request):
            self.session.repository_context_detected = True
            
            # Step 2: Has memory been queried?
            if not self.session.memory_query_called:
                return self.require_memory_query()
            
            # Step 3: Have tasks been checked?
            if not self.session.tasks_checked:
                return self.require_task_check()
        
        # Allow request to proceed
        return request
    
    def intercept_tool_call(self, tool_name, args):
        """Agent calling a tinyMem tool"""
        
        if tool_name == "memory_query":
            self.session.memory_query_called = True
            self.session.memory_call_succeeded = True
            
        elif tool_name == "memory_recent":
            self.session.memory_query_called = True
            self.session.memory_call_succeeded = True
        
        elif tool_name == "memory_write":
            # Validate before writing
            if not self.is_high_quality_memory(args):
                return self.reject_memory_write(args)
            self.session.memory_written.append(args)
        
        # ... etc for other tools
    
    def intercept_file_operation(self, path, operation):
        """Agent reading/writing tinyTasks.md or repo files"""
        
        if path == "tinyTasks.md" and operation == "read":
            self.session.tasks_checked = True
        
        if path == "tinyTasks.md" and operation == "write":
            # Validate task format
            if not self.is_valid_task_format(args):
                return self.reject_write("Invalid task format")
            self.session.tasks_updated.append(args)
```

---

## Final Recommendation

**Your contract is EXCELLENT with server enforcement context.**

The key insight: **You're not relying on agent discipline. You're relying on server validation.**

This changes everything:

| Aspect | Before (No Server) | After (With Server) |
|--------|-------------------|-------------------|
| Step adherence | Agent must remember all 7 steps | Server enforces each step |
| Error handling | Agent must know what to do | Server catches and guides |
| Memory quality | Agent must know what's durable | Server validates writes |
| Task integrity | Agent must not break state | Server locks/validates |
| Concurrent agents | Chaos | Server mediates with locks |
| Compliance | Unverifiable | Server validates each response |

**Recommendation:** Reposit the contract to emphasize **server as enforcer**, not **agent as complier**.

Keep the rigor. Keep the philosophy. But make clear: **This is a client-server protocol, not a prayer.**

🎯
