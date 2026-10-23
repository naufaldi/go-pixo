# go-pixo

Private image compression and conversion in the browser. PNG/JPEG use Go → WebAssembly (WASM); modern codecs run locally in browser workers. Images are never uploaded.

## What this repo is

- A new **Go rewrite** inspired by the Rust project `pixo` (MIT licensed).
- Target UX: “drag/drop → compress → download”, **no upload**, **no API**, no server required.

## Setup & Development

### Requirements

- Go 1.25.5+ (standard Go WASM support)
- Bun (for web frontend)

### Installation

Install the locked web dependencies:
```bash
cd web
bun install --frozen-lockfile
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

The dev command builds Go WASM, then starts ReScript watch and Vite. Development does not register an offline service worker.

### Running locally with offline support

From the repository root:

```bash
./scripts/build-wasm.sh
cd web
bun run build
bunx vite preview --host 127.0.0.1 --port 4173 --strictPort
```

Open `http://127.0.0.1:4173` online and wait for **Ready offline** before disconnecting. The production build precaches the complete emitted asset list, including Go WASM and lazy codec JS/WASM. Installation verifies each asset's content hash; one missing or mismatched file prevents readiness. Browser installation is optional, using the browser's install/add-to-home-screen menu.

Offline support requires HTTPS or localhost, service workers, Cache Storage and available browser storage. A first-ever visit without a connection cannot work. Clearing site data or browser storage eviction can remove offline files; reconnect to prepare again. Selected files and results stay in memory and are lost on reload. They are never put in the app cache. Updates wait until all Go-Pixo tabs/windows close, so an active batch keeps its matching code and codecs.

See [web/README.md](web/README.md) for format limits, production browser tests and the cache/update design.

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

Go Pixo source is MIT licensed. Bundled third-party codecs retain their own licenses; see [NOTICE.md](NOTICE.md) and [distributed codec notices](web/public/licenses/third-party.txt).
