# plan

Expose the new lossless “already-compressed PNG recompress” pipeline (implemented in Go under `src/png/recompress.go`) through the WASM bridge so the web UI (`web/src/App.res` + `web/src/worker.ts`) can use it for PNG inputs in lossless mode (size never increases, output stays valid, and color-management chunks are preserved when present).

## User Stories

- As a user, when I upload a PNG that’s already compressed, I still want go-pixo (Web/WASM) to try a better lossless recompress and produce a smaller PNG when possible.
- As a user, I want lossless mode to never make the PNG bigger than the original upload and never produce a broken file.
- As a developer, I want the web pipeline to explicitly use the new “recompress from original PNG bytes” path (not the pixels-only encoder path) so the behavior matches the CLI regression benchmark.

## Tasks

1. Add a new WASM-exposed function that recompresses a PNG from its original bytes (lossless).

- Files:
 - Modify: `src/wasm/bridge.go`
 - Modify: `src/cmd/wasm/main.go`
- Shape:
 - JS signature: `(pngBytes: Uint8Array, preset: number, zopfliIterations: number, progressCallback?: (phase, progress) => void) => Uint8Array | "error: ..."`
 - Go implementation should call `png.RecompressPNGBytesLossless(...)` and return `Uint8Array`.

2. Register the new function on the worker global.

- Files:
 - Modify: `src/cmd/wasm/main.go`
- Expected: `self.recompressPngLossless` (or similar) is available in `web/src/worker.ts` after WASM init.

3. Update the worker to use the new recompress path for PNG + lossless.

- Files:
 - Modify: `web/src/worker.ts`
- Behavior:
 - If `req.lossy === false` AND `req.originalFileBytes` exists AND file kind is PNG (or we assume PNG for this request type), call `recompressPngLossless(req.originalFileBytes, preset, zopfliIterations, onProgress?)`.
 - Otherwise fall back to existing `encodePngAdvanced(...)` pixels-based path.
- Keep the existing JS “never larger” guard as defense-in-depth, but it should become redundant for the lossless PNG recompress path.

4. Update the app to route PNG lossless compression through the new worker path.

- Files:
 - Modify: `web/src/App.res`
 - (Optional) Modify: `web/src/interop/imageDecode.ts`
- Approach:
 - For lossless PNG, avoid relying on pixels as the compression input; pass `originalFileBytes` through to the worker and let Go decode/recompress.
 - Keep `previewUrl`, `width`, and `height` population unchanged for UI.
 - Optional optimization: in `imageDecode.ts`, add a “metadata-only decode” path for PNG lossless that avoids `getImageData()` (saves memory and time).

5. Align preset mapping between web and Go.

- Files:
 - Modify: `src/wasm/bridge.go`
 - Modify (if needed): `web/src/types.res`
- Ensure `Types.presetToInt` maps to the intended Go presets for recompress (Smaller/Balanced/Faster).

6. Add verification/bench workflow for the web/WASM path using `images/cursor-meetup.png`.

- Files:
 - (Optional) Add: a tiny helper doc or script under `docs/learning/png/` describing “how to verify in web”.
- Commands:
 - `./scripts/build-wasm.sh`
 - `cd web && bun run dev`
 - Upload `images/cursor-meetup.png` and confirm the downloaded “compressed” file is smaller than the original.

7. Update documentation (dated) explaining the WASM bridging.

- Files:
 - Add: `docs/learning/png/2026-01-05-wasm-lossless-recompress-bridge.md`
 - Modify: `docs/learning/png/index.md` (link it)
- Content:
 - Which codepaths exist (pixels-based encode vs PNG-bytes recompress)
 - When each is used (lossless PNG uses bytes path)
 - What guarantees are enforced (never larger than original PNG, preserve sRGB/iCCP/gAMA/cHRM)

## Acceptable Criteria

- [ ] Web worker uses the new WASM function for PNG + lossless (verify via a debug log or a clear codepath condition).
- [ ] Uploading `images/cursor-meetup.png` in the web UI (lossless) yields a valid PNG smaller than the uploaded file.
- [ ] Lossless PNG path never returns output larger than the input PNG bytes (for any PNG upload).
- [ ] Existing pixel-based path still works for lossy mode (palette/quantization) and for non-PNG inputs (as currently supported).
- [ ] `go test ./...` passes.
- [ ] `golangci-lint run` passes with no warnings.
- [ ] `cd web && rescript build 2>&1 | grep -qi "Warning" && exit 1 || exit 0` passes with no warnings.

## Finding

- The new lossless improvements live in Go as “recompress from original PNG bytes” (`png.RecompressPNGBytesLossless`) which preserves color-critical chunks and guarantees “never larger than input PNG bytes”; the current WASM bridge only exposes “encode from pixels”, so the web path can’t fully benefit until we add a PNG-bytes API to `src/wasm/bridge.go` and route `web/src/worker.ts`/`web/src/App.res` through it.

## Risks

- Passing large `Uint8Array` buffers between main thread and worker can increase memory usage: mitigate by using `Transferable` (postMessage second arg) if needed, and by avoiding pixels extraction for lossless PNG.
- Adding more WASM entrypoints increases API surface and potential mismatches between web preset enums and Go options: mitigate by centralizing preset mapping and adding a small “which path used” debug log.
- Progress reporting may regress if the new recompress function does not emit progress phases: mitigate by either (a) keeping progress coarse (“decode”, “recompress”, “finalize”) or (b) leaving progress unchanged for pixels-based paths only.

## Assumptions

- Lossless PNG recompress path is only required when `lossless=true` and the uploaded file is a PNG.
- We can keep using the existing worker architecture (no main-thread WASM calls).
- The web UI currently treats PNG inputs as RGBA pixels; for the new lossless PNG bytes path, Go will decode the PNG container directly.