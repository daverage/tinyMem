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
-   `semantic_enabled` (boolean): Enable semantic search (requires embedding model). Default: `false`.

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
