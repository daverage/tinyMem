# Chain-of-Verification (CoVe) in tinyMem

## Overview

Chain-of-Verification (CoVe) is a probabilistic filtering and prioritization layer integrated into tinyMem. **CoVe is NOT a truth authority** - it is a noise filter that reduces hallucinated memory candidates and improves recall relevance.

## Design Principles

### What CoVe Does
- ✅ Reduces hallucinated memory candidates before storage
- ✅ Ranks and filters candidate memories by confidence
- ✅ Suppresses low-confidence extractions
- ✅ Optionally filters recall results for relevance (advisory only)

### What CoVe Does NOT Do
- ❌ Decide truth (only evidence verification does this)
- ❌ Create or promote facts
- ❌ Override evidence verification
- ❌ Introduce new memory types
- ❌ Bypass fact promotion rules
- ❌ Increase token usage unboundedly

## Architecture

### Integration Points

#### 1. Memory Extraction (Primary)
```
LLM Response
  ↓
Regex/heuristic extraction
  ↓
Candidate memories (non-fact only)
  ↓
→ CoVe verification pass ← [INTEGRATION POINT 1]
  ↓
Confidence-scored candidates
  ↓
Filter by threshold (default: 0.6)
  ↓
Store only above threshold
```

#### 2. Recall Filtering (Optional)
```
Recall candidates (already bounded)
  ↓
Optional CoVe relevance check ← [INTEGRATION POINT 2]
  ↓
Re-rank or suppress
  ↓
Injection into prompt
```

## Configuration

### TOML Configuration

Add to `.tinyMem/config.toml`:

```toml
[cove]
enabled = true                    # Enable CoVe filtering (default: true)
confidence_threshold = 0.6        # Minimum confidence to keep (default: 0.6)
max_candidates = 20               # Max candidates per batch (default: 20)
timeout_seconds = 30              # LLM call timeout (default: 30)
model = ""                        # Model to use, empty = default (default: "")
recall_filter_enabled = false     # Enable recall filtering (default: false)
```

### Disabling CoVe

If you need to disable CoVe (for performance reasons or to reduce token usage), you can set `enabled = false`:

```toml
[cove]
enabled = false                   # CoVe completely disabled
# Other settings ignored when disabled
```

Alternatively, you can disable CoVe using an environment variable:

```bash
# Disable CoVe
export TINYMEM_COVE_ENABLED=false
```

### Performance and Token Usage Considerations

While CoVe significantly improves memory quality by filtering out hallucinated candidates, it does add some overhead:

- **Token Usage**: CoVe makes additional LLM calls to evaluate memory candidates, which can slightly increase your token usage.
- **Latency**: Each extraction event will have a small delay while CoVe evaluates candidates (typically 0.5-2 seconds).
- **Cost**: Additional API calls to your LLM provider may incur extra costs.

If you're concerned about token usage or performance, you can disable CoVe or adjust the confidence threshold to be more permissive.

### Environment Variables

```bash
# Enable CoVe
export TINYMEM_COVE_ENABLED=true

# Set confidence threshold (0.0-1.0)
export TINYMEM_COVE_CONFIDENCE_THRESHOLD=0.6

# Set max candidates per batch
export TINYMEM_COVE_MAX_CANDIDATES=20

# Set timeout in seconds
export TINYMEM_COVE_TIMEOUT_SECONDS=30

# Set model (optional, empty = default)
export TINYMEM_COVE_MODEL=""

# Enable recall filtering (optional)
export TINYMEM_COVE_RECALL_FILTER_ENABLED=false
```

## Example Configuration

### Conservative (Recommended for Production)
```toml
[cove]
enabled = true
confidence_threshold = 0.7        # Higher threshold = fewer false positives
max_candidates = 10               # Lower limit = faster processing
timeout_seconds = 20              # Shorter timeout = fail-fast
recall_filter_enabled = false     # Disabled for simpler behavior
```

### Aggressive (Experimental)
```toml
[cove]
enabled = true
confidence_threshold = 0.5        # Lower threshold = more permissive
max_candidates = 50               # Higher limit = more thorough
timeout_seconds = 60              # Longer timeout = more patience
recall_filter_enabled = true      # Enable recall filtering
```

### Disabled
```toml
[cove]
enabled = false                   # CoVe completely disabled
# Other settings ignored when disabled
```

## Verification Prompt

CoVe uses a carefully designed prompt to assess candidate memories:

```
You are verifying candidate memory items extracted from an LLM response.

IMPORTANT RULES:
- Do NOT assume any item is true
- Do NOT invent evidence
- Only assess internal consistency and confidence

For each item:
1. Is this a concrete claim, plan, decision, or note?
2. Does it appear speculative or uncertain?
3. Could this be hallucinated or overconfident?
4. Should this be kept (confidence >= threshold) or discarded?

Respond in strict JSON:
[
  {
    "id": "<candidate_id>",
    "confidence": 0.0–1.0,
    "reason": "<short explanation>"
  }
]
```

## Safety Guarantees

### Fail-Safe Behavior
- If CoVe is disabled: All candidates pass through unfiltered
- If CoVe times out: All candidates pass through unfiltered
- If CoVe errors: All candidates pass through unfiltered
- If CoVe returns invalid JSON: All candidates pass through unfiltered

### Bounded Processing
- Maximum candidates per batch: `CoVeMaxCandidates` (default: 20)
- Timeout per LLM call: `CoVeTimeoutSeconds` (default: 30s)
- Candidates exceeding limit are truncated (oldest discarded)

### Invariant Preservation
- **Fact Creation**: CoVe NEVER creates facts (enforced at multiple layers)
- **Evidence Verification**: CoVe does NOT participate in fact promotion
- **Database Triggers**: All DB-level fact constraints remain active
- **Type System**: CoVe cannot change memory types
- **Token Usage**: Recall is already bounded before CoVe filtering

## Statistics

View CoVe statistics using the MCP `memory_stats` tool:

```
Memory Statistics

Total memories: 42

By type:
  fact: 5
  claim: 12
  plan: 8
  decision: 7
  constraint: 6
  observation: 4

Last updated: 2026-01-25T10:30:00Z

CoVe (Chain-of-Verification) Statistics:
  Candidates evaluated: 156
  Candidates discarded: 23
  Average confidence: 0.73
  Discard rate: 14.7%
  Errors: 0
  Last updated: 2026-01-25T10:29:45Z
```

## Performance Impact

### Token Usage
- **Per verification batch**: ~200-500 tokens (prompt) + ~50-100 tokens (response)
- **Typical overhead**: 300-600 tokens per extraction event
- **Frequency**: Once per LLM response (not per memory)

### Latency
- **Synchronous**: Extraction waits for CoVe (30s timeout)
- **Typical delay**: 0.5-2 seconds per batch
- **Mitigation**: Processing happens in background goroutine

### Cost
- **Model**: Uses configured LLM (same as main system)
- **Small model recommended**: Consider using `gpt-3.5-turbo` or equivalent
- **Typical cost**: $0.001-0.005 per extraction event (depending on model)

## Testing

Run CoVe-specific tests:

```bash
# Unit tests
go test ./internal/cove -v

# Integration tests
go test ./internal/extract -run CoVe -v
```

### Test Coverage
- ✅ CoVe disabled = identical behavior
- ✅ CoVe enabled filters low-confidence claims
- ✅ No path to create/promote facts
- ✅ Bounded token usage
- ✅ Fallback on error
- ✅ All invariants preserved

## Troubleshooting

### CoVe Not Filtering Anything
- Check `TINYMEM_COVE_ENABLED=true`
- Check `confidence_threshold` (may be too low)
- Check logs for CoVe errors

### Too Many Memories Discarded
- Lower `confidence_threshold` (e.g., 0.5 instead of 0.7)
- Check CoVe stats with `memory_stats`
- Review discarded candidates in logs

### CoVe Errors/Timeouts
- Increase `timeout_seconds` (e.g., 60 instead of 30)
- Check LLM availability and API key
- Reduce `max_candidates` for faster processing

### Unexpected Behavior
- Disable CoVe temporarily with `TINYMEM_COVE_ENABLED=false`
- Compare behavior with/without CoVe
- Check that invariants are preserved (no facts created)

## Implementation Details

### Code Structure
```
internal/cove/
├── types.go          # Data structures for CoVe
├── stats.go          # Statistics tracking
├── verifier.go       # Core verification logic
└── verifier_test.go  # Unit tests

internal/extract/
├── extractor.go                  # Integration point 1
└── cove_integration_test.go      # Integration tests

internal/inject/
└── injector.go                   # Integration point 2 (optional)
```

### Key Functions
- `cove.NewVerifier()`: Creates CoVe verifier
- `verifier.VerifyCandidates()`: Filters candidate memories
- `verifier.FilterRecall()`: Optional recall filtering
- `verifier.GetStats()`: Returns statistics

## Future Enhancements

Potential improvements (not currently implemented):

- [ ] Adaptive threshold based on false positive rate
- [ ] Per-type confidence thresholds (e.g., higher for claims)
- [ ] Multi-model ensemble voting
- [ ] Fine-tuned model for memory assessment
- [ ] Confidence-based ranking (not just filtering)
- [ ] Historical accuracy tracking

## References

- tinyMem Architecture: `README.md`
- Configuration: `.tinyMem/config.toml`
- MCP Tools: `internal/server/mcp/server.go`
- Evidence System: `internal/evidence/`

## License

Same as tinyMem (see LICENSE file)
