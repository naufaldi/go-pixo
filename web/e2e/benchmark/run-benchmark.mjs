import { chromium, expect } from '@playwright/test';
import { mkdir, readFile, rm } from 'node:fs/promises';
import { createServer } from 'node:net';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn } from 'node:child_process';

import {
  buildBenchmarkSummary,
  formatDuration,
  formatStepTable,
  formatThroughput,
  summarizeStep,
} from './metrics.mjs';

const benchmarkDir = dirname(fileURLToPath(import.meta.url));
const webRoot = resolve(benchmarkDir, '..', '..');
const target = process.env.BENCHMARK_TARGET ?? 'http://localhost:5173';
const iterations = readIterations();
const fixturePath = join(
  benchmarkDir,
  '..',
  'fixtures',
  'benchmark',
  'generated-photo-like-1024x768.png',
);
const downloadDir = join(benchmarkDir, '..', 'test-results', 'benchmark');
const steps = [
  '1. Load App',
  '2. Upload Image',
  '3. Initial Compress',
  '4. Switch To WebP',
  '5. Resize Output',
  '6. Download WebP',
];

function readIterations() {
  const fromArg = process.argv.find((arg) => arg.startsWith('--iterations='))?.split('=')[1];
  const rawValue = fromArg ?? process.env.BENCHMARK_ITERATIONS ?? '3';
  const parsed = Number.parseInt(rawValue, 10);

  if (!Number.isFinite(parsed) || parsed < 1) {
    throw new Error(`Invalid iteration count: ${rawValue}`);
  }

  return parsed;
}

async function isPortOpen(port) {
  return new Promise((resolvePromise) => {
    const socket = createServer();
    socket.once('error', () => resolvePromise(true));
    socket.once('listening', () => {
      socket.close(() => resolvePromise(false));
    });
    socket.listen(port, '127.0.0.1');
  });
}

async function waitForTarget(url, timeoutMs = 30000) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    try {
      const response = await fetch(url);
      if (response.ok) return;
    } catch {
      // Keep waiting until the dev server accepts requests.
    }
    await new Promise((resolvePromise) => setTimeout(resolvePromise, 250));
  }

  throw new Error(`Timed out waiting for ${url}`);
}

async function ensureServer() {
  const url = new URL(target);
  const port = Number.parseInt(url.port || '80', 10);

  if (url.hostname === 'localhost' || url.hostname === '127.0.0.1') {
    if (await isPortOpen(port)) {
      await waitForTarget(target);
      return null;
    }
  }

  const child = spawn('bun', ['run', 'dev'], {
    cwd: webRoot,
    env: process.env,
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  child.stdout.on('data', (chunk) => process.stdout.write(chunk));
  child.stderr.on('data', (chunk) => process.stderr.write(chunk));

  await waitForTarget(target);
  return child;
}

async function waitForDone(page) {
  await page.waitForSelector('[data-testid="file-item-done"], [data-testid="file-item-optimized"]', {
    timeout: 90000,
  });
}

async function waitForIdle(page) {
  await page.waitForFunction(
    () =>
      !document.body.innerText.toLowerCase().includes('compressing') &&
      !document.body.innerText.toLowerCase().includes('processing image'),
    { timeout: 30000 },
  );
}

async function waitForCompressedImage(page, expected) {
  await expect(async () => {
    const metadata = await page.locator('img[alt="Compressed"]').evaluate(async (img) => {
      const blob = await fetch(img.src).then((response) => response.blob());
      const bitmap = await createImageBitmap(blob);
      return {
        type: blob.type,
        width: bitmap.width,
        height: bitmap.height,
      };
    });

    expect(metadata.type).toBe(expected.type);
    if (expected.width != null) expect(metadata.width).toBe(expected.width);
    if (expected.height != null) expect(metadata.height).toBe(expected.height);
  }).toPass({ timeout: 90000 });
}

async function validateWebpDownload(download, iteration) {
  await mkdir(downloadDir, { recursive: true });

  const filename = download.suggestedFilename();
  if (!filename.endsWith('.webp')) {
    throw new Error(`Expected .webp download, received ${filename}`);
  }

  const outputPath = join(downloadDir, `${iteration}-${filename}`);
  await download.saveAs(outputPath);
  const bytes = await readFile(outputPath);

  if (bytes.subarray(0, 4).toString('ascii') !== 'RIFF') {
    throw new Error(`Downloaded file is missing RIFF signature: ${outputPath}`);
  }

  if (bytes.subarray(8, 12).toString('ascii') !== 'WEBP') {
    throw new Error(`Downloaded file is missing WEBP signature: ${outputPath}`);
  }
}

function createStepRecords() {
  return new Map(
    steps.map((step) => [
      step,
      {
        durations: [],
        pass: 0,
        fail: 0,
      },
    ]),
  );
}

async function recordStep(records, step, action) {
  const record = records.get(step);
  const started = performance.now();

  try {
    await action();
    record.pass += 1;
  } catch (error) {
    record.fail += 1;
    throw error;
  } finally {
    record.durations.push(performance.now() - started);
  }
}

async function runIteration(browser, records, iteration) {
  const page = await browser.newPage({ acceptDownloads: true });

  try {
    await recordStep(records, '1. Load App', async () => {
      await page.goto(target, { waitUntil: 'domcontentloaded' });
      await expect(page.locator('h1')).toContainText('Go-Pixo', { timeout: 15000 });
    });

    await recordStep(records, '2. Upload Image', async () => {
      await page.locator('input[type="file"]').setInputFiles(fixturePath);
      await expect(page.getByText('generated-photo-like-1024x768.png')).toBeVisible({
        timeout: 15000,
      });
    });

    await recordStep(records, '3. Initial Compress', async () => {
      await waitForDone(page);
      await waitForIdle(page);
    });

    await recordStep(records, '4. Switch To WebP', async () => {
      await page.getByTestId('download-format-toggle').click();
      await page.getByTestId('download-as-WebP').click();
      await waitForDone(page);
      await waitForIdle(page);
      await waitForCompressedImage(page, { type: 'image/webp' });
    });

    await recordStep(records, '5. Resize Output', async () => {
      await page.getByPlaceholder('W').fill('512');
      await page.getByPlaceholder('H').fill('384');
      await waitForDone(page);
      await waitForIdle(page);
      await waitForCompressedImage(page, { type: 'image/webp', width: 512, height: 384 });
    });

    await recordStep(records, '6. Download WebP', async () => {
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }),
        page.getByTestId('download-primary').click(),
      ]);
      await validateWebpDownload(download, iteration);
    });
  } finally {
    await page.close();
  }
}

function printReport(records, totalDurations, passedIterations, elapsedSeconds) {
  const stepSummaries = steps.map((step) => {
    const record = records.get(step);
    return summarizeStep(step, record.durations, record.pass, record.fail, elapsedSeconds);
  });
  const summary = buildBenchmarkSummary(
    totalDurations,
    passedIterations,
    iterations,
    elapsedSeconds,
  );

  console.log('\nStep Breakdown:');
  console.log(formatStepTable(stepSummaries));
  console.log('');
  console.log(
    `E2E Total:    Avg ${formatDuration(summary.avg)} | P50 ${formatDuration(summary.p50)} | P95 ${formatDuration(summary.p95)} | P99 ${formatDuration(summary.p99)}`,
  );
  console.log(
    `Success Rate: ${summary.successRate.toFixed(1)}% (${summary.passedIterations}/${summary.totalIterations})`,
  );
  console.log(
    `Throughput:   iterations=${formatThroughput(summary.submittedThroughput)} completion=${formatThroughput(summary.completedThroughput)}`,
  );
  console.log(`Config:       Iterations ${iterations} | Mode full-flow | Target ${target}`);
}

await readFile(fixturePath);
await rm(downloadDir, { recursive: true, force: true });

let server = null;
let browser = null;
const records = createStepRecords();
const totalDurations = [];
let passedIterations = 0;
const suiteStarted = performance.now();

try {
  server = await ensureServer();
  browser = await chromium.launch();

  for (let iteration = 1; iteration <= iterations; iteration += 1) {
    const started = performance.now();
    try {
      await runIteration(browser, records, iteration);
      totalDurations.push(performance.now() - started);
      passedIterations += 1;
    } catch (error) {
      console.error(`Iteration ${iteration} failed: ${error.message}`);
    }
  }
} finally {
  if (browser != null) await browser.close();
  if (server != null) {
    server.kill('SIGTERM');
  }
}

const elapsedSeconds = (performance.now() - suiteStarted) / 1000;
printReport(records, totalDurations, passedIterations, elapsedSeconds);

if (passedIterations === 0) {
  process.exitCode = 1;
}
