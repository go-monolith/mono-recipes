#!/bin/bash

# URL Shortener Demo Script
# This script demonstrates the URL shortener API endpoints

set -e

BASE_URL="${BASE_URL:-http://localhost:3000}"

echo "=== URL Shortener Demo ==="
echo "Base URL: $BASE_URL"
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print section headers
section() {
    echo -e "${BLUE}=== $1 ===${NC}"
}

# Function to print success messages
success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print info messages
info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Wait for service to be ready
section "Checking Service Health"
info "Waiting for service to be ready..."

for i in {1..30}; do
    if curl -s "${BASE_URL}/health" > /dev/null 2>&1; then
        success "Service is healthy!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Error: Service not responding after 30 seconds"
        exit 1
    fi
    sleep 1
done

echo ""
curl -s "${BASE_URL}/health" | jq .
echo ""

# Demo 1: Create a shortened URL
section "Demo 1: Create a Shortened URL"
info "POST /api/v1/shorten"
echo ""

RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/shorten" \
    -H "Content-Type: application/json" \
    -d '{"url": "https://github.com/go-monolith/mono"}')

echo "$RESPONSE" | jq .
SHORT_CODE=$(echo "$RESPONSE" | jq -r '.short_code')
success "Created short URL with code: $SHORT_CODE"
echo ""

# Demo 2: Create with custom code
section "Demo 2: Create with Custom Code"
info "POST /api/v1/shorten with custom_code"
echo ""

CUSTOM_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/shorten" \
    -H "Content-Type: application/json" \
    -d '{"url": "https://example.com/documentation", "custom_code": "docs"}')

echo "$CUSTOM_RESPONSE" | jq .
success "Created custom short URL: docs"
echo ""

# Demo 3: Create with TTL
section "Demo 3: Create with TTL (1 hour expiration)"
info "POST /api/v1/shorten with ttl_seconds"
echo ""

TTL_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/shorten" \
    -H "Content-Type: application/json" \
    -d '{"url": "https://example.com/temporary-link", "ttl_seconds": 3600}')

echo "$TTL_RESPONSE" | jq .
TTL_CODE=$(echo "$TTL_RESPONSE" | jq -r '.short_code')
success "Created short URL with 1 hour TTL: $TTL_CODE"
echo ""

# Demo 4: Resolve URL (follow redirect)
section "Demo 4: Resolve Short URL"
info "GET /:code (following redirect)"
echo ""

info "Resolving short code: $SHORT_CODE"
REDIRECT_URL=$(curl -s -o /dev/null -w "%{redirect_url}" "${BASE_URL}/${SHORT_CODE}")
success "Redirects to: $REDIRECT_URL"
echo ""

# Demo 5: Access the URL multiple times to generate stats
section "Demo 5: Generate Access Statistics"
info "Accessing short URL multiple times..."
echo ""

for i in {1..5}; do
    curl -s -o /dev/null "${BASE_URL}/${SHORT_CODE}"
    echo "  Access #$i completed"
done
success "Completed 5 accesses"
echo ""

# Demo 6: Get URL statistics
section "Demo 6: Get URL Statistics"
info "GET /api/v1/stats/:code"
echo ""

# Small delay to allow async stats update
sleep 1

STATS=$(curl -s "${BASE_URL}/api/v1/stats/${SHORT_CODE}")
echo "$STATS" | jq .
ACCESS_COUNT=$(echo "$STATS" | jq -r '.access_count')
success "Total access count: $ACCESS_COUNT"
echo ""

# Demo 7: Error handling - invalid URL
section "Demo 7: Error Handling - Invalid URL"
info "POST /api/v1/shorten with invalid URL"
echo ""

ERROR_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/shorten" \
    -H "Content-Type: application/json" \
    -d '{"url": "not-a-valid-url"}')

echo "$ERROR_RESPONSE" | jq .
success "Properly rejected invalid URL"
echo ""

# Demo 8: Error handling - duplicate custom code
section "Demo 8: Error Handling - Duplicate Custom Code"
info "POST /api/v1/shorten with existing custom_code"
echo ""

DUPLICATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/shorten" \
    -H "Content-Type: application/json" \
    -d '{"url": "https://example.com/another", "custom_code": "docs"}')

echo "$DUPLICATE_RESPONSE" | jq .
success "Properly rejected duplicate custom code"
echo ""

# Demo 9: Error handling - not found
section "Demo 9: Error Handling - URL Not Found"
info "GET /api/v1/stats/nonexistent"
echo ""

NOTFOUND_RESPONSE=$(curl -s "${BASE_URL}/api/v1/stats/nonexistent")
echo "$NOTFOUND_RESPONSE" | jq .
success "Properly returned not found error"
echo ""

# Summary
section "Demo Complete!"
echo ""
echo "Summary of created short URLs:"
echo "  1. ${BASE_URL}/${SHORT_CODE} → https://github.com/go-monolith/mono"
echo "  2. ${BASE_URL}/docs → https://example.com/documentation"
echo "  3. ${BASE_URL}/${TTL_CODE} → https://example.com/temporary-link (expires in 1 hour)"
echo ""
echo "Try these commands yourself:"
echo "  curl ${BASE_URL}/api/v1/stats/${SHORT_CODE}"
echo "  curl -L ${BASE_URL}/${SHORT_CODE}"
echo ""
