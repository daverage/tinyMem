# tinyMem

**Local, project-scoped memory system for language models with evidence-based truth validation.**

tinyMem is a standalone Go executable that gives small and medium language models reliable long-term memory in complex codebases. It acts as a truth-aware prompt governor, sitting between the developer and the LLM to inject verified context and capture validated facts—all without requiring model retraining or cloud services.

## Philosophy

tinyMem operates on three core principles:

1. **Memory is not gospel** – Model output is never trusted by default
2. **Facts require evidence** – Claims without verification are stored as claims, not facts
3. **Reality checks are free** – Evidence verification happens locally using filesystem checks, grep, command execution, and test runs

This approach prevents language models from hallucinating institutional knowledge while dramatically improving their ability to maintain context across long development sessions.

## Key Features

- **Evidence-Based Truth System**: All memory entries are typed (fact, claim, plan, decision, constraint, observation, note). Only claims with verified evidence become facts.
- **Chain-of-Verification (CoVe)**: Optional LLM-based quality filter that reduces hallucinated memory candidates before storage (disabled by default).
- **Local Execution**: Runs entirely on your machine as a single executable. No cloud dependencies.
- **Project-Scoped**: All state lives in `.tinyMem/` directory within your project
- **Streaming First**: Responses stream immediately—no buffering delays
- **Zero Configuration**: Works out of the box with sensible defaults
- **Dual Integration Mode**: Operates as HTTP proxy or MCP server for IDE integration
- **Token Budget Control**: Deterministic prompt injection with configurable limits
- **Hybrid Search**: Combines FTS (lexical) with optional semantic search

## Installation

### Download Pre-built Binary

Download the latest release for your platform from the [releases page](https://github.com/tinyMem/releases).

```bash
# macOS/Linux
curl -L https://github.com/tinyMem/releases/latest/download/tinymem-$(uname -s)-$(uname -m) -o tinymem
chmod +x tinymem
sudo mv tinymem /usr/local/bin/

# Or keep it local to your project
mv tinymem /path/to/your/project/
```

### Build from Source

```bash
git clone https://github.com/a-marczewski/tinymem.git
cd tinymem
go build -o tinymem ./cmd/tinymem
```

Once built, the `tinymem` executable will be in your current directory. For easier access, consider moving it to a directory included in your system's PATH (e.g., `/usr/local/bin/` on macOS/Linux) or adding your project directory to your PATH environment variable.

It's highly recommended to have the `tinymem` executable available in your system's PATH. This allows you to run `tinymem` commands from any directory without specifying the full path (e.g., `tinymem health` instead of `./tinymem health`). This is particularly important for seamless integration with IDEs and other tools that expect `tinymem` to be globally accessible.

### Adding `tinymem` to your PATH

To make `tinymem` easily callable from any directory:

**Option 1: Move to a system PATH directory (recommended for global access)**

```bash
# For macOS/Linux users, after building or downloading:
# Move the compiled binary to a directory already in your PATH, like /usr/local/bin/
sudo mv tinymem /usr/local/bin/
```
*Note: This requires administrator/root privileges.*

**Option 2: Add your project directory to your PATH (recommended for project-specific versions)**

If you prefer to keep the `tinymem` binary within your project directory, you can add that directory to your PATH. This is useful if you work on multiple projects that might require different `tinymem` versions.

*   **macOS/Linux (Bash/Zsh):**
    Open your `~/.bashrc`, `~/.bash_profile`, or `~/.zshrc` file and add the following line. Replace `/path/to/your/project` with the actual absolute path to your `tinymem` executable.
    ```bash
    export PATH="/path/to/your/project:$PATH"
    ```
    After saving, run `source ~/.bashrc` (or your respective shell config file) or restart your terminal.

*   **Windows (Command Prompt):**
    Open Command Prompt as administrator and run:
    ```cmd
    setx PATH "%PATH%;C:\path\to\your\project"
    ```
    Replace `C:\path\to/your/project` with the actual absolute path. You may need to restart your command prompt or computer for changes to take effect.

*   **Windows (PowerShell):**
    Run PowerShell as administrator and execute:
    ```powershell
    [Environment]::SetEnvironmentVariable("Path", "$env:Path;C:\path\to\your\project", "User")
    ```
    Replace `C:\path\to\your\project` with the actual absolute path. Restart PowerShell for changes to apply.

**Requirements**: Go 1.22 or later

## Quick Start

### 1. Initialize in Your Project

```bash
cd /path/to/your/project
tinymem health
```

This creates `.tinyMem/` directory structure and initializes the SQLite database.

### 2. Run as Proxy (Transparent Integration)

```bash
# Start the proxy server
tinymem proxy
```

Now, in a separate terminal where you run your LLM client (e.g., a script using the OpenAI library), configure it to use the `tinymem` proxy by setting the API base URL. This directs your client to send requests to `tinymem` instead of directly to the LLM provider.

```bash
# In your LLM client's terminal:
export OPENAI_API_BASE_URL=http://localhost:8080/v1
```

The proxy intercepts requests to your LLM, injects relevant memories, and captures new context automatically.

### 3. Run as MCP Server (IDE Integration)

```bash
# Start MCP server for stdio-based IDEs
tinymem mcp
```

Configure your IDE (Cursor, VS Code, etc.) to use tinyMem as an MCP server. See [IDE Integration](#ide-integration) below.

## Usage

### CLI Commands

```bash
# Health and diagnostics
tinymem health          # Check system health
tinymem doctor          # Run detailed diagnostics
tinymem stats           # Show memory statistics

# Memory operations
tinymem query "authentication flow"    # Search memories
tinymem recent                         # Show recent memories

# Server modes
tinymem proxy                          # Start HTTP proxy server
tinymem mcp                            # Start MCP server

# Utilities
tinymem run -- your-command            # Run command with memory context
tinymem version                        # Show version
```

### Memory Types

tinyMem categorizes all memory entries into typed buckets:

| Type | Description | Evidence Required | Auto-Promoted |
|------|-------------|-------------------|---------------|
| **fact** | Verified truth about the codebase | Yes | No |
| **claim** | Model assertion not yet verified | No | No |
| **plan** | Intended future action | No | No |
| **decision** | Confirmed choice or direction | Yes (confirmation) | No |
| **constraint** | Hard requirement or limitation | Yes | No |
| **observation** | Neutral context or state | No | Yes (low priority) |
| **note** | General information | No | Yes (lowest priority) |

### Evidence System

Evidence is verified locally without LLM calls:

```bash
# Example: Model claims "User authentication is handled in auth.go"
# tinyMem checks:
- file_exists: auth.go
- grep_hit: "func.*[Aa]uthenticate" in auth.go
- test_pass: go test ./internal/auth/...

# If checks pass → stored as fact
# If checks fail → stored as claim
```

Evidence types:
- `file_exists`: File or directory exists
- `grep_hit`: Pattern matches in file
- `cmd_exit0`: Command exits successfully
- `test_pass`: Test suite passes

## Architecture

```
┌─────────────┐
│  LLM Client │  (IDE, CLI tool, API client)
└──────┬──────┘
       │
       ↓
┌─────────────────────────────────────────┐
│           tinyMem Proxy/MCP             │
│  ┌───────────────────────────────────┐  │
│  │  1. Recall Engine                 │  │
│  │     - FTS search (BM25)           │  │
│  │     - Optional semantic search    │  │
│  │     - Token budget enforcement    │  │
│  │     - Optional CoVe filtering     │  │
│  └───────────────────────────────────┘  │
│                  ↓                       │
│  ┌───────────────────────────────────┐  │
│  │  2. Prompt Injection              │  │
│  │     - Bounded system message      │  │
│  │     - Type annotations            │  │
│  │     - Evidence markers            │  │
│  └───────────────────────────────────┘  │
└──────────┬──────────────────────────────┘
           │
           ↓
    ┌──────────────┐
    │  LLM Backend │  (Ollama, LM Studio, etc.)
    └──────┬───────┘
           │
           ↓ (streaming response)
    ┌──────────────────────────┐
    │  3. Extraction           │
    │     - Parse response     │
    │     - Extract claims     │
    │     - CoVe filter (opt.) │◄─ Quality gate
    │     - Validate evidence  │
    │     - Store safely       │
    └──────────────────────────┘
           ↓
    ┌──────────────────┐
    │  SQLite Storage  │
    │  (.tinyMem/store.sqlite3)
    └──────────────────┘
```

## Configuration

tinyMem works with zero configuration. Override defaults via `.tinyMem/config.toml`:

```toml
[proxy]
port = 8080
base_url = "http://localhost:11434/v1"  # Ollama default

[recall]
max_items = 10
max_tokens = 2000
semantic_enabled = false
hybrid_weight = 0.5  # Balance between FTS and semantic

[memory]
auto_extract = true
require_confirmation = false

[cove]
# Chain-of-Verification quality filter (disabled by default)
enabled = false
confidence_threshold = 0.6      # Min confidence to keep (0.0-1.0)
max_candidates = 20             # Max candidates per batch
timeout_seconds = 30            # LLM call timeout
model = ""                      # Empty = use default LLM
recall_filter_enabled = false   # Enable recall filtering

[logging]
level = "info"  # off, error, warn, info, debug
file = ".tinyMem/logs/tinymem.log"
```

### Environment Variables

```bash
TINYMEM_PROXY_PORT=8080
TINYMEM_LLM_BASE_URL=http://localhost:11434/v1
TINYMEM_LOG_LEVEL=debug

# CoVe settings (optional)
TINYMEM_COVE_ENABLED=true
TINYMEM_COVE_CONFIDENCE_THRESHOLD=0.7
TINYMEM_COVE_MAX_CANDIDATES=20
```

### Chain-of-Verification (CoVe)

CoVe is an optional quality filter that evaluates memory candidates before storage. When enabled, it:

- Assigns confidence scores (0.0-1.0) to each candidate based on specificity and certainty
- Filters out low-confidence, speculative, or hallucinated extractions
- Operates transparently with fail-safe fallback (errors don't block storage)
- **Never** participates in fact promotion (evidence verification is separate)

Enable CoVe when you want higher memory quality at the cost of an additional LLM call per extraction:

```bash
export TINYMEM_COVE_ENABLED=true
```

See [COVE.md](./COVE.md) for detailed documentation, configuration examples, and performance considerations.

## IDE Integration

### Claude Desktop / Cursor (MCP)

**Quick Start:** Run the verification script to ensure MCP is ready:
```bash
./verify_mcp.sh
```

This will test your setup and provide the exact configuration to copy.

**Manual Configuration:**

Add the following server configuration to your `claude_desktop_config.json` file. Note that the exact path to this file may vary slightly depending on your operating system and how you installed Claude Desktop.

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "/path/to/tinymem",
      "args": ["mcp"],
      "env": {}
    }
  }
}
```

**Important**: Use the absolute path to your tinymem executable. After updating the configuration, restart Claude Desktop.

For detailed MCP troubleshooting, see [MCP_TROUBLESHOOTING.md](./MCP_TROUBLESHOOTING.md).

Available MCP tools:
- `memory_query` - Search memories using full-text or semantic search
- `memory_recent` - Retrieve the most recent memories
- `memory_write` - Create a new memory entry with optional evidence
- `memory_stats` - Get statistics about stored memories (includes CoVe metrics when enabled)
- `memory_health` - Check the health status of the memory system
- `memory_doctor` - Run diagnostics on the memory system

### VS Code (via Continue or Similar)

Configure your LLM extension to use the `tinymem` proxy. Since the proxy forwards the request to your actual LLM backend (which is configured with the real API key), you can often use a dummy key in your editor's settings.

```json
{
  "continue.apiBase": "http://localhost:8080/v1",
  "continue.apiKey": "dummy" 
}
```

### Qwen Code CLI

See [QWEN.md](./QWEN.md) for detailed Qwen integration setup.

## AI Agent Directives

`tinyMem` is designed to be integrated with AI agents, providing them with a local, project-scoped memory system. To ensure effective and reliable interaction, AI agents should adhere to specific directives when using `tinyMem`.

### Core Principle: Memory-First Reasoning

**Your primary function is to leverage `tinyMem`'s memory to provide contextually-aware answers.** Before providing any code or explanation from your general knowledge, you MUST first consult `tinyMem`'s memory using `memory_query`.

### Available Memory Tools

- **`memory_query(query: str, limit: int = 10)`** - Search project memory (use this FIRST)
- **`memory_recent(count: int = 10)`** - Get recent memory entries
- **`memory_write(type: str, summary: str, ...)`** - Create new memory entries
- **`memory_stats()`** - View memory statistics (includes CoVe metrics when enabled)
- **`memory_health()`** - Check system health
- **`memory_doctor()`** - Run detailed diagnostics

### Quick Example

**User asks:** "How should we implement authentication?"

**Wrong approach (❌):**
```
You respond with general JWT/OAuth advice from your training...
```

**Correct approach (✅):**
```python
# Step 1: Query memory FIRST
memory_query(query='authentication implementation')

# Step 2: Synthesize from results
# Found: DECISION - "Use OAuth2 with Google/GitHub"
# Found: CONSTRAINT - "Must support enterprise SSO"

# Step 3: Answer based on memory
"Based on project decisions, you've chosen OAuth2 with
Google and GitHub providers, with plans to add enterprise
SSO. Would you like me to outline the implementation?"
```

### Complete Directives

For comprehensive AI Assistant Directives including:
- Mandatory reasoning process for every query
- Detailed tool usage guidelines
- Chain-of-Verification (CoVe) transparency
- Memory type best practices
- Complete workflow examples
- Critical reminders and error patterns

**See the full directives in [claude.md](./claude.md)**

### Adding Directives to Your AI Assistant

Pick the directive file that matches your model and paste its contents verbatim into your system prompt or project instructions (do not summarize or paraphrase).

**Choose the correct file:**
- **Claude:** `claude.md`
- **Gemini:** `GEMINI.md`
- **Qwen:** `QWEN.md`
- **Custom/other agents:** `AGENT.md`

This ensures the assistant:
1. Always queries memory before providing project-specific answers
2. Understands all memory types and tools
3. Knows when to create new memories
4. Recognizes CoVe's transparent operation
5. Follows evidence-based truth discipline

**Concrete examples:**
- **Claude Desktop/Cursor:** Paste `claude.md` into project instructions or `.clinerules`
- **Continue (VS Code):** Paste the matching file into the system message or a context file
- **Custom agents:** Prepend the matching file to the system prompt at initialization

## Example Workflow

1. **Start tinyMem proxy in your project:**
   ```bash
   cd ~/projects/myapp
   tinymem proxy
   ```

2. **Configure your LLM client** to point to `http://localhost:8080/v1`

3. **Work naturally with your LLM:**
   - Ask questions about your codebase
   - Request changes
   - Discuss architecture decisions

4. **tinyMem automatically:**
   - Injects relevant memories into each prompt
   - Captures facts from responses (with evidence)
   - Maintains truth discipline (claims ≠ facts)
   - Streams responses without delay

5. **Query memory state:**
   ```bash
   tinymem stats
   tinymem query "database schema"
   tinymem recent
   ```

## Advanced Usage

### Manual Memory Management

```bash
# Query specific topic
tinymem query "API endpoints" --limit 5

# View recent activity
tinymem recent --count 20

# Clear all memories (nuclear option)
rm -rf .tinyMem/store.sqlite3
tinymem health  # Recreates DB
```

### Running Commands with Context

```bash
# Inject memory context into command environment
tinymem run -- your-test-runner --verbose
```

### Troubleshooting

#### General Diagnostics

```bash
# Run comprehensive diagnostics
tinymem doctor

# Check what's failing:
# - DB connectivity
# - FTS availability
# - Semantic search status
# - LLM backend reachability
# - Filesystem permissions
# - Port conflicts
```

#### MCP Server Issues

**Error: "Request timed out" or "Client is not connected"**

These errors indicate the MCP server isn't maintaining a stable connection. The most common cause is logging output interfering with the stdio protocol. The latest version fixes this by using silent logging (file-only) for MCP mode.

To verify the fix worked:

1. **Verify tinymem path is absolute:**
   ```bash
   # Find the full path
   which tinymem
   # or if it's in your project directory
   pwd  # then use /full/path/to/tinymem
   ```

2. **Test MCP server manually:**
   ```bash
   cd /path/to/your/project
   ./tinymem mcp
   # Then send a test message:
   {"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05"},"id":1}
   # You should see a JSON response immediately
   ```

3. **Check logs:**
   ```bash
   # If logging is enabled, check the logs
   cat .tinyMem/logs/tinymem.log
   ```

4. **Verify database initialization:**
   ```bash
   # Run from your project directory
   ./tinymem health
   ```

5. **Restart Claude Desktop** after updating the MCP configuration

**Error: "Tool not found"**

If you get "tool not found" errors, make sure you're using underscore names (`memory_query`) not dot names (`memory.query`).

**MCP Logging**

When running in MCP mode, tinyMem automatically uses silent logging - all log messages go to `.tinyMem/logs/tinymem-YYYY-MM-DD.log` and nothing is written to stderr/stdout (which are reserved for JSON-RPC). This prevents log output from interfering with the MCP protocol.

To view logs while MCP is running:
```bash
tail -f .tinyMem/logs/tinymem-$(date +%Y-%m-%d).log
```

### Semantic Search Setup (Optional)

Enable semantic search for better phrasing flexibility:

1. **Install Ollama** with an embedding model:
   ```bash
   ollama pull nomic-embed-text
   ```

2. **Update config:**
   ```toml
   [recall]
   semantic_enabled = true
   embedding_model = "nomic-embed-text"
   ```

3. **Restart tinyMem**

Semantic search gracefully degrades to FTS-only if unavailable.

## Project Structure

```
.tinyMem/
├── store.sqlite3       # Memory database with FTS5
├── config.toml         # Optional configuration
├── logs/               # Log files (if enabled)
└── run/                # Runtime state
```

## How It Works

### 1. Recall Phase
When a prompt arrives, tinyMem:
- Searches memories using FTS (BM25 ranking)
- Optionally combines with semantic similarity
- Prioritizes constraints and decisions
- Enforces token budget

### 2. Injection Phase
Selected memories are formatted into a bounded system message:
```
[tinyMem Context]

CONSTRAINT: API keys must be stored in environment variables
(evidence: .env.example exists, grep confirms pattern)

FACT: Authentication uses JWT tokens
(evidence: auth.go:42, test suite passes)

CLAIM: Frontend uses React 18
(no evidence verification yet)
```

### 3. Extraction Phase
After the LLM responds:
- Parse output for claims, plans, decisions
- **Optional: CoVe filtering** - Assign confidence scores and filter low-quality candidates
- Default to non-fact types (never auto-promote to facts)
- Verify evidence for fact promotion (independent of CoVe)
- Store with timestamps and supersession tracking

**Note:** CoVe filtering is disabled by default and operates transparently when enabled. It reduces hallucinated extractions but never participates in fact promotion—only evidence verification can promote claims to facts.

## Invariants (Truth Discipline)

These guarantees hold everywhere in tinyMem:

1. **Memory ≠ Gospel**: Model output never auto-promoted to truth
2. **Typed Memory**: All entries have explicit types
3. **Evidence Required**: No evidence → not a fact (CoVe cannot bypass this)
4. **Bounded Injection**: Prompt injection is deterministic and token-limited
5. **Streaming Mandatory**: No response buffering (where supported)
6. **Project-Scoped**: All state lives in `.tinyMem/`
7. **Single Executable**: No dependencies beyond SQLite (embedded)
8. **CoVe Safety**: When enabled, CoVe filters quality but never changes types, creates facts, or overrides evidence verification

Violating any invariant is a bug, not a feature gap.

## Development

### Build

```bash
go build -o tinymem ./cmd/tinymem
```

### Test

```bash
go test ./...
```

### Cross-Platform Build

You can build `tinymem` for different operating systems and architectures by setting the `GOOS` (target operating system) and `GOARCH` (target architecture) environment variables before running the `go build` command.

Here are some common examples:

**For Linux:**
```bash
# For AMD64 (most common desktops and servers)
GOOS=linux GOARCH=amd64 go build -o tinymem-linux-amd64 ./cmd/tinymem

# For ARM64 (e.g., Raspberry Pi, some cloud instances)
GOOS=linux GOARCH=arm64 go build -o tinymem-linux-arm64 ./cmd/tinymem
```

**For macOS:**
```bash
# For Apple Silicon (M1, M2, etc.)
GOOS=darwin GOARCH=arm64 go build -o tinymem-darwin-arm64 ./cmd/tinymem

# For Intel-based Macs
GOOS=darwin GOARCH=amd64 go build -o tinymem-darwin-amd64 ./cmd/tinymem
```

**For Windows:**
```bash
# For AMD64
GOOS=windows GOARCH=amd64 go build -o tinymem-windows-amd64.exe ./cmd/tinymem

# For ARM64
GOOS=windows GOARCH=arm64 go build -o tinymem-windows-arm64.exe ./cmd/tinymem
```

The output binary will be named according to the `-o` flag in the command. You can then move this binary to the target machine and run it.

## Contributing

Contributions welcome! Please ensure:

1. **Truth discipline is maintained**: No shortcuts around evidence validation
2. **Streaming is preserved**: No buffering regressions
3. **Zero-config remains**: Defaults must work out of the box
4. **Tests pass**: `go test ./...`
5. **Doctor explains it**: If it can fail, `tinymem doctor` should diagnose it

See [TASKS.md](./TASKS.md) for the full implementation roadmap and design principles.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Why tinyMem?

Language models are powerful but have limited context windows and no persistent memory. Existing solutions either:
- Require expensive fine-tuning
- Depend on cloud services
- Trust model output uncritically
- Add latency through buffering
- Lack quality filtering for hallucinated extractions

tinyMem takes a different approach: treat the model as a conversational partner, but verify everything it claims against reality. Optional Chain-of-Verification (CoVe) filtering reduces hallucinated extractions before they pollute the memory system. This gives small models (7B-13B) the behavior of much larger models with long-term memory, while reducing token costs for all models through smart context injection.

The result: better model performance, lower costs, higher memory quality, and guaranteed truth discipline—all running locally with zero configuration.

---

**Built for developers who want their LLMs to remember context without hallucinating facts.**
