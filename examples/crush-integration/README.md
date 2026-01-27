# Using tinyMem with Crush/Rush CLI

This guide shows how to integrate tinyMem as an MCP (Model Context Protocol) server with [Crush](https://github.com/charmbracelet/crush), the terminal-based AI coding assistant from Charm.

## Prerequisites

1. **Install Crush/Rush CLI**
   ```bash
   # Using Homebrew
   brew install charmbracelet/tap/crush

   # Or using Go
   go install github.com/charmbracelet/crush@latest
   ```

2. **Install tinyMem**
   ```bash
   # Build from source
   cd /path/to/tinyMem
   go build -tags fts5 -o tinymem ./cmd/tinymem

   # Or install globally
   go install -tags fts5 ./cmd/tinymem
   ```

3. **Initialize tinyMem in your project**
   ```bash
   cd /your/project
   tinymem init
   ```

## Configuration

Create a `.crush.json` file in your project root (or `~/.config/crush/crush.json` for global config):

```json
{
  "mcp": {
    "tinymem": {
      "type": "stdio",
      "command": "tinymem",
      "args": ["mcp"],
      "timeout": 120,
      "env": {
        "TINYMEM_METRICS_ENABLED": "true"
      }
    }
  }
}
```

### Configuration Options

| Option | Description |
|--------|-------------|
| `type` | Must be `"stdio"` for tinyMem |
| `command` | Path to tinymem binary (or just `"tinymem"` if in PATH) |
| `args` | Must include `["mcp"]` to start MCP server mode |
| `timeout` | Request timeout in seconds (default: 120) |
| `env` | Optional environment variables |

### Full Path Example

If tinymem is not in your PATH:

```json
{
  "mcp": {
    "tinymem": {
      "type": "stdio",
      "command": "/usr/local/bin/tinymem",
      "args": ["mcp"],
      "timeout": 120
    }
  }
}
```

### With Custom Configuration

```json
{
  "mcp": {
    "tinymem": {
      "type": "stdio",
      "command": "tinymem",
      "args": ["mcp"],
      "timeout": 120,
      "env": {
        "TINYMEM_METRICS_ENABLED": "true",
        "TINYMEM_LOG_LEVEL": "info",
        "TINYMEM_SEMANTIC_ENABLED": "false"
      }
    }
  }
}
```

## Available Tools

Once configured, Rush will have access to these tinyMem tools:

| Tool | Description |
|------|-------------|
| `memory_query` | Search memories using full-text or semantic search |
| `memory_recent` | Retrieve the most recent memories |
| `memory_write` | Create a new memory entry with optional evidence |
| `memory_stats` | Get statistics about stored memories |
| `memory_health` | Check the health status of the memory system |
| `memory_doctor` | Run diagnostics on the memory system |

## Usage Examples

### Querying Memories

In Rush, you can ask the AI to use tinyMem:

```
> What decisions have been made about the database schema?
```

Rush will use the `memory_query` tool to search for relevant memories.

### Writing Memories

```
> Remember that we decided to use SQLite for the local database
```

Rush can use `memory_write` to store this decision.

### Checking Memory Health

```
> Check if the memory system is healthy
```

Rush will use `memory_health` to verify the system status.

## Memory Types

When writing memories through Rush, these types are available:

| Type | Description |
|------|-------------|
| `fact` | Verified information with evidence |
| `claim` | Unverified assertions |
| `plan` | Future intentions |
| `decision` | Choices that were made |
| `constraint` | Limitations or requirements |
| `observation` | Things noticed during work |
| `note` | General notes |

## Best Practices

1. **Initialize per project**: Run `tinymem init` in each project root to create project-specific memory stores.

2. **Use project-local config**: Place `.crush.json` in your project root for project-specific settings.

3. **Enable metrics**: Set `TINYMEM_METRICS_ENABLED=true` to track recall effectiveness.

4. **Query before coding**: Ask Rush to check memories before making changes to stay aligned with past decisions.

5. **Write decisions**: Have Rush record important decisions as memories for future context.

## Troubleshooting

### MCP Server Not Starting

1. Verify tinymem is in PATH:
   ```bash
   which tinymem
   ```

2. Test MCP server manually:
   ```bash
   echo '{"method":"initialize","id":1}' | tinymem mcp
   ```

3. Check for errors in logs:
   ```bash
   cat .tinyMem/logs/tinymem-*.log
   ```

### No Memories Found

1. Ensure you're in the correct project directory
2. Check if `.tinyMem` directory exists
3. Run `tinymem health` to verify the database

### Permission Errors

Ensure the tinymem binary has execute permissions:
```bash
chmod +x /path/to/tinymem
```

## Example Session

```
$ rush

> Check what memories exist for this project
[Rush uses memory_recent tool]

Found 3 memories:
1. [decision] Use SQLite for local storage
2. [constraint] Must support offline operation
3. [note] API endpoints defined in api/routes.go

> Remember that we decided to add caching with Redis
[Rush uses memory_write tool]

Memory created successfully!

> What do we know about caching?
[Rush uses memory_query tool]

Found 1 memory about caching:
- [decision] Added caching with Redis
```
