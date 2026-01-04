go-pixo: Go → WASM PNG compression, client-side only (no upload/API).

## Workflow

**Always follow this sequence for every task:**

1. Implement & code the feature/fix
2. Test Go: `go test ./...` (must pass)
3. Lint Go: `golangci-lint run` (must pass, no warnings)
4. Lint Web: `cd web && rescript build 2>&1 | grep -qi "Warning" && exit 1 || exit 0` (must pass, no warnings)
5. Commit changes

## Commands

```bash
# Testing Go
go test ./...                       # All Go tests
go test -run TestFunc ./src/pkg     # Single Go test

# Formatting & Linting Go
go fmt ./...                        # Format Go code
golangci-lint run                   # Comprehensive Go linting (no warnings allowed)

# Pipeline Go (test then lint)
go test ./... && golangci-lint run  # Full Go validation

# Building
./scripts/build-wasm.sh             # Build WASM

# Web UI
cd web && bun run dev               # Web UI dev

# Web UI Linting
cd web && rescript build 2>&1 | grep -qi "Warning" && echo "Warnings found!" && exit 1 || echo "No warnings"

# Combined Lint (all code)
golangci-lint run && cd web && rescript build 2>&1 | grep -qi "Warning" && exit 1 || exit 0
```

## Code Style

**Imports**: std lib first, then local (full module path: `github.com/mac/go-pixo/src/...`)
**Naming**: Exported PascalCase, private camelCase. Constants: Exported PascalCase, private camelCase
**Error handling**: Return `error` as second value, never suppress
**Testing**: Table-driven with `t.Run`, descriptive names
**WASM code**: Use `//go:build js && wasm` build tag
**Comments**: Godoc on exported functions
**Rescript**: No warnings allowed, fix unused imports, shadowing, fragile patterns
**Linting**: Both Go and Web must pass without warnings

## Architecture

`png/` → PNG encoder, `compress/` → DEFLATE/zlib/CRC32, `wasm/` → syscall/js bridge, `cmd/wasm/` → WASM entrypoint, `web/` → Vite+TS+Tailwind UI
