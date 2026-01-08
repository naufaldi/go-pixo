# PNG Optimization Plan (go-pixo) — Speed First, Quality Unchanged

This doc is the **execution plan** for speeding up PNG compression in go-pixo while keeping:
- **Lossless pixels identical** (decoded output matches input pixels).
- **Good compression** for the “Balanced” preset (don’t regress badly).
- **Maintainability** (small, readable changes; minimal flags; tests + benches).

Source context:
- Current architecture: `docs/learning/architecture.md`
- Optimization checklist: `docs/learning/optimize/optimization-guide.md`
- Rust comparison notes: `docs/learning/optimize/diff-png.md`

---

## 1) Where time goes today (in this repo)

PNG encode time is dominated by:

1. **Scanline filtering + selection** (`src/png/filter_selector.go`, `src/png/filter_*.go`, `src/png/idat_writer.go`)
2. **DEFLATE (zlib) compression** (`src/compress/*` via `src/png/idat_writer.go`)
3. **(Lossy mode only)** quantization + palette indexing (`src/png/quantize.go`, `src/png/palette.go`)

The biggest speed wins come from:
- **Removing allocations** in filter selection (currently allocates per filter, per row).
- Avoiding “compress twice” behavior where it doesn’t match the preset goals.

---

## 2) “Quality not change” — define it explicitly

There are two separate modes:

### A) Lossless PNG (primary goal)

“Quality not change” means: **output PNG decodes to the same pixels** as input.
- Bytes can differ (different filters, different DEFLATE blocks, different chunk order).
- Pixel equivalence must be verified via decode + compare in tests.

### B) Lossy PNG (optional / secondary)

If `MaxColors` / dithering is enabled, pixel data changes by design.
Here “quality not change” means: **for the same user settings**, visuals stay consistent.

This plan prioritizes **lossless speed** first.

---

## 3) Measurement setup (do this before optimizing)

If you don’t measure, you won’t know which work paid off.

### 3.1 Benchmarks (native Go)

Add Go benchmarks for:
- Filter selection on representative rows:
  - `SelectFilterWithStrategy` (RGBA and RGB) across widths (256, 1024, 2048, 4096).
- Scanline build + zlib build:
  - `buildZlibData` from `src/png/idat_writer.go`
- DEFLATE encoder:
  - `EncodeAuto`, `EncodeWithFallback`, and “fast path” variants in `src/compress/deflate_encoder.go`
- Lossy-only:
  - `Quantize`, `QuantizeWithDithering` in `src/png/quantize.go`

Use a small “fixture” corpus (10–30 images) for integration timing, but keep micro-benchmarks separate.

### 3.2 In-browser timing (WASM / worker)

In the worker layer, time:
- `filtering`
- `deflate`
- (if used) `quantize`

Use `performance.now()` and log results to console (or show dev-only metrics).

---

## 4) Priority 1: Fastest win with lowest risk (lossless)

### 4.1 Eliminate per-row allocations in filter selection

**Current behavior:**
`ApplyFilterSub`, `ApplyFilterUp`, etc. each do `make([]byte, len(row))` per call.
In `MinSum` / `Adaptive`, you evaluate up to 5 filters per row → **5 allocations per row**.

For a 4K image (~2160 rows), this can be 10k+ allocations per image, which kills throughput in WASM.

**Plan:**
1. Introduce a reusable scratch buffer container:
   - `type FilterScratch struct { none, sub, up, avg, paeth []byte }`
2. Add “write into dst” filter APIs:
   - `ApplyFilterSubTo(dst, row []byte, bpp int)`
   - `ApplyFilterUpTo(dst, row, prev []byte)`
   - etc.
3. Rewrite selectors to reuse scratch per row:
   - Compute each candidate filter into scratch, score it, keep best.
4. Optional: use `sync.Pool` so scratch is reused across encode calls (especially in worker).

**Why it’s safe:** `idat_writer.go` immediately appends filtered bytes into `scanlineData` (copy),
so scratch can be reused on the next filter and next row.

**Expected outcome:** large speedup + lower GC pressure, with pixel-identical output.

### 4.2 Speed up scoring (SumAbs) without changing results

`SumAbsoluteValues` currently:
- casts per byte: `int8(b)` then branches

**Plan:** replace with a 256-entry lookup table:
- `absInt8[256]uint8`
- sum using table lookup (still identical scoring)

This is simple, safe, and measurable.

### 4.3 Don’t “compress twice” on speed presets

In `buildZlibData` (`src/png/idat_writer.go`) you may build your own zlib stream, then also run stdlib zlib and take the smaller result.

That’s great for “smallest”, but expensive for “fast”.

**Plan:**
- Add an option like `CompareStdlibZlib bool`
- Set:
  - `false` for speed presets
  - `true` for size presets

This doesn’t affect lossless correctness, only CPU time.

---

## 5) Priority 2: DEFLATE speed policy + reduced work

### 5.1 Add a true “fast DEFLATE” mode

`EncodeAuto` tries fixed and dynamic and picks the smaller. That’s extra work.

**Plan:**
- For “fast” presets, choose **one**:
  - fixed-only (often faster), OR
  - dynamic-only (may compress better)
- Keep `EncodeAuto` for Balanced/Small presets.

### 5.2 Reuse encoder state across files (WASM workloads)

WASM workers often process many images in sequence.

**Plan:**
- Reuse `compress.DeflateEncoder` instances between calls (reset slices with `[:0]`).
- Avoid repeated allocations of token buffers inside LZ77 if possible.

This is a maintainable “engineering” win: fewer allocations, faster steady-state.

---

## 6) Priority 3: Lossy quantization speed (only if you use it)

If you aren’t using `MaxColors`, skip this section.

### 6.1 Exact-speedup: unique-color → palette-index map (non-dither)

In `Quantize`, you already count unique colors, but you still do a full palette search per pixel.

**Plan (exact behavior, no approximation):**
1. After palette build, compute nearest palette index **once per unique color**.
2. During pixel scan, map pixel color → nearest index via the precomputed map.

This keeps results identical, but saves a ton of repeated work on images with repeated colors.

### 6.2 Make `Palette.FindNearest` faster (exact behavior)

Current `Palette.FindNearest` uses `int64` and `math.MaxUint64`.

**Plan:**
- Use `uint32`/`int32` math for distance.
- Keep code readable and tested.

### 6.3 Optional “fast approximate” LUT (guarded behind an option)

The Rust comparison notes mention a 6-6-6 LUT approach.

**Caution:** LUT bucketing can change which palette entry wins in edge cases.

If you want it:
- Make it opt-in via a new option/preset.
- Document “may differ slightly” (but should look the same).

---

## 7) “Other ways” (bigger changes; only if needed)

These are real options, but they increase complexity:

### 7.1 Add Bigrams filter strategy (size-first)

`docs/learning/optimize/diff-png.md` calls out Rust’s **Bigrams** filter strategy which can improve DEFLATE correlation.

This can improve size but may add CPU cost; keep it for “smallest” presets.

### 7.2 libdeflate / oxipng / zopfli (WASM integration)

Potentially large size/speed wins, but:
- bigger build surface
- more debugging complexity
- larger WASM

Only consider if the “allocation + policy” wins aren’t enough.

---

## 8) Recommended execution order (best ROI)

### Phase A (speed foundation, lossless) — do first
1. Filter scratch buffers + `ApplyFilter*To` APIs
2. Fast scoring (lookup table)
3. Disable stdlib-zlib comparison for speed presets
4. Add micro-benchmarks

### Phase B (DEFLATE policy + reuse)
1. “fast deflate” mode (skip EncodeAuto)
2. Reuse encoder state across worker runs
3. Bench + validate decode correctness

### Phase C (lossy-only quantize speed)
1. Unique-color map for nearest palette index (exact)
2. Optimize `Palette.FindNearest` (exact)
3. Optional LUT (approx) behind flag

### Phase D (size-only extras)
1. Bigrams strategy
2. More expensive DEFLATE strategies only for “smallest”

---

## 9) Definition of Done (per phase)

For each phase:
- `go test ./...` passes
- Benchmarks show measurable improvement (record before/after numbers)
- Output PNG decodes successfully via `image/png`
- Lossless mode preserves pixels (decode + compare)
- Balanced preset doesn’t regress size meaningfully (set a threshold, e.g. ≤2–3%)

---

## 10) Comparison vs `docs/brain/optimize/png.md`

`docs/brain/optimize/png.md` is a **big, ambitious roadmap** aimed at “Rust-level performance” and includes many ideas from the diff docs (LUT, k-means, Bigrams, Redmean, parallelism, SIMD).

This file (`png-codex.md`) is intentionally **narrower and more practical** for the constraint you stated: **speed-first while keeping quality unchanged** (especially in **lossless** mode).

### What to keep from `png.md` (aligns with speed + maintainability)

- **Fix per-row allocations** in filter selection (scratch buffers / pooling / reuse).
- **Add early termination** to cut wasted filter work (only after you have benchmarks).
- **Benchmark-first workflow** (micro-bench + real-image timings).

### What to treat as “optional / not first” from `png.md`

- **Palette LUT / k-means / Redmean**: these are mainly about **lossy quantization** quality/speed.
  - In this repo, quantization only runs when `opts.MaxColors > 0 && opts.MaxColors < 256` (`src/png/encoder.go`).
  - Your default lossless presets (`FastOptions`, `BalancedOptions`, etc.) use `MaxColors: 0` (`src/png/options.go`), so LUT/k-means **won’t help lossless speed**.
- **Bigrams filter strategy**: mostly a **size win**, not a speed win (it adds work).
- **Parallel filter selection / SIMD**: can help in native builds, but is constrained in browser/WASM:
  - Web Worker ≠ multi-core Go; without WASM threads (SharedArrayBuffer/COOP/COEP), Go code stays single-threaded.

### Important: `png.md` has code that doesn’t match the current codebase

Example mismatch:
- In this repo, `FilterStrategy` is an `int` enum (`src/png/options.go`).
- Some snippets in the docs use `FilterStrategy = "Bigrams"` style strings (not compatible as-is).

So: treat `png.md` as **idea inventory**, not copy/paste implementation.

---

## 11) How to use `diff-png.md` + `diff-rust-go.md` for *this* repo

Those docs are still valuable, but you should translate them into “what is feasible + worth it in Go→WASM”.

### What transfers cleanly (high value here)

1. **Allocation discipline / scratch reuse**
   - Rust wins partly from “better memory locality through scratch buffer reuse” (`docs/learning/optimize/diff-rust-go.md`).
   - In this repo, filter functions allocate every call (`src/png/filter_sub.go`, `src/png/filter_up.go`, `src/png/filter_average.go`, `src/png/filter_paeth.go`).
2. **Avoid repeated work**
   - Your filter selection currently builds filtered bytes then scores them (extra passes).
   - A fast, maintainable alternative is “score without materializing”, then apply best filter once:
     - `scoreSub(row, prev, bpp) -> int`
     - pick best filter type
     - `ApplyFilterXTo(dst, row, prev, bpp)` once for output
3. **Policy: don’t do expensive “try-everything” on speed presets**
   - `buildZlibData` can do a second stdlib zlib compress and pick smaller (`src/png/idat_writer.go`).
   - `EncodeAuto` tries multiple Huffman modes (`src/compress/deflate_encoder.go`).
   - For speed presets, pick one strategy and move on.

### What is mostly “size-first” or “native-first”

1. **Bigrams strategy**
   - From `diff-png.md`: 2–5% size improvement is plausible, but it’s more CPU.
   - Keep it for “smallest” presets only.
2. **SIMD**
   - Rust uses AVX2/SSSE3/NEON for scoring and other hot paths (`diff-rust-go.md`).
   - Go→WASM SIMD is a moving target; plan as “nice-to-have later”, not Phase A.
3. **Parallel filtering**
   - Rust uses Rayon (multi-core). In-browser, you only get this if you adopt WASM threads or multi-worker sharding.

---

## 12) Codebase audit notes (what actually matters for speed today)

### 12.1 Filter selection allocates heavily (biggest easy win)

- Each filter allocates a new `[]byte` of row size (`ApplyFilter*`).
- `selectMinSum` / `selectEntropy` evaluate multiple filters per row (`src/png/filter_selector.go`).
- This creates many allocations and lots of memory bandwidth work (especially bad in WASM).

### 12.2 Quantization is only for lossy mode

- Quantization happens only under `opts.MaxColors > 0 && opts.MaxColors < 256` (`src/png/encoder.go`).
- If your main goal is speeding up **lossless recompress / encode**, LUT/k-means/Redmean do not move the needle.

### 12.3 “Compress twice then pick smaller” is not speed-friendly

- `buildZlibData` can compute your custom zlib stream and then stdlib zlib stream and pick smaller (`src/png/idat_writer.go`).
- That is a deliberate size optimization; it should be disabled for speed presets.

### 12.4 `ZopfliEncode` here is “Zopfli-style”, not full Zopfli

- `src/compress/zopfli.go` iterates a few alternative encodings; it’s not a full reference-quality zopfli.
- Treat this as “slow size mode”, not something to optimize for speed.

---

## 13) Recommendation (best plan for speed + quality unchanged)

If you only do one thing: **fix filter allocations first**.

**My recommended sequence (re-validated against current code):**
1. Implement **allocation-free filter selection** (scratch + apply-to or score-only).
2. Add a speed preset policy: **no stdlib zlib compare**, no “try multiple deflate modes”.
3. Add benchmarks and capture before/after numbers.
4. Only then consider optional size features (Bigrams) and lossy-quality features (k-means/Redmean/LUT).
