#!/bin/bash

# File Upload Demo Script
# Demonstrates file upload, download, listing, and deletion

set -e

BASE_URL="${BASE_URL:-http://localhost:3000}"
DEMO_DIR=$(mktemp -d)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Cleanup on exit
cleanup() {
    rm -rf "$DEMO_DIR"
}
trap cleanup EXIT

# Print colored output
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Check if the server is running
check_server() {
    print_header "Checking Server Health"

    response=$(curl -s "$BASE_URL/health" || echo "FAILED")

    if echo "$response" | grep -q "healthy"; then
        print_success "Server is running"
        echo "$response" | jq .
    else
        print_error "Server is not running at $BASE_URL"
        echo "Please start the server first: go run main.go"
        exit 1
    fi
}

# Create sample files for testing
create_sample_files() {
    print_header "Creating Sample Files"

    # Create a text file
    echo "Hello, this is a sample text file for the file upload demo." > "$DEMO_DIR/sample.txt"
    print_success "Created sample.txt"

    # Create a JSON file
    cat > "$DEMO_DIR/config.json" << 'EOF'
{
    "name": "File Upload Demo",
    "version": "1.0.0",
    "features": ["upload", "download", "list", "delete"]
}
EOF
    print_success "Created config.json"

    # Create a larger file
    for i in {1..100}; do
        echo "Line $i: This is some content to make the file larger for testing purposes."
    done > "$DEMO_DIR/large-file.txt"
    print_success "Created large-file.txt ($(wc -c < "$DEMO_DIR/large-file.txt") bytes)"
}

# Upload a single file
upload_single_file() {
    print_header "Uploading Single File"

    print_info "Uploading sample.txt..."
    response=$(curl -s -X POST "$BASE_URL/api/v1/files" \
        -F "file=@$DEMO_DIR/sample.txt")

    echo "$response" | jq .

    # Extract file ID for later use
    FILE_ID_1=$(echo "$response" | jq -r '.file.id')
    if [ "$FILE_ID_1" != "null" ] && [ -n "$FILE_ID_1" ]; then
        print_success "Uploaded sample.txt with ID: $FILE_ID_1"
    else
        print_error "Failed to upload sample.txt"
        return 1
    fi
}

# Upload multiple files
upload_multiple_files() {
    print_header "Uploading Multiple Files (Batch)"

    print_info "Uploading config.json and large-file.txt..."
    response=$(curl -s -X POST "$BASE_URL/api/v1/files/batch" \
        -F "files=@$DEMO_DIR/config.json" \
        -F "files=@$DEMO_DIR/large-file.txt")

    echo "$response" | jq .

    count=$(echo "$response" | jq '.count')
    if [ "$count" = "2" ]; then
        print_success "Uploaded 2 files successfully"

        # Extract file IDs for later use
        FILE_ID_2=$(echo "$response" | jq -r '.uploaded[0].file.id')
        FILE_ID_3=$(echo "$response" | jq -r '.uploaded[1].file.id')
    else
        print_error "Expected 2 files, got $count"
    fi
}

# List all files
list_files() {
    print_header "Listing All Files"

    print_info "Fetching file list..."
    response=$(curl -s "$BASE_URL/api/v1/files")

    echo "$response" | jq .

    total=$(echo "$response" | jq '.total')
    print_success "Total files: $total"
}

# Get file info
get_file_info() {
    print_header "Getting File Metadata"

    if [ -z "$FILE_ID_1" ]; then
        print_error "No file ID available"
        return 1
    fi

    print_info "Getting info for file: $FILE_ID_1"
    response=$(curl -s "$BASE_URL/api/v1/files/$FILE_ID_1/info")

    echo "$response" | jq .
    print_success "Retrieved file metadata"
}

# Download a file
download_file() {
    print_header "Downloading File"

    if [ -z "$FILE_ID_1" ]; then
        print_error "No file ID available"
        return 1
    fi

    print_info "Downloading file: $FILE_ID_1"

    # Download to temp file
    output_file="$DEMO_DIR/downloaded_sample.txt"
    curl -s -o "$output_file" "$BASE_URL/api/v1/files/$FILE_ID_1"

    if [ -f "$output_file" ]; then
        print_success "File downloaded to: $output_file"
        echo -e "${YELLOW}Content:${NC}"
        cat "$output_file"
        echo ""

        # Verify content matches
        if diff -q "$DEMO_DIR/sample.txt" "$output_file" > /dev/null 2>&1; then
            print_success "Downloaded content matches original!"
        else
            print_error "Downloaded content differs from original"
        fi
    else
        print_error "Failed to download file"
    fi
}

# Delete a file
delete_file() {
    print_header "Deleting File"

    if [ -z "$FILE_ID_1" ]; then
        print_error "No file ID available"
        return 1
    fi

    print_info "Deleting file: $FILE_ID_1"
    response=$(curl -s -X DELETE "$BASE_URL/api/v1/files/$FILE_ID_1")

    echo "$response" | jq .

    if echo "$response" | grep -q "successfully"; then
        print_success "File deleted successfully"
    else
        print_error "Failed to delete file"
    fi
}

# Verify file was deleted
verify_deletion() {
    print_header "Verifying File Deletion"

    if [ -z "$FILE_ID_1" ]; then
        print_error "No file ID available"
        return 1
    fi

    print_info "Attempting to access deleted file: $FILE_ID_1"
    response=$(curl -s "$BASE_URL/api/v1/files/$FILE_ID_1/info")

    echo "$response" | jq .

    if echo "$response" | grep -q "not found"; then
        print_success "File correctly returns 404 (not found)"
    else
        print_error "File should not exist after deletion"
    fi
}

# Final listing
final_list() {
    print_header "Final File List"

    print_info "Listing remaining files..."
    response=$(curl -s "$BASE_URL/api/v1/files")

    echo "$response" | jq .

    total=$(echo "$response" | jq '.total')
    print_success "Remaining files: $total"
}

# Main demo flow
main() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════╗"
    echo "║          File Upload Demo - fs-jetstream Plugin            ║"
    echo "║                                                            ║"
    echo "║  Demonstrating file operations with Mono framework         ║"
    echo "║  and embedded JetStream storage                            ║"
    echo "╚════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"

    # Run demo steps
    check_server
    create_sample_files

    echo ""
    read -p "Press Enter to start the demo..."

    upload_single_file

    echo ""
    read -p "Press Enter to continue with batch upload..."

    upload_multiple_files

    echo ""
    read -p "Press Enter to list all files..."

    list_files

    echo ""
    read -p "Press Enter to get file metadata..."

    get_file_info

    echo ""
    read -p "Press Enter to download a file..."

    download_file

    echo ""
    read -p "Press Enter to delete a file..."

    delete_file

    echo ""
    read -p "Press Enter to verify deletion..."

    verify_deletion

    echo ""
    read -p "Press Enter to see final file list..."

    final_list

    print_header "Demo Complete"
    echo -e "${GREEN}"
    echo "The demo has completed successfully!"
    echo ""
    echo "Key takeaways:"
    echo "  • Files are stored in JetStream object store"
    echo "  • Each file gets a unique UUID identifier"
    echo "  • Metadata (content-type, size, digest) is tracked"
    echo "  • No external NATS server required - uses embedded NATS"
    echo -e "${NC}"
}

# Run the demo
main "$@"
