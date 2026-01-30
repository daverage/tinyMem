# tinyMem Crush/Rush Integration Guide

[Crush](https://github.com/charmbracelet/crush) (and its CLI `rush`) supports the **Model Context Protocol (MCP)**, making integration with tinyMem native and powerful.

## Configuration

Crush looks for a configuration file at `.crush.json` (project local) or `~/.config/crush/crush.json` (global).

1.  **Install tinyMem** and ensure it's in your PATH.
2.  **Create/Edit `.crush.json`:**

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

    *If `tinymem` is not in PATH, replace `"command": "tinymem"` with `"command": "/absolute/path/to/tinymem"`.*

## Usage

Start Rush:
```bash
rush
```

### Querying Memory
You can ask Rush natural language questions about the project:
> "What decisions did we make about the API structure?"

Rush will automatically call `memory_query` to fetch context before answering.

### Writing Memory
> "Remember that we decided to use Postgres for production."

Rush will call `memory_write` to store this decision.

### Health Check
> "Check if the memory system is working."

Rush will call `memory_health`.

## Advanced Crush Config

You can tune the integration in `.crush.json`:

```json
{
  "mcp": {
    "tinymem": {
      "type": "stdio",
      "command": "tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "debug",
        "TINYMEM_RECALL_MAX_ITEMS": "20"
      },
      "disabled_tools": [] // You can explicitly disable tools if needed
    }
  },
  "system_prompt_suffix": "\n\nUse tinyMem to check context before answering code questions."
}
```

## Configuration Reference

For full configuration options, see [Configuration.md](Configuration.md).

## Troubleshooting

-   **Tools Not Available:** Check `rush --version` to ensure you have a version with MCP support.
-   **Execution Errors:** Run `rush` with debug flags or check `.tinyMem/logs/` to see if tinyMem is crashing or erroring.
