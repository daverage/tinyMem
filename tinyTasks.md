# Tasks – Documentation & Examples Accuracy Audit

- [x] Repository-wide consistency pass
  - [x] Update canonical repo URLs and module paths where referenced
  - [x] Fix installation instructions (release asset names, arch mapping, container usage)
  - [x] Confirm CLI command names/flags match current implementation
- [x] Examples validation (CLI/IDE/LLM)
  - [x] Validate Claude Desktop MCP example config
  - [x] Validate VS Code / Cursor / Zed MCP examples
  - [x] Validate proxy example configs (Ollama/LM Studio/Qwen) match current config schema
  - [x] Validate CLI client examples (Claude/Gemini/Qwen/Aider/Codex) are accurate
- [x] Documentation review
  - [x] Remove stale claims and align docs to current behavior
  - [x] Cross-check key instructions with upstream vendor docs (internet)
- [x] Verification
  - [x] Run unit tests
  - [x] Spot-check common user flows (install, run proxy, run mcp)
