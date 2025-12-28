# TSLP Quick Start Guide

Get TSLP running with LM Studio in under 5 minutes.

---

## Prerequisites

- **Go 1.22+** installed
- **LM Studio** installed ([download here](https://lmstudio.ai))
- **Terminal/Command Prompt**

---

## Step 1: Setup LM Studio (2 minutes)

1. **Download & Install LM Studio**
   - Visit https://lmstudio.ai
   - Download for your OS (macOS/Windows/Linux)
   - Install and launch

2. **Load a Model**
   - Click "Search" tab in LM Studio
   - Search for a small model (recommended: 7B parameters)
   - Popular choices:
     - `TheBloke/Mistral-7B-Instruct-v0.2-GGUF`
     - `TheBloke/Llama-2-7B-Chat-GGUF`
     - `lmstudio-community/Meta-Llama-3-8B-Instruct-GGUF`
   - Click download (choose Q4_K_M quantization for balance)

3. **Start Local Server**
   - Click "Local Server" tab (←→ icon)
   - Select your downloaded model
   - Click "Start Server"
   - Server should start on `http://localhost:1234`
   - Verify: `curl http://localhost:1234/v1/models`

---

## Step 2: Build TSLP (1 minute)

```bash
# Clone repository (if not already cloned)
git clone https://github.com/yourusername/tslp.git
cd tslp

# Build binary
go build -o tslp ./cmd/tslp

# Create runtime directory
mkdir -p runtime

# Verify build
./tslp --version
```

**Expected output:**
```
TSLP (Transactional State-Ledger Proxy) v5.3-gold
```

---

## Step 3: Start TSLP (30 seconds)

```bash
# Start TSLP (uses default config pointing to LM Studio)
./tslp
```

**Expected output:**
```
TSLP (Transactional State-Ledger Proxy) v5.3-gold
Per Specification v5.3 (Gold)

Phase 1/5: Loading configuration from config/config.toml
✓ Configuration validated

Phase 2/5: Initializing logger
✓ Logger initialized

Phase 3/5: Opening database at ./runtime/tslp.db
✓ Database opened

Phase 4/5: Running database migrations
✓ Migrations complete (WAL mode enabled)

Phase 5/5: Starting HTTP server
✓ HTTP server started

========================================
TSLP Ready
========================================

Endpoint: http://127.0.0.1:4321/v1/chat/completions
Log file: ./runtime/tslp.log

Press Ctrl+C to shutdown
```

---

## Step 4: Test It (1 minute)

**Open a new terminal window** (leave TSLP running):

```bash
# Test request
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-model",
    "messages": [
      {"role": "user", "content": "Write a Go function that adds two numbers"}
    ],
    "stream": false
  }'
```

**Expected behavior:**
1. Request goes to TSLP (port 4321)
2. TSLP hydrates context (empty on first request)
3. TSLP forwards to LM Studio (port 1234)
4. LM Studio generates code
5. TSLP parses response via Tree-sitter
6. TSLP promotes to AUTHORITATIVE if valid
7. Returns response

**Example response:**
```json
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "Here's a Go function that adds two numbers:\n\n```go\npackage main\n\nfunc Add(a, b int) int {\n    return a + b\n}\n```"
      }
    }
  ]
}
```

---

## Step 5: Verify State (30 seconds)

```bash
# Check State Map
curl http://localhost:4321/state | jq

# Expected: One entity (the Add function)
{
  "authoritative_count": 1,
  "entities": [
    {
      "entity_key": "unknown::Add",
      "symbol": "Add",
      "state": "AUTHORITATIVE",
      "confidence": "CONFIRMED",
      "stale": false
    }
  ]
}
```

---

## Step 6: Test Continuity (1 minute)

```bash
# Send another request referencing previous code
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-model",
    "messages": [
      {"role": "user", "content": "Now write a Subtract function"}
    ],
    "stream": false
  }'
```

**What happens:**
1. TSLP hydrates the `Add` function into context
2. LM Studio sees the previous code
3. LM Studio writes `Subtract` in the same style
4. TSLP promotes `Subtract` to AUTHORITATIVE
5. Both functions now in State Map

**Verify:**
```bash
curl http://localhost:4321/state | jq '.authoritative_count'
# Expected: 2
```

---

## 🎉 Success!

You now have:
- ✅ LM Studio running a local model (port 1234)
- ✅ TSLP proxy managing state (port 4321)
- ✅ AST-based entity resolution working
- ✅ State Map tracking authoritative code
- ✅ Continuity across multiple requests

---

## Next Steps

### 1. Enable Streaming
```bash
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-model",
    "messages": [
      {"role": "user", "content": "Write a hello world program"}
    ],
    "stream": true
  }'
```

### 2. Paste Manual Code
```bash
curl -X POST http://localhost:4321/v1/user/code \
  -H "Content-Type: application/json" \
  -d '{
    "content": "package main\n\nfunc Multiply(a, b int) int {\n  return a * b\n}",
    "filepath": "/project/math.go"
  }'
```

### 3. Monitor Diagnostics
```bash
# Health check
curl http://localhost:4321/health

# System status
curl http://localhost:4321/doctor | jq

# View State Map
curl http://localhost:4321/state | jq

# Recent activity
curl http://localhost:4321/recent | jq
```

### 4. Enable Debug Logging
Edit `config/config.toml`:
```toml
[logging]
debug = true
```

Restart TSLP, then:
```bash
tail -f runtime/tslp.log
```

### 5. Test ETV (Disk Divergence Detection)
```bash
# 1. Create a file
echo 'package main

func Test() string {
  return "original"
}' > test.go

# 2. Paste it to TSLP
curl -X POST http://localhost:4321/v1/user/code \
  -H "Content-Type: application/json" \
  -d @- << 'EOF'
{
  "content": "package main\n\nfunc Test() string {\n  return \"original\"\n}",
  "filepath": "/absolute/path/to/test.go"
}
EOF

# 3. Manually edit test.go on disk
echo 'package main

func Test() string {
  return "modified"
}' > test.go

# 4. Try to update via LLM
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-model",
    "messages": [
      {"role": "user", "content": "Update the Test function"}
    ]
  }'

# Expected: STATE NOTICE about divergence, promotion blocked
```

---

## Troubleshooting

### LM Studio Connection Failed
```bash
# Check if LM Studio is running
curl http://localhost:1234/v1/models

# If failed:
# 1. Open LM Studio
# 2. Go to "Local Server" tab
# 3. Click "Start Server"
```

### TSLP Port Already in Use
```bash
# Check what's using port 4321
lsof -i :4321

# Or change port in config/config.toml:
[proxy]
listen_address = "127.0.0.1:5432"  # Use different port
```

### Model Not Found
```bash
# Check loaded model name in LM Studio
curl http://localhost:1234/v1/models

# Update config/config.toml:
[llm]
llm_model = "actual-model-name-from-lmstudio"
```

### Database Locked
```bash
# Stop TSLP (Ctrl+C)
# Remove lock
rm -f runtime/tslp.db-wal runtime/tslp.db-shm
# Restart TSLP
./tslp
```

---

## Common Commands Cheat Sheet

```bash
# Build
go build -o tslp ./cmd/tslp

# Run
./tslp

# Run with custom config
./tslp --config /path/to/config.toml

# Test connection
curl http://localhost:4321/health

# Check state
curl http://localhost:4321/state | jq

# View logs
tail -f runtime/tslp.log

# Stop TSLP
# Press Ctrl+C in terminal where TSLP is running
```

---

## File Locations

```
tslp/
├── tslp                    # Binary (after build)
├── config/config.toml      # Configuration
└── runtime/
    ├── tslp.db            # SQLite database
    ├── tslp.db-wal        # Write-ahead log
    ├── tslp.db-shm        # Shared memory
    └── tslp.log           # Log file
```

---

## What's Next?

- Read full documentation: `README.md`
- Learn about ETV: `ETV_IMPLEMENTATION_COMPLETE.md`
- Understand spec: `specification.md`
- View conformance: `CONFORMANCE_REVIEW.md`

---

**You're ready to use TSLP for agentic coding with small local models!**

*Remember: TSLP makes small models reliable by providing external memory and structural verification. The model doesn't need to remember what it wrote—TSLP hydrates that knowledge on every request.*
