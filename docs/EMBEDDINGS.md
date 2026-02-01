# Embedding System Documentation

tinyMem supports two modes for generating text embeddings used in semantic search:

1. **Local embeddings** (built-in, offline) - Default for full builds
2. **HTTP embeddings** (external service) - Fallback for lightweight builds

## Quick Start

### Full Build (Recommended for Offline Use)

Build with embedded model (~150-200 MB):

```bash
make build-full
# or
TINYMEM_EXTRA_BUILD_TAGS="embeddings" ./build/build.sh
```

Enable semantic search in configuration:

```toml
[recall]
semantic_enabled = true
```

**That's it!** The embedded model will be used automatically.

### Lightweight Build (HTTP Fallback)

Build without embedded model (~50-60 MB):

```bash
make build
# or
./build/build.sh
```

Configure HTTP endpoint:

```toml
[recall]
semantic_enabled = true

[embedding]
base_url = "http://localhost:11434"
model = "nomic-embed-text"
```

Start Ollama or another embedding service:

```bash
ollama pull nomic-embed-text
ollama serve
```

## Architecture

### Auto-Detection

tinyMem automatically selects the best available embedder:

1. **Try local first** - If built with `-tags embeddings`, use embedded model
2. **Fallback to HTTP** - If local unavailable and `embedding.base_url` configured
3. **Graceful degradation** - If no embedder available, fall back to lexical search

### Embedder Interface

All embedding providers implement a common interface:

```go
type Embedder interface {
    GenerateEmbedding(text string) ([]float32, error)
}
```

This allows seamless switching between local and HTTP modes.

## Local Embeddings (Embedded Model)

### How It Works

1. Model is embedded in the binary using Go's `//go:embed` directive
2. On first use, model bytes are written to a temporary file
3. kelindar/search library loads the model using llama.cpp
4. Embeddings are generated on-device (CPU or GPU)

### Model Information

**Model:** nomic-embed-text-v1.5 (Q4_K_M quantization)

| Property | Value |
|----------|-------|
| **Size** | ~80 MB (90 MB embedded) |
| **Dimensions** | 768 |
| **Context Length** | 8192 tokens |
| **Quantization** | Q4_K_M (4-bit mixed) |
| **Performance** | ~50-100ms per embedding (CPU) |
| **Memory Usage** | ~200-400 MB RSS |
| **License** | Apache 2.0 |

**Source:** [nomic-ai/nomic-embed-text-v1.5-GGUF](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF)

### Build Configuration

**Build Tags:**

- `fts5` - Required for full-text search (always needed)
- `embeddings` - Include embedded model for local inference
- `nollm` - Disable CoVe and Ralph features (keeps semantic search working)

**Examples:**

```bash
# Full build with embeddings
go build -tags "fts5 embeddings" -o tinymem ./cmd/tinymem

# Lightweight build (HTTP only)
go build -tags "fts5" -o tinymem ./cmd/tinymem

# Minimal build (no CoVe, no Ralph, semantic via HTTP)
go build -tags "fts5 nollm" -o tinymem ./cmd/tinymem

# Semantic-only build (embedded search, no LLM features)
go build -tags "fts5 embeddings nollm" -o tinymem ./cmd/tinymem
```

**Feature Matrix by Build Tags:**

| Features | `fts5` | `fts5 embeddings` | `fts5 nollm` | `fts5 embeddings nollm` |
|----------|--------|-------------------|--------------|-------------------------|
| **Lexical Search** | ✅ | ✅ | ✅ | ✅ |
| **Semantic Search** | HTTP only | ✅ Embedded | HTTP only | ✅ Embedded |
| **CoVe** | ✅ | ✅ | ❌ | ❌ |
| **Ralph** | ✅ | ✅ | ❌ | ❌ |
| **Binary Size** | ~15 MB | ~96 MB | ~14 MB | ~95 MB |

**When to use `nollm` tag:**

- You only want semantic search (embeddings), not fact verification or autonomous repair
- You're deploying in constrained environments where text generation models are overkill
- You want faster builds (fewer dependencies to compile)
- You're running in MCP mode where the calling AI handles verification/repairs itself

**Using Makefile:**

```bash
# Full build
make build-full

# Lightweight build
make build
```

**Using build scripts:**

```bash
# Full build (bash)
TINYMEM_EXTRA_BUILD_TAGS="embeddings" ./build/build.sh

# Full build (Windows)
set TINYMEM_EXTRA_BUILD_TAGS=embeddings
.\build\build.bat
```

### Benefits

✅ **Works completely offline** - No external dependencies
✅ **Fast startup** - No network latency
✅ **Consistent results** - Same model version everywhere
✅ **Single binary** - Easy deployment
✅ **No configuration needed** - Works out of the box

### Trade-offs

❌ **Larger binary** - ~150-200 MB vs ~50-60 MB lightweight
❌ **CPU-bound** - Slower than GPU-accelerated services
❌ **Memory usage** - ~200-400 MB RSS when active
❌ **Fixed model** - Can't change without rebuilding
❌ **Platform-specific compilation** - Requires libllama_go shared library:
  - **Linux/Windows**: Precompiled binaries included
  - **macOS**: Requires manual compilation from source (see below)

## HTTP Embeddings (External Service)

### How It Works

1. tinyMem makes HTTP requests to an embedding service (Ollama, OpenAI, etc.)
2. Service generates embeddings using its own models
3. Embeddings are returned via JSON API
4. Compatible with OpenAI embedding API format

### Configuration

```toml
[recall]
semantic_enabled = true

[embedding]
base_url = "http://localhost:11434"  # Service endpoint
model = "nomic-embed-text"            # Model name
```

### Supported Services

#### Ollama (Recommended)

```bash
# Install and run Ollama
ollama pull nomic-embed-text
ollama serve
```

Configuration:

```toml
[embedding]
base_url = "http://localhost:11434"
model = "nomic-embed-text"
```

#### OpenAI

Configuration:

```toml
[embedding]
base_url = "https://api.openai.com/v1"
model = "text-embedding-3-small"

[llm]
api_key = "sk-..."  # Set your API key
```

#### Other Services

Any service implementing the OpenAI embeddings API format:

```http
POST /v1/embeddings
Content-Type: application/json

{
  "model": "model-name",
  "input": ["text to embed"]
}
```

### Benefits

✅ **Smaller binary** - ~50-60 MB
✅ **Flexible models** - Change model without rebuilding
✅ **GPU acceleration** - If service supports it
✅ **Latest models** - Update service independently

### Trade-offs

❌ **External dependency** - Requires running service
❌ **Network latency** - HTTP overhead
❌ **Service availability** - Fails if service down
❌ **Configuration required** - Must set base_url

## Configuration Reference

### Minimal (Local Embeddings)

Full builds only need to enable semantic search:

```toml
[recall]
semantic_enabled = true
```

### HTTP Fallback

Lightweight builds or custom models:

```toml
[recall]
semantic_enabled = true

[embedding]
base_url = "http://localhost:11434"
model = "nomic-embed-text"
```

### Environment Variables

Override configuration via environment:

```bash
# Enable semantic search
export TINYMEM_SEMANTIC_ENABLED=true

# Configure HTTP endpoint (optional)
export TINYMEM_EMBEDDING_BASE_URL="http://localhost:11434"
export TINYMEM_EMBEDDING_MODEL="nomic-embed-text"
```

## Performance Comparison

| Mode | First Embedding | Subsequent | Memory | Network |
|------|----------------|-----------|---------|---------|
| **Local (CPU)** | ~100-200ms | ~50-100ms | ~300 MB | None |
| **HTTP (Ollama)** | ~50-100ms | ~50-100ms | Minimal | Required |
| **HTTP (OpenAI)** | ~200-500ms | ~200-500ms | Minimal | Required |

*Benchmarks on Apple M1 Pro, varies by hardware*

## Troubleshooting

### "local embeddings not available" Error

**Cause:** Binary was built without `-tags embeddings`

**Solution:** Rebuild with embeddings tag:

```bash
make build-full
```

### "Library 'libllama_go.dylib' not found" Error (macOS)

**Cause:** kelindar/search requires a compiled shared library that is not precompiled for macOS

**Solution 1:** Compile the library from source:

```bash
# Clone kelindar/search with submodules
cd /tmp
git clone --recurse-submodules https://github.com/kelindar/search.git
cd search
git lfs pull

# Compile the library (requires CMake and C++ compiler)
mkdir build && cd build
cmake -DBUILD_SHARED_LIBS=ON -DCMAKE_BUILD_TYPE=Release ..
cmake --build . --config Release

# Copy to system library path
sudo cp libllama_go.dylib /usr/local/lib/

# OR copy to tinyMem project directory
cp libllama_go.dylib /path/to/tinyMem/

# Rebuild tinyMem
cd /path/to/tinyMem
make build-full
```

**Solution 2:** Use HTTP mode instead (recommended for macOS):

```bash
# Build lightweight version (no compilation needed)
make build

# Configure HTTP endpoint
cat > .tinyMem/config.toml << EOF
[recall]
semantic_enabled = true

[embedding]
base_url = "http://localhost:11434"
model = "nomic-embed-text"
EOF

# Start Ollama
ollama pull nomic-embed-text
ollama serve
```

### "embedding request failed" Error

**Cause:** HTTP service not running or unreachable

**Solutions:**

1. Check service is running: `curl http://localhost:11434/v1/models`
2. Verify `base_url` in config matches service
3. Check firewall/network settings
4. Use local embeddings instead (rebuild with full tags)

### Model File Missing

**Error:** `no such file or directory: internal/embedding/models/...`

**Cause:** Model file not downloaded before build

**Solution:**

```bash
# Download model (run from project root)
curl -L -o internal/embedding/models/nomic-embed-text-v1.5.Q4_K_M.gguf \
  https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf

# Rebuild
make build-full
```

### High Memory Usage

**Cause:** Model loaded into memory

**Expected:** ~200-400 MB RSS when semantic search active

**Solutions:**

- Use HTTP mode instead (minimal memory)
- Disable semantic search: `semantic_enabled = false`
- Increase available RAM

### "CoVe is unavailable in no-llm builds" Error

**Cause:** Binary was built with `-tags nollm` which disables LLM-dependent features

**Why:** CoVe (Chain-of-Verification) requires a text generation LLM to verify memory claims. This is different from embeddings (which convert text to vectors for semantic search).

**Solutions:**

1. **Rebuild without nollm tag:**
   ```bash
   go build -tags "fts5 embeddings" -o tinymem ./cmd/tinymem
   ```

2. **Use MCP mode instead** - The calling AI (Claude, etc.) can handle verification itself

3. **Disable CoVe in config** - If you don't need fact verification:
   ```toml
   [cove]
   enabled = false
   ```

**Note:** Semantic search still works in `nollm` builds because it only uses embeddings, not text generation.

### "Ralph is unavailable in no-llm builds" Error

**Cause:** Binary was built with `-tags nollm` which disables LLM-dependent features

**Why:** Ralph (autonomous repair loop) requires a text generation LLM to generate code fixes. This is different from embeddings.

**Solutions:**

1. **Rebuild without nollm tag:**
   ```bash
   go build -tags "fts5 embeddings" -o tinymem ./cmd/tinymem
   ```

2. **Use MCP mode instead** - The calling AI can generate repairs itself

**Note:** Ralph is primarily designed for MCP mode where AI agents call tinyMem, not for direct CLI use.

## Alternative Models

### Using Different GGUF Models

You can substitute the embedded model:

1. Download alternative GGUF model to `internal/embedding/models/`
2. Update `//go:embed` path in `internal/embedding/local.go`
3. Rebuild with `-tags embeddings`

**Options:**

| Model | Size | Quality | Speed | Dimensions |
|-------|------|---------|-------|------------|
| **all-MiniLM-L6-v2** (Q4) | ~40 MB | Good | Fast | 384 |
| **nomic-embed-text-v1.5** (Q4_K_M) | ~80 MB | Better | Medium | 768 |
| **nomic-embed-text-v1.5** (Q8) | ~180 MB | Best | Slower | 768 |

**Trade-off:** Larger models = better quality, slower inference, more memory

### Using HTTP Mode with Different Models

HTTP mode supports any model your service provides:

```toml
[embedding]
base_url = "http://localhost:11434"
model = "mxbai-embed-large"  # or any Ollama model
```

## Best Practices

### For Production Deployments

✅ **Use full builds** - Include embedded model for reliability
✅ **Enable semantic search** - Improves recall quality
✅ **Monitor memory** - Ensure sufficient RAM (~500 MB minimum)
✅ **Test offline** - Verify works without network

### For Development

✅ **Use lightweight builds** - Faster iteration
✅ **Run local Ollama** - Easy to swap models
✅ **Use HTTP mode** - Flexibility to experiment

### For CI/CD

✅ **Lightweight builds** - Faster tests, smaller artifacts
✅ **Mock HTTP service** - Test without actual embeddings
✅ **Disable semantic** - Use lexical search for speed

## References

- [kelindar/search](https://github.com/kelindar/search) - Go embedding library
- [Nomic AI Models](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF) - Model source
- [GGUF Format](https://github.com/ggerganov/ggml/blob/master/docs/gguf.md) - Model format spec
- [llama.cpp](https://github.com/ggerganov/llama.cpp) - Underlying inference engine
