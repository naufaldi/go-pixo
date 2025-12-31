go-pixo is a **standard Go + WASM** rewrite project (PNG-only MVP) inspired by the Rust project `pixo`.
The goal is a **client-only** web tool: users compress images locally and download results (no upload/API).

## Commands (planned)

```bash
go test ./...                       # Run all tests
go fmt ./...                        # Format
go vet ./...                        # Basic lint
```

### Building WASM (planned)

Standard Go can compile to WASM:

```bash
GOOS=js GOARCH=wasm go build -o web/main.wasm ./cmd/wasm
```

You’ll also need to copy Go’s `wasm_exec.js` into `web/` (pin a Go version and document it).

## Architecture (target)

- **`png/`** - PNG encoder (filters, chunks, bit depth later)
- **`compress/`** - DEFLATE + zlib wrapper + CRC32/Adler32
- **`wasm/`** - Go WASM bridge (`syscall/js` exports)
- **`cmd/wasm/`** - WASM entrypoint
- **`web/`** - Demo web UI

## Reference docs

The `docs/` folder currently includes Rust-oriented reference material copied from upstream `pixo`.
The Go rewrite timeline is in `docs/brain/go-pixo-standard-go-wasm-rewrite-plan.md`.
