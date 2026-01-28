# tinyMem Quick Start Guide (for Beginners)

Welcome to tinyMem! This guide walks you through the most straightforward way to get tinyMem running in your project, with clear steps, minimal jargon, and pointers to the deeper docs when you are ready for integrations.

## Why tinyMem first?
Language models forget context quickly and can hallucinate facts without checking the codebase. tinyMem solves this by keeping a local, project-scoped repository of classified, evidence-backed memories. That means:

- your assistant can remember decisions, constraints, and facts without re-reading every file;
- every claim is backed by verifiable evidence before it becomes a fact;
- everything runs locally, so you stay in control of your data and tooling.

This guide helps you reach the point where tinyMem is running and ready to connect to your AI client. For more detail on integrations, recall discipline, and CLI usage, see the full [README](README.md).

## Before you begin

1.  **Pick the right binary** for your machine (64-bit Windows, macOS Intel/ARM, or Linux) from the [releases page](https://github.com/andrzejmarczewski/tinyMem/releases).
2.  **Keep a terminal handy** (PowerShell, Command Prompt, or Terminal.app) and stay in the folder where you dropped the binary.
3.  **Plan to keep the tinyMem process running** while you use your AI assistant; closing it stops the proxy/ MCP server.

## Step 1: Download tinyMem

### Windows

1.  Visit the [Releases page](https://github.com/andrzejmarczewski/tinyMem/releases).
2.  Download `tinymem-windows-amd64.exe` (or the architecture that matches your machine).
3.  Create a folder (e.g., `C:\tinyMem`) and move the executable there.
4.  Rename it to `tinymem.exe` so commands look cleaner.

### macOS

1.  Visit the [Releases page](https://github.com/andrzejmarczewski/tinyMem/releases).
2.  If you have Apple Silicon (M1, M2, M3), download `tinymem-darwin-arm64`. Otherwise, choose `tinymem-darwin-amd64`.
3.  Move the file to a folder you can access easily (Desktop, Documents, or a dedicated tooling folder).
4.  Rename it to `tinymem` and run `chmod +x tinymem` in Terminal to make it executable.

## Step 2: Start tinyMem

### Windows

1.  Open Command Prompt, then `cd` into the folder where `tinymem.exe` lives.
2.  Run:

    ```cmd
    tinymem.exe proxy
    ```

    or, if you plan to use MCP agents:

    ```cmd
    tinymem.exe mcp
    ```

3.  Leave the window open. tinyMem keeps listening until you close the terminal.

### macOS (and Linux)

1.  Open Terminal and `cd` into the folder containing `tinymem`.
2.  Run:

    ```bash
    ./tinymem proxy
    ```

    or for MCP mode:

    ```bash
    ./tinymem mcp
    ```

3.  If macOS reports “unverified developer,” open **System Settings > Privacy & Security** and click “Open Anyway,” then rerun the command.

## Step 3: Connect your AI tool

1.  For HTTP clients (Curl, SDKs, VS Code extensions), point the base URL at `http://localhost:8080/v1` so tinyMem can intercept requests.
2.  For MCP-aware IDEs (Claude Desktop, Cursor, Qwen, Gemini), configure the MCP server to run `tinymem mcp` from the folder where the executable lives.
3.  Use the `tinymem health`, `tinymem stats`, and `tinymem dashboard` commands to confirm the service is healthy and observing your requests.

## Need help?

- The main [README](README.md) has sections on IDE integration, Memory types, and Chain-of-Verification (CoVe).  
- `tinymem health` shows initialization problems.  
- `tinymem doctor` runs diagnostics if something misbehaves.  
- `AGENT_CONTRACT.md`, `claude.md`, `GEMINI.md`, and `QWEN.md` explain how to write agent prompts if you are embedding tinyMem into an assistant.

## Optional cleanup

When you finish experimenting, press `Ctrl+C` (or `⌘+C`) in the terminal to stop tinyMem. Your `.tinyMem/` folder will keep any collected memories in case you want to resume later.

Happy coding with a memory-aware assistant!
