#!/bin/bash
echo "Testing MCP config loading..."
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | tinymem mcp 2>&1 | grep -A 2 "CoVe enabled"
