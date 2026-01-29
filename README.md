# tinyMem

<div align="center">
  <img src="assets/tinymem-logo.png" alt="tinyMem logo" width="280" />

  <p>
    <a href="https://github.com/andrzejmarczewski/tinyMem/blob/main/LICENSE">
      <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT" />
    </a>
    <a href="https://go.dev/dl/">
      <img src="https://img.shields.io/badge/Go-1.22+-00ADD8.svg" alt="Go 1.22+" />
    </a>
    <img src="https://img.shields.io/badge/Build-Passing-brightgreen.svg" alt="Build Status" />
  </p>

  <h3>Local, project-scoped memory system for language models with evidence-based truth validation.</h3>
</div>

---

tinyMem gives small and medium language models (7B–13B) reliable long-term memory in complex codebases. It sits between you and the LLM, injecting verified context and capturing validated facts—all locally, without model retraining or cloud dependencies.

## 📖 Table of Contents

- [Purpose](#-purpose)
- [Key Features](#-key-features)
- [Quick Start](#-quick-start)
- [Installation](#-installation)
- [Usage](#-usage)
  - [CLI Commands](#cli-commands)
  - [Writing Memories](#writing-memories)
  - [Memory Types & Truth](#memory-types--truth)
- [The Ralph Loop](#-the-ralph-loop-autonomous-repair)
- [Integration](#-integration)
  - [Proxy Mode](#proxy-mode)
  - [MCP Server (IDE Integration)](#mcp-server-ide-integration)
  - [AI Agent Directives](#ai-agent-directives)
- [Architecture](#-architecture)
- [Token Economics](#-token-efficiency--economics)
- [Configuration](#-configuration)
- [Development](#-development)
- [Contributing](#-contributing)
- [License](#-license)

---

## 🎯 Why tinyMem?

If you've ever used an AI for a large project, you know it eventually starts to "forget." It forgets which database you chose, it forgets the naming conventions you agreed on, and it starts making things up (hallucinating).

**tinyMem is a "Hard Drive for your AI's Brain."**

Instead of the AI trying to remember everything in its limited "short-term memory" (the chat window), tinyMem saves important facts and decisions to a local database on your computer. When the AI needs to answer a question or write code, tinyMem "reminds" it of the relevant facts.

*   **No more repeating yourself**: "Remember, we use Go for the backend."
*   **No more AI hallucinations**: If the AI isn't sure, it checks its memory.
*   **Total Privacy**: Your project data never leaves your machine to "train" a model.

---

## ✨ Key Features

*   **Evidence-Based Truth**: Typed memories (`fact`, `claim`, `decision`, etc.). Only verified claims become facts.
*   **Chain-of-Verification (CoVe)**: LLM-based quality filter to reduce hallucinations before storage and improve recall relevance (enabled by default).
*   **Local & Private**: Runs as a single binary. Data lives in `.tinyMem/`.
*   **Zero Configuration**: Works out of the box.
*   **Dual Mode**: Works as an HTTP Proxy or Model Context Protocol (MCP) server.
*   **Hybrid Search**: FTS (lexical) + Optional Semantic Search.
*   **Recall Tiers**: Prioritizes `Always` (facts) > `Contextual` (decisions) > `Opportunistic` (notes).

---

## 🚀 Quick Start

Get up and running in seconds.

### 1. Initialize
Go to your project root and initialize the memory database:
```bash
cd /path/to/your/project
tinymem health
```

### 2. Run
Start the server (choose one mode):

**Option A: Proxy Mode** (for generic LLM clients)
```bash
tinymem proxy
# Then point your client (e.g., OpenAI SDK) to http://localhost:8080/v1
```

**Option B: MCP Mode** (for Claude Desktop, Cursor, VS Code)
```bash
tinymem mcp
# Configure your IDE to run this command
```

---

## 📦 Installation

See the [Quick Start Guide for Beginners](docs/QUICK_START_GUIDE.md) for a detailed walkthrough.

### Option 1: Pre-built Binary (Recommended)
Download from the [Releases Page](https://github.com/andrzejmarczewski/tinyMem/releases).

**macOS / Linux**:
```bash
curl -L "https://github.com/andrzejmarczewski/tinyMem/releases/latest/download/tinymem-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)" -o tinymem
chmod +x tinymem
sudo mv tinymem /usr/local/bin/
```

**Windows**:
Download `tinymem-windows-amd64.exe`, rename to `tinymem.exe`, and add to your system `PATH`.

### Option 2: Build from Source
Requires Go 1.25.6+.
```bash
git clone https://github.com/andrzejmarczewski/tinyMem.git
cd tinyMem
./build/build.sh   # Build only
# or
./build/build.sh patch  # Release (patch version bump)
```

---

## 💻 Usage

### CLI Commands
The tinyMem CLI is your primary way to interact with the system from your terminal.

| Command | What it is | Why use it? | Example |
|:---|:---|:---|:---|
| `health` | **System Check** | To make sure tinyMem is installed correctly and can talk to its database. | `tinymem health` |
| `stats` | **Memory Overview** | To see how many memories you've stored and how your tasks are progressing. | `tinymem stats` |
| `dashboard` | **Visual Status** | To get a quick, beautiful summary of your project's memory "health." | `tinymem dashboard` |
| `query` | **Search** | To find specific information you or the AI saved previously. | `tinymem query "API"` |
| `recent` | **Recent History** | To see the last few things tinyMem learned or recorded. | `tinymem recent` |
| `write` | **Manual Note** | To tell the AI something important that it should never forget. | `tinymem write --type decision --summary "Use Go 1.25"` |
| `run` | **Command Wrapper**| To run a script or tool (like `make` or `npm test`) while "reminding" it of project context. | `tinymem run make build` |
| `proxy` / `mcp` | **Server Modes** | To start the "brain" that connects tinyMem to your IDE or AI client. | `tinymem mcp` |
| `doctor` | **Diagnostics** | To fix the system if it stops working or has configuration issues. | `tinymem doctor` |
| `addContract` | **Agent Setup** | To automatically configure your AI agents to use tinyMem properly. | `tinymem addContract` |

### Writing Memories
Think of writing memories as "tagging" reality for the AI.
```bash
# Record a decision so the AI doesn't suggest an alternative later
tinymem write --type decision --summary "Switching to REST" --detail "GraphQL was too complex for this scale."

# Add a simple note for yourself or the AI
tinymem write --type note --summary "The database password is in the vault, not .env"
```

### Memory Types & Truth

| Type | Evidence Required? | Truth State | Recall Tier |
|------|--------------------|-------------|-------------|
| **Fact** | ✅ Yes | Verified | Always |
| **Decision** | ✅ Yes (Confirmation) | Asserted | Contextual |
| **Constraint** | ✅ Yes | Asserted | Always |
| **Claim** | ❌ No | Tentative | Contextual |
| **Plan** | ❌ No | Tentative | Opportunistic |

*Evidence types supported: `file_exists`, `grep_hit`, `cmd_exit0`, `test_pass`.*

---

## 🤖 The Ralph Loop (Autonomous Repair)

The **Ralph Loop** (`memory_ralph`) is a deterministic governor for autonomous codebase repair. **It is not automatic; the AI must explicitly choose to "engage" it** when it detects a complex failure that requires iterative fixing. Once triggered, tinyMem takes control, iterating until evidence passes or limits are reached.

### 🔄 Execution Phases
| Phase | Action | Purpose |
|-------|--------|---------|
| **Execute** | Run Command | Executes the target verification (e.g., `go test`). |
| **Evidence** | Validate | Checks predicates (`test_pass`, `file_exists`). |
| **Recall** | Search | Retrieves failure patterns from long-term memory. |
| **Repair** | Apply Fix | tinyMem's internal LLM applies code changes. |

### 📜 Execution Contract
- **Evidence is King**: Only successful evidence checks can terminate the loop.
- **Safety First**: Supports path blacklisting (`forbid_paths`) and command blocking.
- **Shell Pipelines Are Opt-In**: Commands containing shell metacharacters (pipes, redirects, subshells) are rejected unless `allow_shell: true` is explicitly set in `memory_ralph` safety options.
- **Durable Memory**: The loop results are stored even if the agent is reset.

---

## 🔌 Integration

### Proxy Mode
Intercepts standard OpenAI-compatible requests.
```bash
export OPENAI_API_BASE_URL=http://localhost:8080/v1
# Your existing scripts now use tinyMem automatically
```

While proxying, tinyMem now reports recall activity back to the client so that downstream UIs or agents can show “memory checked” indicators:
* **Streaming responses** append an SSE event of type `tinymem.memory_status` once the upstream LLM finishes. The payload includes `recall_count`, `recall_status` (`none`/`injected`/`failed`), and a timestamp.
* **Non-streaming responses** carry the same data via new headers: `X-TinyMem-Recall-Status` and `X-TinyMem-Recall-Count`.
Agents or dashboards that read those fields can display whenever recall was applied or when the proxy skipped it.

### MCP Server (IDE Integration)
Compatible with Claude Desktop, Cursor, and other MCP clients.

**Claude Desktop Configuration** (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "tinymem": {
      "command": "/absolute/path/to/tinymem",
      "args": ["mcp"]
    }
  }
}
```
*Run `./verify_mcp.sh` to validate your setup.*

#### Available MCP Tools:
When tinyMem is running in MCP mode, your AI agent (like Claude or Gemini) gains these "superpowers":

*   **`memory_query`**: **Search the past.** The AI uses this to find facts, decisions, or notes related to its current task.
*   **`memory_recent`**: **Get up to speed.** The AI uses this when it first starts to see what has happened recently in the project.
*   **`memory_write`**: **Learn something new.** The AI uses this to save a new fact or decision it just discovered or made. *Facts require "Evidence" (like checking if a file exists).*
*   **`memory_ralph`**: **Self-Repair.** This is the "Nuclear Option." The AI uses this to try and fix a bug autonomously by running tests, reading errors, and retrying until it works.
*   **`memory_stats` & `memory_health`**: **System Check.** The AI uses these to check if its memory is working correctly or how much it has learned.
*   **`memory_doctor`**: **Self-Diagnosis.** If the AI feels "confused" or senses memory issues, it can run this to identify problems.

### AI Agent Directives
**CRITICAL**: If you are building an AI agent, you MUST include the appropriate directive in its system prompt to ensure it uses tinyMem correctly.

**Quick Setup:** Run `tinymem addContract` to automatically create these files in your project.

*   **Claude**: [`docs/agents/CLAUDE.md`](docs/agents/CLAUDE.md)
*   **Gemini**: [`docs/agents/GEMINI.md`](docs/agents/GEMINI.md)
*   **Qwen**: [`docs/agents/QWEN.md`](docs/agents/QWEN.md)
*   **Other**: [`docs/agents/AGENT_CONTRACT.md`](docs/agents/AGENT_CONTRACT.md)

---

## 🏗 Architecture

```mermaid
flowchart TD
    User[LLM Client / IDE] <-->|Request/Response| Proxy[TinyMem Proxy / MCP]
    
    subgraph "1. Recall Phase"
        Proxy --> Recall[Recall Engine]
        Recall -->|FTS + Semantic| DB[(SQLite)]
        Recall -->|Filter| Tiers{Recall Tiers}
        Tiers -->|Always/Contextual| Context[Context Injection]
    end
    
    subgraph "2. Extraction Phase"
        LLM[LLM Backend] -->|Stream| Proxy
        Proxy --> Extractor[Extractor]
        Extractor -->|Parse| CoVe{CoVe Filter}
        CoVe -->|High Conf| Evidence{Evidence Check}
        Evidence -->|Verified| DB
    end

    Context --> LLM
```

### File Structure
```
.
├── .tinyMem/             # Project-scoped storage (DB, logs, config)
├── assets/               # Logos and icons
├── build/                # Build scripts
├── cmd/                  # Application entry points
├── docs/                 # Documentation & Agent Contracts
├── internal/             # Core logic (Memory, Evidence, Recall)
└── README.md             # This file
```

## 📉 Token Efficiency & Economics

tinyMem uses more tokens per minute but **significantly fewer tokens per task** compared to standard agents.

| Feature | Token Impact | Why? |
| :--- | :--- | :--- |
| **Recall Engine** | 📉 **Saves** | Replaces "Read All Files" with targeted context snippets. |
| **Context Reset** | 📉 **Saves** | Prevents chat history from snowballing by starting iterations fresh. |
| **Truth Discipline**| 📉 **Saves** | Stops expensive "hallucination rabbit holes" before they start. |
| **Ralph Loop** | 📈 **Uses** | Requires multiple internal completions to reach autonomous success. |

**The Verdict:** tinyMem acts as a "Sniper Rifle" for context. By ensuring the few tokens sent are the *correct* ones, it avoids the massive waste of re-reading files and un-breaking hallucinated code.

---

## ⚙ Configuration

Zero-config by default. Override in `.tinyMem/config.toml`:

```toml
[recall]
max_items = 10
semantic_enabled = false # Set true if you have an embedding model

[cove]
enabled = true           # Chain-of-Verification (Extraction + Recall filtering)
confidence_threshold = 0.6
```

See [Configuration Docs](docs/QUICK_START_GUIDE.md) for details.

---

## 🛠 Development

```bash
# Run tests
go test ./...

# Build
./build/build.sh
```
See [Task Management](docs/tinyTasks.md) for how we track work.

---

## 🤝 Contributing

We value truth and reliability.
1.  **Truth Discipline**: No shortcuts on verification.
2.  **Streaming**: No buffering allowed.
3.  **Tests**: Must pass `go test ./...`.

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## 📄 License

[MIT](LICENSE) © 2026 Andrzej Marczewski
