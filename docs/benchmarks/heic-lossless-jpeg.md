# HEIC, lossless JPEG and JPEG encoder: before/after benchmark

This records the measured effect of branch `t3/heic-perfect-quality` against `master` at `2c3dfc7` (2 November 2026). The branch:

- restores HEIC/HEIF input on a patched libheif build;
- adds a lossless JPEG optimizer, so Perfect Quality makes JPEGs smaller with identical pixels;
- fixes the Go JPEG encoder's progressive and optimized-Huffman output, which decoded to a visibly wrong image;
- makes trellis quantization about 2.3× faster with bit-identical output.

## How it was measured

Both versions were built for production from the same toolchain (Go 1.26.0 WASM, Vite 8, Node 24.22.2) and served with `vite preview`: before on port 4101, after on 4102. The harness drives headless Chromium (Playwright 1.64.1) on Windows 11 (16 threads, 31 GB RAM):

```bash
cd web
node e2e/benchmark/compare-versions.mjs --before=http://127.0.0.1:4101 \
  --after=http://127.0.0.1:4102 --iterations=2 --heic=/path/to/example.heic
```

For each image and setting, the harness reloads the page, applies the setting, uploads one file and waits for the batch result. Before and after alternate per scenario, so warm-up and thermal drift affect both.

- **Time** is the median from upload to the finished result. It includes a cold Go WASM start, because every scenario reloads the page.
- **Quality** is computed in the page, comparing the app's own "Original" preview with the compressed output: PSNR over RGB and SSIM over luma in 8×8 windows. "identical" means every decoded pixel matches.

Images:

| Image | What it stands for |
|---|---|
| `codex-photo-1536x1024-q92.jpg` | Typical web photo. The committed Codex still life, saved by Chromium's JPEG encoder at quality 0.92. |
| `codex-photo-4032x2688-q90.jpg` | Phone-size photo (10.8 MP), same source upscaled, quality 0.90. |
| `codex-still-life.png` | Photo stored as PNG (`web/e2e/fixtures/benchmark`). |
| `indomaret-screenshot.png` | UI screenshot (`web/e2e/fixtures/browser-verify`). |
| `rainbow-451x461.heic` | libheif test image (`web/src/interop/codecs-fixtures`). |
| `example.heic` | libheif's `examples/example.heic`, 1280×854. Not committed. |

Each row comes from one machine run with two iterations, and a synthetic scene is not proof for every image. Treat differences under about 10% in time as noise. The harness is in the repository so the numbers can be reproduced.

## Results

Cells read: time · output size (saving) · PSNR / SSIM.

### JPEG photo, 1536×1024 (original 454.8 KB)

| Setting | Before | After |
|---|---|---|
| Balanced (default) | 2352 ms · 356.0 KB (21.7%) · **19.5 dB / 0.248** | **1345 ms · 332.1 KB (27.0%) · 37.8 dB / 0.985** |
| Smaller file | 2875 ms · 283.2 KB (37.7%) · **19.6 dB / 0.253** | **1845 ms · 265.7 KB (41.6%) · 36.8 dB / 0.979** |
| Faster | 300 ms · 419.5 KB (7.8%) · 40.9 dB / 0.994 | 305 ms · 419.5 KB (7.8%) · 40.9 dB / 0.994 |
| Perfect Quality | 84 ms · 454.8 KB (0%) · identical | 815 ms · **435.4 KB (4.3%)** · identical |
| Auto | 189 ms · 261.3 KB WebP (42.5%) · 38.7 dB / 0.983 | 191 ms · same |

### JPEG photo, 4032×2688 (original 1.52 MB)

| Setting | Before | After |
|---|---|---|
| Balanced (default) | 19271 ms · 1.34 MB (11.5%) · **24.4 dB / 0.493** | **9054 ms · 1.21 MB (20.1%) · 46.4 dB / 0.995** |
| Smaller file | 19263 ms · 1.07 MB (29.4%) · **24.4 dB / 0.494** | **9026 ms · 949.6 KB (37.4%) · 43.4 dB / 0.992** |
| Faster | 1334 ms · 1.49 MB (1.6%) · 50.9 dB / 0.998 | 1309 ms · same |
| Perfect Quality | 96 ms · 1.52 MB (0%) · identical | 1805 ms · **1.48 MB (2.5%)** · identical |
| Auto | 1349 ms · 690.5 KB WebP (54.5%) · 42.0 dB / 0.987 | 829 ms · same bytes (no code change; time difference is noise) |

Before, Balanced and Smaller decoded to a wrong image: SSIM 0.25 to 0.49 means visible damage, not compression artifacts. The cause was the Go encoder's progressive writer and its optimized Huffman tables, which emitted entropy data that decoders reconstructed wrongly; Go's `image/jpeg` and Chrome agree on this. On a q85 photo in a Go unit test, progressive measured 19.5 dB and optimized Huffman 10.9 dB, against 37.4 dB for baseline. The encoder now writes baseline JPEG and re-codes the same coefficients through the lossless optimizer, so these options only change entropy coding.

### PNG inputs (no PNG code changed)

| Image | Setting | Before | After |
|---|---|---|---|
| Still life PNG (2.70 MB) | Balanced | 2869 ms · 675.1 KB (75.0%) · 33.3 dB | 2860 ms · same |
| | Perfect Quality | 2339 ms · 2.70 MB (0%) | 2833 ms · same |
| | Auto | 197 ms · 266.7 KB WebP (90.1%) · 37.5 dB | 205 ms · same |
| Screenshot PNG (878 KB) | Balanced | 1325 ms · 273.6 KB (68.9%) · 37.5 dB | 1332 ms · same |
| | Auto | 196 ms · 77.7 KB WebP (91.2%) · 42.7 dB | 200 ms · same |

PNG output and timings are unchanged within noise.

### HEIC inputs

| Image | Setting | Before | After |
|---|---|---|---|
| `example.heic` (718.1 KB) | Balanced | ❌ rejected as "animated" | 804 ms · 333.4 KB JPEG (53.6%) · 33.4 dB / 0.984 |
| | Smaller file | ❌ | 816 ms · 287.2 KB JPEG (60.0%) · 32.0 dB / 0.978 |
| | Faster | ❌ | 804 ms · 406.6 KB JPEG (43.4%) · 34.4 dB / 0.988 |
| | Perfect Quality | ❌ | 1336 ms · 2.44 MB PNG (larger; lossless) |
| | Auto | ❌ | 829 ms · 320.1 KB WebP (55.4%) · 36.0 dB / 0.991 |
| `rainbow-451x461.heic` (7.1 KB) | Balanced | ❌ "temporarily unavailable" | 201 ms · 8.8 KB JPEG · 49.4 dB |
| | Auto | ❌ | 201 ms · 4.0 KB WebP (43.4%) · 46.2 dB |

HEIC encodes efficiently, so a small or synthetic HEIC can grow when converted. With "Same as input", lossy mode now produces JPEG; Perfect Quality produces PNG, which is lossless and therefore large for photos.

### Loading (cold cache, throttled to 20 Mbps / 40 ms RTT)

Measured with 3 runs per case. Each run uses a fresh browser context and uploads one small file.

| | Before | After |
|---|---|---|
| Page ready | 240 ms | 266 ms |
| First PNG result | 499 ms | 589 ms |
| First HEIC result | ❌ | 1101 ms (includes loading libheif) |
| Bytes after first result | 8.2 MB | 11.6 MB |

Static assets:

| Asset | Before | After |
|---|---|---|
| `main.wasm` | 4.14 MB (gzip 1.135 MB) | 4.30 MB (gzip 1.179 MB) |
| Initial JS (`index-*.js`) | 302 KB (gzip 93.8 KB) | 302 KB (gzip 93.9 KB) |
| Offline precache, first visit | 9.57 MB | 13.14 MB |

The page loads as before. The extra bytes are the libheif decoder (3.4 MB raw, 803 KB gzip at the time; see the follow-up below), which the offline service worker precaches so HEIC also works offline. The app imports it only when a HEIC file arrives and the browser cannot decode HEIC natively. Safari, including iOS, decodes HEIC itself and never loads it.

## Go microbenchmarks

| Benchmark | Before | After |
|---|---|---|
| Trellis encode, 1024×768 (`BenchmarkTrellisEncode`) | ~535 ms | ~220 ms (2.3×), identical output |
| Lossless optimize, 12 MP standard-table JPEG (`BenchmarkOptimizeLossless`) | — | ~665 ms, 9.4% smaller, identical pixels |

`TestTrellisMatchesReferenceOutput` and `TestTrellisKeepsTieBreaking` pin trellis output to hashes computed from the `master` implementation: 120,000 random block and configuration combinations, plus coefficients on exact half steps where candidates tie. The distortion weight is capped at 1e6 so the prefix sums stay finite.

## Settings and what they now do

- **Perfect Quality, JPEG:** the original's quantized DCT coefficients are re-coded with optimized Huffman tables and progressive scans; four layouts are cost-estimated and the smallest of the best two is written. The result is used only if it is smaller and, in the browser, decodes to exactly the original's pixels. Metadata is reduced to ICC and Adobe segments, JFIF without thumbnail, and the EXIF orientation. The cost is one Go WASM start plus two decodes for the pixel check, which took 0.8 s and 1.8 s above.
- **Balanced / Smaller, JPEG:** quantization and trellis are unchanged. Progressive and optimized-Huffman output is now correct and about 7% smaller. The lossless re-code of the original is also tried when the lossy result stays above 80% of the original's non-metadata bytes (APPn and COM segments are excluded, since the re-code drops most of them). This is a speed heuristic, not a guarantee.
- **Faster:** unchanged, and it skips the lossless re-code in lossy mode.
- **Go API and CLI:** `Progressive` now always yields a progressive file and `OptimizeHuffman` alone a sequential one, both with optimized tables.
- **HEIC:** the browser's native decoder is tried first; otherwise libheif 1.23.7 with libde265 1.1.3 is used. Builds older than libheif 1.23.7 or libde265 1.1.3 are refused at runtime. Rotation (`irot`) and crop (`clap`) are supported.

## Known limits

- The lossless optimizer rejects CMYK/YCCK, 12-bit, arithmetic-coded and lossless/hierarchical JPEGs, and progressive files whose scans re-code coefficients. In each case the original is kept.
- On the dev server only, the first two HEIC files decoded at the same moment can fail to load the decoder module; a retry works, and production builds are unaffected.
- Encoding still runs on one worker thread per image; see issue #128.

## Follow-up: in-house HEIC decoder build

The `heic-to` 1.6.7 package bundled libde265 1.0.16, which has known memory-safety bugs fixed in 1.1.2 and 1.1.3, as a wasm2js build. Go Pixo now builds libheif 1.23.7 with libde265 1.1.3 to real WebAssembly itself (`scripts/libheif-wasm`, workflow "Build libheif WASM") and ships the unmodified output from `web/third_party/libheif/`. Measured in Node 24 on this machine, median of 7 decodes after warm-up:

| | heic-to 1.6.7 (wasm2js) | Own build (WebAssembly) |
|---|---|---|
| libde265 | 1.0.16 | 1.1.3 |
| Decoder download | 3.40 MB raw, 803 KB gzip | 1.63 MB raw, 529 KB gzip |
| Module start | 32 ms | 9 ms |
| `example.heic` 1280×854 | 124 ms | 61 ms (2.0×) |
| `rainbow-451x461.heic` | 9.6 ms | 2.9 ms (3.3×) |

Trellis quantization was also rewritten after this benchmark (PR #130): Balanced/Smaller JPEG encodes are 4–6× faster in native Go with the same PSNR and sizes within ±0.05%.
