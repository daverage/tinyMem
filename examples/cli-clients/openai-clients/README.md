# Configuring OpenAI-Compatible Clients with tinyMem

tinyMem's `proxy` mode exposes an OpenAI-compatible API at:

`http://localhost:8080/v1`

Use this when your client supports a custom OpenAI base URL.

## Example: OpenAI Python client

```bash
export OPENAI_BASE_URL="http://localhost:8080/v1"
export OPENAI_API_KEY="dummy"
```

Then run your client as usual.

Notes:
- The `model` you request must be supported by the backend configured in `.tinyMem/config.toml`.
- If your backend requires an API key, configure it in `.tinyMem/config.toml` under `[llm]`.
