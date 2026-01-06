# WASM Bridge Package

Exposes Go encoders to JavaScript via WebAssembly syscall/js bridge.

## Package Identity

- **Purpose**: Bridge between Go compression and JavaScript/Web UI
- **Technology**: Go with `//go:build js && wasm` build tag
- **Tests**: Integration tests with JS interop

## Setup & Run

```bash
# Run WASM bridge tests
go test ./src/wasm/...

# Build for WASM
GOOS=js GOARCH=wasm go build -o /tmp/test.wasm ./src/cmd/wasm

# Test with node (if needed)
# Note: Requires wasm_exec.js

# Lint
golangci-lint run ./src/wasm/...
```

## Patterns & Conventions

**File Organization**:
```
src/wasm/
├── bridge.go       # Main exports (EncodePng, EncodeJpeg)
└── *_test.go      # Tests
```

**Key Patterns**:
- ✅ DO: Use `//export` for functions callable from JS
- ✅ DO: Return `[]byte` (auto-converted to Uint8Array)
- ✅ DO: Return `error` as second value (auto-converted to JS Error)
- ❌ DON'T: Use Go channels or pointers (not serializable)
- ❌ DON'T: Return complex Go structs (use simple types)

**Function Signatures** (see `src/wasm/bridge.go:11-89`):
```go
// Simple: returns []byte
//export EncodePng
func EncodePng(pixels []byte, width, height, colorType, preset int, lossy bool, maxColors int) []byte

// Advanced: returns (result, error)
//export EncodeJpegAdvanced
func EncodeJpegAdvanced(...) ([]byte, error)
```

**Integration with Web**:
- ✅ DO: Use `worker.ts` for off-main-thread compression
- ✅ DO: Convert RGBA→RGB for JPEG (JPEG has no alpha)
- ✅ DO: Handle large images with progress callbacks
- ❌ DON'T: Block main thread with synchronous WASM calls

## Touch Points / Key Files

- **Main Bridge**: `src/wasm/bridge.go:13-89` (exported functions)
- **WASM Entry**: `src/cmd/wasm/main.go` (build target)
- **Web Worker**: `web/src/worker.ts` (consumes WASM exports)

## JIT Index Hints

```bash
# Find exported functions
rg -n "//export" src/wasm/*.go

# Find JS interop
rg -n "syscall/js" src/wasm/*.go

# Find test files
ls src/wasm/*_test.go
```

## Common Gotchas

- **Build Tag**: Must use `//go:build js && wasm` (not `// +build js wasm`)
- **Array Conversion**: Go `[]byte` → JS `Uint8Array` (zero-copy)
- **Error Handling**: Return `(result, error)` tuple, never panic
- **Memory**: WASM has limited memory, avoid large allocations
- **Callbacks**: Use `js.FuncOf` for JS callbacks (progress reporting)

## Pre-PR Checks

```bash
go test ./src/wasm/... && golangci-lint run ./src/wasm/...
```
