#!/bin/bash

# Redis Caching Demo Script
# Demonstrates cache-aside pattern with Redis

set -e

BASE_URL="${BASE_URL:-http://localhost:3000}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
}

print_step() {
    echo ""
    echo -e "${CYAN}▶ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

wait_for_input() {
    # Skip interactive prompt if running non-interactively
    if [ -t 0 ]; then
        echo ""
        read -p "Press Enter to continue..."
    else
        echo ""
        sleep 0.5
    fi
}

# Check if server is running
check_server() {
    print_step "Checking if server is running..."
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        print_success "Server is running at $BASE_URL"
    else
        echo -e "${RED}✗ Server is not running. Please start it with: go run main.go${NC}"
        exit 1
    fi
}

# Note: Cache statistics endpoints were removed in the refactoring.
# Use Redis monitoring tools (redis-cli INFO stats) for cache metrics.

# Show Redis stats (optional - requires redis-cli)
show_redis_stats() {
    print_step "Redis cache info:"
    if command -v redis-cli &> /dev/null; then
        redis-cli INFO stats 2>/dev/null | grep -E "(keyspace_hits|keyspace_misses)" || echo "Redis stats not available"
    else
        print_info "redis-cli not available - skipping stats"
    fi
}

# Create a product
create_product() {
    local name="$1"
    local price="$2"
    local stock="$3"
    local category="$4"
    local description="$5"

    print_step "Creating product: $name"
    curl -s -X POST "$BASE_URL/api/v1/products" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"$name\",
            \"description\": \"$description\",
            \"price\": $price,
            \"stock\": $stock,
            \"category\": \"$category\"
        }" | jq .
}

# Get a product by ID
get_product() {
    local id="$1"
    print_step "Getting product ID=$id"
    curl -s "$BASE_URL/api/v1/products/$id" | jq .
}

# List products
list_products() {
    local offset="${1:-0}"
    local limit="${2:-10}"
    print_step "Listing products (offset=$offset, limit=$limit)"
    curl -s "$BASE_URL/api/v1/products?offset=$offset&limit=$limit" | jq .
}

# Update a product
update_product() {
    local id="$1"
    local price="$2"
    print_step "Updating product ID=$id with new price=$price"
    curl -s -X PUT "$BASE_URL/api/v1/products/$id" \
        -H "Content-Type: application/json" \
        -d "{\"price\": $price}" | jq .
}

# Delete a product
delete_product() {
    local id="$1"
    print_step "Deleting product ID=$id"
    curl -s -X DELETE "$BASE_URL/api/v1/products/$id" | jq .
}

# Main demo
main() {
    print_header "Redis Caching Demo - Cache-Aside Pattern"

    check_server
    wait_for_input

    # Step 1: Create products
    print_header "Step 1: Create Sample Products"
    print_info "Creating products populates the database but NOT the cache"

    create_product "MacBook Pro" 2499.99 25 "Electronics" "High-performance laptop for professionals"
    create_product "iPhone 15" 999.99 100 "Electronics" "Latest smartphone with advanced features"
    create_product "AirPods Pro" 249.99 200 "Electronics" "Wireless earbuds with noise cancellation"
    create_product "Magic Keyboard" 99.99 150 "Accessories" "Wireless keyboard for Mac"
    create_product "USB-C Cable" 19.99 500 "Accessories" "High-quality USB-C charging cable"

    print_success "Created 5 products"
    print_info "Notice: No cache hits yet (all data in database only)"
    wait_for_input

    # Step 2: Demonstrate cache-aside on individual product
    print_header "Step 2: Cache-Aside Pattern - Single Product"

    print_info "First request - CACHE MISS (data fetched from database)"
    get_product 1
    print_info "Notice: 'from_cache': false - data came from database"
    wait_for_input

    print_info "Second request - CACHE HIT (data fetched from Redis)"
    get_product 1
    print_info "Notice: 'from_cache': true - data came from cache (much faster!)"
    wait_for_input

    print_info "Third request - Another CACHE HIT"
    get_product 1
    print_info "Notice: Multiple cache hits show the caching is working"
    wait_for_input

    # Step 3: Demonstrate cache-aside on list
    print_header "Step 3: Cache-Aside Pattern - Product List"

    print_info "First list request - CACHE MISS"
    list_products 0 10
    wait_for_input

    print_info "Second list request - CACHE HIT"
    list_products 0 10
    print_info "Notice: 'from_cache': true for the list"
    wait_for_input

    # Step 4: Cache invalidation on update
    print_header "Step 4: Cache Invalidation - Update"

    print_info "Current product 1 state (from cache):"
    get_product 1
    wait_for_input

    print_info "Updating product 1 price..."
    update_product 1 2299.99
    print_info "Cache has been invalidated!"
    wait_for_input

    print_info "Getting product 1 after update - CACHE MISS (re-fetched from database)"
    get_product 1
    print_info "Notice: 'from_cache': false - cache was invalidated, new data from DB"
    wait_for_input

    # Step 5: Cache invalidation on delete
    print_header "Step 5: Cache Invalidation - Delete"

    print_info "Getting product 5 to populate cache..."
    get_product 5
    wait_for_input

    print_info "Deleting product 5..."
    delete_product 5
    print_info "Cache has been invalidated!"
    wait_for_input

    print_info "Trying to get deleted product 5..."
    get_product 5
    print_info "Product not found - correctly deleted from both DB and cache"
    wait_for_input

    # Step 6: Different pagination cached separately
    print_header "Step 6: Pagination - Separate Cache Entries"

    print_info "List with offset=0, limit=2 (first time - MISS):"
    list_products 0 2
    wait_for_input

    print_info "List with offset=2, limit=2 (different cache key - MISS):"
    list_products 2 2
    wait_for_input

    print_info "List with offset=0, limit=2 again (CACHE HIT):"
    list_products 0 2
    print_info "Each pagination combination has its own cache entry"
    wait_for_input

    # Summary
    print_header "Demo Complete!"
    echo ""
    echo "Key takeaways:"
    echo "  1. Cache-aside pattern: Check cache first, DB on miss, populate cache"
    echo "  2. Cache hits are much faster than database queries"
    echo "  3. Updates and deletes automatically invalidate cache"
    echo "  4. Different query parameters result in different cache entries"
    echo "  5. Use Redis monitoring tools (redis-cli INFO stats) for cache metrics"
    echo ""
}

# Run main demo
main
