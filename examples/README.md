# tinyMem Example Configurations

This folder contains example configuration files for connecting `tinyMem` to various local LLM providers and IDEs.

## Directory Structure

-   `proxy/`: Contains `config.toml` examples for using `tinyMem` in proxy mode. In this mode, `tinyMem` sits between your LLM client and your local LLM server.
-   `mcp/`: Contains examples for integrating `tinyMem` with IDEs that support the "Meta-Code Protocol" (MCP), such as Claude Desktop (formerly Cursor) and Zed.
-   `cli-clients/`: Contains examples and instructions for configuring various command-line interface (CLI) tools to use `tinyMem`'s proxy.

## How to Use These Examples

1.  **Choose your setup:** Find the folder that matches your LLM provider (e.g., `ollama`, `lmstudio`) and your integration method (`proxy` or `mcp`).
2.  **Copy the configuration:** Copy the example configuration file to the correct location.
    -   For `config.toml` files, you should place them in a `.tinyMem` directory inside your project folder (i.e., `your-project/.tinyMem/config.toml`). `tinyMem` will automatically load this file if it exists.
    -   For IDE configuration files (like `claude_desktop_config.json` or VS Code's `settings.json`), you'll need to merge the contents with your existing IDE settings file. Please refer to your IDE's documentation for the exact location of its configuration file.
3.  **Adjust the settings:** The provided examples use default ports and settings. If you have configured your LLM provider to use a different port or URL, you will need to update the `base_url` in the `config.toml` file accordingly.

For more detailed information, please refer to the main `README.md` file in the root of this repository.
