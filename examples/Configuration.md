# tinyMem Configuration Reference

This guide lists all available configuration options for `.tinyMem/config.toml`.

## Core Structure

A standard configuration file looks like this:

```toml
[proxy]
port = 8080
base_url = "http://localhost:11434/v1"

[llm]
model = "llama3"
timeout = 120

[recall]
max_items = 10
semantic_enabled = false

[cove]
enabled = true
confidence_threshold = 0.6

[logging]
level = "info"
file = "tinymem.log"
```

## Section Details

### `[proxy]`
Settings for the HTTP proxy server.
-   `port` (integer): The port tinyMem listens on. Default: `8080`.
-   `base_url` (string): The upstream LLM provider's base URL (e.g., `http://localhost:11434/v1`).

### `[llm]`
Settings for the internal LLM client (used for CoVe, summarization, and proxying).
-   `model` (string): The model identifier to send to the backend.
-   `timeout` (integer): Request timeout in seconds. Default: `120`.
-   `api_key_env` (string): Name of the environment variable containing the API key.
-   `base_url` (string): Override backend URL specifically for internal tasks (defaults to `[proxy].base_url`).

### `[recall]`
Settings for memory retrieval.
-   `max_items` (integer): Maximum number of memories to retrieve per query. Default: `10`.
-   `max_tokens` (integer): Maximum total tokens for injected context. Default: `2000`.
-   `semantic_enabled` (boolean): Enable semantic search. Full builds use embedded model, lightweight builds require HTTP endpoint. Default: `false`.
-   `hybrid_weight` (float): Weight for semantic vs lexical search (0.0 = all lexical, 1.0 = all semantic). Default: `0.5`.

### `[embedding]`
Settings for text embeddings used in semantic search. See [docs/EMBEDDINGS.md](../docs/EMBEDDINGS.md) for details.

**Full builds** (built with `-tags embeddings`):
- No configuration needed - uses embedded model automatically
- `base_url` and `model` are optional (only used as HTTP fallback if local model fails)

**Lightweight builds** (built without embeddings tag):
- `base_url` (string): **Required** - URL of embedding service (e.g., `http://localhost:11434`)
- `model` (string): Model name to use for embeddings. Default: `nomic-embed-text`

**Example (Full Build - No Config Needed):**
```toml
[recall]
semantic_enabled = true
# That's it! Embedded model is used automatically.
```

**Example (Lightweight Build - HTTP Required):**
```toml
[recall]
semantic_enabled = true

[embedding]
base_url = "http://localhost:11434"  # Ollama or other embedding service
model = "nomic-embed-text"
```

**Example (Full Build with HTTP Override):**
```toml
[recall]
semantic_enabled = true

[embedding]
# Force HTTP mode even with full build (e.g., to use different model)
base_url = "http://localhost:11434"
model = "mxbai-embed-large"
```

### `[cove]`
Chain-of-Verification settings for truth validation.
-   `enabled` (boolean): Enable CoVe. Default: `false`.
-   `confidence_threshold` (float): Minimum confidence (0.0 - 1.0) to accept a fact. Default: `0.6`.
-   `max_candidates` (integer): Number of candidate facts to generate/check. Default: `3`.

### `[logging]`
-   `level` (string): Log verbosity. Options: `debug`, `info`, `warn`, `error`. Default: `info`.
-   `file` (string): Log file path relative to `.tinyMem/`. Default: `tinymem.log`.

### `[memory_ralph]`
Settings for the Autonomous Repair Loop.
-   `max_iterations` (integer): Maximum repair attempts before giving up. Default: `5`.
-   `allow_shell` (boolean): Allow execution of shell commands. Default: `false`.
-   `forbid_paths` (array of strings): Paths that Ralph must never modify.

### `[database]`
Database maintenance and retention policy settings.
-   `auto_maintenance` (boolean): Enable automatic database maintenance (PRAGMA optimize + incremental_vacuum). Default: `true`.
-   `maintenance_interval_hours` (integer): How often to run maintenance (in hours). Default: `24` (daily).

**Example:**
```toml
[database]
auto_maintenance = true
maintenance_interval_hours = 24  # Run daily

[database.retention]
max_age_days = 90  # Delete memories older than 90 days
max_count = 10000  # Keep only 10,000 most recent memories
exclude_types = ["fact", "decision"]  # Never delete facts or decisions
```

### `[database.retention]`
Retention policy for automatic memory cleanup.
-   `max_age_days` (integer): Delete memories older than this many days. Default: `0` (unlimited).
-   `max_count` (integer): Keep only the N most recent memories. Default: `0` (unlimited).
-   `exclude_types` (array of strings): Memory types to never delete. Default: `["fact"]`.

**Retention Policy Behavior:**
- Age-based and count-based retention can be used together
- Both policies respect the `exclude_types` list
- Retention runs on startup and periodically (if auto_maintenance is enabled)
- Use `max_age_days = 0` and `max_count = 0` for unlimited retention
- Excluded types are always preserved regardless of age or count

**Example Use Cases:**

*Long-term project (keep everything, just maintain):*
```toml
[database]
auto_maintenance = true
maintenance_interval_hours = 168  # Weekly
```

*Short-lived sessions (aggressive cleanup):*
```toml
[database]
auto_maintenance = true
maintenance_interval_hours = 1  # Hourly

[database.retention]
max_age_days = 7  # One week
exclude_types = ["fact", "decision"]
```

*Size-limited (keep only recent memories):*
```toml
[database]
auto_maintenance = true

[database.retention]
max_count = 5000  # Keep 5,000 most recent
exclude_types = ["fact"]
```
