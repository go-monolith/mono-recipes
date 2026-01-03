#!/bin/bash
# Demo script for GORM + SQLite Recipe
# This script demonstrates full CRUD operations using NATS CLI
#
# Usage: ./demo.sh

set -e

APP_PID=""
PRODUCT1_ID=""
PRODUCT2_ID=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

print_success() {
    echo -e "${GREEN}$1${NC}"
}

print_info() {
    echo -e "${YELLOW}$1${NC}"
}

print_error() {
    echo -e "${RED}$1${NC}"
}

# Helper function to extract JSON from nats request output
# nats outputs timing info to stderr and JSON to stdout
nats_request() {
    nats request "$@" 2>/dev/null | grep -E '^\{' || echo '{"error":"no response"}'
}

# Cleanup function to stop the application
cleanup() {
    if [ -n "$APP_PID" ] && kill -0 "$APP_PID" 2>/dev/null; then
        echo -e "\n${YELLOW}Stopping application...${NC}"
        kill "$APP_PID" 2>/dev/null || true
        wait "$APP_PID" 2>/dev/null || true
    fi
    # Clean up database file
    rm -f products.db
}

trap cleanup EXIT

echo "=========================================="
echo "   GORM + SQLite Recipe Demo"
echo "   Using NATS Request-Reply Services"
echo "=========================================="
echo ""

# Check if nats CLI is available
if ! command -v nats &> /dev/null; then
    print_error "Error: nats CLI is required but not installed"
    echo ""
    echo "Install nats CLI:"
    echo "  macOS:   brew install nats-io/nats-tools/nats"
    echo "  Linux:   curl -sf https://binaries.nats.dev/nats-io/natscli/nats@latest | sh"
    echo "  Manual:  https://github.com/nats-io/natscli/releases"
    exit 1
fi

# Check if jq is available
if ! command -v jq &> /dev/null; then
    print_error "Error: jq is required but not installed"
    exit 1
fi

# Build the application
print_info "Building application..."
go build -o bin/gorm-sqlite-demo .
print_success "Build successful!"
echo ""

# Start the application in background
print_info "Starting application..."
./bin/gorm-sqlite-demo &
APP_PID=$!

# Wait for the application to start (NATS server needs time to initialize)
sleep 3

# Check if application is running
if ! kill -0 "$APP_PID" 2>/dev/null; then
    print_error "Error: Application failed to start"
    exit 1
fi
print_success "Application started (PID: $APP_PID)"
echo ""

# ============================================
# CREATE PRODUCTS
# ============================================
print_header "1. Create Products"

echo "Creating Product 1: Widget..."
RESPONSE1=$(nats_request services.product.create '{"name":"Widget","description":"A versatile widget for all your needs","price":19.99,"stock":100}' --timeout 5s)
echo "$RESPONSE1" | jq .
PRODUCT1_ID=$(echo "$RESPONSE1" | jq -r '.id')
print_success "Created product with ID: $PRODUCT1_ID"
echo ""

echo "Creating Product 2: Gadget..."
RESPONSE2=$(nats_request services.product.create '{"name":"Gadget","description":"An innovative gadget with smart features","price":49.99,"stock":50}' --timeout 5s)
echo "$RESPONSE2" | jq .
PRODUCT2_ID=$(echo "$RESPONSE2" | jq -r '.id')
print_success "Created product with ID: $PRODUCT2_ID"
echo ""

echo "Creating Product 3: Gizmo..."
RESPONSE3=$(nats_request services.product.create '{"name":"Gizmo","description":"A compact gizmo that fits in your pocket","price":9.99,"stock":200}' --timeout 5s)
echo "$RESPONSE3" | jq .
PRODUCT3_ID=$(echo "$RESPONSE3" | jq -r '.id')
print_success "Created product with ID: $PRODUCT3_ID"
echo ""

# ============================================
# LIST PRODUCTS
# ============================================
print_header "2. List All Products"

echo "Sending request to services.product.list..."
nats_request services.product.list '{}' --timeout 5s | jq .
echo ""

# ============================================
# GET SINGLE PRODUCT
# ============================================
print_header "3. Get Single Product"

echo "Getting product by ID: $PRODUCT1_ID"
nats_request services.product.get "{\"id\":\"$PRODUCT1_ID\"}" --timeout 5s | jq .
echo ""

# ============================================
# UPDATE PRODUCT
# ============================================
print_header "4. Update Product"

echo "Updating Widget price and stock..."
nats_request services.product.update "{\"id\":\"$PRODUCT1_ID\",\"price\":24.99,\"stock\":75}" --timeout 5s | jq .
echo ""

echo "Verifying update..."
nats_request services.product.get "{\"id\":\"$PRODUCT1_ID\"}" --timeout 5s | jq .
echo ""

# ============================================
# DELETE PRODUCT
# ============================================
print_header "5. Delete Product"

echo "Deleting product: $PRODUCT3_ID (Gizmo)"
nats_request services.product.delete "{\"id\":\"$PRODUCT3_ID\"}" --timeout 5s | jq .
echo ""

# ============================================
# VERIFY DELETION
# ============================================
print_header "6. Verify Deletion"

echo "Trying to get deleted product (should return error)..."
nats_request services.product.get "{\"id\":\"$PRODUCT3_ID\"}" --timeout 5s | jq .
echo ""

# ============================================
# FINAL LIST
# ============================================
print_header "7. Final Product List"

echo "Listing remaining products..."
nats_request services.product.list '{}' --timeout 5s | jq .
echo ""

# ============================================
# VALIDATION TESTS
# ============================================
print_header "8. Validation Tests"

echo "Testing validation: Creating product with empty name..."
nats_request services.product.create '{"name":"","price":10.00}' --timeout 5s | jq .
echo ""

echo "Testing validation: Creating product with negative price..."
nats_request services.product.create '{"name":"BadProduct","price":-5.00}' --timeout 5s | jq .
echo ""

echo "Testing validation: Getting non-existent product..."
nats_request services.product.get '{"id":"non-existent-id"}' --timeout 5s | jq .
echo ""

echo "=========================================="
print_success "Demo completed successfully!"
echo "=========================================="
echo ""
echo "This demo demonstrated:"
echo "  - Creating products via services.product.create"
echo "  - Listing products via services.product.list"
echo "  - Getting a product via services.product.get"
echo "  - Updating a product via services.product.update"
echo "  - Deleting a product via services.product.delete"
echo "  - Input validation for service requests"
echo ""
echo "All operations were performed using NATS request-reply pattern,"
echo "with no HTTP endpoints involved."
