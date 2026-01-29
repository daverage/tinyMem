# Continue (VS Code) Example

Continue uses its own config file (not VS Code `settings.json`) to define models/providers.

This example shows how to point an OpenAI-compatible provider at tinyMem's proxy:
- `apiBase`: `http://localhost:8080/v1`
- `apiKey`: can be a dummy value if your downstream backend does not require it

How to use:
1. Ensure `tinymem proxy` is running.
2. Merge `config.json` into your Continue config file.
