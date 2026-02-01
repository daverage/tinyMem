# Embedded Embedding Models

This directory contains embedding models bundled into tinyMem for offline operation.

## Current Model

**Model:** `nomic-embed-text-v1.5` (Q4_K_M quantization)
**Size:** ~90 MB
**Dimensions:** 768
**Source:** Nomic AI
**License:** Apache 2.0

### Model Information

- **Type:** Sentence transformer / embedding model
- **Context Length:** 8192 tokens
- **Use Case:** Semantic search, text similarity
- **Quantization:** Q4_K_M (4-bit mixed quantization)
  - Balanced quality/size trade-off
  - ~90 MB (vs ~550 MB for full fp16)
  - Minimal quality loss for semantic search

### Download Instructions

To download the model for full builds:

```bash
curl -L -o internal/embedding/models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
```

### Build Integration

The model is embedded using Go's `//go:embed` directive in `local.go`:

```go
//go:embed models/nomic-embed-text-v1.5.Q4_K_M.gguf
var embeddedModel []byte
```

This means:
- **Full builds** (with `-tags embeddings`): Model is compiled into the binary
- **Lightweight builds** (without tag): No model included, uses HTTP fallback

### Alternative Models

You can substitute with other GGUF-format embedding models:

- **all-MiniLM-L6-v2** (Q4): ~40 MB, faster, lower quality
- **nomic-embed-text-v1.5** (Q8): ~180 MB, higher quality
- **nomic-embed-text-v1.5** (fp16): ~550 MB, maximum quality

To use a different model:
1. Download the GGUF file to this directory
2. Update the `//go:embed` path in `local.go`
3. Rebuild with `-tags embeddings`

## Performance

Expected performance on typical hardware:

- **First embedding:** ~100-200ms (model loading)
- **Subsequent embeddings:** <50ms
- **Memory usage:** ~200-400 MB RSS (model in memory)

## References

- [Nomic AI Model Card](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF)
- [GGUF Format Specification](https://github.com/ggerganov/ggml/blob/master/docs/gguf.md)
- [Model Quantization Guide](https://github.com/ggerganov/llama.cpp/blob/master/examples/quantize/README.md)
