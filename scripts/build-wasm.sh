#!/bin/bash
set -e

# Pin Go version
GO_VERSION=$(go version)
echo "Building with $GO_VERSION"

# Build WASM
echo "Building main.wasm..."
GOOS=js GOARCH=wasm go build -o web/public/main.wasm ./src/cmd/wasm

# Copy wasm_exec.js (ensuring it's up to date with the current Go version)
GOROOT=$(go env GOROOT)
echo "Looking for wasm_exec.js in $GOROOT..."

# Try multiple possible locations for wasm_exec.js
IF_FILE="$GOROOT/lib/wasm/wasm_exec.js"
if [ ! -f "$IF_FILE" ]; then
    IF_FILE="$GOROOT/misc/wasm/wasm_exec.js"
fi

if [ -f "$IF_FILE" ]; then
    echo "Copying wasm_exec.js from $IF_FILE..."
    cp "$IF_FILE" web/public/
else
    echo "Error: could not find wasm_exec.js"
    exit 1
fi

echo "WASM build complete: web/public/main.wasm and web/wasm_exec.js updated."
