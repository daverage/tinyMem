# tinyMem Examples & Integration Guides

This directory contains comprehensive guides for integrating `tinyMem` with various LLM providers, clients, and IDEs.

## Provider Guides

*   **[Claude](Claude.md):** Integration with Claude Desktop, Claude CLI, and MCP.
*   **[GitHub Copilot](GitHubCopilot.md):** Configuration for Copilot Chat in VS Code.
*   **[Qwen](Qwen.md):** Setup for Qwen CLI, Ollama, and LM Studio.
*   **[Gemini](Gemini.md):** Using Gemini via MCP or with an adapter.
*   **[OpenAI](OpenAI.md):** Using the standard OpenAI Python/Node SDKs with tinyMem.
*   **[DeepSeek](DeepSeek.md):** Configuration for DeepSeek API and local R1 models.
*   **[Aider](Aider.md):** Configuring the Aider AI pair programmer.
*   **[Crush/Rush](Crush.md):** Using Charm's Crush CLI with native MCP support.
*   **[LangChain](LangChain.md):** Integration examples for LangChain Python.
*   **[Windsurf](Windsurf.md):** Setup for Codeium's Windsurf IDE.
*   **[Cline](Cline.md):** Setup for the Cline VS Code agent.

## Ecosystem Guides

*   **[IDEs](IDEs.md):** VS Code, Cursor, Zed, and Continue configuration.
*   **[Local LLMs](LocalLLMs.md):** Generic configuration for backends like Ollama, LM Studio, and Llama.cpp.
*   **[Configuration](Configuration.md):** **Full reference** for `.tinyMem/config.toml` options.

## About the Agent Directives (AGENT MD)

The `AGENT MD` folder (legacy) contains specific prompt directives for different AI models. These are now maintained in the root `docs/agents/` directory:
- [Claude Directive](../docs/agents/CLAUDE.md)
- [Gemini Directive](../docs/agents/GEMINI.md)
- [Qwen Directive](../docs/agents/QWEN.md)

## Quick Reference: Modes

| Mode | Best For | How it works |
|------|----------|--------------|
| **MCP** | Claude Desktop, Crush, Cursor, Zed, Windsurf, Cline | `tinymem` runs as a stdio server, responding to tool calls. |
| **Proxy** | OpenAI SDK, Aider, LangChain, Copilot, DeepSeek | `tinymem` runs an HTTP server (`:8080`), intercepting and injecting memory into API calls. |
