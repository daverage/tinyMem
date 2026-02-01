#!/bin/bash
# Example: Add tinyMem to Claude Code MCP servers

# Method 1: Simple (recommended)
claude mcp add tinymem -- tinymem mcp

# Method 2: With environment variables
claude mcp add tinymem \
  --env TINYMEM_LOG_LEVEL=info \
  --env TINYMEM_METRICS_ENABLED=true \
  -- tinymem mcp

# Verify it's added
claude mcp list

# Test it
echo "Now start Claude Code and try:"
echo '  claude'
echo '  > "Use memory_health to check tinyMem status"'
