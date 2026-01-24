# MCP Connection Fix - Changelog

## Issue
MCP clients (Claude Desktop, Cursor, etc.) were experiencing connection errors:
- "Request timed out" (MCP error -32001)
- "Client is not connected, must connect before interacting with the server. Current state is disconnected"

## Root Cause
The logger was writing to both stderr and log files. When using MCP over stdio protocol, any output to stderr/stdout other than JSON-RPC messages causes protocol violations and connection failures.

## Changes Made

### 1. Fixed Logging System (internal/logging/logger.go:34-59)
- Added `NewLoggerWithStderr()` function with `includeStderr` parameter
- When `includeStderr=false`, logs only go to file
- When `includeStderr=true`, logs go to both stderr and file (default behavior)
- MCP mode uses `includeStderr=false` to keep stdio clean

### 2. Updated MCP Command (cmd/tinymem/main.go:94-134)
- Changed to use `NewLoggerWithStderr(cfg, false)` for silent logging
- Updated error handling to use stderr directly for fatal pre-protocol errors
- Added comments explaining stdio protocol requirements

### 3. Fixed MCP Protocol Implementation (internal/server/mcp/server.go)
- Changed method names to match MCP spec:
  - `call/tool` → `tools/call`
  - Added `tools/list` endpoint
  - Added `prompts/list` endpoint
- Changed tool names from dot notation to underscore:
  - `memory.query` → `memory_query`
  - `memory.recent` → `memory_recent`
  - etc.
- Fixed response format:
  - Removed HTTP-style Content-Length headers
  - Changed to plain JSON-RPC over stdio (one message per line)
  - Tool responses now return `content` arrays with text
- Added `sendResponse()` helper to centralize response handling
- Updated initialize response to include proper protocolVersion and serverInfo

### 4. Created Documentation
- **README.md**: Updated with MCP troubleshooting section and verification script reference
- **MCP_TROUBLESHOOTING.md**: Comprehensive troubleshooting guide covering:
  - Common errors and solutions
  - Debugging steps
  - Configuration examples
  - Protocol details
- **verify_mcp.sh**: Automated verification script that:
  - Checks binary exists and is executable
  - Tests database initialization
  - Verifies MCP protocol works
  - Generates ready-to-use configuration

## Testing
All changes verified with:
1. Manual MCP protocol tests (initialize, tools/list, tools/call)
2. Python-based integration tests
3. Verification that stderr remains clean
4. Confirmation that logs still write to file

## Verification
Run the verification script to confirm the fix works:
```bash
./verify_mcp.sh
```

Expected output:
- All checks pass (✅)
- Clean JSON-RPC responses
- Configuration ready to copy
- Logs written to file only

## Backward Compatibility
- All existing functionality preserved
- Non-MCP commands (proxy, health, stats, etc.) unchanged
- Default logging behavior (with stderr) unchanged
- Only MCP mode uses silent logging

## Files Changed
1. `internal/logging/logger.go` - Added silent logging option
2. `cmd/tinymem/main.go` - Use silent logging for MCP
3. `internal/server/mcp/server.go` - Fix protocol compliance
4. `README.md` - Updated MCP documentation
5. `MCP_TROUBLESHOOTING.md` - New comprehensive guide
6. `verify_mcp.sh` - New verification script

## Next Steps for Users
1. Rebuild: `go build -o tinymem ./cmd/tinymem`
2. Verify: `./verify_mcp.sh`
3. Copy configuration from verification output
4. Update IDE MCP config
5. Restart IDE
6. Confirm tinyMem tools appear in IDE

## Technical Details

### MCP Protocol Requirements
- Transport: JSON-RPC 2.0 over stdio
- Format: One message per line
- Critical requirement: No non-JSON output on stdout
- Initialization sequence:
  1. Client sends `initialize`
  2. Server responds with capabilities
  3. Client sends `tools/list`
  4. Server responds with tool definitions
  5. Client calls tools as needed

### Why Logging Caused Issues
```
# Before (broken):
stderr: [INFO] Starting MCP server
stdout: {"jsonrpc":"2.0",...}

# After (fixed):
file: [INFO] Starting MCP server
stdout: {"jsonrpc":"2.0",...}
```

MCP clients parse stdout line-by-line expecting only JSON. Any non-JSON output causes parsing errors and immediate disconnection.

### Silent Logging Implementation
```go
// Before
logger.logger = log.New(io.MultiWriter(os.Stderr, file), ...)

// After (MCP mode)
logger.logger = log.New(file, ...)  // File only
```

## Version
Fixed in tinyMem v0.1.0 (2026-01-24)
