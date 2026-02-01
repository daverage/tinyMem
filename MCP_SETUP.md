# tinyMem MCP Setup Guide

## Quick Start

### For Claude Desktop

**1. Find your config file:**
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

**2. Add tinyMem MCP server:**

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "tinymem",
      "args": ["mcp"]
    }
  }
}
```

**3. Restart Claude Desktop**

That's it! tinyMem is now available as an MCP server.

---

## For Claude Code (CLI)

```bash
# Add tinyMem MCP server
claude mcp add tinymem -- tinymem mcp

# Verify it's running
claude mcp list

# Start using it
claude
# In chat: "Read the project memory"
```

---

## Expected Diagnostics in MCP Mode

When you run `tinymem doctor` in MCP mode, you'll see these **expected** warnings:

```
⚠️  LLM backend unreachable (expected in MCP mode - uses calling AI)
⚠️  Proxy not listening on port 8080 (expected in MCP mode - uses stdio transport)
```

**These are NOT errors!** They're informational messages that:
- LLM backend is unreachable: ✅ Normal - MCP uses the calling AI (Claude) for CoVe
- Proxy not listening: ✅ Normal - MCP uses stdio, not HTTP proxy

### What Should Concern You

Only these indicate real problems:
- ❌ Database file corrupted
- ❌ .tinyMem directory not writable
- ❌ Insufficient disk space

---

## Configuration Options

### Basic (No Configuration Needed)

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "tinymem",
      "args": ["mcp"]
    }
  }
}
```

### Advanced (With Logging)

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "/usr/local/bin/tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "debug",
        "TINYMEM_METRICS_ENABLED": "true"
      }
    }
  }
}
```

### Environment Variables for MCP

| Variable | Default | Purpose |
|----------|---------|---------|
| `TINYMEM_LOG_LEVEL` | `info` | Logging verbosity (debug, info, error) |
| `TINYMEM_METRICS_ENABLED` | `false` | Track recall statistics |
| `TINYMEM_RECALL_MAX_ITEMS` | `10` | Max memories per query |
| `TINYMEM_COVE_ENABLED` | `true` | Enable CoVe filtering |

---

## Available MCP Tools

When tinyMem is running as an MCP server, these tools are available:

| Tool | Purpose |
|------|---------|
| `memory_query` | Search project memories |
| `memory_recent` | Get recent memories |
| `memory_write` | Store new memory |
| `memory_stats` | Get memory statistics |
| `memory_health` | Check system health |
| `memory_doctor` | Run full diagnostics |

---

## Troubleshooting

### "command not found: tinymem"

**Fix:** Use absolute path in config:

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

Find the path:
```bash
which tinymem
```

### "No memories found"

**Cause:** Claude is not running from your project directory.

**Fix:** 
1. Close Claude Desktop
2. Open Terminal
3. `cd /path/to/your/project`
4. Launch Claude Desktop from that directory
5. Or set `TINYMEM_PROJECT_ROOT` env var

### MCP server not starting

**Check logs:**
- macOS: `~/.claude/logs/`
- Windows: `%APPDATA%\Claude\logs\`

**Common issues:**
- Binary not executable: `chmod +x /usr/local/bin/tinymem`
- Binary not in PATH: Use absolute path in config
- Config JSON syntax error: Validate JSON

### Doctor shows warnings

**If you see:**
- "LLM backend unreachable (expected in MCP mode)": ✅ Normal
- "Proxy not listening (expected in MCP mode)": ✅ Normal

**These are informational, not errors!**

---

## Testing Your Setup

1. **Verify tinyMem is installed:**
   ```bash
   tinymem version
   ```

2. **Add to Claude Desktop config**

3. **Restart Claude Desktop**

4. **Test in Claude:**
   ```
   Use the tinymem tool to check memory health
   ```

5. **Expected response:**
   Claude will call `memory_health` and report database status

---

## MCP vs Proxy Mode

| Feature | MCP Mode | Proxy Mode |
|---------|----------|------------|
| **Transport** | stdio | HTTP |
| **Use Case** | IDE integration | External apps |
| **LLM** | Calling AI (Claude) | External LLM |
| **Proxy Server** | Not needed | Required |
| **Best For** | Claude Desktop/Code | Custom integrations |

**For Claude Desktop/Code:** Always use MCP mode (what you're using now) ✅

---

## Next Steps

- ✅ MCP server configured
- ✅ Diagnostics explained
- ✅ Ready to use

**Try it:**
```
In Claude: "Store a memory that we prefer TypeScript for new features"
```

tinyMem will write this to your project's `.tinyMem/store/tinymem.db` and recall it in future conversations!
