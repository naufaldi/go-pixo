# Go-Pixo rewrite plan (standard Go + WASM, no server)

Goal: create a **Go** reimplementation of the *idea* of `pixo` (PNG/JPEG encoding + compression), runnable:

- as a normal Go library (native),
- and **in the browser** via **Go → WASM**, with **no upload / no API**.

This document is a timeline + implementation order guide. It assumes you will **read the Rust code as reference**, but **write new Go code**.

## Scope decision (MVP)

We start with **PNG-only**.

- Why: PNG gets a useful product shipped faster (valid output early), and teaches the full "container format + compression" pipeline.
- What's out of scope for MVP: JPEG (DCT/bitstream complexity), advanced PNG lossy quantization, and SIMD.
- What "done" looks like for MVP: users can drag/drop PNG/JPEG inputs in the browser, and download **smaller PNG outputs** (re-encoded) without uploading anywhere.

## Current Status (Updated)

**PNG implementation is feature-complete** with advanced compression capabilities:

✅ **Completed Phases (1-5):**
- Phase 1: PNG minimum valid encoder (signature, chunks, zlib wrapper)
- Phase 2: Real DEFLATE compression (LZ77, Fixed/Dynamic Huffman)
- Phase 3: PNG filters (all 5 types + multiple selection strategies)
- Phase 4: PNG lossless optimizations (alpha optimization, color reduction, presets)
- Phase 5: PNG lossy mode (quantization with dithering)

✅ **Advanced Features (beyond original plan):**
- **Zopfli-style optimal DEFLATE compression** - iterative refinement for maximum compression
- Multiple filter strategies: MinSum, Adaptive, AdaptiveFast, Entropy, BruteForce
- Comprehensive preset system: Fast, Faster, Balanced, Smaller, Max, Extreme, Lossy
- Progress callback support for WASM
- Size guarantee option (ensure output ≤ original)

❌ **Not Started:**
- JPEG encoder (Phase 6-7)
- SIMD optimizations

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

## Repo structure (current implementation)

Current layout (Go-idiomatic, similar to Rust architecture):

- ✅ `src/png/` – PNG encoder (complete: chunks, filters, palette, quantization, optimizations)
- ❌ `src/jpeg/` – JPEG encoder (not started; planned for Phase 6)
- ✅ `src/compress/` – DEFLATE + zlib wrapper + CRC32/Adler32 + Zopfli optimal compression
- ✅ `src/wasm/` – Go WASM bridge (JS<->Go glue via `syscall/js`)
- ✅ `src/cmd/wasm/` – WASM entrypoint (builds `main.wasm`)
- ✅ `src/cmd/cli/` – CLI tool for testing/development
- ✅ `web/` – demo UI (Vite + TypeScript + Rescript)

## API shape (current implementation)

Current WASM API (in `src/wasm/bridge.go`):

**Basic API:**
- ✅ `EncodePng(pixels []byte, width, height int, colorType, preset int, lossy bool, maxColors int) ([]byte, error)`
- ✅ `BytesPerPixel(colorType int) int`
- ✅ `HandleQuantizeInfo() map[string]interface{}` — Returns quantization capabilities
- ✅ `HandleGetPresets() map[string]interface{}` — Returns available presets

**Advanced API:**
- ✅ `EncodePngAdvanced(pixels []byte, width, height int, colorType, preset int, lossy bool, maxColors int, dithering bool, ditherStrength float64, qualityTarget int, zopfliIterations int, progressFunc func(string, int)) ([]byte, error)`
  - Full control over compression options
  - Zopfli iteration configuration
  - Progress callback support
  - Configurable dithering strength and quality targets

**Presets:**
- `0` — Smaller (maximum compression with quality preservation)
- `1` — Balanced (standard trade-off)
- `2` — Faster (fast with size guarantee)
- `3` — Extreme (maximum compression with Zopfli)

JS/TS wrapper available in `web/` that returns `Promise<Uint8Array>`.

❌ **Not implemented:** `EncodeJpeg` (planned for Phase 6)

## Implementation timeline (rewrite order)

### Phase 0 — Bootstrapping (COMPLETED)

Deliverables:

- Go module initialized with `src/` layout.
- Vite + TypeScript + Tailwind v4 `web/` page.
- Go WASM build script (`scripts/build-wasm.sh`).
- End-to-end flow: file → bytes → wasm → bytes → download (placeholder).

### Phase 1 — PNG "minimum valid encoder" (correctness-first) ✅ COMPLETED

Goal: output a valid PNG for small RGB/RGBA images without fancy compression yet.

Implemented (mirror Rust concepts in `src/png/`):

- ✅ PNG signature + chunk writer
- ✅ Required chunks:
  - `IHDR`
  - `IDAT`
  - `IEND`
- ✅ CRC32 for chunks
- ✅ Raw scanline format:
  - filter byte per row (start with `0` = None)
  - pixel bytes follow
- ✅ Zlib wrapper around DEFLATE:
  - DEFLATE stored/uncompressed blocks (initial implementation)
  - Adler32 checksum (zlib)

Exit criteria: ✅ Met — Generated PNG opens everywhere (Chrome/Safari/Firefox).

### Phase 2 — Real DEFLATE compression (size improvements) ✅ COMPLETED

Goal: reduce output size without changing PNG semantics.

Implemented in `compress/` (mirror Rust `src/compress/`):

- ✅ LZ77 matcher with configurable compression levels (1-9)
- ✅ Huffman coding:
  - ✅ Fixed Huffman tables
  - ✅ Dynamic Huffman coding (with auto-selection)
- ✅ Zlib stream writer (`CMF/FLG`, blocks, Adler32)
- ✅ **Advanced:** Zopfli-style optimal DEFLATE compression (iterative refinement)

Exit criteria: ✅ Met — PNG size is significantly smaller than "stored blocks" baseline.

**Advanced Compression Features (Beyond Phase 2):**

The implementation includes advanced compression capabilities beyond the original plan:

- ✅ **Zopfli-style Optimal DEFLATE Compression** (`compress/zopfli.go`):
  - Iterative refinement algorithm for maximum compression
  - Configurable iterations (default 15, up to 50+)
  - Optimal block splitting
  - Used in `Extreme` and `Smaller` presets
  - Provides best compression ratios at the cost of slower encoding

### Phase 3 — PNG filters (big win for compression ratio) ✅ COMPLETED

Implemented filters:

- ✅ All 5 filter types: Sub, Up, Average, Paeth, None
- ✅ Multiple per-row filter selection strategies:
  - ✅ MinSum ("min sum of absolute values" heuristic)
  - ✅ Adaptive (best compression, slower)
  - ✅ AdaptiveFast (faster adaptive)
  - ✅ Entropy (minimize distinct bigrams for better DEFLATE correlation)
  - ✅ BruteForce (try all filters per row)

Why now: filters often matter more than tiny DEFLATE tweaks.

Exit criteria: ✅ Met — Size improves noticeably vs "filter none" across all strategies.

### Phase 4 — PNG lossless optimizations (optional but useful) ✅ COMPLETED

These match the Rust `PngOptions` knobs:

- ✅ Optimize alpha (zero RGB when alpha=0)
- ✅ Reduce color type (RGB→Gray, RGBA→RGB/GrayAlpha when safe)
- ✅ Strip metadata (ancillary chunk handling)
- ✅ Palette reduction when ≤256 colors (with median cut quantization)
- ✅ Options builder pattern for flexible configuration

Exit criteria: ✅ Met — Comprehensive preset system exists:
- `Fast` / `Faster` / `Balanced` / `Smaller` / `Max` / `Extreme` / `Lossy`
- All with documented trade-offs and use cases

### Phase 5 — PNG lossy mode (quantization) (optional) ✅ COMPLETED

Added:

- ✅ Palette quantization (max 256 colors, median cut algorithm)
- ✅ Optional dithering with configurable strength (0.0-1.0)
- ✅ Quality target for lossy compression (0-100)
- ✅ Configurable dithering algorithms

Exit criteria: ✅ Met — Lossy quantization fully implemented and available via WASM API.

### Phase 6 — JPEG baseline encoder (bigger project) ❌ NOT STARTED

Planned implementation in `jpeg/` (mirror Rust `src/jpeg/`):

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

### Phase 7 — JPEG features/presets (after baseline works) ❌ NOT STARTED

Planned enhancements (in increasing difficulty):

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

**Status: MVP achieved!** ✅

PNG-only implementation is complete and functional:
- ✅ Phase 0 → Phase 5 all completed
- ✅ Advanced compression features implemented
- ✅ WASM integration working
- ✅ Multiple presets available

PNG gets you a working product sooner because it's "just bytes + DEFLATE", while JPEG is a larger math/bitstream project.

**Next milestone:** Consider JPEG baseline encoder (Phase 6) if expanding beyond PNG-only, or focus on performance optimization and web UI polish.

---

## Current Focus & Next Steps

**Current Status:** PNG implementation is feature-complete with advanced compression capabilities.

**Current Focus Areas:**
1. **PNG Optimization & Advanced Compression** — Continue refining Zopfli integration, filter strategies, and compression ratios
2. **Performance Tuning** — Optimize encoding speed while maintaining compression quality
3. **WASM Integration** — Polish the web interface and user experience

**Next Steps (optional, future work):**

1. **JPEG Baseline Encoder (Phase 6)** — If expanding beyond PNG-only:
   - Start with RGB → YCbCr conversion and 8x8 block processing
   - Implement DCT and quantization
   - Build JPEG marker writing system

2. **Additional PNG Enhancements (if needed):**
   - Bit depth reduction (16-bit → 8-bit when safe)
   - Additional filter heuristics
   - SIMD optimizations (if targeting native performance)

3. **Web Product Polish (Phase 8):**
   - Enhanced UI/UX features
   - Batch processing
   - Progress indicators (already supported in API)
   - Web Worker integration for large files
