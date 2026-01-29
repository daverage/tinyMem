# Tasks - Version Automation

- [x] Implement Git-based version injection
- [x] Consolidate build and release scripts into `build/build.sh` and `build/build.bat`
- [x] Improve release robustness (handle existing tags/releases, clobber assets)
- [x] Remove obsolete "REPOSITORY VERSIONING RULES" from Agent docs
  - [x] Update `AGENTS.md`
  - [x] Update `CLAUDE.md`
  - [x] Update `GEMINI.md`
  - [x] Update `QWEN.md`

# Tasks - Dashboard Fixes

- [x] Fix task pulling in Dashboard
  - [x] Update parser to handle tasks without subtasks
  - [x] Add auto-sync from `tinyTasks.md` to Dashboard load
  - [x] Fix ProjectID consistency to ensure correct memory retrieval

# Tasks - Ralph Loop (The Governor)

- [x] Design and implement `memory_ralph` MCP tool
  - [x] Create `internal/ralph` package for loop state management
  - [x] Implement Evidence Gating (connecting to `internal/evidence`)
  - [x] Implement Safety Layer (path/command blacklists)
  - [x] Implement "Repair Phase" logic (internal LLM coordination)
  - [x] Add `memory_ralph` to MCP server

# Tasks - Documentation & Cleanup

- [x] Document `memory_ralph` in README.md
- [x] Document `addContract` in README.md
- [x] Move agent directive files to `docs/agents/`
- [x] Update `AddContract` command to use new directory structure
- [x] Remove redundant `cmd/add_contract/` utility
- [x] Fix `addContract` to look for local `AGENT_CONTRACT.md` first and update GitHub URL
- [x] Implement logic to replace old contracts with the new version in `addContract`

# Tasks - README Review (User Request)

- [x] Audit codebase to identify all public tools/commands
- [x] Review current README.md content
- [x] Update README.md to explain what each tool is, why it exists, and how to use it
- [x] Ensure non-technical explanations are included

# Tasks - Investigation

- [x] Investigate why `memory_stats` is not exposed in MCP server
- [x] Investigate why COVE stats never show

# Tasks - LM Studio & Qwen Configuration

- [x] Verify LM Studio proxy compatibility with `qwen2.5-coder-7b-instruct`
- [x] Create/Update example configuration for Qwen on LM Studio
- [x] Create Aider integration guide for tinyMem + LM Studio
- [x] Document specific requirements for Qwen (chat templates)
- [x] Add `llm.model` support to config for better local LLM compatibility
- [x] Implement automatic model normalization (prefix stripping) for LiteLLM/Aider compatibility
- [x] Improve proxy error reporting for unreachable backends
