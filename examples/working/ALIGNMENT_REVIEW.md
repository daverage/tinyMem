# tinyMem Integration Guides Alignment Review

**Comparing:** Integration Guides (examples/) vs Main Repository README  
**Status:** 🟢 **Very Good Alignment** with minor improvements needed

---

## Executive Summary

Your integration guides are **highly aligned** with the main README. They correctly represent tinyMem's architecture, modes, and philosophy. However, there are some **minor inconsistencies** and **missing cross-references** that should be fixed for a cohesive documentation experience.

---

## What's Aligned Well ✅

### 1. **Core Concepts**
- ✅ Both README and guides correctly explain Proxy Mode vs MCP Mode
- ✅ Evidence-based truth validation is consistently mentioned
- ✅ Chain-of-Verification (CoVe) is correctly described
- ✅ Local-first, privacy-focused approach is consistent
- ✅ MCP tool names match (`memory_query`, `memory_write`, `memory_ralph`, etc.)

### 2. **Configuration**
- ✅ `.tinyMem/config.toml` structure is correct in guides
- ✅ Environment variables mentioned match README
- ✅ Zero-config philosophy is represented

### 3. **Target Audience**
- ✅ Guides correctly identify who should use each mode
- ✅ Focus on small-to-medium LLMs (7B–13B) is consistent
- ✅ "Project-scoped" memory emphasis is throughout

### 4. **Installation**
- ✅ Pre-built binaries, from-source, and Docker options all mentioned where relevant
- ✅ No contradictions on how to get tinyMem running

---

## Gaps & Inconsistencies

### 🟡 Gap 1: Integration Directory Not Mentioned in README

**Location:** README.md line 35-38

Currently:
```markdown
- [Integration](#-integration)
  - [Proxy Mode](#proxy-mode)
  - [MCP Server (IDE Integration)](#mcp-server-ide-integration)
  - [AI Agent Directives](#ai-agent-directives)
```

**Issue:** The README doesn't mention that detailed provider-specific guides exist in `examples/` directory.

**Recommendation:**
```markdown
## 🔌 Integration

tinyMem supports two primary modes. For **detailed integration guides for your specific tool/provider**, see the [Integration Guides Directory](examples/README.md):

- [Claude Integration](examples/Claude.md)
- [Qwen Integration](examples/Qwen.md)
- [OpenAI & SDK Guide](examples/OpenAI.md)
- [IDE Setup (VS Code, Cursor, Zed)](examples/IDEs.md)
- [Local LLM Configuration](examples/LocalLLMs.md)
- [Aider Setup](examples/Aider.md)
- [Crush/Rush Setup](examples/Crush.md)
- [Gemini Integration](examples/Gemini.md)

### Quick Overview: Proxy Mode
```

This makes the relationship between the README and integration guides crystal clear.

---

### 🟡 Gap 2: Agent Directives Not Cross-Referenced

**Location:** README.md lines 277-285

Currently mentions directives exist but doesn't explain their purpose clearly:
```markdown
**CRITICAL**: If you are building an AI agent, you MUST include the appropriate directive 
in its system prompt to ensure it uses tinyMem correctly.

**Quick Setup:** Run `tinymem addContract` to automatically create these files in your project.

*   **Claude**: [`docs/agents/CLAUDE.md`](docs/agents/CLAUDE.md)
```

**Integration Guide Issue:** The integration guides (Claude.md, Qwen.md, etc.) don't mention these directives or recommend users read them.

**Recommendation in Claude.md:**
```markdown
## 3. System Prompt Integration (Optional but Recommended)

For Claude agents using tinyMem, include the **tinyMem Control Protocol** in your system prompt:

```markdown
# tinyMem Control Protocol

You have access to a project memory system (tinyMem) via the `memory_*` tools.

... [embed or link to docs/agents/CLAUDE.md content]
```

See [docs/agents/CLAUDE.md](../../docs/agents/CLAUDE.md) for the complete protocol.
```

Similar additions needed in Qwen.md, Gemini.md, etc.

---

### 🟡 Gap 3: Configuration Reference Inconsistency

**README.md lines 340-365** shows:
```toml
[recall]
max_items = 10
semantic_enabled = false

[cove]
enabled = true
confidence_threshold = 0.6

[logging]
level = "info"
file = "tinymem.log"
```

**Integration Guides show various snippets:**
- Qwen.md: Shows `[cove]` config
- LocalLLMs.md: Shows `[proxy]` and `[llm]` config
- No guide shows complete `.tinyMem/config.toml` reference

**Issue:** Users don't know what other config sections exist (e.g., `[proxy]`, `[llm]`).

**Recommendation:** Create a canonical `.tinyMem/config.toml` reference that can be linked from all guides:

Create `docs/CONFIGURATION_REFERENCE.md`:
```markdown
# Configuration Reference

## Complete `.tinyMem/config.toml` Template

```toml
# Proxy Mode Settings
[proxy]
port = 8080                          # HTTP port for proxy
base_url = "http://localhost:11434"  # Backend LLM provider URL

# LLM Backend Selection
[llm]
model = "llama3"                     # Model identifier for backend
timeout = 120                        # Request timeout (seconds)

# Memory Recall Settings
[recall]
max_items = 10                       # Max memories per query
semantic_enabled = false             # Enable semantic search
hybrid_search = true                 # Use FTS + semantic

# Chain-of-Verification Settings
[cove]
enabled = true                       # Enable CoVe filter
confidence_threshold = 0.6           # Min confidence for storage

# Logging Configuration
[logging]
level = "info"                       # Log level (debug, info, warn, error, off)
file = "tinymem.log"                 # Log file location
max_size_mb = 50                     # Max log file size before rotation

# The Ralph Loop (Autonomous Repair)
[memory_ralph]
max_iterations = 5                   # Max repair attempts
allow_shell = false                  # Allow shell pipelines
forbid_paths = [".git", "node_modules"]  # Protected paths
```

## All Options Explained

[... detailed table for each option]
```

Then link from all guides:
```markdown
See [Configuration Reference](../../docs/CONFIGURATION_REFERENCE.md) for all available options.
```

---

### 🟡 Gap 4: MCP Tools Documentation

**README.md lines 267-275** lists MCP tools:
```
- `memory_query`: Search the past...
- `memory_recent`: Get up to speed...
- `memory_write`: Learn something new...
- `memory_ralph`: Self-Repair...
- `memory_stats` & `memory_health`: System Check...
- `memory_doctor`: Self-Diagnosis...
```

**Integration Guides:** Don't explain what these tools do or how to use them from the agent's perspective.

**Recommendation:** Add to Claude.md (and similar guides):
```markdown
## 4. Using Memory Tools

Once connected, Claude can automatically use these memory tools:

### `memory_query`
Claude uses this to **search for relevant memories**. Example flow:
> User: "What was our decision on database schema?"
> Claude: (calls `memory_query` with "database schema")
> tinyMem: (returns matching facts/decisions)
> Claude: "We decided to use PostgreSQL because..."

### `memory_write`
Claude uses this to **save important decisions**. Example:
> Claude: "I've identified that we need to switch from REST to GraphQL. Should I save this?"
> tinyMem: (stores as a claim, waits for evidence)

### `memory_ralph`
Claude uses this for **autonomous repair**. Example:
> Claude: "The tests are failing. Let me try fixing this autonomously."
> (calls `memory_ralph` with test command)
> tinyMem: (runs tests → analyzes failures → suggests fixes → re-runs)

See [Available MCP Tools](../../README.md#available-mcp-tools) for complete reference.
```

---

### 🟡 Gap 5: Recall Tiers Not Explained in Integration Guides

**README.md lines 73** mentions:
```
**Recall Tiers**: Prioritizes `Always` (facts) > `Contextual` (decisions) > `Opportunistic` (notes).
```

**Integration Guides:** None explain what these tiers mean or when each is used.

**Recommendation:** Add to each guide that discusses memory:
```markdown
## Understanding Recall Tiers

tinyMem prioritizes memories based on confidence:

| Tier | Confidence | Examples | Used When |
|------|-----------|----------|-----------|
| **Always** | ✅ Verified Facts | "We use PostgreSQL", "Deployment URL is..." | AI needs reliable facts |
| **Contextual** | 🤔 Validated Decisions | "We chose Go over Rust", "API is REST not GraphQL" | Relevant to current task |
| **Opportunistic** | 💭 Tentative Notes | "Consider this approach", "Maybe use caching" | If space/time allows |

When you ask Claude a question, it retrieves memories in this order, ensuring the most reliable information is considered first.
```

---

### 🔴 Critical Issue: Gemini.md Contradicts README Philosophy

**README.md line 21:**
> "tinyMem gives small and medium language models (7B–13B) reliable long-term memory"

**README.md lines 268:**
> "When tinyMem is running in MCP mode, your AI agent (like Claude or **Gemini**) gains these 'superpowers'"

**Gemini.md Section 1:**
```markdown
## 1. Proxy Mode (Gemini as Backend)

If you want to use Google's Gemini API as the intelligence behind tinyMem...
```

**Issue:** README suggests Gemini works with MCP (which it does), but also lists it as a backend option. Gemini.md then claims you can use Gemini as a backend for tinyMem's internal LLM, which is misleading (Gemini API isn't OpenAI-compatible).

**What README Actually Supports:**
1. ✅ Using Gemini agent with tinyMem MCP
2. ❌ Using Gemini as tinyMem's internal LLM backend (not supported without custom adapter)

**Fix Gemini.md:**
```markdown
# tinyMem Gemini Integration Guide

> **Two Use Cases:**
> 1. ✅ **Use a Gemini agent with tinyMem** (Recommended) - Your Gemini agent queries tinyMem for memory
> 2. ❌ **Use Gemini as tinyMem's backend** - Not directly supported (Gemini API isn't OpenAI-compatible)

See [README](../../README.md) for supported backends.
```

---

## Cross-Reference Opportunities

### Missing Links from README to Guides

**Location:** README.md lines 91-101 (Quick Start)

Currently shows generic examples. Should point to specific guides:

```markdown
### 2. Run

Start the server (choose one mode):

**Option A: Proxy Mode** (for generic LLM clients)
```bash
tinymem proxy
# Then point your client (e.g., OpenAI SDK) to http://localhost:8080/v1

# Examples for specific tools:
# - Using with Aider: see examples/Aider.md
# - Using with Qwen CLI: see examples/Qwen.md
# - Using with Ollama: see examples/LocalLLMs.md
```

**Option B: MCP Mode** (for Claude Desktop, Cursor, VS Code)
```bash
tinymem mcp
# Configure your IDE to run this command

# Examples for specific IDEs:
# - Claude Desktop: see examples/Claude.md
# - Cursor: see examples/IDEs.md#2-cursor
# - VS Code: see examples/IDEs.md#1-vs-code
# - Zed: see examples/IDEs.md#3-zed
```
```

---

## Missing Integration Guides

Based on the README's feature set, consider adding:

### 1. **The Ralph Loop Deep Dive** (`docs/RALPH_LOOP.md`)
- README mentions it but integration guides don't
- Users need a step-by-step guide on using `memory_ralph`
- Should include safety best practices

### 2. **tinyTasks Integration** (`examples/tinyTasks.md`)
- README mentions `tinyTasks.md` but no integration guide exists
- Users need to know how to use it with their agent
- Should show examples of task tracking in action

### 3. **Chain-of-Verification (CoVe) Explained** (`docs/COVE_EXPLAINED.md`)
- README mentions CoVe but doesn't explain how it works
- Users should understand confidence thresholds
- Should show examples of good vs. bad CoVe filtering

---

## Documentation Structure Recommendation

**Current:**
```
tinyMem/
├── README.md
├── docs/
│   ├── agents/
│   │   ├── CLAUDE.md
│   │   ├── GEMINI.md
│   │   ├── QWEN.md
│   │   └── AGENT_CONTRACT.md
│   └── QUICK_START_GUIDE.md
└── examples/
    ├── README.md
    ├── Claude.md
    ├── Qwen.md
    ├── OpenAI.md
    ├── IDEs.md
    ├── LocalLLMs.md
    ├── Aider.md
    ├── Crush.md
    └── Gemini.md
```

**Recommended Structure (Add):**
```
tinyMem/
├── README.md
├── docs/
│   ├── agents/
│   │   ├── CLAUDE.md
│   │   ├── GEMINI.md
│   │   ├── QWEN.md
│   │   └── AGENT_CONTRACT.md
│   ├── QUICK_START_GUIDE.md
│   ├── CONFIGURATION_REFERENCE.md  # NEW: Central config reference
│   ├── COVE_EXPLAINED.md            # NEW: CoVe deep dive
│   └── RALPH_LOOP_GUIDE.md          # NEW: Ralph Loop tutorial
└── examples/
    ├── README.md
    ├── Claude.md
    ├── Qwen.md
    ├── OpenAI.md
    ├── IDEs.md
    ├── LocalLLMs.md
    ├── Aider.md
    ├── Crush.md
    ├── Gemini.md
    └── tinyTasks_Integration.md     # NEW: tinyTasks tutorial
```

---

## Priority Fixes (Before Release)

### 🔴 Critical
1. **Fix Gemini.md** — Remove misleading Gemini-as-backend claim
2. **Link examples/ from README** — Make discovery of guides obvious

### 🟡 Important
3. **Create CONFIGURATION_REFERENCE.md** — Central config reference
4. **Add agent directives to integration guides** — Link to docs/agents/
5. **Add MCP tools explanation to each guide** — How to use them

### 🟢 Nice to Have
6. **Create COVE_EXPLAINED.md** — Help users understand confidence thresholds
7. **Create RALPH_LOOP_GUIDE.md** — Step-by-step autonomous repair tutorial
8. **Create tinyTasks_Integration.md** — Show how to use task tracking

---

## Summary Table

| Issue | Location | Severity | Impact | Fix |
|-------|----------|----------|--------|-----|
| No discovery of guides in README | README line 35 | 🔴 High | Users don't find integration docs | Add section linking to examples/ |
| Gemini.md contradicts README | Gemini.md + README | 🔴 High | Users get false expectations | Rewrite Gemini.md |
| Config options scattered | Multiple docs | 🟡 Medium | Users miss configuration options | Create CONFIGURATION_REFERENCE.md |
| Agent directives not mentioned in guides | Claude.md, Qwen.md, etc. | 🟡 Medium | Users don't use system prompts | Link to docs/agents/ in each guide |
| MCP tools not explained | Integration guides | 🟡 Medium | Users don't leverage tools | Add tool explanations |
| Recall tiers unexplained | Integration guides | 🟡 Medium | Users don't understand priority | Add tier explanation table |

---

## Overall Assessment

✅ **Alignment is Strong**

Your integration guides correctly represent tinyMem's core concepts and properly guide users. The main improvements are:

1. **Better cross-referencing** between README and examples/
2. **Central configuration reference** (currently scattered)
3. **Fixing Gemini.md** (misleading claims)
4. **More explicit agent directive guidance** (system prompt usage)

With these additions, your documentation will be **cohesive, discoverable, and complete**. 

Well done overall! 🎯
