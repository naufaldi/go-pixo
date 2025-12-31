# Go-Pixo rewrite plan (standard Go + WASM, no server)

Goal: create a **Go** reimplementation of the *idea* of `pixo` (PNG/JPEG encoding + compression), runnable:

- as a normal Go library (native),
- and **in the browser** via **Go → WASM**, with **no upload / no API**.

This document is a timeline + implementation order guide. It assumes you will **read the Rust code as reference**, but **write new Go code**.

## Scope decision (MVP)

We start with **PNG-only**.

- Why: PNG gets a useful product shipped faster (valid output early), and teaches the full “container format + compression” pipeline.
- What’s out of scope for MVP: JPEG (DCT/bitstream complexity), advanced PNG lossy quantization, and SIMD.
- What “done” looks like for MVP: users can drag/drop PNG/JPEG inputs in the browser, and download **smaller PNG outputs** (re-encoded) without uploading anywhere.

## Important notes (public repo + attribution)

- The upstream repo uses the **MIT license**. MIT allows reimplementation and even copying, but:
  - if you copy text/code from the Rust repo, keep MIT notice + attribution in your repo.
  - if you reimplement from scratch, still add “Inspired by pixo” + link in your README (good practice and avoids confusion).
- Don’t “translate” Rust code line-by-line and call it original. Use the Rust repo to understand the pipeline, but write your own implementation and tests.

## Why WASM (client-only) is possible in Go

Browser flow is still the same:

1. User selects a file in UI (`File` API)
2. JS reads bytes (`await file.arrayBuffer()`)
3. JS calls your Go/WASM functions (via `syscall/js`)
4. Go returns output bytes
5. JS makes a `Blob` and downloads it

No server needed.

## Repo structure recommendation (for `go-pixo`)

Keep the layout similar to the Rust architecture, but Go-idiomatic:

- `png/` – PNG encoder (chunks, filters, palette/bit-depth later)
- `jpeg/` – JPEG encoder (baseline first; progressive later)
- `compress/` – DEFLATE + zlib wrapper + CRC32/Adler32
- `color/` – color type enum + RGB↔YCbCr helpers
- `resize/` – optional (can be later)
- `wasm/` – Go WASM bridge (JS<->Go glue via `syscall/js`)
- `cmd/wasm/` – WASM entrypoint (builds `main.wasm`)
- `web/` – demo UI (can start minimal; later polish)

## API shape (copy the “minimal surface area” idea)

Start with the smallest stable API, similar to `pixo`’s WASM layer:

- `EncodePng(pixels []byte, w, h int, colorType, preset int, lossy bool) ([]byte, error)`
- `EncodeJpeg(pixels []byte, w, h int, colorType, quality, preset int, subsampling420 bool) ([]byte, error)`
- `BytesPerPixel(colorType int) int`

JS/TS wrapper should hide details and return `Promise<Uint8Array>`.

## Implementation timeline (rewrite order)

### Phase 0 — Bootstrapping (COMPLETED)

Deliverables:

- Go module initialized with `src/` layout.
- Vite + TypeScript + Tailwind v4 `web/` page.
- Go WASM build script (`scripts/build-wasm.sh`).
- End-to-end flow: file → bytes → wasm → bytes → download (placeholder).

### Phase 1 — PNG “minimum valid encoder” (correctness-first)

Goal: output a valid PNG for small RGB/RGBA images without fancy compression yet.

Implement (mirror Rust concepts in `src/png/`):

- PNG signature + chunk writer
- Required chunks:
  - `IHDR`
  - `IDAT`
  - `IEND`
- CRC32 for chunks
- Raw scanline format:
  - filter byte per row (start with `0` = None)
  - pixel bytes follow
- Zlib wrapper around DEFLATE:
  - simplest first: DEFLATE “stored/uncompressed blocks”
  - Adler32 checksum (zlib)

Test approach:

- encode 1x1 and 2x2 fixed patterns; verify browsers can display.
- cross-check with a reference decoder (browser, or Go’s stdlib image/png during development).

Exit criteria:

- “Generated PNG opens everywhere” (Chrome/Safari/Firefox).

### Phase 2 — Real DEFLATE compression (size improvements)

Goal: reduce output size without changing PNG semantics.

Implement in `compress/` (mirror Rust `src/compress/`):

- LZ77 matcher (start simple, then improve)
- Huffman coding:
  - Fixed Huffman first
  - Dynamic Huffman next (bigger work, needed for good ratios)
- Zlib stream writer (`CMF/FLG`, blocks, Adler32)

Exit criteria:

- PNG size is smaller than “stored blocks” baseline for typical images.

### Phase 3 — PNG filters (big win for compression ratio)

Implement filters:

- Sub, Up, Average, Paeth
- Per-row filter selection:
  - start: “min sum of absolute values” heuristic
  - later: add faster heuristics / presets

Why now: filters often matter more than tiny DEFLATE tweaks.

Exit criteria:

- size improves noticeably vs “filter none”.

### Phase 4 — PNG lossless optimizations (optional but useful)

These match the Rust `PngOptions` knobs:

- optimize alpha (zero RGB when alpha=0)
- reduce color type (RGB→Gray, RGBA→RGB/GrayAlpha when safe)
- strip metadata (if you ever add ancillary chunks)
- palette reduction when ≤256 colors
- bit depth reduction (when possible)

Exit criteria:

- preset system exists: `fast / balanced / max` with documented trade-offs.

### Phase 5 — PNG lossy mode (quantization) (optional)

Add:

- palette quantization (max 256 colors)
- optional dithering

Exit criteria:

- user-facing “lossy PNG smaller” switch in web UI.

### Phase 6 — JPEG baseline encoder (bigger project)

Implement in `jpeg/` (mirror Rust `src/jpeg/`):

- RGB → YCbCr conversion
- 8x8 block splitting
- DCT (integer first)
- quantization tables (quality scaling)
- zigzag reorder
- DC differential coding + AC RLE
- Huffman encoding
- write required markers (SOI/APP0/DQT/SOF0/DHT/SOS/EOI)

Exit criteria:

- can encode a photo into a valid baseline JPEG that opens in browsers.

### Phase 7 — JPEG features/presets (after baseline works)

Enhancements (in increasing difficulty):

- chroma subsampling 4:2:0
- optimized Huffman tables (image-dependent)
- progressive mode
- trellis quantization

### Phase 8 — Web product polish (make it “easy for people”)

User-experience features:

- drag/drop, batch processing, progress
- “before/after” preview
- presets with plain language labels
- “privacy: runs locally” messaging
- use Web Worker (optional) to avoid blocking UI

Performance:

- avoid repeated allocations (reuse buffers)
- chunk big files in JS + show progress

## “Where to look in Rust” (reference map)

Use these Rust areas as conceptual reference while rewriting:

- PNG pipeline/options: `src/png/mod.rs`
- JPEG pipeline/options: `src/jpeg/mod.rs`
- DEFLATE + CRC/Adler: `src/compress/`
- WASM API surface: `src/wasm.rs`
- Web demo’s WASM wrapper: `web/src/lib/wasm.ts`

## MVP recommendation (so you finish)

If your goal is “ship something usable quickly”:

1) Build **PNG-only** first (Phase 0 → Phase 3).  
2) Add JPEG baseline later.

PNG gets you a working product sooner because it’s “just bytes + DEFLATE”, while JPEG is a larger math/bitstream project.

---

## Next step (actionable)

Start implementing **Phase 0** and **Phase 1** only, then ship a tiny demo:

- Phase 0: Go→WASM build + web UI wiring (file → bytes → wasm → bytes → download)
- Phase 1: PNG signature + chunks + zlib wrapper + “filter none” scanlines

After Phase 1 works end-to-end in the browser, iterate Phase 2 (DEFLATE) and Phase 3 (filters) to make files smaller.
