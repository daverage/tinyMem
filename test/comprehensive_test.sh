#!/bin/bash

# tinyMem Comprehensive Test Suite
# Validates core functionality and measures tangible benefits of features

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test results
TEST_RESULTS=()

echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}${BOLD}  tinyMem Comprehensive Test Suite${NC}"
echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# ============================================================
# Helper Functions
# ============================================================

run_test() {
  local test_name="$1"
  local test_command="$2"

  ((TESTS_RUN++))
  echo -ne "${BLUE}→${NC} Testing: $test_name ... "

  if eval "$test_command" > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✓ $test_name")
    return 0
  else
    echo -e "${RED}✗${NC}"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("✗ $test_name")
    return 1
  fi
}

run_test_with_output() {
  local test_name="$1"
  local test_command="$2"
  local expected_output="$3"

  ((TESTS_RUN++))
  echo -ne "${BLUE}→${NC} Testing: $test_name ... "

  local output
  output=$(eval "$test_command" 2>&1 || true)

  if echo "$output" | grep -q "$expected_output"; then
    echo -e "${GREEN}✓${NC}"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✓ $test_name")
    return 0
  else
    echo -e "${RED}✗${NC} (expected: '$expected_output', got: '$output')"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("✗ $test_name")
    return 1
  fi
}

cleanup() {
  echo
  echo -e "${YELLOW}Cleaning up test environment...${NC}"
  rm -rf "$TEST_DIR"
}

# ============================================================
# Setup Test Environment
# ============================================================

echo -e "${BLUE}Setting up test environment...${NC}"

# Find tinymem binary
if [[ -f "$PROJECT_ROOT/tinymem" ]]; then
  TINYMEM="$PROJECT_ROOT/tinymem"
elif [[ -f "$PROJECT_ROOT/build/releases/tinymem" ]]; then
  TINYMEM="$PROJECT_ROOT/build/releases/tinymem"
elif command -v tinymem >/dev/null 2>&1; then
  TINYMEM="tinymem"
else
  echo -e "${RED}❌ tinymem binary not found${NC}"
  echo "   Build it first with: make build"
  exit 1
fi

echo -e "${GREEN}✓${NC} Using binary: $TINYMEM"

# Create temporary test directory
TEST_DIR=$(mktemp -d)
trap cleanup EXIT

cd "$TEST_DIR"

# Initialize git repo (required for tinymem to detect project root)
git init >/dev/null 2>&1
git config user.email "test@example.com" >/dev/null 2>&1
git config user.name "Test User" >/dev/null 2>&1

# Copy libllama_go.dylib if it exists (for embedded model support)
if [[ -f "$PROJECT_ROOT/libllama_go.dylib" ]]; then
  cp "$PROJECT_ROOT/libllama_go.dylib" .
  echo -e "${GREEN}✓${NC} Copied libllama_go.dylib for embedded model support"
fi

# Configure for testing with CoVe enabled (production mode)
mkdir -p .tinyMem
cat > .tinyMem/config.toml << 'EOF'
[cove]
enabled = true
confidence_threshold = 0.6
EOF

echo -e "${GREEN}✓${NC} Test directory: $TEST_DIR"
echo -e "${GREEN}✓${NC} Isolated .tinyMem will be created here"
echo

# ============================================================
# Core Functionality Tests
# ============================================================

echo -e "${BLUE}${BOLD}Core Functionality Tests${NC}"
echo

run_test "Health check" \
  "$TINYMEM health"

run_test "Stats command" \
  "$TINYMEM stats"

run_test "Write note memory" \
  "$TINYMEM write --type note --summary 'Test note' --detail 'Testing basic functionality'"

run_test "Write claim memory" \
  "$TINYMEM write --type claim --summary 'Go version' --detail 'Project uses Go 1.21'"

run_test "Write decision memory" \
  "$TINYMEM write --type decision --summary 'Use PostgreSQL' --detail 'Decided to use PostgreSQL for production'"

run_test "Query memories" \
  "$TINYMEM query 'test'"

run_test "Recent memories" \
  "$TINYMEM recent"

run_test "Dashboard view" \
  "$TINYMEM dashboard"

echo

# ============================================================
# Memory Type Tests
# ============================================================

echo -e "${BLUE}${BOLD}Memory Type Validation Tests${NC}"
echo

# Test all memory types (excluding fact which requires evidence via MCP)
for type in claim decision constraint observation plan note; do
  run_test "Memory type: $type" \
    "$TINYMEM write --type $type --summary 'Test $type' --detail 'Testing memory type: $type'"
done

echo

# ============================================================
# Recall Quality Tests
# ============================================================

echo -e "${BLUE}${BOLD}Recall Quality Tests${NC}"
echo

# Create diverse test memories
$TINYMEM write --type claim --summary "Backend technology" \
  --detail "The application backend is built with Go programming language" >/dev/null 2>&1

$TINYMEM write --type claim --summary "Database choice" \
  --detail "PostgreSQL is used for data persistence and reliability" >/dev/null 2>&1

$TINYMEM write --type decision --summary "API framework" \
  --detail "Decided to use Gin framework for REST API development" >/dev/null 2>&1

$TINYMEM write --type note --summary "Code style" \
  --detail "Follow standard Go formatting with gofmt and golint" >/dev/null 2>&1

# Test recall precision
run_test_with_output "Recall: exact match" \
  "$TINYMEM query 'PostgreSQL'" \
  "PostgreSQL"

run_test_with_output "Recall: partial match" \
  "$TINYMEM query 'database'" \
  "PostgreSQL\|Database"

run_test_with_output "Recall: conceptual match" \
  "$TINYMEM query 'backend programming'" \
  "Go"

echo

# ============================================================
# Data Persistence Tests
# ============================================================

echo -e "${BLUE}${BOLD}Data Persistence Tests${NC}"
echo

# Write a memory
$TINYMEM write --type claim --summary "Persistence test" \
  --detail "This memory tests data persistence across sessions" >/dev/null 2>&1

# Query it back
run_test_with_output "Data persists after write" \
  "$TINYMEM query 'persistence'" \
  "Persistence test"

# Restart simulation (query in new invocation)
run_test_with_output "Data persists across invocations" \
  "$TINYMEM query 'persistence test'" \
  "This memory tests data persistence"

echo

# ============================================================
# Evidence System Tests
# ============================================================
# Note: Evidence can only be provided via MCP interface, not CLI
# So we skip evidence tests in this CLI-focused test suite

echo -e "${BLUE}${BOLD}Evidence System Tests${NC}"
echo -e "${YELLOW}ℹ Evidence tests skipped (evidence only available via MCP interface)${NC}"
echo

# ============================================================
# Performance Tests
# ============================================================

echo -e "${BLUE}${BOLD}Performance Tests${NC}"
echo

# Bulk write test
echo -ne "${BLUE}→${NC} Bulk write (100 memories) ... "
START_TIME=$(date +%s)
for i in {1..100}; do
  $TINYMEM write --type note --summary "Bulk test $i" \
    --detail "Performance test memory number $i" >/dev/null 2>&1
done
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
echo -e "${GREEN}✓${NC} (${DURATION}s)"

# Bulk query test
echo -ne "${BLUE}→${NC} Bulk query (100 queries) ... "
START_TIME=$(date +%s)
for i in {1..100}; do
  $TINYMEM query "bulk test $((RANDOM % 100))" >/dev/null 2>&1
done
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))
echo -e "${GREEN}✓${NC} (${DURATION}s)"

echo

# ============================================================
# Edge Case Tests
# ============================================================

echo -e "${BLUE}${BOLD}Edge Case Tests${NC}"
echo

run_test "Empty query handling" \
  "$TINYMEM query ''"

run_test "Very long summary" \
  "$TINYMEM write --type note --summary 'This is a very long summary that tests the system ability to handle extended text content without truncation or errors' --detail 'Test'"

run_test "Special characters in content" \
  "$TINYMEM write --type note --summary 'Special chars: @#\$%^&*()' --detail 'Testing: <>&\"'"

run_test "Unicode content" \
  "$TINYMEM write --type note --summary 'Unicode test: 你好世界 🚀' --detail 'Testing international characters'"

echo

# ============================================================
# Integration Tests
# ============================================================

echo -e "${BLUE}${BOLD}Integration Tests${NC}"
echo

# Test complete workflow
$TINYMEM write --type decision --summary "Use Redis for caching" \
  --detail "Decided to implement Redis for session management and caching layer" \
  --evidence "cmd_exit0::which redis-cli" >/dev/null 2>&1 || true

run_test_with_output "Complete workflow: write → query" \
  "$TINYMEM query 'Redis caching'" \
  "Redis"

echo

# ============================================================
# Test Results Summary
# ============================================================

echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}${BOLD}  Test Results Summary${NC}"
echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo "Total Tests: $TESTS_RUN"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [[ $TESTS_FAILED -gt 0 ]]; then
  echo -e "${RED}Failed: $TESTS_FAILED${NC}"
fi

PASS_RATE=$((TESTS_PASSED * 100 / TESTS_RUN))
echo
echo -e "Pass Rate: ${BOLD}${PASS_RATE}%${NC}"

# Print all results
echo
echo "Detailed Results:"
for result in "${TEST_RESULTS[@]}"; do
  if [[ $result == ✓* ]]; then
    echo -e "  ${GREEN}$result${NC}"
  else
    echo -e "  ${RED}$result${NC}"
  fi
done

echo
if [[ $TESTS_FAILED -eq 0 ]]; then
  echo -e "${GREEN}${BOLD}✅ All tests passed!${NC}"
  exit 0
else
  echo -e "${RED}${BOLD}❌ Some tests failed${NC}"
  exit 1
fi
