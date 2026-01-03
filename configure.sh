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

echo "This script sets up all system config/files required to run the project"