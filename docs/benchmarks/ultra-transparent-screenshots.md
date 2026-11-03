# Ultra on transparent screenshots: before/after benchmark

This records a bug report from after #141 shipped. The user reported Ultra (smallest) PNG compression still failing on macOS screenshots with `failed to encode PNG: png: palette quantization does not support transparency; use lossless encoding`. The settings were Same as input, with Lossless off.

## Likely cause (not confirmed in the user's session)

The deployed build already contained the #141 fix. Live `main.wasm` and `sw.js` matched `master` at `68b4c0a`, and the attached screenshots all compress on a fresh visit to the live site. The user's own browser session was not inspected, so the cause of their failure is not established. A stale offline cache is the most likely explanation, because the offline service worker can keep serving an old build:

- It serves the app cache-first and never calls `skipWaiting`.
- A returning visitor therefore keeps running the cached build, even after reloading.
- A new build stays `waiting` until every Go-Pixo tab closes.

A controlled reproduction showed this behaviour can produce the reported error:

1. Visit the pre-#141 build.
2. Deploy the #141 build and reload twice.
3. The page still loads the old bundle. Ultra fails on the transparent screenshots with the exact error.

## Update behaviour

This change keeps the browser's update lifecycle. A new build activates only after every Go-Pixo tab has closed, so no open tab is switched to a different build's JS, workers or WASM. `main.wasm` and `wasm_exec.js` have fixed URLs, so a mid-session takeover could mix builds. The "Update ready" notice now says that reloading keeps the current version, and that the user should download their results and then close every Go-Pixo tab and reopen.

An earlier draft of this change activated the waiting build automatically, before the app mounted. Review found that design unsafe:

- It could not atomically confirm that no other tab was open.
- It had no safe recovery when activation was ambiguous or timed out.

It was removed. The production e2e test `a redeploy never takes over open tabs and applies once every tab has closed` covers this behaviour. A tab that holds a result keeps its result and build while a second tab opens and reloads. That tab can still compress afterwards, and the new build activates only once both tabs have closed.

## How it was measured

- **Inputs.** The inputs were the ten screenshots attached to the bug report, labelled `shot-01` to `shot-10`.
  - They are private, so they are not committed. The harness checks each file against the SHA-256 hashes below.
  - They are downscaled attachments of at most 2,000 px wide, not the original files.
  - Five of them (`shot-05` to `shot-08` and `shot-10`) have a macOS window shadow, so each has 431,514 non-opaque pixels. The other five are fully opaque.
- **Before.** A production build of `8d1ffa6`, the last build before #141 and the one a stale cache serves.
- **After.** A production build of an earlier draft of this branch, based on `68b4c0a`. That draft has the same Go and compression code as `68b4c0a`. It differs only in the offline update code, which was later removed.
- **Toolchain.** Both builds used Go 1.27.3 WASM and were served by a static server. They ran in headless Chromium from Playwright 1.64.1, on Windows 11 with 16 threads and 31 GB of RAM. Each version ran once, so the timings are indicative only.

```bash
cd web
node e2e/benchmark/compare-versions.mjs --before=http://127.0.0.1:4181 \
  --after=http://127.0.0.1:4182 --iterations=1 --scenarios=ultra --cold=false --batch \
  --fixtures=/path/to/screenshots --manifest=/path/to/fixtures.json --out=ultra.json
```

The settings were Ultra (smallest), Same as input, Lossless off. Each file was uploaded on its own, then all ten were uploaded together as one batch.

The quality columns compare the decoded output with the input:

- **Alpha mismatches** compares alpha exactly, across all pixels.
- **PSNR and SSIM** are measured after both images are composited over white.

## Results

| Image | Size (px) | Input | Before | After | Time after | Alpha mismatches after | PSNR / SSIM after |
|---|---|---:|---|---:|---:|---:|---|
| shot-01 | 1098×868 | 196,247 | 74,792 | 74,792 | 0.80 s | 0 / 0 | 48.17 / 1.0000 |
| shot-02 | 2000×1255 | 3,018,036 | 377,492 | 377,492 | 3.76 s | 0 / 0 | 30.14 / 0.8752 |
| shot-03 | 1184×1278 | 229,990 | 109,690 | 109,690 | 1.72 s | 0 / 0 | 48.08 / 0.9999 |
| shot-04 | 1130×926 | 203,420 | 96,015 | 96,015 | 1.16 s | 0 / 0 | 48.19 / 0.9999 |
| shot-05 | 2000×1232 | 1,360,852 | error | 1,360,852 | 3.14 s | 0 / 431,514 | ∞ / 1.0000 |
| shot-06 | 2000×1232 | 1,454,133 | error | 1,454,133 | 3.33 s | 0 / 431,514 | ∞ / 1.0000 |
| shot-07 | 2000×1232 | 1,187,322 | error | 1,187,322 | 4.66 s | 0 / 431,514 | ∞ / 1.0000 |
| shot-08 | 2000×1232 | 1,397,808 | error | 1,397,808 | 3.29 s | 0 / 431,514 | ∞ / 1.0000 |
| shot-09 | 2000×1139 | 867,476 | 133,294 | 133,294 | 2.25 s | 0 / 0 | 41.30 / 0.9897 |
| shot-10 | 2000×1232 | 734,482 | error | 689,062 | 3.39 s | 0 / 431,514 | ∞ / 1.0000 |

- **Size columns.** Input, Before and After are file sizes in bytes. "error" means the run failed with the transparency error.
- **Alpha mismatch columns.** Each cell shows the number of pixels whose alpha changed, followed by the number of non-opaque pixels in the input.
- **Opaque files.** Before and after, these have the same output size, PSNR and SSIM. The output bytes were not hashed, so this does not prove they are byte-identical. Before, the times were 0.83, 3.46, 1.70, 1.21 and 2.37 s. Each version ran only once, so these timings cannot separate a real change from run-to-run variation.

| Batch of 10 | Completed | Failed | Time |
|---|---:|---:|---:|
| Before | 5 | 5 | 7.1 s |
| After | 10 | 0 | 18.3 s |

In this single run, the batch took longer after the fix, mostly because the five transparent files are now encoded instead of failing straight away.

The deployed site at `68b4c0a` was also checked in a fresh incognito T3 Preview tab, with the same settings and all ten files:

- The batch result was "10 completed, 0 failed, 0 remaining".
- `shot-05` produced a PNG with the same dimensions.
- `shot-05` had no alpha or colour mismatches.

## Limitations

- **Transparent files barely shrink.** Ultra on a transparent screenshot now succeeds, but it saves about 0% (`shot-10` saved 6%). Palette quantization doesn't support alpha, so the encoder falls back to lossless RGBA. That output is usually bigger than the input, and the app then keeps the original. Smaller output would need palette quantization with `tRNS` alpha, which is a separate change.
- **Visitors on an old cached build must close every tab.** A reload keeps the cached build, even with a single tab open. To get a new build, they close every Go-Pixo tab and reopen the site, and the "Update ready" notice says so. Visitors running a build from before this change still see its older notice wording.
- **The cause of the reported failure is unconfirmed.** See "Likely cause" above.

## Fixture hashes (SHA-256)

| Label | SHA-256 |
|---|---|
| shot-01 | `43677e256d10cc6a4623aac98fbb8900472c907263d5032a849beddca98af886` |
| shot-02 | `37b44da49bb45ecefeaa0b576f8f1f7f3244fc578800e2fe0a14e2da5192cc10` |
| shot-03 | `eeccc0817a2dc94f7771540dbf1224c19b08afb17004f90e788dfc8dacafb2c6` |
| shot-04 | `27eb5174bc03fc4e10d964aee23f4f4b1fa57e50da2897586f8e941b198b2a35` |
| shot-05 | `a9c39e340396876fc4154aa730663b862b5cf2b4a4a991b2d91543b452dec00e` |
| shot-06 | `de6b9a2eec877a86b72dcd04e51dc60558af88dec3203e648be0b59707dfc880` |
| shot-07 | `08b21bfd06f55cfba8ec13e3a0cbceef7b8bd017feaa6495d21dc5abc296e0f3` |
| shot-08 | `aa93e0d2f3e8cdd5ab83784701e21514ae7fc93af1635ceadec1e7b64a729e16` |
| shot-09 | `29c37c94328fd9d9d41a9cd59c5952cf0ec8c18f8f58b16ebc901b499b09b42f` |
| shot-10 | `3bedf48daa2f632a756ebd1a8ecedc9deb6aaa9434aa26dcaf2c56244e89478e` |
