#!/bin/bash
# Demo script for sqlc + PostgreSQL Recipe
# This script demonstrates full CRUD operations using NATS CLI
# with psql verification of database state
#
# Usage: ./demo.sh

set -e

APP_PID=""
USER1_ID=""
USER2_ID=""
USER3_ID=""

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
nats_request() {
    nats request "$@" 2>/dev/null | grep -E '^\{' || echo '{"error":"no response"}'
}

# psql helper function
psql_query() {
    docker compose exec -T postgres psql -U demo -d users_db -c "$1"
}

# Cleanup function
cleanup() {
    if [ -n "$APP_PID" ] && kill -0 "$APP_PID" 2>/dev/null; then
        echo -e "\n${YELLOW}Stopping application...${NC}"
        kill "$APP_PID" 2>/dev/null || true
        wait "$APP_PID" 2>/dev/null || true
    fi
}

trap cleanup EXIT

echo "=========================================="
echo "   sqlc + PostgreSQL Recipe Demo"
echo "   Using NATS Request-Reply Services"
echo "=========================================="
echo ""

# Check prerequisites
if ! command -v nats &> /dev/null; then
    print_error "Error: nats CLI is required but not installed"
    echo ""
    echo "Install nats CLI:"
    echo "  macOS:   brew install nats-io/nats-tools/nats"
    echo "  Linux:   curl -sf https://binaries.nats.dev/nats-io/natscli/nats@latest | sh"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    print_error "Error: jq is required but not installed"
    exit 1
fi

if ! command -v docker &> /dev/null; then
    print_error "Error: docker is required but not installed"
    exit 1
fi

# Start PostgreSQL
print_info "Starting PostgreSQL via docker compose..."
docker compose up -d --wait
print_success "PostgreSQL is ready!"
echo ""

# Build the application
print_info "Building application..."
go build -o bin/sqlc-postgres-demo .
print_success "Build successful!"
echo ""

# Start the application
print_info "Starting application..."
./bin/sqlc-postgres-demo &
APP_PID=$!

# Wait for application to start
sleep 3

if ! kill -0 "$APP_PID" 2>/dev/null; then
    print_error "Error: Application failed to start"
    exit 1
fi
print_success "Application started (PID: $APP_PID)"
echo ""

# ============================================
# CREATE USERS
# ============================================
print_header "1. Create Users"

echo "Creating User 1: Alice..."
RESPONSE1=$(nats_request services.user.create '{"name":"Alice Johnson","email":"alice@example.com"}' --timeout 5s)
echo "$RESPONSE1" | jq .
USER1_ID=$(echo "$RESPONSE1" | jq -r '.id')
print_success "Created user with ID: $USER1_ID"
echo ""

echo "Creating User 2: Bob..."
RESPONSE2=$(nats_request services.user.create '{"name":"Bob Smith","email":"bob@example.com"}' --timeout 5s)
echo "$RESPONSE2" | jq .
USER2_ID=$(echo "$RESPONSE2" | jq -r '.id')
print_success "Created user with ID: $USER2_ID"
echo ""

echo "Creating User 3: Charlie..."
RESPONSE3=$(nats_request services.user.create '{"name":"Charlie Brown","email":"charlie@example.com"}' --timeout 5s)
echo "$RESPONSE3" | jq .
USER3_ID=$(echo "$RESPONSE3" | jq -r '.id')
print_success "Created user with ID: $USER3_ID"
echo ""

# Verify with psql
print_header "1b. Verify in PostgreSQL"
echo "Querying users table directly with psql..."
psql_query "SELECT id, name, email, created_at FROM users ORDER BY created_at;"
echo ""

# ============================================
# LIST USERS WITH PAGINATION
# ============================================
print_header "2. List Users (with Pagination)"

echo "Listing all users (default limit=10)..."
nats_request services.user.list '{}' --timeout 5s | jq .
echo ""

echo "Listing with limit=2, offset=0..."
nats_request services.user.list '{"limit":2,"offset":0}' --timeout 5s | jq .
echo ""

echo "Listing with limit=2, offset=2 (second page)..."
nats_request services.user.list '{"limit":2,"offset":2}' --timeout 5s | jq .
echo ""

# ============================================
# GET SINGLE USER
# ============================================
print_header "3. Get Single User"

echo "Getting user by ID: $USER1_ID"
nats_request services.user.get "{\"id\":\"$USER1_ID\"}" --timeout 5s | jq .
echo ""

# ============================================
# UPDATE USER
# ============================================
print_header "4. Update User"

echo "Updating Alice's name and email..."
nats_request services.user.update "{\"id\":\"$USER1_ID\",\"name\":\"Alice Johnson-Updated\",\"email\":\"alice.updated@example.com\"}" --timeout 5s | jq .
echo ""

echo "Verifying update..."
nats_request services.user.get "{\"id\":\"$USER1_ID\"}" --timeout 5s | jq .
echo ""

# Verify with psql
print_header "4b. Verify Update in PostgreSQL"
psql_query "SELECT id, name, email, updated_at FROM users WHERE id = '$USER1_ID';"
echo ""

# ============================================
# DELETE USER
# ============================================
print_header "5. Delete User"

echo "Deleting user: $USER3_ID (Charlie)"
nats_request services.user.delete "{\"id\":\"$USER3_ID\"}" --timeout 5s | jq .
echo ""

# ============================================
# VERIFY DELETION
# ============================================
print_header "6. Verify Deletion"

echo "Trying to get deleted user (should return error)..."
nats_request services.user.get "{\"id\":\"$USER3_ID\"}" --timeout 5s | jq .
echo ""

# Verify with psql
print_header "6b. Verify Deletion in PostgreSQL"
echo "Counting users in table..."
psql_query "SELECT COUNT(*) FROM users;"
echo ""
psql_query "SELECT id, name, email FROM users ORDER BY created_at;"
echo ""

# ============================================
# FINAL LIST
# ============================================
print_header "7. Final User List"

echo "Listing remaining users..."
nats_request services.user.list '{}' --timeout 5s | jq .
echo ""

# ============================================
# VALIDATION TESTS
# ============================================
print_header "8. Validation Tests"

echo "Testing: Creating user with empty name..."
nats_request services.user.create '{"name":"","email":"test@example.com"}' --timeout 5s | jq .
echo ""

echo "Testing: Creating user with empty email..."
nats_request services.user.create '{"name":"Test User","email":""}' --timeout 5s | jq .
echo ""

echo "Testing: Creating user with duplicate email..."
nats_request services.user.create '{"name":"Duplicate","email":"alice.updated@example.com"}' --timeout 5s | jq .
echo ""

echo "Testing: Getting non-existent user..."
nats_request services.user.get '{"id":"00000000-0000-0000-0000-000000000000"}' --timeout 5s | jq .
echo ""

echo "=========================================="
print_success "Demo completed successfully!"
echo "=========================================="
echo ""
echo "This demo demonstrated:"
echo "  - Creating users via services.user.create"
echo "  - Listing users with pagination via services.user.list"
echo "  - Getting a user via services.user.get"
echo "  - Updating a user via services.user.update"
echo "  - Deleting a user via services.user.delete"
echo "  - Direct psql verification of PostgreSQL data"
echo "  - Input validation and error handling"
echo ""
echo "All operations were performed using NATS request-reply pattern,"
echo "with type-safe SQL generated by sqlc."
echo ""
echo "To clean up:"
echo "  docker compose down -v"
