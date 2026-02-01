#!/bin/bash

# tinyMem Recall Benchmark Suite
# Proves tangible benefits of semantic search over lexical search

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

echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}${BOLD}  tinyMem Semantic vs Lexical Recall Benchmark${NC}"
echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# ============================================================
# Setup
# ============================================================

# Find tinymem binary
if [[ -f "$PROJECT_ROOT/tinymem" ]]; then
  TINYMEM="$PROJECT_ROOT/tinymem"
elif [[ -f "$PROJECT_ROOT/build/releases/tinymem" ]]; then
  TINYMEM="$PROJECT_ROOT/build/releases/tinymem"
elif command -v tinymem >/dev/null 2>&1; then
  TINYMEM="tinymem"
else
  echo -e "${RED}❌ tinymem binary not found${NC}"
  exit 1
fi

echo -e "${GREEN}✓${NC} Using binary: $TINYMEM"

# Create temporary test environments
TEST_DIR_LEXICAL=$(mktemp -d)
TEST_DIR_SEMANTIC=$(mktemp -d)

cleanup() {
  rm -rf "$TEST_DIR_LEXICAL" "$TEST_DIR_SEMANTIC"
}
trap cleanup EXIT

# Initialize git repos in both directories (required for tinymem)
(cd "$TEST_DIR_LEXICAL" && git init >/dev/null 2>&1 && \
  git config user.email "test@example.com" >/dev/null 2>&1 && \
  git config user.name "Test User" >/dev/null 2>&1)

(cd "$TEST_DIR_SEMANTIC" && git init >/dev/null 2>&1 && \
  git config user.email "test@example.com" >/dev/null 2>&1 && \
  git config user.name "Test User" >/dev/null 2>&1)

# Copy libllama_go.dylib to BOTH test directories if it exists
# (Required for full builds with embedded model)
if [[ -f "$PROJECT_ROOT/libllama_go.dylib" ]]; then
  cp "$PROJECT_ROOT/libllama_go.dylib" "$TEST_DIR_LEXICAL/"
  cp "$PROJECT_ROOT/libllama_go.dylib" "$TEST_DIR_SEMANTIC/"
  echo -e "${GREEN}✓${NC} Copied libllama_go.dylib to both test directories"
fi

# Configure semantic search BEFORE populating memories
# This ensures memories get embedded when written
mkdir -p "$TEST_DIR_SEMANTIC/.tinyMem"
cat > "$TEST_DIR_SEMANTIC/.tinyMem/config.toml" << 'EOF'
[recall]
semantic_enabled = true
hybrid_weight = 0.7
EOF

echo -e "${GREEN}✓${NC} Test directories created with isolated .tinyMem"
echo

# ============================================================
# Populate Test Data
# ============================================================

echo -e "${BLUE}Populating test dataset...${NC}"

# Create test memories with semantic variations
# These test synonym understanding, conceptual similarity, and context awareness

TEST_MEMORIES=(
  "claim|Database Selection|We chose PostgreSQL for its robust ACID compliance and JSON support"
  "claim|Backend Language|The server-side application is implemented in Go for performance"
  "claim|Frontend Framework|React is used for building the user interface components"
  "decision|Cache Strategy|Decided to implement Redis for session storage and rate limiting"
  "decision|API Design|REST API with JSON responses following OpenAPI 3.0 specification"
  "claim|Authentication|JWT tokens with RS256 signing for stateless authentication"
  "claim|Deployment|Application runs in Docker containers orchestrated by Kubernetes"
  "decision|Testing Approach|Use pytest for backend tests and Jest for frontend testing"
  "claim|Monitoring|Prometheus metrics with Grafana dashboards for observability"
  "decision|Logging|Structured logging with JSON format using zap library"
  "claim|Error Handling|Custom error types with stack traces and error codes"
  "claim|Configuration|Environment-based config with validation using Viper"
  "decision|Code Review|Require two approvals before merging to main branch"
  "claim|CI/CD Pipeline|GitHub Actions for automated testing and deployment"
  "decision|Security|Enable CORS, rate limiting, and input validation middleware"
)

# Populate both test environments
POPULATED=0
for memory in "${TEST_MEMORIES[@]}"; do
  IFS='|' read -r type summary detail <<< "$memory"

  # Lexical environment
  (cd "$TEST_DIR_LEXICAL" && $TINYMEM write --type "$type" --summary "$summary" --detail "$detail" >/dev/null 2>&1)

  # Semantic environment
  (cd "$TEST_DIR_SEMANTIC" && $TINYMEM write --type "$type" --summary "$summary" --detail "$detail" >/dev/null 2>&1)

  ((POPULATED++))
  if [[ $POPULATED -eq 1 ]]; then
    echo -e "  ${YELLOW}(Model initialization on first write - takes ~10 seconds)${NC}"
  fi
  echo -ne "\r  Populated: $POPULATED/${#TEST_MEMORIES[@]} memories"
done

echo
echo -e "${GREEN}✓${NC} Populated ${#TEST_MEMORIES[@]} test memories in both environments"
echo -e "${GREEN}✓${NC} Semantic environment uses embedded model (no Ollama needed)"
echo

# ============================================================
# Benchmark Queries
# ============================================================

echo -e "${BLUE}${BOLD}Running Benchmark Queries${NC}"
echo
echo "Testing various query types that benefit from semantic understanding:"
echo

# Test queries that semantic search should handle better
QUERIES=(
  "What database do we use?|PostgreSQL"  # Synonym test
  "How is the backend built?|Go"  # Conceptual similarity
  "UI framework|React"  # Abbreviation understanding
  "Where do we store sessions?|Redis"  # Context awareness
  "Container deployment|Docker"  # Related concepts
  "How do we test code?|pytest\|Jest"  # Multiple answers
  "What handles request logging?|zap"  # Domain knowledge
  "API response format|JSON"  # Format specification
  "Token authentication method|JWT"  # Technical terms
  "Pull request requirements|two approvals"  # Policy questions
)

LEXICAL_HITS=0
SEMANTIC_HITS=0
TOTAL_QUERIES=${#QUERIES[@]}

for query_data in "${QUERIES[@]}"; do
  IFS='|' read -r query expected <<< "$query_data"

  echo -ne "  Query: ${BOLD}$query${NC}\n"
  echo -ne "    Expected: ${YELLOW}$expected${NC}\n"

  # Test lexical recall
  LEXICAL_RESULT=$(cd "$TEST_DIR_LEXICAL" && $TINYMEM query "$query" 2>/dev/null || echo "")
  if echo "$LEXICAL_RESULT" | grep -qi "$expected"; then
    echo -ne "    Lexical: ${GREEN}✓ Found${NC}\n"
    ((LEXICAL_HITS++))
  else
    echo -ne "    Lexical: ${RED}✗ Not found${NC}\n"
  fi

  # Test semantic recall (model already initialized from write phase)
  SEMANTIC_RESULT=$(cd "$TEST_DIR_SEMANTIC" && $TINYMEM query "$query" 2>/dev/null || echo "")
  if echo "$SEMANTIC_RESULT" | grep -qi "$expected"; then
    echo -ne "    Semantic: ${GREEN}✓ Found${NC}\n"
    ((SEMANTIC_HITS++))
  else
    echo -ne "    Semantic: ${YELLOW}✗ Not found${NC}\n"
  fi

  echo
done

# ============================================================
# Results Analysis
# ============================================================

echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}${BOLD}  Benchmark Results${NC}"
echo -e "${BLUE}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

echo "Total Queries: $TOTAL_QUERIES"
echo

echo -e "${BOLD}Lexical Search (FTS5):${NC}"
echo "  Hits: $LEXICAL_HITS / $TOTAL_QUERIES"
LEXICAL_PCT=$((LEXICAL_HITS * 100 / TOTAL_QUERIES))
echo -e "  Precision: ${BOLD}${LEXICAL_PCT}%${NC}"

echo

echo -e "${BOLD}Semantic Search (Embedded Model):${NC}"
echo "  Hits: $SEMANTIC_HITS / $TOTAL_QUERIES"
SEMANTIC_PCT=$((SEMANTIC_HITS * 100 / TOTAL_QUERIES))
echo -e "  Precision: ${BOLD}${SEMANTIC_PCT}%${NC}"

echo
IMPROVEMENT=$((SEMANTIC_PCT - LEXICAL_PCT))
if [[ $IMPROVEMENT -gt 0 ]]; then
  echo -e "  ${GREEN}${BOLD}Improvement: +${IMPROVEMENT}%${NC}"
  echo
  echo -e "${GREEN}✅ Semantic search provides measurably better recall precision${NC}"
elif [[ $IMPROVEMENT -eq 0 ]]; then
  echo -e "  ${YELLOW}Performance: Equal to lexical (both $SEMANTIC_PCT%)${NC}"
  echo
  if [[ $SEMANTIC_HITS -eq 0 ]]; then
    echo -e "${YELLOW}⚠ No results found - possible configuration issue${NC}"
  else
    echo -e "${GREEN}✅ Semantic search working with embedded model${NC}"
  fi
else
  echo -e "  ${RED}Performance: ${IMPROVEMENT}% (worse than lexical)${NC}"
fi

echo

# ============================================================
# Key Insights
# ============================================================

echo -e "${BLUE}${BOLD}Key Insights:${NC}"
echo

echo "Semantic search excels at:"
echo "  • Understanding synonyms and related terms"
echo "  • Conceptual similarity matching"
echo "  • Context-aware retrieval"
echo "  • Handling abbreviations and technical jargon"
echo

echo "Lexical search works best for:"
echo "  • Exact keyword matches"
echo "  • Known terminology"
echo "  • Specific identifiers"
echo

if [[ $SEMANTIC_HITS -gt $LEXICAL_HITS ]]; then
  echo -e "${GREEN}${BOLD}Recommendation: Enable semantic search for production use${NC}"
else
  echo -e "${YELLOW}Recommendation: Lexical search sufficient if exact matches preferred${NC}"
fi

echo
