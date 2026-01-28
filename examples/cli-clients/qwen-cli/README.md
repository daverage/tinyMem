# Configuring Qwen-Code CLI with tinyMem

This guide explains how to configure the Qwen-Code CLI to use `tinyMem` as a proxy. `tinyMem` acts as an HTTP proxy, enabling it to inject context and manage memory for your AI interactions with Qwen models.

Before you begin, ensure `tinyMem` is running in proxy mode (e.g., `tinymem proxy`) and listening on `http://localhost:8080` (or your configured port).

## Using Environment Variables (Recommended)

The Qwen-Code CLI respects standard proxy environment variables. The simplest way to direct its traffic through `tinyMem` is to set `HTTP_PROXY` and `HTTPS_PROXY`.

```bash
# Set the HTTP_PROXY and HTTPS_PROXY environment variables
export HTTP_PROXY="http://localhost:8080"
export HTTPS_PROXY="http://localhost:8080"

# (Optional) If you have hosts that should bypass the proxy, use NO_PROXY
# export NO_PROXY="localhost,127.0.0.1"

# Now, when you run Qwen-Code CLI, its API requests will go through tinyMem
# Example:
# qwen-code chat

# To unset the proxy:
# unset HTTP_PROXY
# unset HTTPS_PROXY
# unset NO_PROXY
```
Ensure these environment variables are set in the terminal *before* you run `qwen-code`.

## Using the `--proxy` Command-Line Argument

The Qwen-Code CLI also supports a `--proxy` argument, which takes precedence over environment variables.

```bash
# Run Qwen-Code CLI with the --proxy argument
qwen-code chat --proxy http://localhost:8080
```
Replace `chat` with the specific Qwen-Code command you intend to use.

This matches the official Qwen Code configuration reference (last updated November 24, 2025), which documents `--proxy` as the flag for redirecting CLI traffic through an HTTP proxy. Because the CLI also honors `HTTP_PROXY`/`HTTPS_PROXY`, you can set those variables once per shell session instead of repeatedly supplying `--proxy`.

> A current issue (Qwen Code #756, opened October 2, 2025) confirms that `NO_PROXY` is ignored when the CLI reads proxy environment variables, so the only reliable way to bypass tinyMem for specific hosts is to temporarily unset the proxy variables before running the CLI for those hosts.

## Considerations

*   **`NO_PROXY` Issues:** There have been reported issues where the Qwen-Code CLI might not correctly honor the `NO_PROXY` environment variable. If you experience unexpected routing, you might need to explicitly unset proxy environment variables for commands that should bypass `tinyMem` entirely.
*   **API Base URL:** If you're using `tinyMem` to proxy to a local LLM that Qwen-Code supports (e.g., a Qwen model served via Ollama), you might also need to configure the Qwen-Code CLI's API base URL if it's not picking it up from the proxy. Refer to Qwen-Code's specific documentation for this, but it often involves environment variables like `OPENAI_API_BASE_URL` if it's using an OpenAI-compatible interface.
