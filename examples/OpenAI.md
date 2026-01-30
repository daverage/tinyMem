# tinyMem OpenAI & Compatible Clients Guide

This guide covers using tinyMem with the official OpenAI Python/Node.js SDKs, as well as any tool that accepts an `OPENAI_BASE_URL`.

## How It Works

tinyMem acts as a **transparent proxy**. You change your client's `base_url` to point to tinyMem (default `http://localhost:8080/v1`).
1.  tinyMem receives the user prompt.
2.  It performs a semantic/keyword search in the local project memory.
3.  It injects relevant memories into the system prompt.
4.  It forwards the enriched request to the *actual* LLM provider (OpenAI, Ollama, LM Studio, etc.) defined in your config.

---

## 1. OpenAI Python SDK

### Installation
```bash
pip install openai
```

### Usage

```python
from openai import OpenAI

# 1. Point to tinyMem Proxy
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="dummy" # tinyMem handles the real auth with the backend
)

# 2. Chat as normal
response = client.chat.completions.create(
    model="gpt-4o", # Model name must be valid for the BACKEND
    messages=[
        {"role": "user", "content": "What is the deployment process for this app?"}
    ]
)

print(response.choices[0].message.content)
```

**Verification:**
Check the tinyMem proxy logs. You should see:
`[Recall] Found 3 memories for query 'deployment process'`

---

## 2. OpenAI Node.js SDK

### Installation
```bash
npm install openai
```

### Usage

```javascript
import OpenAI from 'openai';

const openai = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: 'dummy',
});

async function main() {
  const completion = await openai.chat.completions.create({
    messages: [{ role: 'user', content: 'What is the deployment process?' }],
    model: 'gpt-4o',
  });

  console.log(completion.choices[0].message.content);
}

main();
```

---

## 3. Generic Environment Variables

Many CLI tools (like `fabric`, `interpreter`, etc.) respect standard environment variables.

```bash
export OPENAI_API_BASE="http://localhost:8080/v1"
export OPENAI_BASE_URL="http://localhost:8080/v1" # Some tools use this variant
export OPENAI_API_KEY="dummy"

# Run your tool
my-ai-tool "Plan the next sprint based on recent decisions"
```

---

## 4. Response Headers

tinyMem injects headers into the response to let you know what happened:

-   `X-TinyMem-Recall-Count`: Number of memories found and injected.
-   `X-TinyMem-Recall-Status`: `injected`, `none`, or `failed`.
-   `X-TinyMem-Version`: Version of tinyMem serving the request.

---

## Configuration Reference

For full configuration options, see [Configuration.md](Configuration.md).

## Troubleshooting

-   **Authentication Errors:** If using a real OpenAI backend, ensure the API key is set in `.tinyMem/config.toml` or via `TINYMEM_LLM_API_KEY` env var. The client's `api_key` is ignored by tinyMem but required by SDKs.
-   **Model Not Found:** Ensure the model name you request matches what your backend supports.