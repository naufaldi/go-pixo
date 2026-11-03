// Before/after benchmark for two running builds of the web app.
//
//   node e2e/benchmark/compare-versions.mjs --before=http://127.0.0.1:4101 \
//     --after=http://127.0.0.1:4102 [--iterations=2] [--heic=/path/extra.heic] [--out=results.json]
//
// Serve each build with `vite preview` first. Each scenario reloads the page,
// applies settings, uploads one image and waits for the batch result. Before and
// after runs alternate per scenario so warm-up and thermal drift hit both.
// Quality is measured in the page against the app's own "Original" preview:
// PSNR over RGB and SSIM over luma (8x8 windows).
import { chromium } from '@playwright/test';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { basename, dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..');
const arg = (name, fallback) => process.argv.find(value => value.startsWith(`--${name}=`))?.split('=').slice(1).join('=') ?? fallback;
const targets = { before: arg('before', 'http://127.0.0.1:4101'), after: arg('after', 'http://127.0.0.1:4102') };
const iterations = Number.parseInt(arg('iterations', '2'), 10);
const outPath = resolve(arg('out', join(webRoot, 'test-results', 'compare-versions.json')));
const workDir = join(webRoot, 'test-results', 'compare-fixtures');

const scenarios = [
  { id: 'balanced', label: 'Balanced (default)', slider: '2', lossless: false, output: 'same' },
  { id: 'smaller', label: 'Smaller file', slider: '1', lossless: false, output: 'same' },
  { id: 'faster', label: 'Faster', slider: '3', lossless: false, output: 'same' },
  { id: 'perfect', label: 'Lossless', slider: '2', lossless: true, output: 'same' },
  { id: 'auto', label: 'Auto (smallest)', slider: '2', lossless: false, output: 'auto' },
];

// JPEG fixtures come from Chromium's canvas encoder (standard Huffman tables,
// like most camera and phone encoders), derived from the committed Codex photo.
async function makeJpegFixtures(browser) {
  await mkdir(workDir, { recursive: true });
  const page = await browser.newPage();
  const source = await readFile(join(webRoot, 'e2e', 'fixtures', 'benchmark', 'codex-still-life.png'));
  const made = await page.evaluate(async base64 => {
    const blob = await (await fetch(`data:image/png;base64,${base64}`)).blob();
    const bitmap = await createImageBitmap(blob);
    const encode = async (width, height, quality) => {
      const canvas = new OffscreenCanvas(width, height);
      const context = canvas.getContext('2d');
      context.imageSmoothingQuality = 'high';
      context.drawImage(bitmap, 0, 0, width, height);
      const jpeg = await canvas.convertToBlob({ type: 'image/jpeg', quality });
      const bytes = new Uint8Array(await jpeg.arrayBuffer());
      let binary = '';
      for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
      return btoa(binary);
    };
    return { photo: await encode(1536, 1024, 0.92), phone: await encode(4032, 2688, 0.9) };
  }, source.toString('base64'));
  await page.close();
  const files = {
    'codex-photo-1536x1024-q92.jpg': made.photo,
    'codex-photo-4032x2688-q90.jpg': made.phone,
  };
  for (const [name, data] of Object.entries(files)) await writeFile(join(workDir, name), Buffer.from(data, 'base64'));
  return Object.keys(files).map(name => join(workDir, name));
}

// Locator waitFor backs off to 500 ms polls, which quantizes timings; poll
// every frame instead.
const waitForBatchResult = page => page.waitForFunction(
  () => /1 completed|1 failed/.test(document.querySelector('[data-testid="batch-summary"]')?.textContent ?? ''),
  null, { polling: 'raf', timeout: 180_000 });

// Cold start on a throttled connection (~20 Mbps down, 40 ms RTT, empty cache):
// page load, then the first compression, which downloads the Go WASM and any
// lazily loaded decoder. Bytes are what the page and its workers fetched.
async function measureColdStart(browser, url, file) {
  const context = await browser.newContext();
  const page = await context.newPage();
  const cdp = await context.newCDPSession(page);
  await cdp.send('Network.enable');
  await cdp.send('Network.emulateNetworkConditions', { offline: false, latency: 40,
    downloadThroughput: 20e6 / 8, uploadThroughput: 5e6 / 8 });
  let bytes = 0;
  // Worker fetches bypass page-level throttling hooks, so count every response body.
  context.on('response', async response => { try { bytes += (await response.body()).length; } catch {} });
  const started = Date.now();
  await page.goto(url, { waitUntil: 'load' });
  await page.locator('input[type=file]').waitFor({ state: 'attached' });
  const pageReadyMs = Date.now() - started;
  const pageBytes = bytes;
  await page.locator('input[type=file]').setInputFiles(file);
  await waitForBatchResult(page);
  const firstResultMs = Date.now() - started;
  const ok = /1 completed/.test(await page.getByTestId('batch-summary').innerText());
  await context.close();
  return { pageReadyMs, pageBytes, firstResultMs, firstResultBytes: bytes, ok };
}

async function runScenario(page, url, file, scenario) {
  await page.goto(url, { waitUntil: 'load' });
  await page.getByRole('slider', { name: 'Compression preset' }).fill(scenario.slider);
  const perfect = page.getByRole('checkbox', { name: 'Lossless' });
  if (scenario.lossless) await perfect.check(); else await perfect.uncheck();
  await page.getByRole('combobox', { name: 'Output format' }).selectOption(scenario.output);
  const started = Date.now();
  await page.locator('input[type=file]').setInputFiles(file);
  const summary = page.getByTestId('batch-summary');
  await waitForBatchResult(page);
  const elapsedMs = Date.now() - started;
  const text = await summary.innerText();
  if (/1 failed/.test(text)) {
    const error = await page.getByRole('alert').filter({ hasNotText: /offline/i }).first().innerText().catch(() => 'failed');
    return { ok: false, elapsedMs, error: error.split('\n')[0].slice(0, 160) };
  }
  const metrics = await page.evaluate(async () => {
    const original = document.querySelector('img[alt="Original"]');
    const output = document.querySelector('img[alt="Compressed"]');
    const load = async src => {
      const blob = await fetch(src).then(response => response.blob());
      const bitmap = await createImageBitmap(blob, { imageOrientation: 'from-image' });
      const canvas = new OffscreenCanvas(bitmap.width, bitmap.height);
      const context = canvas.getContext('2d');
      // Match the app: transparent pixels are compared over white, as JPEG output is.
      context.fillStyle = '#fff'; context.fillRect(0, 0, bitmap.width, bitmap.height);
      context.drawImage(bitmap, 0, 0); bitmap.close();
      return { blob, width: canvas.width, height: canvas.height, data: context.getImageData(0, 0, canvas.width, canvas.height).data };
    };
    const a = await load(original.src), b = await load(output.src);
    if (a.width !== b.width || a.height !== b.height) return { outputBytes: b.blob.size, mime: b.blob.type, width: b.width, height: b.height };
    let squared = 0, identical = true;
    const lumaA = new Float32Array(a.width * a.height), lumaB = new Float32Array(a.width * a.height);
    for (let i = 0, p = 0; i < a.data.length; i += 4, p++) {
      for (let c = 0; c < 3; c++) { const d = a.data[i + c] - b.data[i + c]; squared += d * d; if (d) identical = false; }
      lumaA[p] = 0.299 * a.data[i] + 0.587 * a.data[i + 1] + 0.114 * a.data[i + 2];
      lumaB[p] = 0.299 * b.data[i] + 0.587 * b.data[i + 1] + 0.114 * b.data[i + 2];
    }
    const mse = squared / (a.width * a.height * 3);
    const c1 = (0.01 * 255) ** 2, c2 = (0.03 * 255) ** 2;
    let ssim = 0, windows = 0;
    for (let y = 0; y + 8 <= a.height; y += 8) for (let x = 0; x + 8 <= a.width; x += 8) {
      let ma = 0, mb = 0;
      for (let j = 0; j < 8; j++) for (let i = 0; i < 8; i++) { const p = (y + j) * a.width + x + i; ma += lumaA[p]; mb += lumaB[p]; }
      ma /= 64; mb /= 64;
      let va = 0, vb = 0, cov = 0;
      for (let j = 0; j < 8; j++) for (let i = 0; i < 8; i++) {
        const p = (y + j) * a.width + x + i, da = lumaA[p] - ma, db = lumaB[p] - mb;
        va += da * da; vb += db * db; cov += da * db;
      }
      va /= 63; vb /= 63; cov /= 63;
      ssim += ((2 * ma * mb + c1) * (2 * cov + c2)) / ((ma * ma + mb * mb + c1) * (va + vb + c2)); windows++;
    }
    return { outputBytes: b.blob.size, mime: b.blob.type, width: b.width, height: b.height,
      psnr: identical ? Infinity : 10 * Math.log10(255 * 255 / mse), ssim: ssim / windows, identicalPixels: identical };
  });
  return { ok: true, elapsedMs, ...metrics };
}

const median = values => { const sorted = [...values].sort((x, y) => x - y); return sorted[Math.floor((sorted.length - 1) / 2)]; };

const browser = await chromium.launch();
try {
  const fixtures = [
    ...await makeJpegFixtures(browser),
    join(webRoot, 'e2e', 'fixtures', 'benchmark', 'codex-still-life.png'),
    join(webRoot, 'e2e', 'fixtures', 'browser-verify', 'indomaret-screenshot.png'),
    join(webRoot, 'src', 'interop', 'codecs-fixtures', 'rainbow-451x461.heic'),
    ...(arg('heic') ? [resolve(arg('heic'))] : []),
  ].filter(file => !arg('only') || basename(file).includes(arg('only')));
  const report = { date: new Date().toISOString(), targets, iterations, load: {}, runs: [] };
  const coldInputs = {
    png: join(webRoot, 'e2e', 'fixtures', 'sample-32x32.png'),
    heic: join(webRoot, 'src', 'interop', 'codecs-fixtures', 'rainbow-451x461.heic'),
  };
  for (const [kind, file] of Object.entries(coldInputs)) for (const version of ['before', 'after']) {
    const loads = [];
    for (let i = 0; i < 3; i++) loads.push(await measureColdStart(browser, targets[version], file));
    const summary = Object.fromEntries(Object.keys(loads[0]).map(key => [key, key === 'ok' ? loads.every(load => load.ok) : median(loads.map(load => load[key]))]));
    report.load[`${kind}-${version}`] = summary;
    console.log('cold start', kind, version, summary);
  }
  const context = await browser.newContext();
  const page = await context.newPage();
  for (const file of fixtures) {
    const originalBytes = (await readFile(file)).length;
    for (const scenario of scenarios) {
      const samples = { before: [], after: [] };
      for (let i = 0; i < iterations; i++) for (const version of ['before', 'after']) {
        samples[version].push(await runScenario(page, targets[version], file, scenario));
      }
      for (const version of ['before', 'after']) {
        const ok = samples[version].filter(sample => sample.ok);
        const last = ok.at(-1) ?? samples[version].at(-1);
        const row = { image: basename(file), originalBytes, scenario: scenario.id, scenarioLabel: scenario.label, version,
          ok: ok.length === samples[version].length, error: last.error,
          medianMs: median(samples[version].map(sample => sample.elapsedMs)),
          outputBytes: last.outputBytes, mime: last.mime, psnr: last.psnr, ssim: last.ssim, identicalPixels: last.identicalPixels };
        report.runs.push(row);
        console.log([row.image, row.scenario, version, row.ok ? `${row.medianMs}ms` : `FAIL ${row.error}`,
          row.outputBytes ?? '', row.mime ?? '', row.psnr?.toFixed?.(2) ?? '', row.ssim?.toFixed?.(4) ?? ''].join('\t'));
      }
    }
  }
  await context.close();
  await mkdir(dirname(outPath), { recursive: true });
  await writeFile(outPath, JSON.stringify(report, (_key, value) => value === Infinity ? 'Infinity' : value, 2));
  console.log(`wrote ${outPath}`);
} finally {
  await browser.close();
}
