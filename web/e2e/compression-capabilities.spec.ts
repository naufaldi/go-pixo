import { expect, test } from '@playwright/test'
import { fileURLToPath } from 'node:url'

const generatedPhoto = fileURLToPath(new URL('./fixtures/benchmark/codex-still-life.png', import.meta.url))

test('Codex photo compresses substantially in Auto while keeping dimensions and useful detail', async ({ page }, testInfo) => {
  await page.goto('/')
  await page.getByRole('combobox', { name: 'Output format' }).selectOption('auto')
  await page.locator('input[type="file"]').setInputFiles(generatedPhoto)
  await expect(page.getByTestId('batch-summary')).toContainText('1 completed, 0 failed', { timeout: 60000 })

  const result = await page.evaluate(async () => {
    const source = document.querySelector<HTMLImageElement>('img[alt="Original"]')!
    const output = document.querySelector<HTMLImageElement>('img[alt="Compressed"]')!
    const originalBlob = await fetch(source.src).then(response => response.blob())
    const compressedBlob = await fetch(output.src).then(response => response.blob())
    const original = await createImageBitmap(originalBlob)
    const compressed = await createImageBitmap(compressedBlob)
    const pixels = (image: ImageBitmap) => {
      const canvas = document.createElement('canvas')
      canvas.width = image.width
      canvas.height = image.height
      const context = canvas.getContext('2d')!
      context.drawImage(image, 0, 0)
      return context.getImageData(0, 0, image.width, image.height).data
    }
    const before = pixels(original)
    const after = pixels(compressed)
    let squaredError = 0
    for (let offset = 0; offset < before.length; offset += 4) {
      for (let channel = 0; channel < 3; channel++) {
        squaredError += (before[offset + channel] - after[offset + channel]) ** 2
      }
    }
    const mse = squaredError / (original.width * original.height * 3)
    const bytes = new Uint8Array(await compressedBlob.arrayBuffer())
    const metadata = {
      originalBytes: originalBlob.size,
      compressedBytes: compressedBlob.size,
      mime: compressedBlob.type,
      width: compressed.width,
      height: compressed.height,
      originalWidth: original.width,
      originalHeight: original.height,
      psnr: mse === 0 ? 100 : 10 * Math.log10(255 ** 2 / mse),
      header: Array.from(bytes.slice(0, 12)),
    }
    original.close()
    compressed.close()
    return metadata
  })

  expect(result.width).toBe(1536)
  expect(result.height).toBe(1024)
  expect(result.width).toBe(result.originalWidth)
  expect(result.height).toBe(result.originalHeight)
  expect(result.compressedBytes).toBeLessThan(result.originalBytes * 0.75)
  // PSNR catches severe pixel corruption without claiming perceptual equivalence.
  expect(result.psnr).toBeGreaterThan(28)
  expect(result.mime).toBe('image/webp')
  expect(String.fromCharCode(...result.header.slice(0, 4))).toBe('RIFF')
  expect(String.fromCharCode(...result.header.slice(8, 12))).toBe('WEBP')
  console.log('Codex still-life compression:', JSON.stringify(result))
  await testInfo.attach('Codex photo compression', {
    body: JSON.stringify(result, null, 2), contentType: 'application/json',
  })
  await page.screenshot({ path: testInfo.outputPath('codex-photo-compressed.png'), fullPage: true })

  // A second copy also exercises the completed batch controls on a narrow screen.
  await page.locator('input[type="file"]').setInputFiles(generatedPhoto)
  await expect(page.getByTestId('batch-summary')).toContainText('2 completed, 0 failed', { timeout: 60000 })
  await page.setViewportSize({ width: 390, height: 844 })
  const zip = page.getByRole('button', { name: 'Download ZIP', exact: true })
  await expect(zip).toBeVisible()
  const bounds = await zip.boundingBox()
  expect(bounds).not.toBeNull()
  expect(bounds!.x).toBeGreaterThanOrEqual(0)
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390)
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390)
  await page.screenshot({ path: testInfo.outputPath('codex-photo-mobile.png'), fullPage: true })
})
