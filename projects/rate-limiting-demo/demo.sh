#!/bin/bash
# Rate Limiting Demo Script
# Demonstrates rate limiting behavior using curl

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-demo-api-key-12345}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
}

print_step() {
    echo -e "\n${YELLOW}▶ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to make request and show headers
make_request() {
    local endpoint=$1
    local description=$2
    local extra_headers=$3

    echo -e "\n${YELLOW}Request:${NC} curl $extra_headers $BASE_URL$endpoint"

    # Make request and capture both body and headers
    response=$(curl -s -w "\n%{http_code}" $extra_headers "$BASE_URL$endpoint" -D -)

    # Extract status code (last line)
    status_code=$(echo "$response" | tail -1)

    # Extract headers (everything before empty line)
    headers=$(echo "$response" | sed -n '1,/^\r$/p')

    # Extract body (everything after empty line, excluding status code)
    body=$(echo "$response" | sed -n '/^\r$/,$p' | tail -n +2 | head -n -1)

    # Show rate limit headers
    echo -e "${BLUE}Rate Limit Headers:${NC}"
    echo "$headers" | grep -i "x-ratelimit\|retry-after" | sed 's/^/  /'

    # Show response body
    echo -e "${BLUE}Response ($status_code):${NC}"
    echo "$body" | jq . 2>/dev/null || echo "$body"

    return 0
}

# Function to burst requests
burst_requests() {
    local endpoint=$1
    local count=$2
    local extra_headers=$3
    local success=0
    local rate_limited=0

    echo -e "\n${YELLOW}Sending $count rapid requests to $endpoint...${NC}"

    for i in $(seq 1 $count); do
        response=$(curl -s -o /dev/null -w "%{http_code}" $extra_headers "$BASE_URL$endpoint")
        if [ "$response" == "200" ]; then
            ((success++))
        elif [ "$response" == "429" ]; then
            ((rate_limited++))
        fi
        # Small delay to see progress
        if [ $((i % 10)) -eq 0 ]; then
            echo "  Progress: $i/$count requests sent (OK: $success, Rate Limited: $rate_limited)"
        fi
    done

    echo -e "\n${GREEN}Results:${NC}"
    echo "  Successful (200): $success"
    echo "  Rate Limited (429): $rate_limited"
}

# Check if server is running
check_server() {
    print_step "Checking if server is running..."
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        print_success "Server is running at $BASE_URL"
    else
        print_error "Server is not running. Please start it with: go run ."
        exit 1
    fi
}

# Main demo
main() {
    print_header "Rate Limiting Demo"

    check_server

    # Demo 1: Health check (no rate limiting)
    print_header "1. Health Check (No Rate Limiting)"
    make_request "/health" "Health check endpoint"

    # Demo 2: Public endpoint (IP-based rate limiting)
    print_header "2. Public Endpoint (IP-Based Rate Limiting)"
    print_step "Single request to public endpoint (100 req/min by IP)"
    make_request "/api/v1/public" "Public endpoint"

    # Demo 3: Premium endpoint with API key
    print_header "3. Premium Endpoint (API Key Rate Limiting)"
    print_step "Request with API key (1000 req/min by API key)"
    make_request "/api/v1/premium" "Premium endpoint with API key" "-H 'X-API-Key: $API_KEY'"

    # Demo 4: Check statistics
    print_header "4. Rate Limit Statistics"
    print_step "Checking current rate limit stats"
    make_request "/api/v1/stats" "Rate limit statistics" "-H 'X-API-Key: $API_KEY'"

    # Demo 5: Exceed rate limit
    print_header "5. Exceeding Rate Limit"
    echo -e "\n${YELLOW}This demo will send requests rapidly to trigger rate limiting.${NC}"
    echo -e "${YELLOW}The public endpoint allows 100 requests per minute.${NC}"
    read -p "Press Enter to continue (or Ctrl+C to skip)..."

    # Note: We use a smaller number to demonstrate without waiting
    # In real testing, you'd need to exceed the actual limit
    burst_requests "/api/v1/public" 110

    # Show the 429 response
    print_step "Showing rate limit exceeded response:"
    make_request "/api/v1/public" "Rate limited request"

    # Demo 6: Different API keys have separate limits
    print_header "6. Separate Limits Per API Key"
    print_step "Different API keys have independent rate limits"
    make_request "/api/v1/premium" "API Key 1" "-H 'X-API-Key: key-user-1'"
    make_request "/api/v1/premium" "API Key 2" "-H 'X-API-Key: key-user-2'"

    print_header "Demo Complete!"
    echo -e "
${GREEN}Key Takeaways:${NC}
  • Rate limits are enforced per-IP for public endpoints
  • Premium endpoints support per-API-key rate limiting
  • HTTP 429 responses include Retry-After header
  • X-RateLimit-* headers show current limit status
  • Statistics endpoint helps monitor rate limit usage

${BLUE}Try these commands yourself:${NC}
  # Normal request
  curl $BASE_URL/api/v1/public

  # With API key
  curl -H 'X-API-Key: your-key' $BASE_URL/api/v1/premium

  # Check stats
  curl $BASE_URL/api/v1/stats
"
}

# Run main function
main "$@"
