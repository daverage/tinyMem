# Project: tinyMem - Transactional State-Ledger Proxy

  ## Essential Commands

  - **Build:** `go build ./...`
  - **Test:** `go test ./...`
  - **Run:** `./tinyMem` (from cmd/tinyMem/main.go)
  - **Linting:** Not configured in this repository.

  ## Code Organization & Structure

  The project is organized into internal modules:

  - `cmd/tinyMem`: Entry point at `main.go`.
  - `internal/`: Core logic split across packages:
    - `api`: HTTP endpoints.
    - `storage`: SQLite storage layer (WAL mode, no ORM).
    - `audit`: Shadow audit trail logic.
    - `entity`, `ledger`, `vault`, and others implementing core functions.

  ## Naming Conventions & Style

  - **Packages:** Modularized into clear subdirectories under internal/.
  - **Files:** No external tooling; native Go idioms used (e.g., JSON config
in config/config.toml).

  ## Testing Approach

  The project uses standard Go testing with `go test` across the repository.

  - Unit tests should be added to source packages where appropriate.
  - Test coverage is handled via standard tools during CI runs.

  ## Gotchas & Non-obvious Patterns

  1. **No ORM or external tooling**: Storage and parsing rely on native
SQLite and tree-sitter fallbacks.
  2. **Immutable artifacts**: All data operations are append-only
(Ledger/Vault).
  3. **State safety rules:** State transitions only via explicit, auditable
steps.

  ## Security Considerations

  - **Sensitive credentials** (`llm_api_key`) should be provided in
`config/config.toml`.
  - Ensure encryption and access controls for Vault storage if used.

  ## Dependencies & Tooling

  - Go 1.22+ required.
  - SQLite with WAL mode is the persistence layer; no external DBMS or ORM.
