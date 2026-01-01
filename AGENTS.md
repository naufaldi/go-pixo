go-pixo: Go → WASM PNG compression, client-side only (no upload/API).

## Commands
```bash
go test ./...                       # All tests
go test -run TestFunc ./src/pkg     # Single test
go fmt ./...                        # Format
go vet ./...                        # Lint
./scripts/build-wasm.sh             # Build WASM
cd web && bun run dev               # Web UI dev
```

## Code Style
**Imports**: std lib first, then local (full module path: `github.com/mac/go-pixo/src/...`)
**Naming**: Exported PascalCase, private camelCase. Constants: Exported PascalCase, private camelCase
**Error handling**: Return `error` as second value, never suppress
**Testing**: Table-driven with `t.Run`, descriptive names
**WASM code**: Use `//go:build js && wasm` build tag
**Comments**: Godoc on exported functions

## Architecture
`png/` → PNG encoder, `compress/` → DEFLATE/zlib/CRC32, `wasm/` → syscall/js bridge, `cmd/wasm/` → WASM entrypoint, `web/` → Vite+TS+Tailwind UI
