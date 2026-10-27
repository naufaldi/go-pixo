# Go-Pixo Web App

React + ReScript + Tailwind v4 image compression app. PNG/JPEG use Go WASM; bundled codecs and browser APIs handle other supported formats locally.

## Setup

Use Bun 1.4.3 and Node.js 22.22.2+, 24.15+, or 26+. CI uses Node 24.22.1.
`bun.lock` is the supported lockfile; use Bun to update dependencies rather than creating another package manager's lockfile.

1. Install dependencies:
```bash
bun install --frozen-lockfile
```

2. Build ReScript:
```bash
bun run rescript:build
```

3. Build WASM (from repo root):
```bash
./scripts/build-wasm.sh
```

4. Run dev server:
```bash
bun run dev
```

This builds Go WASM, then runs ReScript watch and the Vite dev server together. It does not register an offline service worker.

## Local production and offline use

Build Go WASM from the repository root, then build and preview the final web output:

```bash
# From the repository root
./scripts/build-wasm.sh
cd web
bun run build
bunx vite preview --host 127.0.0.1 --port 4173 --strictPort
```

Open `http://127.0.0.1:4173` with a connection and wait for **Ready offline**. You can then disconnect, reload and compress newly selected files. Installation through the browser's install/add-to-home-screen menu is optional. Deploy this app at the origin root; its Go runtime and public metadata currently use root-relative URLs.

Offline preparation downloads every emitted app asset, including public `main.wasm`/`wasm_exec.js`, worker chunks, lazy codec JS and codec WASM. The generated `dist/sw.js` contains that exact list and each file's SHA-256 integrity hash. The worker itself is managed by the browser's service worker lifecycle. A missing file, an HTML fallback at a WASM URL, or a mismatched deployment fails installation. The UI says ready only when this app's activated worker controls the page and verifies that the whole matching cache still exists.

Only those build assets enter the app cache. There is no runtime response caching, image upload, or persistence of selected images, previews, compressed results or ZIPs. An incomplete cache can be repaired online by downloading the same full, integrity-checked build list. Reloading loses your in-memory queue and results. Updates download in the background, wait for existing tabs to close, and clean only caches owned by this application scope. Finish and download your work, close **all** Go-Pixo tabs/windows, then reopen to activate a waiting update.

Offline limits:

- A first visit with no network cannot load the app. Prepare it online first.
- HTTPS or localhost, service workers, Cache Storage and sufficient browser storage are required. `file://` and ordinary remote HTTP do not support this workflow.
- Clearing site data, private browsing restrictions or browser storage eviction can remove/prevent the cache. Reconnect and reload to prepare it again. Readiness is rechecked on page/network/visibility changes and every 30 seconds while visible; browser storage is not a permanent backup.
- Browser codec support and memory still constrain image processing offline. Cache preparation does not make unsupported input formats or animations supported.

## Format and processing limits

The format pipeline validates file contents rather than trusting the extension or MIME label. PNG, JPEG, WebP, still AVIF, single-frame GIF, BMP and self-contained static SVG are supported inputs where the required browser decoding APIs are available. TIFF decoding is bundled for single-page, upright, strip-based images with interleaved unsigned integer channels: 1–4 samples with matching, uniform 1-, 2-, 4-, or 8-bit depths. Planar and tiled layouts, RAW/DNG, floating-point samples, and higher bit depths are rejected. HEIC/HEIF decoding is bundled locally for single-image, 8-bit still inputs; multi-image containers and sequences are rejected. HEIC input is converted to PNG, JPEG, WebP or AVIF, not re-encoded as HEIC.

PNG, JPEG, WebP and AVIF are output choices. PNG/JPEG use Go WASM, AVIF uses a bundled single-thread codec without requiring cross-origin isolation, and WebP uses the browser's canvas encoder. The output is validated; an unsupported encoder returns an error rather than silently writing another format. Formats without an output encoder use PNG for “same format.” Auto tests fast WebP against a valid original (or PNG for imported formats); lossless Auto uses a lossless AVIF candidate. It selects the smallest tested valid result and does not guarantee every file will become smaller. JPEG/WebP lossless same-format mode retains the original rather than re-encoding lossily. The ordinary pixel pipeline is 8-bit; unsupported precision-preserving conversions are rejected.

Animated GIF, APNG, animated WebP and AVIF/HEIF sequences are rejected rather than flattened to one frame. SVG must have bounded dimensions, contain no scripts/animation, and reference no external resources. Files are limited to 64 MiB; input and resized dimensions are limited to 24 megapixels and 16,384 pixels per side. The app admits at most two read/decode/compression operations (one on low-resource devices), so pending encoded reads are separately bounded to two 64 MiB files. After header inspection and before native or bundled rasterization, each operation reserves an estimate of eight RGBA surfaces (32 bytes per pixel) at the larger source/output area, including possible native JPEG orientation during single-axis resizing. The normal aggregate workspace budget is 384 MiB. One job may exceed it while running alone, bounded to 24 MP × 32 bytes = 768,000,000 bytes (732.4 MiB); other pixel work waits. Small jobs still run together. Reservations survive decode completion, cancellation and settings resets until all owned decode/compression resources settle.

This is an admission estimate, not a guarantee about browser peak memory. Completed preview/output blobs, encoded-byte copies, browser caches and retained codec/WASM heap capacity are outside the workspace estimate. Large batches can still hit browser/device memory limits. Unchanged-dimension lossless JPEG/WebP uses the exact original bytes and an original-file preview URL without canvas extraction or Go initialization. Lossless PNG also uses an original preview URL and sends original bytes to its byte recompressor without frontend RGBA extraction. Bundled TIFF/AVIF/HEIC previews still decode locally to PNG.

## Testing

### Unit Tests (Vitest)
```bash
bun run test
```

### E2E Tests (Playwright)
```bash
bun run test:e2e
```

Place test images in `web/e2e/fixtures/` for conversion tests.

### Production offline browser tests

After building the final Go WASM and `dist` as above:

```bash
bunx playwright install chromium
bunx playwright test --config playwright.offline.config.ts
```

The dedicated config previews the existing `dist` on `127.0.0.1:4173` with service workers enabled. It explicitly selects `e2e/offline.production.ts`; the default dev-server suite's `*.spec.ts` glob excludes that file. Production artifacts use `test-results-offline/`, separate from the dev suite's default output directory. The test disables Chromium's HTTP cache so a warm HTTP response cannot mask a service-worker cache miss. It checks every emitted asset is cached, disconnects and reloads before creating new input, then validates real PNG/JPEG/WebP/AVIF bytes and decoded dimensions. While still offline it adds another image with the same filename, downloads a ZIP, and verifies both unique entries, distinct AVIF contents and decoded dimensions. It also checks cache eviction removes the ready claim and a successful HTML fallback at a missing Go WASM URL fails installation. Keep this suite separate from development-server tests. CI must build Go WASM, compile ReScript and build Vite before invoking it.

Focused offline build/lifecycle tests:

```bash
bunx vitest run build/offline.test.ts build/service-worker.test.ts src/offline/client.test.ts
```

## Architecture

- **ReScript** (`src/*.res`) - Type-safe UI components
- **React** - Component framework
- **Tailwind CSS v4** - Styling (CSS-first, no config)
- **Go WASM** (`public/main.wasm`) - PNG/JPEG compression engine
- **Vite** - Build tool and dev server
- **Offline build plugin** (`build/offline.ts`) - hashes the finished output and generates the service worker after worker/public files are emitted
- **Offline state** (`src/offline/client.ts`) - production registration, active-controller cache verification and subscriptions
- **Offline status** (`src/components/OfflineStatus.res`) - prop-free readiness/update UI; mount as `<OfflineStatus />`

## Development

- ReScript files compile to `.res.js` (in-source)
- Import ReScript modules: `import { App } from './App.res.js'`
- Compression workers announce transport readiness without loading Go. PNG/JPEG operations lazily initialize one reusable Go promise per worker; WebP and AVIF initialize only their required encoder
- Tailwind classes work directly (no config needed)

Codec license texts and pinned source/build references are shipped under `public/licenses/` and available from the app’s Open-source licenses link.
