# TSLP — Transactional State-Ledger Proxy

**Version:** 5.3 (Gold) + ETV
**Status:** Production Ready
**License:** MIT

> *Deterministic continuity for agentic coding with small models (3B–14B) by externalizing working memory into a strictly typed Transactional State Map.*

---

## 📖 Table of Contents

- [What is TSLP?](#what-is-tslp)
- [Why TSLP?](#why-tslp)
- [Core Principles](#core-principles)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [API Endpoints](#api-endpoints)
- [Usage Examples](#usage-examples)
- [External Truth Verification (ETV)](#external-truth-verification-etv)
- [Diagnostics](#diagnostics)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Specification](#specification)

---

## What is TSLP?

TSLP is a **local HTTP proxy** that sits between your code editor and a small language model (3B–14B parameters), providing **deterministic state management** for agentic coding workflows.

Unlike traditional context-window approaches that rely on the model to "remember" what it wrote, TSLP:

1. **Stores every code artifact** in an immutable vault (content-addressed storage)
2. **Tracks which version is authoritative** via a strict state machine
3. **Hydrates the model's context** with current truth on every request
4. **Verifies structural correctness** before allowing changes to advance state
5. **Detects disk divergence** and blocks unsafe overwrites of manual edits

**Result:** Small models (even 3B) can maintain consistent, multi-file codebases without hallucinating or losing track of what they wrote.

---

## Why TSLP?

### The Problem: Small Models Lose Continuity

Small language models (3B–14B) are fast and run locally, but they struggle with:

- ❌ **Context Overflow:** Forget what they wrote beyond context window
- ❌ **Blind Overwrites:** Accidentally overwrite code they didn't see
- ❌ **No Structural Understanding:** Can't verify their output is valid
- ❌ **Hallucinated State:** Make up functions or variables that don't exist
- ❌ **Manual Edit Conflicts:** Unaware when you've edited files on disk

### The Solution: TSLP's Approach

- ✅ **Externalized Memory:** State Map holds authoritative truth, not the model
- ✅ **Structural Proof:** AST parsing verifies code before accepting it
- ✅ **Hydration:** Full current state injected on every request
- ✅ **Overwrite Protection:** Structural parity checks prevent data loss
- ✅ **Disk Divergence Detection:** ETV detects manual file edits
- ✅ **Provable Continuity:** Every state change is logged and rebuildable

---

## Core Principles

### 1. The LLM is Stateless
The model retains no internal history. TSLP provides all necessary context on every request.

### 2. The Proxy is Authoritative
The State Map, not the model, is the source of truth.

### 3. State Advances Only by Structural Proof
Changes must be provably correct (via AST parsing or exact regex match) before becoming authoritative.

### 4. No Blind Overwrites
Nothing is modified without explicit acknowledgement. Structural parity guards prevent accidental data loss.

### 5. Structural Continuity
Continuity is maintained via AST symbols and file structure, not language patterns.

### 6. Materialized Truth
Truth is injected (hydrated), never inferred or "remembered" by the model.

### 7. Disk is Higher Authority (ETV)
Manual file edits are detected and prevent unsafe LLM overwrites.

---

## Quick Start

### Prerequisites

- **Go 1.22+** (for building from source)
- **LM Studio** (or any OpenAI-compatible LLM endpoint)
- **A code editor** that can send HTTP requests

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/tslp.git
cd tslp

# Build the binary
go build -o tslp ./cmd/tslp

# Create runtime directory (for database and logs)
mkdir -p runtime

# Copy and edit configuration
cp config/config.toml config/config.toml.local
# Edit config/config.toml with your LLM settings
```

### Configuration

Edit `config/config.toml`:

```toml
[database]
database_path = "./runtime/tslp.db"

[logging]
log_path = "./runtime/tslp.log"
debug = false

[llm]
llm_provider = "lmstudio"
llm_endpoint = "http://localhost:1234/v1"  # LM Studio default
llm_api_key = ""                            # Empty for local models
llm_model = "local-model"                   # Your loaded model name

[proxy]
listen_address = "127.0.0.1:4321"          # TSLP proxy port
```

### Running TSLP

```bash
# Start the proxy
./tslp --config config/config.toml

# Or use default config location
./tslp
```

**Expected Output:**
```
TSLP (Transactional State-Ledger Proxy) v5.3-gold
Per Specification v5.3 (Gold)

Phase 1/5: Loading configuration from config/config.toml
✓ Configuration validated

Phase 2/5: Initializing logger (log_path=./runtime/tslp.log, debug=false)
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

Core Principles:
  • The LLM is stateless
  • The Proxy is authoritative
  • State advances only by structural proof
  • Nothing is overwritten without acknowledgement
  • Continuity is structural, not linguistic
  • Truth is materialized, never inferred

Endpoint: http://127.0.0.1:4321/v1/chat/completions
Log file: ./runtime/tslp.log

Press Ctrl+C to shutdown
```

### Testing the Proxy

```bash
# Simple test request
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-model",
    "messages": [
      {"role": "user", "content": "Write a hello world function in Go"}
    ]
  }'
```

---

## Architecture

### Three-Layer Storage (Single SQLite Database)

```
┌─────────────────────────────────────────────────────────┐
│                    TSLP Proxy (Port 4321)                │
│  OpenAI-compatible endpoint: /v1/chat/completions       │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                   Runtime Engine                         │
│  • Entity Resolution (AST → Regex → Correlation)        │
│  • Promotion Gates (Structural Proof + Authority Grant) │
│  • External Truth Verification (ETV)                    │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        ▼                   ▼                   ▼
┌──────────────┐  ┌──────────────────┐  ┌──────────────┐
│    VAULT     │  │   STATE MAP      │  │    LEDGER    │
│ (Immutable)  │  │ (Current Truth)  │  │ (Audit Log)  │
├──────────────┤  ├──────────────────┤  ├──────────────┤
│ Content-     │  │ filepath::symbol │  │ Episodes     │
│ Addressed    │  │ → artifact_hash  │  │ Transitions  │
│ Storage      │  │                  │  │ Audits       │
│              │  │ AUTHORITATIVE    │  │              │
│ SHA-256      │  │ PROPOSED         │  │ Chronological│
│ Hashing      │  │ SUPERSEDED       │  │ Evidence     │
│              │  │ TOMBSTONED       │  │              │
└──────────────┘  └──────────────────┘  └──────────────┘
                            │
                            ▼
                  ┌──────────────────┐
                  │  Filesystem      │
                  │  (Read-Only)     │
                  │  ETV Verification│
                  └──────────────────┘
```

### State Machine

```
┌──────────┐
│ PROPOSED │ ◄─── New artifact arrives
└────┬─────┘
     │
     │ Promotion Gates:
     │   Gate A: Structural Proof (CONFIRMED + Parity)
     │   Gate B: Authority Grant (User/Audit/Hydration)
     │   ETV Gate: Disk Consistency Check
     │
     ▼
┌──────────────┐
│AUTHORITATIVE │ ◄─── Current truth, hydrated to LLM
└────┬────┬────┘
     │    │
     │    └──────────┐
     │               ▼
     │         ┌─────────────┐
     │         │ TOMBSTONED  │ ◄─── Symbol removed
     │         └─────────────┘
     │
     ▼
┌────────────┐
│ SUPERSEDED │ ◄─── Replaced by newer version
└────────────┘
```

---

## Configuration

### Minimal Configuration (Required Fields Only)

```toml
[database]
database_path = "./runtime/tslp.db"

[logging]
log_path = "./runtime/tslp.log"
debug = false

[llm]
llm_provider = "lmstudio"
llm_endpoint = "http://localhost:1234/v1"
llm_api_key = ""
llm_model = "local-model"

[proxy]
listen_address = "127.0.0.1:4321"
```

### Configuration for Different LLM Providers

**LM Studio (Local, Default):**
```toml
[llm]
llm_provider = "lmstudio"
llm_endpoint = "http://localhost:1234/v1"
llm_api_key = ""
llm_model = "local-model"
```

**Ollama (Local):**
```toml
[llm]
llm_provider = "ollama"
llm_endpoint = "http://localhost:11434/v1"
llm_api_key = ""
llm_model = "llama3:7b"
```

**OpenAI (Cloud):**
```toml
[llm]
llm_provider = "openai"
llm_endpoint = "https://api.openai.com/v1"
llm_api_key = "sk-..."
llm_model = "gpt-4"
```

**Anthropic (Cloud):**
```toml
[llm]
llm_provider = "anthropic"
llm_endpoint = "https://api.anthropic.com"
llm_api_key = "sk-ant-..."
llm_model = "claude-3-opus-20240229"
```

---

## API Endpoints

### Main Endpoints

#### `POST /v1/chat/completions`
OpenAI-compatible chat completion endpoint.

**Request:**
```json
{
  "model": "local-model",
  "messages": [
    {"role": "user", "content": "Write a function..."}
  ],
  "stream": true
}
```

**Response:**
- Streams SSE (Server-Sent Events) if `stream: true`
- Returns JSON if `stream: false`
- Hydrates current state before sending to LLM
- Processes artifacts after LLM response
- Promotes to AUTHORITATIVE if gates pass

#### `POST /v1/user/code`
User write-head endpoint for pasting code directly.

**Request:**
```json
{
  "content": "package main\n\nfunc Hello() string {\n  return \"world\"\n}",
  "filepath": "/path/to/file.go"
}
```

**Response:**
```json
{
  "artifact_hash": "abc123...",
  "entity_key": "/path/to/file.go::Hello",
  "confidence": "CONFIRMED",
  "state": "AUTHORITATIVE",
  "promoted": true
}
```

**Behavior:**
- User-pasted code is **instantly AUTHORITATIVE**
- Supersedes all prior LLM artifacts
- No promotion gates apply (User Write-Head Rule)

---

### Diagnostic Endpoints

#### `GET /health`
Simple liveness check.

**Response:**
```json
{
  "status": "ok",
  "timestamp": 1735059600
}
```

#### `GET /doctor`
Comprehensive system health check.

**Response:**
```json
{
  "database": {
    "connected": true,
    "vault_count": 42,
    "state_count": 15,
    "ledger_count": 58
  },
  "llm": {
    "provider": "lmstudio",
    "endpoint": "http://localhost:1234/v1",
    "model": "local-model"
  },
  "proxy": {
    "listen_address": "127.0.0.1:4321",
    "uptime_seconds": 3600
  },
  "etv": {
    "stale_count": 0,
    "file_read_errors": []
  }
}
```

#### `GET /state`
Current State Map status.

**Response:**
```json
{
  "authoritative_count": 5,
  "entities": [
    {
      "entity_key": "file.go::Function",
      "filepath": "/path/to/file.go",
      "symbol": "Function",
      "state": "AUTHORITATIVE",
      "confidence": "CONFIRMED",
      "artifact_hash": "abc123...",
      "last_updated": 1735059600,
      "stale": false
    }
  ]
}
```

#### `GET /recent`
Recent episodes (metadata only, no code).

**Response:**
```json
{
  "episodes": [
    {
      "episode_id": "uuid",
      "timestamp": 1735059600,
      "user_prompt_hash": "def456...",
      "assistant_response_hash": "ghi789...",
      "metadata": {
        "hydrated_entities": ["file.go::Func1", "file.go::Func2"]
      }
    }
  ]
}
```

#### `GET /debug/last-prompt` (Debug Mode Only)
Shows the exact prompt sent to the LLM (including hydration).

**Response:**
```json
{
  "episode_id": "uuid",
  "timestamp": 1735059600,
  "user_prompt_hash": "abc123...",
  "prompt_content": "[CURRENT STATE: AUTHORITATIVE]\n..."
}
```

**Note:** Only available when `debug = true` in config.

---

## Usage Examples

### Example 1: Basic Coding Session

```bash
# Start TSLP
./tslp

# In another terminal, send request
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

**What happens:**
1. TSLP receives request
2. Checks State Map for existing authoritative code
3. Hydrates context (none on first request)
4. Forwards to LLM at `http://localhost:1234/v1`
5. LLM generates code
6. TSLP parses response via Tree-sitter
7. Detects `func Add(a, b int) int` → CONFIRMED
8. Promotes to AUTHORITATIVE
9. Stores in State Map
10. Returns response to client

### Example 2: Manual Code Paste

```bash
# Paste your manually written code
curl -X POST http://localhost:4321/v1/user/code \
  -H "Content-Type: application/json" \
  -d '{
    "content": "package math\n\nfunc Multiply(a, b int) int {\n  return a * b\n}",
    "filepath": "/project/math/multiply.go"
  }'
```

**What happens:**
1. Code is parsed via Tree-sitter
2. Entity resolved: `/project/math/multiply.go::Multiply`
3. Immediately promoted to AUTHORITATIVE
4. Supersedes any prior LLM version
5. Will be hydrated in next LLM request

### Example 3: Disk Divergence Detection (ETV)

```bash
# Initial state: TSLP has file.go::Func with hash abc123

# You manually edit file.go on disk (outside TSLP)
# File now has hash def456

# Next LLM request:
curl -X POST http://localhost:4321/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "local-model",
    "messages": [
      {"role": "user", "content": "Update the Func function"}
    ]
  }'
```

**What happens:**
1. TSLP detects disk hash (def456) ≠ State Map hash (abc123)
2. Entity marked as STALE
3. Excluded from hydration
4. LLM receives STATE NOTICE:
   ```
   [STATE NOTICE: DISK DIVERGENCE DETECTED]
   Entity: file.go::Func has been modified on disk
   These entities have been EXCLUDED from hydration
   User must paste updated content via /v1/user/code
   [END NOTICE]
   ```
5. If LLM tries to update: promotion BLOCKED
6. User must paste updated file to resolve

---

## External Truth Verification (ETV)

### What is ETV?

External Truth Verification detects when files on disk have diverged from the State Map, preventing TSLP from operating on stale assumptions.

### How It Works

1. **Hash Comparison:** TSLP reads file from disk, computes SHA-256 hash
2. **STALE Detection:** If `diskHash ≠ stateMapHash`, entity is STALE
3. **Hydration Filtering:** STALE entities excluded from LLM context
4. **Promotion Blocking:** LLM cannot promote artifacts for STALE entities
5. **User Resolution:** User must paste updated content or confirm overwrite

### Authority Model (Verification Only)

```
Highest → User-pasted code      (via POST /v1/user/code)
          ↓
          Disk (read-only)       (verification only, never written)
          ↓
          State Map              (current truth)
          ↓
Lowest  → LLM output             (must prove correctness to promote)
```

### ETV Safety Guarantees

- ✅ **READ-ONLY:** TSLP never writes to disk
- ✅ **Deterministic:** SHA-256 hash comparison (no heuristics)
- ✅ **Fail-Safe:** Unreadable files treated as STALE
- ✅ **Explicit:** User action required to resolve divergence
- ✅ **Transparent:** STALE status visible in diagnostics

### Checking for STALE Entities

```bash
# Check State Map for divergence
curl http://localhost:4321/state | jq '.entities[] | select(.stale == true)'

# Check system health
curl http://localhost:4321/doctor | jq '.etv'
```

### Resolving STALE Entities

**Option 1: Paste Updated Content**
```bash
curl -X POST http://localhost:4321/v1/user/code \
  -H "Content-Type: application/json" \
  -d '{
    "content": "<updated file content>",
    "filepath": "/path/to/file.go"
  }'
```

**Option 2: Let User Confirm Overwrite**
(Future feature - currently requires paste)

---

## Diagnostics

### Monitoring TSLP

**Check if running:**
```bash
curl http://localhost:4321/health
```

**Full system check:**
```bash
curl http://localhost:4321/doctor | jq
```

**View State Map:**
```bash
curl http://localhost:4321/state | jq
```

**Recent activity:**
```bash
curl http://localhost:4321/recent | jq
```

### Log Files

**Location:** `./runtime/tslp.log`

**Debug mode:**
```toml
[logging]
debug = true
```

**View logs:**
```bash
tail -f ./runtime/tslp.log
```

### Database Inspection

**Location:** `./runtime/tslp.db`

**Schema:**
```bash
sqlite3 ./runtime/tslp.db .schema
```

**Query vault:**
```sql
SELECT hash, content_type, byte_size, created_at
FROM vault
ORDER BY created_at DESC
LIMIT 10;
```

**Query state map:**
```sql
SELECT entity_key, filepath, symbol, state, confidence, last_updated
FROM state_map
WHERE state = 'AUTHORITATIVE';
```

---

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/yourusername/tslp.git
cd tslp

# Install dependencies
go mod download

# Build
go build -o tslp ./cmd/tslp

# Run tests
go test ./...
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/entity/...

# With coverage
go test -cover ./...

# Verbose
go test -v ./internal/state/...
```

### Project Structure

```
tslp/
├── cmd/
│   └── tslp/
│       └── main.go              # Entry point
├── config/
│   ├── config.toml              # Default configuration
│   └── config.schema.json       # Configuration schema
├── internal/
│   ├── api/                     # HTTP handlers
│   │   ├── server.go
│   │   ├── diagnostics.go
│   │   └── user_code.go
│   ├── audit/                   # Shadow audit
│   ├── entity/                  # Symbol resolution
│   │   ├── ast.go              # Tree-sitter parsing
│   │   ├── regex.go            # Regex fallback
│   │   ├── correlation.go      # State Map correlation
│   │   └── symbols.json        # Language patterns
│   ├── fs/                      # Filesystem (read-only)
│   │   └── reader.go
│   ├── hydration/               # JIT state injection
│   │   ├── hydration.go
│   │   └── tracking.go
│   ├── ledger/                  # Append-only log
│   ├── llm/                     # LLM client
│   ├── logging/                 # Structured logging
│   ├── runtime/                 # Core lifecycle
│   ├── state/                   # State Map management
│   │   ├── state.go
│   │   ├── parity.go           # Structural parity
│   │   └── consistency.go      # ETV
│   ├── storage/                 # SQLite
│   └── vault/                   # Content-addressed storage
├── runtime/                     # Runtime data (gitignored)
│   ├── tslp.db                 # SQLite database
│   └── tslp.log                # Log file
├── docs/                        # Documentation
├── CONFORMANCE_REVIEW.md        # Spec compliance audit
├── ETV_IMPLEMENTATION_COMPLETE.md
├── ETV_SAFETY_AUDIT.md
├── README.md                    # This file
└── specification.md             # Full specification
```

---

## Troubleshooting

### TSLP Won't Start

**Error:** `FATAL: Configuration error`
- Check `config/config.toml` exists
- Verify all required fields are present
- Ensure database path is writable

**Error:** `Failed to open database`
- Create `runtime/` directory: `mkdir -p runtime`
- Check filesystem permissions
- Ensure SQLite is available

### LLM Connection Issues

**Error:** `Failed to start streaming`
- Verify LM Studio (or LLM endpoint) is running
- Check `llm_endpoint` in config matches LM Studio port
- Test endpoint directly: `curl http://localhost:1234/v1/models`

**No response from LLM:**
- Check LM Studio has a model loaded
- Verify model name matches `llm_model` in config
- Check LM Studio logs for errors

### Promotion Failures

**Artifact stays PROPOSED:**
- Entity resolution may have failed (INFERRED or UNRESOLVED)
- Check logs: `tail -f runtime/tslp.log`
- Enable debug mode: `debug = true` in config

**"STALE - disk content differs" error:**
- File was manually edited on disk
- Paste updated content via `/v1/user/code`
- Or check disk hash vs State Map: `GET /state`

### Performance Issues

**Slow responses:**
- Small models should respond in <2s
- Check LM Studio GPU utilization
- Enable streaming: `"stream": true`

**High memory usage:**
- Check vault size: `SELECT COUNT(*) FROM vault;`
- Database will grow over time (this is expected)
- Consider archiving old episodes

---

## Specification

**Full Specification:** See `specification.md`

**Key Documents:**
- `CONFORMANCE_REVIEW.md` — Spec compliance audit
- `ETV_IMPLEMENTATION_COMPLETE.md` — External Truth Verification
- `ETV_SAFETY_AUDIT.md` — Safety verification
- `IMPLEMENTATION_COMPLETE.md` — Gold implementation status

**Specification Version:** v5.4 (Gold)

**Implementation Status:**
- ✅ Steps 1-8: Complete (Gold spec)
- ✅ External Truth Verification (ETV): Complete
- ✅ All safety guarantees verified
- ✅ Production ready

---

## FAQ

### Q: Why small models (3B–14B)?

Small models run locally, are fast, and have low latency. TSLP makes them viable for complex coding tasks by providing external memory and structural verification.

### Q: Does TSLP work with GPT-4 or Claude?

Yes! TSLP is provider-agnostic. Configure `llm_endpoint` to point to any OpenAI-compatible API. However, TSLP's benefits are most pronounced with smaller models.

### Q: What happens if I edit files outside TSLP?

ETV (External Truth Verification) detects manual edits via hash comparison. STALE entities are excluded from hydration and cannot be overwritten by LLM output. You must paste updated content to resolve.

### Q: Can I use TSLP with my existing IDE?

Yes, if your IDE can send HTTP requests to `http://localhost:4321/v1/chat/completions`. TSLP is a standard OpenAI-compatible proxy.

### Q: Is the State Map stored in the database?

Yes, all three layers (Vault, State Map, Ledger) are stored in a single SQLite database at `runtime/tslp.db`.

### Q: Can I rebuild the State Map from scratch?

Yes! The State Map is rebuildable from Vault + Ledger. This is a core design principle.

### Q: Does TSLP support languages other than Go?

Currently, Tree-sitter AST parsing supports Go. Regex fallback works for any language with patterns in `symbols.json`. Additional languages can be added by extending `internal/entity/ast.go`.

### Q: What does "boring, correct, inspectable" mean?

TSLP prioritizes:
- **Boring:** No clever abstractions, predictable behavior
- **Correct:** Strict adherence to specification, no shortcuts
- **Inspectable:** All state changes are logged and auditable

---

## Contributing

This project follows strict implementation guidelines per `CLAUDE.md`.

**Before contributing:**
1. Read `specification.md`
2. Review `CLAUDE.md` for implementation rules
3. Understand the "no invention" policy
4. All changes must reference spec sections

**Pull requests must:**
- Include spec section references
- Pass all tests: `go test ./...`
- Build successfully: `go build ./cmd/tslp`
- Not weaken any safety guarantees

---

## License

MIT License

Copyright (c) 2024 TSLP Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

---

## Contact & Support

**Issues:** https://github.com/yourusername/tslp/issues
**Documentation:** https://github.com/yourusername/tslp/wiki
**Specification:** `specification.md` in this repository

---

**TSLP** — Making small models reliable for agentic coding through deterministic state management.

*Built with boring, correct, inspectable code. No magic. No surprises.*
