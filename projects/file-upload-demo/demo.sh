#!/bin/bash

# File Upload Demo - Interactive Demo Script
# This script demonstrates the File Upload API with Gin + JetStream Object Store

set -e

BASE_URL="${BASE_URL:-http://localhost:3000}"
TEMP_DIR=$(mktemp -d)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

print_step() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

wait_for_server() {
    echo "Waiting for server to be ready..."
    for i in {1..30}; do
        if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
            print_success "Server is ready!"
            return 0
        fi
        sleep 1
    done
    print_error "Server did not become ready in time"
    exit 1
}

# ============================================================================
# Demo Start
# ============================================================================

echo -e "${GREEN}"
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║       File Upload Demo - Gin + JetStream Object Store         ║"
echo "╠═══════════════════════════════════════════════════════════════╣"
echo "║  This demo showcases:                                         ║"
echo "║  • File upload via multipart/form-data                        ║"
echo "║  • File upload via JSON with base64 encoding                  ║"
echo "║  • File listing with pagination                               ║"
echo "║  • File download                                              ║"
echo "║  • File deletion                                              ║"
echo "║  • NATS JetStream Object Store as storage backend             ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

wait_for_server

# ============================================================================
# Step 1: Health Check
# ============================================================================

print_step "Step 1: Health Check"
echo "Command: curl $BASE_URL/health"
echo ""
curl -s "$BASE_URL/health" | jq .
print_success "Health check passed"

# ============================================================================
# Step 2: Upload a text file using multipart/form-data
# ============================================================================

print_step "Step 2: Upload a text file using multipart/form-data"

# Create a sample text file
echo "Hello, World! This is a test file uploaded via multipart/form-data." > "$TEMP_DIR/hello.txt"
echo "Content of hello.txt:"
cat "$TEMP_DIR/hello.txt"
echo ""

echo "Command: curl -X POST -F 'file=@hello.txt' $BASE_URL/api/v1/files"
echo ""

UPLOAD_RESPONSE=$(curl -s -X POST \
    -F "file=@$TEMP_DIR/hello.txt" \
    "$BASE_URL/api/v1/files")
echo "$UPLOAD_RESPONSE" | jq .
FILE_ID_1=$(echo "$UPLOAD_RESPONSE" | jq -r '.id')
print_success "Text file uploaded with ID: $FILE_ID_1"

# ============================================================================
# Step 3: Upload a JSON file using base64 encoding
# ============================================================================

print_step "Step 3: Upload a JSON file using base64 encoding"

# Create sample JSON content
JSON_CONTENT='{"name":"John Doe","email":"john@example.com","age":30}'
BASE64_DATA=$(echo -n "$JSON_CONTENT" | base64)

echo "Original JSON content:"
echo "$JSON_CONTENT" | jq .
echo ""
echo "Base64 encoded: ${BASE64_DATA:0:50}..."
echo ""

echo "Command: curl -X POST -H 'Content-Type: application/json' -d '{...}' $BASE_URL/api/v1/files"
echo ""

UPLOAD_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"user-data.json\",\"data\":\"$BASE64_DATA\",\"content_type\":\"application/json\"}" \
    "$BASE_URL/api/v1/files")
echo "$UPLOAD_RESPONSE" | jq .
FILE_ID_2=$(echo "$UPLOAD_RESPONSE" | jq -r '.id')
print_success "JSON file uploaded with ID: $FILE_ID_2"

# ============================================================================
# Step 4: Upload a binary file (image simulation)
# ============================================================================

print_step "Step 4: Upload a binary file"

# Create a small binary file (simulated PNG header)
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00' > "$TEMP_DIR/sample.png"

echo "Created sample binary file: sample.png (simulated PNG)"
echo ""
echo "Command: curl -X POST -F 'file=@sample.png' $BASE_URL/api/v1/files"
echo ""

UPLOAD_RESPONSE=$(curl -s -X POST \
    -F "file=@$TEMP_DIR/sample.png" \
    "$BASE_URL/api/v1/files")
echo "$UPLOAD_RESPONSE" | jq .
FILE_ID_3=$(echo "$UPLOAD_RESPONSE" | jq -r '.id')
print_success "Binary file uploaded with ID: $FILE_ID_3"

# ============================================================================
# Step 5: List all files
# ============================================================================

print_step "Step 5: List all files"
echo "Command: curl $BASE_URL/api/v1/files"
echo ""
curl -s "$BASE_URL/api/v1/files" | jq .
print_success "Listed all files"

# ============================================================================
# Step 6: Get file metadata
# ============================================================================

print_step "Step 6: Get file metadata"
echo "Command: curl $BASE_URL/api/v1/files/$FILE_ID_1"
echo ""
curl -s "$BASE_URL/api/v1/files/$FILE_ID_1" | jq .
print_success "Retrieved file metadata"

# ============================================================================
# Step 7: Download a file
# ============================================================================

print_step "Step 7: Download a file"
echo "Command: curl $BASE_URL/api/v1/files/$FILE_ID_1/download"
echo ""
echo "Downloaded content:"
curl -s "$BASE_URL/api/v1/files/$FILE_ID_1/download"
echo ""
print_success "File downloaded successfully"

# ============================================================================
# Step 8: List files with pagination
# ============================================================================

print_step "Step 8: List files with pagination"
echo "Command: curl '$BASE_URL/api/v1/files?limit=2&offset=0'"
echo ""
curl -s "$BASE_URL/api/v1/files?limit=2&offset=0" | jq .
print_success "Listed files with pagination"

# ============================================================================
# Step 9: Delete a file
# ============================================================================

print_step "Step 9: Delete a file"
echo "Command: curl -X DELETE $BASE_URL/api/v1/files/$FILE_ID_3"
echo ""
curl -s -X DELETE "$BASE_URL/api/v1/files/$FILE_ID_3" | jq .
print_success "File deleted"

# ============================================================================
# Step 10: Verify deletion
# ============================================================================

print_step "Step 10: Verify deletion (file should not exist)"
echo "Command: curl $BASE_URL/api/v1/files/$FILE_ID_3"
echo ""
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/files/$FILE_ID_3")
if [ "$HTTP_CODE" = "404" ]; then
    echo '{"error":"not_found","message":"File not found"}'
    print_success "File correctly returns 404 (not found)"
else
    curl -s "$BASE_URL/api/v1/files/$FILE_ID_3" | jq .
    print_error "Expected 404, got $HTTP_CODE"
fi

# ============================================================================
# Step 11: Final file list
# ============================================================================

print_step "Step 11: Final file list (should show 2 files)"
echo "Command: curl $BASE_URL/api/v1/files"
echo ""
FINAL_LIST=$(curl -s "$BASE_URL/api/v1/files")
echo "$FINAL_LIST" | jq .
TOTAL=$(echo "$FINAL_LIST" | jq -r '.total')
print_success "Final count: $TOTAL files remaining"

# ============================================================================
# Demo Complete
# ============================================================================

echo -e "\n${GREEN}"
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                    Demo Complete!                             ║"
echo "╠═══════════════════════════════════════════════════════════════╣"
echo "║  All file operations demonstrated successfully:               ║"
echo "║  ✓ Upload via multipart/form-data                             ║"
echo "║  ✓ Upload via JSON with base64                                ║"
echo "║  ✓ List files with pagination                                 ║"
echo "║  ✓ Get file metadata                                          ║"
echo "║  ✓ Download file content                                      ║"
echo "║  ✓ Delete file                                                ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo -e "${NC}"
