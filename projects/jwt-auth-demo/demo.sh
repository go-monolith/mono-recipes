#!/bin/bash
# JWT Authentication Demo Script
# Usage: ./demo.sh [command]

set -e

BASE_URL="${BASE_URL:-http://localhost:3000}"
API_URL="$BASE_URL/api/v1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Store tokens for the demo
ACCESS_TOKEN=""
REFRESH_TOKEN=""
USER_EMAIL="demo@example.com"
USER_PASSWORD="password123"

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

print_success() {
    echo -e "${GREEN}$1${NC}"
}

print_error() {
    echo -e "${RED}$1${NC}"
}

print_info() {
    echo -e "${YELLOW}$1${NC}"
}

# Health check
health() {
    print_header "Health Check"
    curl -s "$BASE_URL/health" | jq .
}

# Register a new user
register() {
    local email="${1:-$USER_EMAIL}"
    local password="${2:-$USER_PASSWORD}"

    print_header "Registering User"
    print_info "Email: $email"
    print_info "Password: ********"
    echo ""

    response=$(curl -s -X POST "$API_URL/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$email\", \"password\": \"$password\"}")

    echo "$response" | jq .

    if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
        print_success "User registered successfully!"
    else
        print_error "Registration failed"
    fi
}

# Login and get tokens
login() {
    local email="${1:-$USER_EMAIL}"
    local password="${2:-$USER_PASSWORD}"

    print_header "Logging In"
    print_info "Email: $email"
    print_info "Password: ********"
    echo ""

    response=$(curl -s -X POST "$API_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$email\", \"password\": \"$password\"}")

    echo "$response" | jq .

    # Extract tokens for later use
    ACCESS_TOKEN=$(echo "$response" | jq -r '.access_token // empty')
    REFRESH_TOKEN=$(echo "$response" | jq -r '.refresh_token // empty')

    if [ -n "$ACCESS_TOKEN" ]; then
        print_success "Login successful!"
        echo ""
        print_info "Access Token (first 50 chars): ${ACCESS_TOKEN:0:50}..."
        print_info "Token expires in: $(echo "$response" | jq -r '.expires_in') seconds"
    else
        print_error "Login failed"
    fi
}

# Access protected profile endpoint
profile() {
    print_header "Accessing Protected Profile Endpoint"

    if [ -z "$ACCESS_TOKEN" ]; then
        print_info "No token in memory, logging in first..."
        login > /dev/null 2>&1
    fi

    print_info "Using Bearer token for authentication"
    echo ""

    curl -s "$API_URL/profile" \
        -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
}

# Refresh tokens
refresh() {
    print_header "Refreshing Tokens"

    if [ -z "$REFRESH_TOKEN" ]; then
        print_info "No refresh token in memory, logging in first..."
        login > /dev/null 2>&1
    fi

    print_info "Using refresh token to get new tokens"
    echo ""

    response=$(curl -s -X POST "$API_URL/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

    echo "$response" | jq .

    # Update tokens
    new_access=$(echo "$response" | jq -r '.access_token // empty')
    new_refresh=$(echo "$response" | jq -r '.refresh_token // empty')

    if [ -n "$new_access" ]; then
        ACCESS_TOKEN="$new_access"
        REFRESH_TOKEN="$new_refresh"
        print_success "Tokens refreshed successfully!"
    else
        print_error "Token refresh failed"
    fi
}

# Try accessing protected route without token
no_token() {
    print_header "Accessing Protected Route WITHOUT Token"
    print_info "This should return 401 Unauthorized"
    echo ""

    response=$(curl -s -w "\n%{http_code}" "$API_URL/profile")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    echo "$body" | jq .
    echo ""

    if [ "$http_code" -eq 401 ]; then
        print_success "Correctly rejected with 401 Unauthorized!"
    else
        print_error "Unexpected response code: $http_code"
    fi
}

# Try accessing protected route with invalid token
invalid_token() {
    print_header "Accessing Protected Route with INVALID Token"
    print_info "This should return 401 Unauthorized"
    echo ""

    response=$(curl -s -w "\n%{http_code}" "$API_URL/profile" \
        -H "Authorization: Bearer invalid.token.here")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    echo "$body" | jq .
    echo ""

    if [ "$http_code" -eq 401 ]; then
        print_success "Correctly rejected invalid token with 401 Unauthorized!"
    else
        print_error "Unexpected response code: $http_code"
    fi
}

# Run full demo workflow
demo() {
    print_header "Starting JWT Authentication Demo"

    echo -e "${YELLOW}This demo will:${NC}"
    echo "  1. Check API health"
    echo "  2. Register a new user"
    echo "  3. Login and get tokens"
    echo "  4. Access protected profile endpoint"
    echo "  5. Refresh tokens"
    echo "  6. Try accessing without token (should fail)"
    echo "  7. Try accessing with invalid token (should fail)"
    echo ""

    # Generate unique email for this demo run
    TIMESTAMP=$(date +%s)
    USER_EMAIL="demo-${TIMESTAMP}@example.com"

    # Step 1: Health check
    health

    # Step 2: Register
    sleep 1
    register "$USER_EMAIL" "$USER_PASSWORD"

    # Step 3: Login
    sleep 1
    login "$USER_EMAIL" "$USER_PASSWORD"

    # Step 4: Access profile
    sleep 1
    profile

    # Step 5: Refresh tokens
    sleep 1
    refresh

    # Step 6: Verify new tokens work
    print_header "Verifying Refreshed Tokens Work"
    curl -s "$API_URL/profile" \
        -H "Authorization: Bearer $ACCESS_TOKEN" | jq .

    # Step 7: Try without token
    sleep 1
    no_token

    # Step 8: Try with invalid token
    sleep 1
    invalid_token

    # Summary
    print_header "Demo Complete!"
    echo -e "${GREEN}All authentication flows demonstrated successfully!${NC}"
    echo ""
    echo "Key takeaways:"
    echo "  - Passwords are securely hashed with bcrypt"
    echo "  - Access tokens expire after 15 minutes"
    echo "  - Refresh tokens can be used to get new access tokens"
    echo "  - Protected routes require valid Bearer tokens"
    echo "  - Invalid/missing tokens are properly rejected"
}

# Show usage
usage() {
    echo "JWT Authentication Demo Script"
    echo ""
    echo "Usage: $0 <command> [args...]"
    echo ""
    echo "Commands:"
    echo "  health                     - Check API health"
    echo "  register [email] [pass]    - Register a new user"
    echo "  login [email] [pass]       - Login and get tokens"
    echo "  profile                    - Access protected profile (requires login first)"
    echo "  refresh                    - Refresh access token (requires login first)"
    echo "  no-token                   - Try accessing protected route without token"
    echo "  invalid-token              - Try accessing protected route with invalid token"
    echo "  demo                       - Run full demo workflow (recommended)"
    echo ""
    echo "Environment Variables:"
    echo "  BASE_URL  - API base URL (default: http://localhost:3000)"
    echo ""
    echo "Examples:"
    echo "  $0 demo                                    - Run full demo"
    echo "  $0 register user@example.com mypassword   - Register a user"
    echo "  $0 login user@example.com mypassword      - Login"
    echo "  $0 profile                                - Get profile (after login)"
}

# Main
case "${1:-}" in
    health)
        health
        ;;
    register)
        register "$2" "$3"
        ;;
    login)
        login "$2" "$3"
        ;;
    profile)
        profile
        ;;
    refresh)
        refresh
        ;;
    no-token)
        no_token
        ;;
    invalid-token)
        invalid_token
        ;;
    demo)
        demo
        ;;
    *)
        usage
        ;;
esac
