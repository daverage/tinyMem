# Tasks - Version Automation

- [x] Implement Git-based version injection
- [x] Create automated release script (`build/release.sh`)
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

# Tasks - README Review (User Request)

- [x] Audit codebase to identify all public tools/commands
- [x] Review current README.md content
- [x] Update README.md to explain what each tool is, why it exists, and how to use it
- [x] Ensure non-technical explanations are included
