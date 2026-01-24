#!/bin/bash
# Debug wrapper for tinymem MCP server
# Logs all stdin/stdout to debug what Gemini CLI is sending

LOGFILE="/Users/andrzejmarczewski/Documents/GitHub/tinyMem/.tinyMem/mcp_debug.log"

echo "=== MCP Server Started at $(date) ===" >> "$LOGFILE"
echo "Working directory: $(pwd)" >> "$LOGFILE"
echo "Arguments: $@" >> "$LOGFILE"
echo "" >> "$LOGFILE"

# Create named pipes for logging
PIPE_IN="/tmp/tinymem_stdin_$$"
PIPE_OUT="/tmp/tinymem_stdout_$$"
mkfifo "$PIPE_IN" "$PIPE_OUT"

# Tee stdin to log and pipe
tee -a "$LOGFILE" < /dev/stdin > "$PIPE_IN" &
TEE_PID=$!

# Start actual tinymem process
/Users/andrzejmarczewski/Documents/GitHub/tinyMem/tinymem mcp < "$PIPE_IN" | tee -a "$LOGFILE"

# Cleanup
kill $TEE_PID 2>/dev/null
rm -f "$PIPE_IN" "$PIPE_OUT"
echo "=== MCP Server Ended at $(date) ===" >> "$LOGFILE"
echo "" >> "$LOGFILE"
