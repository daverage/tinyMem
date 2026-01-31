# Tasks – Pre-Release Fixes for v0.3.0

- [x] Fix goroutine lifecycle in proxy server
  - [x] Add processorDone channel to ProxyServer struct
  - [x] Signal completion when processResponseCaptures exits
  - [x] Wait for goroutine completion in Stop()
- [x] Extract duplicate service initialization code
  - [x] Create initializeRecallServices helper in app/modules.go
  - [x] Replace duplicated code in proxy/server.go
  - [x] Replace duplicated code in mcp/server.go
- [x] Fix channel drain on shutdown
  - [x] Update processResponseCaptures to drain buffer after shutdown signal
  - [x] Ensure no buffered items lost on shutdown
- [x] Update Cline.md documentation
  - [x] Replace manual path instructions with UI-based method
  - [x] Add reference to official Cline docs
  - [x] Test instructions for accuracy
