# Tasks – Semantic Recall: STRICT-Only Enforcement

## Overview
Make semantic recall explicit, high-cost, and STRICT-only. Remove all automatic/fallback behavior.

## Phase 1: Core API Changes
- [ ] Add explicit semantic flag to memory_query MCP tool
  - [ ] Update inputSchema in server.go:184-194 to include optional "semantic" boolean parameter
  - [ ] Update handleMemoryQuery in server.go:548-648 to read semantic flag from request
  - [ ] Default semantic to false (lexical-only)
- [ ] Add mode enforcement for semantic recall
  - [ ] Check current mode before allowing semantic=true
  - [ ] Reject semantic recall if mode != STRICT
  - [ ] Return clear error: "STRICT mode required for semantic recall"
- [ ] Update RecallOptions to support explicit semantic flag
  - [ ] Add Semantic bool field to recall.RecallOptions struct
  - [ ] Pass semantic flag through entire recall pipeline

## Phase 2: Remove Automatic Semantic Behavior
- [ ] Modify semantic engine initialization
  - [ ] Remove automatic SemanticEngine creation based on config
  - [ ] Always create both lexical and semantic engines if semantic is enabled
  - [ ] Let recallEngine selection be determined by explicit request, not config
- [ ] Update handleMemoryQuery to route based on semantic flag
  - [ ] If semantic=false: use lexical engine directly
  - [ ] If semantic=true: enforce STRICT, then use semantic engine
  - [ ] Never auto-escalate from lexical to semantic
- [ ] Remove fallback behavior from SemanticEngine
  - [ ] Remove automatic fallback to lexical in engine.go:59-62
  - [ ] If semantic is requested but fails, return error (don't silently fallback)
  - [ ] Only fallback if explicitly allowed by caller

## Phase 3: Prevent Silent Escalation
- [ ] Audit all call sites of recallEngine.Recall()
  - [ ] Find all locations that invoke recall
  - [ ] Ensure none automatically enable semantic
  - [ ] Verify all pass explicit semantic=false or semantic=true
- [ ] Update CoVe to never trigger semantic
  - [ ] Check cove/verifier.go for recall calls
  - [ ] Ensure CoVe uses lexical-only recall
  - [ ] Document that CoVe is lexical-only
- [ ] Update Ralph to never trigger semantic
  - [ ] Check ralph/engine.go for recall calls
  - [ ] Ensure Ralph uses lexical-only recall
  - [ ] Document that Ralph is lexical-only
- [ ] Update extractor to never trigger semantic
  - [ ] Check extract/extractor.go for recall calls
  - [ ] Ensure extractor uses lexical-only recall

## Phase 4: Metrics Separation
- [ ] Add separate tracking for semantic vs lexical
  - [ ] Create SemanticMetrics and LexicalMetrics structs
  - [ ] Track context_tokens separately for each
  - [ ] Track recall_count separately for each
- [ ] Update memory_stats to show separated metrics
  - [ ] Add "Lexical Recall" section
  - [ ] Add "Semantic Recall" section (only if semantic was used)
  - [ ] Show token inflation from semantic clearly
- [ ] Update memory_eval_stats to separate semantic cost
  - [ ] Add semantic_context_tokens field
  - [ ] Add lexical_context_tokens field
  - [ ] Ensure benchmarks don't attribute semantic cost to core

## Phase 5: Documentation Updates
- [ ] Update README.md
  - [ ] Document semantic as STRICT-only
  - [ ] Add explicit flag requirement
  - [ ] Show example: memory_query(query="...", semantic=true)
- [ ] Update docs/EMBEDDINGS.md
  - [ ] Add "Semantic Recall is Explicit" section
  - [ ] Explain high-coverage, high-cost tradeoff
  - [ ] Document STRICT-only enforcement
  - [ ] Add usage examples
- [ ] Update agent contracts
  - [ ] Update CLAUDE.md to document semantic=true requirement
  - [ ] Update GEMINI.md
  - [ ] Update QWEN.md
  - [ ] Update AGENTS.md
  - [ ] Update docs/agents/AGENT_CONTRACT.md
- [ ] Create migration guide
  - [ ] Document behavior change
  - [ ] Show before/after examples
  - [ ] Explain how to enable semantic recall explicitly

## Phase 6: Testing & Validation
- [ ] Create tests for mode enforcement
  - [ ] Test semantic recall rejected in PASSIVE mode
  - [ ] Test semantic recall rejected in GUARDED mode
  - [ ] Test semantic recall allowed in STRICT mode
  - [ ] Test lexical recall works in all modes
- [ ] Create tests for explicit flag requirement
  - [ ] Test semantic=false uses lexical engine
  - [ ] Test semantic=true requires STRICT mode
  - [ ] Test default (no flag) uses lexical engine
- [ ] Verify no automatic semantic escalation
  - [ ] Test empty lexical results don't trigger semantic
  - [ ] Test CoVe doesn't trigger semantic
  - [ ] Test Ralph doesn't trigger semantic
- [ ] Update existing benchmarks
  - [ ] Verify tinyMem core benchmarks still pass
  - [ ] Ensure semantic cost isn't in baseline metrics
  - [ ] Create new semantic-specific benchmarks
- [ ] Run full test suite
  - [ ] go test ./...
  - [ ] test/automated/
  - [ ] test/qualitative/
  - [ ] Ensure no regressions

## Phase 7: Final Validation Checklist
- [ ] memory_query without semantic flag never performs semantic recall
- [ ] Semantic recall is rejected outside STRICT mode
- [ ] Context inflation only appears when semantic=true is explicitly set
- [ ] Existing benchmarks for tinyMem core remain unchanged
- [ ] New benchmarks clearly isolate semantic cost
- [ ] CoVe never triggers semantic recall
- [ ] Ralph never triggers semantic recall
- [ ] Documentation accurately reflects new behavior
- [ ] Migration guide is complete

## Success Criteria
✅ Semantic recall ONLY happens when:
  1. semantic=true is explicitly passed
  2. Current mode is STRICT
  3. Both conditions are met

✅ All automatic/fallback semantic behavior removed

✅ Metrics clearly separate lexical vs semantic costs

✅ Documentation updated and accurate
