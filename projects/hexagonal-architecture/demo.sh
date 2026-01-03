#!/bin/bash
# Demo script for Hexagonal Architecture Task API
# Usage: ./demo.sh [command] [args...]

set -e

BASE_URL="${BASE_URL:-http://localhost:3000}"
API_URL="$BASE_URL/api/v1"

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

# Health check
health() {
    print_header "Health Check"
    curl -s "$BASE_URL/health" | jq .
}

# Create a task
create_task() {
    local title="${1:-My Task}"
    local description="${2:-Task description}"
    local user_id="${3:-user-1}"

    print_header "Creating Task"
    print_info "Title: $title"
    print_info "Description: $description"
    print_info "User ID: $user_id"
    echo ""

    curl -s -X POST "$API_URL/tasks" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"$title\",
            \"description\": \"$description\",
            \"user_id\": \"$user_id\"
        }" | jq .
}

# List all tasks
list_tasks() {
    local user_id="$1"

    print_header "Listing Tasks"

    if [ -n "$user_id" ]; then
        print_info "Filtering by user: $user_id"
        curl -s "$API_URL/tasks?user_id=$user_id" | jq .
    else
        curl -s "$API_URL/tasks" | jq .
    fi
}

# Get a task by ID
get_task() {
    local task_id="$1"

    if [ -z "$task_id" ]; then
        echo -e "${RED}Error: task_id is required${NC}"
        echo "Usage: $0 get <task_id>"
        exit 1
    fi

    print_header "Getting Task: $task_id"
    curl -s "$API_URL/tasks/$task_id" | jq .
}

# Update a task
update_task() {
    local task_id="$1"
    local title="$2"
    local description="$3"

    if [ -z "$task_id" ]; then
        echo -e "${RED}Error: task_id is required${NC}"
        echo "Usage: $0 update <task_id> [title] [description]"
        exit 1
    fi

    print_header "Updating Task: $task_id"

    # Build JSON payload
    local payload="{}"
    if [ -n "$title" ]; then
        payload=$(echo "$payload" | jq --arg t "$title" '. + {title: $t}')
    fi
    if [ -n "$description" ]; then
        payload=$(echo "$payload" | jq --arg d "$description" '. + {description: $d}')
    fi

    print_info "Payload: $payload"
    echo ""

    curl -s -X PUT "$API_URL/tasks/$task_id" \
        -H "Content-Type: application/json" \
        -d "$payload" | jq .
}

# Complete a task
complete_task() {
    local task_id="$1"

    if [ -z "$task_id" ]; then
        echo -e "${RED}Error: task_id is required${NC}"
        echo "Usage: $0 complete <task_id>"
        exit 1
    fi

    print_header "Completing Task: $task_id"
    curl -s -X POST "$API_URL/tasks/$task_id/complete" | jq .
}

# Delete a task
delete_task() {
    local task_id="$1"

    if [ -z "$task_id" ]; then
        echo -e "${RED}Error: task_id is required${NC}"
        echo "Usage: $0 delete <task_id>"
        exit 1
    fi

    print_header "Deleting Task: $task_id"
    curl -s -X DELETE "$API_URL/tasks/$task_id" | jq .
}

# Run full demo workflow
demo() {
    print_header "Starting Full Demo Workflow"

    echo -e "${YELLOW}Demo Users Available:${NC}"
    echo "  - user-1: Alice Johnson"
    echo "  - user-2: Bob Smith"
    echo "  - user-3: Charlie Brown"

    # Health check
    health

    # Create tasks for different users
    print_header "Creating Tasks for Demo"

    echo -e "${GREEN}Creating task for Alice (user-1)...${NC}"
    TASK1=$(curl -s -X POST "$API_URL/tasks" \
        -H "Content-Type: application/json" \
        -d '{"title": "Buy groceries", "description": "Milk, eggs, bread", "user_id": "user-1"}')
    TASK1_ID=$(echo "$TASK1" | jq -r '.id')
    echo "$TASK1" | jq .

    echo -e "\n${GREEN}Creating task for Bob (user-2)...${NC}"
    TASK2=$(curl -s -X POST "$API_URL/tasks" \
        -H "Content-Type: application/json" \
        -d '{"title": "Finish report", "description": "Q4 financial report", "user_id": "user-2"}')
    TASK2_ID=$(echo "$TASK2" | jq -r '.id')
    echo "$TASK2" | jq .

    echo -e "\n${GREEN}Creating another task for Alice (user-1)...${NC}"
    TASK3=$(curl -s -X POST "$API_URL/tasks" \
        -H "Content-Type: application/json" \
        -d '{"title": "Call dentist", "description": "Schedule appointment", "user_id": "user-1"}')
    TASK3_ID=$(echo "$TASK3" | jq -r '.id')
    echo "$TASK3" | jq .

    # List all tasks
    print_header "All Tasks"
    curl -s "$API_URL/tasks" | jq .

    # List tasks for user-1
    print_header "Tasks for Alice (user-1)"
    curl -s "$API_URL/tasks?user_id=user-1" | jq .

    # Get specific task
    print_header "Get Task Details: $TASK1_ID"
    curl -s "$API_URL/tasks/$TASK1_ID" | jq .

    # Update task
    print_header "Updating Task: $TASK1_ID"
    curl -s -X PUT "$API_URL/tasks/$TASK1_ID" \
        -H "Content-Type: application/json" \
        -d '{"title": "Buy groceries (URGENT)", "description": "Milk, eggs, bread, cheese"}' | jq .

    # Complete task
    print_header "Completing Task: $TASK2_ID"
    curl -s -X POST "$API_URL/tasks/$TASK2_ID/complete" | jq .

    # Delete task
    print_header "Deleting Task: $TASK3_ID"
    curl -s -X DELETE "$API_URL/tasks/$TASK3_ID" | jq .

    # Final state
    print_header "Final State - All Tasks"
    curl -s "$API_URL/tasks" | jq .

    print_success "\nDemo completed!"
    echo -e "${YELLOW}Check server logs for notification events triggered by task operations.${NC}"
}

# Show usage
usage() {
    echo "Hexagonal Architecture Task API Demo Script"
    echo ""
    echo "Usage: $0 <command> [args...]"
    echo ""
    echo "Commands:"
    echo "  health                           - Check API health"
    echo "  create <title> <desc> <user_id>  - Create a new task"
    echo "  list [user_id]                   - List all tasks (optionally filter by user)"
    echo "  get <task_id>                    - Get a task by ID"
    echo "  update <task_id> [title] [desc]  - Update a task"
    echo "  complete <task_id>               - Mark task as completed"
    echo "  delete <task_id>                 - Delete a task"
    echo "  demo                             - Run full demo workflow"
    echo ""
    echo "Environment Variables:"
    echo "  BASE_URL  - API base URL (default: http://localhost:3000)"
    echo ""
    echo "Examples:"
    echo "  $0 demo                          - Run full demo"
    echo "  $0 health                        - Check health"
    echo "  $0 create \"My Task\" \"Description\" user-1"
    echo "  $0 list                          - List all tasks"
    echo "  $0 list user-1                   - List tasks for user-1"
    echo "  $0 get abc-123                   - Get task by ID"
    echo "  $0 complete abc-123              - Complete a task"
    echo "  $0 delete abc-123                - Delete a task"
}

# Main
case "${1:-}" in
    health)
        health
        ;;
    create)
        create_task "$2" "$3" "$4"
        ;;
    list)
        list_tasks "$2"
        ;;
    get)
        get_task "$2"
        ;;
    update)
        update_task "$2" "$3" "$4"
        ;;
    complete)
        complete_task "$2"
        ;;
    delete)
        delete_task "$2"
        ;;
    demo)
        demo
        ;;
    *)
        usage
        ;;
esac
