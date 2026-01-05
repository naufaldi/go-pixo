# Plan

Make go-pixo reliably recompress already-compressed PNGs (e.g. `images/cursor-meetup.png`) to a smaller file size while staying visually lossless (no corruption, no unexpected color shifts) by fixing the DEFLATE dynamic-Huffman path, adding a real “never make it bigger” guard against the original PNG bytes, and preserving color-critical PNG chunks when present.

## Scope

- In: Fix DEFLATE dynamic header reliability (code-length alphabet constraints), make `EncodeAuto`/dynamic usable for PNG IDAT, implement a file-size guarantee vs the original PNG bytes, wire `ZopfliIterations` meaningfully, and preserve color-critical chunks (sRGB/iCCP/gAMA/cHRM) for PNG inputs.
- Out: Default-on lossy quantization/dithering, UI/UX changes, or implementing a full oxipng feature set (all transforms/heuristics).

## User Stories

- As a user, I want to compress a PNG that already looks optimized and still get a smaller download when additional lossless wins exist.
- As a user, I want the recompressed PNG to look the same (no visible quality loss, no broken output, no unexpected color changes).

## Acceptable Criteria

- [ ] Lossless recompress outputs a valid PNG that decodes and renders correctly (no corruption).
- [ ] When the input is a PNG, the lossless recompress output is never larger than the original input PNG bytes (size guarantee).
- [ ] For `images/cursor-meetup.png`, at least one lossless preset produces output smaller than the input file size.
- [ ] Dynamic Huffman blocks do not error with `invalid symbol` on large/realistic scanline inputs (tests cover this).
- [ ] PNG inputs with color profiles keep their appearance by preserving sRGB/iCCP/gAMA/cHRM when present.
- [ ] `-iterations N` changes the DEFLATE work done (and presets map to meaningful iteration counts).
- [ ] A Go test uses `images/cursor-meetup.png` as an integration/benchmark fixture for the CLI/encoder and passes in `go test ./...`.
- [ ] A dated write-up is added under `docs/learning/png/` documenting results and the final approach (for later doc sync).

## Findings

- Dynamic Huffman currently fails on real scanline data because `WriteDynamicHeader` can produce a code-length alphabet that exceeds DEFLATE’s 7-bit limit; the current behavior can collapse to an all-zero table and later trigger `invalid symbol`.
- Because dynamic blocks are unreliable, we effectively underuse dynamic Huffman; on PNG scanlines this is a big reason the output isn’t competitive for “already-compressed” PNGs.
- `ZopfliEncode` is currently not Zopfli-style optimization, and `png.Options.ZopfliIterations` is not wired into the “optimal” path, so the knob doesn’t represent real work.
- The current “EnsureSizeNotLarger” check compares output PNG size against raw pixel length (not the original PNG file size), so it does not prevent “recompress makes it bigger” for already-compressed PNG inputs.
- The current PNG pipeline does not preserve ancillary chunks from input PNGs; dropping color-management chunks (sRGB/iCCP/gAMA/cHRM) can change appearance in some viewers.

## Risks

- Dynamic Huffman fixes that violate RFC1951 can create subtly-invalid DEFLATE streams: Mitigation: add inflate (`compress/zlib`) + PNG decode roundtrip tests on produced output.
- Preserving color metadata incorrectly can still change appearance (e.g., mismatched sRGB/iCCP): Mitigation: copy-through exact bytes for supported color chunks; add fixtures with known profiles.
- Stronger compression can be slow in WASM: Mitigation: keep heavy modes opt-in and consider a time/effort budget with safe fallback behavior.

## Assumptions

- Default mode must be visually lossless; any lossy mode remains explicit/opt-in.
- We can access the original PNG bytes in CLI/WASM flows (so we can enforce the “never larger than original PNG” guarantee for PNG inputs).

## Action items

- [ ] Add a repro test that builds realistic PNG scanline input and asserts dynamic header + dynamic block writing succeeds (no `invalid symbol`).
- [ ] Implement a length-limited Huffman strategy for the code-length alphabet (max 7 bits), with a guaranteed-encodable fallback when needed.
- [ ] Enforce DEFLATE max code length constraints for literal/length and distance trees (max 15 bits) and add regression tests for out-of-range lengths.
- [ ] Add an API/entrypoint that can recompress a PNG with access to the original PNG bytes, so we can choose the smallest of {original, recompressed candidates}.
- [ ] Preserve/copy-through color-critical PNG chunks (sRGB/iCCP/gAMA/cHRM) for PNG inputs; add fixtures/tests that fail if these chunks are dropped.
- [ ] Wire `png.Options.ZopfliIterations` into the DEFLATE path so `-iterations` and presets map to real optimization work.
- [ ] Fix or rename `FilterStrategyBruteForce` so it’s not misleading (if kept, base it on estimated deflate cost or sampled compression, not `len(filtered)`).
- [ ] Add CLI validation for `images/cursor-meetup.png` showing (a) decode success and (b) a smaller lossless output for at least one preset, while never producing output larger than the original PNG bytes.
- [ ] Add a Go integration test that runs the CLI/encoder against `images/cursor-meetup.png` and asserts: (1) output decodes, (2) output size <= input size, and (3) at least one preset beats input size (this file is our regression benchmark).
- [ ] After implementation stabilizes, write a dated doc: `docs/learning/png/2026-01-05-already-compressed-png-recompress.md` summarizing baseline sizes, what changed (dynamic Huffman + size guarantee + color-chunk preservation), and current best presets; then link it from `docs/learning/png/index.md`.
- [ ] Run verification: `go test ./...`, `golangci-lint run`, and `cd web && rescript build 2>&1 | grep -qi "Warning" && exit 1 || exit 0`.

## Architecture (if applicable)

PNG recompress should treat the original PNG bytes as a first-class candidate output (for the “never larger” guarantee). The encoder produces multiple lossless candidates (different filter strategies + dynamic/fixed/optimal deflate configurations) and picks the smallest valid output while preserving color-critical chunks when present. DEFLATE dynamic Huffman must always be encodable within RFC1951 limits to be usable on real scanline distributions.

## Diagram (if applicable)

```mermaid
graph TD
  A[Input PNG bytes] --> B[Decode to pixels + parse color chunks]
  B --> C[Generate lossless candidates (filters + deflate)]
  A --> D[Original bytes candidate]
  C --> E[Validate decode + choose smallest vs original]
  E --> F[Output PNG bytes]
```

## Open questions

- Do you want a stdlib `compress/flate` fallback path (used only when custom deflate is too slow or fails), or should we keep “custom deflate only” and accept slower fixes?
- Should “preserve sRGB/iCCP/gAMA/cHRM when present” be the default for lossless recompress, even if it reduces the maximum possible size savings vs stripping them?