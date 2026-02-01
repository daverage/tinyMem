# tinyMem Testing Contract & Evaluation Invariants

This document defines the authoritative invariants that the tinyMem testing suite must enforce. All tests in `test/automated` and `test/qualitative` are governed by this contract.

## 1. Architectural Invariants

### CoVe (Confidence Verification)
- **Zero-LLM Mandate**: CoVe must never trigger a text-generation LLM call. It must operate exclusively on semantic similarity scores and deterministic threshold gating.
- **Determinism**: For a fixed set of memories and query, the filtered result set must be identical across runs.
- **Explicit Zero-Recall**: High thresholds must result in an empty set, never a fallback to unverified inference.

### Authority Boundaries
- **Lexical Dominance**: Semantic recall must never suppress or override an exact lexical match in the memory ledger.
- **Evidence-Gated Facts**: No memory can be promoted to a `Fact` without explicit evidence validation. Claims are advisory only.
- **Scope Protection**: File operations and evidence verification must never escape the defined project root.

### Ralph (Autonomous Repair)
- **Evidence-Only Success**: Success can ONLY be signaled by passing evidence predicates (exit codes, file content, etc.). 
- **Bounded Execution**: Loops must be strictly bounded by `max_iterations`.
- **Mock Integrity**: LLM mocks should focus on system response to failure (invalid formatting, incorrect repairs) rather than model quality.

## 2. Evaluation Strategy

### Quantitative (Regression Detection)
- **Token Accounting**: Instrumentation must track tokens per task. CI should compare against baselines and fail on unexplained regressions.
- **Pass/Fail Precision**: Automated tests must use fixed data to ensure machine-verifiable correctness.

### Qualitative (System Outcome Audit)
- **Structured Scenarios**: System behavior is audited using canonical scenarios with fixed constraints and success criteria.
- **Numeric Outcomes**: Audit results must be numeric or boolean (e.g., `RepairIterations`, `EvidenceValidated`). Prose or "vibe-based" reviews are forbidden.

## 3. CI/CD Integration

### Execution
Run the unit and scenario suite using:
```bash
go test -v -tags "fts5" ./test/automated/... ./test/qualitative/...
```

Run the real-world quality evaluation and comparative benchmark harness using:
```bash
go run -tags "fts5 embeddings" test/harness/main.go
```
Set `TINYMEM_BENCH_FULL=true` for 100-run stability and delta analysis across all modes.

### Artifacts
The following artifacts are produced in `test/results/` and should be archived by CI:
- `raw_runs.jsonl`: Individual run metrics for deep analysis.
- `aggregated_metrics.json`: Comparative stats across all modes and scenarios.
- `scorecard.md`: Single table summary of all mode performances.
- `deltas.md`: Pairwise comparisons between modes with improvement classifications.
- `per_scenario_reports.md`: Factual summaries of performance per task scenario.
- `success_rates.svg`: Visual bar chart of Success vs False Success rates.
- `tokens_per_success.svg`: Visual bar chart of token efficiency with error bars.
- `scatter_context_vs_total.svg`: Scatter plot of Context Tokens vs Total Tokens.

### Gating
CI should fail if:
1. Any automated test in `test/automated` fails.
2. `FalseSuccessRate > 0` for any mode including Ralph.
3. `AuthorityOverrideAttempts > 0` detected.
4. `DeterminismViolations > 0` detected in tinyMem modes.
5. Regression in `TokensPerSuccess` exceeds 10% vs baseline or prior mode.

