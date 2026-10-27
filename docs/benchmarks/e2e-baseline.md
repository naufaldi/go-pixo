# E2E benchmark baseline

Baselines for the default `bun run benchmark:e2e` flow (issue #30). The benchmark is diagnostic first. This document records measured numbers, how much they vary, what CI does with them, and the conditions for turning thresholds into a failing gate.

## What is measured

`web/e2e/benchmark/run-benchmark.mjs` starts the Vite dev server through `bun run dev` (which rebuilds Go WASM first) unless one already listens on `localhost:5173`. It then drives headless Chromium through six steps per iteration using the deterministic fixture `web/e2e/fixtures/benchmark/generated-photo-like-1024x768.png`:

1. Load App: navigate and wait for the `Go-Pixo` heading
2. Upload Image: set the file input and wait for the filename
3. Initial Compress: wait for the first result and an idle UI
4. Switch To WebP: change output format and verify a WebP preview
5. Resize Output: set 512 × 384 and verify the decoded preview size
6. Download WebP: download and check the RIFF/WEBP signature

The default is 3 iterations (`--iterations=N` or `BENCHMARK_ITERATIONS` overrides it).

### Caveats when reading the report

- **P95 and P99 of a default run are the slowest of 3 samples.** `percentile()` uses a ceil-index, so with 3 samples both resolve to the maximum. Within one run they carry no more information than "max". Cross-run ranges, below, are the real variance signal.
- **Throughput mostly measures suite startup.** `elapsedSeconds` starts before the dev server is launched, so it includes the WASM build and Vite startup, and every step reports the same value. Do not use it as a regression signal.
- **The target is the dev server, not a production build.** Load App includes dev-mode module serving.
- **Partial failure exits 0.** The runner exits non-zero only when every iteration fails. A 2/3 run is "green" unless the success rate is checked.

## CI baseline (GitHub-hosted `ubuntu-latest`)

Five runs of the `benchmark` job in `verify.yml` (run 40463040100, attempts 1 to 5; `45096f9` is the PR merge commit of `4d4d16c`) on 2026-10-27. Each runner reported Linux X64, 4 vCPU, Node v24.22.1, Bun 1.4.3 and Go 1.26.10; Playwright is 1.64.1 from the lockfile.

Per-run step **Avg** in ms (P95 = slowest of 3 in parentheses):

| Step | #1 EPYC 9V45 | #2 EPYC 7763 | #3 EPYC 7763 | #4 EPYC 7763 | #5 EPYC 9V74 |
| --- | --- | --- | --- | --- | --- |
| 1. Load App | 184.2 (239.9) | 258.2 (350.1) | 274.0 (338.7) | 254.0 (334.4) | 272.8 (389.3) |
| 2. Upload Image | 20.4 (23.1) | 28.4 (29.0) | 31.7 (32.9) | 30.9 (33.7) | 29.5 (32.0) |
| 3. Initial Compress | 1302.4 (1303.2) | 2321.5 (2325.8) | 2324.7 (2331.5) | 2324.5 (2329.9) | 2323.3 (2330.7) |
| 4. Switch To WebP | 280.9 (288.0) | 384.1 (385.6) | 392.0 (404.4) | 391.6 (394.5) | 389.8 (393.8) |
| 5. Resize Output | 123.9 (133.8) | 230.2 (241.6) | 256.2 (263.3) | 247.0 (250.4) | 205.1 (243.6) |
| 6. Download WebP | 51.3 (56.9) | 49.1 (54.7) | 49.9 (57.5) | 80.9 (145.0) | 47.7 (52.1) |
| **E2E total** | 1996.4 (2073.1) | 3320.9 (3432.3) | 3380.4 (3460.9) | 3409.1 (3478.4) | 3317.0 (3503.8) |
| Success rate | 100% (3/3) | 100% | 100% | 100% | 100% |
| Throughput (completion) | 0.43/s | 0.24/s | 0.24/s | 0.25/s | 0.26/s |

Job wall time: 56 to 59 s (setup, install, Playwright browser install and the benchmark).

### What the CI data shows

- **The runner hardware changes between runs, and that is the dominant variance.** On the EPYC 9V45 runner Initial Compress took 1302 ms; on the four other runners it took 2321 to 2325 ms (≈1.8×). E2E total moved the same way (1996 ms vs 3317 to 3409 ms).
- **On the same hardware, the compute-heavy steps are very stable.** Across the four 2.32 s runs, Initial Compress Avg spans 3.2 ms (<0.2%) and Switch To WebP spans 8 ms (≈2%).
- **Short steps are noisy.** Download WebP had one 145 ms outlier (run #4) against a typical 52 to 58 ms P95. Load App P95 spans 334 to 389 ms on the same hardware class.

## Local baseline (Apple M1)

Machine: Apple M1, 8 cores, 8 GB RAM, macOS 15.1.1 (24B91). Node v24.18.1, Bun 1.4.3, Go 1.27.3, Playwright 1.64.1, commit `311092b`. Two series of five sequential default runs on 2026-10-27. Other sessions shared the machine and could not be stopped, so neither series is from an idle host; the 1-minute load average is recorded for each.

### Series B: lighter load (preferred local reference)

Started once the 1-minute load average fell below 3 (2.87); it rose to 10.8 by the end. About 9 to 14 s per run.

| Run | Load before | 3. Initial Compress Avg (P95) | E2E total Avg (P95) |
| --- | --- | --- | --- |
| B1 | 2.87 | 1995.5 (2336.7) | 3406.0 (4381.5) |
| B2 | 4.10 | 1819.5 (1829.5) | 2633.5 (2669.1) |
| B3 | 3.93 | 2392.7 (2937.0) | 3686.5 (4960.4) |
| B4 | 6.64 | 2215.1 (2421.9) | 3313.3 (3589.5) |
| B5 | 10.52 | 1815.0 (1833.7) | 2706.4 (3090.3) |

Range across the 5 runs (per-run Avg; P95 = slowest of 3):

| Step | Avg range (median) | P95 range (median) | CV of Avg |
| --- | --- | --- | --- |
| 1. Load App | 204.0–511.3 (293.2) | 233.9–1157.1 (567.6) | 38% |
| 2. Upload Image | 25.4–90.5 (33.2) | 28.5–213.5 (39.4) | 61% |
| 3. Initial Compress | 1815.0–2392.7 (1995.5) | 1829.5–2937.0 (2336.7) | 12% |
| 4. Switch To WebP | 302.3–451.6 (401.9) | 319.3–668.0 (509.3) | 18% |
| 5. Resize Output | 123.1–211.5 (147.6) | 124.8–250.1 (154.7) | 23% |
| 6. Download WebP | 53.4–80.9 (62.2) | 55.0–111.0 (83.4) | 16% |
| **E2E total** | 2633.5–3686.5 (3313.3) | 2669.1–4960.4 (3589.5) | 15% |

Success rate 100% (3/3) in all 5 runs; completion throughput 0.22–0.34/s. The two fastest runs (B2, B5) agree within 5 ms on Initial Compress and are the fastest observed in either local series.

### Series A: heavily contended

Run first, while other browser and benchmark workloads were active (1-minute load average about 19 on 8 cores at the end). About 20 to 25 s per run. Kept to show how much host contention moves the numbers; do not use it as a reference.

| Step | Avg range (median) | P95 range (median) | CV of Avg |
| --- | --- | --- | --- |
| 1. Load App | 398.9–1219.5 (527.3) | 504.4–2840.5 (716.9) | 50% |
| 2. Upload Image | 49.1–153.8 (71.7) | 57.4–351.8 (98.0) | 48% |
| 3. Initial Compress | 3061.1–3793.7 (3482.4) | 3435.3–4475.4 (4106.8) | 8% |
| 4. Switch To WebP | 575.8–656.0 (633.6) | 654.0–777.3 (751.0) | 5% |
| 5. Resize Output | 242.9–354.0 (304.5) | 301.4–454.9 (354.4) | 15% |
| 6. Download WebP | 66.1–91.1 (71.9) | 69.5–131.8 (98.2) | 14% |
| **E2E total** | 4793.1–6626.0 (5416.4) | 5609.0–9632.2 (6303.4) | 12% |

Success rate 100% in all 5 runs; completion throughput 0.12–0.16/s. Median Initial Compress is 1.7× series B. Local numbers are only comparable when the load is recorded.

## CI behavior: artifact-only

The `benchmark` job in `.github/workflows/verify.yml` runs on every pull request (and `workflow_dispatch`) beside the other verification jobs. It:

- runs `bun run benchmark:e2e` and tees the full output to `web/benchmark-report/report.txt`
- records runner OS, vCPU count, CPU model and tool versions in `web/benchmark-report/environment.txt`
- copies the step table into the job summary
- uploads `web/benchmark-report/` as the `e2e-benchmark-report` artifact (30-day retention), even when the step fails

Timings never fail the job. The job fails only when the runner exits non-zero, which means every iteration failed, i.e. the flow itself is broken.

Why no threshold gate yet: the CI data shows a ≈1.8× swing caused only by which runner type GitHub assigns. Any absolute threshold tight enough to catch a real regression on the faster runner would either fail on the slower one or miss a large regression on the faster one. Warning-only is also premature for the same reason: warnings that fire based on runner luck get ignored.

## Proposed thresholds (not enforced)

Based on the slowest observed CI hardware class (EPYC 7763 / 9V74, 4 vCPU), so a slower runner assignment does not trip them:

| Metric | Baseline (max observed P95) | Proposed warn at (× 1.5) |
| --- | --- | --- |
| Success rate | 100% | below 100% |
| 3. Initial Compress P95 | 2331.5 ms | 3500 ms |
| 4. Switch To WebP P95 | 404.4 ms | 610 ms |
| 5. Resize Output P95 | 263.3 ms | 400 ms |
| E2E total P95 | 3503.8 ms | 5260 ms |
| 1. Load App, 2. Upload Image, 6. Download WebP | — | no timing threshold (short and noisy) |
| Throughput | — | no threshold (dominated by startup) |

The 1.5× factor sits below the measured 1.8× hardware swing and well above the <2% same-hardware spread of the stable steps. Because the fast runner is 1.8× quicker, these thresholds would not catch a regression smaller than about 2.7× on that runner. Comparing against a per-CPU-model baseline would fix that.

## Promoting to a gate

Promote in this order, each only once its condition holds:

1. **Success rate below 100% fails the job.** Lowest risk and catches flakiness or partial breakage. Condition: 20 consecutive CI runs at 100%.
2. **Warning annotations for the thresholds above.** Condition: at least 20 CI runs collected (from the `e2e-benchmark-report` artifacts, including `environment.txt`) covering every CPU model seen, with no run above the proposed thresholds. Recompute the table from those runs.
3. **Failing on timing regressions.** Condition: thresholds keyed by CPU model (or a relative comparison against the base branch on the same runner), per-step CV under 10% within each CPU model over those runs, and no false warnings during the warning-only period.

## How to run

```bash
./scripts/build-wasm.sh
cd web
bun install --frozen-lockfile
bun run rescript:build
bunx playwright install chromium
bun run benchmark:e2e                          # default, 3 iterations
BENCHMARK_ITERATIONS=10 bun run benchmark:e2e  # more samples
```

Close other heavy workloads first and record the machine, versions and load average with the numbers. If something already listens on port 5173 the runner uses it instead of starting the dev server, so make sure it is this checkout's server.

To compare a PR with this baseline, download its `e2e-benchmark-report` artifact from the Actions run and check the CPU model in `environment.txt` before comparing timings.
