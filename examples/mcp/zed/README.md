# Configuring Zed IDE with tinyMem (MCP)

This guide provides information on how to potentially integrate Zed IDE with `tinyMem` running in MCP (Meta-Code Protocol) server mode.

Zed IDE is designed with robust AI integration, supporting various LLM providers and utilizing open standards like the Agent Client Protocol (ACP) and Model Context Protocol (MCP) for external AI tools.

## Current Understanding of Zed MCP Integration

While Zed IDE explicitly mentions supporting an "MCP" (Model Context Protocol), public documentation does not currently provide a direct example configuration for connecting Zed to an *external* MCP server using a `command` and `args` structure, similar to how Cursor (Claude Desktop) integrates external MCP servers.

Zed's primary method for AI integration involves configuring API keys for various LLM providers (OpenAI, Anthropic, Ollama, etc.) directly within its "Agent Panel settings" or via the `settings.json` file for custom API endpoints.

## Potential Integration Approach (if Zed supports external command execution)

If Zed IDE eventually provides a mechanism to configure an external AI agent or MCP server by specifying an executable command and its arguments, the configuration might look similar to the `claude_desktop_config.json` example:

```json
// This is a hypothetical example based on other IDE integrations.
// The exact format and location of Zed's configuration for an external
// MCP server are not explicitly documented as of this writing.
{
  "externalAIAgents": {
    "tinymem_mcp_agent": {
      "command": "/path/to/tinymem", // Absolute path to your tinymem executable
      "args": ["mcp"],
      "env": {} // Any environment variables needed by tinymem
    }
  }
}
```

**Where to look for future updates or configuration options:**

*   **Zed IDE Documentation:** Regularly check the official Zed IDE documentation and release notes for updates on external AI integration and MCP configuration.
*   **Zed Community Forums/Discord:** Engage with the Zed community to inquire about advanced AI integration possibilities.
*   **Zed `settings.json`:** Explore the `settings.json` file for any undocumented or experimental settings related to external AI providers or custom protocol servers.

## How to Use tinyMem with Zed Today (via Proxy)

Even without direct MCP integration, you can still use `tinyMem`'s proxy mode with Zed if Zed allows you to configure a custom API base URL for its built-in LLM providers (e.g., if you're using Zed with Ollama or LM Studio configured via `tinyMem`'s proxy).

1.  **Run tinyMem in proxy mode:**
    ```bash
    tinymem proxy
    ```
2.  **Configure Zed's LLM provider:** In Zed's "Agent Panel settings" or `settings.json`, set the API base URL for your chosen provider (e.g., OpenAI, Ollama) to point to `tinyMem`'s proxy: `http://localhost:8080/v1`.

For more details on `tinyMem`'s proxy mode, refer to the `examples/proxy/` directory.
