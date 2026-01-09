# tinyMem CLI Integration Guide

## 🎯 Overview

tinyMem now supports CLI-based LLM tools in addition to HTTP endpoints! This allows you to:

- ✅ Use Claude CLI, Gemini CLI, Shell GPT, and other CLI tools
- ✅ Reduce token usage by providing only relevant context
- ✅ Maintain deterministic state across CLI invocations
- ✅ Get all the benefits of tinyMem's state management with any CLI tool

---

## 🚀 Quick Start

### 1. Install a CLI Tool

Choose one of these CLI tools:

#### **Claude CLI** (Recommended)
```bash
# Install Claude CLI
pip install claude-cli
# or follow: https://docs.anthropic.com/en/docs/claude-cli

# Set API key
export ANTHROPIC_API_KEY="your-key-here"
```

#### **Gemini CLI**
```bash
# Install Gemini CLI
npm install -g @google/generative-ai-cli

# Set API key
export GEMINI_API_KEY="your-key-here"
```

#### **Shell GPT (sgpt)**
```bash
# Install Shell GPT
pip install shell-gpt

# Set API key
export OPENAI_API_KEY="your-key-here"
```

#### **AIChat**
```bash
# Install AIChat
cargo install aichat
# or: brew install aichat

# Configure with: aichat --configure
```

### 2. Configure tinyMem

Use one of the example configs:

```bash
# Copy example config
cp config/config.claude-cli.toml config/config.toml

# Edit if needed
nano config/config.toml
```

### 3. Start tinyMem

```bash
./tinyMem --config config/config.toml
```

You should see:
```
Using CLI provider: claude
tinyMem Ready
Endpoint: http://127.0.0.1:4321/v1/chat/completions
```

---

## 📋 Configuration Examples

### Claude CLI
```toml
[llm]
llm_provider = "claude"
llm_endpoint = "cli"
llm_api_key = ""
llm_model = "claude-3-5-sonnet-20241022"
```

### Gemini CLI
```toml
[llm]
llm_provider = "gemini"
llm_endpoint = "cli"
llm_api_key = ""
llm_model = "gemini-1.5-pro"
```

### Shell GPT (sgpt)
```toml
[llm]
llm_provider = "sgpt"
llm_endpoint = "cli"
llm_api_key = ""
llm_model = "gpt-4"
```

### Custom CLI Tool
```toml
[llm]
llm_provider = "cli:mycustomtool"
llm_endpoint = "cli"
llm_api_key = ""
llm_model = "model-name"
```

---

## 🔧 How It Works

### Architecture

```
User Request
    ↓
tinyMem Proxy (localhost:4321)
    ↓
[Hydration] Inject only relevant code from State Map
    ↓
CLI Adapter
    ↓
Shell Command (e.g., `claude`, `gemini`, `sgpt`)
    ↓
LLM Response
    ↓
[Processing] Parse, resolve entities, evaluate gates
    ↓
[State Map] Commit code if gates pass
```

### What Gets Sent to CLI

Instead of sending the full conversation history, tinyMem sends:

```
## Context
[CURRENT STATE: AUTHORITATIVE]
Entity: /file.go::Add
Source: Confirmed via AST

func Add(a, b int) int {
    return a + b
}
[END CURRENT STATE]

[RECENT CONTEXT]
Pair 1:
USER: Write an addition function
ASSISTANT: I'll create a simple addition function...
[END RECENT CONTEXT]

Can you modify Add to handle overflow?
```

**Key Benefit:** Only relevant code + recent context, not the entire history!

---

## 💰 Token Savings Example

### Without tinyMem (Traditional CLI usage)
```bash
# Request 1
claude "Write a function to add numbers"
# Tokens: 20 (prompt) + 100 (response) = 120

# Request 2 - Must provide full context manually
claude "Here's what I have: [paste 100 tokens of code]
Now modify it to handle overflow"
# Tokens: 150 (prompt with full context) + 100 (response) = 250

# Request 3 - Context keeps growing
claude "Here's what I have: [paste 200 tokens of code]
Now add error handling"
# Tokens: 250 (prompt with full context) + 100 (response) = 350

Total: 720 tokens
```

### With tinyMem (Managed context)
```bash
# Request 1
curl -X POST http://localhost:4321/v1/chat/completions \
  -d '{"messages":[{"role":"user","content":"Write a function to add numbers"}]}'
# Tokens: 20 (prompt) + 100 (response) = 120
# State Map: func Add stored (AUTHORITATIVE)

# Request 2 - tinyMem provides only relevant code
curl -X POST http://localhost:4321/v1/chat/completions \
  -d '{"messages":[{"role":"user","content":"Modify Add to handle overflow"}]}'
# Tokens: 50 (prompt + hydrated Add function) + 100 (response) = 150
# State Map: func Add updated

# Request 3 - Still only relevant code
curl -X POST http://localhost:4321/v1/chat/completions \
  -d '{"messages":[{"role":"user","content":"Add error handling"}]}'
# Tokens: 50 (prompt + hydrated Add function) + 100 (response) = 150

Total: 420 tokens (42% savings!)
```

---

## 🎯 Use Cases

### 1. **Reduce API Costs**
- Only send relevant context to expensive models like Claude Opus or GPT-4
- Avoid repeating full conversation history

### 2. **Use Free Tier Efficiently**
- Maximize free tier limits by minimizing tokens per request
- Anthropic free tier: 1,000 requests/day
- With tinyMem: More requests fit in the limit

### 3. **Integrate with Existing Workflows**
- Keep using your favorite CLI tool
- Add state management without changing your workflow
- Example: Use Claude CLI with tinyMem in VS Code terminal

### 4. **Local + Cloud Hybrid**
- Use tinyMem for state management (local)
- Use Claude/Gemini CLI for LLM calls (cloud)
- Best of both worlds

---

## 🔍 Advanced Usage

### Custom CLI Tool

If you have a custom CLI tool, you can integrate it:

```toml
[llm]
llm_provider = "cli:your-custom-tool"
llm_endpoint = "cli"
llm_model = "model-name"
```

The tool must:
1. Accept input via stdin OR as the last argument
2. Output the response to stdout
3. Be in your PATH

Example custom tool:
```bash
#!/bin/bash
# /usr/local/bin/my-llm-tool
# Simple wrapper around curl

INPUT=$(cat)  # Read from stdin

curl -s https://api.example.com/chat \
  -H "Authorization: Bearer $API_KEY" \
  -d "{\"prompt\": \"$INPUT\"}" | \
  jq -r '.response'
```

### Debugging CLI Commands

Enable debug mode to see exact commands executed:

```toml
[logging]
debug = true
```

Then check logs:
```bash
tail -f ./runtime/tinyMem.log
```

### Environment Variables

CLI tools often use environment variables for API keys:

```bash
# Claude
export ANTHROPIC_API_KEY="sk-ant-..."

# Gemini
export GEMINI_API_KEY="..."

# OpenAI (for sgpt)
export OPENAI_API_KEY="sk-..."

# Start tinyMem
./tinyMem
```

---

## 📊 Comparison: HTTP vs CLI

| Feature | HTTP (LM Studio) | CLI (Claude/Gemini) |
|---------|-----------------|---------------------|
| **Setup** | Run local server | Install CLI tool |
| **Speed** | Very fast (local) | Moderate (network) |
| **Cost** | Free (local model) | Pay-per-token (cloud) |
| **Quality** | Depends on model | High (Claude Opus, GPT-4) |
| **Context Management** | ✅ tinyMem | ✅ tinyMem |
| **Token Savings** | N/A (local) | ✅ Up to 50% |
| **Use Case** | Fast iteration | Production quality |

**Recommendation:**
- **Development**: Use HTTP with LM Studio (fast, free)
- **Production**: Use CLI with Claude/GPT-4 (high quality)
- **Best**: Use both! Test locally, deploy with CLI

---

## 🐛 Troubleshooting

### CLI command not found
```
Error: exec: "claude": executable file not found in $PATH
```

**Solution:**
```bash
# Check if tool is installed
which claude

# Install if missing
pip install claude-cli

# Verify PATH
echo $PATH
```

### API key errors
```
Error: ANTHROPIC_API_KEY not set
```

**Solution:**
```bash
# Set environment variable
export ANTHROPIC_API_KEY="your-key-here"

# Or add to ~/.bashrc or ~/.zshrc
echo 'export ANTHROPIC_API_KEY="your-key-here"' >> ~/.bashrc
source ~/.bashrc
```

### Slow responses
```
CLI calls taking > 5 seconds
```

**Possible causes:**
- Network latency to API
- Large context being sent
- API rate limiting

**Solutions:**
- Check internet connection
- Reduce `recentContextPairs` in server.go
- Use a faster model (e.g., Claude Haiku vs Opus)

### Token limits exceeded
```
Error: prompt too long
```

**Solution:**
- Reduce context: Edit `server.go:43-44`
- Use smaller model with larger context window
- Clean up State Map: `curl -X POST http://localhost:4321/debug/reset` (warning: deletes all state!)

---

## 📚 Example Workflow

### Scenario: Building a REST API with Claude CLI

```bash
# 1. Start tinyMem with Claude CLI
./tinyMem --config config/config.claude-cli.toml

# 2. Request: Create a basic handler
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Write a Go HTTP handler for GET /users"
    }]
  }'

# Result: UsersHandler function committed to State Map

# 3. Request: Add POST endpoint (tinyMem hydrates UsersHandler)
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Add a POST /users endpoint"
    }]
  }'

# Result: Only sends:
#  [CURRENT STATE] UsersHandler (GET)
#  + "Add a POST /users endpoint"
# Instead of full conversation history!

# 4. Check state
curl http://localhost:4321/state | jq
```

**Tokens saved:** ~40-60% compared to manual context management

---

## 🚀 Next Steps

1. **Try it out**: Start with Claude CLI config
2. **Monitor usage**: Check logs and API billing
3. **Optimize**: Adjust context settings for your use case
4. **Scale**: Use CLI mode for production deployments

For more information:
- Main README: [README.md](README.md)
- Configuration: [config/config.toml](config/config.toml)
- Specification: [specification.md](specification.md)

---

**tinyMem CLI Integration** — Making cloud LLMs efficient with deterministic state management! 🎯
