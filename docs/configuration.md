# TSLP v5.3 Configuration System

## Overview

The configuration system implements **strict validation** per the Gold Specification:
- **Fail fast** on missing or invalid configuration
- **No defaults** except where explicitly allowed
- **Immutable** after startup
- **JSON schema validation** on every load
- **No feature flags, no tuning parameters**

## Configuration File

Location: `config/config.toml`

All fields are **REQUIRED** unless explicitly noted.

### Fields (Per Gold Spec)

```toml
[database]
database_path = "./runtime/tslp.db"  # Required: Path to SQLite database

[logging]
log_path = "./runtime/tslp.log"      # Required: Path to log file
debug = false                         # Required: Enable debug logging (boolean)

[llm]
llm_provider = "openai"              # Required: Provider identifier
llm_endpoint = "https://api.openai.com/v1"  # Required: API endpoint (must start with http:// or https://)
llm_api_key = ""                     # Required: API key (can be empty for local models)
llm_model = "gpt-4"                  # Required: Model identifier

[proxy]
listen_address = "127.0.0.1:8080"    # Required: Proxy listen address (format: host:port)
```

## Validation Rules

### Startup Validation

1. **File Existence**: Config file must exist
2. **TOML Parsing**: Must be valid TOML syntax
3. **Required Fields**: All fields must be present
4. **Schema Validation**: Must pass JSON schema validation
5. **Format Validation**: Endpoint URLs and listen addresses must match required patterns

### Field-Specific Rules

- `database_path`: Non-empty string
- `log_path`: Non-empty string
- `debug`: Boolean (true/false)
- `llm_provider`: Non-empty string
- `llm_endpoint`: Must match pattern `^https?://`
- `llm_api_key`: String (can be empty)
- `llm_model`: Non-empty string
- `listen_address`: Must match pattern `^[0-9.]+:[0-9]+$`

### Prohibited

Per specification, the following are **NOT ALLOWED**:
- Additional fields beyond those specified
- Feature flags
- Tuning parameters
- Thresholds
- Default values (except config file path flag)

## Usage

### Load Configuration

```bash
# Use default location
./tslp

# Specify config file
./tslp -config /path/to/config.toml
```

### Validation Failure

If configuration is invalid, TSLP will:
1. Print error message to stderr
2. List all required fields
3. Exit with code 1

Example:
```
FATAL: Configuration error: llm.llm_endpoint must start with http:// or https://

Configuration must include all required fields:
  - database.database_path
  - logging.log_path
  - logging.debug
  ...
```

## Implementation

### Files

- `config/config.toml` - Example configuration file
- `config/config.schema.json` - JSON schema for validation
- `config/config.go` - Configuration loader and validator
- `cmd/tslp/main.go` - Loads config at startup

### Immutability

Configuration is loaded once at startup and passed by value to prevent modification. The Config struct is never modified after initial load and validation.

### No Defaults Function

Per specification, there is **NO** `Default()` function. All configuration must be explicit in the config file.

## Testing Configuration

### Valid Config Test

```bash
./tslp -config config/config.toml
```

Should output:
```
TSLP (Transactional State-Ledger Proxy) v5.3-gold
Per Specification v5.3 (Gold)

Loading configuration from: config/config.toml
✓ Configuration validated
...
```

### Invalid Config Test

Create a test file with missing fields:
```toml
[database]
database_path = "./test.db"
```

Running `./tslp -config test.toml` should fail with validation error.

## JSON Schema

The JSON schema (`config/config.schema.json`) enforces:
- Required fields
- No additional properties
- Type validation
- Format validation (URLs, addresses)

Schema is embedded in the binary and loaded at runtime for validation.

## Philosophy

**"Configuration is minimal and boring."**

If behavior needs to change, it changes in code, not in configuration. The configuration system exists only to specify:
- Where to store data (database_path, log_path)
- How to connect to LLM (provider, endpoint, key, model)
- Where to listen for requests (listen_address)
- Debug mode (debug)

Nothing more, nothing less.
