# 2026-01-05: Recompressing “Already-Compressed” PNGs (Lossless)

## Goal

Make `go-pixo` reliably recompress PNGs that were previously optimized by other tools to a smaller file size **without visible quality loss**, **without pixel changes**, and **without producing broken PNGs**.

## What “lossless” means here

- The output PNG must decode correctly (`image/png` + browsers).
- Pixel data must roundtrip (decode → recompress → decode) without changes.
- Color appearance should not change due to dropped color-management chunks, so we preserve `sRGB`, `iCCP`, `gAMA`, and `cHRM` when present in the input PNG.

## Key implementation changes

1. **Avoid corrupt dynamic Huffman output**
   - `DeflateEncoder.EncodeAuto` now validates the dynamic candidate by inflating it and comparing to the original uncompressed bytes; if validation fails, it uses fixed Huffman instead.

2. **Make dynamic headers encodable under DEFLATE constraints**
   - The code-length alphabet used in dynamic headers must fit in 3-bit lengths (<= 7). When the optimal build exceeds that, we fall back to a guaranteed-encodable table.
   - Dynamic code lengths are validated to stay within DEFLATE limits (<= 15) before RLE/header encoding.

3. **Use stdlib zlib when it’s smaller**
   - For PNG IDAT, we compare our custom zlib stream to a stdlib `compress/zlib` stream and keep the smaller valid result (still “standard Go”, no third-party deps).

4. **PNG recompress “never larger than input” guarantee (for PNG inputs)**
   - `RecompressPNGBytesLossless` returns the smaller of `{original input bytes, recompressed output}`.

5. **Preserve color-critical chunks from the input PNG**
   - We parse the input PNG container and copy through `sRGB`, `iCCP`, `gAMA`, `cHRM` into the output PNG (written after `IHDR`).

## Benchmarks (CLI, lossless, preset: balanced)

Outputs are written to `images/compress/`:

- `images/cursor-meetup.png` → `images/compress/cursor-meetup.go-pixo.png` (864K → 705K)
- `images/code.png` → `images/compress/code.go-pixo.png` (768K → 629K)
- `images/cursor-2025-models.png` → `images/compress/cursor-2025-models.go-pixo.png` (194K → 154K)

## Tests

- Added an integration/regression test using `images/cursor-meetup.png` to ensure:
  - output size is never larger than input (PNG recompress),
  - output decodes, and
  - decoded pixels match the input.

