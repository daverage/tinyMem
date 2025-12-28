# Configuration Changes Summary

**Date:** 2025-12-24
**Changes:** Default configuration updated for LM Studio integration

---

## Changes Made

### 1. Proxy Port Changed
**From:** `127.0.0.1:8080`
**To:** `127.0.0.1:4321`

**Reason:** Port 4321 is less commonly used, reducing conflicts with other local services.

**Impact:**
- TSLP proxy now listens on port 4321 by default
- Update any client configurations to use: `http://localhost:4321/v1/chat/completions`

---

### 2. LLM Provider Configured for LM Studio
**From:**
```toml
llm_provider = "openai"
llm_endpoint = "https://api.openai.com/v1"
llm_api_key = ""
llm_model = "gpt-4"
```

**To:**
```toml
llm_provider = "lmstudio"
llm_endpoint = "http://localhost:1234/v1"
llm_api_key = ""
llm_model = "local-model"
```

**Reason:** LM Studio is a popular local LLM server that runs on port 1234 by default. This provides an out-of-the-box experience for users running small models locally.

**Impact:**
- TSLP now works with LM Studio without configuration changes
- No API key required (local models)
- Users must have LM Studio running on port 1234 with a model loaded

---

## Default Configuration (config/config.toml)

```toml
# TSLP v5.3 (Gold) Configuration
# Per Specification: minimal and boring, no tuning knobs, no feature flags
# All fields are REQUIRED unless explicitly noted

[database]
# Path to SQLite database file
# The database will be created if it doesn't exist
database_path = "./runtime/tslp.db"

[logging]
# Path to log file
log_path = "./runtime/tslp.log"

# Enable debug logging (true/false)
debug = false

[llm]
# LLM provider identifier
# Examples: "lmstudio", "openai", "anthropic", "ollama"
llm_provider = "lmstudio"

# Full LLM API endpoint URL
# Must start with http:// or https://
# Default: LM Studio local server
# Examples:
#   - "http://localhost:1234/v1" (LM Studio - default)
#   - "https://api.openai.com/v1" (OpenAI)
#   - "https://api.anthropic.com" (Anthropic)
#   - "http://localhost:11434/v1" (Ollama)
llm_endpoint = "http://localhost:1234/v1"

# API key for the LLM provider
# Can be empty string for local models that don't require authentication
# LM Studio does not require an API key
llm_api_key = ""

# Model identifier
# For LM Studio: Use the model name as shown in LM Studio UI
# Examples:
#   - "local-model" (LM Studio - use whatever model you have loaded)
#   - "gpt-4" (OpenAI)
#   - "gpt-3.5-turbo" (OpenAI)
#   - "claude-3-opus-20240229" (Anthropic)
#   - "llama3:7b" (Ollama)
llm_model = "local-model"

[proxy]
# Address and port for the local proxy server
# Format: "host:port"
# The proxy will listen on this address for OpenAI-compatible requests
# Default: Port 4321
# Endpoint: http://{listen_address}/v1/chat/completions
listen_address = "127.0.0.1:4321"
```

---

## Quick Start with New Defaults

### 1. Setup LM Studio
```bash
# Download and install LM Studio from https://lmstudio.ai
# Load a model (e.g., Llama 3 7B, Mistral 7B, etc.)
# Start local server (port 1234)
```

### 2. Run TSLP
```bash
# Build TSLP
go build -o tslp ./cmd/tslp

# Create runtime directory
mkdir -p runtime

# Start TSLP (uses default config)
./tslp
```

### 3. Test the Connection
```bash
# Simple test
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-model",
    "messages": [
      {"role": "user", "content": "Say hello"}
    ],
    "stream": false
  }'
```

---

## Switching to Other Providers

### OpenAI
Edit `config/config.toml`:
```toml
[llm]
llm_provider = "openai"
llm_endpoint = "https://api.openai.com/v1"
llm_api_key = "sk-your-api-key-here"
llm_model = "gpt-4"
```

### Ollama
Edit `config/config.toml`:
```toml
[llm]
llm_provider = "ollama"
llm_endpoint = "http://localhost:11434/v1"
llm_api_key = ""
llm_model = "llama3:7b"
```

### Anthropic
Edit `config/config.toml`:
```toml
[llm]
llm_provider = "anthropic"
llm_endpoint = "https://api.anthropic.com"
llm_api_key = "sk-ant-your-api-key-here"
llm_model = "claude-3-opus-20240229"
```

---

## Troubleshooting

### "Connection refused" to port 1234
- LM Studio is not running
- Start LM Studio and load a model
- Enable "Local Server" in LM Studio

### "Connection refused" to port 4321
- TSLP is not running
- Start TSLP: `./tslp`
- Check logs: `tail -f runtime/tslp.log`

### Wrong model name
- Check LM Studio UI for exact model name
- Update `llm_model` in config to match
- LM Studio shows model name in server tab

---

## Summary

✅ **Proxy Port:** Now 4321 (was 8080)
✅ **LLM Provider:** Now LM Studio (was OpenAI)
✅ **LLM Endpoint:** Now http://localhost:1234/v1 (was https://api.openai.com/v1)
✅ **API Key:** Empty (local models don't need keys)
✅ **Model:** "local-model" (use your loaded model name)

**Result:** TSLP works out-of-the-box with LM Studio for local, privacy-preserving agentic coding with small models.
