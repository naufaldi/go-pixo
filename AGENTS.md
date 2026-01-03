go-pixo: Go → WASM PNG compression, client-side only (no upload/API).

## Workflow
**Always follow this sequence for every task:**
1. Implement & code the feature/fix
2. Test: `go test ./...` (must pass)
3. Lint: `golangci-lint run` (must pass)
4. Commit changes

## Commands
```bash
# Testing
go test ./...                       # All tests
go test -run TestFunc ./src/pkg     # Single test

# Formatting & Linting
go fmt ./...                        # Format code
golangci-lint run                   # Comprehensive linting (replaces go vet)

# Pipeline (test then lint)
go test ./... && golangci-lint run  # Full validation pipeline

# Building
./scripts/build-wasm.sh             # Build WASM

# Web UI
cd web && bun run dev               # Web UI dev
```

## Code Style
**Imports**: std lib first, then local (full module path: `github.com/mac/go-pixo/src/...`)
**Naming**: Exported PascalCase, private camelCase. Constants: Exported PascalCase, private camelCase
**Error handling**: Return `error` as second value, never suppress
**Testing**: Table-driven with `t.Run`, descriptive names
**WASM code**: Use `//go:build js && wasm` build tag
**Comments**: Godoc on exported functions
**Linting**: Enforced by `.golangci.yml` (v2.7.2 compatible with Go 1.25.5)

## Architecture
`png/` → PNG encoder, `compress/` → DEFLATE/zlib/CRC32, `wasm/` → syscall/js bridge, `cmd/wasm/` → WASM entrypoint, `web/` → Vite+TS+Tailwind UI
