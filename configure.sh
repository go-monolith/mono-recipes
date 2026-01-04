#!/bin/bash

# set up env files from default file
[ -f ".env" ] || cp default.env .env

# TODO: add more setup steps here

# check and install NATS cli if not installed
if ! command -v nats &> /dev/null
then
    echo "NATS CLI not found, installing..."
    go install github.com/nats-io/natscli/nats@latest
    echo "NATS CLI installed."
fi

# Install Go development tools (from mono-framework make install)
echo "Installing Go development tools..."

# Install golangci-lint v2.7.2
if ! command -v golangci-lint &> /dev/null || [[ "$(golangci-lint --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')" != "2.7.2" ]]
then
    echo "Installing golangci-lint v2.7.2..."
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.7.2
    echo "golangci-lint installed."
else
    echo "golangci-lint v2.7.2 already installed."
fi

# Install goimports
if ! command -v goimports &> /dev/null
then
    echo "Installing goimports..."
    go install golang.org/x/tools/cmd/goimports@latest
    echo "goimports installed."
else
    echo "goimports already installed."
fi

echo "✓ Development tools installed successfully"

echo "This script sets up all system config/files required to run the project"