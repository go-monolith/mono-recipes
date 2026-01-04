#!/bin/bash

# Script to create a new branch from the latest main branch
# Usage: ./scripts/new-branch.sh <branch-name>

set -e

# Check if branch name is provided
if [ -z "$1" ]; then
    echo "Error: Branch name is required"
    echo "Usage: $0 <branch-name>"
    exit 1
fi

BRANCH_NAME="$1"
MAIN_BRANCH="main"

echo "Fetching latest changes from remote..."
git fetch origin

echo "Switching to $MAIN_BRANCH branch..."
git checkout "$MAIN_BRANCH"

echo "Pulling latest changes from remote $MAIN_BRANCH..."
git pull origin "$MAIN_BRANCH"

echo "Creating new branch: $BRANCH_NAME"
git checkout -b "$BRANCH_NAME"

echo "✓ Successfully created and switched to branch: $BRANCH_NAME"
echo "Branch is based on the latest $MAIN_BRANCH from remote"
