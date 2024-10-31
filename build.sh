#!/bin/bash

echo "Building for Linux..."

# For GNU Linux
GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-amd64
GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-arm64

echo "Build completed successfully!"