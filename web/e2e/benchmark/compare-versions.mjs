// Before/after benchmark for two running builds of the web app.
//
//   node e2e/benchmark/compare-versions.mjs --before=http://127.0.0.1:4101 \
//     --after=http://127.0.0.1:4102 [--iterations=2] [--heic=/path/extra.heic] [--out=results.json]
//     [--scenarios=ultra,balanced] [--fixtures=/private/dir [--manifest=fixtures.json]] [--batch] [--cold=false]
//
// --fixtures replaces the built-in images with every PNG/JPEG in a local folder,
// such as private user screenshots that must not be committed. --manifest lists
// {"files":[{"label":"shot-01","file":"name.png","sha256":"…"}]}: each file must
// match its hash, and only labels appear in the report. --batch also uploads all
// fixtures at once per scenario and records the batch result.
//
// Serve each build with `vite preview` first. Each scenario reloads the page,
// applies settings, uploads one image and waits for the batch result. Before and
// after runs alternate per scenario so warm-up and thermal drift hit both.
// Quality is measured in the page against the app's own "Original" preview:
// PSNR over RGB and SSIM over luma (8x8 windows).
import { chromium } from '@playwright/test';
import { createHash } from 'node:crypto';
import { mkdir, readdir, readFile, writeFile } from 'node:fs/promises';
import { basename, dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const webRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..', '..');
const arg = (name, fallback) => process.argv.find(value => value.startsWith(`--${name}=`))?.split('=').slice(1).join('=') ?? fallback;
const targets = { before: arg('before', 'http://127.0.0.1:4101'), after: arg('after', 'http://127.0.0.1:4102') };
const iterations = Number.parseInt(arg('iterations', '2'), 10);
const outPath = resolve(arg('out', join(webRoot, 'test-results', 'compare-versions.json')));
const workDir = join(webRoot, 'test-results', 'compare-fixtures');

const allScenarios = [
  { id: 'ultra', label: 'Ultra (smallest)', slider: '0', lossless: false, output: 'same' },
  { id: 'balanced', label: 'Balanced (default)', slider: '2', lossless: false, output: 'same' },
  { id: 'smaller', label: 'Smaller file', slider: '1', lossless: false, output: 'same' },
  { id: 'faster', label: 'Faster', slider: '3', lossless: false, output: 'same' },
  { id: 'perfect', label: 'Lossless', slider: '2', lossless: true, output: 'same' },
  { id: 'auto', label: 'Auto (smallest)', slider: '2', lossless: false, output: 'auto' },
];
// Without --scenarios, keep the established default set; Ultra is opt-in.
const scenarios = arg('scenarios') ? allScenarios.filter(scenario => arg('scenarios').split(',').includes(scenario.id))
  : allScenarios.filter(scenario => scenario.id !== 'ultra');

// Local fixtures, verified against the manifest's SHA-256 when one is given.
async function localFixtures(dir, manifestPath) {
  if (!manifestPath) {
    const names = (await readdir(dir)).filter(name => /\.(png|jpe?g)$/i.test(name)).sort();
    return names.map(name => ({ file: join(dir, name), label: name }));
  }
  const manifest = JSON.parse(await readFile(manifestPath, 'utf8'));
  return Promise.all(manifest.files.map(async entry => {
    const file = join(dir, entry.file);
    const actual = createHash('sha256').update(await readFile(file)).digest('hex');
    if (actual !== entry.sha256) throw new Error(`${entry.label}: SHA-256 ${actual} does not match the manifest`);
    return { file, label: entry.label };
  }));
}

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
const waitForBatchResult = (page, count = 1) => page.waitForFunction(count => {
  const match = /(\d+) completed, (\d+) failed, 0 remaining/.exec(document.querySelector('[data-testid="batch-summary"]')?.textContent ?? '');
  return match !== null && Number(match[1]) + Number(match[2]) === count;
}, count, { polling: 'raf', timeout: 180_000 * count });

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

async function applyScenario(page, url, scenario) {
  await page.goto(url, { waitUntil: 'load' });
  await page.getByRole('slider', { name: 'Compression preset' }).fill(scenario.slider);
  const perfect = page.getByRole('checkbox', { name: 'Lossless' });
  if (scenario.lossless) await perfect.check(); else await perfect.uncheck();
  await page.getByRole('combobox', { name: 'Output format' }).selectOption(scenario.output);
}

// All fixtures in one upload, as a user drops a folder of screenshots.
async function runBatch(page, url, files, scenario) {
  await applyScenario(page, url, scenario);
  const started = Date.now();
  await page.locator('input[type=file]').setInputFiles(files);
  await waitForBatchResult(page, files.length);
  const elapsedMs = Date.now() - started;
  const summary = (await page.getByTestId('batch-summary').innerText()).replace(/\s+/g, ' ');
  const [, completed, failed] = /(\d+) completed, (\d+) failed/.exec(summary);
  const errors = [...new Set((await page.locator('body').innerText()).match(/failed to [^\n]*/g) ?? [])];
  return { elapsedMs, completed: Number(completed), failed: Number(failed), errors };
}

async function runScenario(page, url, file, scenario) {
  await applyScenario(page, url, scenario);
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
      const bitmap = await createImageBitmap(blob, { imageOrientation: 'from-image', premultiplyAlpha: 'none' });
      const canvas = new OffscreenCanvas(bitmap.width, bitmap.height);
      const context = canvas.getContext('2d');
      context.drawImage(bitmap, 0, 0);
      const alpha = context.getImageData(0, 0, canvas.width, canvas.height).data;
      // Match the app: transparent pixels are compared over white, as JPEG output is.
      context.globalCompositeOperation = 'destination-over';
      context.fillStyle = '#fff'; context.fillRect(0, 0, bitmap.width, bitmap.height); bitmap.close();
      return { blob, width: canvas.width, height: canvas.height, alpha, data: context.getImageData(0, 0, canvas.width, canvas.height).data };
    };
    const a = await load(original.src), b = await load(output.src);
    if (a.width !== b.width || a.height !== b.height) return { outputBytes: b.blob.size, mime: b.blob.type, width: b.width, height: b.height };
    // Alpha is compared exactly, before the white background is applied.
    let transparentPixels = 0, alphaMismatches = 0;
    for (let i = 3; i < a.alpha.length; i += 4) {
      if (a.alpha[i] !== 255) transparentPixels++;
      if (a.alpha[i] !== b.alpha[i]) alphaMismatches++;
    }
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
    return { outputBytes: b.blob.size, mime: b.blob.type, width: b.width, height: b.height, transparentPixels, alphaMismatches,
      psnr: identical ? Infinity : 10 * Math.log10(255 * 255 / mse), ssim: ssim / windows, identicalPixels: identical };
  });
  return { ok: true, elapsedMs, ...metrics };
}

const median = values => { const sorted = [...values].sort((x, y) => x - y); return sorted[Math.floor((sorted.length - 1) / 2)]; };

const browser = await chromium.launch();
try {
  const fixtures = arg('fixtures') ? await localFixtures(resolve(arg('fixtures')), arg('manifest') && resolve(arg('manifest'))) : [
    ...await makeJpegFixtures(browser),
    join(webRoot, 'e2e', 'fixtures', 'benchmark', 'codex-still-life.png'),
    join(webRoot, 'e2e', 'fixtures', 'browser-verify', 'indomaret-screenshot.png'),
    join(webRoot, 'src', 'interop', 'codecs-fixtures', 'rainbow-451x461.heic'),
    ...(arg('heic') ? [resolve(arg('heic'))] : []),
  ].map(file => ({ file, label: basename(file) }));
  const selected = fixtures.filter(({ label }) => !arg('only') || label.includes(arg('only')));
  const report = { date: new Date().toISOString(), targets, iterations, load: {}, runs: [], batches: [] };
  const coldInputs = {
    png: join(webRoot, 'e2e', 'fixtures', 'sample-32x32.png'),
    heic: join(webRoot, 'src', 'interop', 'codecs-fixtures', 'rainbow-451x461.heic'),
  };
  if (arg('cold') !== 'false') for (const [kind, file] of Object.entries(coldInputs)) for (const version of ['before', 'after']) {
    const loads = [];
    for (let i = 0; i < 3; i++) loads.push(await measureColdStart(browser, targets[version], file));
    const summary = Object.fromEntries(Object.keys(loads[0]).map(key => [key, key === 'ok' ? loads.every(load => load.ok) : median(loads.map(load => load[key]))]));
    report.load[`${kind}-${version}`] = summary;
    console.log('cold start', kind, version, summary);
  }
  const context = await browser.newContext();
  const page = await context.newPage();
  for (const { file, label } of selected) {
    const originalBytes = (await readFile(file)).length;
    for (const scenario of scenarios) {
      const samples = { before: [], after: [] };
      for (let i = 0; i < iterations; i++) for (const version of ['before', 'after']) {
        samples[version].push(await runScenario(page, targets[version], file, scenario));
      }
      for (const version of ['before', 'after']) {
        const ok = samples[version].filter(sample => sample.ok);
        const last = ok.at(-1) ?? samples[version].at(-1);
        const row = { image: label, originalBytes, scenario: scenario.id, scenarioLabel: scenario.label, version,
          ok: ok.length === samples[version].length, error: last.error?.replaceAll(basename(file), label),
          medianMs: median(samples[version].map(sample => sample.elapsedMs)),
          outputBytes: last.outputBytes, mime: last.mime, width: last.width, height: last.height,
          transparentPixels: last.transparentPixels, alphaMismatches: last.alphaMismatches,
          psnr: last.psnr, ssim: last.ssim, identicalPixels: last.identicalPixels };
        report.runs.push(row);
        console.log([row.image, row.scenario, version, row.ok ? `${row.medianMs}ms` : `FAIL ${row.error}`,
          row.outputBytes ?? '', row.mime ?? '', row.psnr?.toFixed?.(2) ?? '', row.ssim?.toFixed?.(4) ?? '',
          row.alphaMismatches === undefined ? '' : `alpha mismatches ${row.alphaMismatches}/${row.transparentPixels}`].join('\t'));
      }
    }
  }
  if (process.argv.includes('--batch')) for (const scenario of scenarios) for (const version of ['before', 'after']) {
    const files = selected.map(({ file }) => file);
    const batch = { scenario: scenario.id, version, files: files.length, ...await runBatch(page, targets[version], files, scenario) };
    report.batches.push(batch);
    console.log('batch', scenario.id, version, `${batch.completed} completed, ${batch.failed} failed, ${batch.elapsedMs}ms`, batch.errors.join(' | '));
  }
  await context.close();
  await mkdir(dirname(outPath), { recursive: true });
  await writeFile(outPath, JSON.stringify(report, (_key, value) => value === Infinity ? 'Infinity' : value, 2));
  console.log(`wrote ${outPath}`);
} finally {
  await browser.close();
}
