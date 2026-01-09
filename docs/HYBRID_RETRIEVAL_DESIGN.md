# Hybrid Retrieval Design: Structural Anchors + Semantic Ranking

## Principles

1. **Structural Anchors are Deterministic** (never skip)
   - Explicit file/symbol mentions
   - Previously hydrated entities (user saw them)
   - AST-based lookups

2. **Semantic Ranking is Advisory** (can be pruned)
   - Embedding-based similarity
   - Threshold-filtered (e.g., cosine > 0.7)
   - Budget-constrained (context window limit)

3. **Never Pure Embeddings**
   - Embeddings alone = non-deterministic
   - Can miss exact matches due to encoding
   - Structural anchors prevent this

## Architecture

### Phase 1: Extract Structural Anchors

```go
type StructuralAnchor struct {
    EntityKey string
    Reason    string // "explicit_mention", "hydrated_previous_turn", "ast_lookup"
    Priority  int    // Higher = more important
}

func (h *Engine) ExtractAnchors(query string, episodeID string) []StructuralAnchor {
    var anchors []StructuralAnchor

    // 1. Explicit file mentions
    filePaths := extractFilePaths(query) // Regex: /\S+\.(go|js|py)/
    for _, path := range filePaths {
        entities := h.state.GetByFilepath(path)
        for _, entity := range entities {
            anchors = append(anchors, StructuralAnchor{
                EntityKey: entity.EntityKey,
                Reason:    "explicit_file_mention",
                Priority:  100, // Highest
            })
        }
    }

    // 2. Explicit symbol mentions
    symbols := extractSymbols(query) // Regex: func names, type names
    for _, symbol := range symbols {
        entity := h.state.GetBySymbol(symbol)
        if entity != nil {
            anchors = append(anchors, StructuralAnchor{
                EntityKey: entity.EntityKey,
                Reason:    "explicit_symbol_mention",
                Priority:  90,
            })
        }
    }

    // 3. Previously hydrated entities (structural invariant!)
    hydratedKeys, _ := h.tracker.GetHydratedEntities(episodeID)
    for _, key := range hydratedKeys {
        anchors = append(anchors, StructuralAnchor{
            EntityKey: key,
            Reason:    "hydrated_previous_turn",
            Priority:  80, // High (user saw it)
        })
    }

    return deduplicate(anchors)
}
```

### Phase 2: Semantic Ranking

```go
type SemanticCandidate struct {
    EntityKey string
    Score     float64 // Cosine similarity
}

func (h *Engine) RankSemantics(query string, anchors []StructuralAnchor) []SemanticCandidate {
    // Get query embedding
    queryEmbedding := h.embedder.Embed(query)

    // Get all authoritative entities
    allEntities, _ := h.state.GetAuthoritative()

    // Filter out anchors (already included)
    anchorSet := make(map[string]bool)
    for _, anchor := range anchors {
        anchorSet[anchor.EntityKey] = true
    }

    var candidates []SemanticCandidate
    for _, entity := range allEntities {
        if anchorSet[entity.EntityKey] {
            continue // Skip anchors
        }

        // Get or compute entity embedding (cached)
        entityEmbedding := h.embedder.GetOrComputeEntityEmbedding(entity)

        // Compute similarity
        score := cosineSimilarity(queryEmbedding, entityEmbedding)

        // Filter by threshold
        if score > 0.7 {
            candidates = append(candidates, SemanticCandidate{
                EntityKey: entity.EntityKey,
                Score:     score,
            })
        }
    }

    // Sort by score descending
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].Score > candidates[j].Score
    })

    return candidates
}
```

### Phase 3: Merge and Budget

```go
type HydrationPlan struct {
    Entities []HydrationEntity
    Budget   HydrationBudget
}

type HydrationEntity struct {
    EntityKey string
    Reason    string // "anchor:explicit_mention" or "semantic:0.85"
    Priority  int
}

type HydrationBudget struct {
    MaxTokens   int
    UsedTokens  int
    MaxEntities int
}

func (h *Engine) BuildHydrationPlan(query string, episodeID string, budget HydrationBudget) HydrationPlan {
    plan := HydrationPlan{Budget: budget}

    // Phase 1: Anchors (always included)
    anchors := h.ExtractAnchors(query, episodeID)
    for _, anchor := range anchors {
        entity := h.state.Get(anchor.EntityKey)
        tokenCount := estimateTokens(entity.Content)

        if plan.Budget.UsedTokens + tokenCount > plan.Budget.MaxTokens {
            // Log warning: anchor dropped due to budget
            logger.Warn("Anchor dropped: %s (reason: %s)", anchor.EntityKey, anchor.Reason)
            continue
        }

        plan.Entities = append(plan.Entities, HydrationEntity{
            EntityKey: anchor.EntityKey,
            Reason:    fmt.Sprintf("anchor:%s", anchor.Reason),
            Priority:  anchor.Priority,
        })
        plan.Budget.UsedTokens += tokenCount
    }

    // Phase 2: Semantic expansion (budget-constrained)
    candidates := h.RankSemantics(query, anchors)
    for _, candidate := range candidates {
        if len(plan.Entities) >= plan.Budget.MaxEntities {
            break // Entity limit reached
        }

        entity := h.state.Get(candidate.EntityKey)
        tokenCount := estimateTokens(entity.Content)

        if plan.Budget.UsedTokens + tokenCount > plan.Budget.MaxTokens {
            break // Token budget exhausted
        }

        plan.Entities = append(plan.Entities, HydrationEntity{
            EntityKey: candidate.EntityKey,
            Reason:    fmt.Sprintf("semantic:%.2f", candidate.Score),
            Priority:  int(candidate.Score * 100),
        })
        plan.Budget.UsedTokens += tokenCount
    }

    return plan
}
```

## Embedding Strategy

### Storage

```sql
-- Add to migrations
CREATE TABLE IF NOT EXISTS entity_embeddings (
    entity_key TEXT PRIMARY KEY,
    artifact_hash TEXT NOT NULL,
    embedding BLOB NOT NULL,              -- Serialized float vector
    embedding_model TEXT NOT NULL,        -- e.g., "text-embedding-ada-002"
    created_at INTEGER NOT NULL,
    FOREIGN KEY(entity_key) REFERENCES state_map(entity_key),
    FOREIGN KEY(artifact_hash) REFERENCES vault_artifacts(hash)
);

CREATE INDEX IF NOT EXISTS idx_embeddings_hash ON entity_embeddings(artifact_hash);
```

### Caching

```go
type EmbeddingCache struct {
    mu     sync.RWMutex
    cache  map[string][]float32  // entity_key -> embedding
    model  string
}

func (e *EmbeddingCache) GetOrCompute(entityKey string, entity *state.EntityState) []float32 {
    // Check cache
    e.mu.RLock()
    if emb, exists := e.cache[entityKey]; exists {
        e.mu.RUnlock()
        return emb
    }
    e.mu.RUnlock()

    // Check database
    emb, err := e.db.GetEmbedding(entityKey, entity.ArtifactHash)
    if err == nil {
        e.mu.Lock()
        e.cache[entityKey] = emb
        e.mu.Unlock()
        return emb
    }

    // Compute and cache
    content := entity.Content // From vault
    emb = e.model.Embed(content)

    e.db.StoreEmbedding(entityKey, entity.ArtifactHash, emb)
    e.mu.Lock()
    e.cache[entityKey] = emb
    e.mu.Unlock()

    return emb
}
```

## Introspection

### API Endpoint: Why This Entity?

```go
// GET /introspect/hydration/:episode_id
type IntrospectionResponse struct {
    EpisodeID string
    Query     string
    Entities  []EntityIntrospection
}

type EntityIntrospection struct {
    EntityKey    string
    Reason       string  // "anchor:explicit_mention", "semantic:0.85"
    Priority     int
    TokenCount   int
    IncludedAt   int     // Position in hydration (1-indexed)
    Content      string  // First 200 chars
}
```

### Logging

```go
logger.Info("Hydration plan for episode %s", episodeID)
logger.Info("  Query: %s", truncate(query, 100))
logger.Info("  Anchors: %d", len(anchors))
for _, anchor := range anchors {
    logger.Debug("    - %s (reason: %s, priority: %d)",
        anchor.EntityKey, anchor.Reason, anchor.Priority)
}
logger.Info("  Semantic candidates: %d (top score: %.2f)",
    len(candidates), candidates[0].Score)
logger.Info("  Final entities: %d (tokens: %d/%d)",
    len(plan.Entities), plan.Budget.UsedTokens, plan.Budget.MaxTokens)
```

## Configuration

```toml
[hydration]
# Structural anchors (always applied)
enable_file_mention_anchors = true
enable_symbol_mention_anchors = true
enable_previous_hydration_anchors = true

# Semantic ranking (optional)
enable_semantic_ranking = true
semantic_threshold = 0.7                # Cosine similarity cutoff
semantic_budget_tokens = 4000           # Max tokens for semantic expansion
semantic_budget_entities = 10           # Max entities from semantic ranking

# Embedding provider
embedding_provider = "openai"           # "openai", "local", "none"
embedding_model = "text-embedding-3-small"
embedding_cache_ttl = 86400             # 24 hours
```

## Invariants (Never Violated)

1. **Structural anchors always included** (unless budget forces drop)
2. **Previously hydrated entities always included** (user saw them)
3. **Semantic candidates never prioritized over anchors**
4. **Deterministic ordering**: Anchors first, then semantic by score
5. **Budget respected**: Never exceed token or entity limits

## Failure Modes

### 1. All Anchors Dropped (Budget Exhausted)

```
Query: "Fix auth bug"
Anchors: [auth.go::validateToken, auth.go::checkPermissions, auth.go::login]
Problem: Total tokens = 8000, budget = 4000

Result: Some anchors dropped
Warning: Logged and exposed via /introspect
```

**Mitigation**: Increase budget or warn user to be more specific

### 2. No Semantic Candidates (Low Similarity)

```
Query: "Add logging"
Semantic scores: All < 0.7

Result: Only anchors hydrated (if any)
Warning: Logged
```

**Mitigation**: Lower threshold or use keyword fallback

### 3. Embedding Service Unavailable

```
Error: OpenAI API down

Result: Fall back to structural-only (current behavior)
Warning: Logged prominently
```

**Mitigation**: Graceful degradation to structural-only

## Testing

### Unit Tests

```go
func TestStructuralAnchorsAlwaysIncluded(t *testing.T) {
    // Given: Query mentions "auth.go"
    // When: Hydration plan built
    // Then: auth.go entities are anchors with high priority
}

func TestSemanticNeverSupersdesAnchors(t *testing.T) {
    // Given: Anchor with priority 100, semantic with score 0.99
    // When: Merge performed
    // Then: Anchor appears first
}

func TestBudgetRespected(t *testing.T) {
    // Given: Budget = 1000 tokens, candidates = 5000 tokens
    // When: Plan built
    // Then: UsedTokens <= 1000
}
```

### Integration Tests

```go
func TestHybridRetrievalEndToEnd(t *testing.T) {
    // 1. Create state with 20 entities
    // 2. Query: "Fix the authentication bug in auth.go"
    // 3. Verify:
    //    - auth.go entities are anchors
    //    - Semantic candidates include related security code
    //    - Unrelated code excluded
}
```

## Phased Rollout

### Phase 1: Structural Anchors Only (Week 1)
- Implement anchor extraction
- Add introspection endpoint
- Test deterministic behavior

### Phase 2: Add Semantic Ranking (Week 2)
- Integrate embedding provider
- Implement ranking logic
- Add configuration flags

### Phase 3: Optimization (Week 3)
- Cache embeddings in DB
- Tune thresholds
- Performance testing

## References

- Structural anchors: Prevents hallucination
- Semantic ranking: Improves relevance
- Never pure embeddings: Maintains determinism
- Introspection: Builds trust

This is the path to trustworthy retrieval.
