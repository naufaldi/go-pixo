# libheif + libde265 (WebAssembly)

HEIC/HEIF fallback decoder used by `web/src/interop/codecs.ts` when the browser has no native HEIC decoder.

| | |
|---|---|
| libheif | 1.23.7 (LGPL-3.0), https://github.com/strukturag/libheif/tree/v1.23.7 |
| libde265 | 1.1.3 (LGPL-3.0), https://github.com/strukturag/libde265/tree/v1.1.3 |
| Toolchain | Emscripten 3.1.61 (`emscripten/emsdk:3.1.61`) |
| Build | `scripts/libheif-wasm/build.sh`, run by `.github/workflows/libheif-wasm.yml` |
| CI run | https://github.com/naufaldi/go-pixo/actions/runs/41563381308 |

`BUILDINFO` records the source tarball hashes and compiler; `SHA256SUMS` records the output hashes:

```
590bc19156b831d628677f70b07b175ea92bb43943dfeb64887a7b63f636df51  libheif.js
c2db515bd9b94178fe7c3ceb08feb0c3d3733b39e24f904b5485070eb6d1e612  libheif.wasm
```

Both files are copied from the CI artifact unmodified. The app emits them as separate assets and refuses to decode when the loaded build reports libheif older than 1.23.7 or libde265 older than 1.1.3.

## Rebuild or replace

1. Run `scripts/libheif-wasm/build.sh out/libheif` inside an Emscripten 3.1.61 toolchain (for example `docker run --rm -v "$PWD:/src" -w /src emscripten/emsdk:3.1.61 scripts/libheif-wasm/build.sh out/libheif`), or trigger the "Build libheif WASM" workflow and download its `libheif-wasm` artifact.
2. Copy `libheif.js`, `libheif.wasm`, `SHA256SUMS` and `BUILDINFO` into this directory.
3. Build the Go WASM, then run `bun run build` in `web/`. This regenerates the bundled assets and offline integrity hashes.

A modified library can be dropped in the same way, as long as it keeps the exported functions `build.sh` requests.
