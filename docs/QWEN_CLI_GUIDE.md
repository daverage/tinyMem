# Setting Up tinyMem with Qwen Code CLI

 

This guide shows you how to integrate tinyMem with **Qwen Code CLI** - a separate command-line tool for Qwen models that provides stateful memory and intelligent code retrieval.

 

---

 

## 📋 **What is Qwen Code CLI?**

 

Qwen Code CLI is an **official CLI tool** from QwenLM that lets you interact with Qwen Code models from your terminal. It's similar to Claude CLI or GitHub Copilot CLI.

 

**GitHub:** https://github.com/QwenLM/qwen-code

 

---

 

## 🔧 **Step 1: Install Qwen Code CLI**

 

### **Option A: Install via npm (Recommended)**

 

```bash

# Install globally

npm install -g @qwen-code/qwen-code@latest

 

# Verify installation

qwen --version

```

 

### **Option B: Install via Homebrew (macOS/Linux)**

 

```bash

brew install qwen-code

 

# Verify installation

qwen --version

```

 

### **Prerequisites:**

- **Node.js 20+** (required for npm installation)

 

---

 

## 📦 **Step 2: Build tinyMem**

 

```bash

# Clone the repository (if not already done)

git clone https://github.com/yourusername/tinyMem.git

cd tinyMem

 

# Build tinyMem

go build -o tinyMem ./cmd/tinyMem

 

# Verify build

./tinyMem --version

```

 

---

 

## ⚙️ **Step 3: Configure tinyMem for Qwen Code**

 

### **Create Qwen-specific config:**

 

```bash

# Create config directory if it doesn't exist

mkdir -p config

 

# Create config file

cat > config/config.qwen-code.toml << 'EOF'

# tinyMem Configuration for Qwen Code CLI

 

[database]

database_path = "./runtime/tinyMem.db"

 

[logging]

log_path = "./runtime/tinyMem.log"

debug = true

 

[llm]

# Use "qwen-code" to indicate Qwen Code CLI provider

llm_provider = "qwen-code"

llm_endpoint = "cli"

llm_api_key = ""

llm_model = "qwen-code"

 

[proxy]

listen_address = "127.0.0.1:4321"

 

[hydration]

# Budget settings (tuned for Qwen Code models)

max_tokens = 16000      # Conservative limit for smaller models

max_entities = 30       # Limit entities to prevent context overflow

 

# Structural anchors (always enabled for deterministic retrieval)

enable_file_mention_anchors = true

enable_symbol_mention_anchors = true

enable_previous_hydration_anchors = true

 

# Semantic ranking (optional - start disabled)

enable_semantic_ranking = false

semantic_threshold = 0.7

semantic_budget_tokens = 4000

semantic_budget_entities = 5

 

# Embedding provider

embedding_provider = "simple"

embedding_model = "simple-384"

embedding_cache_ttl = 86400

EOF

```

 

### **Key Configuration Notes:**

 

- **`llm_provider = "qwen-code"`** - Tells tinyMem to use Qwen Code CLI

- **`llm_endpoint = "cli"`** - Indicates CLI mode (not HTTP)

- **`max_tokens = 16000`** - Conservative budget to prevent context overflow

- **`max_entities = 30`** - Limits number of code entities hydrated

 

---

 

## 🚀 **Step 4: Start tinyMem**

 

```bash

# Create runtime directory

mkdir -p runtime

 

# Start tinyMem with Qwen Code config

./tinyMem --config config/config.qwen-code.toml

```

 

**Expected output:**

```

[INFO] Starting tinyMem proxy server on 127.0.0.1:4321

[INFO] Using CLI provider: qwen-code

[INFO] CLI command: qwen -p [prompt]

[INFO] Database initialized at ./runtime/tinyMem.db

[INFO] Hydration budget: 16000 tokens, 30 entities

[INFO] Server ready for requests

```

 

---

 

## ✅ **Step 5: Test the Setup**

 

### **Test 1: Health Check**

 

```bash

curl http://localhost:4321/health

```

 

**Expected:**

```json

{"status": "ok", "timestamp": 1704931200}

```

 

### **Test 2: Doctor Check**

 

```bash

curl http://localhost:4321/doctor | jq

```

 

**Expected:**

```json

{

  "database": {

    "connected": true,

    "vault_count": 0,

    "state_count": 0,

    "ledger_count": 0

  },

  "llm": {

    "provider": "qwen-code",

    "endpoint": "cli",

    "model": "qwen-code"

  },

  "proxy": {

    "listen_address": "127.0.0.1:4321",

    "uptime_seconds": 10

  }

}

```

 

### **Test 3: Simple Chat Request**

 

```bash

curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen-code",
    "messages": [
      {"role": "user", "content": "Write a Python function that adds two numbers."}
    ],
    "stream": false
  }'

```

 

**Expected:** JSON response with Qwen's generated code.

 

---

 

## 🎯 **Step 6: Configure Your Code Editor**

### **Option A: VSCode with Continue.dev**

 

Add to `.continue/config.json`:

 

```json

{

  "models": [

    {

      "title": "Qwen Code via tinyMem",

      "provider": "openai",

      "model": "qwen-code",

      "apiBase": "http://localhost:4321/v1",

      "apiKey": "not-needed"

    }

  ]

}

```

 

### **Option B: Cursor**

 

Settings → Models → Add Custom Model:

- **Base URL:** `http://localhost:4321/v1`

- **Model Name:** `qwen-code`

- **API Key:** Leave empty

 

### **Option C: Direct curl (for testing)**

 

```bash

curl -X POST http://localhost:4321/v1/chat/completions \

  -H "Content-Type: application/json" \

  -d '{

    "model": "qwen-code",

    "messages": [

      {"role": "user", "content": "Create a REST API endpoint for user login"}

    ]
    
  }'

```

 

---

 

## 📊 **How tinyMem Works with Qwen Code**

 

### **Without tinyMem:**

 

```

You: Write a Flask app

Qwen: [generates app.py]

 

You: Add authentication

Qwen: [doesn't know about previous app.py, starts fresh]

```

 

### **With tinyMem:**

 

```

You: Write a Flask app

tinyMem → Stores app.py entities in state map

Qwen: [generates app.py]

 

You: Add authentication

tinyMem → Hydrates app.py entities

tinyMem → Sends: [CURRENT STATE: app.py] + your prompt

Qwen: [generates auth.py knowing about existing app structure]

```

 

---

 

## 🔍 **Monitor tinyMem State**

 

### **Check State Map:**

 

```bash

curl http://localhost:4321/state | jq

```

 

**Example output:**

```json

{

  "authoritative_count": 3,

  "entities": [

    {

      "entity_key": "/app.py::create_app",

      "filepath": "/app.py",

      "symbol": "create_app",

      "state": "AUTHORITATIVE",

      "confidence": "CONFIRMED",

      "artifact_hash": "7f8a9b...",

      "stale": false

    }

  ]

}

```

 

### **Check Recent Conversations:**

 

```bash

curl http://localhost:4321/recent | jq

```

 

### **Introspect Hydration Decisions:**

 

```bash

# Get latest episode ID

EPISODE_ID=$(curl -s http://localhost:4321/recent | jq -r '.episodes[0].episode_id')

 

# See what was hydrated

curl "http://localhost:4321/introspect/hydration?episode_id=$EPISODE_ID" | jq

```

 

**Shows:**

- Which code entities were included

- Why (file mention, symbol mention, previous hydration)

- Token counts and budget usage

 

---

 

## 🔧 **Advanced Configuration**

 

### **Increase Budget for Larger Projects:**

 

```toml

[hydration]

max_tokens = 32000      # More context for complex tasks

max_entities = 50       # More code entities

```

 

### **Enable Semantic Ranking:**

 

```toml

[hydration]

enable_semantic_ranking = true

semantic_threshold = 0.7

semantic_budget_tokens = 8000

semantic_budget_entities = 10

```

 

**Benefits:**

- Finds related code even without explicit mentions

- Example: "fix auth bug" → automatically includes `ValidateToken`, `CheckPermissions`, etc.

 

### **Use Different Qwen Model:**

 

If Qwen Code CLI supports model selection (check docs):

```toml

[llm]

llm_model = "qwen2.5-coder-7b"  # Adjust based on available models

```

 

---

 

## 🐛 **Troubleshooting**

 

### **Problem 1: "qwen: command not found"**

 

**Solution:**

```bash

# Check if qwen is in PATH

which qwen

 

# If not found, reinstall:

npm install -g @qwen-code/qwen-code@latest

 

# Verify:

qwen --version

```

 

### **Problem 2: Qwen Code requires authentication**

 

Qwen Code CLI may require authentication. Check the official docs:

 

```bash

# Authenticate Qwen Code

qwen --auth

 

# Or set API key if needed

export QWEN_API_KEY="your-key"

```

 

### **Problem 3: tinyMem can't execute qwen**

 

**Check logs:**

```bash

tail -f runtime/tinyMem.log

```

 

**Look for:**

```

[INFO] Using CLI provider: qwen-code

[ERROR] Failed to execute command: qwen -p [prompt]

```

 

**Solutions:**

- Verify `qwen` is executable: `qwen --version`

- Check PATH: `echo $PATH`

- Try absolute path in config (custom provider):

  ```toml

  [llm]

  llm_provider = "cli:/usr/local/bin/qwen"

  ```

 

### **Problem 4: Context overflow errors**

 

**Error:** `"maximum context length exceeded"`

 

**Solution:** Reduce hydration budget:

```toml

[hydration]

max_tokens = 8000       # Lower limit

max_entities = 15       # Fewer entities

```

 

### **Problem 5: Slow response times**

 

**Causes:**

- Qwen Code CLI loads model on every request

- Large hydration context

 

**Solutions:**

 

**A) Reduce budget:**

```toml

[hydration]

max_tokens = 8000

max_entities = 20

```

 

**B) Check Qwen Code CLI performance:**

```bash

# Test direct performance

time qwen -p "Hello"

```

 

If slow, consider using Qwen via Ollama instead (faster):

```bash

ollama pull qwen:7b

 

# Update config:

[llm]

llm_provider = "ollama"

llm_endpoint = "http://localhost:11434/v1"

llm_model = "qwen:7b"

```

 

---

 

## 📊 **Token Usage Example**

 

### **Without tinyMem:**

```

Request 1: "Write Flask app" → 200 tokens

Request 2: "Add auth" + manual paste of app.py → 500 tokens

Request 3: "Fix bug" + manual paste of all code → 800 tokens

Total: 1500 tokens

```

 

### **With tinyMem:**

```

Request 1: "Write Flask app" → 200 tokens

Request 2: "Add auth" + auto-hydrated app.py → 350 tokens

Request 3: "Fix bug" + auto-hydrated relevant entities → 400 tokens

Total: 950 tokens (37% savings)

```

 

---

 

## 🎓 **Usage Examples**

 

### **Example 1: Build a Multi-File Project**

 

```bash

# Request 1: Create initial structure

curl -X POST http://localhost:4321/v1/chat/completions \

  -H "Content-Type: application/json" \

  -d '{

    "model": "qwen-code",

    "messages": [{"role": "user", "content": "Create a Python REST API with Flask"}]

  }'

 

# tinyMem stores: /app.py entities

 

# Request 2: Add authentication

curl -X POST http://localhost:4321/v1/chat/completions \

  -H "Content-Type: application/json" \

  -d '{

    "model": "qwen-code",

    "messages": [{"role": "user", "content": "Add JWT authentication"}]

  }'

 

# tinyMem hydrates app.py, Qwen generates auth.py with context

 

# Request 3: Add database

curl -X POST http://localhost:4321/v1/chat/completions \

  -H "Content-Type: application/json" \

  -d '{

    "model": "qwen-code",

    "messages": [{"role": "user", "content": "Add PostgreSQL database"}]

  }'

 

# tinyMem hydrates app.py + auth.py, Qwen generates db.py

```

 

### **Example 2: Check What's in Memory**

 

```bash

# See all tracked code entities

curl http://localhost:4321/state | jq '.entities[] | {symbol, filepath}'

```

 

Output:

```json

{"symbol": "create_app", "filepath": "/app.py"}

{"symbol": "run_server", "filepath": "/app.py"}

{"symbol": "validate_token", "filepath": "/auth.py"}

{"symbol": "init_db", "filepath": "/db.py"}

```

 

---

 

## 🔄 **Alternative: Use Ollama for Better Performance**

 

If Qwen Code CLI is slow, use Ollama instead:

 

### **Install Ollama:**

```bash

curl -fsSL https://ollama.com/install.sh | sh

ollama pull qwen:7b

```

 

### **Update tinyMem config:**

```toml

[llm]

llm_provider = "ollama"

llm_endpoint = "http://localhost:11434/v1"

llm_api_key = ""

llm_model = "qwen:7b"

```

 

**Benefits:**

- Model stays loaded in memory (fast subsequent requests)

- Optimized inference

- GPU acceleration

 

---

 

## 📚 **Next Steps**

 

1. **Enable Semantic Ranking:**

   ```toml

   enable_semantic_ranking = true

   ```

 

2. **Read Documentation:**

   - `RETRIEVAL_INVARIANTS.md` - System guarantees and failure modes

   - `HYBRID_RETRIEVAL_DESIGN.md` - Retrieval architecture

   - `README.md` - Full feature documentation

 

3. **Experiment with Budget:**

   - Tune `max_tokens` based on your projects

   - Monitor with `/introspect/hydration` endpoint

 

4. **Check Qwen Code CLI docs:**

   - https://github.com/QwenLM/qwen-code

 

---

 

## ✨ **What You Get with tinyMem + Qwen Code**

 

- ✅ **Stateful Memory** - Qwen remembers all your code across sessions

- ✅ **Token Budget** - Never exceed context limits

- ✅ **Structural Anchors** - Deterministic code retrieval

- ✅ **AST Verification** - Code is validated before storage

- ✅ **ETV (External Truth)** - Detects manual file edits

- ✅ **Introspection** - See why code was included

- ✅ **Budget Control** - Prevent context overflow

 

**Enjoy stateful, intelligent coding with Qwen! 🚀**
