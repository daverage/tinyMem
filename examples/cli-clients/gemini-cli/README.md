# Configuring Gemini API Clients with tinyMem

This guide explains how to configure various Gemini API clients to use `tinyMem` as a proxy. `tinyMem` acts as an HTTP proxy, enabling it to inject context and manage memory for your AI interactions.

Before you begin, ensure `tinyMem` is running in proxy mode (e.g., `tinymem proxy`) and listening on `http://localhost:8080` (or your configured port).

## 1. Google Cloud SDK (`gcloud CLI`)

The `gcloud CLI` (which includes tools for interacting with Google's Gemini API) can be configured to use a proxy.

### Using Environment Variables (Recommended for simplicity)

You can set standard proxy environment variables before running `gcloud` commands. This is often the simplest method if your CLI supports it.

```bash
# Set the HTTP_PROXY and HTTPS_PROXY environment variables
export HTTP_PROXY="http://localhost:8080"
export HTTPS_PROXY="http://localhost:8080"

# (Optional) If you have hosts that should bypass the proxy, use NO_PROXY
# export NO_PROXY="localhost,127.0.0.1"

# Now, any gcloud commands will route through tinyMem
gcloud genai models list
```

The Google Cloud SDK docs (November 2025) describe those same proxy commands and explicitly state that values persisted via `gcloud config set proxy/*` override the HTTP_PROXY/HTTPS_PROXY environment variables, so the CLI will keep routing through tinyMem even if `NO_PROXY` is present.

### Using `gcloud config` commands

You can also configure the proxy settings directly within `gcloud`'s configuration.

```bash
# Set the proxy type to http
gcloud config set proxy/type http

# Set the proxy address to tinyMem's listening address
gcloud config set proxy/address localhost

# Set the proxy port to tinyMem's listening port
gcloud config set proxy/port 8080

# To verify your settings:
gcloud config list
```
**Note:** `gcloud config` settings will override environment variables.

## 2. Python Client Library (`google-generativeai`)

If you're using the `google-generativeai` Python library, you can configure the proxy by passing `HttpOptions` to the client.

```python
from google import generativeai as genai
from google.generativeai.types import HttpOptions
import os

# Ensure tinyMem is running on port 8080
PROXY_ADDRESS = "http://localhost:8080"

# Method 1: Using HttpOptions for direct client configuration
# (This method explicitly configures the client's HTTPX instance)
try:
    http_options = HttpOptions(client_args={"proxy": PROXY_ADDRESS})
    client = genai.Client(http_options=http_options)
    print("Gemini client configured with explicit proxy.")

    # Example usage:
    # for m in client.list_models():
    #     print(m.name)

except Exception as e:
    print(f"Error configuring client with explicit proxy: {e}")

# Method 2: Using standard environment variables (if your httpx version picks them up)
# This often works if you don't explicitly configure http_options.
# Ensure these are set BEFORE your script runs, or uncomment the lines below:
# os.environ['HTTP_PROXY'] = PROXY_ADDRESS
# os.environ['HTTPS_PROXY'] = PROXY_ADDRESS
#
# client_env_proxy = genai.Client()
# print("Gemini client configured via environment variables (if supported).")

# Example usage with either client
# model = client.GenerativeModel('gemini-pro')
# response = model.generate_content("Tell me a story about a magical cat.")
# print(response.text)
```
The Google Generative AI client docs (October 2025) describe passing `HttpOptions(client_args={"proxy": ...})` to expose a custom HTTPX proxy while also respecting the standard `HTTP_PROXY`/`HTTPS_PROXY` environment variables if you don't configure `HttpOptions`.

Choose the method that best suits your setup. The `HttpOptions` method is generally more explicit and reliable for programmatic configuration.

## 3. MCP Integration (for supported Gemini CLI versions)

If you are using a Gemini CLI that supports MCP (Meta-Code Protocol), you can configure it to use `tinyMem` as an MCP server. This is typically done in a `settings.json` file used by the Gemini CLI.

**Example `settings.json`:**
```json
{
  "mcpServers": {
    "tinymem": {
      "command": "tinymem",
      "args": ["mcp"],
      "env": {
        "TINYMEM_LOG_LEVEL": "info"
      }
    }
  }
}
```
**How to use:**
1.  Locate the `settings.json` file for your Gemini CLI.
2.  Add the `tinymem` server configuration to the `mcpServers` object.
3.  Ensure `tinymem` is in your system's PATH, or provide the absolute path to the executable in the `command` field.
4.  Once configured, the Gemini CLI will be able to access `tinyMem`'s memory tools (e.g., `memory_query`, `memory_write`).

The Gemini CLI configuration guide (late 2025) describes exactly this `mcpServers` structure within `.gemini/settings.json`, so copying the above snippet matches the official schema.
