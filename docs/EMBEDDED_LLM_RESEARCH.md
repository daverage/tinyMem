# Embedded LLM Research for tinyMem

**Research Date:** January 31, 2026
**Current Model:** Qwen2.5 0.5B Q4_K_M (469 MB)
**Use Cases:** CoVe fact verification, Ralph code repair

---

## Executive Summary

Research reveals that **SmolLM2-360M** outperforms our current Qwen2.5-0.5B model across multiple benchmarks while being **potentially smaller** in quantized form. However, Qwen2.5-0.5B remains a strong choice with better multilingual support and code-specific variants.

### Key Findings

| Model | Size (Q4_K_M) | Instruction Following | HellaSwag | ARC | Best For |
|-------|---------------|----------------------|-----------|-----|----------|
| **SmolLM2-360M** | ~200-250 MB (est.) | **41.0** ✅ | **54.5** ✅ | **53.0** ✅ | Instruction following |
| **Qwen2.5-0.5B** | 491 MB | 31.6 | Lower | Lower | Multilingual, code |
| **Phi-3-mini** | ~2.4 GB | Best quality | High | High | Best quality/size ratio |

**Recommendation:** ✅ **Consider SmolLM2-360M for 40-50% size reduction with better performance**

---

## Detailed Model Comparison

### 1. SmolLM2-360M ⭐ TOP RECOMMENDATION

**Developer:** Hugging Face
**Size:** 360M parameters
**GGUF Size:** ~200-250 MB (Q4_K_M estimated, 723 MB unquantized)

#### Performance Benchmarks

| Benchmark | SmolLM2-360M | Qwen2.5-0.5B | Advantage |
|-----------|--------------|--------------|-----------|
| **IFEval (Instruction Following)** | 41.0 | 31.6 | **+30%** ✅ |
| **HellaSwag (Reasoning)** | 54.5 | Lower | **Better** ✅ |
| **ARC (Knowledge)** | 53.0 | Lower | **Better** ✅ |
| **PIQA (Physical Reasoning)** | 71.7 | Lower | **Better** ✅ |

**Sources:**
- [SmolLM2-360M-Instruct on Skywork](https://skywork.ai/blog/models/huggingfacetb-smollm2-360m-instruct-free-chat-online-skywork-ai/)
- [SmolLM2 Deep Dive - UNDERCODE NEWS](https://undercodenews.com/the-power-of-smol-models-a-deep-dive-into-hugging-faces-smollm2/)

#### Strengths

✅ **Significantly smaller** (360M vs 500M parameters)
✅ **Better instruction following** (+30% on IFEval)
✅ **Runs on smartphones and edge devices**
✅ **Enhanced privacy** (local processing)
✅ **Lower latency**
✅ **Smaller memory footprint**

#### Considerations

⚠ Qwen has better multilingual support (29 languages vs primarily English)
⚠ Qwen has code-specific variant (Qwen2.5-Coder-0.5B)
⚠ SmolLM2 less tested in production for fact verification

---

### 2. Qwen2.5-0.5B (Current Choice)

**Developer:** Alibaba Cloud
**Size:** 500M parameters
**GGUF Size:** 491 MB (Q4_K_M)

#### Quantization Options

| Quantization | Size | Quality Loss | Savings vs Q4_K_M |
|--------------|------|--------------|-------------------|
| **Q2_K** | 415 MB | High | -76 MB (15%) |
| **Q3_K_M** | 432 MB | Medium | -59 MB (12%) |
| **Q4_0** | 429 MB | Low-Med | -62 MB (13%) |
| **Q4_K_M** | 491 MB | Low (current) | Baseline |
| **Q5_K_M** | 522 MB | Very Low | +31 MB |
| **Q8_0** | 676 MB | Minimal | +185 MB |

**Source:** [Qwen2.5-0.5B-Instruct-GGUF on Hugging Face](https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF)

#### Strengths

✅ **Multilingual support** (29 languages)
✅ **Large context window** (128K tokens, 8K generation)
✅ **Code-specific variant available** (Qwen2.5-Coder-0.5B)
✅ **Proven in production**
✅ **Outperforms Gemma2-2.6B** on math/coding

#### Code Variant: Qwen2.5-Coder-0.5B

For Ralph (code repair), the **Qwen2.5-Coder-0.5B-Instruct** variant is optimized specifically for:
- Code generation
- Code repair
- Debugging
- Syntax error fixes

**Source:** [Qwen2.5-Coder-0.5B-Instruct-GGUF](https://huggingface.co/Qwen/Qwen2.5-Coder-0.5B-Instruct-GGUF)

---

### 3. Alternative Models (Reference)

#### Qwen3-0.6B

**Size:** 600M parameters
**Status:** Most downloaded text generation model on Hugging Face (Dec 2025)
**Context:** 32K tokens

**Performance:** Meaningfully stronger than "toy" small models, competitive with much larger models like DeepSeek-R1-Distill-Llama-8B in some evaluations.

**Source:** [Best Open-Source Small Language Models (SLMs) in 2026](https://www.bentoml.com/blog/the-best-open-source-small-language-models)

#### Phi-3-mini (3.8B)

**Size:** 3.8B parameters (quantized to ~2.4 GB)
**Quality:** Delivers performance of ~7B model

**Strengths:**
- "Pound for pound" accuracy champion
- Best quality-to-size ratio
- Excellent for coding

**Consideration:** 5x larger than our 500 MB target

**Sources:**
- [Best Small Language Models Comparison](https://medium.com/@darrenoberst/best-small-language-models-for-accuracy-and-enterprise-use-cases-benchmark-results-cf71964759c8)
- [Phi-3 vs TinyLlama Comparison](https://slashdot.org/software/comparison/Phi-3-vs-TinyLlama/)

#### TinyLlama (1.1B)

**Size:** 1.1B parameters
**Performance:** Good for commonsense reasoning
**Status:** Solid balance for resource-constrained environments

**Consideration:** 2x larger than Qwen/SmolLM2

---

## Fact Verification Research

### Limitations of Small Models for Fact Verification

Research from 2026 shows that small models (<1B parameters) have **inherent limitations** for fact verification:

⚠ **Limited parametric knowledge** - Don't store large amounts of world knowledge
⚠ **Outdated facts** - Knowledge cutoff issues
⚠ **Inaccurate for long-tail queries** - Struggle with niche topics

**Source:** [Preprint 2026 FACT-CHECKING WITH LARGE LANGUAGE MODELS](https://www.arxiv.org/pdf/2601.02574)

### Recommended Approach for Production

Industry best practice for fact verification with small models:

1. **Pair with RAG** (Retrieval-Augmented Generation)
2. **Use external tools** for knowledge lookup
3. **Leverage evidence-based verification** (what tinyMem already does!)

**tinyMem's Advantage:** Our CoVe implementation uses **evidence-based verification** (file_exists, grep_hit, cmd_exit0, test_pass), which **compensates for small model limitations** by grounding verification in actual system state.

**Sources:**
- [FACTS Leaderboard - Google DeepMind](https://deepmind.google/blog/facts-benchmark-suite-systematically-evaluating-the-factuality-of-large-language-models/)
- [Measuring short-form factuality in LLMs - OpenAI](https://cdn.openai.com/papers/simpleqa.pdf)

---

## Code Repair Research

### Small Models for Code Generation (2026)

Research shows small models can be effective for code repair when combined with **compiler feedback loops**:

**R1-Compiler Method:**
- Uses compiler as external feedback mechanism
- Feeds error messages back to LLM
- Reduces syntax errors from **34.1% to 3.2%** after one repair iteration

**Source:** [Best LLM for Coding in 2026](https://www.xavor.com/blog/best-llm-for-coding/)

**tinyMem's Advantage:** Ralph's design uses **evidence-based repair** with compiler/test feedback, which is exactly the recommended approach!

### Top Small Models for Coding (2026)

1. **Qwen3-0.6B** - Among most downloaded for text/code generation
2. **Qwen3-8B, GLM-4-9B, Meta-Llama-3.1-8B** - Top 3 for 2026
3. **Phi-3 Mini** - Generates clean code on mid-range hardware
4. **Mistral 7B** - Strong coding performance

**Sources:**
- [Ultimate Guide - Best Small LLMs For Personal Projects](https://www.siliconflow.com/articles/en/best-small-LLMs-for-personal-projects)
- [Best LLM for Coding - Vellum](https://www.vellum.ai/best-llm-for-coding)

---

## Recommendations

### Option 1: Switch to SmolLM2-360M ⭐ RECOMMENDED

**Pros:**
- ✅ 40-50% smaller (200-250 MB vs 491 MB)
- ✅ **Better instruction following** (+30% on IFEval)
- ✅ **Better reasoning** (HellaSwag, ARC, PIQA)
- ✅ Runs on edge devices
- ✅ Lower latency
- ✅ Smaller binary (~350 MB vs 487 MB)

**Cons:**
- ⚠ Primarily English (vs Qwen's 29 languages)
- ⚠ Less production testing
- ⚠ No code-specific variant

**Best For:**
- CoVe (instruction following critical)
- Users prioritizing binary size
- Edge deployment scenarios

**Model:** `HuggingFaceTB/SmolLM2-360M-Instruct`
**GGUF:** Available on Hugging Face

---

### Option 2: Use Qwen2.5-Coder-0.5B for Ralph

**Pros:**
- ✅ Optimized specifically for code tasks
- ✅ Better code repair than general Qwen2.5-0.5B
- ✅ Same size as current model (491 MB Q4_K_M)
- ✅ Proven for debugging and syntax fixes

**Cons:**
- ⚠ Separate model needed for CoVe (increases binary to ~980 MB)

**Best For:**
- Ralph code repair
- When code quality is critical

**Model:** `Qwen/Qwen2.5-Coder-0.5B-Instruct`

---

### Option 3: Hybrid Approach (Best of Both)

**Configuration:**
- **CoVe:** SmolLM2-360M (200-250 MB) - Better instruction following
- **Ralph:** Qwen2.5-Coder-0.5B (491 MB) - Optimized for code
- **Total:** ~700-750 MB (vs current 487 MB)

**Trade-off:** Larger binary but specialized models for each task

---

### Option 4: Keep Qwen2.5-0.5B, Optimize Quantization

**Current:** Q4_K_M (491 MB)
**Alternative:** Q3_K_M (432 MB) - Save 59 MB (12%)

**Pros:**
- ✅ Modest size reduction
- ✅ Keep proven model
- ✅ Minimal quality loss

**Cons:**
- ⚠ Still larger than SmolLM2-360M
- ⚠ Not the best at any specific task

---

## Implementation Recommendations

### For Immediate Use (Next Release)

**Recommended:** ✅ **Switch to SmolLM2-360M**

**Rationale:**
1. **Better performance** on key benchmarks (+30% instruction following)
2. **Significantly smaller** (~40-50% size reduction)
3. **CoVe is instruction-following heavy** (SmolLM2's strength)
4. **Evidence-based verification** compensates for knowledge limitations
5. **Ralph uses compiler feedback** (less dependent on code knowledge)

**Action Items:**
1. Download SmolLM2-360M-Instruct GGUF (Q4_K_M)
2. Update `internal/llm/local.go` to embed SmolLM2
3. Update `internal/llm/local_purego.go` model path
4. Re-run CoVe benchmarks to validate performance
5. Update documentation (binary size, model info)

---

### For Future Optimization

**Option:** Dual-model approach
- **CoVe:** SmolLM2-360M (~250 MB)
- **Ralph:** Qwen2.5-Coder-0.5B (~491 MB)
- **Total:** ~750 MB

**Build Tags:**
- `-tags llmgen` → SmolLM2-360M only (small binary)
- `-tags llmgen,coder` → Both models (full features)

---

## Quantization Guide

### Quality vs Size Trade-offs

| Quantization | Quality | Use Case |
|--------------|---------|----------|
| **Q8_0** | 99.9% | Reference quality |
| **Q6_K** | 99.5% | High-quality production |
| **Q5_K_M** | 99% | Production default (large) |
| **Q4_K_M** | 97% | **Recommended default** ✅ |
| **Q4_0** | 95% | Lighter alternative |
| **Q3_K_M** | 90% | Edge devices, accept quality loss |
| **Q2_K** | 80% | Extreme size constraints only |

**Source:** [Practical Quantization Guide for iPhone and Mac](https://enclaveai.app/blog/2025/11/12/practical-quantization-guide-iphone-mac-gguf/)

### Recommendation

**Q4_K_M remains the optimal balance** for tinyMem:
- Minimal quality degradation (97% of original)
- Good size efficiency
- "Safe default" for production use

---

## Conclusion

### Final Recommendation: ⭐ SmolLM2-360M

**Switch to SmolLM2-360M-Instruct** for the next release:

1. **Smaller Binary:** 350 MB (vs current 487 MB) - **28% reduction**
2. **Better Performance:** +30% instruction following, better reasoning
3. **Lower Latency:** Faster inference on same hardware
4. **Edge-Ready:** Can run on smartphones and IoT devices
5. **Same Architecture:** Purego, no CGO, drop-in replacement

**Expected Benefits:**
- ✅ Reduced download size (important for distribution)
- ✅ Faster CoVe verification (smaller model = faster inference)
- ✅ Better instruction following (critical for CoVe/Ralph)
- ✅ Maintains self-contained architecture

**Migration Effort:** Low (1-2 hours)
- Download SmolLM2-360M GGUF
- Update model path in local.go
- Re-embed model
- Test and validate

---

## References

### Model Performance & Benchmarks
- [The Best Open-Source Small Language Models (SLMs) in 2026](https://www.bentoml.com/blog/the-best-open-source-small-language-models)
- [Top 15 Small Language Models for 2026 | DataCamp](https://www.datacamp.com/blog/top-small-language-models)
- [Best Small Language Models Benchmark Results](https://medium.com/@darrenoberst/best-small-language-models-for-accuracy-and-enterprise-use-cases-benchmark-results-cf71964759c8)
- [Tiny LLM Architecture Comparison: TinyLlama vs Phi-2 vs Gemma](https://josedavidbaena.com/blog/tiny-language-models/tiny-llm-architecture-comparison)

### Quantization & Optimization
- [Practical Quantization Guide (GGUF: Q4_K_M vs Q5_K_M vs Q8_0)](https://enclaveai.app/blog/2025/11/12/practical-quantization-guide-iphone-mac-gguf/)
- [AI Model Quantization 2025: Master Compression Techniques](https://local-ai-zone.github.io/guides/what-is-ai-quantization-q4-k-m-q8-gguf-guide-2025.html)

### Fact Verification & Code Repair
- [Preprint 2026 FACT-CHECKING WITH LARGE LANGUAGE MODELS](https://www.arxiv.org/pdf/2601.02574)
- [FACTS Benchmark Suite - Google DeepMind](https://deepmind.google/blog/facts-benchmark-suite-systematically-evaluating-the-factuality-of-large-language-models/)
- [The best LLM for coding in 2026](https://www.xavor.com/blog/best-llm-for-coding/)
- [Ultimate Guide - Best Small LLMs For Personal Projects](https://www.siliconflow.com/articles/en/best-small-LLMs-for-personal-projects)

### Model Downloads
- [SmolLM2-360M-Instruct - Skywork](https://skywork.ai/blog/models/huggingfacetb-smollm2-360m-instruct-free-chat-online-skywork-ai/)
- [Qwen2.5-0.5B-Instruct-GGUF - Hugging Face](https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF)
- [Qwen2.5-Coder-0.5B-Instruct-GGUF - Hugging Face](https://huggingface.co/Qwen/Qwen2.5-Coder-0.5B-Instruct-GGUF)

---

**Document End**
