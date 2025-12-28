# TSLP Logging and Startup Lifecycle

## Structured Logging

### Log Levels

TSLP implements four log levels:

- **INFO** - Informational messages (always logged)
- **WARN** - Warning messages (always logged)
- **ERROR** - Error messages (always logged)
- **DEBUG** - Debug messages (only when `config.debug = true`)

### Log Format

```
[LEVEL] YYYY/MM/DD HH:MM:SS.microseconds message
```

Example:
```
[INFO]  2025/12/24 10:33:35.879356 TSLP v5.3-gold starting
[DEBUG] 2025/12/24 10:34:00.552046 Initializing runtime components
[ERROR] 2025/12/24 10:35:12.123456 Failed to process artifact: error details
```

### Log Destination

**Production Mode** (`debug = false`):
- Logs written to `log_path` only
- No stdout output after logger initialization
- Clean separation of user-facing output and operational logs

**Debug Mode** (`debug = true`):
- Logs written to `log_path` only
- DEBUG level messages included
- No stdout output after logger initialization

### Controlling Debug Logging

Debug mode is controlled **exclusively** by the `logging.debug` configuration field.

```toml
[logging]
log_path = "./runtime/tslp.log"
debug = true  # Enable DEBUG level logs
```

There are no runtime controls, environment variables, or command-line flags that affect debug logging. This is intentional per specification.

## Startup Lifecycle

TSLP follows an **explicit, ordered startup sequence**. Each phase must complete successfully before the next begins.

### Startup Phases

```
Phase 1: Load Configuration
    ↓
Phase 2: Initialize Logger
    ↓
Phase 3: Open Database
    ↓
Phase 4: Run Migrations
    ↓
Phase 5: Start HTTP Server
    ↓
    READY
```

### Phase Details

#### Phase 1: Load Configuration

**Actions:**
- Parse `config.toml`
- Validate required fields
- Validate JSON schema
- Validate field formats

**Failure Behavior:**
- Print error to stderr
- List all required fields
- Exit with code 1

**Output:**
```
Phase 1/5: Loading configuration from config/config.toml
✓ Configuration validated
```

#### Phase 2: Initialize Logger

**Actions:**
- Create log directory if needed
- Open log file for append
- Initialize log levels (INFO, WARN, ERROR, DEBUG)

**Failure Behavior:**
- Print error to stderr
- Exit with code 1

**Output:**
```
Phase 2/5: Initializing logger (log_path=./runtime/tslp.log, debug=false)
✓ Logger initialized
```

**Log Entries:**
```
[INFO]  STARTUP_PHASE phase=1_config_loaded
[INFO]  TSLP v5.3-gold starting
[INFO]  Configuration loaded from: config/config.toml
[INFO]    Database: ./runtime/tslp.db
[INFO]    Log file: ./runtime/tslp.log
[INFO]    Debug mode: false
[INFO]    LLM Provider: openai
[INFO]    LLM Endpoint: https://api.openai.com/v1
[INFO]    LLM Model: gpt-4
[INFO]    Proxy Address: 127.0.0.1:8080
[INFO]  STARTUP_PHASE phase=2_logger_initialized
```

#### Phase 3: Open Database

**Actions:**
- Create database directory if needed
- Open SQLite database file
- Enable WAL mode
- Enable foreign keys

**Failure Behavior:**
- Log to file
- Print error to stderr
- Exit with code 1

**Output:**
```
Phase 3/5: Opening database at ./runtime/tslp.db
✓ Database opened
```

**Log Entries:**
```
[INFO]  STARTUP_PHASE phase=3_opening_database
[INFO]  Opening database: ./runtime/tslp.db
[INFO]  Database opened successfully
```

#### Phase 4: Run Migrations

**Actions:**
- Execute schema migrations
- Create tables if they don't exist
- Create indexes

**Failure Behavior:**
- Log to file
- Print error to stderr
- Exit with code 1

**Output:**
```
Phase 4/5: Running database migrations
✓ Migrations complete (WAL mode enabled)
```

**Log Entries:**
```
[INFO]  STARTUP_PHASE phase=4_running_migrations
[INFO]  Database migrations completed (WAL mode enabled)
```

#### Phase 5: Start HTTP Server

**Actions:**
- Initialize runtime components
- Initialize hydration engine
- Initialize LLM client
- Initialize shadow auditor
- Start HTTP server on configured address

**Failure Behavior:**
- Log to file
- Print error to stderr
- Exit with code 1

**Output:**
```
Phase 5/5: Starting HTTP server
✓ HTTP server started

========================================
TSLP Ready
========================================
```

**Log Entries:**
```
[INFO]  STARTUP_PHASE phase=5_starting_server
[DEBUG] Initializing runtime components
[DEBUG] Initializing hydration engine
[DEBUG] Initializing LLM client
[DEBUG] Initializing shadow auditor
[DEBUG] Initializing API server on 127.0.0.1:8080
[INFO]  HTTP server listening on 127.0.0.1:8080
[INFO]  STARTUP_COMPLETE listen_addr=127.0.0.1:8080
```

### Startup Success

When all phases complete successfully:

```
========================================
TSLP Ready
========================================

Core Principles:
  • The LLM is stateless
  • The Proxy is authoritative
  • State advances only by structural proof
  • Nothing is overwritten without acknowledgement
  • Continuity is structural, not linguistic
  • Truth is materialized, never inferred

Endpoint: http://127.0.0.1:8080/v1/chat/completions
Log file: ./runtime/tslp.log

Press Ctrl+C to shutdown
```

## Shutdown Lifecycle

### Graceful Shutdown

Triggered by: `SIGINT` (Ctrl+C) or `SIGTERM`

**Sequence:**
1. Receive shutdown signal
2. Log shutdown initiation
3. Stop accepting new requests
4. Complete in-flight requests (30s timeout)
5. Close HTTP server
6. Close database connection
7. Close log file
8. Exit

**Output:**
```
Received signal: interrupt
Initiating graceful shutdown...
✓ Shutdown complete
```

**Log Entries:**
```
[INFO]  SHUTDOWN_INITIATED reason="signal=interrupt"
[INFO]  Shutting down proxy server
[INFO]  SHUTDOWN_COMPLETE
```

### Error Shutdown

If a fatal error occurs during runtime:

**Log Entries:**
```
[ERROR] FATAL: Server error: <error details>
```

**Exit:** Code 1

## Domain-Specific Log Events

### State Transitions

```
[INFO]  STATE_TRANSITION episode=<uuid> entity=<filepath::symbol> from=<state> to=<state> reason="<reason>"
```

### Artifact Storage

```
[DEBUG] ARTIFACT_STORED hash=<sha256> type=<type> size=<bytes>
```

### Entity Resolution

```
[DEBUG] ENTITY_RESOLVED artifact=<hash> entity=<key> confidence=<level> method=<method>
```

### Promotion Evaluation

```
[INFO]  PROMOTION_EVAL artifact=<hash> entity=<key> promoted=<bool> reason="<reason>"
```

### Episode Creation

```
[DEBUG] EPISODE_CREATED episode_id=<uuid>
```

### Hydration

```
[DEBUG] HYDRATION_START entity_count=<count>
```

### Shadow Audit

```
[DEBUG] AUDIT_STARTED episode=<uuid> artifact=<hash>
[INFO]  AUDIT_COMPLETED episode=<uuid> artifact=<hash> status=<status>
```

### Proxy Requests

```
[DEBUG] PROXY_REQUEST method=<method> path=<path>
```

## Log File Management

### Location

Configured via `logging.log_path` in `config.toml`.

Default: `./runtime/tslp.log`

### Rotation

TSLP does **not** implement log rotation internally. Use external tools:

**logrotate (Linux):**
```
/path/to/tslp.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    postrotate
        pkill -HUP tslp
    endscript
}
```

**newsyslog (macOS/BSD):**
```
/path/to/tslp.log    644  7    *    @T00  J
```

### File Format

- Plain text
- One log entry per line
- Chronological order
- Microsecond precision timestamps
- No log aggregation or batching

## Philosophy

**"Logs are evidence, not memory."**

TSLP logging follows these principles:

1. **Deterministic** - Same input always produces same log output
2. **Explicit** - Every significant action is logged
3. **Inspectable** - Logs are plain text, grep-friendly
4. **Structured** - Consistent format for parsing
5. **Minimal** - No verbose or redundant logging
6. **Immutable** - Logs are append-only

Logs serve as an audit trail for debugging and compliance, not as a substitute for proper error handling or user feedback.
