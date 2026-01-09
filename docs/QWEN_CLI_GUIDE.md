# tinyMem with Qwen CLI Integration Guide

## 🎯 Overview

This guide explains how to integrate tinyMem with the Qwen CLI tool. tinyMem provides state management and context hydration capabilities that work seamlessly with the Qwen CLI, allowing you to maintain conversation history and code state while leveraging the power of Qwen's AI models.

**Key Benefits:**
- ✅ Deterministic state management across Qwen CLI sessions
- ✅ Automatic context hydration with relevant code snippets
- ✅ Reduced token usage by sending only relevant context
- ✅ Session continuity with `--continue` and `--resume` options

---

## 🚀 Quick Start

### 1. Verify Qwen CLI Installation

First, ensure you have the Qwen CLI installed (which you already have via Homebrew):

```bash
# Check installation
qwen --version
# Should output: 0.6.0 or similar

# Test basic functionality
qwen --prompt "Say hello"
```

### 2. Configure tinyMem for Qwen CLI

Create a configuration file that tells tinyMem to use the Qwen CLI as the LLM provider:

```bash
# Create a new config file for Qwen CLI
cp config/config.example.toml config/config.qwen-cli.toml
```

Edit the configuration file to use the CLI endpoint:

```toml
[llm]
llm_provider = "qwen"
llm_endpoint = "cli"
llm_api_key = ""
llm_model = "qwen-max"  # or whatever model you prefer

[server]
port = 4321
host = "127.0.0.1"

[context]
recentContextPairs = 5
maxContextTokens = 8000
```

### 3. Start tinyMem with Qwen CLI

```bash
# Start tinyMem with the Qwen CLI configuration
./tinyMem --config config/config.qwen-cli.toml
```

You should see:
```
Using CLI provider: qwen
tinyMem Ready
Endpoint: http://127.0.0.1:4321/v1/chat/completions
```

### 4. Use with Your Favorite Tools

Now you can use tinyMem's HTTP endpoint with any tool that supports OpenAI-compatible APIs:

```bash
# Use with curl
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "Write a Go function to add two integers"}
    ]
  }'

# Use with VS Code, JetBrains IDEs, or any other tool
# that supports OpenAI-compatible endpoints
```

---

## 🔧 Configuration Details

### Qwen CLI Provider Configuration

tinyMem recognizes the Qwen CLI through the `cli:qwen` provider format:

```toml
[llm]
llm_provider = "cli:qwen"
llm_endpoint = "cli"
llm_model = "qwen-max"
```

This tells tinyMem to:
1. Use the CLI adapter
2. Execute the `qwen` command
3. Pass the conversation context to the CLI tool
4. Process the response back through tinyMem's state management

### Session Management Integration

tinyMem's session management works perfectly with Qwen CLI's session features:

```bash
# Start tinyMem with Qwen CLI
./tinyMem --config config/config.qwen-cli.toml

# In another terminal, use the tinyMem endpoint
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "Create a simple calculator class in Python"}],
    "session_id": "calculator-project"
  }'

# Later, continue the same session
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type" \
  -d '{
    "messages": [{"role": "user", "content": "Add division method to the calculator"}],
    "session_id": "calculator-project"
  }'
```

### Context Hydration with Qwen CLI

tinyMem's context hydration works seamlessly with Qwen CLI:

```
## Context Provided to Qwen CLI
[CURRENT STATE: AUTHORITATIVE]
Entity: /calculator.py::Calculator
Source: Confirmed via AST

class Calculator:
    def __init__(self):
        pass
    
    def add(self, a, b):
        return a + b

    def subtract(self, a, b):
        return a - b
[END CURRENT STATE]

[RECENT CONTEXT]
Pair 1:
USER: Create a simple calculator class in Python
ASSISTANT: Here's a basic calculator class...
[END RECENT CONTEXT]

Can you add a division method to the calculator?
```

**Key Benefit:** Only relevant code + recent context, not the entire history!

---

## 💰 Token Savings Example

### Without tinyMem (Direct Qwen CLI usage)
```bash
# Request 1
qwen "Write a function to add numbers"
# Tokens: 20 (prompt) + 100 (response) = 120

# Request 2 - Must provide full context manually
qwen "Here's what I have: [paste 100 tokens of code]
Now modify it to handle overflow"
# Tokens: 150 (prompt with full context) + 100 (response) = 250

# Request 3 - Context keeps growing
qwen "Here's what I have: [paste 200 tokens of code]
Now add error handling"
# Tokens: 250 (prompt with full context) + 100 (response) = 350

Total: 720 tokens
```

### With tinyMem + Qwen CLI (Managed context)
```bash
# Start tinyMem with Qwen CLI
./tinyMem --config config/config.qwen-cli.toml

# Request 1 - tinyMem stores the function
curl -X POST http://localhost:4321/v1/chat/completions \
  -d '{"messages":[{"role":"user","content":"Write a function to add numbers"}]}'
# Tokens: 20 (prompt) + 100 (response) = 120
# State Map: Add function stored (AUTHORITATIVE)

# Request 2 - tinyMem provides only relevant code
curl -X POST http://localhost:4321/v1/chat/completions \
  -d '{"messages":[{"role":"user","content":"Modify Add to handle overflow"}]}'
# Tokens: 50 (prompt + hydrated Add function) + 100 (response) = 150
# State Map: Add function updated

# Request 3 - Still only relevant code
curl -X POST http://localhost:4321/v1/chat/completions \
  -d '{"messages":[{"role":"user","content":"Add error handling"}]}'
# Tokens: 50 (prompt + hydrated Add function) + 100 (response) = 150

Total: 420 tokens (42% savings!)
```

---

## 🎯 Use Cases

### 1. **Enhanced Development Workflow**
- Use tinyMem's state management with Qwen CLI's advanced features
- Leverage Qwen CLI's approval modes (`--approval-mode`) with tinyMem's entity tracking
- Combine Qwen CLI's extensions with tinyMem's context hydration

### 2. **Reduced API Costs**
- Only send relevant context to Qwen models
- Avoid repeating full conversation history
- Maximize efficiency of Qwen's token usage

### 3. **Integration with Existing Tools**
- Use tinyMem's HTTP endpoint with IDE plugins
- Maintain Qwen CLI's rich features while getting state management
- Example: Use Qwen CLI with tinyMem in VS Code terminal

### 4. **Advanced Features Combination**
- Use Qwen CLI's `--sandbox` mode with tinyMem's state tracking
- Combine Qwen CLI's `--experimental-skills` with tinyMem's entity resolution
- Leverage Qwen CLI's web search (`--web-search-default`) with tinyMem's context

---

## 🔍 Advanced Configuration

### Custom Qwen CLI Options

You can pass specific Qwen CLI options through tinyMem by configuring environment variables:

```bash
# Set Qwen CLI specific options as environment variables
export QWEN_MODEL="qwen-plus"
export QWEN_APPROVAL_MODE="auto-edit"

# Start tinyMem
./tinyMem --config config/config.qwen-cli.toml
```

### Multiple Model Configuration

You can configure different Qwen models in your config:

```toml
[llm]
llm_provider = "cli:qwen"
llm_endpoint = "cli"
llm_model = "qwen-max"  # Use qwen-max for complex tasks

# Or use qwen-plus for balanced performance
# llm_model = "qwen-plus"

# Or use qwen-turbo for faster responses
# llm_model = "qwen-turbo"
```

### Qwen CLI Extensions Integration

If you're using Qwen CLI extensions, you can configure them in your setup:

```bash
# First, install extensions with Qwen CLI directly
qwen extensions install https://github.com/example/qwen-git-extension

# Then configure tinyMem to use them
# This would typically be handled by setting the appropriate environment
# variables that Qwen CLI recognizes
```

---

## 📊 Comparison: Direct Qwen CLI vs tinyMem + Qwen CLI

| Feature | Direct Qwen CLI | tinyMem + Qwen CLI |
|---------|----------------|-------------------|
| **State Management** | ❌ Manual context | ✅ Automatic state tracking |
| **Context Hydration** | ❌ Manual pasting | ✅ Automatic code injection |
| **Session Continuity** | ❌ Limited | ✅ Full session support |
| **Token Efficiency** | ❌ Full context sent | ✅ Relevant context only |
| **IDE Integration** | ❌ Terminal only | ✅ HTTP endpoint for tools |
| **Entity Tracking** | ❌ None | ✅ Full AST-based tracking |
| **Use Case** | Quick tasks | Complex projects |

**Recommendation:**
- **Direct Qwen CLI**: For quick, standalone tasks
- **tinyMem + Qwen CLI**: For ongoing projects requiring state management

---

## 🐛 Troubleshooting

### Qwen CLI not found
```
Error: exec: "qwen": executable file not found in $PATH
```

**Solution:**
```bash
# Check if qwen is in PATH
which qwen

# Verify installation
qwen --version

# If not found, reinstall
brew install qwen
# or
npm install -g @qwen/qwen-cli
```

### Configuration errors
```
Error: Unknown provider: qwen
```

**Solution:**
```toml
# Make sure to use the correct provider format
[llm]
llm_provider = "cli:qwen"  # Note the cli: prefix
llm_endpoint = "cli"
```

### Context not hydrating
```
Qwen CLI receives full context instead of relevant snippets
```

**Solution:**
- Verify that tinyMem is running with the correct configuration
- Check that the LLM provider is set to `"cli:qwen"` and not `"http"`
- Ensure the state map is properly populated with entities

### Session management issues
```
Sessions not persisting between calls
```

**Solution:**
```bash
# Use the --chat-recording flag with Qwen CLI if needed
# This is handled automatically by tinyMem, but you can verify:
export QWEN_CHAT_RECORDING=true
./tinyMem --config config/config.qwen-cli.toml
```

---

## 📚 Example Workflow

### Scenario: Building a Python Application with tinyMem + Qwen CLI

```bash
# 1. Configure tinyMem for Qwen CLI
cat > config/config.qwen-python.toml << EOF
[llm]
llm_provider = "cli:qwen"
llm_endpoint = "cli"
llm_model = "qwen-max"

[server]
port = 4321
host = "127.0.0.1"

[context]
recentContextPairs = 3
maxContextTokens = 12000
EOF

# 2. Start tinyMem with Qwen CLI
./tinyMem --config config/config.qwen-python.toml

# 3. Request: Create a basic calculator
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Write a Python class for a basic calculator with add, subtract, multiply, divide methods"
    }]
  }'

# Result: Calculator class committed to State Map

# 4. Request: Add scientific functions (tinyMem hydrates Calculator class)
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{
      "role": "user",
      "content": "Add power and square root methods to the calculator"
    }]
  }'

# Result: Only sends:
#  [CURRENT STATE] Calculator class (with 4 methods)
#  + "Add power and square root methods"
# Instead of full conversation history!

# 5. Check state and continue development
curl http://localhost:4321/state | jq
```

**Tokens saved:** ~40-60% compared to manual context management

---

## 🚀 Next Steps

1. **Start Small**: Begin with simple projects to understand the workflow
2. **Configure Properly**: Set up your config file with appropriate context sizes
3. **Monitor Usage**: Watch how tinyMem manages your code entities
4. **Scale Up**: Apply to larger projects where state management becomes critical

For more information:
- Main README: [README.md](../README.md)
- General CLI Integration: [CLI_GUIDE.md](CLI_GUIDE.md)
- Configuration: [config/config.toml](../config/config.toml)
- Specification: [specification.md](specification.md)

---

**tinyMem + Qwen CLI Integration** — Combining Qwen's powerful AI with deterministic state management! 🎯