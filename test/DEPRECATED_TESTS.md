# Deprecated Tests

The following tests have been removed as they relied on embedded text generation LLM which has been removed from the architecture.

## Removed Files

### benchmark_cove_kpi.sh
- **Why removed:** Required embedded LLM (llmgen build) to test CoVe
- **Replaced by:** test/automated/cove_test.go (proves CoVe works without LLM using semantic scores)

### test_cove.sh  
- **Why removed:** Tested CoVe with LLM-based scoring
- **Replaced by:** test/automated/cove_test.go

### benchmark_semantic_kpi.sh
- **Why removed:** Bash 4+ requirement, flaky bash arrays
- **Replaced by:** test/automated/ Go tests (deterministic, CI-friendly)

### run_kpi_benchmarks.sh
- **Why removed:** Orchestrated deprecated tests
- **Replaced by:** test/run_verification.sh (new framework)

### test/test_embedded_llm.sh
- **Why removed:** Tested broken embedded text generation
- **Architecture change:** Embedded text generation removed entirely

## New Test Architecture

All testing now uses:
- **Go tests** in test/automated/ (CI-enforceable, deterministic)
- **Isolated environments** (temporary directories, no contamination)
- **Zero LLM calls** for CoVe (semantic similarity scoring)
- **Evidence gating** for Ralph (orchestration-controlled)
- **Structured metrics** (JSON/CSV output, no prose)

See test/automated/README.md for new framework documentation.
