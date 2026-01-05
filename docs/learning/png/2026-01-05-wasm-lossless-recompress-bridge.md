# 2026-01-05: WASM Bridge for Lossless PNG “Recompress from Bytes”

## Goal

Make the web worker use the same “already-compressed PNG recompress” pipeline as the CLI by calling a WASM-exposed function that accepts the **original PNG file bytes** (not just decoded pixels). This enables:

- **Never larger than input PNG** in lossless mode
- **Preserved color-management chunks** (`sRGB`, `iCCP`, `gAMA`, `cHRM`) when present
- Better compression on “already-compressed” PNGs (e.g. `images/cursor-meetup.png`)

## Why not pixels-only encoding?

The pixels-only WASM API (`encodePng` / `encodePngAdvanced`) can’t guarantee “never larger than original file” because it doesn’t have the original PNG bytes to compare against or return verbatim. Also, preserving certain ancillary chunks requires reading the PNG container.

## Implementation

### Go (WASM)

- New WASM bridge entrypoint:
  - `src/wasm/bridge.go` exposes `recompressPngLossless(pngBytes, preset, zopfliIterations, progressCb?)`
- Registered in WASM main:
  - `src/cmd/wasm/main.go` sets `js.Global().Set("recompressPngLossless", ...)`
- Core logic:
  - Calls `png.RecompressPNGBytesLossless` (from `src/png/recompress.go`)

### Web worker

- `web/src/worker.ts` chooses the bytes-based recompress path when:
  - `lossy === false`, and
  - `originalFileBytes` is present (PNG uploads)
- Otherwise it falls back to the existing pixels-based `encodePngAdvanced`/`encodePng` path.

### UI

- `web/src/App.res` already sends `originalFileBytes` to the worker for PNG inputs, so no UI changes are required to enable the new path.

## How to verify

1. Build WASM: `./scripts/build-wasm.sh`
2. Run web: `cd web && bun run dev`
3. Upload: `images/cursor-meetup.png`
4. In lossless mode, confirm the downloaded PNG is smaller than the original and opens correctly.

