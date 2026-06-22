import { test, expect } from '@playwright/test'
import { mkdir, readFile } from 'fs/promises'
import { dirname, join } from 'path'
import { fileURLToPath } from 'url'

const e2eDir = dirname(fileURLToPath(import.meta.url))
const samplePng = join(e2eDir, 'fixtures', 'sample-32x32.png')
const flatBluePng = join(e2eDir, 'fixtures', 'browser-verify', 'flat-blue-256.png')
const gradientPng = join(e2eDir, 'fixtures', 'browser-verify', 'gradient-800x600.png')
const indomaretScreenshot = join(
  e2eDir,
  'fixtures',
  'browser-verify',
  'indomaret-screenshot.png',
)

async function openApp(page: import('@playwright/test').Page): Promise<void> {
  await page.goto('/')
  await expect(page.locator('h1')).toContainText('Go-Pixo')
}

async function uploadSamplePng(page: import('@playwright/test').Page): Promise<void> {
  await openApp(page)
  await page.locator('input[type="file"]').setInputFiles(samplePng)
}

async function uploadGeneratedPng(
  page: import('@playwright/test').Page,
  options: { name: string; width: number; height: number; transparent?: boolean },
): Promise<void> {
  await page.evaluate(async ({ name, width, height, transparent }) => {
    const canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')!

    if (transparent) {
      ctx.clearRect(0, 0, width, height)
      ctx.fillStyle = 'rgba(220, 20, 60, 0.95)'
      ctx.fillRect(Math.floor(width / 4), Math.floor(height / 4), Math.floor(width / 2), Math.floor(height / 2))
    } else {
      const imageData = ctx.createImageData(width, height)
      for (let y = 0; y < height; y += 1) {
        for (let x = 0; x < width; x += 1) {
          const i = (y * width + x) * 4
          imageData.data[i] = (x * 17 + y * 3) % 256
          imageData.data[i + 1] = (x * 5 + y * 11) % 256
          imageData.data[i + 2] = (x * 13 + y * 7) % 256
          imageData.data[i + 3] = 255
        }
      }
      ctx.putImageData(imageData, 0, 0)
    }

    const blob = await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob((value) => {
        if (value == null) reject(new Error('failed to create PNG fixture'))
        else resolve(value)
      }, 'image/png')
    })
    const file = new File([blob], name, { type: 'image/png' })
    const input = document.querySelector<HTMLInputElement>('input[type="file"]')
    if (input == null) throw new Error('file input not found')
    const transfer = new DataTransfer()
    transfer.items.add(file)
    input.files = transfer.files
    input.dispatchEvent(new Event('change', { bubbles: true }))
  }, options)
}

async function waitForCompression(page: import('@playwright/test').Page): Promise<void> {
  await page.waitForSelector('[data-testid="file-item-done"], [data-testid="file-item-optimized"]', {
    timeout: 90000,
  })
}

async function waitForTwoCompressions(page: import('@playwright/test').Page): Promise<void> {
  await expect(async () => {
    const done = await page
      .locator('[data-testid="file-item-done"], [data-testid="file-item-optimized"]')
      .count()
    expect(done).toBeGreaterThanOrEqual(2)
  }).toPass({ timeout: 90000 })
}

async function waitForIdle(page: import('@playwright/test').Page): Promise<void> {
  await page.waitForFunction(
    () =>
      !document.body.innerText.toLowerCase().includes('compressing') &&
      !document.body.innerText.toLowerCase().includes('processing image'),
    { timeout: 30000 },
  )
}

async function expectSelectedCompressedImage(
  page: import('@playwright/test').Page,
  expectedMime: string,
  expectedSignature: string,
): Promise<void> {
  const metadata = await page
    .locator('img[alt="Compressed"]')
    .evaluate(async (img: HTMLImageElement) => {
      const blob = await fetch(img.src).then((response) => response.blob())
      const bytes = new Uint8Array(await blob.arrayBuffer())
      const bitmap = await createImageBitmap(blob)
      const header = Array.from(bytes.slice(0, 12))
      return { type: blob.type, size: blob.size, width: bitmap.width, height: bitmap.height, header }
    })

  expect(metadata.type).toBe(expectedMime)
  expect(metadata.size).toBeGreaterThan(0)

  if (expectedSignature === 'webp') {
    const headerText = String.fromCharCode(...metadata.header)
    expect(headerText.slice(0, 4)).toBe('RIFF')
    expect(headerText.slice(8, 12)).toBe('WEBP')
  }
}

async function expectDownloadedWebp(download: import('@playwright/test').Download): Promise<void> {
  expect(download.suggestedFilename()).toMatch(/\.webp$/)
  const downloadDir = join(e2eDir, 'test-results')
  await mkdir(downloadDir, { recursive: true })
  const outputPath = join(downloadDir, download.suggestedFilename())
  await download.saveAs(outputPath)
  const bytes = await readFile(outputPath)
  expect(bytes.subarray(0, 4).toString('ascii')).toBe('RIFF')
  expect(bytes.subarray(8, 12).toString('ascii')).toBe('WEBP')
}

async function expectDownloadedJpeg(download: import('@playwright/test').Download): Promise<void> {
  expect(download.suggestedFilename()).toMatch(/\.jpg$/)
  const downloadDir = join(e2eDir, 'test-results')
  await mkdir(downloadDir, { recursive: true })
  const outputPath = join(downloadDir, download.suggestedFilename())
  await download.saveAs(outputPath)
  const bytes = await readFile(outputPath)
  expect(bytes.subarray(0, 2).toString('hex')).toBe('ffd8')
  expect(bytes.subarray(-2).toString('hex')).toBe('ffd9')
}

async function readCompressedImagePixel(
  page: import('@playwright/test').Page,
  x: number,
  y: number,
): Promise<{ type: string; width: number; height: number; pixel: number[] }> {
  return page.locator('img[alt="Compressed"]').evaluate(async (img: HTMLImageElement, point) => {
    const blob = await fetch(img.src).then((response) => response.blob())
    const bitmap = await createImageBitmap(blob)
    const canvas = document.createElement('canvas')
    canvas.width = bitmap.width
    canvas.height = bitmap.height
    const ctx = canvas.getContext('2d')!
    ctx.drawImage(bitmap, 0, 0)
    const pixel = Array.from(ctx.getImageData(point.x, point.y, 1, 1).data)
    return { type: blob.type, width: bitmap.width, height: bitmap.height, pixel }
  }, { x, y })
}

const MIN_PRESET_SLIDER_WIDTH_PX = 120

test.describe('go-pixo', () => {
  test('app loads with correct title', async ({ page }) => {
    await openApp(page)
  })

  test('PNG file is accepted and compresses to Done status', async ({ page }) => {
    await uploadSamplePng(page)
    await expect(page.getByTestId('compression-percent')).toBeVisible({ timeout: 15000 })
    await waitForCompression(page)
  })

  test('compression slider updates preset label state', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    const slider = page.locator('input[type="range"]')
    await expect(slider).toHaveValue('2')
    await slider.fill('0')
    await expect(slider).toHaveValue('0')
  })

  test('download format menu offers JPEG and triggers recompress', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    await page.getByTestId('download-format-toggle').click()
    await expect(page.getByTestId('download-as-JPEG')).toBeVisible()
    await page.getByTestId('download-as-JPEG').click()
    await waitForCompression(page)
  })

  test('JPEG output selection updates labels and downloads JPEG bytes', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    await expect(page.getByTestId('bottom-bar')).toContainText('Output: PNG')
    await expect(page.getByTestId('bottom-bar')).not.toContainText('Format: PNG')

    await page.getByTestId('download-format-toggle').click()
    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByTestId('download-as-JPEG').click(),
    ])
    await waitForCompression(page)

    await expect(page.getByTestId('bottom-bar')).toContainText('Output: JPEG')
    await expect(page.getByTestId('bottom-bar')).not.toContainText('Format: PNG')
    await expect(page.getByText('Input: PNG')).toBeVisible()
    await expectDownloadedJpeg(download)
  })

  test('PNG to JPEG Ultra finishes without fake 91 percent progress and downloads valid JPEG', async ({ page }) => {
    await openApp(page)
    await uploadGeneratedPng(page, { name: 'generated-screenshot.png', width: 640, height: 480 })
    await waitForCompression(page)

    await page.getByTestId('download-format-toggle').click()
    const [initialJpegDownload] = await Promise.all([
      page.waitForEvent('download'),
      page.getByTestId('download-as-JPEG').click(),
    ])
    await waitForIdle(page)
    await expectDownloadedJpeg(initialJpegDownload)

    const progress = page.getByTestId('compression-percent')
    const progressStarted = progress.waitFor({ state: 'visible', timeout: 5000 }).catch(() => null)
    const start = Date.now()
    await page.locator('input[type="range"]').fill('0')
    await progressStarted
    await page.waitForTimeout(1200)
    if (await progress.isVisible()) {
      await expect(progress).not.toHaveText('91%')
    }
    await waitForIdle(page)
    expect(Date.now() - start).toBeLessThan(15000)

    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByTestId('download-primary').click(),
    ])
    await expectDownloadedJpeg(download)

    const metadata = await readCompressedImagePixel(page, 10, 10)
    expect(metadata.type).toBe('image/jpeg')
    expect(metadata.width).toBe(640)
    expect(metadata.height).toBe(480)
  })

  test('transparent PNG to JPEG composites transparent pixels on white', async ({ page }) => {
    await openApp(page)
    await uploadGeneratedPng(page, { name: 'transparent.png', width: 48, height: 48, transparent: true })
    await waitForCompression(page)

    await page.getByTestId('download-format-toggle').click()
    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByTestId('download-as-JPEG').click(),
    ])
    await waitForIdle(page)
    await expectDownloadedJpeg(download)

    const corner = await readCompressedImagePixel(page, 1, 1)
    expect(corner.type).toBe('image/jpeg')
    expect(corner.pixel[0]).toBeGreaterThanOrEqual(240)
    expect(corner.pixel[1]).toBeGreaterThanOrEqual(240)
    expect(corner.pixel[2]).toBeGreaterThanOrEqual(240)
  })

  test('multiple images convert to WebP once and download valid WebP bytes', async ({ page }) => {
    const compressStarts: string[] = []
    const consoleErrors: string[] = []
    page.on('console', (message) => {
      const text = message.text()
      if (text.includes('[worker] compress start')) compressStarts.push(text)
      if (message.type() === 'error') consoleErrors.push(text)
    })
    page.on('pageerror', (error) => consoleErrors.push(error.message))

    await openApp(page)
    await page.locator('input[type="file"]').setInputFiles([flatBluePng, gradientPng])
    await waitForTwoCompressions(page)
    await waitForIdle(page)
    expect(compressStarts).toHaveLength(2)

    await page.getByTestId('download-format-toggle').click()
    await page.getByTestId('download-as-WebP').click()
    await waitForTwoCompressions(page)
    await waitForIdle(page)
    await page.waitForTimeout(1000)

    expect(compressStarts).toHaveLength(4)
    expect(consoleErrors).toEqual([])
    await expectSelectedCompressedImage(page, 'image/webp', 'webp')

    await page.getByPlaceholder('W').fill('128')
    await page.getByPlaceholder('H').fill('96')
    await waitForTwoCompressions(page)
    await waitForIdle(page)
    const resized = await page
      .locator('img[alt="Compressed"]')
      .evaluate(async (img: HTMLImageElement) => {
        const blob = await fetch(img.src).then((response) => response.blob())
        const bitmap = await createImageBitmap(blob)
        return { width: bitmap.width, height: bitmap.height, type: blob.type }
      })
    expect(resized).toEqual({ width: 128, height: 96, type: 'image/webp' })

    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.getByTestId('download-primary').click(),
    ])
    await expectDownloadedWebp(download)
  })

  test('WebP format change during multi-image processing settles without a loop', async ({ page }) => {
    const compressStarts: string[] = []
    const consoleErrors: string[] = []
    page.on('console', (message) => {
      const text = message.text()
      if (text.includes('[worker] compress start')) compressStarts.push(text)
      if (message.type() === 'error') consoleErrors.push(text)
    })
    page.on('pageerror', (error) => consoleErrors.push(error.message))

    await openApp(page)
    await page.locator('input[type="file"]').setInputFiles([flatBluePng, gradientPng])
    await expect(page.getByText('Files (2)')).toBeVisible()

    const downloadPromise = page.waitForEvent('download', { timeout: 90000 }).catch(() => null)
    await page.getByTestId('download-format-toggle').click()
    await page.getByTestId('download-as-WebP').click()

    await waitForTwoCompressions(page)
    await waitForIdle(page)
    await page.waitForTimeout(1500)

    expect(compressStarts.length).toBeGreaterThanOrEqual(2)
    expect(compressStarts.length).toBeLessThanOrEqual(4)
    expect(consoleErrors).toEqual([])
    await expectSelectedCompressedImage(page, 'image/webp', 'webp')

    const download = await downloadPromise
    expect(download?.suggestedFilename()).toMatch(/\.webp$/)
  })

  test('perfect quality checkbox toggles', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    const checkbox = page.getByRole('checkbox', { name: 'Perfect Quality' })
    await expect(checkbox).not.toBeChecked()
    await checkbox.click()
    await expect(checkbox).toBeChecked()
    await waitForCompression(page)
  })

  test('resize inputs accept dimensions', async ({ page }) => {
    await uploadSamplePng(page)
    await waitForCompression(page)

    const widthInput = page.getByPlaceholder('W')
    await widthInput.fill('16')
    await expect(widthInput).toHaveValue('16')
  })

  test('bottom bar keeps preset slider usable after multi-file completion', async ({ page }) => {
    await openApp(page)
    await page.locator('input[type="file"]').setInputFiles([samplePng, samplePng])

    await waitForTwoCompressions(page)

    const bottomBar = page.getByTestId('bottom-bar')
    await expect(bottomBar).toBeVisible()
    await expect(bottomBar.getByRole('button', { name: 'Download ZIP' })).toBeVisible()
    await expect(bottomBar.getByText(/What we did:/)).toBeVisible()

    const slider = bottomBar.locator('input[type="range"]')
    await expect(slider).toBeVisible()
    const box = await slider.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.width).toBeGreaterThanOrEqual(MIN_PRESET_SLIDER_WIDTH_PX)
  })

  test('Indomaret screenshot compresses without pixel buffer mismatch', async ({ page }) => {
    await openApp(page)
    await page.locator('input[type="file"]').setInputFiles(indomaretScreenshot)
    await expect(page.getByText(/pixel count mismatch/i)).not.toBeVisible()
    await waitForCompression(page)
    await expect(page.getByTestId('file-item-done').or(page.getByTestId('file-item-optimized'))).toBeVisible()
  })
})
