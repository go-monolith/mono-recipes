#!/bin/bash
# Demo script for Prisma + sqlc PostgreSQL Recipe
# This script demonstrates full CRUD operations using NATS CLI
#
# Prerequisites:
#   - Node.js v20+ (for Prisma CLI)
#   - sqlc CLI
#   - nats CLI
#   - jq (for JSON formatting)
#
# Usage: ./demo.sh

set -e

APP_PID=""
ARTICLE1_ID=""
ARTICLE2_ID=""
ARTICLE3_ID=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

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

nats_request() {
    nats request "$@" 2>/dev/null | grep -E '^\{' || echo '{"error":"no response"}'
}

cleanup() {
    if [ -n "$APP_PID" ] && kill -0 "$APP_PID" 2>/dev/null; then
        echo -e "\n${YELLOW}Stopping application...${NC}"
        kill "$APP_PID" 2>/dev/null || true
        wait "$APP_PID" 2>/dev/null || true
    fi
    print_info "Stopping Prisma Postgres..."
    npx prisma dev stop prisma-postgres-demo 2>/dev/null || true
}

trap cleanup EXIT

echo "=========================================="
echo "   Prisma + sqlc PostgreSQL Recipe Demo"
echo "   Using NATS Request-Reply Services"
echo "=========================================="
echo ""

# Check prerequisites
print_info "Checking prerequisites..."

if ! command -v node &> /dev/null; then
    print_error "Error: Node.js is required but not installed"
    exit 1
fi

NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 20 ]; then
    print_error "Error: Node.js v20+ is required (found v$NODE_VERSION)"
    exit 1
fi

if ! command -v nats &> /dev/null; then
    print_error "Error: nats CLI is required but not installed"
    echo ""
    echo "Install nats CLI: go install github.com/nats-io/natscli/nats@latest"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    print_error "Error: jq is required but not installed"
    exit 1
fi

if ! command -v sqlc &> /dev/null; then
    print_error "Error: sqlc CLI is required but not installed"
    echo ""
    echo "Install sqlc CLI:"
    echo "  go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"
    exit 1
fi

print_success "All prerequisites satisfied!"
echo ""

# Install npm dependencies if needed
if [ ! -d "node_modules" ]; then
    print_info "Installing npm dependencies..."
    npm install
fi

# Check Prisma Postgres local server
print_header "Checking Local Prisma Postgres (PGlite)"
make prisma-dev-ls && print_success "Prisma Postgres is running!"
echo ""

# Build the application
print_header "Building Application"
print_info "Compiling Go application..."
go build -o bin/prisma-postgres-demo .
print_success "Build successful!"
echo ""

# Start the application
print_header "Starting Application"
print_info "Starting mono application..."
./bin/prisma-postgres-demo &
APP_PID=$!

sleep 3

if ! kill -0 "$APP_PID" 2>/dev/null; then
    print_error "Error: Application failed to start"
    exit 1
fi
print_success "Application started (PID: $APP_PID)"
echo ""

# ============================================
# CREATE ARTICLES
# ============================================
print_header "1. Create Articles"

echo "Creating Article 1: Getting Started with Go (draft)..."
RESPONSE1=$(nats_request services.article.create '{"title":"Getting Started with Go","content":"Go is a statically typed, compiled language designed at Google. This comprehensive guide covers installation, basic syntax, and your first program.","slug":"getting-started-with-go","published":false}' --timeout 5s)
echo "$RESPONSE1" | jq .
ARTICLE1_ID=$(echo "$RESPONSE1" | jq -r '.id')
print_success "Created draft article with ID: $ARTICLE1_ID"
echo ""

echo "Creating Article 2: Prisma with sqlc (published)..."
RESPONSE2=$(nats_request services.article.create '{"title":"Prisma with sqlc - Best of Both Worlds","content":"Combining Prisma migrations with sqlc type-safe queries gives you the best of both worlds: excellent local development experience and compile-time SQL validation.","slug":"prisma-with-sqlc","published":true}' --timeout 5s)
echo "$RESPONSE2" | jq .
ARTICLE2_ID=$(echo "$RESPONSE2" | jq -r '.id')
print_success "Created published article with ID: $ARTICLE2_ID"
echo ""

echo "Creating Article 3: Mono Framework Guide (draft)..."
RESPONSE3=$(nats_request services.article.create '{"title":"Building Modular Monoliths with Mono","content":"The mono framework enables building modular monolith applications with clear module boundaries, request-reply services, and event-driven architecture.","slug":"mono-framework-guide","published":false}' --timeout 5s)
echo "$RESPONSE3" | jq .
ARTICLE3_ID=$(echo "$RESPONSE3" | jq -r '.id')
print_success "Created draft article with ID: $ARTICLE3_ID"
echo ""

# ============================================
# LIST ARTICLES
# ============================================
print_header "2. List Articles"

echo "Listing all articles..."
nats_request services.article.list '{}' --timeout 5s | jq .
echo ""

echo "Listing only published articles..."
nats_request services.article.list '{"published":true}' --timeout 5s | jq .
echo ""

echo "Listing only draft articles..."
nats_request services.article.list '{"published":false}' --timeout 5s | jq .
echo ""

echo "Listing with pagination (limit=2, offset=0)..."
nats_request services.article.list '{"limit":2,"offset":0}' --timeout 5s | jq .
echo ""

# ============================================
# GET ARTICLE
# ============================================
print_header "3. Get Article"

echo "Getting article by ID: $ARTICLE1_ID"
nats_request services.article.get "{\"id\":\"$ARTICLE1_ID\"}" --timeout 5s | jq .
echo ""

echo "Getting article by slug: prisma-with-sqlc"
nats_request services.article.get '{"slug":"prisma-with-sqlc"}' --timeout 5s | jq .
echo ""

# ============================================
# UPDATE ARTICLE
# ============================================
print_header "4. Update Article"

echo "Updating article title..."
nats_request services.article.update "{\"id\":\"$ARTICLE1_ID\",\"title\":\"Getting Started with Go (Updated)\"}" --timeout 5s | jq .
echo ""

echo "Verifying update..."
nats_request services.article.get "{\"id\":\"$ARTICLE1_ID\"}" --timeout 5s | jq .
echo ""

# ============================================
# PUBLISH ARTICLE
# ============================================
print_header "5. Publish Draft Article"

echo "Publishing draft article: $ARTICLE1_ID"
nats_request services.article.publish "{\"id\":\"$ARTICLE1_ID\"}" --timeout 5s | jq .
echo ""

echo "Verifying publish status..."
nats_request services.article.get "{\"id\":\"$ARTICLE1_ID\"}" --timeout 5s | jq '.published'
echo ""

echo "Listing published articles (should now have 2)..."
nats_request services.article.list '{"published":true}' --timeout 5s | jq '.total'
echo ""

# ============================================
# DELETE ARTICLE
# ============================================
print_header "6. Delete Article"

echo "Deleting article: $ARTICLE3_ID"
nats_request services.article.delete "{\"id\":\"$ARTICLE3_ID\"}" --timeout 5s | jq .
echo ""

echo "Verifying deletion (should return error)..."
nats_request services.article.get "{\"id\":\"$ARTICLE3_ID\"}" --timeout 5s | jq .
echo ""

# ============================================
# FINAL LIST
# ============================================
print_header "7. Final Article List"

echo "Listing remaining articles..."
nats_request services.article.list '{}' --timeout 5s | jq .
echo ""

# ============================================
# VALIDATION TESTS
# ============================================
print_header "8. Validation Tests"

echo "Testing: Creating article with empty title..."
nats_request services.article.create '{"title":"","content":"Some content","slug":"test-slug"}' --timeout 5s | jq .
echo ""

echo "Testing: Creating article with empty content..."
nats_request services.article.create '{"title":"Title","content":"","slug":"test-slug"}' --timeout 5s | jq .
echo ""

echo "Testing: Creating article with empty slug..."
nats_request services.article.create '{"title":"Title","content":"Content","slug":""}' --timeout 5s | jq .
echo ""

echo "Testing: Creating article with duplicate slug..."
nats_request services.article.create '{"title":"Duplicate","content":"Content","slug":"prisma-with-sqlc"}' --timeout 5s | jq .
echo ""

echo "Testing: Getting non-existent article..."
nats_request services.article.get '{"id":"00000000-0000-0000-0000-000000000000"}' --timeout 5s | jq .
echo ""

echo "=========================================="
print_success "Demo completed successfully!"
echo "=========================================="
echo ""
echo "This demo demonstrated:"
echo "  - Creating articles via services.article.create"
echo "  - Listing articles with filtering via services.article.list"
echo "  - Getting articles by ID or slug via services.article.get"
echo "  - Updating articles via services.article.update"
echo "  - Publishing draft articles via services.article.publish"
echo "  - Deleting articles via services.article.delete"
echo "  - Input validation and error handling"
echo ""
echo "Key benefits of Prisma + sqlc approach:"
echo "  - Prisma: Easy local dev (PGlite, no Docker), declarative migrations"
echo "  - sqlc: Type-safe Go code, compile-time SQL validation"
echo ""
