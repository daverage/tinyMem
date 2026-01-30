# tinyMem Local LLM Configuration Guide

This guide covers configuring generic Local LLM runners (Ollama, LM Studio) to work as **backends** for tinyMem.

tinyMem (Proxy) -> **Local LLM Runner**

---

## 1. Ollama

[Ollama](https://ollama.ai) provides an OpenAI-compatible API by default.

### Setup
1.  Run Ollama: `ollama serve`
2.  Pull a model: `ollama pull llama3`

### tinyMem Config (`.tinyMem/config.toml`)

```toml
[proxy]
port = 8080
base_url = "http://localhost:11434/v1" # Ollama's default OpenAI endpoint

[llm]
model = "llama3" # Must match 'ollama list'
```

---

## 2. LM Studio

[LM Studio](https://lmstudio.ai) is a GUI for running local models.

### Setup
1.  Load a model in LM Studio.
2.  Go to the **Local Server** tab (double-arrow icon).
3.  **Start Server**. Default port is `1234`.

### tinyMem Config (`.tinyMem/config.toml`)

```toml
[proxy]
port = 8080
base_url = "http://localhost:1234/v1"

[llm]
# LM Studio often uses "local-model" or the exact filename.
# Check the "Model Identifier" field in the server tab.
model = "llama-3-8b-instruct"
```

---

## 3. Llama.cpp (Server)

If running `llama-server` directly:

### Setup
```bash
./llama-server -m models/7B/ggml-model-q4_0.gguf -c 2048 --port 8000
```

### tinyMem Config (`.tinyMem/config.toml`)

```toml
[proxy]
port = 8080
base_url = "http://localhost:8000/v1"

[llm]
model = "default" # Llama.cpp server usually ignores model name if only one is loaded
```

## Configuration Reference

For full configuration options, see [Configuration.md](Configuration.md).

## Troubleshooting

-   **"Connection refused"**: Ensure your local runner (Ollama/LM Studio) is actually running and listening on the expected port.
-   **Context Limit Errors**: Local models often have smaller context windows. Decrease `max_items` in `[recall]` if you hit limits.