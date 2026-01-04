#!/bin/bash
# Demo script for Rate Limiting Middleware recipe
# Demonstrates per-client, per-service rate limiting using Redis

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CLIENT1="demo-client-1"
CLIENT2="demo-client-2"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Rate Limiting Middleware Demo${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if nats CLI is available
if ! command -v nats &> /dev/null; then
    echo -e "${RED}Error: nats CLI not found${NC}"
    echo "Install with: go install github.com/nats-io/natscli/nats@latest"
    exit 1
fi

# Function to make a request and show result
make_request() {
    local service=$1
    local data=$2
    local client_id=$3
    local show_response=${4:-true}

    if [ "$show_response" = true ]; then
        echo -e "${YELLOW}→ Request to ${service} (client: ${client_id})${NC}"
        response=$(nats request "$service" "$data" --header "X-Client-ID:${client_id}" 2>&1 || true)
        echo -e "${GREEN}← Response: ${response}${NC}"
        echo ""
    else
        nats request "$service" "$data" --header "X-Client-ID:${client_id}" > /dev/null 2>&1 || true
    fi
}

# Function to burst requests
burst_requests() {
    local service=$1
    local data=$2
    local client_id=$3
    local count=$4

    echo -e "${BLUE}Sending ${count} rapid requests to ${service}...${NC}"
    for i in $(seq 1 "$count"); do
        make_request "$service" "$data" "$client_id" false
        printf "."
    done
    echo ""
    echo -e "${GREEN}Burst complete!${NC}"
    echo ""
}

echo -e "${GREEN}=== Test 1: Basic Service Requests ===${NC}"
echo "Each service has different rate limits:"
echo "  - get-data:     100 req/min (default)"
echo "  - create-order:  50 req/min (restrictive)"
echo "  - get-status:   200 req/min (permissive)"
echo ""

echo -e "${YELLOW}Testing get-data (100 req/min limit):${NC}"
make_request "services.api.get-data" "{}" "$CLIENT1"

echo -e "${YELLOW}Testing get-status (200 req/min limit):${NC}"
make_request "services.api.get-status" "{}" "$CLIENT1"

echo -e "${YELLOW}Testing create-order (50 req/min limit):${NC}"
make_request "services.api.create-order" '{"product_id":"prod-123","quantity":2,"price":49.99}' "$CLIENT1"

echo ""
echo -e "${GREEN}=== Test 2: Per-Client Isolation ===${NC}"
echo "Different clients have separate rate limits."
echo ""

echo -e "${YELLOW}Client 1 request:${NC}"
make_request "services.api.get-data" "{}" "$CLIENT1"

echo -e "${YELLOW}Client 2 request (separate limit):${NC}"
make_request "services.api.get-data" "{}" "$CLIENT2"

echo ""
echo -e "${GREEN}=== Test 3: Rate Limit Enforcement ===${NC}"
echo "Sending 55 requests to create-order (limit: 50 req/min)"
echo "Client: rate-limit-test"
echo ""

# Use a fresh client to test rate limiting
TEST_CLIENT="rate-limit-test-$$"

burst_requests "services.api.create-order" '{"product_id":"test","quantity":1,"price":9.99}' "$TEST_CLIENT" 55

echo -e "${YELLOW}Now checking if rate limited (request 56):${NC}"
make_request "services.api.create-order" '{"product_id":"test","quantity":1,"price":9.99}' "$TEST_CLIENT"

echo ""
echo -e "${GREEN}=== Test 4: Anonymous Client ===${NC}"
echo "Requests without X-Client-ID header use 'anonymous' as client ID."
echo ""

echo -e "${YELLOW}Request without client ID:${NC}"
response=$(nats request "services.api.get-data" "{}" 2>&1 || true)
echo -e "${GREEN}← Response: ${response}${NC}"
echo ""

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Demo Complete!${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "Key takeaways:"
echo "  1. Rate limits are per-client AND per-service"
echo "  2. Different services can have different limits"
echo "  3. Middleware intercepts all request-reply services"
echo "  4. Redis sliding window ensures accurate limiting"
echo ""
echo "To reset rate limits, flush Redis:"
echo "  docker-compose exec redis redis-cli FLUSHALL"
