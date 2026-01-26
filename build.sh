#!/bin/bash

# tinyMem Build Script
# Builds all platform binaries and places them in the releases folder

set -e  # Exit immediately if a command exits with a non-zero status

# Create releases directory if it doesn't exist
mkdir -p releases

echo "Building tinyMem binaries..."

# Build macOS binaries
echo "Building macOS ARM64..."
GOOS=darwin GOARCH=arm64 go build -o releases/tinymem-darwin-arm64 ./cmd/tinymem
echo "✓ Built releases/tinymem-darwin-arm64"

echo "Building macOS AMD64..."
GOOS=darwin GOARCH=amd64 go build -o releases/tinymem-darwin-amd64 ./cmd/tinymem
echo "✓ Built releases/tinymem-darwin-amd64"

# Build Windows binaries
echo "Building Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -o releases/tinymem-windows-amd64.exe ./cmd/tinymem
echo "✓ Built releases/tinymem-windows-amd64.exe"

echo "Building Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -o releases/tinymem-windows-arm64.exe ./cmd/tinymem
echo "✓ Built releases/tinymem-windows-arm64.exe"

# Build Linux binaries
echo "Building Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -o releases/tinymem-linux-amd64 ./cmd/tinymem
echo "✓ Built releases/tinymem-linux-amd64"

echo "Building Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -o releases/tinymem-linux-arm64 ./cmd/tinymem
echo "✓ Built releases/tinymem-linux-arm64"

echo ""
echo "Build completed successfully!"
echo ""
echo "Binaries created in releases/:"
ls -la releases/
echo ""
echo "Build script completed."