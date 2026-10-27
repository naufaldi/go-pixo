# go-pixo

Private image compression and conversion in the browser. PNG/JPEG use Go → WebAssembly (WASM); modern codecs run locally in browser workers. Images are never uploaded.

## What this repo is

- A new **Go rewrite** inspired by the Rust project `pixo` (MIT licensed).
- Target UX: “drag/drop → compress → download”, **no upload**, **no API**, no server required.

## Setup & Development

### Requirements

- Go 1.26+ (required by golang.org/x/image; CI tracks Go 1.26 patches)
- Bun 1.4.3 (for web frontend; `web/bun.lock` is the only dependency lockfile)
- Node.js 22.22.2+, 24.15+, or 26+ (CI uses 24.22.1)

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

## Image resource limits

The Go PNG/JPEG recompression APIs, CLI, and WASM exports accept encoded files up to **64 MiB** and images up to **24,000,000 pixels**, with **16,384 pixels per side**, matching the browser. Recompression checks `DecodeConfig` before full decoding or conversion. Raw encoders and CLI resize targets enforce the same dimension budget; pixel buffers must have the exact checked length. PNG IHDR construction separately validates the original dimensions against the format's 2³¹−1 limit before converting to `uint32`.

At this pixel budget, a single 8-byte decoded raster uses at most 192 MB and an RGBA conversion uses at most 96 MB. Encoders also keep scanlines, coefficients, compression state, and output, so these limits bound raster workspaces rather than guaranteeing a peak memory or processing-time cap. Expensive presets and concurrent calls still need caller-side scheduling. Native APIs use the same fixed limits in `src/internal/imagelimit`; there is no runtime override. Supporting larger trusted inputs needs a separately designed streaming or configurable-budget API.

WASM validates numeric types, finite integer dimensions, option ranges, and actual byte-array lengths before allocating or copying into Go. It accepts `Uint8Array` and `Uint8ClampedArray` inputs, returns the existing error-string form for invalid calls, and keeps both advanced PNG callback signatures. JPEG quality must be 1–100; PNG Zopfli iterations must be 0–50 (0 keeps the preset default).

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
