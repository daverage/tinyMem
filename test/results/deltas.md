# tinyMem Pairwise Deltas

## Executive Summary

- tinyMem core reduced TokensPerSuccess by 37.4% vs baseline (strong)
- tinyMem core eliminated false success claims vs baseline (strong)
- tinyMem core improved LLM honesty by reducing false success rate by 100.0% vs baseline (strong)
- tinyMem + CoVe reduced TokensPerSuccess by 31.2% vs tinyMem core (strong)
- tinyMem + Ralph reduced TokensPerSuccess by 8.3% vs tinyMem core (neutral)
- tinyMem + CoVe + Ralph + Semantic increased TokensPerSuccess by 27.3% vs tinyMem + CoVe + Ralph (regression)

## Detailed Deltas

| Comparison | Metric | Base | New | Delta | Classification |
| :--- | :--- | :--- | :--- | :--- | :--- |
| baseline → tinyMem core | TokensPerSuccess | 6231.40 | 3900.00 | -37.4% | strong |
| baseline → tinyMem core | SuccessRate | 0.00 | 0.28 | +0.0% | strong |
| baseline → tinyMem core | FalseSuccessRate | 0.25 | 0.00 | -100.0% | strong |
| baseline → tinyMem core | AvgRetriesPerSuccess | 0.00 | 0.18 | +0.0% | neutral |
| baseline → tinyMem core | ContextTokens | 0.00 | 22.50 | +0.0% | neutral |
| tinyMem core → tinyMem + CoVe | TokensPerSuccess | 3900.00 | 2681.25 | -31.2% | strong |
| tinyMem core → tinyMem + CoVe | SuccessRate | 0.28 | 0.40 | +45.5% | strong |
| tinyMem core → tinyMem + CoVe | FalseSuccessRate | 0.00 | 0.00 | +0.0% | neutral |
| tinyMem core → tinyMem + CoVe | AvgRetriesPerSuccess | 0.18 | 0.75 | +312.5% | neutral |
| tinyMem core → tinyMem + CoVe | ContextTokens | 22.50 | 22.50 | +0.0% | neutral |
| tinyMem core → tinyMem + Ralph | TokensPerSuccess | 3900.00 | 3575.00 | -8.3% | neutral |
| tinyMem core → tinyMem + Ralph | SuccessRate | 0.28 | 0.30 | +9.1% | neutral |
| tinyMem core → tinyMem + Ralph | FalseSuccessRate | 0.00 | 0.00 | +0.0% | neutral |
| tinyMem core → tinyMem + Ralph | AvgRetriesPerSuccess | 0.18 | 0.33 | +83.3% | neutral |
| tinyMem core → tinyMem + Ralph | ContextTokens | 22.50 | 22.50 | +0.0% | neutral |
| tinyMem + CoVe + Ralph → tinyMem + CoVe + Ralph + Semantic | TokensPerSuccess | 3064.29 | 3900.00 | +27.3% | regression |
| tinyMem + CoVe + Ralph → tinyMem + CoVe + Ralph + Semantic | SuccessRate | 0.35 | 0.28 | -21.4% | regression |
| tinyMem + CoVe + Ralph → tinyMem + CoVe + Ralph + Semantic | FalseSuccessRate | 0.00 | 0.00 | +0.0% | neutral |
| tinyMem + CoVe + Ralph → tinyMem + CoVe + Ralph + Semantic | AvgRetriesPerSuccess | 0.57 | 0.27 | -52.3% | neutral |
| tinyMem + CoVe + Ralph → tinyMem + CoVe + Ralph + Semantic | ContextTokens | 22.50 | 22.50 | +0.0% | neutral |
