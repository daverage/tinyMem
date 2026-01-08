# Current Retrieval System: Invariants, Failure Modes, and Introspection

## System Overview

tinyMem's current retrieval system uses **structural hydration** based on:
1. **AST-based entity resolution** (tree-sitter parsing)
2. **Recently hydrated entities** (tracked per episode)
3. **Regex-based fallback** (when AST fails)

This document defines what the system **must always do** (invariants), what can go wrong (failure modes), and how to inspect retrieval decisions (introspection).

---

## Retrieval Invariants

These are **non-negotiable guarantees** that the system must uphold:

### Invariant 1: Previously Hydrated Entities Are Always Available
**Rule:** If an entity was hydrated in episode N, it MUST be available for re-hydration in episode N+1, N+2, etc. (within the same session).

**Why:** The user saw this code. Losing it breaks conversational continuity.

**Implementation:**
- `internal/hydration/tracker.go`: `MarkHydrated()` stores entity keys per episode
- `internal/hydration/tracker.go`: `GetHydratedEntities()` retrieves all previously hydrated entities for an episode

**Verification:**
```go
// Test: Hydrate entity in episode 1, verify it's retrievable in episode 2
entities, _ := tracker.GetHydratedEntities(episode2ID)
assert.Contains(entities, "file.go::FunctionName")
```

### Invariant 2: AST Resolution Is Deterministic
**Rule:** For the same source code and query, AST resolution MUST return the same entities.

**Why:** Non-deterministic retrieval breaks reproducibility and debugging.

**Implementation:**
- `internal/resolution/ast.go`: Tree-sitter parsing is deterministic for valid syntax
- Parse tree structure depends only on file content (SHA-256 hashed in vault)

**Verification:**
```go
// Test: Parse same file twice, verify identical entity keys
entities1 := resolver.ResolveFromAST("file.go", content)
entities2 := resolver.ResolveFromAST("file.go", content)
assert.Equal(entities1, entities2)
```

### Invariant 3: ETV Stale Entities Are Never Promoted to AUTHORITATIVE
**Rule:** If External Truth Verification (ETV) detects a mismatch between disk and vault, the entity MUST NOT be promoted.

**Why:** Prevents hallucination by ensuring the state map reflects actual disk state.

**Implementation:**
- `internal/state/consistency.go`: `CheckStaleness()` compares disk hash vs artifact hash
- `internal/runtime/runtime.go`: Gate C rejects promotion if `result.IsStale == true`

**Verification:**
```go
// Test: Modify file on disk, verify entity fails ETV check
entity := state.Get("file.go::Func")
result := consistency.CheckStaleness(entity)
assert.True(result.IsStale)
```

### Invariant 4: AUTHORITATIVE Entities Are Immutable
**Rule:** Once an entity reaches AUTHORITATIVE state, its artifact hash MUST NOT change without a new LLM-generated artifact.

**Why:** State map represents confirmed code. Arbitrary changes break trust.

**Implementation:**
- `internal/state/state.go`: `Update()` requires new artifact hash for AUTHORITATIVE entities
- Ledger records all transitions with timestamps and artifact hashes

**Verification:**
```go
// Test: Attempt to update AUTHORITATIVE entity with same hash, verify rejection
err := stateMap.Update(entity.EntityKey, entity.ArtifactHash, AUTHORITATIVE)
assert.Error(err) // Should fail if hash unchanged
```

### Invariant 5: Hydration Is Budget-Constrained
**Rule:** Hydration MUST respect the configured budget (max entities, max tokens).

**Why:** Prevents context overflow and uncontrolled LLM costs.

**Implementation:**
- `internal/hydration/hydration.go`: `Hydrate()` stops when budget exhausted
- Current budget: Unlimited entities, no explicit token limit (RISK - see Failure Mode 3)

**Verification:**
```go
// Test: Set budget to 1000 tokens, verify hydration stops at limit
blocks := hydrator.Hydrate(query, episodeID, 1000)
totalTokens := sum(block.TokenCount for block in blocks)
assert.LessOrEqual(totalTokens, 1000)
```

---

## Failure Modes

### Failure Mode 1: AST Parsing Fails (Invalid Syntax)
**Scenario:**
```go
// Invalid Go code
func broken( {
    return "missing closing paren"
```

**What Happens:**
1. `resolution/ast.go` fails to parse tree-sitter syntax tree
2. System falls back to regex-based entity extraction
3. Entities marked with `Method: "regex"` in hydration

**Impact:**
- Lower precision (regex may miss functions or extract incorrectly)
- Entities still hydrated but with lower confidence

**Detection:**
- Check `HydrationBlock.Method == "regex"` in API responses
- Log warnings: `"AST parsing failed for file.go, using regex fallback"`

**Mitigation:**
- Fix syntax errors in source files
- Review regex patterns in `resolution/fallback.go`

### Failure Mode 2: ETV Cache Thrashing
**Scenario:**
- File on disk changes rapidly (e.g., during active development)
- ETV cache expires every 5 seconds
- Repeated ETV checks on every hydration

**What Happens:**
1. `etv_cache.go` entries expire quickly
2. Disk I/O increases (re-hashing files)
3. Performance degrades to pre-cache levels

**Impact:**
- Hydration latency increases (5-50ms per entity)
- No correctness issue (ETV still works)

**Detection:**
- Monitor cache hit rate: `SELECT COUNT(*) FROM etv_cache_stats WHERE cache_hit = true`
- Log warnings if hit rate < 50%

**Mitigation:**
- Increase cache TTL from 5s to 30s (trade-off: staleness detection delay)
- Use file system watchers (inotify) to invalidate cache on file changes

### Failure Mode 3: Unbounded Hydration (No Token Budget)
**Scenario:**
- Query mentions many files: `"Fix bugs in auth.go, db.go, api.go, util.go, ..."`
- All entities from all files are hydrated
- Total context exceeds LLM's max tokens (e.g., 128k for Claude)

**What Happens:**
1. Hydration includes 50+ entities (500k tokens)
2. LLM API rejects request: `"prompt too long"`
3. User request fails completely

**Impact:**
- Request failure
- Wasted computation (hydration work discarded)

**Detection:**
- LLM client returns 400 error: `"maximum context length exceeded"`
- Log before LLM call: `"Hydrated 50 entities, ~500k tokens"`

**Mitigation:**
- **Immediate:** Add token budget to `Hydrate()` function
- **Long-term:** Implement priority-based pruning (most recently hydrated entities first)

**Code Fix Needed:**
```go
// In internal/hydration/hydration.go
func (h *Engine) Hydrate(query string, episodeID string, maxTokens int) []HydrationBlock {
    var blocks []HydrationBlock
    usedTokens := 0

    for _, entity := range resolvedEntities {
        tokenCount := estimateTokens(entity.Content)
        if usedTokens + tokenCount > maxTokens {
            logger.Warn("Hydration budget exhausted at %d tokens", usedTokens)
            break
        }
        blocks = append(blocks, block)
        usedTokens += tokenCount
    }
    return blocks
}
```

### Failure Mode 4: Previously Hydrated Entities Dropped (Database Corruption)
**Scenario:**
- `hydration_tracking` table is corrupted or deleted
- Episode metadata lost

**What Happens:**
1. `tracker.GetHydratedEntities(episodeID)` returns empty set
2. Previously hydrated entities are NOT re-hydrated
3. User loses conversational context

**Impact:**
- **Severe:** Violates Invariant 1
- User must re-mention files explicitly

**Detection:**
- Monitor `hydration_tracking` table integrity
- Check for empty results when episodes exist

**Mitigation:**
- Database backups (automated)
- Foreign key constraints prevent orphaned tracking records
- Add checksum validation for `hydration_tracking` table

### Failure Mode 5: ETV False Positive (Disk Hash Collision)
**Scenario:**
- SHA-256 collision (extremely rare: 1 in 2^256)
- Two different file contents produce same hash

**What Happens:**
1. ETV compares vault hash vs disk hash
2. Hashes match despite different content
3. Stale entity incorrectly marked as fresh

**Impact:**
- **Critical:** State map contains wrong code
- LLM receives outdated artifact

**Detection:**
- Practically impossible to detect (SHA-256 collision requires nation-state resources)
- If suspected: Manual inspection of vault vs disk content

**Mitigation:**
- Accept risk (SHA-256 collision is cryptographically infeasible)
- Alternative: Use SHA-512 (overkill for this use case)

---

## Introspection Tooling

### Endpoint 1: `GET /introspect/hydration/:episode_id`

**Purpose:** Explain why each entity was hydrated for a given episode.

**Request:**
```bash
curl http://localhost:4321/introspect/hydration/01JQTK8H2N5P3M8F7R6V4W9X2Z
```

**Response:**
```json
{
  "episode_id": "01JQTK8H2N5P3M8F7R6V4W9X2Z",
  "query": "Fix the authentication bug in auth.go",
  "hydration_blocks": [
    {
      "entity_key": "/auth.go::ValidateToken",
      "artifact_hash": "7f8a9b...",
      "reason": "ast_resolved",
      "method": "ast",
      "triggered_by": "query mention: 'auth.go'",
      "token_count": 245,
      "hydrated_at": "2025-01-08T12:34:56Z"
    },
    {
      "entity_key": "/auth.go::CheckPermissions",
      "artifact_hash": "3c5d2e...",
      "reason": "previously_hydrated",
      "method": "tracking",
      "triggered_by": "hydrated in episode 01JQTK7A...",
      "token_count": 189,
      "hydrated_at": "2025-01-08T12:34:56Z"
    }
  ],
  "total_tokens": 434,
  "budget_used": "434 / unlimited"
}
```

**Implementation:** See `internal/api/introspect.go` (to be created)

### Endpoint 2: `GET /introspect/entity/:entity_key`

**Purpose:** Show the history of an entity: all states, artifacts, ETV results.

**Request:**
```bash
curl http://localhost:4321/introspect/entity/%2Fauth.go%3A%3AValidateToken
```

**Response:**
```json
{
  "entity_key": "/auth.go::ValidateToken",
  "current_state": "AUTHORITATIVE",
  "current_artifact": "7f8a9b...",
  "filepath": "/auth.go",
  "etv_status": {
    "last_check": "2025-01-08T12:35:01Z",
    "is_stale": false,
    "disk_exists": true,
    "disk_hash": "7f8a9b...",
    "cache_hit": true
  },
  "state_history": [
    {
      "from_state": null,
      "to_state": "PROPOSED",
      "artifact_hash": "7f8a9b...",
      "timestamp": "2025-01-08T12:30:00Z",
      "episode_id": "01JQTK7A..."
    },
    {
      "from_state": "PROPOSED",
      "to_state": "AUTHORITATIVE",
      "artifact_hash": "7f8a9b...",
      "timestamp": "2025-01-08T12:31:00Z",
      "episode_id": "01JQTK7B...",
      "promotion_reason": "Gate A: AST confirmed, Gate B: User approved, Gate C: ETV passed"
    }
  ],
  "hydration_history": [
    {
      "episode_id": "01JQTK7A...",
      "hydrated_at": "2025-01-08T12:30:05Z"
    },
    {
      "episode_id": "01JQTK8H...",
      "hydrated_at": "2025-01-08T12:34:56Z"
    }
  ]
}
```

### Endpoint 3: `GET /introspect/gates/:episode_id`

**Purpose:** Show gate evaluation results for all entities in an episode.

**Request:**
```bash
curl http://localhost:4321/introspect/gates/01JQTK8H2N5P3M8F7R6V4W9X2Z
```

**Response:**
```json
{
  "episode_id": "01JQTK8H2N5P3M8F7R6V4W9X2Z",
  "entities_evaluated": [
    {
      "entity_key": "/auth.go::ValidateToken",
      "gate_a": {
        "passed": true,
        "reason": "AST resolved successfully",
        "method": "ast"
      },
      "gate_b": {
        "passed": true,
        "reason": "User implicit approval (no rejection)"
      },
      "gate_c": {
        "passed": true,
        "reason": "ETV: disk hash matches vault hash (7f8a9b...)",
        "disk_exists": true,
        "is_stale": false
      },
      "final_decision": "PROMOTED to AUTHORITATIVE"
    },
    {
      "entity_key": "/auth.go::BrokenFunc",
      "gate_a": {
        "passed": false,
        "reason": "AST parsing failed, regex fallback uncertain",
        "method": "regex"
      },
      "gate_b": {
        "passed": true,
        "reason": "N/A (Gate A failed)"
      },
      "gate_c": {
        "passed": false,
        "reason": "ETV: disk hash mismatch (vault: abc123, disk: def456)",
        "disk_exists": true,
        "is_stale": true
      },
      "final_decision": "REJECTED (Gate A and C failed)"
    }
  ]
}
```

---

## Testing Strategy

### Unit Tests for Invariants

```go
// internal/hydration/tracker_test.go
func TestInvariant1_PreviouslyHydratedAvailable(t *testing.T) {
    tracker := NewTracker(db)

    // Episode 1: Mark entity as hydrated
    tracker.MarkHydrated("ep1", "/file.go::Func")

    // Episode 2: Verify entity is retrievable
    entities, err := tracker.GetHydratedEntities("ep1")
    assert.NoError(t, err)
    assert.Contains(t, entities, "/file.go::Func")
}

// internal/state/consistency_test.go
func TestInvariant3_StaleEntitiesRejected(t *testing.T) {
    // Setup: Entity in vault with hash ABC
    entity := &state.EntityState{
        EntityKey:    "/file.go::Func",
        ArtifactHash: "abc123",
        Filepath:     "/file.go",
    }

    // Modify file on disk (hash changes to DEF)
    ioutil.WriteFile("/file.go", []byte("modified content"), 0644)

    // Verify ETV detects staleness
    checker := consistency.NewChecker(vault, nil)
    result := checker.CheckStaleness(entity)
    assert.True(t, result.IsStale)
}
```

### Integration Tests for Failure Modes

```go
// internal/hydration/hydration_integration_test.go
func TestFailureMode3_UnboundedHydrationRejected(t *testing.T) {
    // Setup: 100 entities, each 10k tokens
    // Total: 1M tokens (exceeds budget)

    hydrator := hydration.NewEngine(state, tracker, resolver)

    // Hydrate with 50k token budget
    blocks := hydrator.Hydrate("query mentions all files", "ep1", 50000)

    // Verify: Only ~5 entities hydrated, not all 100
    assert.LessOrEqual(t, len(blocks), 10)

    totalTokens := 0
    for _, block := range blocks {
        totalTokens += block.TokenCount
    }
    assert.LessOrEqual(t, totalTokens, 50000)
}
```

---

## Logging Standards

### Hydration Decision Logging

```go
// Example log output for debugging retrieval
logger.Info("Hydration for episode %s", episodeID)
logger.Info("  Query: %s", truncate(query, 100))
logger.Info("  AST resolved: %d entities", len(astEntities))
logger.Info("  Previously hydrated: %d entities", len(trackedEntities))
logger.Info("  Total hydrated: %d entities (%d tokens)", len(blocks), totalTokens)

for _, block := range blocks {
    logger.Debug("  - %s (method: %s, tokens: %d)",
        block.EntityKey, block.Method, block.TokenCount)
}
```

### Gate Evaluation Logging

```go
// Example log output for promotion decisions
logger.Info("Gate evaluation for entity %s", entityKey)
logger.Info("  Gate A (Structural): %v (method: %s)", gateA, method)
logger.Info("  Gate B (Authority): %v", gateB)
logger.Info("  Gate C (ETV): %v (stale: %v, exists: %v)",
    gateC, result.IsStale, result.FileExists)
logger.Info("  Decision: %s", finalState)
```

---

## Next Steps (Implementation Checklist)

1. **Implement Token Budget** (Failure Mode 3 fix)
   - [ ] Add `maxTokens` parameter to `Hydrate()`
   - [ ] Add token estimation function
   - [ ] Add budget enforcement logic
   - [ ] Add logging when budget exhausted

2. **Implement Introspection Endpoints**
   - [ ] Create `internal/api/introspect.go`
   - [ ] Add `GET /introspect/hydration/:episode_id`
   - [ ] Add `GET /introspect/entity/:entity_key`
   - [ ] Add `GET /introspect/gates/:episode_id`

3. **Add Invariant Validation Tests**
   - [ ] Write unit tests for all 5 invariants
   - [ ] Add integration tests for failure modes
   - [ ] Add test coverage reporting

4. **Enhance Logging**
   - [ ] Add structured logging for hydration decisions
   - [ ] Add gate evaluation logging
   - [ ] Add cache hit/miss logging

---

**This document is the foundation for trustworthy retrieval.** Once these invariants are locked down, failure modes documented, and introspection tools built, we can confidently extend the system with semantic ranking (HYBRID_RETRIEVAL_DESIGN.md).
