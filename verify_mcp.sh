#!/bin/bash
# verify_mcp.sh - Verify tinyMem MCP server is ready for IDE integration

set -e

echo "🔍 tinyMem MCP Verification"
echo "=============================="
echo

# Check if tinymem binary exists
if [ ! -f "./tinymem" ]; then
    echo "❌ Error: ./tinymem not found in current directory"
    echo "   Run this script from the directory containing tinymem"
    exit 1
fi
echo "✅ tinymem binary found"

# Check if executable
if [ ! -x "./tinymem" ]; then
    echo "❌ Error: ./tinymem is not executable"
    echo "   Run: chmod +x ./tinymem"
    exit 1
fi
echo "✅ tinymem is executable"

# Get absolute path
TINYMEM_PATH=$(cd "$(dirname "./tinymem")" && pwd)/tinymem
echo "📍 Absolute path: $TINYMEM_PATH"
echo

# Check version
echo "📦 Version check..."
./tinymem version
echo

# Initialize database
echo "💾 Database initialization..."
./tinymem health
echo

# Test MCP protocol
echo "🔌 Testing MCP protocol..."

# Start MCP server in background and capture output
./tinymem mcp > /tmp/mcp_test_out.txt 2> /tmp/mcp_test_err.txt &
MCP_PID=$!

# Give it a moment to start
sleep 1

# Send initialize message
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"verify","version":"1.0"}},"id":1}' > /tmp/mcp_test_in.txt

# Send to MCP server via stdin (it's already running, so we need to use a different approach)
# Kill the background process
kill $MCP_PID 2>/dev/null || true
wait $MCP_PID 2>/dev/null || true

# Alternative: Test with pipe
RESPONSE=$(echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"verify","version":"1.0"}},"id":1}' | ./tinymem mcp 2>/dev/null | head -1 & sleep 2; kill $! 2>/dev/null; wait $! 2>/dev/null)

if [ -z "$RESPONSE" ]; then
    # Try python test if available
    if command -v python3 &> /dev/null; then
        echo "   Using Python for protocol test..."
        python3 << 'EOF'
import subprocess, json, sys, time
proc = subprocess.Popen(['./tinymem', 'mcp'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, bufsize=0)
try:
    req = {"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05"},"id":1}
    proc.stdin.write(json.dumps(req) + '\n')
    proc.stdin.flush()
    time.sleep(1)
    response = proc.stdout.readline()
    if response and '"jsonrpc":"2.0"' in response:
        print("✅ MCP server responds correctly")
        sys.exit(0)
    else:
        print("❌ Error: Invalid or no response")
        sys.exit(1)
finally:
    proc.terminate()
    proc.wait(timeout=2)
EOF
        if [ $? -ne 0 ]; then
            echo "   Check logs: tail .tinyMem/logs/tinymem-$(date +%Y-%m-%d).log"
            exit 1
        fi
    else
        echo "⚠️  Cannot test MCP protocol automatically (install python3 for full test)"
        echo "   Manual test: echo '{\"jsonrpc\":\"2.0\",\"method\":\"initialize\",\"params\":{},\"id\":1}' | ./tinymem mcp"
    fi
else
    if echo "$RESPONSE" | grep -q '"jsonrpc":"2.0"'; then
        echo "✅ MCP server responds correctly"
    else
        echo "❌ Error: Invalid MCP response"
        echo "   Got: $RESPONSE"
        exit 1
    fi
fi

# Cleanup
rm -f /tmp/mcp_test_*.txt

# Check logs directory
if [ -d ".tinyMem/logs" ]; then
    echo "✅ Logging configured"
    LATEST_LOG=$(ls -t .tinyMem/logs/*.log 2>/dev/null | head -1)
    if [ -n "$LATEST_LOG" ]; then
        echo "   Latest log: $LATEST_LOG"
    fi
else
    echo "⚠️  Warning: No logs directory"
fi

echo
echo "=============================="
echo "✅ All checks passed!"
echo
echo "Your MCP configuration:"
echo
echo '{'
echo '  "mcpServers": {'
echo '    "tinymem": {'
echo "      \"command\": \"$TINYMEM_PATH\","
echo '      "args": ["mcp"]'
echo '    }'
echo '  }'
echo '}'
echo
echo "Next steps:"
echo "1. Copy the configuration above"
echo "2. Add it to your IDE's MCP config file:"
echo "   - Claude Desktop: ~/Library/Application Support/Claude/claude_desktop_config.json"
echo "   - Cursor: Settings > MCP"
echo "3. Restart your IDE"
echo "4. Look for 'tinymem' in available MCP servers"
echo
echo "Tools available:"
echo "  - memory_query: Search memories"
echo "  - memory_recent: View recent memories"
echo "  - memory_write: Create new memory"
echo "  - memory_stats: Show statistics"
echo "  - memory_health: Check system health"
echo "  - memory_doctor: Run diagnostics"
