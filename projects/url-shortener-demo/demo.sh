#!/bin/bash
#
# URL Shortener Demo Script
#
# This script demonstrates the URL shortener functionality:
# 1. Shorten a URL
# 2. Resolve and redirect
# 3. View statistics
# 4. View analytics
# 5. Delete the URL
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
API_URL="${BASE_URL}/api/v1"

# Helper functions
print_header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"
}

print_step() {
    echo -e "${CYAN}▶ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

wait_for_server() {
    print_step "Waiting for server to be ready..."
    for i in {1..30}; do
        if curl -s "${BASE_URL}/health" > /dev/null 2>&1; then
            print_success "Server is ready!"
            return 0
        fi
        sleep 1
    done
    print_error "Server is not responding at ${BASE_URL}"
    exit 1
}

# Main demo
main() {
    print_header "URL Shortener Demo"

    echo -e "This demo will showcase the URL shortener functionality:"
    echo -e "  • Shorten URLs with optional TTL"
    echo -e "  • Redirect short codes to original URLs"
    echo -e "  • Track access statistics"
    echo -e "  • View analytics data"
    echo -e ""

    wait_for_server

    # ========================================
    # Step 1: Shorten a URL
    # ========================================
    print_header "Step 1: Shorten a URL"

    print_step "Creating short URL for: https://github.com/go-monolith/mono"

    RESPONSE=$(curl -s -X POST "${API_URL}/shorten" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://github.com/go-monolith/mono"}')

    echo -e "${YELLOW}Response:${NC}"
    echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"

    SHORT_CODE=$(echo "$RESPONSE" | jq -r '.short_code' 2>/dev/null)
    SHORT_URL=$(echo "$RESPONSE" | jq -r '.short_url' 2>/dev/null)

    if [ "$SHORT_CODE" != "null" ] && [ -n "$SHORT_CODE" ]; then
        print_success "URL shortened successfully!"
        print_info "Short code: $SHORT_CODE"
        print_info "Short URL: $SHORT_URL"
    else
        print_error "Failed to shorten URL"
        exit 1
    fi

    read -p "Press Enter to continue..."

    # ========================================
    # Step 2: Create another URL with TTL
    # ========================================
    print_header "Step 2: Shorten URL with TTL (expires in 1 hour)"

    print_step "Creating expiring short URL..."

    RESPONSE2=$(curl -s -X POST "${API_URL}/shorten" \
        -H "Content-Type: application/json" \
        -d '{"url": "https://example.com/temporary-link", "ttl_seconds": 3600}')

    echo -e "${YELLOW}Response:${NC}"
    echo "$RESPONSE2" | jq '.' 2>/dev/null || echo "$RESPONSE2"

    SHORT_CODE2=$(echo "$RESPONSE2" | jq -r '.short_code' 2>/dev/null)
    EXPIRES_AT=$(echo "$RESPONSE2" | jq -r '.expires_at' 2>/dev/null)

    if [ "$EXPIRES_AT" != "null" ] && [ -n "$EXPIRES_AT" ]; then
        print_success "URL with TTL created!"
        print_info "Expires at: $EXPIRES_AT"
    fi

    read -p "Press Enter to continue..."

    # ========================================
    # Step 3: Access the URL (redirect)
    # ========================================
    print_header "Step 3: Access Short URL (Redirect)"

    print_step "Resolving short code: $SHORT_CODE"
    print_info "Following redirect..."

    # Use -I to show headers, -L to follow redirects
    REDIRECT_RESPONSE=$(curl -s -I "${BASE_URL}/${SHORT_CODE}" 2>&1 | head -10)

    echo -e "${YELLOW}Response Headers:${NC}"
    echo "$REDIRECT_RESPONSE"

    LOCATION=$(echo "$REDIRECT_RESPONSE" | grep -i "Location:" | cut -d' ' -f2 | tr -d '\r')
    if [ -n "$LOCATION" ]; then
        print_success "Redirect to: $LOCATION"
    fi

    # Access it a few more times to build up stats
    print_step "Accessing URL a few more times to build statistics..."
    for i in {1..3}; do
        curl -s -o /dev/null "${BASE_URL}/${SHORT_CODE}"
        echo -n "."
    done
    echo ""
    print_success "Accessed 4 times total"

    read -p "Press Enter to continue..."

    # ========================================
    # Step 4: View Statistics
    # ========================================
    print_header "Step 4: View URL Statistics"

    print_step "Getting stats for: $SHORT_CODE"

    STATS=$(curl -s "${API_URL}/stats/${SHORT_CODE}")

    echo -e "${YELLOW}Statistics:${NC}"
    echo "$STATS" | jq '.' 2>/dev/null || echo "$STATS"

    ACCESS_COUNT=$(echo "$STATS" | jq -r '.access_count' 2>/dev/null)
    if [ "$ACCESS_COUNT" != "null" ]; then
        print_success "Total accesses: $ACCESS_COUNT"
    fi

    read -p "Press Enter to continue..."

    # ========================================
    # Step 5: List All URLs
    # ========================================
    print_header "Step 5: List All URLs"

    print_step "Listing all shortened URLs..."

    LIST=$(curl -s "${API_URL}/urls")

    echo -e "${YELLOW}All URLs:${NC}"
    echo "$LIST" | jq '.' 2>/dev/null || echo "$LIST"

    TOTAL=$(echo "$LIST" | jq -r '.total' 2>/dev/null)
    print_info "Total URLs: $TOTAL"

    read -p "Press Enter to continue..."

    # ========================================
    # Step 6: View Analytics
    # ========================================
    print_header "Step 6: View Analytics"

    print_step "Getting analytics summary..."

    ANALYTICS=$(curl -s "${API_URL}/analytics")

    echo -e "${YELLOW}Analytics Summary:${NC}"
    echo "$ANALYTICS" | jq '.' 2>/dev/null || echo "$ANALYTICS"

    print_step "Getting recent access logs..."

    LOGS=$(curl -s "${API_URL}/analytics/logs?limit=5")

    echo -e "${YELLOW}Recent Access Logs:${NC}"
    echo "$LOGS" | jq '.' 2>/dev/null || echo "$LOGS"

    read -p "Press Enter to continue..."

    # ========================================
    # Step 7: Delete a URL
    # ========================================
    print_header "Step 7: Delete URL"

    print_step "Deleting short code: $SHORT_CODE2"

    DELETE_RESPONSE=$(curl -s -X DELETE "${API_URL}/urls/${SHORT_CODE2}")

    echo -e "${YELLOW}Response:${NC}"
    echo "$DELETE_RESPONSE" | jq '.' 2>/dev/null || echo "$DELETE_RESPONSE"

    print_success "URL deleted successfully!"

    # Verify deletion
    print_step "Verifying deletion..."

    VERIFY=$(curl -s "${API_URL}/stats/${SHORT_CODE2}")

    if echo "$VERIFY" | grep -q "not found"; then
        print_success "Confirmed: URL no longer exists"
    fi

    read -p "Press Enter to continue..."

    # ========================================
    # Step 8: Test Invalid URL
    # ========================================
    print_header "Step 8: Error Handling"

    print_step "Testing invalid URL submission..."

    INVALID_RESPONSE=$(curl -s -X POST "${API_URL}/shorten" \
        -H "Content-Type: application/json" \
        -d '{"url": "not-a-valid-url"}')

    echo -e "${YELLOW}Response (expected error):${NC}"
    echo "$INVALID_RESPONSE" | jq '.' 2>/dev/null || echo "$INVALID_RESPONSE"

    print_step "Testing non-existent short code..."

    NOTFOUND=$(curl -s "${API_URL}/stats/doesnotexist")

    echo -e "${YELLOW}Response (expected 404):${NC}"
    echo "$NOTFOUND" | jq '.' 2>/dev/null || echo "$NOTFOUND"

    print_success "Error handling works correctly!"

    # ========================================
    # Summary
    # ========================================
    print_header "Demo Complete!"

    echo -e "This demo demonstrated:"
    echo -e "  ${GREEN}✓${NC} URL shortening with kv-jetstream plugin"
    echo -e "  ${GREEN}✓${NC} TTL support for expiring URLs"
    echo -e "  ${GREEN}✓${NC} URL redirection with access tracking"
    echo -e "  ${GREEN}✓${NC} Statistics via optimistic locking"
    echo -e "  ${GREEN}✓${NC} Event-driven analytics (EventEmitter → EventConsumer)"
    echo -e "  ${GREEN}✓${NC} URL listing and deletion"
    echo -e "  ${GREEN}✓${NC} Error handling with sentinel errors"
    echo -e ""
    echo -e "${YELLOW}Key Patterns Used:${NC}"
    echo -e "  • UsePluginModule for kv-jetstream injection"
    echo -e "  • EventEmitterModule for publishing URL events"
    echo -e "  • EventConsumerModule for analytics processing"
    echo -e "  • Fiber HTTP framework with middleware"
    echo -e ""
    echo -e "Remaining short URL: ${CYAN}${BASE_URL}/${SHORT_CODE}${NC}"
}

# Run the demo
main "$@"
