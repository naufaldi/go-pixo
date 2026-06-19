/**
 * Browser verification — run while dev server is up:
 *   node e2e/browser-verify.mjs
 */
import { chromium } from '@playwright/test';
import { writeFileSync } from 'fs';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';

const dir = dirname(fileURLToPath(import.meta.url));
const fixtureDir = join(dir, 'fixtures', 'browser-verify');
const report = [];

function log(id, pass, detail = {}) {
  report.push({ id, pass, ...detail, at: new Date().toISOString() });
  console.log(`${pass ? 'PASS' : 'FAIL'} ${id}`, detail.summary ?? detail);
}

async function uploadCanvas(page, name, w, h, drawBody) {
  await page.evaluate(
    async ({ name, w, h, drawBody }) => {
      const drawFn = new Function('ctx', 'w', 'h', drawBody);
      const canvas = document.createElement('canvas');
      canvas.width = w;
      canvas.height = h;
      drawFn(canvas.getContext('2d'), w, h);
      const blob = await new Promise((r) => canvas.toBlob(r, 'image/png'));
      const file = new File([blob], name, { type: 'image/png' });
      const input = document.querySelector('input[type=file]');
      const dt = new DataTransfer();
      dt.items.add(file);
      input.files = dt.files;
      input.dispatchEvent(new Event('change', { bubbles: true }));
    },
    { name, w, h, drawBody },
  );
}

async function waitDone(page, timeout = 90000) {
  await page.waitForSelector(
    '[data-testid="file-item-done"], [data-testid="file-item-optimized"]',
    { timeout },
  );
  await page.waitForFunction(
    () =>
      !document.body.innerText.toLowerCase().includes('compressing') &&
      !document.body.innerText.toLowerCase().includes('processing image'),
    { timeout: 30000 },
  );
}

async function selectDownloadFormat(page, label) {
  await page.getByTestId('download-format-toggle').click();
  await page.getByTestId(`download-as-${label}`).click();
}

async function readState(page) {
  return page.evaluate(() => {
    return {
      itemStatus: document.querySelector('[data-testid^="file-item-"]')?.getAttribute('data-testid'),
      compressed: document.body.innerText.match(/Compressed: [^\n]+/)?.[0] ?? null,
      savings: document.body.innerText.match(/Saved [^\n]+|No smaller output[^\n]*/)?.[0] ?? null,
      slider: document.querySelector('input[type=range]')?.value,
      presetLabel: document.body.innerText.match(/Ultra \(smallest\)|Smaller|Balanced|Faster \(quickest\)/)?.[0] ?? null,
      lossless: document.querySelector('input[type=checkbox]')?.checked,
      w: document.querySelector('input[placeholder="W"]')?.value,
      h: document.querySelector('input[placeholder="H"]')?.value,
      downloadMenu: !!document.querySelector('[data-testid="download-format-toggle"]'),
      downloadAsJpeg: !!document.querySelector('[data-testid="download-as-JPEG"]'),
      downloadAll: [...document.querySelectorAll('button')].some((b) => b.textContent === 'Download All'),
      downloadZip: [...document.querySelectorAll('button')].some((b) => b.textContent === 'Download ZIP'),
    };
  });
}

const browser = await chromium.launch();
const page = await browser.newPage();

try {
  await page.goto('http://localhost:5173/');
  log('01-app-load', (await page.locator('h1').textContent())?.includes('Go-Pixo'));

  await uploadCanvas(
    page,
    'verify-noisy.png',
    800,
    600,
    `for (let y = 0; y < h; y += 4) for (let x = 0; x < w; x += 4) { ctx.fillStyle = 'rgb(' + ((x+y)%256) + ',' + ((x*2)%256) + ',' + ((y*2)%256) + ')'; ctx.fillRect(x, y, 4, 4); }`,
  );
  await waitDone(page);
  log('02-initial-compress', true, { summary: await readState(page) });

  const slider = page.locator('input[type=range]');
  await slider.fill('0');
  await waitDone(page);
  log('03-slider-ultra', (await slider.inputValue()) === '0', { summary: await readState(page) });

  await slider.fill('3');
  await waitDone(page);
  log('04-slider-fast', (await slider.inputValue()) === '3', { summary: await readState(page) });

  for (const fmt of ['JPEG', 'WebP', 'PNG']) {
    await selectDownloadFormat(page, fmt);
    await waitDone(page);
    const s = await readState(page);
    log(`05-format-${fmt.toLowerCase()}`, !!s.itemStatus, { summary: s });
  }

  await selectDownloadFormat(page, 'AVIF');
  try {
    await waitDone(page, 45000);
    const s = await readState(page);
    log('06-format-avif', !!s.compressed, { summary: s });
  } catch (e) {
    log('06-format-avif', false, { summary: String(e) });
  }

  const cb = page.getByRole('checkbox', { name: 'Perfect Quality' });
  await cb.click();
  await waitDone(page);
  log('07-lossless-on', await cb.isChecked(), { summary: await readState(page) });
  await cb.click();
  await waitDone(page);
  log('08-lossless-off', !(await cb.isChecked()), { summary: await readState(page) });

  await selectDownloadFormat(page, 'JPEG');
  await waitDone(page);
  await page.getByPlaceholder('W').fill('320');
  await page.getByPlaceholder('H').fill('240');
  await waitDone(page);
  const resizeState = await readState(page);
  log('09-resize-jpeg', resizeState.w === '320' && resizeState.h === '240', { summary: resizeState });

  const [download] = await Promise.all([
    page.waitForEvent('download', { timeout: 10000 }).catch(() => null),
    page.getByRole('button', { name: 'Download', exact: true }).click(),
  ]);
  log('10-download-extension', download?.suggestedFilename()?.endsWith('.jpg') ?? false, {
    filename: download?.suggestedFilename() ?? null,
  });

  const imgs = await page.locator('img[alt="Original"], img[alt="Compressed"]').count();
  log('11-compare-view', imgs >= 2, { images: imgs });

  await page.evaluate(async () => {
    const canvas = document.createElement('canvas');
    canvas.width = 64;
    canvas.height = 64;
    const ctx = canvas.getContext('2d');
    ctx.fillStyle = '#ef4444';
    ctx.fillRect(0, 0, 64, 64);
    const blob = await new Promise((r) => canvas.toBlob(r, 'image/png'));
    const file = new File([blob], 'paste-red.png', { type: 'image/png' });
    const dt = new DataTransfer();
    dt.items.add(file);
    const event = new ClipboardEvent('paste', { bubbles: true, cancelable: true });
    Object.defineProperty(event, 'clipboardData', { value: dt });
    window.dispatchEvent(event);
  });
  await waitDone(page);
  const multi = await page.locator('h3:has-text("Files")').textContent();
  log('12-paste-second-file', multi?.includes('(2)') ?? false, { summary: multi });

  const sMulti = await readState(page);
  log('13-download-all-zip-buttons', sMulti.downloadAll && sMulti.downloadZip, { summary: sMulti });
} finally {
  await browser.close();
}

const out = join(dir, 'browser-verify-report.json');
writeFileSync(out, JSON.stringify(report, null, 2));
console.log(`\nWrote ${out}`);
console.log(`Passed ${report.filter((r) => r.pass).length}/${report.length}`);
