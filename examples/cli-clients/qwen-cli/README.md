# Qwen CLI Notes

tinyMem's `proxy` mode exposes an OpenAI-compatible API at `http://localhost:8080/v1`.

If your Qwen CLI supports setting an OpenAI-compatible API base URL, point it at tinyMem:
- API base: `http://localhost:8080/v1`
- API key: can be a dummy value if your downstream backend does not require it

If your Qwen tool does not support an OpenAI-compatible base URL (and only supports a native Qwen API), use tinyMem via MCP instead (if supported by the tool/IDE).
