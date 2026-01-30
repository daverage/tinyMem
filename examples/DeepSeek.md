# tinyMem + DeepSeek Integration

DeepSeek (V2, R1, etc.) provides an OpenAI-compatible API. You can use it as a backend for tinyMem.

## Option 1: DeepSeek API (Cloud)

Configure tinyMem to proxy requests to DeepSeek's cloud API.

### `.tinyMem/config.toml`

```toml
[proxy]
port = 8080
base_url = "https://api.deepseek.com/v1"

[llm]
# Use 'deepseek-chat' or 'deepseek-coder'
model = "deepseek-chat"
api_key_env = "DEEPSEEK_API_KEY" # Reads from environment variable
```

### Run
```bash
export DEEPSEEK_API_KEY="sk-..."
tinymem proxy
```

## Option 2: Local DeepSeek (via Ollama)

If running DeepSeek R1 locally via Ollama:

### `.tinyMem/config.toml`

```toml
[proxy]
port = 8080
base_url = "http://localhost:11434/v1"

[llm]
model = "deepseek-r1" # Match your `ollama list` name
```

## Recommended: CoVe for Reasoning Models

DeepSeek R1 is a reasoning model. Enabling Chain-of-Verification (CoVe) works very well with it to filter out hallucinations in memory recall.

```toml
[cove]
enabled = true
confidence_threshold = 0.6
```
