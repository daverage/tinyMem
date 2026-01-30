# tinyMem + GitHub Copilot Integration

GitHub Copilot Chat can be configured to use tinyMem as a proxy, giving it access to persistent project memory.

## Prerequisites

-   **VS Code** installed
-   **GitHub Copilot Chat** extension installed
-   **tinyMem** installed and running in proxy mode

## Configuration

1.  **Start tinyMem Proxy:**
    ```bash
    tinymem proxy
    ```
    *Ensure your backend LLM (e.g., OpenAI, Ollama) is configured in `.tinyMem/config.toml`.*

2.  **Configure VS Code Settings:**
    Open your `settings.json` (Command Palette -> `Preferences: Open User Settings (JSON)`) and add:

    ```json
    {
      "github.copilot.advanced": {
        "debug.overrideProxyUrl": "http://localhost:8080"
      }
    }
    ```

    *Note: The exact setting name for overriding the base URL may vary by extension version. Check the latest Copilot documentation for "custom OpenAI base URL" or "proxy support". Some versions might require an experimental flag.*

## Usage

Use Copilot Chat normally:
> "What are the key architectural decisions in this project?"

Copilot will route the request through tinyMem, which injects relevant context before forwarding to the underlying model.

## Troubleshooting

-   **"Connection Refused"**: Ensure `tinymem proxy` is running.
-   **No Memory**: Check tinyMem logs to see if requests are hitting the proxy.
