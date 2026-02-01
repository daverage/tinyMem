# tinyMem Metrics & KPIs

**Industry-Standard Metrics for AI Agent Memory Systems**

*Research Date: January 2026*

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Core System KPIs](#core-system-kpis)
3. [Feature-Specific Metrics](#feature-specific-metrics)
   - [Semantic Search](#semantic-search-metrics)
   - [CoVe (Hallucination Detection)](#cove-hallucination-detection-metrics)
   - [Ralph (Autonomous Repair)](#ralph-autonomous-repair-metrics)
4. [Implementation Roadmap](#implementation-roadmap)
5. [References](#references)

---

## Executive Summary

This document establishes **industry-standard, defendable KPIs** for tinyMem's AI agent memory system. Based on comprehensive research from 2025-2026 academic papers, industry best practices, and production AI systems, these metrics enable **objective measurement** of tinyMem's value proposition.

### Key Findings

- **AI systems with persistent memory deliver 26% higher accuracy** ([Digital Bricks, 2026](https://www.digitalbricks.ai/blog-posts/2026-the-year-of-the-ai-agent))
- **Memory-enabled AI generates 116-446% ROI** vs modest/negative returns from generic deployments ([Medium, 2025](https://medium.com/@johnpettynaible/why-memory-changes-enterprise-ai-in-2026-de5ebf07e5c8))
- **82% of organizations plan to adopt AI agents by 2026** ([Cubeo AI, 2026](https://www.cubeo.ai/top-10-ai-agent-trends-redefining-2026-with-data-that-proves-it/))

### tinyMem Target KPIs (Industry-Aligned)

| Feature | Metric | Industry Standard | tinyMem Target |
|---------|--------|-------------------|----------------|
| **Core** | Memory Retrieval Accuracy | ≥95% | ≥95% |
| **Core** | Response Time | <500ms | <200ms |
| **Semantic** | Recall@10 | 0.6-0.8 | ≥0.7 |
| **Semantic** | NDCG@10 | 0.7-0.9 | ≥0.8 |
| **CoVe** | Hallucination Detection (AUC) | 0.85-0.95 | ≥0.90 |
| **CoVe** | False Positive Rate | <10% | <5% |
| **Ralph** | Plausible@1 (Fix Success) | 60-75% | ≥70% |
| **Ralph** | Compilation Success | 95-98% | ≥96% |

---

## Core System KPIs

### 1. Accuracy Metrics

**Memory Retrieval Accuracy**
- **Definition**: Percentage of queries returning factually correct information
- **Industry Standard**: ≥95% ([Sendbird, 2026](https://sendbird.com/blog/ai-metrics-guide))
- **Measurement**: `(Correct Retrievals / Total Retrievals) × 100`
- **tinyMem Target**: ≥95%

**Task Completion Rate**
- **Definition**: Percentage of user requests successfully fulfilled
- **Industry Standard**: ≥90% ([Pendo, 2026](https://www.pendo.io/essential-kpis-measuring-ai-agent-performance/))
- **Measurement**: `(Successful Tasks / Total Tasks) × 100`
- **tinyMem Target**: ≥92%

### 2. Performance Metrics

**Response Time (Latency)**
- **Definition**: Time from query to result delivery
- **Industry Standard**: <500ms ([Neontri, 2025](https://neontri.com/blog/measure-ai-performance/))
- **Measurement**: P50, P95, P99 latencies
- **tinyMem Target**: P95 <200ms (2.5× better than industry)

**Memory Consumption**
- **Definition**: RAM usage during operations
- **Industry Standard**: Alert threshold at >90% capacity ([Ardor, 2025](https://ardor.cloud/blog/ai-agent-monitoring-essential-metrics-and-best-practices))
- **Measurement**: Peak memory usage, average memory footprint
- **tinyMem Target**: <500MB for typical workloads

### 3. Reliability Metrics

**Error Rate**
- **Definition**: Percentage of failed operations
- **Industry Standard**: <5% failure ([Zapier, 2026](https://zapier.com/blog/ai-metrics/))
- **Measurement**: `(Failed Operations / Total Operations) × 100`
- **tinyMem Target**: <3%

**API Success Rate**
- **Definition**: Percentage of successful API calls
- **Industry Standard**: ≥95% ([Ardor, 2025](https://ardor.cloud/blog/ai-agent-monitoring-essential-metrics-and-best-practices))
- **Measurement**: `(Successful API Calls / Total API Calls) × 100`
- **tinyMem Target**: ≥97%

### 4. Resource Efficiency

**Compute Efficiency**
- **Definition**: Tokens/API calls per user request
- **Industry Standard**: Minimize without sacrificing quality ([Sendbird, 2026](https://sendbird.com/blog/ai-metrics-guide))
- **Measurement**: Tokens consumed, API calls made
- **tinyMem Target**: <1000 tokens per query (with embedded models: 0 external tokens)

---

## Feature-Specific Metrics

### Semantic Search Metrics

**Objective**: Prove semantic search provides **60%+ improvement** over lexical-only recall.

#### Primary Metrics

**1. Recall@k**
- **Definition**: Proportion of relevant documents retrieved in top-k results
- **Industry Standard**: 0.6-0.8 for RAG systems ([Qdrant, 2024](https://qdrant.tech/blog/rag-evaluation-guide/))
- **Formula**: `Recall@k = (Relevant Docs in Top-k) / (Total Relevant Docs)`
- **tinyMem Measurement**:
  - Lexical-only (FTS5): Baseline
  - Semantic (Embedded): Target ≥60% improvement
  - **Target**: Recall@10 ≥ 0.7

**2. Precision@k**
- **Definition**: Proportion of retrieved documents that are relevant
- **Industry Standard**: 0.7-0.9 ([Meilisearch, 2024](https://www.meilisearch.com/blog/rag-evaluation))
- **Formula**: `Precision@k = (Relevant Docs in Top-k) / k`
- **tinyMem Target**: Precision@10 ≥ 0.75

**3. NDCG@k (Normalized Discounted Cumulative Gain)**
- **Definition**: Ranking quality considering position of relevant results
- **Industry Standard**: 0.7-0.9 for high-quality systems ([GeeksforGeeks, 2024](https://www.geeksforgeeks.org/nlp/evaluation-metrics-for-retrieval-augmented-generation-rag-systems/))
- **Formula**: `NDCG@k = DCG@k / IDCG@k`
- **tinyMem Target**: NDCG@10 ≥ 0.8

**4. Mean Reciprocal Rank (MRR)**
- **Definition**: Average reciprocal rank of first relevant result
- **Industry Standard**: 0.6-0.8 ([EvidentlyAI, 2024](https://www.evidentlyai.com/llm-guide/rag-evaluation))
- **Formula**: `MRR = (1/N) Σ (1/rank_i)`
- **tinyMem Target**: MRR ≥ 0.75

#### Comparative Testing

**Test Methodology**:
1. Create benchmark dataset (100 queries with known relevant memories)
2. Run identical queries through:
   - Lexical-only engine (FTS5)
   - Semantic engine (Embedded embeddings)
3. Measure Recall@k, Precision@k, NDCG@k
4. Calculate improvement: `(Semantic - Lexical) / Lexical × 100`

**Success Criteria**: Semantic search shows ≥60% improvement in Recall@10.

---

### CoVe (Hallucination Detection) Metrics

**Objective**: Prove CoVe achieves **100% hallucination detection** (or near-100% AUC ≥0.95).

#### Primary Metrics

**1. Area Under Curve (AUC)**
- **Definition**: Probability that model ranks positive (hallucination) higher than negative (factual)
- **Industry Standard**: 0.85-0.95 for SOTA systems ([arXiv 2512.02772](https://arxiv.org/html/2512.02772))
- **Measurement**: ROC-AUC on hallucination detection task
- **tinyMem Target**: AUC ≥ 0.90

**2. Precision, Recall, F1 for Hallucination Detection**
- **Industry Standard**: Balanced F1 > 0.80 ([OpenReview](https://openreview.net/pdf?id=LYx4w3CAgy))
- **Metrics**:
  - **Hallucination Precision**: `TP / (TP + FP)` — How many flagged hallucinations are real?
  - **Hallucination Recall**: `TP / (TP + FN)` — How many real hallucinations were caught?
  - **F1**: Harmonic mean of Precision and Recall
- **tinyMem Target**:
  - Precision ≥ 0.85 (≤15% false positives)
  - Recall ≥ 0.95 (≥95% hallucinations caught)
  - F1 ≥ 0.90

**3. False Positive Rate (FPR)**
- **Definition**: Percentage of factual statements incorrectly flagged as hallucinations
- **Industry Standard**: <10% ([AI Multiple, 2026](https://research.aimultiple.com/ai-hallucination/))
- **Formula**: `FPR = FP / (FP + TN)`
- **tinyMem Target**: <5%

**4. Context Sensitivity Ratio (CSR)**
- **Definition**: Ratio of token probabilities with vs without retrieved context
- **Usage**: REFIND uses CSR to detect inconsistencies ([arXiv 2601.09929](https://arxiv.org/pdf/2601.09929))
- **tinyMem Application**: Monitor CSR to ensure CoVe verification is grounded

#### Comparative Testing

**Test Methodology**:
1. Create hallucination benchmark:
   - 50 factual statements (ground truth verified)
   - 50 hallucinated statements (known false)
2. Run CoVe verification on all 100 statements
3. Measure: AUC, Precision, Recall, F1, FPR
4. Compare against baseline (no CoVe)

**Success Criteria**: CoVe detects ≥95% of hallucinations with ≤5% false positives.

---

### Ralph (Autonomous Repair) Metrics

**Objective**: Prove Ralph successfully repairs **70%+ of failures** autonomously.

#### Primary Metrics

**1. Plausible@k (Fix Success Rate)**
- **Definition**: Percentage of bugs fixed with k patch attempts
- **Industry Standard**:
  - Plausible@1: 60-75% for LLM-based APR ([arXiv 2506.03283](https://arxiv.org/html/2506.03283v1))
  - Plausible@5: 75-90%
- **Measurement**: `(Successful Fixes with ≤k attempts) / (Total Bugs)`
- **tinyMem Target**:
  - Plausible@1 ≥ 70%
  - Plausible@5 ≥ 85%

**2. Compilation Success Rate**
- **Definition**: Percentage of generated patches that compile
- **Industry Standard**: 95-98.5% ([arXiv 2510.06187](https://arxiv.org/html/2510.06187v2))
- **Measurement**: `(Compilable Patches / Total Patches) × 100`
- **tinyMem Target**: ≥96%

**3. Correct vs Plausible Patches**
- **Definition**: Plausible patches pass tests; correct patches satisfy actual requirements
- **Industry Standard**: Address test overfitting ([ScienceDirect](https://www.sciencedirect.com/science/article/abs/pii/S0164121220302156))
- **Measurement**: Human evaluation or extended test suite
- **tinyMem Target**: Correct@1 ≥ 60% (of plausible patches are actually correct)

**4. Iteration Efficiency**
- **Definition**: Average number of iterations to successful fix
- **Industry Standard**: Minimize iterations to reduce cost ([Springer](https://link.springer.com/article/10.1007/s10115-025-02383-9))
- **Formula**: `Avg Iterations = Σ(Iterations per Bug) / (Successful Fixes)`
- **tinyMem Target**: ≤3 iterations on average

**5. Multi-Dimensional Quality**
- **Structural Preservation (SP)**: How well does patch preserve code structure?
- **Logical Preservation (LP)**: How well does patch preserve original logic?
- **Edit Distance**: Number of changes required
- **tinyMem Target**: SP ≥ 0.8, LP ≥ 0.8, minimal edit distance

#### Comparative Testing

**Test Methodology**:
1. Create repair benchmark:
   - 50 failing test scenarios (compilation errors, test failures, runtime errors)
   - Known fixes documented
2. Run Ralph autonomous repair loop
3. Measure: Plausible@k, Compilation Success, Iteration count
4. Human evaluation for correctness

**Success Criteria**: Ralph achieves Plausible@1 ≥70% with ≥96% compilation success.

---

## Implementation Roadmap

### Phase 1: Metrics Infrastructure (Week 1)

**Tasks**:
- [ ] Implement metrics collection for all LLM providers (✅ COMPLETE)
- [ ] Create metrics export API (JSON, Prometheus format)
- [ ] Add metrics endpoints to health/doctor commands
- [ ] Build metrics visualization dashboard (optional)

**Deliverables**:
- Metrics SDK (`internal/metrics/`)
- Export to `metrics.json` on demand
- CLI: `tinymem metrics --export`

---

### Phase 2: Semantic Search Evaluation (Week 2)

**Tasks**:
- [ ] Create benchmark dataset (100 queries, labeled relevance)
- [ ] Implement Recall@k, Precision@k, NDCG@k calculators
- [ ] Run comparative tests: Lexical vs Semantic
- [ ] Generate evaluation report

**Deliverables**:
- `test/benchmark_semantic.sh` — Automated semantic evaluation
- `benchmarks/semantic_dataset.json` — Benchmark queries + expected results
- Report: `results/semantic_evaluation_report.md`

**Success Metrics**:
- Recall@10 improvement ≥60%
- NDCG@10 ≥ 0.8
- Report published to docs/

---

### Phase 3: CoVe Hallucination Detection (Week 3)

**Tasks**:
- [ ] Create hallucination benchmark (50 factual + 50 false statements)
- [ ] Implement AUC, Precision, Recall, F1 calculators
- [ ] Run CoVe on benchmark dataset
- [ ] Compare against baseline (no verification)

**Deliverables**:
- `test/benchmark_cove.sh` — Automated CoVe evaluation
- `benchmarks/cove_hallucination_dataset.json` — Labeled statements
- Report: `results/cove_evaluation_report.md`

**Success Metrics**:
- AUC ≥ 0.90
- Precision ≥ 0.85, Recall ≥ 0.95
- FPR ≤ 5%

---

### Phase 4: Ralph Autonomous Repair (Week 4)

**Tasks**:
- [ ] Create repair benchmark (50 failing scenarios)
- [ ] Implement Plausible@k, Compilation Success calculators
- [ ] Run Ralph on benchmark with max_iterations=5
- [ ] Human evaluation of correctness

**Deliverables**:
- `test/benchmark_ralph.sh` — Automated Ralph evaluation
- `benchmarks/ralph_repair_dataset/` — Test scenarios + expected fixes
- Report: `results/ralph_evaluation_report.md`

**Success Metrics**:
- Plausible@1 ≥ 70%
- Compilation Success ≥ 96%
- Correct@1 ≥ 60%

---

### Phase 5: Integration & Reporting (Week 5)

**Tasks**:
- [ ] Aggregate all metrics into unified dashboard
- [ ] Create comprehensive evaluation report
- [ ] Publish results to README.md
- [ ] Document KPI tracking process

**Deliverables**:
- `tinymem metrics --all` — Unified metrics command
- `docs/EVALUATION_RESULTS.md` — Published benchmark results
- Updated README with "Proven Benefits" section

---

## References

### AI Agent Memory Systems
- [10 essential KPIs to prove the value of AI Agents](https://www.pendo.io/essential-kpis-measuring-ai-agent-performance/)
- [AI Agent Monitoring: Essential Metrics and Best Practices](https://ardor.cloud/blog/ai-agent-monitoring-essential-metrics-and-best-practices)
- [Memory in the Age of AI Agents](https://arxiv.org/abs/2512.13564)
- [AI Metrics: How to Measure and Evaluate AI Performance](https://sendbird.com/blog/ai-metrics-guide)
- [AI metrics: 6 ways to measure AI performance](https://zapier.com/blog/ai-metrics/)
- [How to Measure AI KPI: Critical Metrics That Matter Most](https://neontri.com/blog/measure-ai-performance/)
- [Why Memory Changes Enterprise AI in 2026](https://medium.com/@johnpettynaible/why-memory-changes-enterprise-ai-in-2026-de5ebf07e5c8)
- [Top 10 AI Agent Trends Redefining 2026](https://www.cubeo.ai/top-10-ai-agent-trends-redefining-2026-with-data-that-proves-it/)

### Hallucination Detection & CoVe
- [awesome-hallucination-detection](https://github.com/EdinburghNLP/awesome-hallucination-detection)
- [Hallucination Detection and Mitigation in Large Language Models](https://arxiv.org/pdf/2601.09929)
- [LLM-Check: Investigating Detection of Hallucinations](https://openreview.net/pdf?id=LYx4w3CAgy)
- [Towards Unification of Hallucination Detection and Fact Verification](https://arxiv.org/html/2512.02772)
- [Hallucination Detection and Evaluation of Large Language Model](https://www.arxiv.org/pdf/2512.22416)
- [Survey and analysis of hallucinations in LLMs](https://pmc.ncbi.nlm.nih.gov/articles/PMC12518350/)
- [AI Hallucination: Compare top LLMs](https://research.aimultiple.com/ai-hallucination/)

### Semantic Search & RAG Evaluation
- [A complete guide to RAG evaluation](https://www.evidentlyai.com/llm-guide/rag-evaluation)
- [Semantic search vs. RAG: A side-by-side comparison](https://www.meilisearch.com/blog/semantic-search-vs-rag)
- [RAG evaluation: Metrics, methodologies, best practices](https://www.meilisearch.com/blog/rag-evaluation)
- [Best Practices in RAG Evaluation](https://qdrant.tech/blog/rag-evaluation-guide/)
- [RAG evaluation: a technical guide](https://toloka.ai/blog/rag-evaluation-a-technical-guide-to-measuring-retrieval-augmented-generation/)
- [Evaluation Metrics for RAG Systems](https://www.geeksforgeeks.org/nlp/evaluation-metrics-for-retrieval-augmented-generation-rag-systems/)

### Autonomous Code Repair (Ralph)
- [Advancements in automated program repair](https://link.springer.com/article/10.1007/s10115-025-02383-9)
- [RELREPAIR: Retrieving Relevant Code](https://arxiv.org/pdf/2509.16701)
- [A Survey of Learning-based APR](https://dl.acm.org/doi/10.1145/3631974)
- [A critical review on APR evaluation](https://www.sciencedirect.com/science/article/abs/pii/S0164121220302156)
- [Comprehensive evaluations of APR benchmarks](https://arxiv.org/html/2508.15135)
- [Automated Program Repair of Uncompilable Student Code](https://arxiv.org/html/2510.06187v2)
- [Empirical Evaluation of Generalizable APR with LLMs](https://arxiv.org/html/2506.03283v1)

---

## Summary

tinyMem's metrics framework is grounded in **2026 industry standards** from academic research, production AI systems, and best practices. By measuring against these defendable KPIs, we can:

1. **Prove tinyMem's core value**: Memory systems deliver 26% higher accuracy and 116-446% ROI
2. **Validate Semantic Search**: Target 60%+ improvement in Recall@10 over lexical-only
3. **Validate CoVe**: Target 90%+ AUC for hallucination detection with <5% FPR
4. **Validate Ralph**: Target 70%+ Plausible@1 with 96%+ compilation success

These metrics enable **objective, industry-standard evaluation** of tinyMem's benefits, providing defendable proof of value for each feature.
