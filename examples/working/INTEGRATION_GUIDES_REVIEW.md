# tinyMem Integration Guides Review

**Date:** January 30, 2026  
**Reviewed By:** Claude  
**Scope:** All integration documentation (README.md, LocalLLMs.md, IDEs.md, Crush.md, Aider.md, OpenAI.md, Gemini.md, Qwen.md, Claude.md)

---

## Executive Summary

**Overall Quality:** 🟢 **Strong Foundation**

Your integration guides are **well-structured and accurate**. They successfully explain how tinyMem connects to various ecosystems. However, there are **consistency gaps**, **missing details**, and **clarity issues** that should be addressed before public release.

### Key Findings

| Category | Status | Priority |
|----------|--------|----------|
| **Accuracy** | ✅ Mostly Correct | Low |
| **Completeness** | ⚠️ Gaps in Details | Medium |
| **Clarity** | ⚠️ Inconsistent Tone | Medium |
| **Examples** | ✅ Good | Low |
| **Troubleshooting** | ⚠️ Sparse | Medium |

---

## Document-by-Document Analysis

### 1. README.md

**Status:** ✅ **Good Opener**

#### Strengths
- Clear table comparing MCP vs Proxy modes
- Good navigation structure
- Concise directory overview

#### Issues
- **Missing:** No link to main tinyMem repo
- **Missing:** Quick troubleshooting link (all guides redirect to individual docs)
- **Clarity:** "AGENT MD" comment mentions legacy folder but doesn't explain where the modern agent directives are (docs/agents/)

#### Recommendations
```markdown
// ADD at top:
> **For the main tinyMem documentation, see:** https://github.com/daverage/tinyMem

// IMPROVE "About the Agent Directives":
The AI Agent Directives (system prompts) are now maintained in the root `docs/agents/` directory:
- [Claude Directive](../agents/CLAUDE.md)
- [Gemini Directive](../agents/GEMINI.md)
- [Qwen Directive](../agents/QWEN.md)
- [Generic Agent Contract](../agents/AGENT_CONTRACT.md)

(Legacy: The old `AGENT MD` folder is deprecated.)
```

---

### 2. Claude.md

**Status:** ✅ **Excellent - Most Detailed**

#### Strengths
- Clear separation of Claude Desktop vs CLI
- Good troubleshooting section
- Environment variable table is helpful
- Proper warning about PATH resolution

#### Issues
- **Minor:** "Registration" section shows inconsistent command syntax
  - First example: `claude mcp add tinymem -- tinymem mcp`
  - Second example: `claude mcp add tinymem -- /usr/local/bin/tinymem mcp`
  - These are subtly different (relative vs absolute path). Should clarify when to use which.

#### Recommendations
```markdown
### Registration

**Simple (if tinymem is in PATH):**
```bash
claude mcp add tinymem -- tinymem mcp
```

**Explicit Path (recommended):**
```bash
# Find the path:
which tinymem  # e.g., /usr/local/bin/tinymem

# Register with full path:
claude mcp add tinymem -- /usr/local/bin/tinymem mcp
```

**With Environment Variables:**
```bash
claude mcp add tinymem -- /usr/local/bin/tinymem mcp \
  TINYMEM_LOG_LEVEL=debug TINYMEM_METRICS_ENABLED=true
```
// (Note: Exact syntax depends on Claude CLI version; fallback to manual JSON editing if needed)
```
```

---

### 3. Qwen.md

**Status:** ⚠️ **Good but Needs Cross-Reference Clarity**

#### Strengths
- Excellent quick-start table
- Clear Ollama + LM Studio examples
- Good tips section on context window and CoVe

#### Issues
- **Duplication Risk:** The Ollama and LM Studio sections **duplicate info from LocalLLMs.md**
- **Missing:** No mention of how Qwen performs with CoVe vs other models
- **Clarity:** "Qwen CLI (Native)" section is vague. Doesn't name a specific tool.

#### Recommendations
```markdown
### 1. Qwen via Ollama (Proxy Mode)

> **See also:** [LocalLLMs.md - Ollama Section](LocalLLMs.md#1-ollama) for detailed Ollama setup.

// Then just show the tinyMem-specific config, not duplicate the Ollama setup steps.
```

Also:
```markdown
### Tips for Qwen

- **CoVe Performance:** Qwen 2.5 Coder has strong logical reasoning. It performs very well with CoVe enabled, often catching hallucinations that other models miss.
  ```toml
  [cove]
  enabled = true
  confidence_threshold = 0.65  # Conservative for better filtering
  ```
```

---

### 4. OpenAI.md

**Status:** ✅ **Solid but Missing Real Examples**

#### Strengths
- Clear explanation of how proxy mode works (3-step flow)
- Good coverage of response headers
- Both Python and Node.js examples provided

#### Issues
- **Missing:** No mention of Azure OpenAI compatibility
- **Example Problem:** The test `curl` example doesn't show expected output with tinyMem headers
- **Clarity:** Doesn't explain what happens if the backend API key is wrong (should mention `.tinyMem/config.toml`)

#### Recommendations
```markdown
## 5. Azure OpenAI (Proxy Mode)

If you're using Azure OpenAI, configure tinyMem to proxy to your Azure endpoint:

```toml
[proxy]
port = 8080
base_url = "https://<your-resource>.openai.azure.com/v1"

[llm]
model = "your-deployment-name"
```

Then point your client to `http://localhost:8080/v1` as before. tinyMem handles the Azure authentication.
```

And enhance the curl example:
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{...}' -v  # -v shows response headers

# Expected headers in response:
# X-TinyMem-Recall-Count: 3
# X-TinyMem-Recall-Status: injected
```
```

---

### 5. Aider.md

**Status:** ⚠️ **Good but Has Critical Note About Model Names**

#### Strengths
- Clear setup steps
- Good troubleshooting section
- **Critical point well-highlighted:** The `openai/` prefix is essential

#### Issues
- **Example command is long and hard to follow:** Should break into sections
- **Missing:** No mention of what happens if Aider tries to fetch model info (might fail with "unknown context")
- **Config File:** `.aider.model.metadata.json` example is helpful, but should show WHERE to place it in project structure

#### Recommendations
```markdown
### Option A: Command Line (Simplest)

```bash
tinymem proxy  # Terminal 1
```

Then in another terminal:
```bash
aider \
  --openai-api-base http://localhost:8080/v1 \
  --openai-api-key dummy \
  --model openai/qwen2.5-coder
```

**CRITICAL:** Always prefix the model with `openai/` (e.g., `openai/qwen2.5-coder`, not just `qwen2.5-coder`). This tells Aider to use the generic OpenAI client, which respects the custom API base.
```

And for the metadata file:
```markdown
### Metadata File Location

Place `.aider.model.metadata.json` in your **project root**:
```
your-project/
├── .aider.model.metadata.json
├── .tinyMem/
└── src/
```
```

---

### 6. Gemini.md

**Status:** ⚠️ **Incomplete - Needs Clarity on Setup Barriers**

#### Strengths
- Acknowledges the limitation (Gemini API isn't OpenAI-compatible)
- Shows MCP as an alternative

#### Issues
- **Critical Gap:** Section 1 is vague and incomplete. It says "Assuming tinyMem internal support for Gemini provider exists" — this is confusing. Does it or doesn't it?
- **Missing:** No actual working example. The config doesn't tell users HOW to set up the Gemini bridge.
- **Reality Check:** Most users will NOT be able to use Gemini as a backend without significant custom work. This should be stated upfront.

#### Recommendations
```markdown
# tinyMem Gemini Integration Guide

> **Current Limitation:** Google Gemini's API is NOT OpenAI-compatible. To use Gemini with tinyMem, you have limited options.

## Your Options

| Setup | Feasibility | Why |
|-------|-------------|-----|
| **Gemini as Backend for tinyMem** | ❌ Not Supported | Gemini API doesn't match OpenAI schema. Requires custom adapter. |
| **Use Gemini Agent with tinyMem (MCP)** | ✅ Supported | If you're building a Gemini agent that supports MCP. |
| **Use tinyMem with Local LLM, Query Gemini Separately** | ✅ Workaround | Run tinyMem with Ollama/LM Studio, use Gemini for different tasks. |

---

## Option 1: Gemini Agent with MCP (Recommended)

If you are building an autonomous agent using Gemini that supports Model Context Protocol (MCP), register tinyMem as a context provider...

[rest of existing MCP section]
```

---

### 7. Crush.md

**Status:** ✅ **Good and Complete**

#### Strengths
- Clear `.crush.json` structure
- Good example of natural language usage
- Advanced config section shows extensibility

#### Issues
- **Minor:** Doesn't mention whether Crush respects project-local `.crush.json` or only global `~/.config/crush/crush.json`
- **Missing:** No example of what to do if `tinymem` command is not found

#### Recommendations
```markdown
## Configuration

Crush looks for configuration in this order:
1. **Project-local:** `.crush.json` (in your project root) — recommended
2. **Global:** `~/.config/crush/crush.json` (applies to all projects)

Project-local config takes precedence.

1.  **Install tinyMem** and ensure it's in your PATH.
    ```bash
    which tinymem  # Should output: /usr/local/bin/tinymem (or similar)
    ```

2.  **Create/Edit `.crush.json` in your project root:**
    ```json
    {
      "mcp": {
        "tinymem": {
          "type": "stdio",
          "command": "tinymem",
          "args": ["mcp"]
        }
      }
    }
    ```
    
    If `tinymem` is NOT in your PATH, use the absolute path:
    ```json
    "command": "/usr/local/bin/tinymem"
    ```
```

---

### 8. IDEs.md

**Status:** ⚠️ **Lacks Depth - Needs Actual Tested Examples**

#### Strengths
- Covers multiple IDEs (good breadth)
- Shows both MCP and Proxy approaches

#### Issues
- **VS Code:** Section is vague. No link to which extension it's talking about.
- **Cursor:** The config location "typically in project settings or global settings" is unclear. Should specify actual paths.
- **Zed:** Config looks reasonable, but no verification step (e.g., how to test if it's connected).
- **Continue:** Good, but should mention restarting Continue after changing config.

#### Recommendations
```markdown
## 1. VS Code (via MCP Extension)

MCP support in VS Code requires an extension. **Recommended**: Use the official [Anthropic MCP extension](https://marketplace.visualstudio.com/items?itemName=...) or [Continue](https://continue.dev/).

### Using Continue (Recommended)

[Move the Continue section up and expand it]

```

For Cursor:
```markdown
## 2. Cursor

Cursor supports MCP natively since version 0.35+.

### Configuration

Create `.cursor/mcp.json` in your project root (or edit existing):

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "/usr/local/bin/tinymem",
      "args": ["mcp"]
    }
  }
}
```

Or globally at `~/.cursor/config/mcp.json`.

### Testing

1. Restart Cursor.
2. Open Composer (Cmd+K).
3. Type: "What does tinyMem know about this project?"
4. Cursor should show memory results in the response.
```
```

For Zed:
```markdown
## 3. Zed

Zed supports "language servers" and context providers natively.

### Configuration

Open Zed Settings (Cmd+, on macOS):

```json
{
  "context_servers": {
    "tinymem": {
      "command": "/usr/local/bin/tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Testing

1. Restart Zed.
2. Open Zed Assistant.
3. Ask a question about the project.
4. You should see memory being queried in the logs (`zed log` command).
```
```

---

### 9. LocalLLMs.md

**Status:** ✅ **Accurate but Sparse**

#### Strengths
- Clear structure (one provider per section)
- Correct configuration examples
- Good default explanations

#### Issues
- **Missing:** No verification/testing step (how do users test that Ollama is reachable?)
- **Missing:** What if the model name is wrong? How do users debug?
- **Llama.cpp:** The note about "usually ignores model name" is vague. Should clarify what happens if you pass a wrong name.

#### Recommendations
```markdown
## 1. Ollama

[Setup section stays the same]

### tinyMem Config (`.tinyMem/config.toml`)

```toml
[proxy]
port = 8080
base_url = "http://localhost:11434/v1"

[llm]
model = "llama3"
```

### Testing

Verify Ollama is reachable:
```bash
curl http://localhost:11434/v1/models
# Should list your models
```

Then test tinyMem:
```bash
tinymem proxy
# In another terminal:
curl http://localhost:8080/v1/models
# Should show the same models, with memory injected
```

### Troubleshooting

- **"Connection refused"**: Ensure `ollama serve` is running.
- **"Unknown model"**: Run `ollama list` to see available models and update `.tinyMem/config.toml` accordingly.
- **Slow responses**: Ollama may be downloading the model on first run. This can take several minutes.
```

Similar additions for LM Studio and Llama.cpp.

---

## Cross-Document Issues

### 1. **Inconsistent Configuration Format**

Some docs show full example configs:
```toml
[proxy]
port = 8080
base_url = "http://localhost:11434/v1"

[llm]
model = "llama3"
```

Others just mention it exists:
```
Point `base_url` there.
```

**Recommendation:** Always show a complete, copy-pasteable `.tinyMem/config.toml` snippet.

---

### 2. **No Central "Configuration Reference"**

Currently spread across docs. Users have to hunt for what config options exist.

**Recommendation:** Add a `Configuration.md` file that lists all available options:
```markdown
# tinyMem Configuration Reference

## `.tinyMem/config.toml` Options

### [proxy]
- `port` (int): HTTP port. Default: 8080
- `base_url` (string): Backend LLM provider URL

### [llm]
- `model` (string): Model identifier for the backend
- `timeout` (int): Request timeout in seconds. Default: 120

### [recall]
- `max_items` (int): Max memories per query. Default: 10
- `semantic_enabled` (bool): Enable semantic search. Default: false

[... etc]
```

---

### 3. **Inconsistent Troubleshooting**

Some guides have detailed troubleshooting (Claude.md, Aider.md). Others have none (LocalLLMs.md, Crush.md).

**Recommendation:** Add a "Common Issues" section to every guide:
```markdown
## Troubleshooting

### "Connection refused"
[How to diagnose and fix]

### "tinyMem not found"
[How to diagnose and fix]

### "No memories found"
[How to diagnose and fix]
```

---

### 4. **Missing Environment Variable Reference**

Each guide mentions different env vars:
- Claude.md: `TINYMEM_LOG_LEVEL`, `TINYMEM_METRICS_ENABLED`
- Qwen.md: `TINYMEM_RECALL_MAX_ITEMS` (not mentioned in others)
- LocalLLMs.md: None listed

**Recommendation:** Create a single source of truth. Add to README.md:
```markdown
## Environment Variables (All Modes)

| Variable | Default | Example |
|----------|---------|---------|
| `TINYMEM_LOG_LEVEL` | `info` | `TINYMEM_LOG_LEVEL=debug` |
| `TINYMEM_METRICS_ENABLED` | `false` | `TINYMEM_METRICS_ENABLED=true` |
| `TINYMEM_RECALL_MAX_ITEMS` | `10` | `TINYMEM_RECALL_MAX_ITEMS=20` |
| `TINYMEM_LLM_API_KEY` | (uses backend) | `TINYMEM_LLM_API_KEY=sk-...` |
| `TINYMEM_PROXY_PORT` | `8080` | `TINYMEM_PROXY_PORT=9090` |
```

---

### 5. **Unclear When to Use MCP vs Proxy**

Each guide mentions both, but doesn't always clarify the trade-offs.

**Recommendation:** Add a matrix to README.md:
```markdown
## Choosing Your Integration Mode

### MCP (Model Context Protocol)
**Use when:** Claude Desktop, Cursor, Zed, custom MCP-compatible tools  
**Pros:** Native IDE integration, no extra processes, automatic memory injection  
**Cons:** Only works with MCP-aware clients  

### Proxy Mode
**Use when:** OpenAI SDK, Aider, generic CLI tools, local LLM runners  
**Pros:** Works with any OpenAI-compatible client, central memory for all tools  
**Cons:** Extra HTTP process, requires port configuration  

### Dual Mode
**Use when:** You want both IDEs and CLI tools to share memory  
**Setup:** Run `tinymem mcp` (via IDE config) + `tinymem proxy` (separate terminal)  
```

---

## Recommendations by Priority

### 🔴 High Priority (Before Release)

1. **Gemini.md** — Rewrite to be honest about limitations. Currently misleading.
2. **IDEs.md** — Add actual tested configurations and verification steps.
3. **Cross-document consistency** — Standardize config examples, env vars, troubleshooting.

### 🟡 Medium Priority (Should Add)

4. **Configuration.md** — Central reference for all config options.
5. **LocalLLMs.md** — Add testing/verification steps.
6. **Qwen.md** — Remove duplication with LocalLLMs.md.
7. **Aider.md** — Clarify metadata file location and add more comprehensive model examples.

### 🟢 Low Priority (Nice to Have)

8. Add "Common Issues" to every guide.
9. Expand Crush.md troubleshooting.
10. Add Azure OpenAI section to OpenAI.md.

---

## Summary Checklist for Next Draft

- [ ] Rewrite Gemini.md to be clear about what is/isn't supported
- [ ] Standardize all `.tinyMem/config.toml` examples (always show complete snippets)
- [ ] Add environment variable reference to README.md
- [ ] Add MCP vs Proxy decision matrix to README.md
- [ ] Add "Testing/Verification" step to every integration guide
- [ ] Add "Troubleshooting" section to guides that lack it
- [ ] Fix Claude.md path examples (relative vs absolute clarity)
- [ ] Add VS Code extension recommendation and link
- [ ] Fix Qwen.md duplication with LocalLLMs.md (use cross-references)
- [ ] Enhance OpenAI.md with Azure and curl examples with actual response headers
- [ ] Clarify IDEs.md config file locations (project vs global)
- [ ] Add working test commands to LocalLLMs.md
- [ ] Create Configuration.md with full reference

---

## Closing Notes

Your guides are **well-written and mostly accurate**. The main issues are:

1. **Consistency** — Different styles, different levels of detail
2. **Completeness** — Some guides lack testing/verification steps
3. **Honesty** — A few sections (Gemini, VS Code) gloss over real limitations

With the above revisions, these will be **publication-quality documentation** that developers will trust and follow successfully.

Great work so far! 🎯
