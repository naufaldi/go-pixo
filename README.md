# go-pixo

Client-side PNG compression in **standard Go**, runnable in the browser via **Go → WebAssembly (WASM)**.

## What this repo is

- A new **Go rewrite** inspired by the Rust project `pixo` (MIT licensed).
- Target UX: “drag/drop → compress → download”, **no upload**, **no API**, no server required.

## Setup & Development

### Requirements

- Go 1.25.5+ (standard Go WASM support)
- Bun (for web frontend)

### Installation

1. Initialize Go module (done):
   ```bash
   go mod init github.com/mac/go-pixo
   ```

2. Install web dependencies:
   ```bash
   cd web
   bun install
   ```

### Building WASM

Use the provided build script to compile the Go code to WebAssembly and copy the required `wasm_exec.js`:

```bash
./scripts/build-wasm.sh
```

### Running the Web UI

```bash
cd web
bun run dev
```

## Project Structure

- `src/` — Go source code
  - `cmd/wasm/` — WASM entrypoint
  - `wasm/` — JS bridge via `syscall/js`
  - `png/` — PNG encoder implementation (Phase 1)
  - `compress/` — DEFLATE/zlib implementation (Phase 2)
- `web/` — Vite + TypeScript + Tailwind CSS v4 frontend
- `scripts/` — Build and helper scripts

## Attribution

This repository started by copying documentation and project scaffolding from the upstream `pixo` repo as reference material.

- Upstream project: `pixo` (Rust) — https://github.com/leerob/pixo
- License: MIT (see `LICENSE`)

## License

MIT
