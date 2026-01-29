# Quick Start Guide

**Get your AI to remember your project context in 5 minutes.**

tinyMem is a local tool that sits between your code and your AI assistant. It creates a "memory brain" for your specific project so you don't have to keep repeating context.

---

## 1. Install tinyMem

First, get the single executable file. No complex installers or dependencies required.

### Windows
1.  Download the **[latest release](https://github.com/andrzejmarczewski/tinyMem/releases)** (`tinymem-windows-amd64.exe`).
2.  Create a folder `C:\Tools` (or use an existing one) and put the file there.
3.  Rename it to `tinymem.exe` for convenience.
4.  **Important:** Add this folder to your PATH so you can run it from anywhere.
    *   *Search "Edit the system environment variables" > "Environment Variables" > Select "Path" in User variables > "Edit" > "New" > Paste `C:\Tools` > OK > OK.*

### macOS / Linux
1.  Open your terminal.
2.  Run this command to download and install to `/usr/local/bin`:
    ```bash
    curl -L "https://github.com/andrzejmarczewski/tinyMem/releases/latest/download/tinymem-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)" -o tinymem
    chmod +x tinymem
    sudo mv tinymem /usr/local/bin/
    ```

---

## 2. Initialize Your Project

tinyMem creates a memory database *inside* your project folder (in a hidden `.tinyMem` directory). You need to tell it which project to manage.

1.  Open your terminal.
2.  Navigate to your project's root folder:
    ```bash
    cd /path/to/my-cool-app
    ```
3.  Initialize the memory system:
    ```bash
    tinymem health
    ```
    *(You should see "✅ System is healthy" and a new `.tinyMem` folder created).*

---

## 3. Connect Your AI

Choose the method that matches how you work.

### Option A: IDE Integration (Claude Desktop, Cursor, VS Code)
*Best for: Coding assistants and chat interfaces.*

You need to tell your IDE to run tinyMem as a "Model Context Protocol" (MCP) server.

**For Claude Desktop:**
Edit your config file (usually `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS or `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```json
{
  "mcpServers": {
    "tinymem": {
      "command": "tinymem",
      "args": ["mcp"]
    }
  }
}
```
*Restart Claude Desktop. The 🔌 icon should appear, indicating tinyMem is connected.*

### Option B: Proxy Mode (Scripts & API Clients)
*Best for: Running Python scripts, Aider, generic OpenAI clients, or terminal tools.*

1.  **Configure for Local LLMs (Optional):**
    If you use LM Studio or Ollama, create a file at `.tinyMem/config.toml`:
    ```toml
    [proxy]
    base_url = "http://localhost:1234/v1" # Point to LM Studio
    ```

2.  Start the proxy in a separate terminal window:
    ```bash
    cd /path/to/my-cool-app
    tinymem proxy
    ```
3.  In your main terminal, set the environment variable to route requests through tinyMem:
    ```bash
    export OPENAI_API_BASE_URL=http://localhost:8080/v1
    ```
    *For Aider: `aider --openai-api-base http://localhost:8080/v1 --model openai/qwen2.5-coder-7b-instruct`*

4.  Run your tool or script as usual. tinyMem will transparently intercept and inject memory.

---

## 4. Verify It's Working

1.  Ask your AI something about your project.
2.  Check the tinyMem status:
    ```bash
    tinymem stats
    ```
3.  See your memories visually:
    ```bash
    tinymem dashboard
    ```

---

## Next Steps

*   **Read the full [README](../README.md)** for advanced configuration.
*   **Learn about Memory Types** to understand the difference between a `fact` (verified) and a `claim` (unverified).
*   **Check `tinymem doctor`** if you run into any issues.