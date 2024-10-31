#!/bin/bash

echo "Building for Linux..."

# For GNU Linux
GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-amd64
GOARCH=386 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-i386
GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-arm64
GOARCH=arm go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-arm

echo "Build completed successfully!"