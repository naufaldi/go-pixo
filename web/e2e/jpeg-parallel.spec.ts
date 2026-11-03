import { test, expect, type Browser } from '@playwright/test'

// #128: a large JPEG encode runs across band workers. Making band worker
// creation fail forces the serial fallback, which must produce the same bytes.
async function encodeLargeJpeg(browser: Browser, blockBandWorkers: boolean) {
  const context = await browser.newContext()
  const patched: Promise<unknown>[] = []
  let bandWorkers = 0
  try {
    const page = await context.newPage()
    page.on('worker', worker => {
      if (/jpegBand\.worker/.test(worker.url())) bandWorkers++
      // Network routes do not reach nested worker scripts, so break the
      // constructor inside each compression worker instead.
      else if (blockBandWorkers && /\/worker[.-]/.test(worker.url())) {
        patched.push(worker.evaluate(() => {
          (self as unknown as { Worker: unknown }).Worker = class { constructor() { throw new Error('band workers blocked') } }
        }))
      }
    })
    // Compression workers start Go, and then band workers, only after this.
    if (blockBandWorkers) await context.route('**/main.wasm', async route => { await Promise.all(patched); await route.continue() })
    await page.goto('/')
    await expect(page.locator('h1')).toContainText('Go-Pixo')
    const cores = await page.evaluate(() => navigator.hardwareConcurrency)
    await page.getByRole('combobox', { name: 'Output format' }).selectOption('jpeg')
    // 2400x1800 (4.32 MP) is just above the parallel threshold.
    await page.evaluate(async () => {
      const width = 2400, height = 1800
      const canvas = new OffscreenCanvas(width, height)
      const ctx = canvas.getContext('2d')!
      const image = ctx.createImageData(width, height)
      for (let y = 0; y < height; y++) for (let x = 0; x < width; x++) {
        const i = (y * width + x) * 4
        image.data[i] = (x * 7 + y * 3 + ((x * y) % 13)) & 255
        image.data[i + 1] = (128 + 100 * Math.sin(x / 9) * Math.cos(y / 7)) & 255
        image.data[i + 2] = (x ^ y) & 255
        image.data[i + 3] = 255
      }
      ctx.putImageData(image, 0, 0)
      const blob = await canvas.convertToBlob({ type: 'image/png' })
      const transfer = new DataTransfer()
      transfer.items.add(new File([blob], 'large.png', { type: 'image/png' }))
      const input = document.querySelector<HTMLInputElement>('input[type="file"]')!
      input.files = transfer.files
      input.dispatchEvent(new Event('change', { bubbles: true }))
    })
    await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 0 failed', { timeout: 60000 })
    const output = await page.locator('img[alt="Compressed"]').evaluate(async (img: HTMLImageElement) => {
      const blob = await fetch(img.src).then(r => r.blob())
      const digest = await crypto.subtle.digest('SHA-256', await blob.arrayBuffer())
      return { type: blob.type, size: blob.size, sha256: Array.from(new Uint8Array(digest), b => b.toString(16).padStart(2, '0')).join('') }
    })
    return { cores, bandWorkers, output }
  } finally {
    await context.close()
  }
}

test('large JPEG encode across band workers matches the serial fallback byte for byte', async ({ browser }) => {
  const parallel = await encodeLargeJpeg(browser, false)
  test.skip(parallel.cores < 4, 'Parallel JPEG needs at least 4 logical cores')
  // The compression worker encodes one band itself.
  expect(parallel.bandWorkers).toBe(Math.min(parallel.cores - 1, 4) - 1)
  const serial = await encodeLargeJpeg(browser, true)
  expect(serial.bandWorkers).toBe(0)
  expect(parallel.output.type).toBe('image/jpeg')
  expect(parallel.output).toEqual(serial.output)
})
