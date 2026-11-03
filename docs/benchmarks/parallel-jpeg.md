# Parallel JPEG encode: before/after benchmark

This records the measured effect of branch `t3/parallel-jpeg-encode` (issue #128) against `master` at `8ba5cd9` (3 November 2026). The branch:

- splits colour conversion, chroma subsampling, DCT, quantization and trellis across band workers for JPEG output of 4 MP or more. Each band worker runs its own Go WASM instance on a band of MCU rows. The compression worker encodes the first band itself, then entropy-codes the whole file once;
- quantizes straight into coefficients for progressive and optimized-Huffman output. Previously it encoded a baseline file and then re-parsed it. This also speeds up the serial path.

The output is byte-identical to the serial encoder. This is checked in three places:

- `TestEncodeBandsMatchesSerialEncode` in Go;
- `jpegParallel.wasm.test.ts` against the real `main.wasm`;
- `e2e/jpeg-parallel.spec.ts`, which compares the SHA-256 of a parallel encode in Chromium with a forced serial fallback.

The output sizes in every table below match too.

## How it was measured

Both versions were built for production with the same toolchain: Go 1.26.0 WASM, Vite 8, Node 24 and Playwright 1.64.1. Each was served with `vite preview`, before on port 4101 and after on 4102. Master's `main.wasm` was rebuilt from `8ba5cd9` with that toolchain, because the committed binary came from a different build. Measurements ran in headless Chromium on Windows 11, 16 threads (`navigator.hardwareConcurrency = 16`), 31 GB RAM.

```bash
cd web
node e2e/benchmark/compare-versions.mjs --before=http://127.0.0.1:4101 \
  --after=http://127.0.0.1:4102 --iterations=3 --only=codex-photo
```

**Time** is the median of 3 runs, from upload to the finished result. It includes a cold Go WASM start (each scenario reloads the page), and on the branch it also includes band worker start-up.

This branch also changes how the harness detects the end of a run. It now polls the batch summary every animation frame. Before, it used Playwright's `locator.waitFor`, which backs off to 500 ms polls, so earlier timings were quantized to about 0.5 s; the issue's baseline of 2.35, 2.31, 1.33 and 1.85 s shows this.

On this machine, 4 bands run: the compression worker plus 3 band workers.

## Results

### 4032 × 2688 photo (10.8 MP, `codex-photo-4032x2688-q90.jpg`)

| Setting | master | branch | Change | Output bytes (both) |
|---|---:|---:|---:|---:|
| Balanced (default) | 2234 ms | 1549 ms | −31% | 1,213,019 |
| Smaller file | 1906 ms | 1302 ms | −32% | 950,305 |
| Faster | 981 ms | 730 ms | −26% | 1,492,986 |
| Lossless | 1510 ms | 1575 ms | unchanged, within noise | 1,479,849 |
| Auto (WebP wins) | 858 ms | 875 ms | unchanged, not JPEG output | 690,526 |

### 1536 × 1024 photo (1.6 MP, `codex-photo-1536x1024-q92.jpg`), serial on both

| Setting | master | branch | Output bytes (both) |
|---|---:|---:|---:|
| Balanced (default) | 446 ms | 375 ms | 331,886 |
| Smaller file | 460 ms | 395 ms | 265,562 |
| Faster | 227 ms | 237 ms | 419,522 |
| Lossless | 452 ms | 458 ms | 435,396 |
| Auto (WebP wins) | 195 ms | 210 ms | 261,280 |

Small images do not regress. The Balanced and Smaller gains here come from the shorter serial path, not from band workers. Differences of 10–15 ms are within run-to-run noise. Lossless JPEG re-coding is unchanged.

PSNR and SSIM are identical between master and branch in every row.

### Parallel versus serial in one build

To find where band workers pay off, a build with the threshold set to 0 timed each size two ways: parallel, and serial forced by making `new Worker` throw inside the compression worker. Each value is the median of 3 runs, with sizes identical.

| Image | Balanced parallel | Balanced serial | Faster parallel | Faster serial |
|---|---:|---:|---:|---:|
| 1448 × 1086 (1.6 MP) | 570 ms | 619 ms | 216 ms | 289 ms |
| 2048 × 1536 (3.1 MP) | 842 ms | 921 ms | 252 ms | 309 ms |
| 2400 × 1800 (4.3 MP) | 1107 ms | 1319 ms | 317 ms | 395 ms |
| 4032 × 2688 (10.8 MP) | 1160 ms | 1467 ms | 599 ms | 776 ms |

Below about 4 MP, parallel saves 50–80 ms, close to the noise. It also costs three more Go heaps and band copies of the pixels. From 4 MP the saving reaches about 200–300 ms for Balanced, so `PARALLEL_JPEG_MIN_PIXELS` is 4,000,000.

Entropy coding stays serial and limits the speedup. In single-threaded Node runs of the 10.8 MP photo, encoding takes 1439 ms in total for Balanced. Each of the 4 bands takes 145–179 ms, but the final entropy pass (Huffman statistics, progressive scans) takes 776 ms.

## Worker and memory policy

- Bands = min(`hardwareConcurrency` − 1, 4). The compression worker encodes one band itself.
- The encode stays serial when:
  - fewer than 3 bands would result;
  - the image is under 4 MP;
  - the device is mobile (`userAgentData.mobile`, or a phone/tablet user agent);
  - `deviceMemory` ≤ 4;
  - nested workers are unavailable.
- Any band failure re-runs the whole image serially, with the same bytes.
- Band workers get the compiled `WebAssembly.Module` from the compression worker, so they never fetch or compile `main.wasm` again.
- RGBA band slices are transferred, not cloned.
- Band workers stop 3 s after the last encode, immediately on failure, and together with their compression worker on cancel.
- Admission (`imageOperation.ts`) charges JPEG-capable pixel jobs an extra 22 bytes per output pixel plus 16 MiB per band worker, only where band workers would run. The per-pixel figure covers the 4:4:4 worst case:
  - band RGBA copies: 4 bytes;
  - RGB composite and its Go copy: 3 + 3 bytes;
  - band coefficients in the band worker and in transit: 6 + 6 bytes.
- The service worker precaches `jpegBand.worker-*.js`, so parallel encoding also works offline.
